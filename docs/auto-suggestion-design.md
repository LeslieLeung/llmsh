# Auto-Suggestion with Debounce - Technical Design Document

**Version:** 1.0
**Date:** 2024-11-19
**Author:** Claude
**Status:** Draft

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Requirements](#requirements)
3. [System Architecture](#system-architecture)
4. [Detailed Design](#detailed-design)
5. [Implementation Plan](#implementation-plan)
6. [Testing Strategy](#testing-strategy)
7. [Performance Considerations](#performance-considerations)
8. [Security Considerations](#security-considerations)
9. [Deployment Plan](#deployment-plan)
10. [Risk Assessment](#risk-assessment)
11. [Future Enhancements](#future-enhancements)

---

## Executive Summary

### Goal
Implement an intelligent, non-intrusive command auto-suggestion feature with debouncing mechanism for llmsh, similar to zsh-autosuggestions but powered by LLM.

### Key Features
- **Silent Suggestions**: Display LLM-powered command suggestions in gray text without interrupting user input
- **Debounce Mechanism**: Prevent excessive LLM API calls by implementing 500ms debounce delay
- **Simple Keybindings**:
  - `Right Arrow (→)`: Accept suggestion
  - `ESC`: Clear suggestion
- **Cache-First**: Leverage existing SQLite cache to minimize API costs
- **Configurable**: Allow users to enable/disable and customize behavior

### Success Metrics
- ✅ No more than 1 LLM API call per 500ms of user typing
- ✅ Suggestions appear within 200ms for cached results
- ✅ Zero impact on terminal responsiveness
- ✅ Cache hit rate > 60% for common workflows

---

## Requirements

### Functional Requirements

#### FR-1: Debounce Input
- **Priority**: P0 (Critical)
- **Description**: System must wait for 500ms of user inactivity before triggering LLM request
- **Acceptance Criteria**:
  - User typing continuously should NOT trigger multiple API calls
  - Timer resets on each keystroke
  - Configurable delay via `prediction.debounce_delay_ms` config

#### FR-2: Display Silent Suggestions
- **Priority**: P0 (Critical)
- **Description**: Show LLM-generated suggestions as gray text after user's current input
- **Acceptance Criteria**:
  - Suggestion appears in POSTDISPLAY area
  - Text color is gray (fg=8)
  - Does not interfere with user typing
  - Automatically clears when user continues typing different content

#### FR-3: Accept Suggestion
- **Priority**: P0 (Critical)
- **Description**: User can accept suggestion by pressing Right Arrow key
- **Acceptance Criteria**:
  - Right Arrow key inserts suggestion into BUFFER
  - Cursor moves to end of accepted text
  - Suggestion clears after acceptance
  - Works in emacs and vi insert modes

#### FR-4: Clear Suggestion
- **Priority**: P0 (Critical)
- **Description**: User can dismiss suggestion by pressing ESC
- **Acceptance Criteria**:
  - ESC key clears POSTDISPLAY and region_highlight
  - Does not affect user's current BUFFER
  - Visual feedback is immediate

#### FR-5: Minimum Prefix Length
- **Priority**: P1 (High)
- **Description**: Only trigger suggestions when user has typed minimum characters
- **Acceptance Criteria**:
  - Default minimum length: 3 characters
  - Configurable via `prediction.min_prefix_length`
  - Empty buffer should not trigger suggestions

#### FR-6: Async LLM Calls
- **Priority**: P1 (High)
- **Description**: LLM API calls must not block terminal input
- **Acceptance Criteria**:
  - Terminal remains responsive during API calls
  - User can continue typing while request is in-flight
  - In-flight requests are cancelled if user modifies input

#### FR-7: Configuration Options
- **Priority**: P1 (High)
- **Description**: Expose configuration for user customization
- **Configuration Keys**:
  ```yaml
  prediction:
    auto_suggest: true                # Enable/disable feature
    debounce_delay_ms: 500            # Debounce delay
    min_prefix_length: 3              # Minimum chars to trigger
    max_suggestion_length: 150        # Max suggestion length
    show_loading_indicator: false     # Show "..." while loading
  ```

### Non-Functional Requirements

#### NFR-1: Performance
- Suggestions for cached commands: < 50ms
- Suggestions for new commands: < 2s (LLM latency)
- Memory overhead: < 10MB additional
- CPU overhead: < 5% when idle

#### NFR-2: Reliability
- Gracefully handle LLM API errors (show nothing, don't crash)
- Work offline when cache is available
- Maintain terminal stability (no freezes or crashes)

#### NFR-3: Compatibility
- Support ZSH 5.0+
- Work alongside existing widgets (zsh-autosuggestions, etc.)
- Support both emacs and vi keymaps
- Cross-platform: Linux, macOS

#### NFR-4: Security
- Filter sensitive patterns (password, token, secret, key)
- Do not send sensitive commands to LLM
- Respect existing sensitive filtering in `pkg/context/filter.go`

---

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         User Terminal                        │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  ZSH Session                                           │ │
│  │  ┌──────────────────────────────────────────────────┐ │ │
│  │  │  ZLE (Zsh Line Editor)                           │ │ │
│  │  │  ┌────────────────────────────────────────────┐  │ │ │
│  │  │  │ BUFFER: "git comm"                         │  │ │ │
│  │  │  │ POSTDISPLAY: "it -m ''" [gray]             │  │ │ │
│  │  │  └────────────────────────────────────────────┘  │ │ │
│  │  └──────────────────────────────────────────────────┘ │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ (keypress events)
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              llmsh.plugin.zsh (ZSH Plugin Layer)             │
│                                                              │
│  ┌──────────────────┐      ┌────────────────────────────┐  │
│  │ Event Handler    │      │  Debounce Timer            │  │
│  │ _llmsh_on_change │─────▶│  (500ms delay)             │  │
│  └──────────────────┘      └────────────────────────────┘  │
│           │                            │                    │
│           │ (buffer changed)           │ (timer fired)      │
│           ▼                            ▼                    │
│  ┌──────────────────┐      ┌────────────────────────────┐  │
│  │ Reset Timer      │      │  _llmsh_fetch_suggestion   │  │
│  └──────────────────┘      └────────────────────────────┘  │
│                                        │                    │
│                                        │ (async call)       │
│                                        ▼                    │
│                            ┌────────────────────────────┐  │
│                            │ _llmsh_call_binary_async   │  │
│                            │  (background process)      │  │
│                            └────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                                        │
                                        │ (JSON request)
                                        ▼
┌─────────────────────────────────────────────────────────────┐
│                   llmsh Binary (Go Layer)                    │
│                                                              │
│  ┌──────────────────┐      ┌────────────────────────────┐  │
│  │ cmd/complete.go  │─────▶│  Cache Check               │  │
│  │ runComplete()    │      │  (SQLite)                  │  │
│  └──────────────────┘      └────────────────────────────┘  │
│           │                            │                    │
│           │ (cache miss)               │ (cache hit)        │
│           ▼                            ▼                    │
│  ┌──────────────────┐      ┌────────────────────────────┐  │
│  │ pkg/llm/client   │      │  Return Cached Result      │  │
│  │ Complete()       │      └────────────────────────────┘  │
│  └──────────────────┘                                       │
│           │                                                 │
│           │ (LLM API call)                                  │
│           ▼                                                 │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         OpenAI API / Compatible Endpoint             │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                            │
                            │ (response)
                            ▼
┌─────────────────────────────────────────────────────────────┐
│              Callback: _llmsh_display_suggestion             │
│                                                              │
│  1. Extract command from JSON response                      │
│  2. Remove prefix that user already typed                   │
│  3. Set POSTDISPLAY with remaining suffix                   │
│  4. Apply gray color via region_highlight                   │
│  5. Call zle -R to refresh display                          │
└─────────────────────────────────────────────────────────────┘
```

### Component Breakdown

#### 1. ZSH Plugin Layer (`zsh/llmsh.plugin.zsh`)

**Responsibilities:**
- Listen to ZLE events (buffer changes)
- Manage debounce timer
- Display suggestions via POSTDISPLAY
- Handle keybindings (Right Arrow, ESC)
- Call Go binary asynchronously

**Key Functions:**
```zsh
_llmsh_on_buffer_change()       # Hook triggered on every keystroke
_llmsh_debounced_suggest()      # Debounce logic with timer
_llmsh_fetch_suggestion()       # Trigger async LLM call
_llmsh_call_binary_async()      # Execute Go binary in background
_llmsh_display_suggestion()     # Show suggestion in POSTDISPLAY
_llmsh_accept_suggestion()      # Right Arrow handler
_llmsh_clear_suggestion()       # ESC handler
_llmsh_should_suggest()         # Filter logic (min length, sensitive words)
```

**State Variables:**
```zsh
LLMSH_DEBOUNCE_TIMER_PID        # PID of background timer process
LLMSH_LAST_BUFFER               # Last buffer value (for change detection)
LLMSH_CURRENT_SUGGESTION        # Currently displayed suggestion
LLMSH_INFLIGHT_REQUEST          # PID of in-flight LLM request
```

#### 2. Go Binary Layer (`cmd/complete.go`)

**Responsibilities:**
- Receive JSON request from stdin
- Check cache for existing suggestion
- Call LLM API if cache miss
- Return JSON response to stdout

**Request Format:**
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

**Response Format:**
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

#### 3. Configuration Layer (`pkg/config/config.go`)

**New Config Struct:**
```go
type PredictionConfig struct {
    HistoryLength          int  `mapstructure:"history_length"`
    MinPrefixLength        int  `mapstructure:"min_prefix_length"`

    // New fields for auto-suggestion
    AutoSuggest            bool `mapstructure:"auto_suggest"`
    DebounceDelayMs        int  `mapstructure:"debounce_delay_ms"`
    MaxSuggestionLength    int  `mapstructure:"max_suggestion_length"`
    ShowLoadingIndicator   bool `mapstructure:"show_loading_indicator"`
}
```

---

## Detailed Design

### 1. Debounce Mechanism

#### Implementation Strategy

**Option A: Pure ZSH Timer (Chosen)**

Pros:
- No external dependencies
- Built into ZSH
- Simple to implement

Cons:
- Less precise timing
- Requires background processes

**Implementation:**
```zsh
_llmsh_debounced_suggest() {
    # Kill previous timer if exists
    if [[ -n "$LLMSH_DEBOUNCE_TIMER_PID" ]]; then
        kill $LLMSH_DEBOUNCE_TIMER_PID 2>/dev/null
        LLMSH_DEBOUNCE_TIMER_PID=""
    fi

    # Start new background timer
    {
        sleep ${LLMSH_DEBOUNCE_DELAY:-0.5}
        _llmsh_fetch_suggestion
    } &
    LLMSH_DEBOUNCE_TIMER_PID=$!
}
```

#### Debounce Flow

```
Time (ms)    0      100    200    300    400    500    600    700    800
User Input   g      i      t      _      c      o      [stop]
             │      │      │      │      │      │      │
Timer        ├─X    ├─X    ├─X    ├─X    ├─X    ├─X    ├──────────────┤
             │      │      │      │      │      │                     │
             Reset  Reset  Reset  Reset  Reset  Reset                Fire!
                                                                       │
                                                                       ▼
                                                              LLM Request
```

### 2. Async LLM Call

#### Background Process Approach

```zsh
_llmsh_call_binary_async() {
    local method="$1"
    local prefix="$2"

    # Build JSON request
    local context=$(_llmsh_get_context 10)
    local request="{\"method\":\"${method}\",${context},\"prefix\":\"${prefix}\"}"

    # Create temporary FIFO for response
    local fifo="/tmp/llmsh_response_$$"
    mkfifo "$fifo"

    # Background job: call binary and write to FIFO
    {
        local response=$(echo "$request" | "$LLMSH_BINARY" "$method" 2>/dev/null)
        echo "$response" > "$fifo"
    } &
    LLMSH_INFLIGHT_REQUEST=$!

    # Use zle -F to register file descriptor handler (non-blocking read)
    exec {fd}<"$fifo"
    zle -F $fd _llmsh_handle_response
}

_llmsh_handle_response() {
    local fd=$1

    # Read response from file descriptor
    local response
    read -r response <&$fd

    # Close file descriptor and cleanup
    exec {fd}<&-
    rm -f "/tmp/llmsh_response_$$"

    # Display suggestion
    _llmsh_display_suggestion "$response"
}
```

#### Request Cancellation

```zsh
_llmsh_cancel_inflight() {
    if [[ -n "$LLMSH_INFLIGHT_REQUEST" ]]; then
        kill $LLMSH_INFLIGHT_REQUEST 2>/dev/null
        LLMSH_INFLIGHT_REQUEST=""
    fi
}

# Cancel when user changes buffer
_llmsh_on_buffer_change() {
    if [[ "$BUFFER" != "$LLMSH_LAST_BUFFER" ]]; then
        _llmsh_cancel_inflight
        _llmsh_clear_suggestion
        _llmsh_debounced_suggest
    fi
}
```

### 3. Suggestion Display

#### Visual Rendering

```zsh
_llmsh_display_suggestion() {
    local response="$1"

    # Extract command from JSON
    local full_command=$(echo "$response" | jq -r '.result.command // empty')

    # Validate
    if [[ -z "$full_command" ]] || [[ "$full_command" == "$BUFFER" ]]; then
        return
    fi

    # Extract suffix (remove already-typed prefix)
    local suffix="${full_command#$BUFFER}"

    # Truncate if too long
    local max_len=${LLMSH_MAX_SUGGESTION_LENGTH:-150}
    if [[ ${#suffix} -gt $max_len ]]; then
        suffix="${suffix:0:$max_len}..."
    fi

    # Set display variables
    LLMSH_CURRENT_SUGGESTION="$suffix"
    POSTDISPLAY="$suffix"

    # Apply gray color (fg=8)
    local start=$#BUFFER
    local end=$(($start + $#POSTDISPLAY))
    region_highlight+=("$start $end fg=8")

    # Refresh display
    zle -R
}
```

#### Color Scheme

| State | Color | ZSH Code |
|-------|-------|----------|
| Normal suggestion | Gray | `fg=8` |
| Loading indicator (optional) | Cyan | `fg=cyan` |
| Cached result (optional) | Light gray | `fg=8,italic` |

### 4. Keybindings

#### Right Arrow: Accept Suggestion

```zsh
_llmsh_accept_suggestion() {
    if [[ -n "$LLMSH_CURRENT_SUGGESTION" ]]; then
        # Append suggestion to buffer
        BUFFER="${BUFFER}${LLMSH_CURRENT_SUGGESTION}"
        CURSOR=$#BUFFER

        # Clear suggestion state
        _llmsh_clear_suggestion

        # Trigger new suggestion for the completed command
        _llmsh_debounced_suggest
    else
        # No suggestion: fallback to default Right Arrow behavior (forward-char)
        zle forward-char
    fi

    zle -R
}

# Register widget
zle -N _llmsh_accept_suggestion
bindkey '^[[C' _llmsh_accept_suggestion  # Right Arrow
```

#### ESC: Clear Suggestion

```zsh
_llmsh_clear_suggestion() {
    POSTDISPLAY=""
    LLMSH_CURRENT_SUGGESTION=""
    region_highlight=()
    zle -R
}

# ESC key
bindkey '^[' _llmsh_clear_suggestion
```

### 5. Smart Filtering

#### When NOT to Suggest

```zsh
_llmsh_should_suggest() {
    local buffer="$BUFFER"

    # 1. Feature disabled
    if [[ "${LLMSH_AUTO_SUGGEST:-true}" != "true" ]]; then
        return 1
    fi

    # 2. Buffer too short
    local min_len=${LLMSH_MIN_PREFIX_LENGTH:-3}
    if [[ ${#buffer} -lt $min_len ]]; then
        return 1
    fi

    # 3. In vi command mode
    if [[ "$KEYMAP" == "vicmd" ]]; then
        return 1
    fi

    # 4. In completion menu
    if [[ -n "$MENUSELECT" ]]; then
        return 1
    fi

    # 5. Sensitive keywords
    local sensitive_pattern='(password|passwd|token|secret|key|apikey|api_key)='
    if [[ "$buffer" =~ $sensitive_pattern ]]; then
        return 1
    fi

    # 6. Already has a suggestion for same prefix
    if [[ -n "$LLMSH_CURRENT_SUGGESTION" ]] && [[ "$buffer" == "$LLMSH_LAST_SUGGESTED_PREFIX" ]]; then
        return 1
    fi

    return 0
}
```

### 6. Event Hooks

#### Buffer Change Detection

```zsh
# Hook into ZLE widgets
_llmsh_zle_line_init() {
    # Install buffer change hook
    zle -N zle-line-pre-redraw _llmsh_on_buffer_change
}

_llmsh_on_buffer_change() {
    # Check if buffer actually changed
    if [[ "$BUFFER" != "$LLMSH_LAST_BUFFER" ]]; then
        LLMSH_LAST_BUFFER="$BUFFER"

        # Cancel any in-flight requests
        _llmsh_cancel_inflight

        # Clear current suggestion
        _llmsh_clear_suggestion

        # Check if we should suggest
        if _llmsh_should_suggest; then
            _llmsh_debounced_suggest
        fi
    fi
}

# Register hooks
zle -N zle-line-init _llmsh_zle_line_init
```

---

## Implementation Plan

### Phase 1: Core Functionality (Week 1)

**Tasks:**
1. ✅ Extend `PredictionConfig` struct in `pkg/config/config.go`
2. ✅ Update `cmd/config.go` to set default values
3. ✅ Implement debounce timer in `zsh/llmsh.plugin.zsh`
4. ✅ Implement `_llmsh_display_suggestion()` function
5. ✅ Implement `_llmsh_accept_suggestion()` (Right Arrow)
6. ✅ Implement `_llmsh_clear_suggestion()` (ESC)
7. ✅ Add buffer change detection hook
8. ✅ Add `_llmsh_should_suggest()` filtering logic

**Files Modified:**
- `pkg/config/config.go` (+8 lines)
- `cmd/config.go` (+4 lines)
- `zsh/llmsh.plugin.zsh` (+120 lines)

**Deliverables:**
- Basic auto-suggestion works
- Debounce prevents excessive calls
- Keybindings functional

### Phase 2: Async & Performance (Week 2)

**Tasks:**
1. ✅ Implement async binary call with FIFO
2. ✅ Add request cancellation logic
3. ✅ Optimize cache key generation
4. ✅ Add loading indicator (optional)
5. ✅ Performance testing & tuning

**Files Modified:**
- `zsh/llmsh.plugin.zsh` (+60 lines)
- `cmd/complete.go` (optimize cache lookup)

**Deliverables:**
- Non-blocking LLM calls
- In-flight requests can be cancelled
- Cache hit rate logged

### Phase 3: Polish & Testing (Week 3)

**Tasks:**
1. ✅ Integration testing with different shells
2. ✅ Compatibility testing (oh-my-zsh, Prezto)
3. ✅ Documentation updates (USAGE.md, README.md)
4. ✅ Add configuration examples
5. ✅ Fix edge cases and bugs

**Files Modified:**
- `USAGE.md` (add auto-suggestion section)
- `README.md` (mention new feature)
- `zsh/llmsh.plugin.zsh` (bug fixes)

**Deliverables:**
- Comprehensive test coverage
- Updated documentation
- Ready for beta release

---

## Testing Strategy

### Unit Tests

#### ZSH Unit Tests (using zunit or manual)

```zsh
# Test: Debounce timer resets on keystroke
test_debounce_reset() {
    _llmsh_debounced_suggest
    local first_pid=$LLMSH_DEBOUNCE_TIMER_PID

    sleep 0.1
    _llmsh_debounced_suggest
    local second_pid=$LLMSH_DEBOUNCE_TIMER_PID

    # Timer should be reset (different PID)
    assert_not_equal "$first_pid" "$second_pid"
}

# Test: Minimum prefix length
test_min_prefix_length() {
    LLMSH_MIN_PREFIX_LENGTH=3
    BUFFER="ab"

    if _llmsh_should_suggest; then
        fail "Should not suggest for buffer < min length"
    fi

    BUFFER="abc"
    if ! _llmsh_should_suggest; then
        fail "Should suggest for buffer >= min length"
    fi
}

# Test: Sensitive keyword filtering
test_sensitive_filtering() {
    BUFFER="export PASSWORD=secret"

    if _llmsh_should_suggest; then
        fail "Should not suggest for sensitive content"
    fi
}
```

#### Go Unit Tests

```go
// Test: Complete with minimum prefix
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

// Test: Cache key generation consistency
func TestCacheKeyGeneration(t *testing.T) {
    history := []string{"ls", "cd project"}
    key1 := generateCacheKey(history, "/home/user", "main")
    key2 := generateCacheKey(history, "/home/user", "main")

    assert.Equal(t, key1, key2)
}
```

### Integration Tests

#### End-to-End Test Script

```bash
#!/bin/bash
# test_auto_suggestion.sh

# Setup
source zsh/llmsh.plugin.zsh
export LLMSH_AUTO_SUGGEST=true
export LLMSH_DEBOUNCE_DELAY=0.1  # Faster for testing

# Test 1: Suggestion appears after delay
echo "Test 1: Suggestion appears"
BUFFER="git sta"
_llmsh_debounced_suggest
sleep 0.2

if [[ -z "$POSTDISPLAY" ]]; then
    echo "FAIL: No suggestion appeared"
    exit 1
fi
echo "PASS: Suggestion = '$POSTDISPLAY'"

# Test 2: Accept suggestion
echo "Test 2: Accept suggestion"
BUFFER="git sta"
LLMSH_CURRENT_SUGGESTION="tus"
_llmsh_accept_suggestion

if [[ "$BUFFER" != "git status" ]]; then
    echo "FAIL: Buffer = '$BUFFER', expected 'git status'"
    exit 1
fi
echo "PASS: Suggestion accepted"

# Test 3: Clear suggestion
echo "Test 3: Clear suggestion"
POSTDISPLAY="some suggestion"
_llmsh_clear_suggestion

if [[ -n "$POSTDISPLAY" ]]; then
    echo "FAIL: Suggestion not cleared"
    exit 1
fi
echo "PASS: Suggestion cleared"

echo "All tests passed!"
```

### Manual Testing Checklist

- [ ] Suggestions appear after typing 3+ characters
- [ ] Suggestions disappear when typing continues
- [ ] Right Arrow accepts suggestion correctly
- [ ] ESC clears suggestion
- [ ] No LLM calls for buffer < 3 chars
- [ ] No LLM calls for sensitive keywords (password, token)
- [ ] Cache hits return instantly
- [ ] Terminal remains responsive during LLM calls
- [ ] Works with oh-my-zsh themes
- [ ] Works with zsh-syntax-highlighting
- [ ] Works in vi mode (insert only)
- [ ] Config changes take effect after plugin reload

---

## Performance Considerations

### Latency Targets

| Scenario | Target | Acceptable |
|----------|--------|------------|
| Cache hit | < 50ms | < 100ms |
| Cache miss (LLM) | < 1000ms | < 2000ms |
| Debounce delay | 500ms | 300-800ms |
| UI refresh | < 16ms | < 50ms |

### Memory Usage

**Baseline:** ~5MB (current llmsh plugin)

**Expected with auto-suggest:** ~8MB
- +1MB for FIFO buffers
- +1MB for suggestion cache (in-memory)
- +1MB for additional ZSH state

**Mitigation:**
- Limit suggestion length (150 chars default)
- Clear old suggestions from memory
- Reuse FIFO files

### CPU Usage

**Target:** < 1% CPU when idle, < 10% during suggestion generation

**Optimization:**
- Use background processes for LLM calls (no blocking)
- Efficient string manipulation (avoid repeated `jq` calls)
- Cache command validation results

### Network Bandwidth

**Assumptions:**
- Average prompt: 500 tokens (~2KB)
- Average completion: 20 tokens (~100 bytes)

**Estimated usage:**
- 10 suggestions/minute = ~20KB/min
- 600 suggestions/hour = ~1.2MB/hour

**Mitigation:**
- High cache hit rate (> 60%)
- Limit max tokens in LLM config (100 tokens)

---

## Security Considerations

### 1. Sensitive Data Filtering

**Threat:** User types sensitive data (passwords, API keys) that gets sent to LLM

**Mitigation:**
```zsh
# Regex patterns for sensitive content
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
            return 0  # Contains sensitive data
        fi
    done
    return 1
}
```

**Existing Protection:** `pkg/context/filter.go` already filters history, extend to real-time buffer

### 2. Command Injection

**Threat:** Malicious LLM response contains command injection

**Mitigation:**
- Display suggestions only in POSTDISPLAY (not executed automatically)
- User must explicitly accept (Right Arrow)
- Validate response format (JSON schema)

### 3. API Key Exposure

**Threat:** API key leaked in logs or error messages

**Mitigation:**
- Already handled by existing config (env vars)
- Ensure error messages don't include API key
- Use HTTPS for all API calls

### 4. Cache Poisoning

**Threat:** Malicious cache entries lead to dangerous suggestions

**Mitigation:**
- Cache is local SQLite (not shared)
- TTL expires old entries (7 days default)
- Hash-based cache keys (collision-resistant)

---

## Deployment Plan

### Pre-Deployment

1. **Config Migration**
   ```bash
   # Backup existing config
   cp ~/.llmsh/config.yaml ~/.llmsh/config.yaml.backup

   # Run config update (add new fields)
   llmsh config migrate
   ```

2. **Plugin Update**
   ```bash
   cd /path/to/llmsh
   git pull origin main
   make install
   ```

3. **Reload ZSH Plugin**
   ```bash
   # Add to ~/.zshrc or reload manually
   source ~/.zsh/plugins/llmsh/llmsh.plugin.zsh
   ```

### Post-Deployment

1. **Verify Installation**
   ```bash
   llmsh config show | grep auto_suggest
   # Should output: auto_suggest: true
   ```

2. **Test Suggestion**
   ```bash
   # Type "git sta" and wait 500ms
   # Gray text should appear: "tus" or "status"
   ```

3. **Monitor Performance**
   ```bash
   llmsh stats --last-hour
   # Check cache hit rate, API call count
   ```

### Rollback Plan

If issues arise:

```bash
# 1. Disable auto-suggest
echo "prediction:\n  auto_suggest: false" >> ~/.llmsh/config.yaml

# 2. Reload plugin
source ~/.zsh/plugins/llmsh/llmsh.plugin.zsh

# 3. Revert to previous version
cd /path/to/llmsh
git checkout v1.0.0  # Previous stable version
make install
```

---

## Risk Assessment

### High Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| **Terminal freezes during LLM call** | Medium | Critical | Implement async calls with timeout |
| **Excessive API costs** | High | High | Enforce cache, debounce, min prefix length |
| **Conflicts with zsh-autosuggestions** | Medium | Medium | Detect and disable conflicting plugins |
| **Sensitive data leakage** | Low | Critical | Robust filtering + user opt-in |

### Medium Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| **Poor cache hit rate** | Medium | Medium | Improve cache key design, longer TTL |
| **ZSH version incompatibility** | Low | Medium | Test on ZSH 5.0-5.9, document requirements |
| **High memory usage** | Low | Low | Limit suggestion length, cleanup old state |

### Low Risks

| Risk | Probability | Impact | Mitigation |
|------|------------|--------|------------|
| **Keybinding conflicts** | Low | Low | Document conflicts, allow custom bindings |
| **Visual glitches with themes** | Low | Low | Test popular themes (oh-my-zsh, Powerlevel10k) |

---

## Future Enhancements

### Phase 4: Advanced Features (Post-MVP)

1. **Partial Acceptance**
   - `Ctrl+Right`: Accept one word at a time
   - Useful for long suggestions

2. **Context-Aware Suggestions**
   - Analyze git status to suggest relevant commands
   - Detect file types to suggest appropriate tools

3. **Multi-Line Suggestions**
   - Suggest command chains (pipelines)
   - Display on multiple lines

4. **Learning from Accepted Suggestions**
   - Track which suggestions user accepts
   - Bias LLM towards accepted patterns
   - Local fine-tuning (optional)

5. **Suggestion Ranking**
   - Show multiple suggestions (numbered)
   - User selects with `Alt+1`, `Alt+2`, etc.

6. **Offline Mode**
   - Preload common suggestions
   - Fallback to regex-based completion
   - Sync when network available

7. **Analytics Dashboard**
   - Web UI showing:
     - Cache hit rate over time
     - Most expensive LLM calls
     - Suggestion acceptance rate
     - Cost savings from cache

8. **Collaborative Filtering**
   - Share anonymized suggestions (opt-in)
   - Benefit from other users' patterns
   - Privacy-preserving aggregation

---

## Appendix

### A. Configuration Example

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

  # Auto-suggestion settings (NEW)
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
    predict: "^O"                     # Ctrl+O (manual)
    accept_suggestion: "^[[C"         # Right Arrow (NEW)
    clear_suggestion: "^["            # ESC (NEW)
```

### B. Environment Variables

```bash
# Override config via environment variables
export LLMSH_AUTO_SUGGEST=true
export LLMSH_DEBOUNCE_DELAY_MS=500
export LLMSH_MIN_PREFIX_LENGTH=3
export LLMSH_MAX_SUGGESTION_LENGTH=150
```

### C. Debugging

```bash
# Enable debug logging
export LLMSH_DEBUG=1

# View logs
tail -f ~/.llmsh/debug.log

# Test specific suggestion
echo '{"method":"complete","prefix":"git sta"}' | llmsh complete
```

### D. Benchmarking

```bash
# Measure cache hit rate
llmsh stats --cache-hit-rate

# Measure average latency
llmsh stats --avg-latency

# Measure API cost
llmsh stats --cost --last-day
```

---

## Glossary

- **Debounce**: Technique to delay function execution until user stops typing
- **POSTDISPLAY**: ZSH variable for displaying text after cursor without affecting buffer
- **ZLE**: Zsh Line Editor, the component handling user input
- **FIFO**: First-In-First-Out pipe for inter-process communication
- **region_highlight**: ZSH array for styling specific text regions
- **Widget**: ZSH term for a function bound to a keybinding

---

## Approval

**Document Status:** Draft
**Next Review Date:** 2024-11-26

**Reviewers:**
- [ ] Engineering Lead
- [ ] Product Manager
- [ ] Security Team
- [ ] UX Designer

**Approval:**
- [ ] Approved for implementation
- [ ] Changes requested
- [ ] Rejected

---

**END OF DOCUMENT**
