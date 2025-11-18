package main

import (
	"os"

	"llmsh/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		// Silent exit on error
		os.Exit(0)
	}
}
