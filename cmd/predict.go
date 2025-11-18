package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"llmsh/pkg/cache"
	"llmsh/pkg/config"
	"llmsh/pkg/context"
	"llmsh/pkg/llm"
	"llmsh/pkg/tracker"

	"github.com/spf13/cobra"
)

// PredictResult represents the result of a prediction
type PredictResult struct {
	Command    string  `json:"command"`
	Confidence float64 `json:"confidence,omitempty"`
	Cached     bool    `json:"cached"`
}

var predictCmd = &cobra.Command{
	Use:   "predict",
	Short: "Predict the next command based on context",
	Long:  `Reads context from stdin as JSON and predicts the next shell command.`,
	RunE:  runPredict,
}

func runPredict(cmd *cobra.Command, args []string) error {
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

	// Filter sensitive information
	filteredHistory := context.FilterSensitive(req.History)

	// Generate cache key
	cacheKey := generateCacheKey(filteredHistory, req.CWD, req.GitBranch)

	// Check cache if enabled
	if cfg.Cache.Enabled {
		cacheDB, err := cache.Open(cfg.Cache.DBPath)
		if err == nil {
			defer cacheDB.Close()

			if cached := cacheDB.Get(cacheKey); cached != nil {
				writeResponse(&Response{
					Result: &PredictResult{
						Command: cached.Command,
						Cached:  true,
					},
				})
				return nil
			}
		}
	}

	// Build prompt
	prompt := buildPredictPrompt(filteredHistory, req.CWD, req.GitBranch, req.OSInfo)

	// Call LLM
	client := llm.NewClient(cfg.LLM)
	result, err := client.Predict(prompt)
	if err != nil {
		// Fail silently for LLM errors
		return err
	}

	// Save to cache
	if cfg.Cache.Enabled {
		if cacheDB, err := cache.Open(cfg.Cache.DBPath); err == nil {
			defer cacheDB.Close()
			cacheDB.Set(cacheKey, result.Command)
		}
	}

	// Record token usage
	if cfg.Tracking.Enabled {
		tracker.RecordUsage(&tracker.Record{
			Method:              "predict",
			Provider:            cfg.LLM.DefaultProvider,
			Model:               result.Model,
			InputTokens:         result.Usage.InputTokens,
			OutputTokens:        result.Usage.OutputTokens,
			CacheCreationTokens: result.Usage.CacheCreationTokens,
			CacheReadTokens:     result.Usage.CacheReadTokens,
		})
	}

	// Write response
	writeResponse(&Response{
		Result: &PredictResult{
			Command: result.Command,
			Cached:  false,
		},
		Tokens: &TokenUsage{
			InputTokens:         result.Usage.InputTokens,
			OutputTokens:        result.Usage.OutputTokens,
			CacheCreationTokens: result.Usage.CacheCreationTokens,
			CacheReadTokens:     result.Usage.CacheReadTokens,
		},
	})

	return nil
}

func generateCacheKey(history []string, cwd, gitBranch string) string {
	h := sha256.New()
	for _, cmd := range history {
		h.Write([]byte(cmd))
	}
	h.Write([]byte(cwd))
	h.Write([]byte(gitBranch))
	return hex.EncodeToString(h.Sum(nil))[:32]
}

func buildPredictPrompt(history []string, cwd, gitBranch, osInfo string) string {
	var sb strings.Builder

	sb.WriteString("You are a shell command prediction assistant.\n\n")
	sb.WriteString("Context:\n")
	if osInfo != "" {
		sb.WriteString(fmt.Sprintf("- OS: %s\n", osInfo))
	}
	sb.WriteString(fmt.Sprintf("- Working directory: %s\n", cwd))
	if gitBranch != "" {
		sb.WriteString(fmt.Sprintf("- Git branch: %s\n", gitBranch))
	}

	if len(history) > 0 {
		sb.WriteString("- Recent commands:\n")
		// Show last 10 commands in reverse order (most recent first)
		start := len(history) - 10
		if start < 0 {
			start = 0
		}
		for i := len(history) - 1; i >= start; i-- {
			sb.WriteString(fmt.Sprintf("  %d. %s\n", len(history)-i, history[i]))
		}
	}

	sb.WriteString("\nPredict the next most likely command the user will execute.\n")
	sb.WriteString("Rules:\n")
	sb.WriteString("- Return ONLY the command, no explanation\n")
	sb.WriteString("- Consider the workflow pattern\n")
	sb.WriteString("- Be concise and practical\n")
	sb.WriteString("- Do not include markdown code blocks\n\n")
	sb.WriteString("Command:")

	return sb.String()
}
