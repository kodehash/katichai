package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize katich in the current directory",
	Long:  `Initialize katich by creating a .katich directory and a default config.yaml file.`,
	Run: func(cmd *cobra.Command, args []string) {
		runInit()
	},
}

func runInit() {
	configDir := ".katich"
	configFile := filepath.Join(configDir, "config.yaml")

	// Check if config already exists
	if _, err := os.Stat(configFile); err == nil {
		fmt.Printf("Configuration file already exists at %s\n", configFile)
		return
	}

	// Create directory
	if err := os.MkdirAll(configDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}

	// Default config content
	defaultConfig := `llm:
  provider: ollama
  model: llama3
  base_url: http://localhost:11434

embeddings:
  provider: ollama
  model: nomic-embed-text

analysis:
  max_function_length: 50
  complexity_threshold: 10
  similarity_threshold: 0.85
`

	// Write config file
	if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
		fmt.Printf("Error writing config file: %v\n", err)
		return
	}

	fmt.Println("✅ Initialized katich configuration!")
	fmt.Printf("Created %s with default settings (Ollama).\n", configFile)
	fmt.Println("Next steps:")
	fmt.Println("  1. Run 'katich context build' to analyze your codebase")
	fmt.Println("  2. Run 'katich review latest' to review changes")
}
