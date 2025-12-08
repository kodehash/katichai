package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/katichai/katich/internal/analysis"
	"github.com/katichai/katich/internal/config"
	"github.com/katichai/katich/internal/context"
	"github.com/katichai/katich/internal/embeddings"
	"github.com/katichai/katich/internal/git"
	"github.com/spf13/cobra"
)

// contextCmd represents the context command group
var contextCmd = &cobra.Command{
	Use:   "context",
	Short: "Manage codebase context and embeddings",
	Long: `Build, view, and manage the semantic context of your codebase.
The context includes framework detection, AST parsing, embeddings generation,
and similarity indexing.`,
}

var (
	// Context build flags
	forceRebuild bool
	incremental  bool
)

func init() {
	// Add subcommands
	contextCmd.AddCommand(contextBuildCmd)
	contextCmd.AddCommand(contextShowCmd)
	contextCmd.AddCommand(contextClearCmd)

	// Flags for context build
	contextBuildCmd.Flags().BoolVarP(&forceRebuild, "force", "f", false, "force full rebuild (ignore cache)")
	contextBuildCmd.Flags().BoolVarP(&incremental, "incremental", "i", true, "incremental update (only changed files)")
}

// contextBuildCmd builds the codebase context
var contextBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build codebase context and embeddings",
	Long: `Scan the repository, detect frameworks and languages, parse ASTs,
generate embeddings, and build a FAISS similarity index.

The context is stored in .katich/context.json and .katich/embeddings.index`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runContextBuild()
	},
}

// contextShowCmd displays the current context
var contextShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current context information",
	Long:  `Show the detected frameworks, languages, patterns, and statistics from the built context.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runContextShow()
	},
}

// contextClearCmd clears the cached context
var contextClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear cached context and embeddings",
	Long:  `Remove all cached context files, including context.json and embeddings.index.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runContextClear()
	},
}

func runContextBuild() error {
	fmt.Println("🔨 Building codebase context...")
	fmt.Println()
	
	// Find Git repository
	repo, err := git.FindRepository()
	if err != nil {
		return fmt.Errorf("failed to find Git repository: %w", err)
	}
	
	if verbose {
		fmt.Println("Verbose mode enabled")
		fmt.Printf("Repository: %s\n", repo.RootPath)
		fmt.Printf("Force rebuild: %v\n", forceRebuild)
		fmt.Printf("Incremental: %v\n", incremental)
		fmt.Println()
	}

	// Create detector
	detector := context.NewDetector(repo.RootPath)
	
	fmt.Println("🔍 Scanning repository...")
	result, err := detector.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect frameworks: %w", err)
	}

	// Run static analysis
	fmt.Println("📊 Analyzing code...")
	analyzer := analysis.NewAnalyzer(repo.RootPath)
	analysisResult, err := analyzer.AnalyzeRepository()
	if err != nil {
		return fmt.Errorf("failed to analyze code: %w", err)
	}

	// Display results
	fmt.Println()
	fmt.Println("📊 Detection Results:")
	fmt.Println()

	// Languages
	if len(result.Languages) > 0 {
		fmt.Println("Languages detected:")
		for lang, count := range result.Languages {
			fmt.Printf("  • %s (%d files)\n", lang, count)
		}
		fmt.Println()
	}

	// Frameworks
	if len(result.Frameworks) > 0 {
		fmt.Println("Frameworks detected:")
		
		// Group by type
		byType := make(map[context.FrameworkType][]context.Framework)
		for _, fw := range result.Frameworks {
			byType[fw.Type] = append(byType[fw.Type], fw)
		}

		// Display by type
		typeOrder := []context.FrameworkType{
			context.FrameworkTypeBackend,
			context.FrameworkTypeFrontend,
			context.FrameworkTypeFullStack,
			context.FrameworkTypeUI,
			context.FrameworkTypeBuild,
		}

		for _, fwType := range typeOrder {
			if frameworks, ok := byType[fwType]; ok && len(frameworks) > 0 {
				fmt.Printf("\n  %s:\n", fwType)
				for _, fw := range frameworks {
					fmt.Printf("    • %s (%s)\n", fw.Name, fw.Language)
				}
			}
		}
		fmt.Println()
	}

	// Code Metrics
	fmt.Println("Code Metrics:")
	fmt.Printf("  • Total Lines of Code: %d\n", analysisResult.TotalMetrics.LinesOfCode)
	fmt.Printf("  • Total Functions: %d\n", analysisResult.TotalMetrics.FunctionCount)
	fmt.Printf("  • Total Classes/Structs: %d\n", analysisResult.TotalMetrics.ClassCount)
	fmt.Printf("  • Average Function Length: %.1f lines\n", analysisResult.TotalMetrics.AvgFunctionLength)
	fmt.Printf("  • Max Function Length: %d lines\n", analysisResult.TotalMetrics.MaxFunctionLength)
	fmt.Printf("  • Total Complexity: %d\n", analysisResult.TotalMetrics.CyclomaticComplexity)
	fmt.Println()

	// Issues Summary
	if analysisResult.IssuesSummary.TotalIssues > 0 {
		fmt.Println("Issues Found:")
		fmt.Printf("  • Total: %d\n", analysisResult.IssuesSummary.TotalIssues)
		
		if len(analysisResult.IssuesSummary.BySeverity) > 0 {
			fmt.Println("  By Severity:")
			for severity, count := range analysisResult.IssuesSummary.BySeverity {
				fmt.Printf("    - %s: %d\n", severity, count)
			}
		}
		fmt.Println()
	}

	// Top Complex Functions
	if len(analysisResult.TopComplexity) > 0 {
		fmt.Println("Most Complex Functions:")
		for i, fn := range analysisResult.TopComplexity {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. %s (complexity: %d, %d lines)\n", i+1, fn.Name, fn.Complexity, fn.LOC)
		}
		fmt.Println()
	}

	// Generate embeddings
	fmt.Println("🧠 Generating embeddings...")
	
	// Load config to get API keys
	cfg, err := config.Load(GetConfig())
	if err != nil {
		fmt.Println("  ⚠️  No config found, using defaults")
		cfg = config.DefaultConfig()
	}

	// Create embedding provider (hybrid)
	provider := embeddings.NewHybridProvider(
		"http://localhost:11434",
		"nomic-embed-text",
		cfg.LLM.APIKey,
		"text-embedding-3-small",
	)

	fmt.Printf("  Using provider: %s\n", provider.GetActiveProvider())

	// Generate embeddings
	generator := embeddings.NewGenerator(provider, repo.RootPath)
	embeddingIndex, err := generator.GenerateForAnalysis(analysisResult)
	if err != nil {
		fmt.Printf("  ⚠️  Failed to generate embeddings: %v\n", err)
		fmt.Println("  Continuing without embeddings...")
	} else {
		fmt.Printf("  ✅ Generated %d embeddings\n", len(embeddingIndex.Embeddings))
		
		// Save embedding index
		embeddingPath := filepath.Join(repo.RootPath, ".katich", "embeddings.json")
		if err := generator.SaveIndex(embeddingIndex, embeddingPath); err != nil {
			fmt.Printf("  ⚠️  Failed to save embeddings: %v\n", err)
		} else {
			fmt.Printf("  💾 Saved to %s\n", embeddingPath)
		}
	}
	fmt.Println()

	// Patterns
	if len(result.Patterns) > 0 {
		fmt.Println("Architectural patterns:")
		for _, pattern := range result.Patterns {
			fmt.Printf("  • %s\n", pattern)
		}
		fmt.Println()
	}

	// Important files
	if len(result.Files) > 0 {
		fmt.Println("Configuration files found:")
		for file := range result.Files {
			fmt.Printf("  • %s\n", file)
		}
		fmt.Println()
	}

	// Create combined context
	combinedContext := map[string]interface{}{
		"detection": result,
		"analysis":  analysisResult,
	}

	// Save context
	fmt.Println("💾 Saving context...")
	contextPath := filepath.Join(repo.RootPath, ".katich", "context.json")
	
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(contextPath), 0755); err != nil {
		return fmt.Errorf("failed to create .katich directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(combinedContext, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	// Write file
	if err := os.WriteFile(contextPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}

	fmt.Printf("✅ Context saved to %s\n", contextPath)
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  • Run 'katich context show' to view the context")
	fmt.Println("  • Run 'katich review latest' to review code with context")

	return nil
}

func runContextShow() error {
	fmt.Println("📊 Codebase Context")
	fmt.Println()

	// Find Git repository
	repo, err := git.FindRepository()
	if err != nil {
		return fmt.Errorf("failed to find Git repository: %w", err)
	}

	// Load context
	contextPath := filepath.Join(repo.RootPath, ".katich", "context.json")
	data, err := os.ReadFile(contextPath)
	if err != nil {
		fmt.Println("⚠️  No context found. Run 'katich context build' first.")
		return nil
	}

	// Parse context
	var result context.DetectionResult
	if err := json.Unmarshal(data, &result); err != nil {
		return fmt.Errorf("failed to parse context: %w", err)
	}

	// Display languages
	if len(result.Languages) > 0 {
		fmt.Println("Languages:")
		for lang, count := range result.Languages {
			fmt.Printf("  • %s (%d files)\n", lang, count)
		}
		fmt.Println()
	}

	// Display frameworks
	if len(result.Frameworks) > 0 {
		fmt.Println("Frameworks:")
		
		// Group by type
		byType := make(map[context.FrameworkType][]context.Framework)
		for _, fw := range result.Frameworks {
			byType[fw.Type] = append(byType[fw.Type], fw)
		}

		// Display by type
		typeOrder := []context.FrameworkType{
			context.FrameworkTypeBackend,
			context.FrameworkTypeFrontend,
			context.FrameworkTypeFullStack,
			context.FrameworkTypeUI,
			context.FrameworkTypeBuild,
		}

		for _, fwType := range typeOrder {
			if frameworks, ok := byType[fwType]; ok && len(frameworks) > 0 {
				fmt.Printf("\n  %s:\n", fwType)
				for _, fw := range frameworks {
					fmt.Printf("    • %s (%s)\n", fw.Name, fw.Language)
				}
			}
		}
		fmt.Println()
	}

	// Display patterns
	if len(result.Patterns) > 0 {
		fmt.Println("Architectural Patterns:")
		for _, pattern := range result.Patterns {
			fmt.Printf("  • %s\n", pattern)
		}
		fmt.Println()
	}

	// Display files
	if len(result.Files) > 0 {
		fmt.Println("Configuration Files:")
		for file := range result.Files {
			fmt.Printf("  • %s\n", file)
		}
		fmt.Println()
	}

	fmt.Printf("Context file: %s\n", contextPath)

	return nil
}

func runContextClear() error {
	fmt.Println("🗑️  Clearing cached context...")
	fmt.Println()

	// Find Git repository
	repo, err := git.FindRepository()
	if err != nil {
		return fmt.Errorf("failed to find Git repository: %w", err)
	}

	katichDir := filepath.Join(repo.RootPath, ".katich")
	
	// Remove context.json
	contextPath := filepath.Join(katichDir, "context.json")
	if err := os.Remove(contextPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove context.json: %w", err)
	}

	// Remove embeddings.index if it exists
	embeddingsPath := filepath.Join(katichDir, "embeddings.index")
	if err := os.Remove(embeddingsPath); err != nil && !os.IsNotExist(err) {
		// Not critical, just warn
		fmt.Printf("⚠️  Could not remove embeddings.index: %v\n", err)
	}

	// Remove cache directory if it exists
	cachePath := filepath.Join(katichDir, "cache")
	if err := os.RemoveAll(cachePath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("⚠️  Could not remove cache directory: %v\n", err)
	}

	fmt.Println("✅ Context cleared successfully")
	fmt.Println()
	fmt.Println("Removed:")
	fmt.Println("  • context.json")
	fmt.Println("  • embeddings.index (if present)")
	fmt.Println("  • cache/ (if present)")

	return nil
}
