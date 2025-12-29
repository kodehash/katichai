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
	sb.WriteString(" 🤖 AI CODE REVIEW REPORT\n")
	sb.WriteString("════════════════════════════════════════════════════════════\n\n")

	// Token Usage
	if report.TokensUsed.TotalTokens > 0 {
		sb.WriteString(fmt.Sprintf("📊 Tokens: %s%d input%s + %s%d output%s = %s%d total%s\n\n",
			"\033[36m", report.TokensUsed.InputTokens, "\033[0m",
			"\033[36m", report.TokensUsed.OutputTokens, "\033[0m",
			"\033[1;36m", report.TokensUsed.TotalTokens, "\033[0m"))
	}

	// Summary
	if report.Summary != "" {
		sb.WriteString("📌 SUMMARY\n")
		sb.WriteString(report.Summary + "\n\n")
	}

	// Critical Issues (exclude static analysis and duplicate summaries)
	criticalIssues := []ReviewIssue{}
	staticIssues := []ReviewIssue{}
	duplicateCount := 0
	
	for _, issue := range report.Issues {
		if issue.Category == "STATIC_ANALYSIS" {
			staticIssues = append(staticIssues, issue)
		} else if strings.Contains(issue.Description, "duplicate") || strings.Contains(issue.Description, "Duplicate") {
			// Count duplicates for summary, don't show details
			duplicateCount++
		} else {
			criticalIssues = append(criticalIssues, issue)
		}
	}
	
	if len(criticalIssues) > 0 {
		sb.WriteString("⚠️  CRITICAL ISSUES\n")
		for _, issue := range criticalIssues {
			icon := "🔴"
			if issue.Severity == "WARNING" {
				icon = "🟡"
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
	
	// Show duplicate summary
	if duplicateCount > 0 {
		sb.WriteString("🔄 DUPLICATE CODE\n")
		sb.WriteString(fmt.Sprintf("  • Found %d duplicate code block%s (see HTML report for details)\n", duplicateCount, pluralize(duplicateCount)))
		sb.WriteString("\n")
	}

	// Suggestions
	if len(report.Suggestions) > 0 {
		sb.WriteString("💡 SUGGESTIONS\n")
		for _, sug := range report.Suggestions {
			sb.WriteString(fmt.Sprintf("• %s\n", sug))
		}
		sb.WriteString("\n")
	}

	// Static Analysis (summary only - show file counts per type)
	if len(staticIssues) > 0 {
		sb.WriteString("📊 STATIC ANALYSIS (informational, not affecting score)\n")
		
		// Group by subcategory and count unique files
		grouped := make(map[string]map[string]bool) // subcategory -> set of files
		for _, issue := range staticIssues {
			subcategory := issue.Subcategory
			if subcategory == "" {
				subcategory = "other"
			}
			if grouped[subcategory] == nil {
				grouped[subcategory] = make(map[string]bool)
			}
			// Use location (file path) to count unique files
			if issue.Location != "" {
				grouped[subcategory][issue.Location] = true
			}
		}
		
		// Display summary with file counts
		categoryNames := map[string]string{
			"complexity":       "High Complexity",
			"function_length":  "Long Functions",
			"naming":           "Naming Issues",
			"duplication":      "Code Duplication",
			"unused_code":      "Unused Code",
			"style_violation":  "Style Violations",
			"ai_generated":     "AI-Generated Patterns",
			"other":            "Other",
		}
		
		for subcategory, files := range grouped {
			categoryName := categoryNames[subcategory]
			if categoryName == "" {
				categoryName = subcategory
			}
			
			fileCount := len(files)
			sb.WriteString(fmt.Sprintf("  • %s: %d file%s\n", categoryName, fileCount, pluralize(fileCount)))
		}
		sb.WriteString("\n")
	}

	// AI-Generated Code Analysis
	f.formatAIAnalysis(&sb, report)

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
	// Status and Score removed - scoring is subjective
	
	// Token Usage
	if report.TokensUsed.TotalTokens > 0 {
		sb.WriteString(fmt.Sprintf("**Tokens Used**: %d input + %d output = **%d total**\n\n",
			report.TokensUsed.InputTokens,
			report.TokensUsed.OutputTokens,
			report.TokensUsed.TotalTokens))
	}
	
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

// formatAIAnalysis formats AI-generated code analysis section (summary only)
func (f *Formatter) formatAIAnalysis(sb *strings.Builder, report *ReviewReport) {
	if report.FileAnalysis == nil || len(report.FileAnalysis) == 0 {
		return
	}

	// Collect all files with AI scores (> 0%)
	aiFiles := make([]struct {
		path      string
		percentage float64
	}, 0)
	
	for filePath, analysis := range report.FileAnalysis {
		if analysis.AIScore != nil && analysis.AIScore.AIPercentage > 0 {
			aiFiles = append(aiFiles, struct {
				path      string
				percentage float64
			}{
				path:      filePath,
				percentage: analysis.AIScore.AIPercentage,
			})
		}
	}

	if len(aiFiles) == 0 {
		return
	}

	sb.WriteString("🤖 AI-GENERATED CODE ANALYSIS\n")
	
	// Show only file names and percentages (no detailed breakdown)
	for _, file := range aiFiles {
		sb.WriteString(fmt.Sprintf("  • %s (%.0f%%)\n", file.path, file.percentage))
	}
	sb.WriteString("\n")
}

// pluralize returns "s" if count != 1, otherwise ""
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
