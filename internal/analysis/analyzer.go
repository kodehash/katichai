package analysis

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/katichai/katich/internal/context"
	"github.com/katichai/katich/internal/safepath"
)

// Analyzer performs static analysis on code files
type Analyzer struct {
	rootPath   string
	aiDetector *AICodeDetector
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer(rootPath string) *Analyzer {
	return &Analyzer{
		rootPath:   rootPath,
		aiDetector: NewAICodeDetector(),
	}
}

// AnalysisResult contains analysis results for a repository
type AnalysisResult struct {
	Files          map[string]*FileAnalysis `json:"files"`
	TotalMetrics   CodeMetrics              `json:"total_metrics"`
	IssuesSummary  IssuesSummary            `json:"issues_summary"`
	TopComplexity  []FunctionInfo           `json:"top_complexity"`
	LongestFuncs   []FunctionInfo           `json:"longest_functions"`
}

// IssuesSummary summarizes issues by type and severity
type IssuesSummary struct {
	TotalIssues int                    `json:"total_issues"`
	ByType      map[IssueType]int      `json:"by_type"`
	BySeverity  map[Severity]int       `json:"by_severity"`
}

// AnalyzeRepository analyzes all source files in the repository
func (a *Analyzer) AnalyzeRepository() (*AnalysisResult, error) {
	result := &AnalysisResult{
		Files:         make(map[string]*FileAnalysis),
		TopComplexity: make([]FunctionInfo, 0),
		LongestFuncs:  make([]FunctionInfo, 0),
		IssuesSummary: IssuesSummary{
			ByType:     make(map[IssueType]int),
			BySeverity: make(map[Severity]int),
		},
	}

	// Walk through repository
	err := filepath.Walk(a.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-source files
		if info.IsDir() {
			name := info.Name()
			if strings.HasPrefix(name, ".") ||
				name == "node_modules" ||
				name == "vendor" ||
				name == "dist" ||
				name == "build" ||
				name == "target" {
				return filepath.SkipDir
			}
			return nil
		}

		// Analyze source files (skip secret / env paths)
		if a.isSourceFile(path) {
			relPath, _ := filepath.Rel(a.rootPath, path)
			if safepath.IsBlockedPath(relPath) {
				return nil
			}
			analysis, err := a.analyzeFile(path)
			if err != nil {
				// Log error but continue
				return nil
			}

			result.Files[relPath] = analysis

			// Aggregate metrics
			a.aggregateMetrics(&result.TotalMetrics, analysis.Metrics)

			// Collect issues
			for _, issue := range analysis.Issues {
				result.IssuesSummary.TotalIssues++
				result.IssuesSummary.ByType[issue.Type]++
				result.IssuesSummary.BySeverity[issue.Severity]++
			}

			// Collect top complexity functions
			for _, fn := range analysis.Functions {
				result.TopComplexity = append(result.TopComplexity, fn)
				result.LongestFuncs = append(result.LongestFuncs, fn)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Sort and limit top lists
	result.TopComplexity = a.getTopByComplexity(result.TopComplexity, 10)
	result.LongestFuncs = a.getTopByLength(result.LongestFuncs, 10)

	return result, nil
}

// analyzeFile analyzes a single file
func (a *Analyzer) analyzeFile(filePath string) (*FileAnalysis, error) {
	lang := context.DetectLanguage(filePath)

	var fileAnalysis *FileAnalysis
	var err error

	switch lang {

	case context.LanguageGo:
		parser := NewGoParser()
		fileAnalysis, err = parser.ParseFile(filePath)
	
	case context.LanguageJava:
		parser := NewRegexParser(string(context.LanguageJava))
		fileAnalysis, err = parser.ParseFile(filePath)

	case context.LanguageCSharp:
		parser := NewRegexParser(string(context.LanguageCSharp))
		fileAnalysis, err = parser.ParseFile(filePath)

	case context.LanguageJavaScript:
		parser := NewRegexParser(string(context.LanguageJavaScript))
		fileAnalysis, err = parser.ParseFile(filePath)
		
	case context.LanguageTypeScript:
		parser := NewRegexParser(string(context.LanguageTypeScript))
		fileAnalysis, err = parser.ParseFile(filePath)
		
	case context.LanguagePython:
		// Try AST parser first
		astParser := NewPythonParser()
		fileAnalysis, err = astParser.ParseFile(filePath)
		if err != nil {
			// Silent fallback to regex parser if AST parsing fails
			regexParser := NewRegexParser(string(context.LanguagePython))
			fileAnalysis, err = regexParser.ParseFile(filePath)
		}
	
	default:
		// For unsupported languages, do basic analysis
		fileAnalysis, err = a.basicAnalysis(filePath, string(lang))
	}

	if err != nil {
		return nil, err
	}

	// Run AI detection
	patterns := a.aiDetector.DetectAIPatterns(fileAnalysis)
	for _, pattern := range patterns {
		fileAnalysis.Issues = append(fileAnalysis.Issues, Issue{
			Type:     IssueTypeAIGenerated,
			Severity: SeverityWarning,
			Line:     pattern.StartLine,
			Message:  fmt.Sprintf("%s: %s", pattern.Pattern, strings.Join(pattern.Indicators, ", ")),
			Suggestion: "Review code for potential AI hallucinations or over-engineering",
		})
	}

	// Calculate AI score
	fileAnalysis.AIScore = CalculateAIScore(fileAnalysis, a.aiDetector)

	return fileAnalysis, nil
}

// basicAnalysis performs basic analysis for unsupported languages
func (a *Analyzer) basicAnalysis(filePath string, language string) (*FileAnalysis, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	metrics := CalculateBasicMetrics(string(content))

	// Calculate hash
	hash := sha256.Sum256(content)
	hashStr := fmt.Sprintf("%x", hash)

	return &FileAnalysis{
		FilePath:  filePath,
		Language:  language,
		Hash:      hashStr,
		Metrics:   metrics,
		Functions: make([]FunctionInfo, 0),
		Classes:   make([]ClassInfo, 0),
		Imports:   make([]ImportInfo, 0),
		Issues:    make([]Issue, 0),
	}, nil
}

// isSourceFile checks if a file is a source code file
func (a *Analyzer) isSourceFile(path string) bool {
	return context.IsSourceFile(path)
}

// aggregateMetrics aggregates metrics
func (a *Analyzer) aggregateMetrics(total *CodeMetrics, file CodeMetrics) {
	total.LinesOfCode += file.LinesOfCode
	total.LinesOfComments += file.LinesOfComments
	total.BlankLines += file.BlankLines
	total.CyclomaticComplexity += file.CyclomaticComplexity
	total.FunctionCount += file.FunctionCount
	total.ClassCount += file.ClassCount
	total.ImportCount += file.ImportCount

	if file.MaxFunctionLength > total.MaxFunctionLength {
		total.MaxFunctionLength = file.MaxFunctionLength
	}
}

// getTopByComplexity returns top N functions by complexity
func (a *Analyzer) getTopByComplexity(functions []FunctionInfo, n int) []FunctionInfo {
	// Simple bubble sort for top N
	for i := 0; i < len(functions)-1; i++ {
		for j := 0; j < len(functions)-i-1; j++ {
			if functions[j].Complexity < functions[j+1].Complexity {
				functions[j], functions[j+1] = functions[j+1], functions[j]
			}
		}
	}

	if len(functions) > n {
		return functions[:n]
	}
	return functions
}

// getTopByLength returns top N functions by length
func (a *Analyzer) getTopByLength(functions []FunctionInfo, n int) []FunctionInfo {
	// Simple bubble sort for top N
	for i := 0; i < len(functions)-1; i++ {
		for j := 0; j < len(functions)-i-1; j++ {
			if functions[j].LOC < functions[j+1].LOC {
				functions[j], functions[j+1] = functions[j+1], functions[j]
			}
		}
	}

	if len(functions) > n {
		return functions[:n]
	}
	return functions
}

// BuildResultFromFiles constructs a complete AnalysisResult from a Files map,
// recomputing all aggregates (TotalMetrics, IssuesSummary, TopComplexity, LongestFuncs).
// Used by incremental context build to produce a result identical in structure to AnalyzeRepository.
func (a *Analyzer) BuildResultFromFiles(files map[string]*FileAnalysis) *AnalysisResult {
	result := &AnalysisResult{
		Files:         files,
		TopComplexity: make([]FunctionInfo, 0),
		LongestFuncs:  make([]FunctionInfo, 0),
		IssuesSummary: IssuesSummary{
			ByType:     make(map[IssueType]int),
			BySeverity: make(map[Severity]int),
		},
	}

	totalFuncLen := 0
	for _, fa := range files {
		a.aggregateMetrics(&result.TotalMetrics, fa.Metrics)
		for _, issue := range fa.Issues {
			result.IssuesSummary.TotalIssues++
			result.IssuesSummary.ByType[issue.Type]++
			result.IssuesSummary.BySeverity[issue.Severity]++
		}
		for _, fn := range fa.Functions {
			result.TopComplexity = append(result.TopComplexity, fn)
			result.LongestFuncs = append(result.LongestFuncs, fn)
			totalFuncLen += fn.LOC
		}
	}

	if result.TotalMetrics.FunctionCount > 0 {
		result.TotalMetrics.AvgFunctionLength = float64(totalFuncLen) / float64(result.TotalMetrics.FunctionCount)
	}

	result.TopComplexity = a.getTopByComplexity(result.TopComplexity, 10)
	result.LongestFuncs = a.getTopByLength(result.LongestFuncs, 10)
	return result
}

// AnalyzeChangedFiles analyzes only the files that changed in a diff
// Processes files in parallel for better performance, especially for Python files
func (a *Analyzer) AnalyzeChangedFiles(changedFiles []string) (map[string]*FileAnalysis, error) {
	results := make(map[string]*FileAnalysis)
	resultsMutex := &sync.Mutex{}
	
	// Filter source files first
	sourceFiles := make([]string, 0)
	for _, file := range changedFiles {
		if safepath.IsBlockedPath(file) {
			continue
		}
		fullPath := filepath.Join(a.rootPath, file)
		if a.isSourceFile(fullPath) {
			sourceFiles = append(sourceFiles, file)
		}
	}
	
	if len(sourceFiles) == 0 {
		return results, nil
	}
	
	// Process files in parallel with limited concurrency
	// Use a semaphore to limit concurrent goroutines (avoid overwhelming system)
	maxConcurrent := 10 // Process up to 10 files at once
	semaphore := make(chan struct{}, maxConcurrent)
	
	var wg sync.WaitGroup
	processedCount := 0
	processedMutex := &sync.Mutex{}
	
	// Show progress for large batches
	showProgress := len(sourceFiles) > 20
	var lastProgressUpdate time.Time
	lastProgressUpdateMutex := &sync.Mutex{}
	
	for _, file := range sourceFiles {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore
		
		go func(filePath string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore
			
			fullPath := filepath.Join(a.rootPath, filePath)
			analysis, err := a.analyzeFile(fullPath)
			
			if err == nil {
				resultsMutex.Lock()
				results[filePath] = analysis
				resultsMutex.Unlock()
			}
			
			// Update progress counter and show progress
			processedMutex.Lock()
			processedCount++
			currentCount := processedCount
			processedMutex.Unlock()
			
			// Show progress every 500ms for large batches
			if showProgress {
				lastProgressUpdateMutex.Lock()
				shouldUpdate := time.Since(lastProgressUpdate) > 500*time.Millisecond
				if shouldUpdate {
					fmt.Printf("   ⏳ Analyzing files: %d/%d (%.0f%%)\r", currentCount, len(sourceFiles), float64(currentCount)/float64(len(sourceFiles))*100)
					lastProgressUpdate = time.Now()
				}
				lastProgressUpdateMutex.Unlock()
			}
		}(file)
	}
	
	wg.Wait()
	
	// Clear progress line and show completion
	if showProgress {
		// Clear the progress line
		fmt.Print("\r" + strings.Repeat(" ", 50) + "\r")
		fmt.Printf("   ✓ Analyzed %d files\n", len(results))
	}

	return results, nil
}
