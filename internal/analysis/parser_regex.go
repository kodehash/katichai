package analysis

import (
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/katichai/katich/internal/context"
)

// RegexParser parses source files using regex patterns
type RegexParser struct {
	language string
}

// NewRegexParser creates a new parser for a specific language
func NewRegexParser(language string) *RegexParser {
	return &RegexParser{language: language}
}

// Language config holds patterns for extraction
type langConfig struct {
	funcPattern *regexp.Regexp
	commentLine string
	commentBlockStart string
	commentBlockEnd   string
	braceStyle  bool // true for {}, false for indentation (Python)
}

var configs = map[string]langConfig{
	string(context.LanguageJava): {
		// Matches: public void methodName(...) { or similar
		funcPattern: regexp.MustCompile(`(?:public|protected|private|static|\s) +[\w\<\>\[\]]+\s+(\w+) *\([^\)]*\) *(\{?|[^;])`),
		commentLine: "//",
		commentBlockStart: "/*",
		commentBlockEnd: "*/",
		braceStyle: true,
	},
	string(context.LanguageCSharp): {
		// Matches: public override void MethodName(...)
		funcPattern: regexp.MustCompile(`(?:public|protected|private|static|override|virtual|async|\s) +[\w\<\>\[\]]+\s+(\w+) *\([^\)]*\) *(\{?|[^;])`),
		commentLine: "//",
		commentBlockStart: "/*",
		commentBlockEnd: "*/",
		braceStyle: true,
	},
	string(context.LanguageJavaScript): {
		// Matches: function name(), const name = () =>, name() { (methods)
		funcPattern: regexp.MustCompile(`function\s+(\w+)|const\s+(\w+)\s*=\s*\(|(\w+)\s*\([^)]*\)\s*\{`),
		commentLine: "//",
		commentBlockStart: "/*",
		commentBlockEnd: "*/",
		braceStyle: true,
	},
	string(context.LanguageTypeScript): {
		funcPattern: regexp.MustCompile(`function\s+(\w+)|const\s+(\w+)\s*=\s*\(|(\w+)\s*\([^)]*\)\s*\{|public\s+(\w+)\s*\(`),
		commentLine: "//",
		commentBlockStart: "/*",
		commentBlockEnd: "*/",
		braceStyle: true,
	},
	string(context.LanguagePython): {
		// Matches: def name(
		funcPattern: regexp.MustCompile(`def\s+(\w+)\s*\(`),
		commentLine: "#",
		braceStyle: false,
	},
}

// ParseFile parses a source file
func (p *RegexParser) ParseFile(filePath string) (*FileAnalysis, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	hash := sha256.Sum256(content)
	
	analysis := &FileAnalysis{
		FilePath:  filePath,
		Language:  p.language,
		Hash:      fmt.Sprintf("%x", hash),
		Functions: make([]FunctionInfo, 0),
		Classes:   make([]ClassInfo, 0),
		Imports:   make([]ImportInfo, 0),
		Issues:    make([]Issue, 0),
	}

	config, ok := configs[p.language]
	if !ok {
		// Fallback for languages we don't have explicit patterns for? 
		// Or just return basic metrics
		analysis.Metrics = CalculateBasicMetrics(string(content))
		return analysis, nil
	}

	// Simple Line-by-Line Parsing for function starts
	// This is not perfect but good enough for embeddings/metrics
	lines := strings.Split(string(content), "\n")
	
	for i, line := range lines {
		matches := config.funcPattern.FindStringSubmatch(line)
		if matches != nil {
			// Extract function name (pick the first non-empty group after full match)
			funcName := ""
			for j := 1; j < len(matches); j++ {
				if matches[j] != "" {
					funcName = matches[j]
					break
				}
			}
			
			if funcName != "" {
				fn := FunctionInfo{
					Name:      funcName,
					StartLine: i + 1,
					EndLine:   i + 1, // Will update if we can track body
				}
				
				// Try to extract body
				if config.braceStyle {
					end, body := p.extractBraceBody(lines, i)
					fn.EndLine = end + 1
					fn.Body = body
					fn.LOC = fn.EndLine - fn.StartLine + 1
				} else {
					// Python indentation style
					end, body := p.extractIndentedBody(lines, i)
					fn.EndLine = end + 1
					fn.Body = body
					fn.LOC = fn.EndLine - fn.StartLine + 1
				}
				
				// Basic Complexity Check (count branching keywords)
				fn.Complexity = p.calculateComplexity(fn.Body)
				
				analysis.Functions = append(analysis.Functions, fn)
			}
		}
	}
	
	// Copy aggregate metrics from functions
	analysis.Metrics = CalculateBasicMetrics(string(content))
	analysis.Metrics.FunctionCount = len(analysis.Functions)
	
	// Helper to update max/average
	totalLOC := 0
	totalComplexity := 0
	for _, fn := range analysis.Functions {
		totalLOC += fn.LOC
		totalComplexity += fn.Complexity
		if fn.LOC > analysis.Metrics.MaxFunctionLength {
			analysis.Metrics.MaxFunctionLength = fn.LOC
		}
		
		// Add basic issues
		if fn.Complexity > 10 {
			analysis.Issues = append(analysis.Issues, Issue{
				Type: IssueTypeComplexity, Severity: SeverityWarning, Line: fn.StartLine,
				Message: fmt.Sprintf("Function '%s' has high complexity: %d", fn.Name, fn.Complexity),
			})
		}
		if fn.LOC > 50 {
			analysis.Issues = append(analysis.Issues, Issue{
				Type: IssueTypeFunctionLength, Severity: SeverityWarning, Line: fn.StartLine,
				Message: fmt.Sprintf("Function '%s' is too long: %d lines", fn.Name, fn.LOC),
			})
		}
	}
	analysis.Metrics.CyclomaticComplexity = totalComplexity
	if len(analysis.Functions) > 0 {
		analysis.Metrics.AvgFunctionLength = float64(totalLOC) / float64(len(analysis.Functions))
	}

	return analysis, nil
}

func (p *RegexParser) extractBraceBody(lines []string, startLine int) (int, string) {
	var body strings.Builder
	braceCount := 0
	started := false
	
	endLine := startLine
	
	// Scan from start line
	for i := startLine; i < len(lines); i++ {
		line := lines[i]
		body.WriteString(line + "\n")
		
		for _, char := range line {
			if char == '{' {
				braceCount++
				started = true
			} else if char == '}' {
				braceCount--
			}
		}
		
		if started && braceCount == 0 {
			endLine = i
			break
		}
	}
	
	// Fallback if mismatched braces (e.g. inside strings/comments which we ignore for simplicity)
	// We'll trust the brace count reaches zero. If not, we might capture whole file.
	// Cap at 200 lines to prevent gigantic mistakes? No, let's just return what we have.
	if braceCount != 0 {
		// Parsing error or weird structure, just return single line
		return startLine, lines[startLine]
	}

	return endLine, body.String()
}

func (p *RegexParser) extractIndentedBody(lines []string, startLine int) (int, string) {
	var body strings.Builder
	baseIndent := -1
	endLine := startLine
	
	// Check definition line indentation
	defIndent := getIndentLevel(lines[startLine])
	
	for i := startLine + 1; i < len(lines); i++ {
		line := lines[i]
		
		// Skip empty lines, include them in body
		if strings.TrimSpace(line) == "" {
			body.WriteString(line + "\n")
			continue
		}
		
		currentIndent := getIndentLevel(line)
		
		if baseIndent == -1 {
			baseIndent = currentIndent
			// If body starts with same or less indent than definition, it's a one-liner or empty
			if baseIndent <= defIndent {
				return startLine, lines[startLine]
			}
		}
		
		if currentIndent < baseIndent {
			// Body ended
			endLine = i - 1
			break
		}
		
		body.WriteString(line + "\n")
		endLine = i
	}
	
	return endLine, body.String()
}

func getIndentLevel(line string) int {
	indent := 0
	for _, char := range line {
		if char == ' ' {
			indent++
		} else if char == '\t' {
			indent += 4
		} else {
			break
		}
	}
	return indent
}

func (p *RegexParser) calculateComplexity(body string) int {
	complexity := 1
	keywords := []string{"if ", "else", "for ", "while", "case ", "&&", "||", "catch"}
	for _, kw := range keywords {
		complexity += strings.Count(body, kw)
	}
	return complexity
}
