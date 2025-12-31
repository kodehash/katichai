package analysis

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ComplexityIssue represents an unnecessarily complex code block or architectural pattern
type ComplexityIssue struct {
	File        string  `json:"file"`
	Line        int     `json:"line"`
	Function    string  `json:"function"`
	Score       float64 `json:"score"` // 0.0-1.0, higher = more unnecessarily complex
	Description string  `json:"description"`
	Reasoning   string  `json:"reasoning"` // Detailed explanation of why it's unnecessarily complex
	Suggestion  string  `json:"suggestion"`
	Type        string  `json:"type"` // "code" or "architectural"
}

// ComplexityDetector detects unnecessarily complex code and architectural patterns
type ComplexityDetector struct{}

// NewComplexityDetector creates a new complexity detector
func NewComplexityDetector() *ComplexityDetector {
	return &ComplexityDetector{}
}

// DetectComplexity analyzes file analysis and detects unnecessarily complex patterns
func (cd *ComplexityDetector) DetectComplexity(filePath string, fileAnalysis *FileAnalysis) []ComplexityIssue {
	issues := make([]ComplexityIssue, 0)

	for _, fn := range fileAnalysis.Functions {
		// Check for over-nesting
		if nestingScore := cd.checkNestingDepth(fn); nestingScore > 0.3 {
			issues = append(issues, ComplexityIssue{
				File:        filePath,
				Line:        fn.StartLine,
				Function:    fn.Name,
				Score:       nestingScore,
				Description: fmt.Sprintf("Function has excessive nesting levels (complexity: %d)", fn.Complexity),
				Suggestion:  "Consider refactoring to reduce nesting depth by extracting functions or using early returns",
				Type:        "code",
			})
		}

		// Check for verbose implementations
		if verboseScore := cd.checkVerboseImplementation(fn); verboseScore > 0.3 {
			issues = append(issues, ComplexityIssue{
				File:        filePath,
				Line:        fn.StartLine,
				Function:    fn.Name,
				Score:       verboseScore,
				Description: fmt.Sprintf("Function is unnecessarily verbose (%d lines for simple logic)", fn.LOC),
				Suggestion:  "Consider simplifying the implementation - this could likely be written more concisely",
				Type:        "code",
			})
		}

		// Check for high cyclomatic complexity
		if complexityScore := cd.checkCyclomaticComplexity(fn); complexityScore > 0.3 {
			issues = append(issues, ComplexityIssue{
				File:        filePath,
				Line:        fn.StartLine,
				Function:    fn.Name,
				Score:       complexityScore,
				Description: fmt.Sprintf("High cyclomatic complexity (%d) - function may be doing too much", fn.Complexity),
				Suggestion:  "Consider breaking this function into smaller, focused functions",
				Type:        "code",
			})
		}

		// Check for unnecessary indirection (function call chains)
		if indirectionScore := cd.checkIndirection(fn); indirectionScore > 0.3 {
			issues = append(issues, ComplexityIssue{
				File:        filePath,
				Line:        fn.StartLine,
				Function:    fn.Name,
				Score:       indirectionScore,
				Description: "Function contains excessive indirection or wrapper calls",
				Suggestion:  "Consider simplifying by removing unnecessary wrapper functions or layers",
				Type:        "code",
			})
		}
	}

	// Check for architectural patterns in file structure
	archIssues := cd.detectArchitecturalComplexity(filePath, fileAnalysis)
	issues = append(issues, archIssues...)

	return issues
}

// checkNestingDepth calculates a score based on nesting depth
func (cd *ComplexityDetector) checkNestingDepth(fn FunctionInfo) float64 {
	// Estimate nesting from complexity and LOC
	// High complexity with moderate LOC suggests deep nesting
	if fn.Complexity > 15 && fn.LOC < 50 {
		return 0.6 // Likely over-nested
	}
	if fn.Complexity > 20 {
		return 0.8 // Very likely over-nested
	}
	return 0.0
}

// checkVerboseImplementation detects unnecessarily verbose code
func (cd *ComplexityDetector) checkVerboseImplementation(fn FunctionInfo) float64 {
	// Simple functions (low complexity) with high LOC suggest verbosity
	if fn.Complexity <= 5 && fn.LOC > 30 {
		return 0.7 // Likely verbose
	}
	if fn.Complexity <= 3 && fn.LOC > 20 {
		return 0.5 // Moderately verbose
	}
	return 0.0
}

// checkCyclomaticComplexity scores based on cyclomatic complexity
func (cd *ComplexityDetector) checkCyclomaticComplexity(fn FunctionInfo) float64 {
	if fn.Complexity > 20 {
		return 0.8 // Very complex
	}
	if fn.Complexity > 15 {
		return 0.5 // Moderately complex
	}
	return 0.0
}

// checkIndirection detects excessive function call chains or wrappers
func (cd *ComplexityDetector) checkIndirection(fn FunctionInfo) float64 {
	// This is a placeholder - would need to analyze function body
	// For now, we'll rely on LLM to detect this pattern
	return 0.0
}

// detectArchitecturalComplexity detects unnecessarily complex architectural patterns
func (cd *ComplexityDetector) detectArchitecturalComplexity(filePath string, fileAnalysis *FileAnalysis) []ComplexityIssue {
	issues := make([]ComplexityIssue, 0)
	pathLower := strings.ToLower(filePath)
	baseName := strings.ToLower(filepath.Base(filePath))

	// Check for multiple layers of abstraction in file path
	pathParts := strings.Split(pathLower, "/")
	layerKeywords := []string{"service", "manager", "handler", "processor", "controller", "facade", "adapter", "delegate"}
	layerCount := 0
	for _, part := range pathParts {
		for _, keyword := range layerKeywords {
			if strings.Contains(part, keyword) {
				layerCount++
				break
			}
		}
	}

	// Multiple layers suggest architectural over-complexity
	if layerCount > 2 {
		issues = append(issues, ComplexityIssue{
			File:        filePath,
			Line:        1,
			Function:    "",
			Score:       0.6,
			Description: fmt.Sprintf("File path suggests multiple layers of abstraction (%d layers detected)", layerCount),
			Suggestion:  "Consider if all these layers are necessary - simpler architecture may suffice",
			Type:        "architectural",
		})
	}

	// Check for design pattern files that might be over-engineered
	patternFiles := []string{
		"factory", "builder", "strategy", "observer", "command", "decorator",
		"adapter", "facade", "proxy", "bridge", "composite", "visitor",
	}
	for _, pattern := range patternFiles {
		if strings.Contains(baseName, pattern) {
		// Check if file is small (suggesting pattern might be unnecessary)
		// Estimate file size from functions
		totalLines := 0
		for _, fn := range fileAnalysis.Functions {
			totalLines += fn.LOC
		}
		if totalLines > 0 && totalLines < 100 {
				issues = append(issues, ComplexityIssue{
					File:        filePath,
					Line:        1,
					Function:    "",
					Score:       0.5,
					Description: fmt.Sprintf("Design pattern file (%s) is relatively small - pattern may be unnecessary", pattern),
					Suggestion:  "Consider if this design pattern adds value or if a simpler approach would work",
					Type:        "architectural",
				})
			}
		}
	}

	return issues
}

