package analysis

// CodeMetrics represents metrics for a code file or function
type CodeMetrics struct {
	LinesOfCode          int     `json:"lines_of_code"`
	LinesOfComments      int     `json:"lines_of_comments"`
	BlankLines           int     `json:"blank_lines"`
	CyclomaticComplexity int     `json:"cyclomatic_complexity"`
	FunctionCount        int     `json:"function_count"`
	ClassCount           int     `json:"class_count"`
	ImportCount          int     `json:"import_count"`
	MaxFunctionLength    int     `json:"max_function_length"`
	AvgFunctionLength    float64 `json:"avg_function_length"`
}

// FunctionInfo represents information about a function
type FunctionInfo struct {
	Name       string   `json:"name"`
	StartLine  int      `json:"start_line"`
	EndLine    int      `json:"end_line"`
	LOC        int      `json:"loc"`
	Complexity int      `json:"complexity"`
	Parameters []string `json:"parameters"`
	ReturnType string   `json:"return_type,omitempty"`
	IsExported bool     `json:"is_exported"`
	Comments   string   `json:"comments,omitempty"`
}

// ClassInfo represents information about a class/struct
type ClassInfo struct {
	Name       string         `json:"name"`
	StartLine  int            `json:"start_line"`
	EndLine    int            `json:"end_line"`
	Methods    []FunctionInfo `json:"methods"`
	Fields     []FieldInfo    `json:"fields"`
	IsExported bool           `json:"is_exported"`
	Comments   string         `json:"comments,omitempty"`
}

// FieldInfo represents a class field/property
type FieldInfo struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

// ImportInfo represents an import statement
type ImportInfo struct {
	Path  string `json:"path"`
	Alias string `json:"alias,omitempty"`
}

// FileAnalysis represents the complete analysis of a file
type FileAnalysis struct {
	FilePath   string         `json:"file_path"`
	Language   string         `json:"language"`
	Metrics    CodeMetrics    `json:"metrics"`
	Functions  []FunctionInfo `json:"functions"`
	Classes    []ClassInfo    `json:"classes"`
	Imports    []ImportInfo   `json:"imports"`
	Issues     []Issue        `json:"issues,omitempty"`
}

// Issue represents a code quality issue
type Issue struct {
	Type        IssueType `json:"type"`
	Severity    Severity  `json:"severity"`
	Line        int       `json:"line"`
	Column      int       `json:"column,omitempty"`
	Message     string    `json:"message"`
	Suggestion  string    `json:"suggestion,omitempty"`
}

// IssueType categorizes issues
type IssueType string

const (
	IssueTypeComplexity      IssueType = "complexity"
	IssueTypeFunctionLength  IssueType = "function_length"
	IssueTypeNaming          IssueType = "naming"
	IssueTypeDuplication     IssueType = "duplication"
	IssueTypeUnusedCode      IssueType = "unused_code"
	IssueTypeStyleViolation  IssueType = "style_violation"
)

// Severity indicates issue severity
type Severity string

const (
	SeverityInfo    Severity = "info"
	SeverityWarning Severity = "warning"
	SeverityError   Severity = "error"
)

// CalculateBasicMetrics calculates basic metrics from source code
func CalculateBasicMetrics(content string) CodeMetrics {
	lines := splitLines(content)
	
	metrics := CodeMetrics{}
	
	for _, line := range lines {
		trimmed := trimWhitespace(line)
		
		if trimmed == "" {
			metrics.BlankLines++
		} else if isComment(trimmed) {
			metrics.LinesOfComments++
		} else {
			metrics.LinesOfCode++
		}
	}
	
	return metrics
}

// splitLines splits content into lines
func splitLines(content string) []string {
	lines := make([]string, 0)
	current := ""
	
	for _, ch := range content {
		if ch == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(ch)
		}
	}
	
	if current != "" {
		lines = append(lines, current)
	}
	
	return lines
}

// trimWhitespace removes leading/trailing whitespace
func trimWhitespace(s string) string {
	start := 0
	end := len(s)
	
	// Trim leading
	for start < len(s) && isWhitespace(rune(s[start])) {
		start++
	}
	
	// Trim trailing
	for end > start && isWhitespace(rune(s[end-1])) {
		end--
	}
	
	return s[start:end]
}

// isWhitespace checks if a rune is whitespace
func isWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\r' || r == '\n'
}

// isComment checks if a line is a comment (basic check)
func isComment(line string) bool {
	if len(line) < 2 {
		return false
	}
	
	// Check for common comment patterns
	return (line[0] == '/' && line[1] == '/') || // Go, JS, Java
		   (line[0] == '#') ||                    // Python, Ruby
		   (line[0] == '/' && line[1] == '*')     // Multi-line
}
