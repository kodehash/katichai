package analysis

import (
	"fmt"
	"strings"
)

// DuplicationDetector detects code duplication
type DuplicationDetector struct{}

// NewDuplicationDetector creates a new duplication detector
func NewDuplicationDetector() *DuplicationDetector {
	return &DuplicationDetector{}
}

// DuplicateBlock represents a duplicated code block
type DuplicateBlock struct {
	File1      string `json:"file1"`
	StartLine1 int    `json:"start_line1"`
	EndLine1   int    `json:"end_line1"`
	File2      string `json:"file2"`
	StartLine2 int    `json:"start_line2"`
	EndLine2   int    `json:"end_line2"`
	Lines      int    `json:"lines"`
	Similarity float64 `json:"similarity"`
}

// DetectDuplicates detects duplicate code blocks
func (d *DuplicationDetector) DetectDuplicates(files map[string]*FileAnalysis) []DuplicateBlock {
	duplicates := make([]DuplicateBlock, 0)

	// Simple hash-based duplicate detection
	// For now, just return empty - full implementation would use more sophisticated algorithms
	
	return duplicates
}

// AICodeDetector detects AI-generated code patterns
type AICodeDetector struct{}

// NewAICodeDetector creates a new AI code detector
func NewAICodeDetector() *AICodeDetector {
	return &AICodeDetector{}
}

// AICodePattern represents a detected AI-generated pattern
type AICodePattern struct {
	File        string   `json:"file"`
	StartLine   int      `json:"start_line"`
	EndLine     int      `json:"end_line"`
	Pattern     string   `json:"pattern"`
	Confidence  float64  `json:"confidence"`
	Indicators  []string `json:"indicators"`
}

// DetectAIPatterns detects AI-generated code patterns
func (d *AICodeDetector) DetectAIPatterns(analysis *FileAnalysis) []AICodePattern {
	patterns := make([]AICodePattern, 0)

	// Check for AI-generated patterns
	for _, fn := range analysis.Functions {
		indicators := make([]string, 0)
		confidence := 0.0

		// Check for generic names
		if d.isGenericName(fn.Name) {
			indicators = append(indicators, "Generic function name")
			confidence += 0.2
		}

		// Check for excessive length
		if fn.LOC > 100 {
			indicators = append(indicators, "Excessively long function")
			confidence += 0.3
		}

		// Check for high complexity
		if fn.Complexity > 20 {
			indicators = append(indicators, "Very high complexity")
			confidence += 0.3
		}

		// Check for too many parameters
		if len(fn.Parameters) > 5 {
			indicators = append(indicators, "Too many parameters")
			confidence += 0.2
		}

		if confidence > 0.5 {
			patterns = append(patterns, AICodePattern{
				File:       analysis.FilePath,
				StartLine:  fn.StartLine,
				EndLine:    fn.EndLine,
				Pattern:    "Potentially AI-generated boilerplate",
				Confidence: confidence,
				Indicators: indicators,
			})
		}
	}

	return patterns
}

// isGenericName checks if a name is generic
func (d *AICodeDetector) isGenericName(name string) bool {
	genericNames := []string{
		"Manager", "Helper", "Util", "Processor",
		"Handler", "Service", "Controller", "Provider",
		"Factory", "Builder", "Wrapper", "Adapter",
	}

	nameLower := strings.ToLower(name)
	for _, generic := range genericNames {
		if strings.Contains(nameLower, strings.ToLower(generic)) {
			return true
		}
	}

	return false
}

// StyleChecker checks code style violations
type StyleChecker struct{}

// NewStyleChecker creates a new style checker
func NewStyleChecker() *StyleChecker {
	return &StyleChecker{}
}

// CheckStyle checks for style violations
func (s *StyleChecker) CheckStyle(analysis *FileAnalysis) []Issue {
	issues := make([]Issue, 0)

	// Check function naming
	for _, fn := range analysis.Functions {
		if s.hasInvalidNaming(fn.Name) {
			issues = append(issues, Issue{
				Type:     IssueTypeNaming,
				Severity: SeverityInfo,
				Line:     fn.StartLine,
				Message:  fmt.Sprintf("Function '%s' may not follow naming conventions", fn.Name),
			})
		}
	}

	return issues
}

// hasInvalidNaming checks for invalid naming
func (s *StyleChecker) hasInvalidNaming(name string) bool {
	// Very basic check - can be enhanced
	if len(name) < 2 {
		return true
	}
	
	// Check for all caps (except single letter)
	if len(name) > 1 && strings.ToUpper(name) == name {
		return true
	}

	return false
}
