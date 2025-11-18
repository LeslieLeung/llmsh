package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var cleanAll bool

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean llmsh data",
	Long:  `Remove logs, cache, and optionally all data except config.`,
	RunE:  runClean,
}

func init() {
	cleanCmd.Flags().BoolVarP(&cleanAll, "all", "a", false, "Remove all data including tokens (keeps config)")
}

func runClean(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home directory: %w", err)
	}

	llmshDir := filepath.Join(home, ".llmsh")

	// Files to clean
	logFile := "/tmp/llmsh_debug.log"
	cacheDB := filepath.Join(llmshDir, "cache.db")
	tokensJSON := filepath.Join(llmshDir, "tokens.json")

	cleaned := []string{}
	errors := []error{}

	// Always clean log file
	if fileExists(logFile) {
		if err := removeIfExists(logFile); err != nil {
			errors = append(errors, fmt.Errorf("log: %w", err))
		} else {
			cleaned = append(cleaned, "debug log")
		}
	}

	// Always clean cache
	if fileExists(cacheDB) {
		if err := removeIfExists(cacheDB); err != nil {
			errors = append(errors, fmt.Errorf("cache: %w", err))
		} else {
			cleaned = append(cleaned, "cache database")
		}
	}

	// Clean tokens if -a flag is set
	if cleanAll {
		if fileExists(tokensJSON) {
			if err := removeIfExists(tokensJSON); err != nil {
				errors = append(errors, fmt.Errorf("tokens: %w", err))
			} else {
				cleaned = append(cleaned, "token tracking data")
			}
		}
	}

	// Report results
	if len(cleaned) > 0 {
		fmt.Fprintf(os.Stderr, "Cleaned:\n")
		for _, item := range cleaned {
			fmt.Fprintf(os.Stderr, "  ✓ %s\n", item)
		}
	} else {
		fmt.Fprintf(os.Stderr, "No files to clean\n")
	}

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "\nErrors:\n")
		for _, err := range errors {
			fmt.Fprintf(os.Stderr, "  ✗ %v\n", err)
		}
		return fmt.Errorf("clean completed with errors")
	}

	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func removeIfExists(path string) error {
	if !fileExists(path) {
		return nil
	}
	return os.Remove(path)
}
