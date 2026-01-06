package review

import (
	"fmt"
	"sort"
	"strings"

	analysispkg "github.com/katichai/katich/internal/analysis"
	"github.com/katichai/katich/internal/git"
	"github.com/katichai/katich/internal/llm"
)

// RepositorySampler samples repository files intelligently for full repository reviews
type RepositorySampler struct {
	maxTokens     int
	maxFiles      int
	contextLines  int
	skipGenerated bool
	skipTests     bool
}

// NewRepositorySampler creates a new repository sampler with a token budget
func NewRepositorySampler(maxTokens int) *RepositorySampler {
	return &RepositorySampler{
		maxTokens:     maxTokens,
		maxFiles:      MAX_FILES_FULL_REVIEW,
		contextLines:  MAX_CONTEXT_LINES,
		skipGenerated: true,
		skipTests:     false,
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
func (s *RepositorySampler) filterNoise(files []*git.DiffFile, report *SamplingReport) []*git.DiffFile {
	filtered := make([]*git.DiffFile, 0)

	for _, file := range files {
		// Skip all hidden files and directories (starting with .)
		if strings.HasPrefix(file.Path, ".") || strings.Contains(file.Path, "/.") {
			report.FilteredReasons["hidden"]++
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

		// Database/API interaction
		if isDatabaseOrAPI(file.Path) {
			risk.Score += 7
			risk.Reasons = append(risk.Reasons, "database/api")
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

// selectFiles selects top N files by risk score
func (s *RepositorySampler) selectFiles(risks []FileRisk) []FileRisk {
	if len(risks) <= s.maxFiles {
		return risks
	}
	return risks[:s.maxFiles]
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

