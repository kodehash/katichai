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
