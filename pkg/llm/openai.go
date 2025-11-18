package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/shared"

	"llmsh/pkg/config"
)

var (
	// ErrProviderNotFound is returned when a provider is not found
	ErrProviderNotFound = errors.New("provider not found")
	// ErrEmptyResponse is returned when the API returns no choices
	ErrEmptyResponse = errors.New("empty response from API")
)

// callOpenAICompatible calls an OpenAI-compatible API endpoint using the official SDK
func callOpenAICompatible(cfg config.ProviderConfig, prompt string) (*Result, error) {
	// Create client options
	opts := []option.RequestOption{}

	// Set API key if provided
	if cfg.APIKey != "" {
		opts = append(opts, option.WithAPIKey(cfg.APIKey))
	}

	// Set base URL if provided
	if cfg.BaseURL != "" {
		baseURL := strings.TrimSuffix(cfg.BaseURL, "/")
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	// Create OpenAI client
	client := openai.NewClient(opts...)

	// Prepare chat completion request
	params := openai.ChatCompletionNewParams{
		Model: shared.ChatModel(cfg.Model),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	}

	// Set optional parameters
	if cfg.MaxTokens > 0 {
		params.MaxTokens = openai.Int(int64(cfg.MaxTokens))
	}
	if cfg.Temperature >= 0 {
		params.Temperature = openai.Float(cfg.Temperature)
	}

	// Make the API call
	ctx := context.Background()
	completion, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("API call failed: %w", err)
	}

	// Check if we got any choices
	if len(completion.Choices) == 0 {
		return nil, ErrEmptyResponse
	}

	// Extract the command
	command := strings.TrimSpace(completion.Choices[0].Message.Content)

	// Remove markdown code blocks if present
	command = removeCodeBlock(command)

	// Build result
	result := &Result{
		Command: command,
		Model:   completion.Model,
		Usage: Usage{
			InputTokens:  int(completion.Usage.PromptTokens),
			OutputTokens: int(completion.Usage.CompletionTokens),
		},
	}

	// Handle cached tokens if available
	if completion.Usage.PromptTokensDetails.CachedTokens > 0 {
		result.Usage.CacheReadTokens = int(completion.Usage.PromptTokensDetails.CachedTokens)
	}

	return result, nil
}

// removeCodeBlock removes markdown code block markers from the command
func removeCodeBlock(cmd string) string {
	// Remove ```bash ... ``` or ```sh ... ``` or just ``` ... ```
	cmd = strings.TrimSpace(cmd)

	// Check if it starts with code block
	if strings.HasPrefix(cmd, "```") {
		lines := strings.Split(cmd, "\n")
		if len(lines) > 2 {
			// Remove first line (```bash or similar)
			lines = lines[1:]
			// Remove last line if it's ```
			if strings.TrimSpace(lines[len(lines)-1]) == "```" {
				lines = lines[:len(lines)-1]
			}
			cmd = strings.Join(lines, "\n")
		}
	}

	return strings.TrimSpace(cmd)
}
