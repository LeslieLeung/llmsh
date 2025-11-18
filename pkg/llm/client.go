package llm

import (
	"llmsh/pkg/config"
)

// Client represents an LLM client
type Client struct {
	config config.LLMConfig
}

// Result represents the result of an LLM call
type Result struct {
	Command string
	Model   string
	Usage   Usage
}

// Usage represents token usage information
type Usage struct {
	InputTokens         int
	OutputTokens        int
	CacheCreationTokens int
	CacheReadTokens     int
}

// NewClient creates a new LLM client
func NewClient(cfg config.LLMConfig) *Client {
	return &Client{config: cfg}
}

// Predict generates a prediction based on context
func (c *Client) Predict(prompt string) (*Result, error) {
	provider, err := c.getDefaultProvider()
	if err != nil {
		return nil, err
	}
	return callOpenAICompatible(provider, prompt)
}

// Complete completes a partial command
func (c *Client) Complete(prompt string) (*Result, error) {
	provider, err := c.getDefaultProvider()
	if err != nil {
		return nil, err
	}
	return callOpenAICompatible(provider, prompt)
}

// Generate generates a command from natural language
func (c *Client) Generate(prompt string) (*Result, error) {
	provider, err := c.getDefaultProvider()
	if err != nil {
		return nil, err
	}
	return callOpenAICompatible(provider, prompt)
}

// getDefaultProvider returns the default provider configuration
func (c *Client) getDefaultProvider() (config.ProviderConfig, error) {
	provider, ok := c.config.Providers[c.config.DefaultProvider]
	if !ok {
		return config.ProviderConfig{}, ErrProviderNotFound
	}
	return provider, nil
}
