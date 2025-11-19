# 带防抖的自动建议功能 - 技术设计文档

**版本:** 1.0
**日期:** 2024-11-19
**作者:** Claude
**状态:** 草稿

---

## 目录

1. [概述](#概述)
2. [需求分析](#需求分析)
3. [系统架构](#系统架构)
4. [详细设计](#详细设计)
5. [实施计划](#实施计划)
6. [测试策略](#测试策略)
7. [性能考量](#性能考量)
8. [安全考量](#安全考量)
9. [部署方案](#部署方案)
10. [风险评估](#风险评估)
11. [未来增强](#未来增强)

---

## 概述

### 目标
为 llmsh 实现一个智能的、非侵入式的命令自动建议功能，配备防抖机制，类似 zsh-autosuggestions 但由 LLM 驱动。

### 核心特性
- **静默建议**: 以灰色文字显示 LLM 生成的命令建议，不打断用户输入
- **防抖机制**: 通过 500ms 防抖延迟防止过度的 LLM API 调用
- **简洁快捷键**:
  - `右方向键 (→)`: 接受建议
  - `ESC`: 清除建议
- **缓存优先**: 利用现有的 SQLite 缓存最小化 API 成本
- **可配置**: 允许用户启用/禁用和自定义行为

### 成功指标
- ✅ 用户输入期间每 500ms 最多触发 1 次 LLM API 调用
- ✅ 缓存结果的建议在 200ms 内出现
- ✅ 对终端响应性零影响
- ✅ 常见工作流的缓存命中率 > 60%

---

## 需求分析

### 功能性需求

#### FR-1: 输入防抖
- **优先级**: P0 (关键)
- **描述**: 系统必须等待 500ms 的用户无操作时间后才触发 LLM 请求
- **验收标准**:
  - 用户连续输入不应触发多次 API 调用
  - 每次按键重置计时器
  - 通过 `prediction.debounce_delay_ms` 配置可自定义延迟时间

#### FR-2: 显示静默建议
- **优先级**: P0 (关键)
- **描述**: 在用户当前输入后以灰色文字显示 LLM 生成的建议
- **验收标准**:
  - 建议出现在 POSTDISPLAY 区域
  - 文字颜色为灰色 (fg=8)
  - 不干扰用户输入
  - 当用户继续输入不同内容时自动清除

#### FR-3: 接受建议
- **优先级**: P0 (关键)
- **描述**: 用户可以通过按右方向键接受建议
- **验收标准**:
  - 右方向键将建议插入 BUFFER
  - 光标移动到接受文本的末尾
  - 接受后清除建议
  - 在 emacs 和 vi 插入模式下均可工作

#### FR-4: 清除建议
- **优先级**: P0 (关键)
- **描述**: 用户可以通过按 ESC 键清除建议
- **验收标准**:
  - ESC 键清除 POSTDISPLAY 和 region_highlight
  - 不影响用户当前的 BUFFER
  - 视觉反馈即时

#### FR-5: 最小前缀长度
- **优先级**: P1 (高)
- **描述**: 仅在用户输入最少字符数时触发建议
- **验收标准**:
  - 默认最小长度: 3 个字符
  - 通过 `prediction.min_prefix_length` 可配置
  - 空缓冲区不应触发建议

#### FR-6: 异步 LLM 调用
- **优先级**: P1 (高)
- **描述**: LLM API 调用不得阻塞终端输入
- **验收标准**:
  - API 调用期间终端保持响应
  - 用户可以在请求进行时继续输入
  - 如果用户修改输入，取消进行中的请求

#### FR-7: 配置选项
- **优先级**: P1 (高)
- **描述**: 暴露配置供用户自定义
- **配置键**:
  ```yaml
  prediction:
    auto_suggest: true                # 启用/禁用功能
    debounce_delay_ms: 500            # 防抖延迟
    min_prefix_length: 3              # 触发的最少字符数
    max_suggestion_length: 150        # 最大建议长度
    show_loading_indicator: false     # 加载时显示 "..."
  ```

### 非功能性需求

#### NFR-1: 性能
- 缓存命令的建议: < 50ms
- 新命令的建议: < 2s (LLM 延迟)
- 内存开销: < 10MB 额外
- CPU 开销: 空闲时 < 5%

#### NFR-2: 可靠性
- 优雅处理 LLM API 错误 (不显示内容,不崩溃)
- 有缓存时可离线工作
- 保持终端稳定性 (无冻结或崩溃)

#### NFR-3: 兼容性
- 支持 ZSH 5.0+
- 与现有 widgets 共存 (zsh-autosuggestions 等)
- 支持 emacs 和 vi 键盘映射
- 跨平台: Linux、macOS

#### NFR-4: 安全性
- 过滤敏感模式 (password、token、secret、key)
- 不向 LLM 发送敏感命令
- 遵守 `pkg/context/filter.go` 中现有的敏感过滤

---

## 系统架构

### 高层架构

```
┌─────────────────────────────────────────────────────────────┐
│                         用户终端                             │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  ZSH 会话                                              │ │
│  │  ┌──────────────────────────────────────────────────┐ │ │
│  │  │  ZLE (Zsh Line Editor)                           │ │ │
│  │  │  ┌────────────────────────────────────────────┐  │ │ │
│  │  │  │ BUFFER: "git comm"                         │  │ │ │
│  │  │  │ POSTDISPLAY: "it -m ''" [灰色]             │  │ │ │
│  │  │  └────────────────────────────────────────────┘  │ │ │
│  │  └──────────────────────────────────────────────────┘ │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ (按键事件)
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              llmsh.plugin.zsh (ZSH 插件层)                   │
│                                                              │
│  ┌──────────────────┐      ┌────────────────────────────┐  │
│  │ 事件处理器       │      │  防抖计时器                 │  │
│  │ _llmsh_on_change │─────▶│  (500ms 延迟)              │  │
│  └──────────────────┘      └────────────────────────────┘  │
│           │                            │                    │
│           │ (缓冲区改变)                │ (计时器触发)       │
│           ▼                            ▼                    │
│  ┌──────────────────┐      ┌────────────────────────────┐  │
│  │ 重置计时器       │      │  _llmsh_fetch_suggestion   │  │
│  └──────────────────┘      └────────────────────────────┘  │
│                                        │                    │
│                                        │ (异步调用)         │
│                                        ▼                    │
│                            ┌────────────────────────────┐  │
│                            │ _llmsh_call_binary_async   │  │
│                            │  (后台进程)                │  │
│                            └────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                                        │
                                        │ (JSON 请求)
                                        ▼
┌─────────────────────────────────────────────────────────────┐
│                   llmsh 二进制 (Go 层)                       │
│                                                              │
│  ┌──────────────────┐      ┌────────────────────────────┐  │
│  │ cmd/complete.go  │─────▶│  缓存检查                  │  │
│  │ runComplete()    │      │  (SQLite)                  │  │
│  └──────────────────┘      └────────────────────────────┘  │
│           │                            │                    │
│           │ (缓存未命中)                │ (缓存命中)         │
│           ▼                            ▼                    │
│  ┌──────────────────┐      ┌────────────────────────────┐  │
│  │ pkg/llm/client   │      │  返回缓存结果              │  │
│  │ Complete()       │      └────────────────────────────┘  │
│  └──────────────────┘                                       │
│           │                                                 │
│           │ (LLM API 调用)                                  │
│           ▼                                                 │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         OpenAI API / 兼容端点                        │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ (响应)
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              回调: _llmsh_display_suggestion                 │
│                                                              │
│  1. 从 JSON 响应中提取命令                                   │
│  2. 移除用户已输入的前缀                                     │
│  3. 用剩余后缀设置 POSTDISPLAY                               │
│  4. 通过 region_highlight 应用灰色                           │
│  5. 调用 zle -R 刷新显示                                    │
└─────────────────────────────────────────────────────────────┘
```

### 组件分解

#### 1. ZSH 插件层 (`zsh/llmsh.plugin.zsh`)

**职责:**
- 监听 ZLE 事件 (缓冲区变化)
- 管理防抖计时器
- 通过 POSTDISPLAY 显示建议
- 处理快捷键 (右方向键、ESC)
- 异步调用 Go 二进制

**关键函数:**
```zsh
_llmsh_on_buffer_change()       # 每次按键触发的钩子
_llmsh_debounced_suggest()      # 带计时器的防抖逻辑
_llmsh_fetch_suggestion()       # 触发异步 LLM 调用
_llmsh_call_binary_async()      # 在后台执行 Go 二进制
_llmsh_display_suggestion()     # 在 POSTDISPLAY 中显示建议
_llmsh_accept_suggestion()      # 右方向键处理器
_llmsh_clear_suggestion()       # ESC 处理器
_llmsh_should_suggest()         # 过滤逻辑 (最小长度、敏感词)
```

**状态变量:**
```zsh
LLMSH_DEBOUNCE_TIMER_PID        # 后台计时器进程的 PID
LLMSH_LAST_BUFFER               # 上次的缓冲区值 (用于变化检测)
LLMSH_CURRENT_SUGGESTION        # 当前显示的建议
LLMSH_INFLIGHT_REQUEST          # 进行中的 LLM 请求的 PID
```

#### 2. Go 二进制层 (`cmd/complete.go`)

**职责:**
- 从 stdin 接收 JSON 请求
- 检查缓存中的现有建议
- 如果缓存未命中则调用 LLM API
- 向 stdout 返回 JSON 响应

**请求格式:**
```json
{
  "method": "complete",
  "prefix": "git comm",
  "history": ["ls", "cd project", "git status"],
  "cwd": "/home/user/project",
  "git_branch": "main",
  "os_info": "Linux"
}
```

**响应格式:**
```json
{
  "result": {
    "command": "git commit -m ''",
    "cached": true,
    "model": "gpt-4-turbo-preview"
  },
  "usage": {
    "prompt_tokens": 120,
    "completion_tokens": 8,
    "total_tokens": 128
  }
}
```

#### 3. 配置层 (`pkg/config/config.go`)

**新配置结构:**
```go
type PredictionConfig struct {
    HistoryLength          int  `mapstructure:"history_length"`
    MinPrefixLength        int  `mapstructure:"min_prefix_length"`

    // 自动建议的新字段
    AutoSuggest            bool `mapstructure:"auto_suggest"`
    DebounceDelayMs        int  `mapstructure:"debounce_delay_ms"`
    MaxSuggestionLength    int  `mapstructure:"max_suggestion_length"`
    ShowLoadingIndicator   bool `mapstructure:"show_loading_indicator"`
}
```

---

## 详细设计

### 1. 防抖机制

#### 实现策略

**选项 A: 纯 ZSH 计时器 (已选)**

优点:
- 无外部依赖
- ZSH 内置
- 简单实现

缺点:
- 计时精度较低
- 需要后台进程

**实现:**
```zsh
_llmsh_debounced_suggest() {
    # 如果存在则杀死之前的计时器
    if [[ -n "$LLMSH_DEBOUNCE_TIMER_PID" ]]; then
        kill $LLMSH_DEBOUNCE_TIMER_PID 2>/dev/null
        LLMSH_DEBOUNCE_TIMER_PID=""
    fi

    # 启动新的后台计时器
    {
        sleep ${LLMSH_DEBOUNCE_DELAY:-0.5}
        _llmsh_fetch_suggestion
    } &
    LLMSH_DEBOUNCE_TIMER_PID=$!
}
```

#### 防抖流程

```
时间 (ms)    0      100    200    300    400    500    600    700    800
用户输入     g      i      t      _      c      o      [停止]
             │      │      │      │      │      │      │
计时器       ├─X    ├─X    ├─X    ├─X    ├─X    ├─X    ├──────────────┤
             │      │      │      │      │      │                     │
             重置   重置   重置   重置   重置   重置                   触发!
                                                                       │
                                                                       ▼
                                                              LLM 请求
```

### 2. 异步 LLM 调用

#### 后台进程方式

```zsh
_llmsh_call_binary_async() {
    local method="$1"
    local prefix="$2"

    # 构建 JSON 请求
    local context=$(_llmsh_get_context 10)
    local request="{\"method\":\"${method}\",${context},\"prefix\":\"${prefix}\"}"

    # 创建临时 FIFO 用于响应
    local fifo="/tmp/llmsh_response_$$"
    mkfifo "$fifo"

    # 后台任务: 调用二进制并写入 FIFO
    {
        local response=$(echo "$request" | "$LLMSH_BINARY" "$method" 2>/dev/null)
        echo "$response" > "$fifo"
    } &
    LLMSH_INFLIGHT_REQUEST=$!

    # 使用 zle -F 注册文件描述符处理器 (非阻塞读取)
    exec {fd}<"$fifo"
    zle -F $fd _llmsh_handle_response
}

_llmsh_handle_response() {
    local fd=$1

    # 从文件描述符读取响应
    local response
    read -r response <&$fd

    # 关闭文件描述符并清理
    exec {fd}<&-
    rm -f "/tmp/llmsh_response_$$"

    # 显示建议
    _llmsh_display_suggestion "$response"
}
```

#### 请求取消

```zsh
_llmsh_cancel_inflight() {
    if [[ -n "$LLMSH_INFLIGHT_REQUEST" ]]; then
        kill $LLMSH_INFLIGHT_REQUEST 2>/dev/null
        LLMSH_INFLIGHT_REQUEST=""
    fi
}

# 当用户改变缓冲区时取消
_llmsh_on_buffer_change() {
    if [[ "$BUFFER" != "$LLMSH_LAST_BUFFER" ]]; then
        _llmsh_cancel_inflight
        _llmsh_clear_suggestion
        _llmsh_debounced_suggest
    fi
}
```

### 3. 建议显示

#### 视觉渲染

```zsh
_llmsh_display_suggestion() {
    local response="$1"

    # 从 JSON 中提取命令
    local full_command=$(echo "$response" | jq -r '.result.command // empty')

    # 验证
    if [[ -z "$full_command" ]] || [[ "$full_command" == "$BUFFER" ]]; then
        return
    fi

    # 提取后缀 (移除已输入的前缀)
    local suffix="${full_command#$BUFFER}"

    # 如果太长则截断
    local max_len=${LLMSH_MAX_SUGGESTION_LENGTH:-150}
    if [[ ${#suffix} -gt $max_len ]]; then
        suffix="${suffix:0:$max_len}..."
    fi

    # 设置显示变量
    LLMSH_CURRENT_SUGGESTION="$suffix"
    POSTDISPLAY="$suffix"

    # 应用灰色 (fg=8)
    local start=$#BUFFER
    local end=$(($start + $#POSTDISPLAY))
    region_highlight+=("$start $end fg=8")

    # 刷新显示
    zle -R
}
```

#### 颜色方案

| 状态 | 颜色 | ZSH 代码 |
|------|------|----------|
| 正常建议 | 灰色 | `fg=8` |
| 加载指示器 (可选) | 青色 | `fg=cyan` |
| 缓存结果 (可选) | 浅灰色 | `fg=8,italic` |

### 4. 快捷键绑定

#### 右方向键: 接受建议

```zsh
_llmsh_accept_suggestion() {
    if [[ -n "$LLMSH_CURRENT_SUGGESTION" ]]; then
        # 将建议附加到缓冲区
        BUFFER="${BUFFER}${LLMSH_CURRENT_SUGGESTION}"
        CURSOR=$#BUFFER

        # 清除建议状态
        _llmsh_clear_suggestion

        # 为完成的命令触发新建议
        _llmsh_debounced_suggest
    else
        # 无建议: 回退到默认的右方向键行为 (forward-char)
        zle forward-char
    fi

    zle -R
}

# 注册 widget
zle -N _llmsh_accept_suggestion
bindkey '^[[C' _llmsh_accept_suggestion  # 右方向键
```

#### ESC: 清除建议

```zsh
_llmsh_clear_suggestion() {
    POSTDISPLAY=""
    LLMSH_CURRENT_SUGGESTION=""
    region_highlight=()
    zle -R
}

# ESC 键
bindkey '^[' _llmsh_clear_suggestion
```

### 5. 智能过滤

#### 何时不建议

```zsh
_llmsh_should_suggest() {
    local buffer="$BUFFER"

    # 1. 功能禁用
    if [[ "${LLMSH_AUTO_SUGGEST:-true}" != "true" ]]; then
        return 1
    fi

    # 2. 缓冲区太短
    local min_len=${LLMSH_MIN_PREFIX_LENGTH:-3}
    if [[ ${#buffer} -lt $min_len ]]; then
        return 1
    fi

    # 3. 在 vi 命令模式
    if [[ "$KEYMAP" == "vicmd" ]]; then
        return 1
    fi

    # 4. 在补全菜单中
    if [[ -n "$MENUSELECT" ]]; then
        return 1
    fi

    # 5. 敏感关键词
    local sensitive_pattern='(password|passwd|token|secret|key|apikey|api_key)='
    if [[ "$buffer" =~ $sensitive_pattern ]]; then
        return 1
    fi

    # 6. 对相同前缀已有建议
    if [[ -n "$LLMSH_CURRENT_SUGGESTION" ]] && [[ "$buffer" == "$LLMSH_LAST_SUGGESTED_PREFIX" ]]; then
        return 1
    fi

    return 0
}
```

### 6. 事件钩子

#### 缓冲区变化检测

```zsh
# 钩入 ZLE widgets
_llmsh_zle_line_init() {
    # 安装缓冲区变化钩子
    zle -N zle-line-pre-redraw _llmsh_on_buffer_change
}

_llmsh_on_buffer_change() {
    # 检查缓冲区是否真的改变了
    if [[ "$BUFFER" != "$LLMSH_LAST_BUFFER" ]]; then
        LLMSH_LAST_BUFFER="$BUFFER"

        # 取消任何进行中的请求
        _llmsh_cancel_inflight

        # 清除当前建议
        _llmsh_clear_suggestion

        # 检查是否应该建议
        if _llmsh_should_suggest; then
            _llmsh_debounced_suggest
        fi
    fi
}

# 注册钩子
zle -N zle-line-init _llmsh_zle_line_init
```

---

## 实施计划

### 第一阶段: 核心功能 (第 1 周)

**任务:**
1. ✅ 扩展 `pkg/config/config.go` 中的 `PredictionConfig` 结构
2. ✅ 更新 `cmd/config.go` 设置默认值
3. ✅ 在 `zsh/llmsh.plugin.zsh` 中实现防抖计时器
4. ✅ 实现 `_llmsh_display_suggestion()` 函数
5. ✅ 实现 `_llmsh_accept_suggestion()` (右方向键)
6. ✅ 实现 `_llmsh_clear_suggestion()` (ESC)
7. ✅ 添加缓冲区变化检测钩子
8. ✅ 添加 `_llmsh_should_suggest()` 过滤逻辑

**修改的文件:**
- `pkg/config/config.go` (+8 行)
- `cmd/config.go` (+4 行)
- `zsh/llmsh.plugin.zsh` (+120 行)

**交付物:**
- 基本自动建议可工作
- 防抖防止过度调用
- 快捷键功能正常

### 第二阶段: 异步 & 性能 (第 2 周)

**任务:**
1. ✅ 使用 FIFO 实现异步二进制调用
2. ✅ 添加请求取消逻辑
3. ✅ 优化缓存键生成
4. ✅ 添加加载指示器 (可选)
5. ✅ 性能测试和调优

**修改的文件:**
- `zsh/llmsh.plugin.zsh` (+60 行)
- `cmd/complete.go` (优化缓存查找)

**交付物:**
- 非阻塞 LLM 调用
- 可以取消进行中的请求
- 记录缓存命中率

### 第三阶段: 完善 & 测试 (第 3 周)

**任务:**
1. ✅ 不同 shell 的集成测试
2. ✅ 兼容性测试 (oh-my-zsh、Prezto)
3. ✅ 文档更新 (USAGE.md、README.md)
4. ✅ 添加配置示例
5. ✅ 修复边缘情况和 bug

**修改的文件:**
- `USAGE.md` (添加自动建议部分)
- `README.md` (提及新功能)
- `zsh/llmsh.plugin.zsh` (bug 修复)

**交付物:**
- 全面的测试覆盖
- 更新的文档
- 准备进行 beta 发布

---

## 测试策略

### 单元测试

#### ZSH 单元测试 (使用 zunit 或手动)

```zsh
# 测试: 按键时防抖计时器重置
test_debounce_reset() {
    _llmsh_debounced_suggest
    local first_pid=$LLMSH_DEBOUNCE_TIMER_PID

    sleep 0.1
    _llmsh_debounced_suggest
    local second_pid=$LLMSH_DEBOUNCE_TIMER_PID

    # 计时器应该被重置 (不同的 PID)
    assert_not_equal "$first_pid" "$second_pid"
}

# 测试: 最小前缀长度
test_min_prefix_length() {
    LLMSH_MIN_PREFIX_LENGTH=3
    BUFFER="ab"

    if _llmsh_should_suggest; then
        fail "不应该为小于最小长度的缓冲区建议"
    fi

    BUFFER="abc"
    if ! _llmsh_should_suggest; then
        fail "应该为大于等于最小长度的缓冲区建议"
    fi
}

# 测试: 敏感关键词过滤
test_sensitive_filtering() {
    BUFFER="export PASSWORD=secret"

    if _llmsh_should_suggest; then
        fail "不应该为敏感内容建议"
    fi
}
```

#### Go 单元测试

```go
// 测试: 最小前缀补全
func TestCompleteMinPrefixLength(t *testing.T) {
    cfg := &config.Config{
        Prediction: config.PredictionConfig{
            MinPrefixLength: 3,
        },
    }

    req := &Request{Prefix: "ab"}
    err := validateRequest(req, cfg)

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "prefix too short")
}

// 测试: 缓存键生成一致性
func TestCacheKeyGeneration(t *testing.T) {
    history := []string{"ls", "cd project"}
    key1 := generateCacheKey(history, "/home/user", "main")
    key2 := generateCacheKey(history, "/home/user", "main")

    assert.Equal(t, key1, key2)
}
```

### 集成测试

#### 端到端测试脚本

```bash
#!/bin/bash
# test_auto_suggestion.sh

# 设置
source zsh/llmsh.plugin.zsh
export LLMSH_AUTO_SUGGEST=true
export LLMSH_DEBOUNCE_DELAY=0.1  # 测试时加快

# 测试 1: 延迟后出现建议
echo "测试 1: 建议出现"
BUFFER="git sta"
_llmsh_debounced_suggest
sleep 0.2

if [[ -z "$POSTDISPLAY" ]]; then
    echo "失败: 没有建议出现"
    exit 1
fi
echo "通过: 建议 = '$POSTDISPLAY'"

# 测试 2: 接受建议
echo "测试 2: 接受建议"
BUFFER="git sta"
LLMSH_CURRENT_SUGGESTION="tus"
_llmsh_accept_suggestion

if [[ "$BUFFER" != "git status" ]]; then
    echo "失败: 缓冲区 = '$BUFFER', 期望 'git status'"
    exit 1
fi
echo "通过: 建议已接受"

# 测试 3: 清除建议
echo "测试 3: 清除建议"
POSTDISPLAY="some suggestion"
_llmsh_clear_suggestion

if [[ -n "$POSTDISPLAY" ]]; then
    echo "失败: 建议未清除"
    exit 1
fi
echo "通过: 建议已清除"

echo "所有测试通过!"
```

### 手动测试检查表

- [ ] 输入 3+ 个字符后出现建议
- [ ] 继续输入时建议消失
- [ ] 右方向键正确接受建议
- [ ] ESC 清除建议
- [ ] 缓冲区 < 3 个字符时不调用 LLM
- [ ] 敏感关键词 (password、token) 时不调用 LLM
- [ ] 缓存命中即时返回
- [ ] LLM 调用期间终端保持响应
- [ ] 与 oh-my-zsh 主题配合使用
- [ ] 与 zsh-syntax-highlighting 配合使用
- [ ] 在 vi 模式下工作 (仅插入模式)
- [ ] 配置更改在插件重新加载后生效

---

## 性能考量

### 延迟目标

| 场景 | 目标 | 可接受 |
|------|------|--------|
| 缓存命中 | < 50ms | < 100ms |
| 缓存未命中 (LLM) | < 1000ms | < 2000ms |
| 防抖延迟 | 500ms | 300-800ms |
| UI 刷新 | < 16ms | < 50ms |

### 内存使用

**基线:** ~5MB (当前 llmsh 插件)

**带自动建议预期:** ~8MB
- +1MB 用于 FIFO 缓冲区
- +1MB 用于建议缓存 (内存中)
- +1MB 用于额外的 ZSH 状态

**缓解措施:**
- 限制建议长度 (默认 150 个字符)
- 从内存中清除旧建议
- 重用 FIFO 文件

### CPU 使用

**目标:** 空闲时 < 1% CPU,建议生成期间 < 10%

**优化:**
- 使用后台进程进行 LLM 调用 (无阻塞)
- 高效的字符串操作 (避免重复 `jq` 调用)
- 缓存命令验证结果

### 网络带宽

**假设:**
- 平均提示: 500 tokens (~2KB)
- 平均完成: 20 tokens (~100 bytes)

**估计使用:**
- 10 次建议/分钟 = ~20KB/分钟
- 600 次建议/小时 = ~1.2MB/小时

**缓解措施:**
- 高缓存命中率 (> 60%)
- 限制 LLM 配置中的最大 tokens (100 tokens)

---

## 安全考量

### 1. 敏感数据过滤

**威胁:** 用户输入敏感数据 (密码、API 密钥) 被发送到 LLM

**缓解措施:**
```zsh
# 敏感内容的正则表达式模式
LLMSH_SENSITIVE_PATTERNS=(
    '(password|passwd|pwd)='
    '(token|apikey|api_key)='
    '(secret|private_key)='
    'Authorization:\s+Bearer'
    'mysql.*-p'
    'psql.*password='
)

_llmsh_contains_sensitive() {
    local buffer="$1"
    for pattern in "${LLMSH_SENSITIVE_PATTERNS[@]}"; do
        if [[ "$buffer" =~ $pattern ]]; then
            return 0  # 包含敏感数据
        fi
    done
    return 1
}
```

**现有保护:** `pkg/context/filter.go` 已过滤历史记录,扩展到实时缓冲区

### 2. 命令注入

**威胁:** 恶意 LLM 响应包含命令注入

**缓解措施:**
- 仅在 POSTDISPLAY 中显示建议 (不自动执行)
- 用户必须明确接受 (右方向键)
- 验证响应格式 (JSON schema)

### 3. API 密钥暴露

**威胁:** API 密钥在日志或错误消息中泄露

**缓解措施:**
- 现有配置已处理 (环境变量)
- 确保错误消息不包含 API 密钥
- 所有 API 调用使用 HTTPS

### 4. 缓存污染

**威胁:** 恶意缓存条目导致危险建议

**缓解措施:**
- 缓存是本地 SQLite (不共享)
- TTL 使旧条目过期 (默认 7 天)
- 基于哈希的缓存键 (抗冲突)

---

## 部署方案

### 部署前

1. **配置迁移**
   ```bash
   # 备份现有配置
   cp ~/.llmsh/config.yaml ~/.llmsh/config.yaml.backup

   # 运行配置更新 (添加新字段)
   llmsh config migrate
   ```

2. **插件更新**
   ```bash
   cd /path/to/llmsh
   git pull origin main
   make install
   ```

3. **重新加载 ZSH 插件**
   ```bash
   # 添加到 ~/.zshrc 或手动重新加载
   source ~/.zsh/plugins/llmsh/llmsh.plugin.zsh
   ```

### 部署后

1. **验证安装**
   ```bash
   llmsh config show | grep auto_suggest
   # 应该输出: auto_suggest: true
   ```

2. **测试建议**
   ```bash
   # 输入 "git sta" 并等待 500ms
   # 灰色文字应该出现: "tus" 或 "status"
   ```

3. **监控性能**
   ```bash
   llmsh stats --last-hour
   # 检查缓存命中率、API 调用次数
   ```

### 回滚计划

如果出现问题:

```bash
# 1. 禁用自动建议
echo "prediction:\n  auto_suggest: false" >> ~/.llmsh/config.yaml

# 2. 重新加载插件
source ~/.zsh/plugins/llmsh/llmsh.plugin.zsh

# 3. 回退到之前版本
cd /path/to/llmsh
git checkout v1.0.0  # 之前的稳定版本
make install
```

---

## 风险评估

### 高风险

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| **LLM 调用期间终端冻结** | 中 | 严重 | 实现带超时的异步调用 |
| **过度的 API 成本** | 高 | 高 | 强制缓存、防抖、最小前缀长度 |
| **与 zsh-autosuggestions 冲突** | 中 | 中 | 检测并禁用冲突的插件 |
| **敏感数据泄露** | 低 | 严重 | 健壮的过滤 + 用户选择加入 |

### 中风险

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| **缓存命中率低** | 中 | 中 | 改进缓存键设计,更长的 TTL |
| **ZSH 版本不兼容** | 低 | 中 | 在 ZSH 5.0-5.9 上测试,记录要求 |
| **高内存使用** | 低 | 低 | 限制建议长度,清理旧状态 |

### 低风险

| 风险 | 概率 | 影响 | 缓解措施 |
|------|------|------|----------|
| **快捷键冲突** | 低 | 低 | 记录冲突,允许自定义绑定 |
| **主题视觉故障** | 低 | 低 | 测试流行主题 (oh-my-zsh、Powerlevel10k) |

---

## 未来增强

### 第四阶段: 高级功能 (MVP 后)

1. **部分接受**
   - `Ctrl+Right`: 一次接受一个单词
   - 对长建议有用

2. **上下文感知建议**
   - 分析 git 状态以建议相关命令
   - 检测文件类型以建议适当的工具

3. **多行建议**
   - 建议命令链 (管道)
   - 在多行上显示

4. **从接受的建议中学习**
   - 跟踪用户接受的建议
   - 使 LLM 偏向接受的模式
   - 本地微调 (可选)

5. **建议排名**
   - 显示多个建议 (编号)
   - 用户用 `Alt+1`、`Alt+2` 等选择

6. **离线模式**
   - 预加载常见建议
   - 回退到基于正则表达式的补全
   - 网络可用时同步

7. **分析仪表板**
   - Web UI 显示:
     - 随时间变化的缓存命中率
     - 最昂贵的 LLM 调用
     - 建议接受率
     - 缓存节省的成本

8. **协作过滤**
   - 共享匿名建议 (选择加入)
   - 从其他用户的模式中受益
   - 隐私保护聚合

---

## 附录

### A. 配置示例

```yaml
# ~/.llmsh/config.yaml

llm:
  default_provider: openai
  providers:
    openai:
      base_url: https://api.openai.com/v1
      api_key: ${OPENAI_API_KEY}
      model: gpt-4-turbo-preview
      max_tokens: 100
      temperature: 0.2

prediction:
  history_length: 20
  min_prefix_length: 3

  # 自动建议设置 (新)
  auto_suggest: true
  debounce_delay_ms: 500
  max_suggestion_length: 150
  show_loading_indicator: false

cache:
  enabled: true
  db_path: ~/.llmsh/cache.db
  ttl_days: 7
  max_entries: 1000

tracking:
  enabled: true
  db_path: ~/.llmsh/tokens.json

zsh:
  keybindings:
    nl2cmd: "^[^M"                    # Alt+Enter
    predict: "^O"                     # Ctrl+O (手动)
    accept_suggestion: "^[[C"         # 右方向键 (新)
    clear_suggestion: "^["            # ESC (新)
```

### B. 环境变量

```bash
# 通过环境变量覆盖配置
export LLMSH_AUTO_SUGGEST=true
export LLMSH_DEBOUNCE_DELAY_MS=500
export LLMSH_MIN_PREFIX_LENGTH=3
export LLMSH_MAX_SUGGESTION_LENGTH=150
```

### C. 调试

```bash
# 启用调试日志
export LLMSH_DEBUG=1

# 查看日志
tail -f ~/.llmsh/debug.log

# 测试特定建议
echo '{"method":"complete","prefix":"git sta"}' | llmsh complete
```

### D. 基准测试

```bash
# 测量缓存命中率
llmsh stats --cache-hit-rate

# 测量平均延迟
llmsh stats --avg-latency

# 测量 API 成本
llmsh stats --cost --last-day
```

---

## 术语表

- **防抖 (Debounce)**: 延迟函数执行直到用户停止输入的技术
- **POSTDISPLAY**: 用于在光标后显示文本而不影响缓冲区的 ZSH 变量
- **ZLE**: Zsh Line Editor,处理用户输入的组件
- **FIFO**: 先进先出管道,用于进程间通信
- **region_highlight**: 用于样式化特定文本区域的 ZSH 数组
- **Widget**: ZSH 术语,指绑定到快捷键的函数

---

## 审批

**文档状态:** 草稿
**下次审查日期:** 2024-11-26

**审查者:**
- [ ] 工程负责人
- [ ] 产品经理
- [ ] 安全团队
- [ ] UX 设计师

**批准:**
- [ ] 批准实施
- [ ] 需要修改
- [ ] 拒绝

---

**文档结束**
