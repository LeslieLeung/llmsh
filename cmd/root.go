package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Request represents the JSON request structure from ZSH
type Request struct {
	Method      string   `json:"method"`
	History     []string `json:"history,omitempty"`
	CWD         string   `json:"cwd,omitempty"`
	GitBranch   string   `json:"git_branch,omitempty"`
	OSInfo      string   `json:"os_info,omitempty"`
	Prefix      string   `json:"prefix,omitempty"`
	Description string   `json:"description,omitempty"`
	Timestamp   int64    `json:"timestamp,omitempty"`
}

// Response represents the JSON response structure to ZSH
type Response struct {
	Result interface{}  `json:"result"`
	Tokens *TokenUsage  `json:"tokens,omitempty"`
	Error  string       `json:"error,omitempty"`
}

// TokenUsage represents token usage information
type TokenUsage struct {
	InputTokens         int `json:"input_tokens"`
	OutputTokens        int `json:"output_tokens"`
	CacheCreationTokens int `json:"cache_creation_tokens"`
	CacheReadTokens     int `json:"cache_read_tokens"`
}

var rootCmd = &cobra.Command{
	Use:   "llmsh",
	Short: "LLM-powered shell command prediction and completion",
	Long: `llmsh is a ZSH plugin that provides intelligent command prediction,
completion, and natural language to command conversion using LLMs.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(predictCmd)
	rootCmd.AddCommand(completeCmd)
	rootCmd.AddCommand(nl2cmdCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(cleanCmd)
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// writeResponse writes a response to stdout as JSON
func writeResponse(resp *Response) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.Encode(resp)
}

// writeError writes an error response to stdout
func writeError(msg string) {
	resp := &Response{Error: msg}
	writeResponse(resp)
}

// readRequest reads a JSON request from stdin
func readRequest() (*Request, error) {
	var req Request
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&req); err != nil {
		return nil, fmt.Errorf("invalid JSON input: %w", err)
	}
	return &req, nil
}
