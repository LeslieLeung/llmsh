package cmd

import (
	"fmt"
	"strings"

	"llmsh/pkg/config"
	"llmsh/pkg/context"
	"llmsh/pkg/llm"
	"llmsh/pkg/tracker"

	"github.com/spf13/cobra"
)

var completeCmd = &cobra.Command{
	Use:   "complete",
	Short: "Complete a partial command",
	Long:  `Reads a partial command and context from stdin and completes it.`,
	RunE:  runComplete,
}

func runComplete(cmd *cobra.Command, args []string) error {
	// Read request from stdin
	req, err := readRequest()
	if err != nil {
		writeError(err.Error())
		return err
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		writeError(fmt.Sprintf("load config: %v", err))
		return err
	}

	// Check minimum prefix length
	if len(req.Prefix) < cfg.Prediction.MinPrefixLength {
		writeError("prefix too short")
		return fmt.Errorf("prefix too short")
	}

	// Filter sensitive information
	filteredHistory := context.FilterSensitive(req.History)

	// Build prompt
	prompt := buildCompletePrompt(req.Prefix, filteredHistory, req.CWD, req.OSInfo)

	// Call LLM
	client := llm.NewClient(cfg.LLM)
	result, err := client.Complete(prompt)
	if err != nil {
		// Fail silently for LLM errors
		return err
	}

	// Record token usage
	if cfg.Tracking.Enabled {
		tracker.RecordUsage(&tracker.Record{
			Method:       "complete",
			Provider:     cfg.LLM.DefaultProvider,
			Model:        result.Model,
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
		})
	}

	// Write response
	writeResponse(&Response{
		Result: &PredictResult{
			Command: result.Command,
			Cached:  false,
		},
		Tokens: &TokenUsage{
			InputTokens:  result.Usage.InputTokens,
			OutputTokens: result.Usage.OutputTokens,
		},
	})

	return nil
}

func buildCompletePrompt(prefix string, history []string, cwd, osInfo string) string {
	var sb strings.Builder

	sb.WriteString("You are a shell command completion assistant.\n\n")
	sb.WriteString("Context:\n")
	if osInfo != "" {
		sb.WriteString(fmt.Sprintf("- OS: %s\n", osInfo))
	}
	sb.WriteString(fmt.Sprintf("- Current directory: %s\n", cwd))
	sb.WriteString(fmt.Sprintf("- Partial command: %s\n", prefix))

	if len(history) > 0 {
		sb.WriteString("- Recent commands:\n")
		// Show last 5 commands
		start := len(history) - 5
		if start < 0 {
			start = 0
		}
		for i := start; i < len(history); i++ {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i-start+1, history[i]))
		}
	}

	sb.WriteString("\nComplete the partial command to a full, valid command.\n")
	sb.WriteString("Rules:\n")
	sb.WriteString("- Return ONLY the completed command\n")
	sb.WriteString("- Ensure it starts with or relates to the given prefix\n")
	sb.WriteString("- Be practical and safe\n")
	sb.WriteString("- Do not include markdown code blocks\n\n")
	sb.WriteString("Completed command:")

	return sb.String()
}
