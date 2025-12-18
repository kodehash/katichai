package review

import (
	"fmt"
	"path/filepath"
	"regexp"

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
	FileAnalysis    map[string]*analysis.FileAnalysis
	Duplicates      map[string][]embeddings.SimilarityResult
	ReuseCandidates map[string][]embeddings.SimilarityResult
	Metrics         analysis.CodeMetrics
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

	// Initialize embedding provider
	// TODO: Make this configurable or load from stored context config
	r.embProvider = embeddings.NewHybridProvider(
		"http://localhost:11434",
		"nomic-embed-text",
		r.config.LLM.APIKey,
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
	if r.hasContext {
		for filePath, fileAnalysis := range fileAnalyses {
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

	return result, nil
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
