package cmd

import (
	"fmt"
	"strings"

	"llmsh/pkg/config"
	"llmsh/pkg/llm"
	"llmsh/pkg/tracker"

	"github.com/spf13/cobra"
)

var nl2cmdCmd = &cobra.Command{
	Use:   "nl2cmd",
	Short: "Convert natural language to shell command",
	Long:  `Reads a natural language description from stdin and generates a shell command.`,
	RunE:  runNL2Cmd,
}

func runNL2Cmd(cmd *cobra.Command, args []string) error {
	// Read request from stdin
	req, err := readRequest()
	if err != nil {
		writeError(err.Error())
		return err
	}

	// Validate description
	if req.Description == "" {
		writeError("description is required")
		return fmt.Errorf("description is required")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		writeError(fmt.Sprintf("load config: %v", err))
		return err
	}

	// Build prompt
	prompt := buildNL2CmdPrompt(req.Description, req.CWD, req.History, req.OSInfo)

	// Call LLM
	client := llm.NewClient(cfg.LLM)
	result, err := client.Generate(prompt)
	if err != nil {
		// Fail silently for LLM errors
		return err
	}

	// Record token usage
	if cfg.Tracking.Enabled {
		tracker.RecordUsage(&tracker.Record{
			Method:       "nl2cmd",
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

func buildNL2CmdPrompt(description, cwd string, history []string, osInfo string) string {
	var sb strings.Builder

	sb.WriteString("You are a shell command generator.\n\n")
	sb.WriteString("Task: Convert natural language description to a shell command.\n")
	sb.WriteString("Context:\n")
	if osInfo != "" {
		sb.WriteString(fmt.Sprintf("- OS: %s\n", osInfo))
	}
	sb.WriteString(fmt.Sprintf("- Current directory: %s\n", cwd))
	sb.WriteString(fmt.Sprintf("- Description: %s\n", description))

	if len(history) > 0 {
		sb.WriteString("- Recent commands (for context):\n")
		// Show last 3 commands
		start := len(history) - 3
		if start < 0 {
			start = 0
		}
		for i := start; i < len(history); i++ {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", i-start+1, history[i]))
		}
	}

	sb.WriteString("\nGenerate a safe, practical shell command that accomplishes the task.\n")
	sb.WriteString("Rules:\n")
	sb.WriteString("- Return ONLY the command, no explanation\n")
	sb.WriteString("- Ensure the command is safe (no destructive operations without confirmation)\n")
	sb.WriteString("- Use common Unix/Linux tools\n")
	sb.WriteString("- Be concise and practical\n")
	sb.WriteString("- Do not include markdown code blocks\n\n")
	sb.WriteString("Command:")

	return sb.String()
}
