package cmd

import (
	"encoding/json"
	"fmt"
	goctx "context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/katichai/katich/internal/analysis"
	"github.com/katichai/katich/internal/config"
	"github.com/katichai/katich/internal/context"
	"github.com/katichai/katich/internal/embeddings"
	"github.com/katichai/katich/internal/git"
	"github.com/katichai/katich/internal/llm"
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
	forceRebuild   bool
	incremental    bool
	publishContext bool
)

func init() {
	// Add subcommands
	contextCmd.AddCommand(contextBuildCmd)
	contextCmd.AddCommand(contextShowCmd)
	contextCmd.AddCommand(contextClearCmd)

	// Flags for context build
	contextBuildCmd.Flags().BoolVarP(&forceRebuild, "force", "f", false, "force full rebuild (ignore cache)")
	contextBuildCmd.Flags().BoolVarP(&incremental, "incremental", "i", true, "incremental update (only changed files)")
	contextBuildCmd.Flags().BoolVar(&publishContext, "publish", false, "push context.json and embeddings.json to origin (katich-ai-context directory)")
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

	// 1. Find Git repository
	repo, err := git.FindRepository()
	if err != nil {
		return fmt.Errorf("failed to find Git repository: %w", err)
	}

	// 2. Load config early (needed for remote seed + incremental decisions)
	cfg, err := config.Load(GetConfig())
	if err != nil {
		fmt.Println("  ⚠️  No config found, using defaults")
		cfg = config.DefaultConfig()
	}

	// 3. Compute effective incremental flag
	useIncremental := incremental && !forceRebuild

	if verbose {
		fmt.Println("Verbose mode enabled")
		fmt.Printf("Repository: %s\n", repo.RootPath)
		fmt.Printf("Force rebuild: %v\n", forceRebuild)
		fmt.Printf("Incremental: %v (effective: %v)\n", incremental, useIncremental)
		fmt.Println()
	}

	katichDir := filepath.Join(repo.RootPath, ".katich")
	if err := os.MkdirAll(katichDir, 0755); err != nil {
		return fmt.Errorf("failed to create .katich directory: %w", err)
	}

	// 4. Seed from remote when incremental + source=remote
	if useIncremental && cfg.Context.Source == "remote" {
		fmt.Println("📡 Fetching remote context as base...")
		if err := fetchRemoteContextSeed(repo.RootPath, cfg); err != nil {
			fmt.Printf("  ⚠️  Could not fetch remote context: %v\n", err)
			fmt.Println("  Continuing with full build...")
		} else {
			fmt.Println("  ✅ Remote context fetched")
		}
	}

	// 5. Early exit if nothing changed since last build (same branch + same commit)
	if useIncremental {
		stateFile := filepath.Join(katichDir, ".last_build_state")
		savedBranch, savedCommit, err := readBuildState(stateFile)
		if err == nil {
			currentBranch := gitOutput(repo.RootPath, "rev-parse", "--abbrev-ref", "HEAD")
			headRef := gitOutput(repo.RootPath, "rev-parse", "HEAD")
			if savedBranch == currentBranch && savedCommit == headRef {
				fmt.Println("✅ No changes found since last context build. Nothing to do.")
				fmt.Println()
				fmt.Println("Use 'katich context build -f' to force a full rebuild.")
				return nil
			}
		}
	}

	// 6. Detection (always full, cheap)
	detector := context.NewDetector(repo.RootPath)
	fmt.Println("🔍 Scanning repository...")
	result, err := detector.Detect()
	if err != nil {
		return fmt.Errorf("failed to detect frameworks: %w", err)
	}

	// 6. Analysis: incremental or full
	fmt.Println("📊 Analyzing code...")
	analyzer := analysis.NewAnalyzer(repo.RootPath)

	var analysisResult *analysis.AnalysisResult

	if useIncremental {
		analysisResult = tryIncrementalAnalysis(repo.RootPath, analyzer, katichDir)
	}
	if analysisResult == nil {
		// Full analysis (first build, force, or incremental fallback)
		var err error
		analysisResult, err = analyzer.AnalyzeRepository()
		if err != nil {
			return fmt.Errorf("failed to analyze code: %w", err)
		}
	}

	// 7. Display results
	displayDetectionResults(result, analysisResult)

	// 8. Generate embeddings
	fmt.Println("🧠 Generating embeddings...")

	embeddingsAPIKey := resolveEmbeddingsKey(cfg)

	provider := embeddings.NewHybridProvider(
		"http://localhost:11434",
		"nomic-embed-text",
		embeddingsAPIKey,
		"text-embedding-3-small",
	)
	fmt.Printf("  Using provider: %s\n", provider.GetActiveProvider())

	generator := embeddings.NewGenerator(provider, repo.RootPath)
	embeddingIndex, err := generator.GenerateForAnalysis(analysisResult, useIncremental)
	if err != nil {
		fmt.Printf("  ⚠️  Failed to generate embeddings: %v\n", err)
		fmt.Println("  Continuing without embeddings...")
	} else {
		fmt.Printf("  ✅ Generated %d embeddings\n", len(embeddingIndex.Embeddings))
		embeddingPath := filepath.Join(katichDir, "embeddings.json")
		if err := generator.SaveIndex(embeddingIndex, embeddingPath); err != nil {
			fmt.Printf("  ⚠️  Failed to save embeddings: %v\n", err)
		} else {
			fmt.Printf("  💾 Saved to %s\n", embeddingPath)
		}
	}
	fmt.Println()

	// Display patterns and config files
	if len(result.Patterns) > 0 {
		fmt.Println("Architectural patterns:")
		for _, pattern := range result.Patterns {
			fmt.Printf("  • %s\n", pattern)
		}
		fmt.Println()
	}
	if len(result.Files) > 0 {
		fmt.Println("Configuration files found:")
		for file := range result.Files {
			fmt.Printf("  • %s\n", file)
		}
		fmt.Println()
	}

	// 9. Save context.json
	fmt.Println("💾 Saving context...")
	combinedContext := map[string]interface{}{
		"detection": result,
		"analysis":  analysisResult,
	}

	contextPath := filepath.Join(katichDir, "context.json")
	data, err := json.MarshalIndent(combinedContext, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}
	if err := os.WriteFile(contextPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write context file: %w", err)
	}
	fmt.Printf("✅ Context saved to %s\n", contextPath)
	fmt.Println()

	// 10. Persist build state (branch + HEAD)
	saveBuildState(repo.RootPath, katichDir)

	// 11. Publish if requested
	if publishContext {
		if err := publishContextToOrigin(repo.RootPath, cfg); err != nil {
			return fmt.Errorf("publish failed: %w", err)
		}
		fmt.Println()
	}

	fmt.Println("Next steps:")
	fmt.Println("  • Run 'katich context show' to view the context")
	fmt.Println("  • Run 'katich review latest' to review code with context")

	return nil
}

// tryIncrementalAnalysis attempts an incremental analysis using the previous build state.
// Returns nil if incremental is not possible (caller should fall back to full analysis).
func tryIncrementalAnalysis(repoRoot string, analyzer *analysis.Analyzer, katichDir string) *analysis.AnalysisResult {
	stateFile := filepath.Join(katichDir, ".last_build_state")
	savedBranch, savedCommit, err := readBuildState(stateFile)
	if err != nil {
		return nil
	}

	currentBranch := gitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	headRef := gitOutput(repoRoot, "rev-parse", "HEAD")
	if currentBranch == "" || headRef == "" {
		return nil
	}

	if savedBranch != currentBranch {
		fmt.Printf("  Branch changed (%s → %s), performing full analysis...\n", savedBranch, currentBranch)
		return nil
	}

	if savedCommit == headRef {
		// Should not reach here (early exit in runContextBuild handles this),
		// but as a safety net, reuse previous analysis.
		prevAnalysis := loadPreviousAnalysis(katichDir)
		if prevAnalysis == nil {
			return nil
		}
		return prevAnalysis
	}

	// Verify the saved commit is reachable
	verifyCmd := exec.Command("git", "cat-file", "-t", savedCommit)
	verifyCmd.Dir = repoRoot
	if err := verifyCmd.Run(); err != nil {
		fmt.Printf("  Previous commit %s unreachable, performing full analysis...\n", savedCommit[:8])
		return nil
	}

	prevAnalysis := loadPreviousAnalysis(katichDir)
	if prevAnalysis == nil {
		return nil
	}

	addedOrModified, deleted := getChangedFiles(repoRoot, savedCommit, headRef)
	if len(addedOrModified) == 0 && len(deleted) == 0 {
		fmt.Println("  No file changes detected, reusing previous analysis...")
		return prevAnalysis
	}

	fmt.Printf("  Incremental: %d changed, %d deleted (reusing %d files)\n",
		len(addedOrModified), len(deleted), len(prevAnalysis.Files)-len(deleted))

	// Start from previous files
	mergedFiles := make(map[string]*analysis.FileAnalysis, len(prevAnalysis.Files))
	for k, v := range prevAnalysis.Files {
		mergedFiles[k] = v
	}

	for _, d := range deleted {
		delete(mergedFiles, d)
	}

	if len(addedOrModified) > 0 {
		newAnalyses, err := analyzer.AnalyzeChangedFiles(addedOrModified)
		if err != nil {
			fmt.Printf("  ⚠️  Incremental analysis failed: %v, falling back to full...\n", err)
			return nil
		}
		for k, v := range newAnalyses {
			mergedFiles[k] = v
		}
	}

	return analyzer.BuildResultFromFiles(mergedFiles)
}

// readBuildState reads the saved branch and commit from .last_build_state
func readBuildState(path string) (branch string, commit string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	lines := strings.SplitN(strings.TrimSpace(string(data)), "\n", 2)
	if len(lines) != 2 || lines[0] == "" || lines[1] == "" {
		return "", "", fmt.Errorf("invalid build state")
	}
	return lines[0], lines[1], nil
}

// saveBuildState writes the current branch and HEAD to .last_build_state
func saveBuildState(repoRoot string, katichDir string) {
	branch := gitOutput(repoRoot, "rev-parse", "--abbrev-ref", "HEAD")
	head := gitOutput(repoRoot, "rev-parse", "HEAD")
	if branch == "" || head == "" {
		return
	}
	stateFile := filepath.Join(katichDir, ".last_build_state")
	_ = os.WriteFile(stateFile, []byte(branch+"\n"+head+"\n"), 0644)
}

// gitOutput runs a git command and returns trimmed stdout, or empty string on error.
func gitOutput(repoRoot string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// getChangedFiles runs git diff --name-status and returns added/modified and deleted file lists.
func getChangedFiles(repoRoot, fromRef, toRef string) (addedOrModified []string, deleted []string) {
	cmd := exec.Command("git", "diff", "--name-status", fromRef, toRef)
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		status := parts[0]
		switch {
		case status == "D":
			deleted = append(deleted, parts[1])
		case strings.HasPrefix(status, "R"):
			// Rename: old path is parts[1], new path is parts[2]
			if len(parts) >= 3 {
				deleted = append(deleted, parts[1])
				addedOrModified = append(addedOrModified, parts[2])
			}
		default:
			// A, M, C, T, etc.
			addedOrModified = append(addedOrModified, parts[1])
		}
	}
	return
}

// loadPreviousAnalysis loads the AnalysisResult from the existing context.json
func loadPreviousAnalysis(katichDir string) *analysis.AnalysisResult {
	contextPath := filepath.Join(katichDir, "context.json")
	data, err := os.ReadFile(contextPath)
	if err != nil {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	analysisData, ok := raw["analysis"]
	if !ok {
		return nil
	}

	var prev analysis.AnalysisResult
	if err := json.Unmarshal(analysisData, &prev); err != nil {
		return nil
	}

	if prev.Files == nil || len(prev.Files) == 0 {
		return nil
	}
	return &prev
}

// fetchRemoteContextSeed fetches context.json and embeddings.json from origin into .katich/
func fetchRemoteContextSeed(repoRoot string, cfg *config.Config) error {
	branch := cfg.Context.Remote.Branch
	if branch == "" {
		branch = "main"
	}
	dir := cfg.Context.Remote.Directory
	if dir == "" {
		dir = "katich-ai-context"
	}
	ref := "origin/" + branch

	fetchCmd := exec.Command("git", "fetch", "origin", branch)
	fetchCmd.Dir = repoRoot
	fetchCmd.Stdout = os.Stdout
	fetchCmd.Stderr = os.Stderr
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("git fetch: %w", err)
	}

	katichDir := filepath.Join(repoRoot, ".katich")
	for _, name := range []string{"context.json", "embeddings.json"} {
		remotePath := dir + "/" + name
		showCmd := exec.Command("git", "show", ref+":"+remotePath)
		showCmd.Dir = repoRoot
		out, err := showCmd.Output()
		if err != nil {
			return fmt.Errorf("git show %s: %w", remotePath, err)
		}
		dst := filepath.Join(katichDir, name)
		if err := os.WriteFile(dst, out, 0644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}
	return nil
}

// resolveEmbeddingsKey resolves the OpenAI API key for embeddings from config/env/API server.
func resolveEmbeddingsKey(cfg *config.Config) string {
	key := cfg.Embeddings.APIKey
	if key == "" {
		key = cfg.LLM.APIKey
	}
	if key == "" && cfg.APIServer.Enabled && cfg.APIServer.URL != "" {
		projectName, _ := cfg.GetProjectName()
		keyFetcher := llm.NewKeyFetcher(cfg.APIServer.URL, cfg.APIServer.Token)
		if fetchedKey, err := keyFetcher.FetchLLMKey(goctx.Background(), projectName, "openai"); err == nil {
			key = fetchedKey
		} else {
			fmt.Printf("  ⚠️  Could not fetch embeddings key from API server: %v\n", err)
		}
	}
	return key
}

// displayDetectionResults prints the analysis summary to stdout.
func displayDetectionResults(result *context.DetectionResult, analysisResult *analysis.AnalysisResult) {
	fmt.Println()
	fmt.Println("📊 Detection Results:")
	fmt.Println()

	if len(result.Languages) > 0 {
		fmt.Println("Languages detected:")
		for lang, count := range result.Languages {
			fmt.Printf("  • %s (%d files)\n", lang, count)
		}
		fmt.Println()
	}

	if len(result.Frameworks) > 0 {
		fmt.Println("Frameworks detected:")
		byType := make(map[context.FrameworkType][]context.Framework)
		for _, fw := range result.Frameworks {
			byType[fw.Type] = append(byType[fw.Type], fw)
		}
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

	fmt.Println("Code Metrics:")
	fmt.Printf("  • Total Lines of Code: %d\n", analysisResult.TotalMetrics.LinesOfCode)
	fmt.Printf("  • Total Functions: %d\n", analysisResult.TotalMetrics.FunctionCount)
	fmt.Printf("  • Total Classes/Structs: %d\n", analysisResult.TotalMetrics.ClassCount)
	fmt.Printf("  • Average Function Length: %.1f lines\n", analysisResult.TotalMetrics.AvgFunctionLength)
	fmt.Printf("  • Max Function Length: %d lines\n", analysisResult.TotalMetrics.MaxFunctionLength)
	fmt.Printf("  • Total Complexity: %d\n", analysisResult.TotalMetrics.CyclomaticComplexity)
	fmt.Println()

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
}

// publishContextToOrigin copies context.json and embeddings.json into the configured directory
// and runs git add / commit / push origin/{branch}.
func publishContextToOrigin(repoRoot string, cfg *config.Config) error {
	branch := cfg.Context.Remote.Branch
	if branch == "" {
		branch = "main"
	}
	dir := cfg.Context.Remote.Directory
	if dir == "" {
		dir = "katich-ai-context"
	}

	targetDir := filepath.Join(repoRoot, dir)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", dir, err)
	}

	katichDir := filepath.Join(repoRoot, ".katich")
	for _, name := range []string{"context.json", "embeddings.json"} {
		src := filepath.Join(katichDir, name)
		dst := filepath.Join(targetDir, name)
		data, err := os.ReadFile(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read %s: %w", name, err)
		}
		if err := os.WriteFile(dst, data, 0644); err != nil {
			return fmt.Errorf("write %s: %w", name, err)
		}
	}

	runGit := func(args ...string) error {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("git %v: %w", args, err)
		}
		return nil
	}

	if err := runGit("add", dir+"/"); err != nil {
		return err
	}
	// Only commit if there are staged changes
	diffCmd := exec.Command("git", "diff", "--cached", "--quiet")
	diffCmd.Dir = repoRoot
	if diffCmd.Run() != nil {
		// exit 1 = there are changes
		if err := runGit("commit", "-m", "chore: update katich ai context"); err != nil {
			return err
		}
	}
	if err := runGit("push", "origin", branch); err != nil {
		return err
	}
	fmt.Printf("  📤 Context published to origin/%s (%s/)\n", branch, dir)
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
