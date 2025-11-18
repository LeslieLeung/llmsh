package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage llmsh configuration",
	Long:  `Initialize or display the llmsh configuration.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize configuration file",
	Long:  `Create a default configuration file at ~/.llmsh/config.yaml`,
	RunE:  runConfigInit,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	Long:  `Display the current configuration settings.`,
	RunE:  runConfigShow,
}

func init() {
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configShowCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".llmsh")
	configFile := filepath.Join(configDir, "config.yaml")

	// Check if config already exists
	if _, err := os.Stat(configFile); err == nil {
		fmt.Fprintf(os.Stderr, "Configuration file already exists at %s\n", configFile)
		fmt.Fprintf(os.Stderr, "Remove it first if you want to reinitialize.\n")
		return nil
	}

	// Create config directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	// Create a new viper instance for writing
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")

	// Set default LLM configuration
	v.Set("llm.default_provider", "openai")

	// OpenAI provider
	v.Set("llm.providers.openai.base_url", "https://api.openai.com/v1")
	v.Set("llm.providers.openai.api_key", "${OPENAI_API_KEY}")
	v.Set("llm.providers.openai.model", "gpt-4-turbo-preview")
	v.Set("llm.providers.openai.max_tokens", 100)
	v.Set("llm.providers.openai.temperature", 0.2)

	// Local provider (Ollama)
	v.Set("llm.providers.local.base_url", "http://localhost:11434/v1")
	v.Set("llm.providers.local.api_key", "")
	v.Set("llm.providers.local.model", "codellama:7b")
	v.Set("llm.providers.local.max_tokens", 100)
	v.Set("llm.providers.local.temperature", 0.2)

	// Prediction settings
	v.Set("prediction.history_length", 20)
	v.Set("prediction.min_prefix_length", 3)

	// Cache settings
	v.Set("cache.enabled", true)
	v.Set("cache.db_path", "~/.llmsh/cache.db")
	v.Set("cache.ttl_days", 7)
	v.Set("cache.max_entries", 1000)

	// Tracking settings
	v.Set("tracking.enabled", true)
	v.Set("tracking.db_path", "~/.llmsh/tokens.json")

	// ZSH keybindings
	v.Set("zsh.keybindings.accept_prediction", "^I")
	v.Set("zsh.keybindings.nl2cmd", "^[^M")

	// Write config file
	if err := v.WriteConfig(); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Configuration file created at: %s\n\n", configFile)
	fmt.Fprintf(os.Stderr, "Next steps:\n")
	fmt.Fprintf(os.Stderr, "1. Set your OpenAI API key:\n")
	fmt.Fprintf(os.Stderr, "   export OPENAI_API_KEY=\"your-api-key\"\n")
	fmt.Fprintf(os.Stderr, "   Or edit %s and replace ${OPENAI_API_KEY}\n\n", configFile)
	fmt.Fprintf(os.Stderr, "2. Alternatively, configure a local LLM provider (like Ollama)\n")
	fmt.Fprintf(os.Stderr, "   by changing 'default_provider' to 'local' in the config\n\n")
	fmt.Fprintf(os.Stderr, "3. Load the ZSH plugin by adding to your ~/.zshrc:\n")
	fmt.Fprintf(os.Stderr, "   source /path/to/llmsh/zsh/llmsh.plugin.zsh\n")

	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	configFile := filepath.Join(home, ".llmsh", "config.yaml")

	// Read the config file directly
	data, err := os.ReadFile(configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		return err
	}

	fmt.Fprintf(os.Stderr, "Current Configuration:\n")
	fmt.Fprintf(os.Stderr, "=====================\n\n")
	fmt.Fprintf(os.Stderr, "%s", data)

	return nil
}
