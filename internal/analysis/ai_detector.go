package analysis

import (
	"strings"
)

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

		// Check for repeated code blocks
		if d.detectRepeatedBlocks(fn) {
			indicators = append(indicators, "Contains repeated code blocks")
			confidence += 0.4
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

// CalculateConfidence calculates AI-generation confidence for a function
func (d *AICodeDetector) CalculateConfidence(fn FunctionInfo, language string) (float64, []string) {
	indicators := []string{}
	baseConfidence := 0.0
	
	// Use language-specific pattern detection
	switch language {
	case "Java":
		detected, langIndicators := detectJavaPatterns(fn)
		if detected {
			baseConfidence += 0.4
			indicators = append(indicators, langIndicators...)
		}
	case "Python":
		detected, langIndicators := detectPythonPatterns(fn)
		if detected {
			baseConfidence += 0.4
			indicators = append(indicators, langIndicators...)
		}
	case "Go":
		detected, langIndicators := detectGoPatterns(fn)
		if detected {
			baseConfidence += 0.4
			indicators = append(indicators, langIndicators...)
		}
	case "JavaScript", "TypeScript":
		detected, langIndicators := detectJSPatterns(fn)
		if detected {
			baseConfidence += 0.4
			indicators = append(indicators, langIndicators...)
		}
	case "C#":
		detected, langIndicators := detectCSharpPatterns(fn)
		if detected {
			baseConfidence += 0.4
			indicators = append(indicators, langIndicators...)
		}
	}
	
	// Add generic pattern checks
	if d.isGenericName(fn.Name) {
		indicators = append(indicators, "Generic function name")
		baseConfidence += 0.1
	}
	
	if fn.LOC > 100 {
		indicators = append(indicators, "Excessively long function")
		baseConfidence += 0.2
	}
	
	if fn.Complexity > 20 {
		indicators = append(indicators, "Very high complexity")
		baseConfidence += 0.2
	}
	
	if d.detectRepeatedBlocks(fn) {
		indicators = append(indicators, "Contains repeated code blocks")
		baseConfidence += 0.3
	}
	
	// Cap confidence at 1.0
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}
	
	return baseConfidence, indicators
}
