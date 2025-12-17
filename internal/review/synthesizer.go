package review

import (
	"regexp"
	"strings"

	"github.com/katichai/katich/internal/analysis"
)

// ReviewReport represents the final synthesized review
type ReviewReport struct {
	Summary     string        `json:"summary"`
	Score       int           `json:"score"`
	Status      string        `json:"status"` // PASS or FAIL
	Issues      []ReviewIssue `json:"issues"`
	Suggestions []string      `json:"suggestions"`
}

// ReviewIssue represents an issue found during review
type ReviewIssue struct {
	Category    string `json:"category"` // SECURITY, ARCHITECTURE, PERFORMANCE, CODE_QUALITY
	Severity    string `json:"severity"` // CRITICAL, WARNING, INFO
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
func (s *Synthesizer) Synthesize(llmOutput string, staticIssues []analysis.Issue, duplicates []string) *ReviewReport {
	report := &ReviewReport{
		Status: "PASS",
		Score:  100,
	}

	// 1. Parse LLM Output
	s.parseLLMOutput(llmOutput, report)

	// 2. Incorporate Static Analysis
	for _, issue := range staticIssues {
		// Convert analysis.Issue to ReviewIssue
		// We might want to deduplicate if LLM found the same thing, but simple append is safer now
		rIssue := ReviewIssue{
			Category:    "STATIC_ANALYSIS",
			Severity:    strings.ToUpper(string(issue.Severity)),
			Description: issue.Message,
			Location:    issue.File,
		}
		report.Issues = append(report.Issues, rIssue)
		
		// Penalty for static issues (Low priority)
		if issue.Severity == "error" || issue.Severity == "critical" {
			report.Score -= 1
		} 
		// Warnings have 0 penalty now

	}

	// 3. Incorporate Duplicates
	for _, dup := range duplicates {
		report.Issues = append(report.Issues, ReviewIssue{
			Category:    "CODE_QUALITY",
			Severity:    "WARNING",
			Description: dup,
		})
		report.Score -= 5
	}

	// 4. Final adjustments
	if report.Score < 0 {
		report.Score = 0
	}
	if report.Score < 70 {
		report.Status = "FAIL"
	}

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

				report.Issues = append(report.Issues, ReviewIssue{
					Category:    category,
					Severity:    "CRITICAL", // Assuming critical section
					Description: desc,
				})
				
				// Apply category-based penalties
				switch strings.ToUpper(category) {
				case "SECURITY":
					report.Score -= 20 // Security issues are critical
				case "ARCHITECTURE":
					report.Score -= 5 // Architectural issues
				case "PERFORMANCE":
					report.Score -= 10 // Performance issues
				default:
					report.Score -= 10 // Other critical issues
				}
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
