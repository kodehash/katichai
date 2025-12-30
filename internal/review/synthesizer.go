package review

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/katichai/katich/internal/analysis"
)

// ReviewReport represents the final synthesized review
type ReviewReport struct {
	Summary         string                            `json:"summary"`
	Score           int                               `json:"score"`
	Status          string                            `json:"status"` // PASS or FAIL
	Issues          []ReviewIssue                     `json:"issues"`
	Suggestions     []string                          `json:"suggestions"`
	FileAnalysis    map[string]*analysis.FileAnalysis `json:"file_analysis,omitempty"`
	TokensUsed      TokenUsage                        `json:"tokens_used"`
	DuplicateBlocks []DuplicateBlockInfo              `json:"duplicate_blocks,omitempty"`
	AIPatterns      map[string][]analysis.AICodePattern `json:"ai_patterns,omitempty"`
	SamplingInfo    *SamplingInfo                     `json:"sampling_info,omitempty"`
}

// SamplingInfo contains information about which files were reviewed and which were ignored
type SamplingInfo struct {
	TotalFiles      int                      `json:"total_files"`
	ReviewedFiles   []string                 `json:"reviewed_files"`
	IgnoredFiles    []IgnoredFile            `json:"ignored_files"`
	FilteredReasons map[string]int           `json:"filtered_reasons"`
}

// IgnoredFile represents a file that was ignored during sampling
type IgnoredFile struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// DuplicateBlockInfo represents structured duplicate code information
type DuplicateBlockInfo struct {
	OriginalFile  string  `json:"original_file"`
	OriginalStart int     `json:"original_start"`
	OriginalEnd   int     `json:"original_end"`
	DuplicateFile string  `json:"duplicate_file"`
	DuplicateStart int    `json:"duplicate_start"`
	DuplicateEnd   int    `json:"duplicate_end"`
	Similarity    float64 `json:"similarity"`
	Lines         int     `json:"lines"`
}

// TokenUsage represents token consumption for the review
type TokenUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// ReviewIssue represents an issue found during review
type ReviewIssue struct {
	Category    string `json:"category"`    // SECURITY, ARCHITECTURE, PERFORMANCE, CODE_QUALITY, STATIC_ANALYSIS
	Subcategory string `json:"subcategory"` // For STATIC_ANALYSIS: complexity, function_length, naming, etc.
	Severity    string `json:"severity"`    // CRITICAL, WARNING, INFO
	Description string `json:"description"`
	Location    string `json:"location,omitempty"`
}

// Synthesizer aggregates review signals
type Synthesizer struct{}

// NewSynthesizer creates a new synthesizer
func NewSynthesizer() *Synthesizer {
	return &Synthesizer{}
}

// Synthesize combines LLM output with static analysis
func (s *Synthesizer) Synthesize(llmOutput string, staticIssues []analysis.Issue, duplicates []string, fileAnalysis map[string]*analysis.FileAnalysis, tokensUsed TokenUsage) *ReviewReport {
	report := &ReviewReport{
		// Status:       "PASS",  // Commented out - scoring is subjective
		// Score:        100,      // Commented out - scoring is subjective
		FileAnalysis: fileAnalysis,
		TokensUsed:   tokensUsed,
	}

	// 1. Parse LLM Output
	s.parseLLMOutput(llmOutput, report)

	// 1.5. Deduplicate security issues
	s.deduplicateSecurityIssues(report)

	// 2. Incorporate Static Analysis
	for _, issue := range staticIssues {
		// Convert analysis.Issue to ReviewIssue
		// We might want to deduplicate if LLM found the same thing, but simple append is safer now
		rIssue := ReviewIssue{
			Category:    "STATIC_ANALYSIS",
			Subcategory: string(issue.Type), // complexity, function_length, naming, etc.
			Severity:    strings.ToUpper(string(issue.Severity)),
			Description: issue.Message,
			Location:    issue.File,
		}
		report.Issues = append(report.Issues, rIssue)
		
		// Static analysis issues are informational only - no score penalty

	}

	// 3. Incorporate Duplicates
	for _, dup := range duplicates {
		report.Issues = append(report.Issues, ReviewIssue{
			Category:    "CODE_QUALITY",
			Severity:    "WARNING",
			Description: dup,
		})
		// report.Score -= 5  // Commented out - scoring is subjective
	}

	// 4. Final adjustments (commented out - scoring is subjective)
	// if report.Score < 0 {
	// 	report.Score = 0
	// }
	// if report.Score < 70 {
	// 	report.Status = "FAIL"
	// }

	return report
}

func (s *Synthesizer) parseLLMOutput(output string, report *ReviewReport) {
	// Simple regex parsing for the Markdown sections 
	// This assumes the prompt instructions were followed

	// Extract Summary
	summaryRe := regexp.MustCompile(`(?s)## Summary\s+(.*?)\s+##`)
	if match := summaryRe.FindStringSubmatch(output); len(match) > 1 {
		report.Summary = strings.TrimSpace(match[1])
	}

	// Extract Critical Issues
	issuesRe := regexp.MustCompile(`(?s)## Critical Issues.*?\n(.*?)(##|$)`)
	if match := issuesRe.FindStringSubmatch(output); len(match) > 1 {
		lines := strings.Split(match[1], "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "-") {
				// Parse issue line: "- [CATEGORY] Description"
				trimmed := strings.TrimPrefix(line, "- ")
				category := "GENERAL"
				desc := trimmed

				if string(trimmed[0]) == "[" {
					endIdx := strings.Index(trimmed, "]")
					if endIdx > 0 {
						category = trimmed[1:endIdx]
						desc = strings.TrimSpace(trimmed[endIdx+1:])
					}
				}

				// Skip if description is "None" or empty (LLM indicating no issues)
				descLower := strings.ToLower(desc)
				if desc == "" || descLower == "none" || descLower == "none." || 
				   strings.HasPrefix(descLower, "none.") || strings.HasPrefix(descLower, "no issues") {
					continue
				}

				// Extract location from description if present (format: "file.go:123" or "in file.go:123")
				location := s.extractLocation(desc)
				
				report.Issues = append(report.Issues, ReviewIssue{
					Category:    category,
					Severity:    "CRITICAL", // Assuming critical section
					Description: desc,
					Location:    location,
				})
				
				// Apply category-based penalties (commented out - scoring is subjective)
				// switch strings.ToUpper(category) {
				// case "SECURITY":
				// 	report.Score -= 20 // Security issues are critical
				// case "ARCHITECTURE":
				// 	report.Score -= 5 // Architectural issues
				// case "PERFORMANCE":
				// 	report.Score -= 10 // Performance issues
				// default:
				// 	report.Score -= 10 // Other critical issues
				// }
			}
		}
	}

	// Extract Suggestions
	suggestionsRe := regexp.MustCompile(`(?s)## Suggestions.*?\n(.*?)(##|$)`)
	if match := suggestionsRe.FindStringSubmatch(output); len(match) > 1 {
		lines := strings.Split(match[1], "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "-") {
				report.Suggestions = append(report.Suggestions, strings.TrimPrefix(line, "- "))
			}
		}
	}
	
	// Extract Score Override (if LLM provided one)
	// For now, we calculate our own based on findings to be deterministic
}

// extractLocation extracts file path and line number from issue description
func (s *Synthesizer) extractLocation(description string) string {
	// Look for patterns like:
	// - "file.go:123"
	// - "in file.go:123"
	// - "at file.go:123"
	// - "file.go, line 123"
	// - "(file.go:123)"
	
	// Pattern 1: file.go:123 or file.go:123:456
	re1 := regexp.MustCompile(`([a-zA-Z0-9_\-./\\]+\.(go|java|js|ts|py|rs|cpp|c|h|hpp)):(\d+)`)
	if match := re1.FindStringSubmatch(description); len(match) > 0 {
		return fmt.Sprintf("%s:%s", match[1], match[3])
	}
	
	// Pattern 2: "in file.go, line 123" or "at file.go, line 123"
	re2 := regexp.MustCompile(`(?:in|at)\s+([a-zA-Z0-9_\-./\\]+\.(go|java|js|ts|py|rs|cpp|c|h|hpp)),\s*line\s+(\d+)`)
	if match := re2.FindStringSubmatch(description); len(match) > 0 {
		return fmt.Sprintf("%s:%s", match[1], match[3])
	}
	
	// Pattern 3: "(file.go:123)"
	re3 := regexp.MustCompile(`\(([a-zA-Z0-9_\-./\\]+\.(go|java|js|ts|py|rs|cpp|c|h|hpp)):(\d+)\)`)
	if match := re3.FindStringSubmatch(description); len(match) > 0 {
		return fmt.Sprintf("%s:%s", match[1], match[3])
	}
	
	return ""
}

// deduplicateSecurityIssues removes duplicate security findings
func (s *Synthesizer) deduplicateSecurityIssues(report *ReviewReport) {
	// Separate security issues from others
	securityIssues := make([]ReviewIssue, 0)
	otherIssues := make([]ReviewIssue, 0)
	
	for _, issue := range report.Issues {
		if strings.ToUpper(issue.Category) == "SECURITY" {
			securityIssues = append(securityIssues, issue)
		} else {
			otherIssues = append(otherIssues, issue)
		}
	}
	
	// Deduplicate security issues
	deduplicated := s.deduplicateIssues(securityIssues)
	
	// Rebuild issues list
	report.Issues = append(otherIssues, deduplicated...)
}

// deduplicateIssues removes duplicate issues based on similarity
func (s *Synthesizer) deduplicateIssues(issues []ReviewIssue) []ReviewIssue {
	if len(issues) == 0 {
		return issues
	}
	
	// Create a map to track unique issues
	// Key: normalized issue signature (category + location + normalized description)
	seen := make(map[string]bool)
	deduplicated := make([]ReviewIssue, 0)
	
	for _, issue := range issues {
		// Create a signature for this issue
		signature := s.createIssueSignature(issue)
		
		// Check if we've seen a similar issue
		if seen[signature] {
			continue // Skip duplicate
		}
		
		// Check for semantic similarity with existing issues
		isDuplicate := false
		for _, existingIssue := range deduplicated {
			if s.areIssuesSimilar(issue, existingIssue) {
				isDuplicate = true
				break
			}
		}
		
		if !isDuplicate {
			seen[signature] = true
			deduplicated = append(deduplicated, issue)
		}
	}
	
	return deduplicated
}

// createIssueSignature creates a unique signature for an issue
func (s *Synthesizer) createIssueSignature(issue ReviewIssue) string {
	// Normalize description (lowercase, remove extra spaces)
	normalizedDesc := strings.ToLower(issue.Description)
	normalizedDesc = regexp.MustCompile(`\s+`).ReplaceAllString(normalizedDesc, " ")
	normalizedDesc = strings.TrimSpace(normalizedDesc)
	
	// Extract key terms (first 50 chars or first sentence)
	keyTerms := normalizedDesc
	if len(keyTerms) > 50 {
		// Try to get first sentence
		if idx := strings.Index(keyTerms, "."); idx > 0 && idx < 50 {
			keyTerms = keyTerms[:idx]
		} else {
			keyTerms = keyTerms[:50]
		}
	}
	
	// Combine category, location, and key terms
	return fmt.Sprintf("%s|%s|%s", strings.ToUpper(issue.Category), issue.Location, keyTerms)
}

// areIssuesSimilar checks if two issues are semantically similar (duplicates)
func (s *Synthesizer) areIssuesSimilar(issue1, issue2 ReviewIssue) bool {
	// Same category required
	if strings.ToUpper(issue1.Category) != strings.ToUpper(issue2.Category) {
		return false
	}
	
	// Normalize descriptions
	normalized1 := strings.ToLower(issue1.Description)
	normalized1 = regexp.MustCompile(`\s+`).ReplaceAllString(normalized1, " ")
	normalized1 = strings.TrimSpace(normalized1)
	
	normalized2 := strings.ToLower(issue2.Description)
	normalized2 = regexp.MustCompile(`\s+`).ReplaceAllString(normalized2, " ")
	normalized2 = strings.TrimSpace(normalized2)
	
	// If same location, check for high similarity
	if issue1.Location != "" && issue2.Location != "" && issue1.Location == issue2.Location {
		// Same location - check if descriptions are very similar
		return s.calculateSimilarity(normalized1, normalized2) > 0.7
	}
	
	// If different locations but very similar descriptions, might be duplicate
	// Only for security issues - be more strict
	if strings.ToUpper(issue1.Category) == "SECURITY" {
		similarity := s.calculateSimilarity(normalized1, normalized2)
		// For security issues, require 85% similarity to consider duplicate
		return similarity > 0.85
	}
	
	return false
}

// calculateSimilarity calculates word-based similarity between two strings
func (s *Synthesizer) calculateSimilarity(text1, text2 string) float64 {
	words1 := strings.Fields(text1)
	words2 := strings.Fields(text2)
	
	if len(words1) == 0 && len(words2) == 0 {
		return 1.0
	}
	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}
	
	// Create word sets (normalize to lowercase, remove very short words)
	set1 := make(map[string]int)
	set2 := make(map[string]int)
	
	for _, w := range words1 {
		w = strings.ToLower(strings.Trim(w, ".,!?;:()[]{}"))
		if len(w) > 2 {
			set1[w]++
		}
	}
	
	for _, w := range words2 {
		w = strings.ToLower(strings.Trim(w, ".,!?;:()[]{}"))
		if len(w) > 2 {
			set2[w]++
		}
	}
	
	// Calculate Jaccard similarity (intersection over union)
	intersection := 0
	union := len(set1) + len(set2)
	
	for word := range set1 {
		if set2[word] > 0 {
			intersection++
			union-- // Remove from union count since it's in both
		}
	}
	
	if union == 0 {
		return 1.0
	}
	
	return float64(intersection) / float64(union)
}
