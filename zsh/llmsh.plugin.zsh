#!/usr/bin/env zsh
# llmsh ZSH Plugin
# Provides natural language to command conversion and manual command prediction

# ============================================================================
# Configuration and Initialization
# ============================================================================

# Binary path
LLMSH_BINARY="${LLMSH_BINARY:-${HOME}/.local/bin/llmsh}"

# Check if binary exists
if [[ ! -x "$LLMSH_BINARY" ]]; then
    echo "llmsh: binary not found at $LLMSH_BINARY" >&2
    echo "llmsh: Please run 'make install' to install the binary" >&2
    return 1
fi

# Check for required dependencies
if ! command -v jq &> /dev/null; then
    echo "llmsh: 'jq' is required but not installed" >&2
    echo "llmsh: Please install jq: brew install jq (macOS) or apt-get install jq (Linux)" >&2
    return 1
fi

# ============================================================================
# Helper Functions
# ============================================================================

# Get shell context for LLM
_llmsh_get_context() {
    local history_count=${1:-10}

    # Get recent history
    local history_json=$(fc -ln -${history_count} 2>/dev/null | sed 's/^[[:space:]]*//' | jq -R -s -c 'split("\n") | map(select(length > 0))')

    # Get git branch
    local git_branch=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")

    # Get OS info
    local os_info=$(uname -s 2>/dev/null || echo "")

    # Current working directory
    local cwd="$PWD"

    # Return as JSON object (without outer braces, for merging)
    echo "\"history\":${history_json},\"cwd\":\"${cwd}\",\"git_branch\":\"${git_branch}\",\"os_info\":\"${os_info}\""
}

# Call llmsh binary with JSON request
_llmsh_call_binary() {
    local method="$1"
    local extra_json="$2"  # Additional JSON fields

    # Build JSON request
    local context=$(_llmsh_get_context 10)
    local request="{\"method\":\"${method}\",${context}${extra_json}}"

    # Call binary and return response
    echo "$request" | "$LLMSH_BINARY" "$method" 2>/dev/null
}

# Extract command from JSON response
_llmsh_extract_command() {
    local response="$1"

    # Check for error
    local error=$(echo "$response" | jq -r '.error // empty' 2>/dev/null)
    if [[ -n "$error" ]]; then
        return 1
    fi

    # Extract command
    local command=$(echo "$response" | jq -r '.result.command // empty' 2>/dev/null)
    if [[ -z "$command" ]]; then
        return 1
    fi

    echo "$command"
    return 0
}

# ============================================================================
# Natural Language to Command Widget
# ============================================================================

_llmsh_nl2cmd_widget() {
    # Save current buffer as the natural language description
    local description="$BUFFER"

    # Don't proceed if buffer is empty
    if [[ -z "$description" ]]; then
        return
    fi

    # Clear the buffer and show loading indicator
    BUFFER=""
    POSTDISPLAY=' [Converting...]'
    # Highlight only POSTDISPLAY (from end of BUFFER to end of BUFFER+POSTDISPLAY)
    region_highlight=("$#BUFFER $(($#BUFFER + $#POSTDISPLAY)) fg=cyan")
    zle -R

    # Call nl2cmd
    local response=$(_llmsh_call_binary "nl2cmd" ",\"description\":\"${description}\"")
    local command=$(_llmsh_extract_command "$response")

    # Clear loading indicator
    POSTDISPLAY=""

    if [[ -n "$command" ]]; then
        # Set buffer to the generated command
        BUFFER="$command"
        CURSOR=$#BUFFER
    else
        # Restore original buffer on error
        BUFFER="$description"
        CURSOR=$#BUFFER

        # Show error briefly
        POSTDISPLAY=" [Conversion failed]"
        zle -R
        sleep 1
        POSTDISPLAY=""
    fi

    # Clear region highlighting to fix color issues
    region_highlight=()

    # Clear zsh-autosuggestions if present
    if (( ${+functions[_zsh_autosuggest_clear]} )); then
        _zsh_autosuggest_clear
    fi

    zle -R
}

# ============================================================================
# Next Command Prediction Widget (Optional - for manual trigger)
# ============================================================================

_llmsh_predict_next_widget() {
    # Save current buffer
    local current_buffer="$BUFFER"

    # Determine which method to use
    local method
    local extra_json=""
    local loading_msg

    if [[ -z "$current_buffer" ]]; then
        # Buffer is empty: predict next command
        method="predict"
        loading_msg='[Predicting next command...]'
    else
        # Buffer has content: complete current command
        method="complete"
        extra_json=",\"prefix\":\"${current_buffer}\""
        loading_msg='[Completing...]'
    fi

    # Show loading indicator
    POSTDISPLAY="$loading_msg"
    # Highlight only POSTDISPLAY (from end of BUFFER to end of BUFFER+POSTDISPLAY)
    region_highlight=("$#BUFFER $(($#BUFFER + $#POSTDISPLAY)) fg=cyan")
    zle -R

    # Call appropriate method
    local response=$(_llmsh_call_binary "$method" "$extra_json")
    local command=$(_llmsh_extract_command "$response")

    # Clear loading indicator
    POSTDISPLAY=""

    if [[ -n "$command" ]]; then
        # Set buffer to the result
        BUFFER="$command"
        CURSOR=$#BUFFER
    else
        # Restore original buffer on error
        BUFFER="$current_buffer"
        CURSOR=$#BUFFER
    fi

    # Clear region highlighting to fix color issues
    region_highlight=()

    # Clear zsh-autosuggestions if present
    if (( ${+functions[_zsh_autosuggest_clear]} )); then
        _zsh_autosuggest_clear
    fi

    zle -R
}

# ============================================================================
# Widget Registration and Keybindings
# ============================================================================

# Register ZLE widgets
zle -N _llmsh_nl2cmd_widget
zle -N _llmsh_predict_next_widget

# Keybindings
# Alt+Enter: Natural language to command
bindkey '^[^M' _llmsh_nl2cmd_widget

# Optional: Ctrl+O for manual next command prediction
bindkey '^O' _llmsh_predict_next_widget

