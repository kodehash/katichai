package review

import (
	"fmt"
	"path/filepath"
	"strings"

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
	// Only analyze files that actually have changes
	changedFiles := make([]string, 0)
	for _, file := range diff.Files {
		// Skip all hidden files and directories (starting with .)
		if strings.HasPrefix(file.Path, ".") || strings.Contains(file.Path, "/.") {
			continue
		}
		
		if file.Additions > 0 || file.Deletions > 0 {
			changedFiles = append(changedFiles, file.Path)
		}
	}

	fileAnalyses, err := r.analyzer.AnalyzeChangedFiles(changedFiles)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}
	result.FileAnalysis = fileAnalyses

	// Build a map of changed line ranges for each file
	changedLineRanges := r.buildChangedLineRanges(diff)

	// 2. Similarity / Duplication Check
	if r.hasContext {
		for filePath, analysis := range fileAnalyses {
			// Get changed line ranges for this file
			lineRanges, hasRanges := changedLineRanges[filePath]
			
			for _, fn := range analysis.Functions {
				// Only check functions that overlap with changed lines
				if hasRanges && !r.functionOverlapsChanges(fn, lineRanges) {
					continue // Skip functions that weren't modified
				}
				
				// Skip small functions (below minimum threshold)
				minLines := r.config.Analysis.MinDuplicateLines
				if fn.LOC < minLines {
					continue // Too small to be a meaningful duplicate
				}
				
				// Create code snippet with actual function body for semantic comparison
				codeSnippet := r.createCodeSnippet(fn, filePath, analysis.Language)

				// Get thresholds from config
				duplicateThreshold := float32(r.config.Analysis.DuplicateThreshold)
				refactorThreshold := float32(r.config.Analysis.RefactorThreshold)

				// Check for duplicates (High threshold - 90%)
				duplicates, err := r.dupDetector.DetectDuplicatesWithThreshold(codeSnippet, filePath, fn.Name, duplicateThreshold)
				if err == nil && len(duplicates) > 0 {
					// Filter by minimum LOC
					filtered := r.filterByMinimumSize(duplicates, minLines)
					if len(filtered) > 0 {
						key := fmt.Sprintf("%s:%s", filePath, fn.Name)
						result.Duplicates[key] = filtered
					}
				}

				// Check for refactoring opportunities (80-90%)
				candidates, err := r.dupDetector.DetectDuplicatesWithThreshold(codeSnippet, filePath, fn.Name, refactorThreshold)
				if err == nil && len(candidates) > 0 {
					// Filter: only include if below duplicate threshold and above min LOC
					uniqueCandidates := make([]embeddings.SimilarityResult, 0)
					for _, c := range candidates {
						candidateLOC := c.EndLine - c.StartLine + 1
						if c.Similarity < duplicateThreshold && candidateLOC >= minLines {
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

// LineRange represents a range of lines
type LineRange struct {
	Start int
	End   int
}

// buildChangedLineRanges extracts which lines were changed in each file
func (r *Reviewer) buildChangedLineRanges(diff *git.Diff) map[string][]LineRange {
	ranges := make(map[string][]LineRange)
	
	for _, file := range diff.Files {
		fileRanges := make([]LineRange, 0)
		
		if file.Patch == "" {
			continue
		}
		
		// Parse the patch to find changed line numbers
		lines := strings.Split(file.Patch, "\n")
		currentLine := 0
		
		for _, line := range lines {
			// Look for @@ markers that indicate line numbers
			// Format: @@ -old_start,old_count +new_start,new_count @@
			if strings.HasPrefix(line, "@@") {
				parts := strings.Split(line, " ")
				for _, part := range parts {
					if strings.HasPrefix(part, "+") {
						// Parse new line numbers
						numPart := strings.TrimPrefix(part, "+")
						numPart = strings.Split(numPart, ",")[0] // Get start line
						var startLine int
						fmt.Sscanf(numPart, "%d", &startLine)
						currentLine = startLine
						break
					}
				}
			} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				// This is an added line
				if currentLine > 0 {
					fileRanges = append(fileRanges, LineRange{Start: currentLine, End: currentLine})
				}
				currentLine++
			} else if !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "@@") {
				// Context line
				currentLine++
			}
		}
		
		// Merge adjacent ranges
		if len(fileRanges) > 0 {
			merged := make([]LineRange, 0)
			current := fileRanges[0]
			
			for i := 1; i < len(fileRanges); i++ {
				if fileRanges[i].Start <= current.End+1 {
					// Merge
					current.End = fileRanges[i].End
				} else {
					merged = append(merged, current)
					current = fileRanges[i]
				}
			}
			merged = append(merged, current)
			ranges[file.Path] = merged
		}
	}
	
	return ranges
}

// functionOverlapsChanges checks if a function overlaps with any changed lines
func (r *Reviewer) functionOverlapsChanges(fn analysis.FunctionInfo, ranges []LineRange) bool {
	for _, r := range ranges {
		// Check if function overlaps with this range
		if fn.StartLine <= r.End && fn.EndLine >= r.Start {
			return true
		}
	}
	return false
}

// createCodeSnippet creates a semantic code snippet for embedding
func (r *Reviewer) createCodeSnippet(fn analysis.FunctionInfo, filePath string, language string) string {
	var snippet strings.Builder
	
	// Include language and context
	snippet.WriteString(fmt.Sprintf("Language: %s\n", language))
	snippet.WriteString(fmt.Sprintf("Function: %s\n", fn.Name))
	
	// Include parameters for signature matching
	if len(fn.Parameters) > 0 {
		snippet.WriteString(fmt.Sprintf("Parameters: %s\n", strings.Join(fn.Parameters, ", ")))
	}
	
	// Include return type if available
	if fn.ReturnType != "" {
		snippet.WriteString(fmt.Sprintf("Returns: %s\n", fn.ReturnType))
	}
	
	// Include complexity and size metrics
	snippet.WriteString(fmt.Sprintf("Complexity: %d, Lines: %d\n\n", fn.Complexity, fn.LOC))
	
	// IMPORTANT: Include actual function body for semantic comparison
	// This is key for detecting true logic similarity vs just naming patterns
	if fn.Body != "" {
		// Normalize the body: remove comments and extra whitespace for better comparison
		normalizedBody := r.normalizeCode(fn.Body)
		snippet.WriteString("Code:\n")
		snippet.WriteString(normalizedBody)
	} else if fn.Comments != "" {
		// Fallback to comments if body not available
		snippet.WriteString("Comments:\n")
		snippet.WriteString(fn.Comments)
	}
	
	return snippet.String()
}

// normalizeCode normalizes code for better semantic comparison
func (r *Reviewer) normalizeCode(code string) string {
	lines := strings.Split(code, "\n")
	normalized := make([]string, 0)
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip empty lines
		if trimmed == "" {
			continue
		}
		
		// Skip comment-only lines
		if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		
		normalized = append(normalized, trimmed)
	}
	
	return strings.Join(normalized, "\n")
}

// filterByMinimumSize filters similarity results by minimum LOC
func (r *Reviewer) filterByMinimumSize(results []embeddings.SimilarityResult, minLines int) []embeddings.SimilarityResult {
	filtered := make([]embeddings.SimilarityResult, 0)
	
	for _, result := range results {
		// Calculate LOC from line range
		loc := result.EndLine - result.StartLine + 1
		
		// Only include if meets minimum size
		if loc >= minLines {
			filtered = append(filtered, result)
		}
	}
	
	return filtered
}
