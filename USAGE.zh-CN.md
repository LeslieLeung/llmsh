# llmsh 使用文档

[English](USAGE.md) | 简体中文

llmsh 是一个 ZSH 插件，使用大语言模型提供智能命令预测、补全和自然语言到命令的转换功能。

## 目录

- [配置](#配置)
- [子命令](#子命令)
- [自定义](#自定义)
- [支持的大语言模型提供商](#支持的大语言模型提供商)
- [故障排除](#故障排除)
- [JSON 请求/响应格式](#json-请求响应格式)

---

## 配置

### 初始化配置

在使用 llmsh 之前，先初始化配置文件：

```bash
llmsh config init
```

这会在 `~/.llmsh/config.yaml` 创建一个配置文件，包含以下默认设置：
- LLM 提供商（OpenAI、本地/Ollama）
- 预测设置
- 缓存设置
- Token 追踪
- ZSH 按键绑定

初始化后：
1. 设置你的 OpenAI API 密钥：`export OPENAI_API_KEY="your-api-key"`
2. 或通过将配置中的 `default_provider` 改为 `local` 来配置本地 LLM 提供商（如 Ollama）
3. 在你的 `~/.zshrc` 中加载 ZSH 插件

### 查看配置

显示当前配置：

```bash
llmsh config show
```

### 编辑配置

编辑 `~/.llmsh/config.yaml` 配置文件：

```yaml
providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    model: gpt-4-turbo-preview
    base_url: https://api.openai.com/v1

  local:
    model: llama2
    base_url: http://localhost:11434/v1  # Ollama
    api_key: "not-needed"

default_provider: openai

cache:
  enabled: true
  ttl_hours: 24
  max_entries: 1000

prediction:
  max_history_length: 10
  min_prefix_length: 3

tracking:
  enabled: true
  db_path: ~/.llmsh/tokens.json
```

---

## 子命令

### config

管理 llmsh 配置。

#### 子命令：

**`llmsh config init`**

在 `~/.llmsh/config.yaml` 初始化配置文件。

- 创建包含 OpenAI 和本地提供商设置的默认配置
- 设置缓存、追踪和预测参数
- 提供后续设置步骤

**`llmsh config show`**

显示当前配置文件内容。

---

### predict

基于上下文预测下一条 shell 命令。

**用法：**
```bash
echo '{"method":"predict","history":["git status","git add ."],"cwd":"/home/user/project","git_branch":"main","os_info":"Darwin"}' | llmsh predict
```

**目的：** 分析最近的命令历史、当前目录和 git 分支，预测你下一步可能执行的命令。

**输入（通过 stdin 的 JSON）：**
- `method`: "predict"（必需）
- `history`: 最近的 shell 命令数组
- `cwd`: 当前工作目录
- `git_branch`: 当前 git 分支（如果在 git 仓库中）
- `os_info`: 操作系统信息

**输出（通过 stdout 的 JSON）：**
```json
{
  "result": {
    "command": "git commit -m \"update\"",
    "cached": false
  },
  "tokens": {
    "input_tokens": 150,
    "output_tokens": 10,
    "cache_creation_tokens": 0,
    "cache_read_tokens": 0
  }
}
```

**特性：**
- 使用命令历史来理解工作流模式
- 考虑 git 上下文和工作目录
- 缓存预测以减少 API 调用
- 从历史记录中过滤敏感信息（密码、token）

---

### complete

基于上下文补全部分命令。

**用法：**
```bash
echo '{"method":"complete","prefix":"git co","history":["git status","git branch"],"cwd":"/home/user/project","os_info":"Darwin"}' | llmsh complete
```

**目的：** 基于上下文和最近历史补全部分输入的命令。

**输入（通过 stdin 的 JSON）：**
- `method`: "complete"（必需）
- `prefix`: 要补全的部分命令（必需）
- `history`: 最近的 shell 命令数组
- `cwd`: 当前工作目录
- `os_info`: 操作系统信息

**输出（通过 stdout 的 JSON）：**
```json
{
  "result": {
    "command": "git checkout main",
    "cached": false
  },
  "tokens": {
    "input_tokens": 120,
    "output_tokens": 8
  }
}
```

**特性：**
- 需要最小前缀长度（可通过 `prediction.min_prefix_length` 配置）
- 使用最近的命令历史提供更好的上下文
- 返回实用、安全的命令补全

---

### nl2cmd

将自然语言描述转换为 shell 命令。

**用法：**
```bash
echo '{"method":"nl2cmd","description":"列出过去24小时内修改的所有文件","history":["ls -la"],"cwd":"/home/user","os_info":"Darwin"}' | llmsh nl2cmd
```

**目的：** 将自然语言描述翻译为可执行的 shell 命令。

**输入（通过 stdin 的 JSON）：**
- `method`: "nl2cmd"（必需）
- `description`: 所需命令的自然语言描述（必需）
- `history`: 最近的 shell 命令数组（可选）
- `cwd`: 当前工作目录
- `os_info`: 操作系统信息

**输出（通过 stdout 的 JSON）：**
```json
{
  "result": {
    "command": "find . -type f -mtime -1",
    "cached": false
  },
  "tokens": {
    "input_tokens": 180,
    "output_tokens": 12
  }
}
```

**特性：**
- 生成安全、实用的命令
- 使用常见的 Unix/Linux 工具
- 考虑操作系统和当前目录上下文
- 显示历史记录中的最后 3 条命令以提供额外上下文

---

### stats

显示 token 使用统计。

**用法：**
```bash
llmsh stats
```

**目的：** 显示关于 LLM token 使用的汇总统计信息，帮助监控成本和缓存效率。

**输出：** 在三个类别中显示统计信息：

1. **按提供商/模型的使用情况：**
   - 提供商和模型名称
   - 请求次数
   - 输入和输出 token
   - 缓存读取 token（如适用）

2. **按天的使用情况：**
   - 每日 token 使用明细
   - 每天的请求数
   - 每天的输入/输出/缓存 token

3. **按方法的使用情况：**
   - 每个子命令的 token 使用（predict、complete、nl2cmd）
   - 每种方法的请求数

4. **总计摘要：**
   - 所有使用的总请求数和 token 数
   - 缓存节省百分比（如果启用了提示缓存）

**示例输出：**
```
Token Usage Statistics
======================

Usage by Provider/Model:
------------------------
openai / gpt-4-turbo-preview:
  Requests:      42
  Input Tokens:  8500
  Output Tokens: 450
  Cache Read:    2100

Usage by Day:
-------------
2024-01-15:
  Requests:      15
  Input Tokens:  3200
  Output Tokens: 180
  Cache Read:    800

Usage by Method:
----------------
predict:
  Requests:      20
  Input Tokens:  4000
  Output Tokens: 200

Total Summary:
--------------
  Total Requests:      42
  Total Input Tokens:  8500
  Total Output Tokens: 450
  Total Cache Read:    2100
  Cache Savings:       19.8%
```

**数据位置：** Token 追踪数据存储在 `~/.llmsh/tokens.json`（可通过 `tracking.db_path` 配置）。

---

### clean

清理 llmsh 数据文件。

**用法：**
```bash
# 仅清理日志和缓存
llmsh clean

# 清理日志、缓存和 token 追踪数据
llmsh clean --all
llmsh clean -a
```

**目的：** 删除临时文件、缓存，以及可选的 token 追踪数据，以释放空间或重置状态。

**清理内容：**

**默认（无标志）：**
- 调试日志文件（`/tmp/llmsh_debug.log`）
- 缓存数据库（`~/.llmsh/cache.db`）

**使用 `--all` 或 `-a` 标志：**
- 以上所有内容，加上：
- Token 追踪数据（`~/.llmsh/tokens.json`）

**注意：** clean 命令永远不会删除配置文件（`~/.llmsh/config.yaml`）。

**示例输出：**
```
Cleaned:
  ✓ debug log
  ✓ cache database
  ✓ token tracking data
```

---

## 自定义

### 按键绑定

编辑 `~/.llmsh/llmsh.plugin.zsh` 插件文件来自定义按键绑定：

```zsh
# 自然语言转命令（默认：Alt+Enter）
bindkey '^[^M' _llmsh_nl2cmd_widget

# 智能补全/预测（默认：Ctrl+O）
# - 空缓冲区：预测下一条命令
# - 有文本：补全当前命令
bindkey '^O' _llmsh_predict_next_widget
```

---

## 支持的大语言模型提供商

### OpenAI
```yaml
providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    model: gpt-4-turbo-preview
    base_url: https://api.openai.com/v1
```

### 本地 LLM（Ollama）
```yaml
providers:
  local:
    model: llama2
    base_url: http://localhost:11434/v1
    api_key: "not-needed"
```

### 其他兼容 OpenAI 的 API

任何兼容 OpenAI 的 API 端点都可以通过设置 `base_url` 来使用：

```yaml
providers:
  custom:
    api_key: your-api-key
    model: your-model
    base_url: https://your-api.com/v1
```

---

## 故障排除

### 插件未加载
```bash
# 检查插件文件是否存在
ls -l ~/.llmsh/llmsh.plugin.zsh

# 检查是否在 ~/.zshrc 中引用
grep llmsh.plugin.zsh ~/.zshrc

# 重新加载 shell
source ~/.zshrc
```

### 没有预测出现
```bash
# 检查二进制文件是否存在且可执行
which llmsh
llmsh config show

# 检查 API 密钥是否已设置
echo $OPENAI_API_KEY

# 手动测试
echo '{"method":"predict","history":["ls","pwd"],"cwd":"'$PWD'","os_info":"Darwin"}' | llmsh predict
```

### 找不到 jq
```bash
# 安装 jq
brew install jq  # macOS
sudo apt-get install jq  # Linux
```

### zsh-autosuggestions 冲突

该插件可以与 zsh-autosuggestions 配合使用或独立使用。如果有冲突：

1. 确保在 `~/.zshrc` 中 llmsh 在 zsh-autosuggestions **之后**加载
2. 检查按键绑定冲突：`bindkey | grep -E "(RIGHT|\\^\\[\\^M)"`

---

## JSON 请求/响应格式

所有与预测相关的子命令（predict、complete、nl2cmd）通过 stdin/stdout 以 JSON 格式通信，专为与 ZSH 插件集成而设计。

### 请求结构

```json
{
  "method": "predict|complete|nl2cmd",
  "history": ["cmd1", "cmd2", "cmd3"],
  "cwd": "/current/working/directory",
  "git_branch": "main",
  "os_info": "Darwin",
  "prefix": "partial command",
  "description": "natural language description",
  "timestamp": 1234567890
}
```

**字段：**
- `method`: 要执行的操作（必需）
- `history`: 最近的 shell 命令（可选）
- `cwd`: 当前工作目录（可选但推荐）
- `git_branch`: Git 分支名称（可选）
- `os_info`: 操作系统信息（可选但推荐）
- `prefix`: "complete" 方法必需
- `description`: "nl2cmd" 方法必需
- `timestamp`: Unix 时间戳（可选）

### 响应结构

**成功响应：**
```json
{
  "result": {
    "command": "the resulting command",
    "confidence": 0.95,
    "cached": false
  },
  "tokens": {
    "input_tokens": 150,
    "output_tokens": 10,
    "cache_creation_tokens": 0,
    "cache_read_tokens": 0
  }
}
```

**错误响应：**
```json
{
  "error": "error message describing what went wrong"
}
```

**响应字段：**
- `result.command`: 预测/补全/生成的命令
- `result.cached`: 结果是否从缓存中检索
- `tokens`: Token 使用信息（仅在非缓存时出现）
- `error`: 错误消息（仅在发生错误时出现）

---

## 与 ZSH 的集成

llmsh 设计为 ZSH 插件的一部分使用。二进制文件处理 LLM 交互，而 ZSH 插件提供：
- 预测和 nl2cmd 的按键绑定
- 上下文收集（历史、cwd、git 分支）
- 显示建议的用户界面

参考 `~/.llmsh/config.yaml` 中 `zsh.keybindings` 下的 ZSH 插件配置来自定义键盘快捷键。
