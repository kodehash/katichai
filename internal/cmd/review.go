package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/katichai/katich/internal/analysis"
	"github.com/katichai/katich/internal/git"
	"github.com/spf13/cobra"
)

// reviewCmd represents the review command group
var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review code changes using AI-assisted analysis",
	Long: `Analyze git diffs, detect AI-generated code, find duplicates,
and provide architecture-aware code reviews.`,
}

var (
	// Review flags
	ciMode       bool
	outputFormat string
	outputFile   string
)

func init() {
	// Add subcommands
	reviewCmd.AddCommand(reviewLatestCmd)
	reviewCmd.AddCommand(reviewDiffCmd)
	reviewCmd.AddCommand(reviewFileCmd)

	// Global review flags
	reviewCmd.PersistentFlags().BoolVar(&ciMode, "ci", false, "CI mode (exit with error code on issues)")
	reviewCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "terminal", "output format (terminal, json, markdown, html)")
	reviewCmd.PersistentFlags().StringVar(&outputFile, "output-file", "", "write output to file")
}

// reviewLatestCmd reviews the latest commit
var reviewLatestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Review the latest commit",
	Long:  `Analyze the most recent commit in the current branch.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReviewLatest()
	},
}

// reviewDiffCmd reviews a specific diff range
var reviewDiffCmd = &cobra.Command{
	Use:   "diff <range>",
	Short: "Review a specific commit range",
	Long: `Analyze changes in a git commit range.

Examples:
  katich review diff HEAD~3..HEAD
  katich review diff main..feature-branch
  katich review diff abc123..def456`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReviewDiff(args[0])
	},
}

// reviewFileCmd reviews a specific file
var reviewFileCmd = &cobra.Command{
	Use:   "file <path>",
	Short: "Review a specific file",
	Long:  `Analyze a specific file for code quality, duplicates, and AI-generated patterns.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReviewFile(args[0])
	},
}

func runReviewLatest() error {
	fmt.Println("🔍 Reviewing latest commit...")
	fmt.Println()
	
	// Find Git repository
	repo, err := git.FindRepository()
	if err != nil {
		return fmt.Errorf("failed to find Git repository: %w", err)
	}
	
	if verbose {
		fmt.Println("Verbose mode enabled")
		fmt.Printf("Repository: %s\n", repo.RootPath)
		fmt.Printf("CI mode: %v\n", ciMode)
		fmt.Printf("Output format: %s\n", outputFormat)
		fmt.Println()
	}

	// Check if context exists
	contextPath := filepath.Join(repo.RootPath, ".katich", "context.json")
	hasContext := false
	if _, err := os.Stat(contextPath); err == nil {
		hasContext = true
		if verbose {
			fmt.Println("✅ Context found, using for enhanced analysis")
		}
	} else {
		fmt.Println("⚠️  No context found. Run 'katich context build' for better analysis.")
		fmt.Println()
	}

	// Get latest commit
	commit, err := repo.GetLatestCommit()
	if err != nil {
		return fmt.Errorf("failed to get latest commit: %w", err)
	}

	// Get diff
	diff, err := repo.GetDiff("HEAD")
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	// Display commit info
	fmt.Printf("📝 Commit: %s\n", commit.ShortHash)
	fmt.Printf("👤 Author: %s <%s>\n", commit.Author, commit.Email)
	fmt.Printf("📅 Date: %s\n", commit.Date.Format("2006-01-02 15:04:05"))
	fmt.Printf("💬 Message: %s\n", commit.Message)
	fmt.Println()

	// Display diff summary
	fmt.Println("📊 Changes:")
	for _, file := range diff.Files {
		status := "M"
		if file.Status != "" {
			status = file.Status
		}
		fmt.Printf("  [%s] %s (+%d -%d)\n", status, file.Path, file.Additions, file.Deletions)
	}
	fmt.Println()

	// Analyze changed files
	if hasContext {
		fmt.Println("🔬 Analyzing changed files...")
		changedFiles := make([]string, 0)
		for _, file := range diff.Files {
			changedFiles = append(changedFiles, file.Path)
		}

		analyzer := analysis.NewAnalyzer(repo.RootPath)
		fileAnalyses, err := analyzer.AnalyzeChangedFiles(changedFiles)
		if err != nil {
			fmt.Printf("⚠️  Analysis error: %v\n", err)
		} else if len(fileAnalyses) > 0 {
			// Display analysis results
			totalIssues := 0
			for filePath, fileAnalysis := range fileAnalyses {
				if len(fileAnalysis.Issues) > 0 {
					fmt.Printf("\n📄 %s:\n", filePath)
					for _, issue := range fileAnalysis.Issues {
						totalIssues++
						severity := "ℹ️"
						if issue.Severity == analysis.SeverityWarning {
							severity = "⚠️"
						} else if issue.Severity == analysis.SeverityError {
							severity = "❌"
						}
						fmt.Printf("  %s Line %d: %s\n", severity, issue.Line, issue.Message)
						if issue.Suggestion != "" {
							fmt.Printf("     💡 %s\n", issue.Suggestion)
						}
					}
				}
			}

			if totalIssues == 0 {
				fmt.Println("✅ No issues found in changed files!")
			} else {
				fmt.Printf("\n⚠️  Found %d issue(s) in changed files\n", totalIssues)
			}
		}
		fmt.Println()
	}

	// AI-powered review placeholder
	fmt.Println("🤖 AI-Powered Review:")
	fmt.Println("  ⚠️  LLM-based review not yet implemented")
	fmt.Println()
	fmt.Println("  Next enhancements:")
	fmt.Println("    • Generate embeddings for new code")
	fmt.Println("    • Search for similar code patterns")
	fmt.Println("    • Detect AI-generated boilerplate")
	fmt.Println("    • Run LLM classifier")
	fmt.Println("    • Synthesize comprehensive review")

	return nil
}

func runReviewDiff(diffRange string) error {
	fmt.Printf("🔍 Reviewing diff range: %s\n", diffRange)
	fmt.Println()
	
	// Find Git repository
	repo, err := git.FindRepository()
	if err != nil {
		return fmt.Errorf("failed to find Git repository: %w", err)
	}
	
	if verbose {
		fmt.Println("Verbose mode enabled")
		fmt.Printf("Repository: %s\n", repo.RootPath)
		fmt.Printf("CI mode: %v\n", ciMode)
		fmt.Printf("Output format: %s\n", outputFormat)
		fmt.Println()
	}

	// Get diff for range
	diff, err := repo.GetDiffRange(diffRange)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	// Display diff summary
	fmt.Println("📊 Changes:")
	for _, file := range diff.Files {
		fmt.Printf("  %s (+%d -%d)\n", file.Path, file.Additions, file.Deletions)
	}
	fmt.Println()

	// TODO: Implement actual review logic
	fmt.Println("⚠️  AI-powered review not yet implemented")

	return nil
}

func runReviewFile(filePath string) error {
	fmt.Printf("🔍 Reviewing file: %s\n", filePath)
	
	if verbose {
		fmt.Println("Verbose mode enabled")
		fmt.Printf("CI mode: %v\n", ciMode)
		fmt.Printf("Output format: %s\n", outputFormat)
	}

	// TODO: Implement review logic
	fmt.Println("⚠️  Review not yet implemented")

	return nil
}
