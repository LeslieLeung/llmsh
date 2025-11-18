# 开发指南

[English](DEVELOPMENT.md) | 简体中文

## 架构

### 组件

- **Go 二进制文件**（`llmsh`）：LLM 交互、缓存和追踪的核心逻辑
  - 命令：`predict`、`complete`、`nl2cmd`、`config`、`stats`、`clean`
  - 通过 stdin/stdout 进行基于 JSON 的通信

- **ZSH 插件**（`llmsh.plugin.zsh`）：用户界面和上下文收集
  - 收集 shell 历史、git 上下文、工作目录
  - 管理防抖和冷却时间
  - 与 zsh-autosuggestions 集成
  - 提供按键绑定和小部件

### 数据流

```
用户输入 → ZSH 插件 → 收集上下文 → llmsh 二进制 → LLM API
              ↓                                        ↓
          显示 ← 解析响应 ← 缓存/追踪 ← JSON 响应
```

### 文件

- `~/.local/bin/llmsh` - 二进制可执行文件
- `~/.llmsh/llmsh.plugin.zsh` - ZSH 插件
- `~/.llmsh/config.yaml` - 配置文件
- `~/.llmsh/cache.db` - SQLite 缓存数据库
- `~/.llmsh/tokens.json` - Token 使用追踪
- `/tmp/llmsh_debug.log` - 调试日志（如果启用）

## 项目结构

```
llmsh/
├── cmd/              # Cobra CLI 命令
│   ├── root.go       # 根命令和 JSON 结构
│   ├── predict.go    # 下一条命令预测
│   ├── complete.go   # 命令补全
│   ├── nl2cmd.go     # 自然语言转换
│   ├── config.go     # 配置管理
│   ├── stats.go      # Token 使用统计
│   └── clean.go      # 数据清理
├── pkg/              # 核心包
│   ├── config/       # 配置管理
│   ├── llm/          # LLM 客户端接口
│   ├── cache/        # SQLite 缓存
│   ├── context/      # 敏感数据过滤
│   └── tracker/      # Token 使用追踪
├── zsh/              # ZSH 插件
│   └── llmsh.plugin.zsh
├── main.go           # 入口点
├── Makefile          # 构建和安装
└── install.sh        # 安装脚本
```

## 从源码构建

```bash
# 安装依赖
make deps

# 构建
make build

# 运行测试
make test

# 本地安装
make install

# 清理构建产物
make clean
```

## 运行测试

```bash
# 运行所有测试
go test -v ./...

# 运行特定包的测试
go test -v ./pkg/cache
go test -v ./pkg/llm
```

## 贡献

欢迎贡献！请随时提交 Pull Request。

1. Fork 本仓库
2. 创建你的功能分支（`git checkout -b feature/amazing-feature`）
3. 提交你的更改（`git commit -m 'Add some amazing feature'`）
4. 推送到分支（`git push origin feature/amazing-feature`）
5. 开启一个 Pull Request

## 许可证

MIT License - 详见 LICENSE 文件
