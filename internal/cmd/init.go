package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/katichai/katich/internal/git"
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
	// Check if we're in a Git repository
	repo, err := git.FindRepository()
	if err != nil {
		fmt.Println("Error: katich works with git repos, kindly initiate git and add a commit for katich to compare and review")
		return
	}

	// Extract project name from Git repository
	projectName, err := repo.GetProjectName()
	if err != nil {
		fmt.Printf("Warning: failed to extract project name: %v\n", err)
		projectName = "" // Will be empty in config
	}

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

	// Build default config content with project name
	projectNameLine := ""
	if projectName != "" {
		projectNameLine = fmt.Sprintf("project_name: %s\n\n", projectName)
	}
	
	defaultConfig := projectNameLine + `llm:
  provider: ollama
  model: llama3
  base_url: http://localhost:11434
  max_input_tokens: 20000

embeddings:
  provider: ollama
  model: nomic-embed-text

analysis:
  max_function_length: 50
  complexity_threshold: 10
  similarity_threshold: 0.70
  min_function_lines: 5
  duplicate_threshold: 0.85
  ignore_trivial_patterns: true
  sampling:
    enabled: true
    max_files: 20
    context_lines: 2
    skip_generated: true
    skip_tests: false
    adaptive_budget: true
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
