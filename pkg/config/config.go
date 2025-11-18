package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the main configuration structure
type Config struct {
	LLM        LLMConfig        `mapstructure:"llm"`
	Prediction PredictionConfig `mapstructure:"prediction"`
	Cache      CacheConfig      `mapstructure:"cache"`
	Tracking   TrackingConfig   `mapstructure:"tracking"`
	ZSH        ZSHConfig        `mapstructure:"zsh"`
}

// LLMConfig contains LLM provider settings
type LLMConfig struct {
	DefaultProvider string                    `mapstructure:"default_provider"`
	Providers       map[string]ProviderConfig `mapstructure:"providers"`
}

// ProviderConfig contains settings for a specific LLM provider
type ProviderConfig struct {
	BaseURL     string  `mapstructure:"base_url"`
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
}

// PredictionConfig contains prediction behavior settings
type PredictionConfig struct {
	HistoryLength   int `mapstructure:"history_length"`
	MinPrefixLength int `mapstructure:"min_prefix_length"`
}

// CacheConfig contains caching settings
type CacheConfig struct {
	Enabled    bool   `mapstructure:"enabled"`
	DBPath     string `mapstructure:"db_path"`
	TTLDays    int    `mapstructure:"ttl_days"`
	MaxEntries int    `mapstructure:"max_entries"`
}

// TrackingConfig contains token tracking settings
type TrackingConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	DBPath  string `mapstructure:"db_path"`
}

// ZSHConfig contains ZSH-specific settings
type ZSHConfig struct {
	Keybindings map[string]string `mapstructure:"keybindings"`
}

var globalConfig *Config

// Load reads and parses the configuration file using Viper
func Load() (*Config, error) {
	if globalConfig != nil {
		return globalConfig, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}

	configPath := filepath.Join(home, ".llmsh")
	configFile := filepath.Join(configPath, "config.yaml")

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at %s. Run 'llmsh config init' to create one", configFile)
	}

	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(configPath)

	// Enable environment variable support
	v.SetEnvPrefix("LLMSH")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// Expand paths with ~
	cfg.Cache.DBPath = expandPath(cfg.Cache.DBPath)
	cfg.Tracking.DBPath = expandPath(cfg.Tracking.DBPath)

	// Expand environment variables in API keys
	for name, provider := range cfg.LLM.Providers {
		provider.APIKey = os.ExpandEnv(provider.APIKey)
		cfg.LLM.Providers[name] = provider
	}

	globalConfig = &cfg
	return globalConfig, nil
}

// expandPath expands ~ to the user's home directory
func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// GetProvider returns the configuration for the specified provider
func (c *Config) GetProvider(name string) (ProviderConfig, error) {
	if name == "" {
		name = c.LLM.DefaultProvider
	}

	provider, ok := c.LLM.Providers[name]
	if !ok {
		return ProviderConfig{}, fmt.Errorf("provider %s not found in config", name)
	}

	return provider, nil
}
