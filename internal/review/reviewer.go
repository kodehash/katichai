package review

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/katichai/katich/internal/analysis"
	"github.com/katichai/katich/internal/config"
	"github.com/katichai/katich/internal/embeddings"
	"github.com/katichai/katich/internal/git"
)

// Reviewer orchestrates the code review process
type Reviewer struct {
	rootPath     string
	config       *config.Config
	analyzer     *analysis.Analyzer
	embProvider  embeddings.EmbeddingProvider
	dupDetector  *embeddings.DuplicateDetector
	hasContext   bool
}

// ReviewResult contains the results of a review
type ReviewResult struct {
	FileAnalysis          map[string]*analysis.FileAnalysis
	Duplicates            map[string][]embeddings.SimilarityResult
	ReuseCandidates       map[string][]embeddings.SimilarityResult
	Metrics               analysis.CodeMetrics
	SimilarityCheckLimited bool   // True if similarity check was limited to fewer files
	SimilarityCheckReason  string // Reason why similarity check was limited/disabled
}

// NewReviewer creates a new reviewer
func NewReviewer(rootPath string, cfg *config.Config) *Reviewer {
	r := &Reviewer{
		rootPath: rootPath,
		config:   cfg,
		analyzer: analysis.NewAnalyzer(rootPath),
	}

	// Try to load context and embeddings
	r.loadContext()

	return r
}

// loadContext loads the embedding index and initializes providers
func (r *Reviewer) loadContext() {
	indexPath := filepath.Join(r.rootPath, ".katich", "embeddings.json")
	index, err := embeddings.LoadIndex(indexPath)
	if err != nil {
		// No context found, that's okay - we'll skip similarity features
		return
	}

	// Resolve OpenAI key for embeddings: prefer dedicated embeddings key, fall back to LLM key
	embeddingsAPIKey := r.config.Embeddings.APIKey
	if embeddingsAPIKey == "" {
		embeddingsAPIKey = r.config.LLM.APIKey
	}

	// Initialize embedding provider
	r.embProvider = embeddings.NewHybridProvider(
		"http://localhost:11434",
		"nomic-embed-text",
		embeddingsAPIKey,
		"text-embedding-3-small",
	)

	// Initialize duplicate detector
	r.dupDetector = embeddings.NewDuplicateDetector(index, r.embProvider, 0.85)
	r.hasContext = true
}

// ReviewDiff reviews changes in a diff
func (r *Reviewer) ReviewDiff(diff *git.Diff) (*ReviewResult, error) {
	result := &ReviewResult{
		FileAnalysis:    make(map[string]*analysis.FileAnalysis),
		Duplicates:      make(map[string][]embeddings.SimilarityResult),
		ReuseCandidates: make(map[string][]embeddings.SimilarityResult),
	}

	// 1. Static Analysis
	changedFiles := make([]string, 0)
	for _, file := range diff.Files {
		changedFiles = append(changedFiles, file.Path)
	}

	fileAnalyses, err := r.analyzer.AnalyzeChangedFiles(changedFiles)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}
	result.FileAnalysis = fileAnalyses

	// 2. Similarity / Duplication Check
	// For large reviews (>15 files), limit similarity check to top 15 files by risk score
	// This provides value while maintaining performance
	similarityLimited := false
	similarityReason := ""
	filesToCheck := make(map[string]*analysis.FileAnalysis)
	
	if !r.hasContext {
		similarityLimited = true
		similarityReason = "no context available"
	} else if len(fileAnalyses) > 15 {
		// Select top 15 files by risk score
		selectedFiles := r.selectTopFilesForSimilarityCheck(diff.Files, fileAnalyses, 15)
		for _, filePath := range selectedFiles {
			if analysis, exists := fileAnalyses[filePath]; exists {
				filesToCheck[filePath] = analysis
			}
		}
		similarityLimited = true
		similarityReason = fmt.Sprintf("limited to top 15 files (out of %d total)", len(fileAnalyses))
	} else {
		// Check all files if 15 or fewer
		filesToCheck = fileAnalyses
	}
	
	if r.hasContext && len(filesToCheck) > 0 {
		for filePath, fileAnalysis := range filesToCheck {
			// Create trivial detector for this language
			trivialDetector := analysis.NewTrivialPatternDetector(fileAnalysis.Language)
			
			for _, fn := range fileAnalysis.Functions {
				// FILTER 1: Minimum lines check
				if fn.LOC < r.config.Analysis.MinFunctionLines {
					continue // Skip small functions
				}
				
				// FILTER 2: Trivial pattern check
				if r.config.Analysis.IgnoreTrivialPatterns && trivialDetector.IsTrivial(fn) {
					continue // Skip trivial patterns
				}
				
				// Create embedding from function body (semantic content)
				codeSnippet := r.createSemanticSnippet(fn, filePath, fileAnalysis.Language)
				
				// Check for duplicates with configurable threshold
				duplicates, err := r.dupDetector.DetectDuplicatesWithThreshold(
					codeSnippet, 
					filePath, 
					fn.Name, 
					float32(r.config.Analysis.DuplicateThreshold),
				)
				
				if err == nil && len(duplicates) > 0 {
					// Further filter: only report if LOC and complexity are similar
					validDuplicates := r.filterByLogicSimilarity(fn, duplicates)
					
					if len(validDuplicates) > 0 {
						key := fmt.Sprintf("%s:%s", filePath, fn.Name)
						result.Duplicates[key] = validDuplicates
					}
				}
				
				// Check for similar logic (lower threshold)
				if r.config.Analysis.SimilarityThreshold < r.config.Analysis.DuplicateThreshold {
					candidates, err := r.dupDetector.DetectDuplicatesWithThreshold(
						codeSnippet,
						filePath,
						fn.Name,
						float32(r.config.Analysis.SimilarityThreshold),
					)
					
					if err == nil && len(candidates) > 0 {
						// Filter out items already in duplicates
						uniqueCandidates := make([]embeddings.SimilarityResult, 0)
						for _, c := range candidates {
							if c.Similarity < float32(r.config.Analysis.DuplicateThreshold) {
								uniqueCandidates = append(uniqueCandidates, c)
							}
						}
						
						// Further semantic filtering
						validCandidates := r.filterByLogicSimilarity(fn, uniqueCandidates)
						
						if len(validCandidates) > 0 {
							key := fmt.Sprintf("%s:%s", filePath, fn.Name)
							result.ReuseCandidates[key] = validCandidates
						}
					}
				}
			}
		}
	}
	
	// Store similarity check status in result for reporting
	if similarityLimited {
		result.SimilarityCheckLimited = true
		result.SimilarityCheckReason = similarityReason
	}

	return result, nil
}

// selectTopFilesForSimilarityCheck selects the top N files by risk score for similarity checking
func (r *Reviewer) selectTopFilesForSimilarityCheck(diffFiles []*git.DiffFile, fileAnalyses map[string]*analysis.FileAnalysis, maxFiles int) []string {
	// Create a map of file paths to diff files for risk scoring
	diffFileMap := make(map[string]*git.DiffFile)
	for _, file := range diffFiles {
		diffFileMap[file.Path] = file
	}
	
	// Calculate risk scores for all files
	type fileRiskScore struct {
		filePath string
		score    int
	}
	
	risks := make([]fileRiskScore, 0, len(fileAnalyses))
	
	for filePath, fileAnalysis := range fileAnalyses {
		score := 0
		diffFile := diffFileMap[filePath]
		
		// Security-sensitive paths
		if isSecuritySensitive(filePath) {
			score += 10
		}
		
		// Core business logic
		if isCoreBusinessLogic(filePath) {
			score += 8
		}
		
		// Database/API interaction
		if isDatabaseOrAPI(filePath) {
			score += 7
		}
		
		// High complexity from static analysis
		highComplexityCount := 0
		for _, fn := range fileAnalysis.Functions {
			if fn.Complexity > 15 {
				highComplexityCount++
			}
		}
		if highComplexityCount > 0 {
			score += 3
		}
		
		// AI-generated patterns
		for _, issue := range fileAnalysis.Issues {
			if issue.Type == analysis.IssueTypeAIGenerated {
				score += 3
				break
			}
		}
		
		// Large files get priority
		if diffFile != nil && diffFile.Additions > 200 {
			score += 2
		}
		
		// Lower priority: tests
		if isTestFile(filePath) {
			score -= 5
		}
		
		// Lower priority: docs
		if isDocFile(filePath) {
			score -= 3
		}
		
		risks = append(risks, fileRiskScore{
			filePath: filePath,
			score:    score,
		})
	}
	
	// Sort by score (highest first)
	sort.Slice(risks, func(i, j int) bool {
		return risks[i].score > risks[j].score
	})
	
	// Select top N
	selected := make([]string, 0, maxFiles)
	for i := 0; i < len(risks) && i < maxFiles; i++ {
		selected = append(selected, risks[i].filePath)
	}
	
	return selected
}

// createSemanticSnippet creates an embedding-friendly representation focusing on logic
func (r *Reviewer) createSemanticSnippet(fn analysis.FunctionInfo, filePath string, language string) string {
	// Use the actual function body for better semantic matching
	body := fn.Body
	
	// Normalize whitespace
	body = regexp.MustCompile(`\s+`).ReplaceAllString(body, " ")
	
	// Create snippet with context
	snippet := fmt.Sprintf("Language: %s\nFunction: %s\nComplexity: %d\nLOC: %d\nBody: %s",
		language, fn.Name, fn.Complexity, fn.LOC, body)
	
	return snippet
}

// filterByLogicSimilarity filters results to only include semantic/logic similarities
func (r *Reviewer) filterByLogicSimilarity(fn analysis.FunctionInfo, results []embeddings.SimilarityResult) []embeddings.SimilarityResult {
	filtered := make([]embeddings.SimilarityResult, 0)
	
	for _, result := range results {
		// Skip if LOC difference is huge (different complexity)
		// This helps filter out false positives
		locDiff := abs(fn.LOC - result.LOC)
		if float64(locDiff) / float64(fn.LOC) > 0.5 {
			// More than 50% LOC difference - likely not same logic
			continue
		}
		
		// Complexity should be similar for similar logic
		complexityDiff := abs(fn.Complexity - result.Complexity)
		if complexityDiff > 5 {
			// Very different complexity - different logic
			continue
		}
		
		filtered = append(filtered, result)
	}
	
	return filtered
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
