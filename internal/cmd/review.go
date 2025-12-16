package cmd

import (
	"fmt"
	"os"

	"github.com/katichai/katich/internal/config"
	"github.com/katichai/katich/internal/embeddings"
	"github.com/katichai/katich/internal/git"
	"github.com/katichai/katich/internal/review"
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
	
	// Find Git repository
	repo, err := git.FindRepository()
	if err != nil {
		return fmt.Errorf("failed to find Git repository: %w", err)
	}
	
	if verbose {
		fmt.Printf("Repository: %s\n", repo.RootPath)
	}

	// Load config
	cfg, err := config.Load(GetConfig())
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Initialize legacy reviewer (used by Engine)
	baseReviewer := review.NewReviewer(repo.RootPath, cfg)
	
	// Initialize Review Engine
	engine, err := review.NewEngine(cfg, baseReviewer)
	if err != nil {
		return fmt.Errorf("failed to initialize review engine: %w", err)
	}

	// Get diff
	diff, err := repo.GetDiff("HEAD")
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	// Run comprehensive review
	fmt.Println("🤖 Analyzing code changes with AI...")
	report, err := engine.Review(diff)
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	// Output Result
	formatter := review.NewFormatter()
	var output string
	
	switch outputFormat {
	case "json":
		output = formatter.FormatJSON(report)
	case "markdown":
		output = formatter.FormatMarkdown(report)
	case "terminal":
		fallthrough
	default:
		output = formatter.FormatText(report)
	}

	// Print or Write to file
	if outputFile != "" {
		if err := writeToFile(outputFile, output); err != nil {
			return err
		}
		fmt.Printf("✅ Report saved to %s\n", outputFile)
	} else {
		fmt.Println(output)
	}

	// Verify status for CI/CD
	if ciMode && report.Status == "FAIL" {
		return fmt.Errorf("review failed with score %d", report.Score)
	}

	return nil
}

func writeToFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func runReviewDiff(diffRange string) error {
	fmt.Printf("🔍 Reviewing diff range: %s\n", diffRange)
	
	// Find Git repository
	repo, err := git.FindRepository()
	if err != nil {
		return fmt.Errorf("failed to find Git repository: %w", err)
	}

	// Load config
	cfg, err := config.Load(GetConfig())
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Initialize reviewer
	reviewer := review.NewReviewer(repo.RootPath, cfg)

	// Get diff for range
	diff, err := repo.GetDiffRange(diffRange)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	// Run review
	result, err := reviewer.ReviewDiff(diff)
	if err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	// TODO: Share display logic with runReviewLatest (removed for brevity of this edit)
	// Display Duplicates
	if len(result.Duplicates) > 0 {
		fmt.Println("\n👯 Potential Duplicates Detected:")
		for source, dups := range result.Duplicates {
			fmt.Printf("  • %s is similar to:\n", source)
			for _, dup := range dups {
				level := embeddings.GetSimilarityLevel(dup.Similarity)
				dupLOC := dup.EndLine - dup.StartLine + 1
				fmt.Printf("    - %s:%s (%.1f%% - %s, %d lines)\n", dup.FilePath, dup.FuncName, dup.Similarity*100, level, dupLOC)
			}
		}
	}

	return nil
}

func runReviewFile(filePath string) error {
	// TODO: Implement file review using Reviewer
	fmt.Println("⚠️  Review file not fully implemented yet")
	return nil
}
