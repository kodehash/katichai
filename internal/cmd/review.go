package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/katichai/katich/internal/config"
	"github.com/katichai/katich/internal/git"
	"github.com/katichai/katich/internal/review"
	"github.com/spf13/cobra"
)

// reviewCmd represents the review command group
var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review code changes using AI-assisted analysis",
	Long: `Analyze git diffs, detect AI-generated code, find duplicates,
and provide architecture-aware code reviews.

Available commands:
  latest    Review the latest commit
  diff      Review a specific commit range (e.g., main..feature)
  file      Review a specific file
  full      Review the entire repository (all tracked files)

Examples:
  katich review latest
  katich review diff HEAD~3..HEAD
  katich review diff main..feature-branch
  katich review file path/to/file.go
  katich review full`,
}

var (
	// Review flags
	ciMode       bool
	outputFormat string
	outputFile   string
	generateHTML bool
	generateGFM  bool
)

func init() {
	// Add subcommands
	reviewCmd.AddCommand(reviewLatestCmd)
	reviewCmd.AddCommand(reviewDiffCmd)
	reviewCmd.AddCommand(reviewFileCmd)
	reviewCmd.AddCommand(reviewFullCmd)

	// Global review flags
	reviewCmd.PersistentFlags().BoolVar(&ciMode, "ci", false, "CI mode (exit with error code on issues)")
	reviewCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "terminal", "output format (terminal, json, markdown, html)")
	reviewCmd.PersistentFlags().StringVar(&outputFile, "output-file", "", "write output to file")
	reviewCmd.PersistentFlags().BoolVar(&generateHTML, "html", false, "generate HTML report (overrides config setting)")
	reviewCmd.PersistentFlags().BoolVar(&generateGFM, "gfm", false, "generate GitHub Flavored Markdown report (overrides config setting)")
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

// reviewFullCmd reviews the entire repository
var reviewFullCmd = &cobra.Command{
	Use:   "full",
	Short: "Review the entire repository",
	Long: `Perform a comprehensive review of all tracked files in the repository.

This command reviews the entire codebase (not just diffs) and provides extensive
coverage of architecture, security, code quality, and patterns across the repository.
Uses intelligent sampling to handle large repositories efficiently.

The review includes:
  - Full codebase analysis with intelligent file sampling (up to 100 files)
  - Architecture and design pattern review
  - Security vulnerability scanning
  - Code quality and maintainability assessment
  - Duplicate code detection
  - AI-generated code detection

Example:
  katich review full`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runReviewFull()
	},
}

// checkKatichInitialized checks if .katich directory exists
// This check happens before Git repository check
func checkKatichInitialized() error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	
	// Walk up the directory tree to find .katich directory
	current := cwd
	for {
		katichDir := filepath.Join(current, ".katich")
		info, err := os.Stat(katichDir)
		if err == nil && info.IsDir() {
			// Found .katich directory
			return nil
		}
		
		// Check if we've reached the filesystem root
		parent := filepath.Dir(current)
		if parent == current {
			// Reached root, .katich not found
			break
		}
		current = parent
	}
	
	return fmt.Errorf("katich not Initialized, run katich init to get started")
}

// printASCIIBanner prints the Katich AI ASCII art banner
func printASCIIBanner() {
	banner := `
██╗  ██╗ █████╗ ████████╗██╗ ██████╗██╗  ██╗     █████╗ ██╗
██║ ██╔╝██╔══██╗╚══██╔══╝██║██╔════╝██║  ██║    ██╔══██╗██║
█████╔╝ ███████║   ██║   ██║██║     ███████║    ███████║██║
██╔═██╗ ██╔══██║   ██║   ██║██║     ██╔══██║    ██╔══██║██║
██║  ██╗██║  ██║   ██║   ██║╚██████╗██║  ██║    ██║  ██║██║
╚═╝  ╚═╝╚═╝  ╚═╝   ╚═╝   ╚═╝ ╚═════╝╚═╝  ╚═╝    ╚═╝  ╚═╝╚═╝
                                                           
                                                             
              Context-aware AI code review tool
`
	fmt.Print(banner)
	fmt.Println()
}

func runReviewLatest() error {
	// Check if katich is initialized
	if err := checkKatichInitialized(); err != nil {
		return err
	}
	
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

	// Override HTML generation if flag is set
	if generateHTML {
		cfg.Review.GenerateHTML = true
	}

	// Override GFM generation if flag is set
	if generateGFM {
		cfg.Review.GenerateGFM = true
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

	// Print ASCII banner
	printASCIIBanner()
	
	// Run comprehensive review
	fmt.Println("🤖 Analyzing code changes with AI...")
	// For "latest" command, pass empty string (engine will extract from diff)
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

	// Verify status for CI/CD (commented out - scoring is subjective)
	// if ciMode && report.Status == "FAIL" {
	// 	return fmt.Errorf("review failed with score %d", report.Score)
	// }

	return nil
}

func writeToFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func runReviewDiff(diffRange string) error {
	// Check if katich is initialized
	if err := checkKatichInitialized(); err != nil {
		return err
	}
	
	// Print ASCII banner
	printASCIIBanner()
	
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

	// Override HTML generation if flag is set
	if generateHTML {
		cfg.Review.GenerateHTML = true
	}

	// Override GFM generation if flag is set
	if generateGFM {
		cfg.Review.GenerateGFM = true
	}

	// Initialize legacy reviewer (used by Engine)
	baseReviewer := review.NewReviewer(repo.RootPath, cfg)
	
	// Initialize Review Engine
	engine, err := review.NewEngine(cfg, baseReviewer)
	if err != nil {
		return fmt.Errorf("failed to initialize review engine: %w", err)
	}

	// Get diff for range
	diff, err := repo.GetDiffRange(diffRange)
	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	// Run comprehensive review with AI
	fmt.Println("🤖 Analyzing code changes with AI...")
	report, err := engine.Review(diff, diffRange)
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

		// Verify status for CI/CD (commented out - scoring is subjective)
		// if ciMode && report.Status == "FAIL" {
		// 	return fmt.Errorf("review failed with score %d", report.Score)
		// }

		return nil
	}

func runReviewFull() error {
	// Check if katich is initialized
	if err := checkKatichInitialized(); err != nil {
		return err
	}
	
	// Print ASCII banner
	printASCIIBanner()
	
	fmt.Println("🔍 Reviewing entire repository...")
	
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

	// Override HTML generation if flag is set
	if generateHTML {
		cfg.Review.GenerateHTML = true
	}

	// Override GFM generation if flag is set
	if generateGFM {
		cfg.Review.GenerateGFM = true
	}

	// Initialize legacy reviewer (used by Engine)
	baseReviewer := review.NewReviewer(repo.RootPath, cfg)
	
	// Initialize Review Engine
	engine, err := review.NewEngine(cfg, baseReviewer)
	if err != nil {
		return fmt.Errorf("failed to initialize review engine: %w", err)
	}

	// Get full repository diff
	diff, err := repo.GetFullRepositoryDiff()
	if err != nil {
		return fmt.Errorf("failed to get repository files: %w", err)
	}

	// Confirmation prompt for full repository review
	fmt.Println()
	fmt.Println("⚠️  WARNING: Full Repository Review")
	fmt.Println()
	fmt.Println("Full repository review is not usually recommended.")
	fmt.Println("Context is huge for a proper code review and your LLM token consumption will be very high.")
	fmt.Println()
	fmt.Print("Still want to continue? (y/N): ")
	
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		response := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if response != "y" && response != "yes" {
			fmt.Println("\n❌ Full repository review cancelled.")
			return nil
		}
	}
	
	fmt.Println()

	// Run comprehensive full repository review
	fmt.Println("🤖 Analyzing entire codebase with AI...")
	report, err := engine.ReviewFullRepository(diff)
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

	// Verify status for CI/CD (commented out - scoring is subjective)
	// if ciMode && report.Status == "FAIL" {
	// 	return fmt.Errorf("review failed with score %d", report.Score)
	// }

	return nil
}

	func runReviewFile(filePath string) error {
	// Check if katich is initialized
	if err := checkKatichInitialized(); err != nil {
		return err
	}
	
	// Print ASCII banner
	printASCIIBanner()
	
	// TODO: Implement file review using Reviewer
	fmt.Println("⚠️  Review file not fully implemented yet")
	return nil
}
