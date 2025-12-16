package analysis

import (
	"strings"
)

// AICodeDetector detects AI-generated code patterns
type AICodeDetector struct {
	language string // Language for context-specific detection
}

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
	d.language = analysis.Language

	// Check for AI-generated patterns
	for _, fn := range analysis.Functions {
		confidence, indicators := d.CalculateConfidence(fn, analysis.Language)

		if confidence > 0.5 {
			patterns = append(patterns, AICodePattern{
				File:       analysis.FilePath,
				StartLine:  fn.StartLine,
				EndLine:    fn.EndLine,
				Pattern:    "Potentially AI-generated code",
				Confidence: confidence,
				Indicators: indicators,
			})
		}
	}

	return patterns
}

// CalculateConfidence calculates comprehensive AI confidence score for a function
func (d *AICodeDetector) CalculateConfidence(fn FunctionInfo, language string) (float64, []string) {
	confidence := 0.0
	indicators := []string{}

	// 1. Comment patterns (30% weight)
	if hasExcessiveComments(fn.Body, fn.LOC) {
		confidence += 0.15
		indicators = append(indicators, "Excessive inline comments")
	}

	if hasGenericComments(fn.Comments) {
		confidence += 0.10
		indicators = append(indicators, "Generic placeholder comments")
	}

	if hasVerboseDocstring(fn.Name, fn.Comments) {
		confidence += 0.05
		indicators = append(indicators, "Verbose docstring")
	}

	// 2. Naming patterns (20% weight)
	if hasGenericPrefix(fn.Name) {
		confidence += 0.10
		indicators = append(indicators, "Generic function name prefix")
	}

	if d.isGenericName(fn.Name) {
		confidence += 0.05
		indicators = append(indicators, "Generic function name")
	}

	if isOverlyDescriptive(fn.Name) {
		confidence += 0.05
		indicators = append(indicators, "Overly descriptive name")
	}

	// 3. Code structure patterns (30% weight)
	if hasExcessiveNullChecks(fn.Body) {
		confidence += 0.10
		indicators = append(indicators, "Excessive defensive null checks")
	}

	if hasExcessiveTryCatch(fn.Body) {
		confidence += 0.10
		indicators = append(indicators, "Excessive try-catch blocks")
	}

	if hasGenericExceptions(fn.Body) {
		confidence += 0.05
		indicators = append(indicators, "Generic exception messages")
	}

	if d.detectRepeatedBlocks(fn) {
		confidence += 0.05
		indicators = append(indicators, "Repeated code blocks")
	}

	// 4. Language-specific patterns (up to 40% weight - 0.05-0.08 per pattern)
	if detected, reasons := d.detectLanguageSpecificPatterns(fn, language); detected {
		// Give weight per detected pattern (max 5 patterns count)
		patternCount := len(reasons)
		if patternCount > 5 {
			patternCount = 5
		}
		confidence += float64(patternCount) * 0.08
		indicators = append(indicators, reasons...)
	}

	// 5. Complexity and size indicators (bonus, can exceed 1.0)
	if fn.Complexity > 20 {
		confidence += 0.10
		indicators = append(indicators, "Very high complexity")
	}

	if fn.LOC > 100 {
		confidence += 0.10
		indicators = append(indicators, "Excessively long function")
	}

	if len(fn.Parameters) > 5 {
		confidence += 0.05
		indicators = append(indicators, "Too many parameters")
	}

	// Cap at 1.0
	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence, indicators
}

// detectLanguageSpecificPatterns detects language-specific AI patterns
func (d *AICodeDetector) detectLanguageSpecificPatterns(fn FunctionInfo, language string) (bool, []string) {
	switch strings.ToLower(language) {
	case "java":
		return detectJavaPatterns(fn)
	case "javascript", "typescript", "js", "ts":
		return detectJSPatterns(fn)
	case "go", "golang":
		return detectGoPatterns(fn)
	case "python", "py":
		return detectPythonPatterns(fn)
	case "c#", "csharp", "cs":
		return detectCSharpPatterns(fn)
	default:
		return false, nil
	}
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

// detectRepeatedBlocks checks for repeated blocks of code within the function
func (d *AICodeDetector) detectRepeatedBlocks(fn FunctionInfo) bool {
	if fn.Body == "" {
		return false
	}

	lines := strings.Split(fn.Body, "\n")
	if len(lines) < 10 {
		return false
	}

	// Clean lines (trim whitespace)
	cleanLines := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && !strings.HasPrefix(trimmed, "//") && !strings.HasPrefix(trimmed, "/*") {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	if len(cleanLines) < 6 {
		return false
	}

	// Look for repeated blocks of 3 lines
	blockSize := 3
	for i := 0; i <= len(cleanLines)-blockSize; i++ {
		block := strings.Join(cleanLines[i:i+blockSize], "\n")
		
		// Search for this block in the rest of the function
		for j := i + blockSize; j <= len(cleanLines)-blockSize; j++ {
			compareBlock := strings.Join(cleanLines[j:j+blockSize], "\n")
			if block == compareBlock {
				return true
			}
		}
	}

	return false
}
