# Development Guide

English | [简体中文](DEVELOPMENT.zh-CN.md)

## Architecture

### Components

- **Go Binary** (`llmsh`): Core logic for LLM interaction, caching, and tracking
  - Commands: `predict`, `complete`, `nl2cmd`, `config`, `stats`, `clean`
  - JSON-based communication via stdin/stdout

- **ZSH Plugin** (`llmsh.plugin.zsh`): User interface and context gathering
  - Collects shell history, git context, working directory
  - Manages debouncing and cooldowns
  - Integrates with zsh-autosuggestions
  - Provides keybindings and widgets

### Data Flow

```
User Types → ZSH Plugin → Collect Context → llmsh Binary → LLM API
                ↓                                              ↓
            Display ← Parse Response ← Cache/Track ← JSON Response
```

### Files

- `~/.local/bin/llmsh` - Binary executable
- `~/.llmsh/llmsh.plugin.zsh` - ZSH plugin
- `~/.llmsh/config.yaml` - Configuration
- `~/.llmsh/cache.db` - SQLite cache database
- `~/.llmsh/tokens.json` - Token usage tracking
- `/tmp/llmsh_debug.log` - Debug logs (if enabled)

## Project Structure

```
llmsh/
├── cmd/              # Cobra CLI commands
│   ├── root.go       # Root command and JSON structures
│   ├── predict.go    # Next command prediction
│   ├── complete.go   # Command completion
│   ├── nl2cmd.go     # Natural language conversion
│   ├── config.go     # Configuration management
│   ├── stats.go      # Token usage statistics
│   └── clean.go      # Data cleanup
├── pkg/              # Core packages
│   ├── config/       # Configuration management
│   ├── llm/          # LLM client interface
│   ├── cache/        # SQLite cache
│   ├── context/      # Sensitive data filtering
│   └── tracker/      # Token usage tracking
├── zsh/              # ZSH plugin
│   └── llmsh.plugin.zsh
├── main.go           # Entry point
├── Makefile          # Build and install
└── install.sh        # Installation script
```

## Building from Source

```bash
# Install dependencies
make deps

# Build
make build

# Run tests
make test

# Install locally
make install

# Clean build artifacts
make clean
```

## Running Tests

```bash
# Run all tests
go test -v ./...

# Run specific package tests
go test -v ./pkg/cache
go test -v ./pkg/llm
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
