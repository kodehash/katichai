package analysis

import (
	"strings"
	"unicode"
)

// DriftDetector checks for code style drift and inconsistencies
type DriftDetector struct {
}

// NewDriftDetector creates a detector
func NewDriftDetector() *DriftDetector {
	return &DriftDetector{}
}

// DriftReport contains findings about code style
type DriftReport struct {
	Issues []Issue
}

// CheckDrift analyzes code for naming and error handling drift
func (d *DriftDetector) CheckDrift(code string, language string) DriftReport {
	report := DriftReport{}
	
	// Only support Go for now for deep pattern matching
	if language != "Go" && !strings.HasSuffix(language, ".go") {
		return report
	}

	lines := strings.Split(code, "\n")
	
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// 1. Variable Naming: Check for snake_case assignments in Go (Go prefers camelCase)
		// Pattern: varname := value OR var varname type
		// This is a naive heuristic
		if strings.Contains(trimmed, ":=") {
			parts := strings.Split(trimmed, ":=")
			varName := strings.TrimSpace(parts[0])
			if isSnakeCase(varName) && len(varName) > 3 {
				report.Issues = append(report.Issues, Issue{
					Type:     IssueTypeStyleViolation,
					Severity: SeverityWarning, // Warning, not Error (Drift)
					Line:     i + 1,
					Message:  "Possible naming drift: Go typically uses camelCase, but snake_case detected: " + varName,
				})
			}
		}

		// 2. Error Handling: Check for unusual error handling
		// Standard: if err != nil
		// Non-Standard: if nil != err, or bare 'check(err)'
		if strings.Contains(trimmed, "if nil != err") {
			report.Issues = append(report.Issues, Issue{
				Type:     IssueTypeStyleViolation,
				Severity: SeverityInfo,
				Line:     i + 1,
				Message:  "Unusual error check: 'if nil != err'. Prevailing pattern is 'if err != nil'.",
			})
		}
	}
	
	return report
}

func isSnakeCase(s string) bool {
	// Must contain underscore
	if !strings.Contains(s, "_") {
		return false
	}
	// Must be lowercase
	for _, r := range s {
		if unicode.IsUpper(r) {
			return false
		}
	}
	return true
}
