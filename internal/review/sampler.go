package review

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	analysispkg "github.com/katichai/katich/internal/analysis"
	"github.com/katichai/katich/internal/git"
	"github.com/katichai/katich/internal/llm"
	"github.com/katichai/katich/internal/safepath"
)

// Token budget constants
const (
	TOTAL_INPUT_BUDGET       = 20000
	SYSTEM_PROMPT_TOKENS     = 1500
	CONTEXT_TOKENS_AVG       = 2000
	BUFFER_TOKENS            = 1000
	MAX_CONTEXT_LINES        = 2
	MAX_FILES_TO_REVIEW      = 20
	LARGE_FILE_THRESHOLD     = 500 // Lines changed
	MAX_FILES_FULL_REVIEW    = 15 // Minimum files for full repository review (actual selection uses 30% of filtered files, max 30)
	FULL_REVIEW_TOKEN_BUDGET = 30000 // Token budget for full repository review
)

// DiffSampler samples diffs intelligently to fit token budgets
type DiffSampler struct {
	maxTokens     int
	maxFiles      int
	contextLines  int
	skipGenerated bool
	skipTests     bool
}

// NewDiffSampler creates a new diff sampler with a token budget
func NewDiffSampler(maxTokens int) *DiffSampler {
	return &DiffSampler{
		maxTokens:     maxTokens,
		maxFiles:      MAX_FILES_TO_REVIEW,
		contextLines:  MAX_CONTEXT_LINES,
		skipGenerated: true,
		skipTests:     false,
	}
}

// SampledDiff represents a sampled/reduced diff
type SampledDiff struct {
	Files         []*SampledFile
	TotalTokens   int
	OriginalFiles int
}

// SampledFile represents a file in the sampled diff
type SampledFile struct {
	Path         string
	Status       string
	Additions    int
	Deletions    int
	Content      string
	RiskScore    int
	RiskReasons  []string
	Truncated    bool
}

// Format formats the sampled diff as a string
func (sd *SampledDiff) Format() string {
	var sb strings.Builder
	
	for _, file := range sd.Files {
		sb.WriteString(fmt.Sprintf("=== %s (%s) (+%d -%d) ===\n", 
			file.Path, file.Status, file.Additions, file.Deletions))
		
		if len(file.RiskReasons) > 0 {
			sb.WriteString(fmt.Sprintf("Risk: %d (%s)\n", 
				file.RiskScore, strings.Join(file.RiskReasons, ", ")))
		}
		
		sb.WriteString(file.Content)
		
		if file.Truncated {
			sb.WriteString("\n... [content truncated to fit token budget] ...\n")
		}
		
		sb.WriteString("\n\n")
	}
	
	return sb.String()
}

// SamplingReport contains statistics about the sampling process
type SamplingReport struct {
	TotalFiles        int
	FilteredFiles     int
	SampledFiles      int
	FilteredReasons   map[string]int
	TotalTokensBefore int
	TotalTokensAfter  int
	TopRiskFiles      []FileRisk
}

// FileRisk represents a file with its risk score
type FileRisk struct {
	File    *git.DiffFile
	Score   int
	Reasons []string
}

// Summary returns a human-readable summary of sampling
func (r *SamplingReport) Summary() string {
	savings := "0%"
	if r.TotalTokensBefore > 0 {
		pct := float64(r.TotalTokensBefore-r.TotalTokensAfter) / float64(r.TotalTokensBefore) * 100
		savings = fmt.Sprintf("%.1f%%", pct)
	}
	
	return fmt.Sprintf(
		"Reviewed %d/%d files (filtered %d). Token reduction: %s (%d → %d tokens)",
		r.SampledFiles, r.TotalFiles, r.FilteredFiles, savings,
		r.TotalTokensBefore, r.TotalTokensAfter,
	)
}

// SampleDiff samples a diff to fit within the token budget
func (s *DiffSampler) SampleDiff(diff *git.Diff, analysisResults map[string]*analysispkg.FileAnalysis) (*SampledDiff, *SamplingReport) {
	report := &SamplingReport{
		TotalFiles:      len(diff.Files),
		FilteredReasons: make(map[string]int),
		TopRiskFiles:    make([]FileRisk, 0),
	}
	
	// Estimate original size
	originalDiff := formatOriginalDiff(diff)
	report.TotalTokensBefore = llm.EstimateTokens(originalDiff)
	
	// Phase 1: Filter noise
	filtered := s.filterNoise(diff.Files, report)
	
	// Phase 2: Calculate risk scores and prioritize
	risks := s.calculateRiskScores(filtered, analysisResults)
	sort.Slice(risks, func(i, j int) bool {
		return risks[i].Score > risks[j].Score
	})
	
	// Store top risk files for reporting
	topN := 10
	if len(risks) < topN {
		topN = len(risks)
	}
	report.TopRiskFiles = risks[:topN]
	
	// Phase 3: Select files to include
	selectedFiles := s.selectFiles(risks)
	report.SampledFiles = len(selectedFiles)
	report.FilteredFiles = report.TotalFiles - report.SampledFiles
	
	// Phase 4: Sample selected files
	sampledDiff := &SampledDiff{
		Files:         make([]*SampledFile, 0),
		OriginalFiles: len(diff.Files),
	}
	
	tokenBudget := s.maxTokens
	for _, risk := range selectedFiles {
		if tokenBudget <= 0 {
			break
		}
		
		sampled := s.sampleFile(risk.File, analysisResults[risk.File.Path], tokenBudget)
		sampled.RiskScore = risk.Score
		sampled.RiskReasons = risk.Reasons
		
		fileTokens := llm.EstimateTokens(sampled.Content)
		if fileTokens <= tokenBudget {
			sampledDiff.Files = append(sampledDiff.Files, sampled)
			tokenBudget -= fileTokens
			sampledDiff.TotalTokens += fileTokens
		}
	}
	
	report.TotalTokensAfter = sampledDiff.TotalTokens
	
	return sampledDiff, report
}

// filterNoise filters out generated, lock, and binary files
func (s *DiffSampler) filterNoise(files []*git.DiffFile, report *SamplingReport) []*git.DiffFile {
	filtered := make([]*git.DiffFile, 0)
	
	for _, file := range files {
		// Skip files with no actual changes
		if file.Additions == 0 && file.Deletions == 0 {
			report.FilteredReasons["no_changes"]++
			continue
		}
		
		// Skip all hidden files and directories (starting with .)
		// Check if path starts with . or contains /.
		if strings.HasPrefix(file.Path, ".") || strings.Contains(file.Path, "/.") {
			report.FilteredReasons["hidden"]++
			continue
		}

		if safepath.IsBlockedPath(file.Path) {
			report.FilteredReasons["sensitive"]++
			continue
		}

		// Check if generated
		if s.skipGenerated && isGeneratedFile(file.Path) {
			report.FilteredReasons["generated"]++
			continue
		}
		
		// Check if lock file
		if isLockFile(file.Path) {
			report.FilteredReasons["lock_file"]++
			continue
		}
		
		// Check if package management file
		if isPackageManagementFile(file.Path) {
			report.FilteredReasons["package_management"]++
			continue
		}
		
		// Check if binary
		if isBinaryFile(file.Path) {
			report.FilteredReasons["binary"]++
			continue
		}
		
		// Check if test (optional)
		if s.skipTests && isTestFile(file.Path) {
			report.FilteredReasons["test"]++
			continue
		}
		
		filtered = append(filtered, file)
	}
	
	return filtered
}

// calculateRiskScores calculates risk scores for files
func (s *DiffSampler) calculateRiskScores(files []*git.DiffFile, analysisResults map[string]*analysispkg.FileAnalysis) []FileRisk {
	risks := make([]FileRisk, 0, len(files))
	
	for _, file := range files {
		risk := FileRisk{
			File:    file,
			Score:   0,
			Reasons: make([]string, 0),
		}
		
		analysis := analysisResults[file.Path]
		
		// Security-sensitive paths
		if isSecuritySensitive(file.Path) {
			risk.Score += 10
			risk.Reasons = append(risk.Reasons, "security-sensitive")
		}
		
		// Core business logic
		if isCoreBusinessLogic(file.Path) {
			risk.Score += 8
			risk.Reasons = append(risk.Reasons, "core-logic")
		}
		
		// Database/API interaction
		if isDatabaseOrAPI(file.Path) {
			risk.Score += 7
			risk.Reasons = append(risk.Reasons, "database/api")
		}
		
		// New file
		if file.Status == "A" {
			risk.Score += 5
			risk.Reasons = append(risk.Reasons, "new-file")
		}
		
		// High complexity from static analysis
		if analysis != nil {
			highComplexityCount := 0
			for _, fn := range analysis.Functions {
				if fn.Complexity > 15 {
					highComplexityCount++
				}
			}
			if highComplexityCount > 0 {
				risk.Score += 3
				risk.Reasons = append(risk.Reasons, fmt.Sprintf("high-complexity(%d)", highComplexityCount))
			}
		
		// AI-generated patterns (check for AI detection issues)
		for _, issue := range analysis.Issues {
			if issue.Type == analysispkg.IssueTypeAIGenerated {
				risk.Score += 3
				risk.Reasons = append(risk.Reasons, "ai-generated")
				break
			}
		}
		}
		
		// Lower priority: tests
		if isTestFile(file.Path) {
			risk.Score -= 5
			risk.Reasons = append(risk.Reasons, "test-file")
		}
		
		// Lower priority: docs
		if isDocFile(file.Path) {
			risk.Score -= 3
			risk.Reasons = append(risk.Reasons, "documentation")
		}
		
		risks = append(risks, risk)
	}
	
	return risks
}

// selectFiles selects top N files by risk score
func (s *DiffSampler) selectFiles(risks []FileRisk) []FileRisk {
	if len(risks) <= s.maxFiles {
		return risks
	}
	return risks[:s.maxFiles]
}

// sampleFile samples a single file to reduce its size
func (s *DiffSampler) sampleFile(file *git.DiffFile, analysis *analysispkg.FileAnalysis, tokenBudget int) *SampledFile {
	sampled := &SampledFile{
		Path:      file.Path,
		Status:    file.Status,
		Additions: file.Additions,
		Deletions: file.Deletions,
	}
	
	totalChanges := file.Additions + file.Deletions
	isPython := strings.HasSuffix(file.Path, ".py")
	
	// For very large files, use signature extraction
	if totalChanges > LARGE_FILE_THRESHOLD && analysis != nil {
		sampled.Content = s.extractSignatures(file, analysis)
		sampled.Truncated = true
		return sampled
	}
	
	// For normal files, reduce context
	if file.Patch != "" {
		if isPython {
			// Python-specific optimization: more aggressive context reduction
			// Python's indentation makes diffs more verbose
			sampled.Content = s.reduceContextPython(file.Patch)
		} else {
			sampled.Content = s.reduceContext(file.Patch)
		}
	}
	
	// Check if still too large
	tokens := llm.EstimateTokens(sampled.Content)
	if tokens > tokenBudget {
		// Truncate further
		sampled.Content = llm.TruncateToTokenLimit(sampled.Content, tokenBudget)
		sampled.Truncated = true
	}
	
	return sampled
}

// reduceContext reduces unchanged context lines in a patch
func (s *DiffSampler) reduceContext(patch string) string {
	lines := strings.Split(patch, "\n")
	reduced := make([]string, 0)
	
	lastIncluded := -1000 // Track last included line to avoid duplicates
	
	for i, line := range lines {
		// Always include diff headers
		if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			reduced = append(reduced, line)
			lastIncluded = i
			continue
		}
		
		// Include changed lines and context
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			// Include context before change
			start := max(lastIncluded+1, i-s.contextLines)
			for j := start; j < i; j++ {
				if j > lastIncluded {
					reduced = append(reduced, lines[j])
				}
			}
			
			// Include the change
			reduced = append(reduced, line)
			lastIncluded = i
			
			// Include context after change (up to contextLines)
			end := min(len(lines), i+s.contextLines+1)
			for j := i + 1; j < end && j < len(lines); j++ {
				if !strings.HasPrefix(lines[j], "+") && !strings.HasPrefix(lines[j], "-") {
					reduced = append(reduced, lines[j])
					lastIncluded = j
				} else {
					break
				}
			}
		}
	}
	
	return strings.Join(reduced, "\n")
}

// reduceContextPython reduces context for Python files more aggressively
// Python's indentation-based syntax makes diffs more verbose
func (s *DiffSampler) reduceContextPython(patch string) string {
	lines := strings.Split(patch, "\n")
	reduced := make([]string, 0)
	
	lastIncluded := -1000
	pythonContextLines := 1 // Reduced from 2 to 1 for Python
	
	for i, line := range lines {
		// Always include diff headers
		if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---") {
			reduced = append(reduced, line)
			lastIncluded = i
			continue
		}
		
		// Include changed lines and minimal context
		if strings.HasPrefix(line, "+") || strings.HasPrefix(line, "-") {
			// Include minimal context before change (1 line for Python)
			start := max(lastIncluded+1, i-pythonContextLines)
			for j := start; j < i; j++ {
				if j > lastIncluded {
					reduced = append(reduced, lines[j])
				}
			}
			
			// Include the change
			reduced = append(reduced, line)
			lastIncluded = i
			
			// Include minimal context after change (1 line for Python)
			end := min(len(lines), i+pythonContextLines+1)
			for j := i + 1; j < end && j < len(lines); j++ {
				if !strings.HasPrefix(lines[j], "+") && !strings.HasPrefix(lines[j], "-") {
					reduced = append(reduced, lines[j])
					lastIncluded = j
				} else {
					break
				}
			}
		}
	}
	
	return strings.Join(reduced, "\n")
}

// extractSignatures extracts function signatures for large files
func (s *DiffSampler) extractSignatures(file *git.DiffFile, analysis *analysispkg.FileAnalysis) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf("File: %s (+%d -%d)\n", file.Path, file.Additions, file.Deletions))
	sb.WriteString("Large file - showing function signatures only:\n\n")
	
	if len(analysis.Functions) > 0 {
		sb.WriteString("Functions:\n")
		for _, fn := range analysis.Functions {
			params := strings.Join(fn.Parameters, ", ")
			returnType := fn.ReturnType
			if returnType == "" {
				returnType = "void"
			}
			
			sb.WriteString(fmt.Sprintf("  • %s(%s) -> %s\n", fn.Name, params, returnType))
			sb.WriteString(fmt.Sprintf("    Lines: %d-%d, Complexity: %d, LOC: %d\n", 
				fn.StartLine, fn.EndLine, fn.Complexity, fn.LOC))
		}
	}
	
	return sb.String()
}

// Helper functions

func isGeneratedFile(path string) bool {
	generated := []string{
		".min.js", ".bundle.js", ".generated.go", "pb.go", "pb_test.go",
		"dist/", "build/", ".next/", ".nuxt/",
		"migrations/", "locale/", "i18n/",
	}
	
	for _, pattern := range generated {
		if strings.Contains(path, pattern) {
			return true
		}
	}
	
	// Check filename patterns
	base := filepath.Base(path)
	if strings.HasSuffix(base, ".min.js") || strings.HasSuffix(base, ".min.css") {
		return true
	}
	
	return false
}

func isLockFile(path string) bool {
	lockFiles := []string{
		"package-lock.json", "yarn.lock", "pnpm-lock.yaml",
		"Gemfile.lock", "Cargo.lock", "go.sum",
		"composer.lock", "poetry.lock",
	}
	
	base := filepath.Base(path)
	for _, lock := range lockFiles {
		if base == lock {
			return true
		}
	}
	
	return false
}

func isBinaryFile(path string) bool {
	binaryExts := []string{
		".png", ".jpg", ".jpeg", ".gif", ".ico", ".pdf",
		".zip", ".tar", ".gz", ".exe", ".dll", ".so",
		".woff", ".woff2", ".ttf", ".eot",
	}
	
	ext := strings.ToLower(filepath.Ext(path))
	for _, binExt := range binaryExts {
		if ext == binExt {
			return true
		}
	}
	
	return false
}

func isTestFile(path string) bool {
	return strings.Contains(path, "_test.") || 
		strings.Contains(path, ".test.") ||
		strings.Contains(path, "/test/") ||
		strings.Contains(path, "/tests/") ||
		strings.Contains(path, "__tests__/")
}

// isPackageManagementFile checks if a file is a package management configuration file
func isPackageManagementFile(path string) bool {
	base := filepath.Base(path)
	pathLower := strings.ToLower(path)
	baseLower := strings.ToLower(base)
	
	// JavaScript/TypeScript/Node.js
	packageFiles := []string{
		"package.json",
		"package-lock.json", // Also in isLockFile, but check here too
		"yarn.lock",         // Also in isLockFile
		"pnpm-lock.yaml",    // Also in isLockFile
		".npmrc",
		".yarnrc",
		".yarnrc.yml",
	}
	
	// Java/Gradle/Maven
	javaFiles := []string{
		"build.gradle",
		"build.gradle.kts",
		"settings.gradle",
		"settings.gradle.kts",
		"gradle.properties",
		"gradle-wrapper.properties",
		"pom.xml",
		"build.xml",
		"project.clj", // Clojure/Leiningen
	}
	
	// Python
	pythonFiles := []string{
		"requirements.txt",
		"requirements-dev.txt",
		"requirements-test.txt",
		"setup.py",
		"setup.cfg",
		"pyproject.toml",
		"Pipfile",
		"Pipfile.lock", // Also in isLockFile
		"poetry.lock",  // Also in isLockFile
		"MANIFEST.in",
		"conda.yml",
		"environment.yml",
	}
	
	// Go
	goFiles := []string{
		"go.mod",
		"go.sum", // Also in isLockFile
		"Gopkg.toml",
		"Gopkg.lock",
		"glide.yaml",
		"glide.lock",
		"vendor.json",
	}
	
	// Rust
	rustFiles := []string{
		"Cargo.toml",
		"Cargo.lock", // Also in isLockFile
	}
	
	// Ruby
	rubyFiles := []string{
		"Gemfile",
		"Gemfile.lock", // Also in isLockFile
		"Rakefile",
		".ruby-version",
		".ruby-gemset",
	}
	
	// PHP
	phpFiles := []string{
		"composer.json",
		"composer.lock", // Also in isLockFile
	}
	
	// .NET (exact filenames)
	dotnetFiles := []string{
		"packages.config",
		"project.json",
		"project.assets.json",
	}
	
	// Swift
	swiftFiles := []string{
		"Package.swift",
		"Package.resolved",
	}
	
	// Dart/Flutter
	dartFiles := []string{
		"pubspec.yaml",
		"pubspec.lock",
		"pubspec.yml",
	}
	
	// Elixir
	elixirFiles := []string{
		"mix.exs",
		"mix.lock",
	}
	
	// Haskell
	haskellFiles := []string{
		"stack.yaml",
		"cabal.project",
	}
	
	// Scala
	scalaFiles := []string{
		"build.sbt",
		"project/build.properties",
		"project/plugins.sbt",
	}
	
	// Combine all package management files
	allPackageFiles := packageFiles
	allPackageFiles = append(allPackageFiles, javaFiles...)
	allPackageFiles = append(allPackageFiles, pythonFiles...)
	allPackageFiles = append(allPackageFiles, goFiles...)
	allPackageFiles = append(allPackageFiles, rustFiles...)
	allPackageFiles = append(allPackageFiles, rubyFiles...)
	allPackageFiles = append(allPackageFiles, phpFiles...)
	allPackageFiles = append(allPackageFiles, dotnetFiles...)
	allPackageFiles = append(allPackageFiles, swiftFiles...)
	allPackageFiles = append(allPackageFiles, dartFiles...)
	allPackageFiles = append(allPackageFiles, elixirFiles...)
	allPackageFiles = append(allPackageFiles, haskellFiles...)
	allPackageFiles = append(allPackageFiles, scalaFiles...)
	
	// Check exact filename matches
	for _, pkgFile := range allPackageFiles {
		if baseLower == strings.ToLower(pkgFile) {
			return true
		}
	}
	
	// Check for .NET project files with extensions
	ext := filepath.Ext(pathLower)
	dotnetExts := []string{".csproj", ".vbproj", ".fsproj"}
	for _, dotnetExt := range dotnetExts {
		if ext == dotnetExt {
			return true
		}
	}
	
	// Check for Haskell .cabal files
	if ext == ".cabal" {
		return true
	}
	
	// Check for files in package management directories
	packageDirs := []string{
		"/gradle/",
		"/.gradle/",
		"/node_modules/",
		"/vendor/",
		"/venv/",
		"/.venv/",
		"/env/",
		"/.env/",
		"/__pycache__/",
		"/target/", // Rust, Scala
		"/.cargo/",
		"/.bundle/",
		"/.mvn/",
	}
	
	for _, pkgDir := range packageDirs {
		if strings.Contains(pathLower, pkgDir) {
			return true
		}
	}
	
	return false
}

func isSecuritySensitive(path string) bool {
	sensitive := []string{"auth", "crypto", "permission", "security", "password", "token", "jwt"}
	pathLower := strings.ToLower(path)
	
	for _, keyword := range sensitive {
		if strings.Contains(pathLower, keyword) {
			return true
		}
	}
	
	return false
}

func isCoreBusinessLogic(path string) bool {
	logic := []string{"service", "model", "controller", "handler", "usecase", "domain"}
	pathLower := strings.ToLower(path)
	
	for _, keyword := range logic {
		if strings.Contains(pathLower, keyword) {
			return true
		}
	}
	
	return false
}

func isDatabaseOrAPI(path string) bool {
	db := []string{"repository", "dao", "database", "db", "api", "endpoint", "rest", "graphql"}
	pathLower := strings.ToLower(path)
	
	for _, keyword := range db {
		if strings.Contains(pathLower, keyword) {
			return true
		}
	}
	
	return false
}

func isDocFile(path string) bool {
	return strings.HasSuffix(path, ".md") || 
		strings.HasSuffix(path, ".txt") ||
		strings.HasSuffix(path, ".rst") ||
		strings.Contains(path, "/docs/")
}

func formatOriginalDiff(diff *git.Diff) string {
	var sb strings.Builder
	for _, file := range diff.Files {
		sb.WriteString(file.Patch)
		sb.WriteString("\n")
	}
	return sb.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

