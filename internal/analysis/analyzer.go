package analysis

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/katichai/katich/internal/context"
)

// Analyzer performs static analysis on code files
type Analyzer struct {
	rootPath string
}

// NewAnalyzer creates a new analyzer
func NewAnalyzer(rootPath string) *Analyzer {
	return &Analyzer{
		rootPath: rootPath,
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

		// Analyze source files
		if a.isSourceFile(path) {
			analysis, err := a.analyzeFile(path)
			if err != nil {
				// Log error but continue
				return nil
			}

			relPath, _ := filepath.Rel(a.rootPath, path)
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

	switch lang {
	case context.LanguageGo:
		parser := NewGoParser()
		return parser.ParseFile(filePath)
	
	// Add more language parsers here
	// case context.LanguageJavaScript, context.LanguageTypeScript:
	//     parser := NewJSParser()
	//     return parser.ParseFile(filePath)
	
	default:
		// For unsupported languages, do basic analysis
		return a.basicAnalysis(filePath, string(lang))
	}
}

// basicAnalysis performs basic analysis for unsupported languages
func (a *Analyzer) basicAnalysis(filePath string, language string) (*FileAnalysis, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	metrics := CalculateBasicMetrics(string(content))

	return &FileAnalysis{
		FilePath:  filePath,
		Language:  language,
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

// AnalyzeChangedFiles analyzes only the files that changed in a diff
func (a *Analyzer) AnalyzeChangedFiles(changedFiles []string) (map[string]*FileAnalysis, error) {
	results := make(map[string]*FileAnalysis)

	for _, file := range changedFiles {
		fullPath := filepath.Join(a.rootPath, file)
		
		if !a.isSourceFile(fullPath) {
			continue
		}

		analysis, err := a.analyzeFile(fullPath)
		if err != nil {
			continue
		}

		results[file] = analysis
	}

	return results, nil
}
