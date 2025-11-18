# llmsh Usage Documentation

llmsh is a ZSH plugin that provides intelligent command prediction, completion, and natural language to command conversion using LLMs.

## Table of Contents

- [Configuration](#configuration)
- [Subcommands](#subcommands)
- [Customization](#customization)
- [Supported LLM Providers](#supported-llm-providers)
- [Troubleshooting](#troubleshooting)
- [JSON Request/Response Format](#json-requestresponse-format)

---

## Configuration

### Initialize Configuration

Before using llmsh, initialize the configuration file:

```bash
llmsh config init
```

This creates a configuration file at `~/.llmsh/config.yaml` with default settings for:
- LLM providers (OpenAI, local/Ollama)
- Prediction settings
- Cache settings
- Token tracking
- ZSH keybindings

After initialization:
1. Set your OpenAI API key: `export OPENAI_API_KEY="your-api-key"`
2. Or configure a local LLM provider (like Ollama) by changing `default_provider` to `local` in the config
3. Load the ZSH plugin in your `~/.zshrc`

### View Configuration

Display the current configuration:

```bash
llmsh config show
```

### Edit Configuration

Edit configuration file at `~/.llmsh/config.yaml`:

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

## Subcommands

### config

Manage llmsh configuration.

#### Subcommands:

**`llmsh config init`**

Initialize the configuration file at `~/.llmsh/config.yaml`.

- Creates default configuration with OpenAI and local provider settings
- Sets up cache, tracking, and prediction parameters
- Provides next steps for setup

**`llmsh config show`**

Display the current configuration file contents.

---

### predict

Predict the next shell command based on context.

**Usage:**
```bash
echo '{"method":"predict","history":["git status","git add ."],"cwd":"/home/user/project","git_branch":"main","os_info":"Darwin"}' | llmsh predict
```

**Purpose:** Analyzes recent command history, current directory, and git branch to predict what command you'll likely execute next.

**Input (JSON via stdin):**
- `method`: "predict" (required)
- `history`: Array of recent shell commands
- `cwd`: Current working directory
- `git_branch`: Current git branch (if in a git repo)
- `os_info`: Operating system information

**Output (JSON to stdout):**
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

**Features:**
- Uses command history to understand workflow patterns
- Considers git context and working directory
- Caches predictions to reduce API calls
- Filters sensitive information (passwords, tokens) from history

---

### complete

Complete a partial command based on context.

**Usage:**
```bash
echo '{"method":"complete","prefix":"git co","history":["git status","git branch"],"cwd":"/home/user/project","os_info":"Darwin"}' | llmsh complete
```

**Purpose:** Completes a partially typed command based on context and recent history.

**Input (JSON via stdin):**
- `method`: "complete" (required)
- `prefix`: The partial command to complete (required)
- `history`: Array of recent shell commands
- `cwd`: Current working directory
- `os_info`: Operating system information

**Output (JSON to stdout):**
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

**Features:**
- Requires minimum prefix length (configurable via `prediction.min_prefix_length`)
- Uses recent command history for better context
- Returns practical, safe command completions

---

### nl2cmd

Convert natural language description to a shell command.

**Usage:**
```bash
echo '{"method":"nl2cmd","description":"list all files modified in the last 24 hours","history":["ls -la"],"cwd":"/home/user","os_info":"Darwin"}' | llmsh nl2cmd
```

**Purpose:** Translates natural language descriptions into executable shell commands.

**Input (JSON via stdin):**
- `method`: "nl2cmd" (required)
- `description`: Natural language description of the desired command (required)
- `history`: Array of recent shell commands (optional)
- `cwd`: Current working directory
- `os_info`: Operating system information

**Output (JSON to stdout):**
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

**Features:**
- Generates safe, practical commands
- Uses common Unix/Linux tools
- Considers OS and current directory context
- Shows last 3 commands from history for additional context

---

### stats

Display token usage statistics.

**Usage:**
```bash
llmsh stats
```

**Purpose:** Shows aggregated statistics about LLM token usage to help monitor costs and cache effectiveness.

**Output:** Displays statistics in three categories:

1. **Usage by Provider/Model:**
   - Provider and model name
   - Number of requests
   - Input and output tokens
   - Cache read tokens (if applicable)

2. **Usage by Day:**
   - Daily breakdown of token usage
   - Request counts per day
   - Input/output/cache tokens per day

3. **Usage by Method:**
   - Token usage per subcommand (predict, complete, nl2cmd)
   - Request counts per method

4. **Total Summary:**
   - Total requests and tokens across all usage
   - Cache savings percentage (if prompt caching is enabled)

**Example Output:**
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

**Data Location:** Token tracking data is stored in `~/.llmsh/tokens.json` (configurable via `tracking.db_path`).

---

### clean

Clean llmsh data files.

**Usage:**
```bash
# Clean logs and cache only
llmsh clean

# Clean logs, cache, and token tracking data
llmsh clean --all
llmsh clean -a
```

**Purpose:** Remove temporary files, cache, and optionally token tracking data to free up space or reset state.

**What Gets Cleaned:**

**Default (no flags):**
- Debug log file (`/tmp/llmsh_debug.log`)
- Cache database (`~/.llmsh/cache.db`)

**With `--all` or `-a` flag:**
- All of the above, plus:
- Token tracking data (`~/.llmsh/tokens.json`)

**Note:** The configuration file (`~/.llmsh/config.yaml`) is never removed by the clean command.

**Example Output:**
```
Cleaned:
  ✓ debug log
  ✓ cache database
  ✓ token tracking data
```

---

## Customization

### Keybindings

Edit the plugin file at `~/.llmsh/llmsh.plugin.zsh` to customize keybindings:

```zsh
# Natural language to command (default: Alt+Enter)
bindkey '^[^M' _llmsh_nl2cmd_widget

# Smart completion/prediction (default: Ctrl+O)
# - Empty buffer: predicts next command
# - Has text: completes current command
bindkey '^O' _llmsh_predict_next_widget
```

---

## Supported LLM Providers

### OpenAI
```yaml
providers:
  openai:
    api_key: ${OPENAI_API_KEY}
    model: gpt-4-turbo-preview
    base_url: https://api.openai.com/v1
```

### Local LLMs (Ollama)
```yaml
providers:
  local:
    model: llama2
    base_url: http://localhost:11434/v1
    api_key: "not-needed"
```

### Other OpenAI-Compatible APIs

Any OpenAI-compatible API endpoint can be used by setting the `base_url`:

```yaml
providers:
  custom:
    api_key: your-api-key
    model: your-model
    base_url: https://your-api.com/v1
```

---

## Troubleshooting

### Plugin not loading
```bash
# Check if plugin file exists
ls -l ~/.llmsh/llmsh.plugin.zsh

# Check if it's sourced in ~/.zshrc
grep llmsh.plugin.zsh ~/.zshrc

# Reload shell
source ~/.zshrc
```

### No predictions appearing
```bash
# Check if binary exists and is executable
which llmsh
llmsh config show

# Check if API key is set
echo $OPENAI_API_KEY

# Test manually
echo '{"method":"predict","history":["ls","pwd"],"cwd":"'$PWD'","os_info":"Darwin"}' | llmsh predict
```

### jq not found
```bash
# Install jq
brew install jq  # macOS
sudo apt-get install jq  # Linux
```

### zsh-autosuggestions conflicts

The plugin works with or without zsh-autosuggestions. If you have conflicts:

1. Make sure llmsh is sourced **after** zsh-autosuggestions in your `~/.zshrc`
2. Check keybinding conflicts: `bindkey | grep -E "(RIGHT|\\^\\[\\^M)"`

---

## JSON Request/Response Format

All prediction-related subcommands (predict, complete, nl2cmd) communicate via JSON through stdin/stdout, designed for integration with the ZSH plugin.

### Request Structure

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

**Fields:**
- `method`: The operation to perform (required)
- `history`: Recent shell commands (optional)
- `cwd`: Current working directory (optional but recommended)
- `git_branch`: Git branch name (optional)
- `os_info`: Operating system info (optional but recommended)
- `prefix`: Required for "complete" method
- `description`: Required for "nl2cmd" method
- `timestamp`: Unix timestamp (optional)

### Response Structure

**Success Response:**
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

**Error Response:**
```json
{
  "error": "error message describing what went wrong"
}
```

**Response Fields:**
- `result.command`: The predicted/completed/generated command
- `result.cached`: Whether the result was retrieved from cache
- `tokens`: Token usage information (only present when not cached)
- `error`: Error message (only present when an error occurs)

---

## Integration with ZSH

llmsh is designed to be used as part of a ZSH plugin. The binary handles the LLM interactions while the ZSH plugin provides:
- Keybindings for prediction and nl2cmd
- Context gathering (history, cwd, git branch)
- User interface for displaying suggestions

Refer to the ZSH plugin configuration in your `~/.llmsh/config.yaml` under `zsh.keybindings` to customize keyboard shortcuts.
