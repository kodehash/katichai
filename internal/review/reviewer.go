package review

import (
	"fmt"
	"path/filepath"

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
		for filePath, analysis := range fileAnalyses {
			for _, fn := range analysis.Functions {
				// Only check functions that are new or significantly modified
				// For now, we check all functions in changed files
				
				// Create a simple code snippet representation for embedding
				// In a real implementation, we'd want the actual function body source
				codeSnippet := fmt.Sprintf("// Function: %s\n// File: %s\n%s", 
					fn.Name, filePath, fn.Comments)

				// Check for duplicates (High threshold)
				duplicates, err := r.dupDetector.DetectDuplicates(codeSnippet, filePath, fn.Name)
				if err == nil && len(duplicates) > 0 {
					key := fmt.Sprintf("%s:%s", filePath, fn.Name)
					result.Duplicates[key] = duplicates
				}

				// Check for reuse opportunities (Lower threshold, e.g. 0.70 - 0.85)
				// We use the same detector but need to access search directly or create a secondary detector
				// creating a temporary search for reuse
				// In production, we'd optimize to search once and filter.
				// Here we just use the provider to get embedding and search manually to avoid double embedding cost?
				// Actually embedding provider caches? No.
				// Let's just create a secondary detector for now or reuse logic.
				// We'll trust the provider to be fast enough or cached.
				
				// Reuse thresholds
				reuseThreshold := float32(0.70)
				candidates, err := r.dupDetector.DetectDuplicatesWithThreshold(codeSnippet, filePath, fn.Name, reuseThreshold)
				if err == nil && len(candidates) > 0 {
					// Filter out high items that are already in duplicates
					uniqueCandidates := make([]embeddings.SimilarityResult, 0)
					for _, c := range candidates {
						if c.Similarity < 0.85 { // Only if strictly less than duplicate threshold
							uniqueCandidates = append(uniqueCandidates, c)
						}
					}
					
					if len(uniqueCandidates) > 0 {
						key := fmt.Sprintf("%s:%s", filePath, fn.Name)
						result.ReuseCandidates[key] = uniqueCandidates
					}
				}
			}
		}
	}

	return result, nil
}
