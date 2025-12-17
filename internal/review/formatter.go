package review

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Formatter handles report formatting
type Formatter struct{}

// NewFormatter creates a new formatter
func NewFormatter() *Formatter {
	return &Formatter{}
}

// FormatText creates a CLI-friendly text report
func (f *Formatter) FormatText(report *ReviewReport) string {
	var sb strings.Builder

	// Header
	sb.WriteString("\n════════════════════════════════════════════════════════════\n")
	sb.WriteString(fmt.Sprintf(" 🤖 AI CODE REVIEW REPORT   (Score: %d/100)   [%s]\n", report.Score, report.Status))
	sb.WriteString("════════════════════════════════════════════════════════════\n\n")

	// Summary
	if report.Summary != "" {
		sb.WriteString("📌 SUMMARY\n")
		sb.WriteString(report.Summary + "\n\n")
	}

	// Issues
	if len(report.Issues) > 0 {
		sb.WriteString("⚠️  ISSUES FOUND\n")
		for _, issue := range report.Issues {
			icon := "🔴"
			if issue.Severity == "WARNING" {
				icon = "🟡"
			} else if issue.Severity == "INFO" || issue.Category == "STATIC_ANALYSIS" {
				icon = "🔵"
			}

			sb.WriteString(fmt.Sprintf("%s [%s] %s\n", icon, issue.Category, issue.Description))
			if issue.Location != "" {
				sb.WriteString(fmt.Sprintf("   📍 %s\n", issue.Location))
			}
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("✅ No critical issues found.\n\n")
	}

	// Suggestions
	if len(report.Suggestions) > 0 {
		sb.WriteString("💡 SUGGESTIONS\n")
		for _, sug := range report.Suggestions {
			sb.WriteString(fmt.Sprintf("• %s\n", sug))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatJSON creates a JSON report
func (f *Formatter) FormatJSON(report *ReviewReport) string {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// FormatMarkdown creates a Markdown report (e.g., for PR comments)
func (f *Formatter) FormatMarkdown(report *ReviewReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Code Review Report\n\n"))
	sb.WriteString(fmt.Sprintf("**Status**: %s | **Score**: %d/100\n\n", report.Status, report.Score))
	
	sb.WriteString("## Summary\n")
	sb.WriteString(report.Summary + "\n\n")

	if len(report.Issues) > 0 {
		sb.WriteString("## Issues\n")
		for _, issue := range report.Issues {
			sb.WriteString(fmt.Sprintf("- **[%s]** %s: %s\n", issue.Severity, issue.Category, issue.Description))
		}
		sb.WriteString("\n")
	}

	if len(report.Suggestions) > 0 {
		sb.WriteString("## Suggestions\n")
		for _, sug := range report.Suggestions {
			sb.WriteString(fmt.Sprintf("- %s\n", sug))
		}
	}

	return sb.String()
}
