package cmd

import (
	"fmt"

	"github.com/katichai/katich/internal/config"
	"github.com/katichai/katich/internal/git"
	"github.com/spf13/cobra"
)

var (
	// Version information (set via build flags)
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"

	// Global flags
	verbose    bool
	configFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "katich",
	Short: "Context-aware, AI-assisted code review CLI tool",
	Long: `Katichai is a CLI tool that prevents unnecessary AI-generated code,
detects duplicated logic, enforces architectural patterns, and ensures
high-quality engineering standards.

It builds a semantic understanding of your codebase using static analysis,
embeddings, and framework detection, then performs intelligent code reviews
on git diffs.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "config file (default is .katich/config.yaml)")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(doctorCmd)
	rootCmd.AddCommand(contextCmd)
	rootCmd.AddCommand(reviewCmd)
}

// GetVerbose returns the verbose flag value
func GetVerbose() bool {
	return verbose
}

// GetConfig returns the config file path
func GetConfig() string {
	return configFile
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display version information",
	Long:  `Display the version, git commit, and build date of katich.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("katich version %s\n", Version)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Build date: %s\n", BuildDate)
	},
}

// doctorCmd represents the doctor command
var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check system requirements and configuration",
	Long: `Verify that all required dependencies are installed and properly configured.
This includes checking for Git, required Go packages, LLM API keys, and more.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDoctor()
	},
}

func runDoctor() error {
	fmt.Println("🔍 Running system diagnostics...")
	fmt.Println()

	checks := make([]struct {
		name   string
		status string
	}, 0)

	// Check Git installation
	gitInstalled := git.IsGitInstalled()
	gitStatus := "❌ Not found"
	if gitInstalled {
		version, err := git.GetGitVersion()
		if err == nil {
			gitStatus = fmt.Sprintf("✅ %s", version)
		} else {
			gitStatus = "✅ Found"
		}
	}
	checks = append(checks, struct {
		name   string
		status string
	}{"Git installation", gitStatus})

	// Check Go installation
	goVersion := "✅ Found"
	checks = append(checks, struct {
		name   string
		status string
	}{"Go installation", goVersion})

	// Check if in Git repository
	repo, err := git.FindRepository()
	repoStatus := "❌ Not in a Git repository"
	if err == nil {
		branch, _ := repo.GetCurrentBranch()
		repoStatus = fmt.Sprintf("✅ Found (branch: %s)", branch)
	}
	checks = append(checks, struct {
		name   string
		status string
	}{"Git repository", repoStatus})

	// Check configuration file
	configPath := configFile
	if configPath == "" {
		configPath = ".katich/config.yaml"
	}
	
	cfg, err := config.Load(configPath)
	configStatus := "⚠️  Not found (optional)"
	if err == nil {
		configStatus = "✅ Found"
	}
	checks = append(checks, struct {
		name   string
		status string
	}{"Configuration file", configStatus})

	// Check LLM API key
	llmStatus := "⚠️  Not configured (required for reviews)"
	if cfg != nil && cfg.LLM.APIKey != "" {
		llmStatus = fmt.Sprintf("✅ Configured (%s)", cfg.LLM.Provider)
	}
	checks = append(checks, struct {
		name   string
		status string
	}{"LLM API key", llmStatus})

	// Check embedding model
	embeddingStatus := "⚠️  Not configured"
	if cfg != nil && cfg.Embeddings.Model != "" {
		embeddingStatus = fmt.Sprintf("✅ Configured (%s)", cfg.Embeddings.Model)
	}
	checks = append(checks, struct {
		name   string
		status string
	}{"Embedding model", embeddingStatus})

	// Print all checks
	for _, check := range checks {
		fmt.Printf("%-30s %s\n", check.name+":", check.status)
	}

	fmt.Println()
	fmt.Println("💡 Tip: Create a .katich/config.yaml file to configure LLM and embedding settings")
	
	return nil
}
