package review

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	analysispkg "github.com/katichai/katich/internal/analysis"
	"github.com/katichai/katich/internal/git"
	"github.com/katichai/katich/internal/llm"
	"github.com/katichai/katich/internal/safepath"
)

// RepositorySampler samples repository files intelligently for full repository reviews
type RepositorySampler struct {
	maxTokens     int
	maxFiles      int
	contextLines  int
	skipGenerated bool
	skipTests     bool
}

// NewRepositorySampler creates a new repository sampler with a token budget and file cap.
// maxFiles controls the upper limit of files selected for review (default: 30).
func NewRepositorySampler(maxTokens int, maxFiles int) *RepositorySampler {
	if maxFiles <= 0 {
		maxFiles = 30
	}
	return &RepositorySampler{
		maxTokens:     maxTokens,
		maxFiles:      maxFiles,
		contextLines:  MAX_CONTEXT_LINES,
		skipGenerated: true,
		skipTests:     false,
	}
}

// AdjustForLargeRepository logs the effective sampling plan for large repositories.
func (s *RepositorySampler) AdjustForLargeRepository(totalFiles int) {
	if totalFiles > 15 {
		percentage := float64(totalFiles) * 0.3
		estimatedFiles := int(percentage)
		if estimatedFiles < 15 {
			estimatedFiles = 15
		}
		if estimatedFiles > s.maxFiles {
			estimatedFiles = s.maxFiles
		}
		fmt.Printf("  📦 Large repository detected (%d files). Processing up to %d files (30%% of important files, max %d).\n", totalFiles, estimatedFiles, s.maxFiles)
	}
}

// SampleRepository samples repository files to fit within the token budget
func (s *RepositorySampler) SampleRepository(diff *git.Diff, analysisResults map[string]*analysispkg.FileAnalysis) (*SampledDiff, *SamplingReport) {
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
	topN := 20
	if len(risks) < topN {
		topN = len(risks)
	}
	report.TopRiskFiles = risks[:topN]

	// Phase 3: Select files to include
	selectedFiles := s.selectFiles(risks)
	// Note: report.SampledFiles will be updated after actual files are included (Phase 4)
	report.FilteredFiles = report.TotalFiles - len(selectedFiles)

	// Phase 4: Sample selected files with score-based token allocation
	sampledDiff := &SampledDiff{
		Files:         make([]*SampledFile, 0),
		OriginalFiles: len(diff.Files),
	}

	numSelected := len(selectedFiles)
	if numSelected == 0 {
		return sampledDiff, report
	}

	// Reserve 10% of budget for overhead
	reservedBudget := s.maxTokens / 10
	availableBudget := s.maxTokens - reservedBudget

	// Allow up to 15% overage for important files
	maxBudgetWithOverage := int(float64(s.maxTokens) * 1.15)

	// Calculate total risk score for proportional distribution
	totalRiskScore := 0
	for _, risk := range selectedFiles {
		// Ensure minimum score of 1 for files with negative scores
		score := risk.Score
		if score < 1 {
			score = 1
		}
		totalRiskScore += score
	}

	// Allocate tokens based on risk score proportion
	tokenAllocations := make([]int, numSelected) // index -> tokens
	minTokensPerFile := 300  // Minimum tokens per file
	maxTokensPerFile := 2000 // Maximum tokens per file (prevent one file from dominating)

	totalAllocated := 0
	for i, risk := range selectedFiles {
		score := risk.Score
		if score < 1 {
			score = 1
		}

		// Calculate proportional allocation
		proportion := float64(score) / float64(totalRiskScore)
		allocated := int(float64(availableBudget) * proportion)

		// Apply min/max constraints
		if allocated < minTokensPerFile {
			allocated = minTokensPerFile
		}
		if allocated > maxTokensPerFile {
			allocated = maxTokensPerFile
		}

		tokenAllocations[i] = allocated
		totalAllocated += allocated
	}

	// If allocations exceed available budget, scale down proportionally
	// But allow up to 15% overage
	if totalAllocated > availableBudget {
		if totalAllocated <= maxBudgetWithOverage {
			// Within 15% overage - allow it
			availableBudget = totalAllocated
		} else {
			// Exceeds 15% overage - scale down to maxBudgetWithOverage
			scaleFactor := float64(maxBudgetWithOverage) / float64(totalAllocated)
			for i := range tokenAllocations {
				tokenAllocations[i] = int(float64(tokenAllocations[i]) * scaleFactor)
				// Ensure minimum is still respected
				if tokenAllocations[i] < minTokensPerFile {
					tokenAllocations[i] = minTokensPerFile
				}
			}
			availableBudget = maxBudgetWithOverage
		}
	}

	// Process files with their allocated token budgets
	tokenBudget := availableBudget

	for i, risk := range selectedFiles {
		if tokenBudget <= 0 {
			break
		}

		// Get allocated budget for this file
		fileBudget := tokenAllocations[i]
		if tokenBudget < fileBudget {
			fileBudget = tokenBudget
		}

		sampled := s.sampleFile(risk.File, analysisResults[risk.File.Path], fileBudget)
		sampled.RiskScore = risk.Score
		sampled.RiskReasons = risk.Reasons

		fileTokens := llm.EstimateTokens(sampled.Content)

		// Include file if it fits (within allocated budget or remaining budget)
		if fileTokens <= fileBudget || (fileTokens <= tokenBudget && fileTokens <= maxTokensPerFile) {
			sampledDiff.Files = append(sampledDiff.Files, sampled)
			tokenBudget -= fileTokens
			sampledDiff.TotalTokens += fileTokens
		} else if tokenBudget >= minTokensPerFile {
			// File exceeded its allocation - try with remaining budget (if sufficient)
			sampled = s.sampleFile(risk.File, analysisResults[risk.File.Path], tokenBudget)
			fileTokens = llm.EstimateTokens(sampled.Content)
			if fileTokens <= tokenBudget {
				sampledDiff.Files = append(sampledDiff.Files, sampled)
				tokenBudget -= fileTokens
				sampledDiff.TotalTokens += fileTokens
			}
		}
	}

	report.TotalTokensAfter = sampledDiff.TotalTokens
	report.SampledFiles = len(sampledDiff.Files) // Update with actual files included

	return sampledDiff, report
}

// filterNoise filters out generated, lock, and binary files
func (s *RepositorySampler) filterNoise(files []*git.DiffFile, report *SamplingReport) []*git.DiffFile {
	filtered := make([]*git.DiffFile, 0)

	for _, file := range files {
		// Skip all hidden files and directories (starting with .)
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

// calculateRiskScores calculates risk scores for files in full repository context
func (s *RepositorySampler) calculateRiskScores(files []*git.DiffFile, analysisResults map[string]*analysispkg.FileAnalysis) []FileRisk {
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

		// Database/API interaction (path-based check - keep for backward compatibility)
		if isDatabaseOrAPI(file.Path) {
			risk.Score += 7
			risk.Reasons = append(risk.Reasons, "database/api")
		}

		// Database/ORM calls in content (more accurate detection)
		if analysis != nil && hasDatabaseOrORMCalls(analysis) {
			risk.Score += 7 // Same weight as path-based check
			risk.Reasons = append(risk.Reasons, "database/orm-calls")
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

			// AI-generated patterns
			for _, issue := range analysis.Issues {
				if issue.Type == analysispkg.IssueTypeAIGenerated {
					risk.Score += 3
					risk.Reasons = append(risk.Reasons, "ai-generated")
					break
				}
			}
		}

		// Large files get priority (more code to review)
		if file.Additions > 200 {
			risk.Score += 2
			risk.Reasons = append(risk.Reasons, "large-file")
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

// hasDatabaseOrORMCalls checks if a file contains database or ORM calls
func hasDatabaseOrORMCalls(analysis *analysispkg.FileAnalysis) bool {
	// Check imports for database/ORM libraries
	for _, imp := range analysis.Imports {
		importPath := strings.ToLower(imp.Path)
		
		// Python database/ORM imports
		if strings.Contains(importPath, "sqlalchemy") ||
			strings.Contains(importPath, "django.db") ||
			strings.Contains(importPath, "peewee") ||
			strings.Contains(importPath, "sqlite3") ||
			strings.Contains(importPath, "psycopg2") ||
			strings.Contains(importPath, "mysql") ||
			strings.Contains(importPath, "pymongo") ||
			strings.Contains(importPath, "sqlmodel") {
			return true
		}
		
		// JavaScript/TypeScript database/ORM imports
		if strings.Contains(importPath, "sequelize") ||
			strings.Contains(importPath, "prisma") ||
			strings.Contains(importPath, "typeorm") ||
			strings.Contains(importPath, "mongoose") ||
			strings.Contains(importPath, "knex") ||
			strings.Contains(importPath, "bookshelf") {
			return true
		}
		
		// Go database/ORM imports
		if strings.Contains(importPath, "database/sql") ||
			strings.Contains(importPath, "gorm.io/gorm") ||
			strings.Contains(importPath, "github.com/jmoiron/sqlx") ||
			strings.Contains(importPath, "gorm.io/driver") {
			return true
		}
		
		// Java database/ORM imports
		if strings.Contains(importPath, "javax.persistence") ||
			strings.Contains(importPath, "org.hibernate") ||
			strings.Contains(importPath, "org.springframework.data") ||
			strings.Contains(importPath, "jakarta.persistence") {
			return true
		}
	}
	
	// Check function bodies for database/ORM method calls
	dbMethodPatterns := []string{
		// Python patterns
		`\.query\(`, `\.filter\(`, `\.get\(`, `\.save\(`, `\.delete\(`, `\.create\(`, `\.update\(`,
		`execute\(`, `cursor\(`, `\.session\(`, `\.commit\(`, `\.rollback\(`,
		// JavaScript/TypeScript patterns
		`\.findOne\(`, `\.findAll\(`, `\.create\(`, `\.update\(`, `\.destroy\(`, `\.save\(`, `\.query\(`,
		`\.find\(`, `\.findById\(`, `\.findOneAndUpdate\(`, `\.findOneAndDelete\(`,
		// Go patterns
		`\.Query\(`, `\.QueryRow\(`, `\.Exec\(`, `\.First\(`, `\.Find\(`, `\.Create\(`, `\.Save\(`,
		`\.Update\(`, `\.Delete\(`, `\.Where\(`, `\.Select\(`,
		// Java patterns
		`\.save\(`, `\.findById\(`, `\.findAll\(`, `\.delete\(`, `\.persist\(`, `\.merge\(`,
		`\.createQuery\(`, `\.getResultList\(`, `\.executeUpdate\(`,
	}
	
	// Compile regex patterns for method calls
	patterns := make([]*regexp.Regexp, 0, len(dbMethodPatterns))
	for _, pattern := range dbMethodPatterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err == nil {
			patterns = append(patterns, re)
		}
	}
	
	// Check all function bodies
	for _, fn := range analysis.Functions {
		if fn.Body == "" {
			continue
		}
		
		bodyLower := strings.ToLower(fn.Body)
		
		// Check against compiled patterns
		for _, pattern := range patterns {
			if pattern.MatchString(fn.Body) {
				return true
			}
		}
		
		// Additional simple string checks for common patterns
		if strings.Contains(bodyLower, "db.query") ||
			strings.Contains(bodyLower, "db.execute") ||
			strings.Contains(bodyLower, "db.session") ||
			strings.Contains(bodyLower, "model.save") ||
			strings.Contains(bodyLower, "model.create") ||
			strings.Contains(bodyLower, "model.update") ||
			strings.Contains(bodyLower, "model.delete") ||
			strings.Contains(bodyLower, "orm.query") ||
			strings.Contains(bodyLower, "orm.save") {
			return true
		}
	}
	
	return false
}

// selectFiles selects files based on percentage of total, capped by s.maxFiles.
func (s *RepositorySampler) selectFiles(risks []FileRisk) []FileRisk {
	totalFiltered := len(risks)

	// If 15 or fewer files, include all (up to the user-defined cap)
	if totalFiltered <= 15 {
		if totalFiltered > s.maxFiles {
			return risks[:s.maxFiles]
		}
		return risks
	}

	// For >15 files: 30% of total, minimum 15, capped at s.maxFiles
	percentage := float64(totalFiltered) * 0.3
	maxFiles := int(percentage)
	if maxFiles < 15 {
		maxFiles = 15
	}
	if maxFiles > s.maxFiles {
		maxFiles = s.maxFiles
	}

	return risks[:maxFiles]
}

// sampleFile samples a single file to reduce its size
func (s *RepositorySampler) sampleFile(file *git.DiffFile, analysis *analysispkg.FileAnalysis, tokenBudget int) *SampledFile {
	sampled := &SampledFile{
		Path:      file.Path,
		Status:    file.Status,
		Additions: file.Additions,
		Deletions: file.Deletions,
	}

	isPython := strings.HasSuffix(file.Path, ".py")

	// For full repository review, we include full file content
	// but truncate if it exceeds token budget
	if file.Patch != "" {
		// Extract content from patch (remove +++ header and + prefixes)
		lines := strings.Split(file.Patch, "\n")
		var content strings.Builder
		for _, line := range lines {
			if strings.HasPrefix(line, "+++") {
				continue
			}
			if strings.HasPrefix(line, "+") {
				content.WriteString(line[1:])
				content.WriteString("\n")
			}
		}
		sampled.Content = content.String()
		
		// Python-specific optimization: remove excessive blank lines and docstrings
		if isPython {
			sampled.Content = s.optimizePythonContent(sampled.Content)
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

// optimizePythonContent optimizes Python content to reduce token usage
func (s *RepositorySampler) optimizePythonContent(content string) string {
	lines := strings.Split(content, "\n")
	optimized := make([]string, 0)
	skipBlankLines := 0
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip excessive consecutive blank lines (keep max 1)
		if trimmed == "" {
			skipBlankLines++
			if skipBlankLines <= 1 {
				optimized = append(optimized, line)
			}
			continue
		}
		skipBlankLines = 0
		
		// Truncate very long docstrings (keep first 3 lines max)
		if strings.HasPrefix(trimmed, `"""`) || strings.HasPrefix(trimmed, `'''`) {
			quote := `"""`
			if strings.HasPrefix(trimmed, `'''`) {
				quote = `'''`
			}
			
			// Check if this is a single-line docstring
			if strings.HasSuffix(trimmed, quote) && len(trimmed) > 6 {
				// Single line docstring - truncate if too long
				if len(trimmed) > 200 {
					optimized = append(optimized, quote+trimmed[len(quote):200]+"... [truncated]"+quote)
					continue
				}
			}
		}
		
		optimized = append(optimized, line)
	}
	
	return strings.Join(optimized, "\n")
}

