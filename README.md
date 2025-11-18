# llmsh

An intelligent ZSH plugin that provides AI-powered command prediction and natural language to command conversion using LLMs.

> 📖 For detailed usage and troubleshooting, see [USAGE.md](USAGE.md)  
> 👨‍💻 For development and contributing, see [DEVELOPMENT.md](DEVELOPMENT.md)

## Features

### Natural Language to Command
- **Alt+Enter**: Convert natural language descriptions into shell commands
- Type what you want to do in plain English, press Alt+Enter, and get the command
- Example: Type "list all files modified in the last 24 hours" → `find . -type f -mtime -1`

### Smart Completion/Prediction
- **Ctrl+O**: Intelligently completes or predicts based on context
  - When buffer is **empty**: Predicts next command based on history
  - When buffer **has text**: Completes the current partial command
- Uses command history, current directory, and git branch for context

### Usage Tracking
- Track token usage by provider, model, method, and day
- Monitor cache effectiveness and cost savings
- View statistics with `llmsh stats`

### Performance
- **Intelligent caching**: Reduces API calls for repeated contexts
- **Fast response**: Built in Go for minimal overhead
- **No interruption**: All operations run on-demand via keybindings

## Installation

### Prerequisites

- **Go**: 1.21 or later
- **zsh**: Your shell must be zsh
- **jq**: For JSON processing (`brew install jq` on macOS)
- **Optional**: [zsh-autosuggestions](https://github.com/zsh-users/zsh-autosuggestions) for enhanced display

### Quick Install

```bash
# Clone the repository
git clone https://github.com/leslieleung/llmsh.git
cd llmsh

# Run the install script
./install.sh
```

### Manual Install

```bash
# Install dependencies and build
make deps
make build

# Install binary and plugin
make install
```

### Setup

1. **Initialize configuration**:
   ```bash
   llmsh config init
   ```

2. **Set your API key**:
   ```bash
   # Option 1: Environment variable
   export OPENAI_API_KEY="your-api-key"

   # Option 2: Edit config file
   # Edit ~/.llmsh/config.yaml and set your API key
   ```

3. **Load the plugin** (if not already added):
   ```bash
   # Add to ~/.zshrc
   echo 'source ~/.llmsh/llmsh.plugin.zsh' >> ~/.zshrc

   # Reload shell
   source ~/.zshrc
   ```

## Quick Start

### Natural Language Conversion

Type a description of what you want to do and press **Alt+Enter**:

```bash
$ find all jpg files larger than 1MB
# Press Alt+Enter
$ find . -type f -name "*.jpg" -size +1M
```

### Smart Completion/Prediction

Press **Ctrl+O** for intelligent completion or prediction:

**When buffer is empty** (predicts next command):
```bash
$ git add .
$ git commit -m "update docs"
# Press Ctrl+O on empty line
$ git push
```

**When buffer has text** (completes current command):
```bash
$ git co
# Press Ctrl+O
$ git checkout main
```

## Uninstallation

```bash
# Using install script
./install.sh uninstall

# Or manually
make uninstall
rm -rf ~/.llmsh  # Remove all data
# Remove "source ~/.llmsh/llmsh.plugin.zsh" from ~/.zshrc
```

## Credits

Inspired by [zsh-copilot](https://github.com/Myzel394/zsh-copilot) and built with modern Go practices.

