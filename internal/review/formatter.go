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

	// AI-Generated Files Count (at the top)
	aiFileCount := f.countAIFiles(report)
	if report.FileAnalysis != nil && len(report.FileAnalysis) > 0 {
		sb.WriteString(fmt.Sprintf("🤖 AI-Generated Code: %d file(s) analyzed\n\n", aiFileCount))
	}

	// Summary
	if report.Summary != "" {
		sb.WriteString("📌 SUMMARY\n")
		sb.WriteString(report.Summary + "\n\n")
	}

	// Separate static analysis from other issues
	criticalIssues := make([]ReviewIssue, 0)
	staticIssues := make([]ReviewIssue, 0)
	
	for _, issue := range report.Issues {
		if issue.Category == "STATIC_ANALYSIS" {
			staticIssues = append(staticIssues, issue)
		} else {
			criticalIssues = append(criticalIssues, issue)
		}
	}

	// Critical Issues (LLM findings)
	if len(criticalIssues) > 0 {
		sb.WriteString("⚠️  ISSUES FOUND\n")
		for _, issue := range criticalIssues {
			icon := "🔴"
			if issue.Severity == "WARNING" {
				icon = "🟡"
			} else if issue.Severity == "INFO" {
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

	// Static Analysis (grouped by type)
	if len(staticIssues) > 0 {
		sb.WriteString("📊 STATIC ANALYSIS (informational, not affecting score)\n")
		
		// Group by subcategory
		grouped := make(map[string][]ReviewIssue)
		for _, issue := range staticIssues {
			subcategory := issue.Subcategory
			if subcategory == "" {
				subcategory = "other"
			}
			grouped[subcategory] = append(grouped[subcategory], issue)
		}
		
		// Display groups
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
		
		for subcategory, issues := range grouped {
			categoryName := categoryNames[subcategory]
			if categoryName == "" {
				categoryName = subcategory
			}
			
			sb.WriteString(fmt.Sprintf("\n  %s (%d)\n", categoryName, len(issues)))
			for _, issue := range issues {
				sb.WriteString(fmt.Sprintf("    • %s", issue.Description))
				if issue.Location != "" {
					sb.WriteString(fmt.Sprintf(" - %s", issue.Location))
				}
				sb.WriteString("\n")
			}
		}
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

	// AI-Generated Code Analysis
	f.formatAIAnalysis(&sb, report)

	return sb.String()
}

// countAIFiles counts files with AI-generated code (> 0%)
func (f *Formatter) countAIFiles(report *ReviewReport) int {
	if report.FileAnalysis == nil {
		return 0
	}
	
	count := 0
	for _, analysis := range report.FileAnalysis {
		if analysis.AIScore != nil && analysis.AIScore.AIPercentage > 0 {
			count++
		}
	}
	return count
}

// formatAIAnalysis formats AI-generated code analysis section
func (f *Formatter) formatAIAnalysis(sb *strings.Builder, report *ReviewReport) {
	if report.FileAnalysis == nil || len(report.FileAnalysis) == 0 {
		return
	}

	// Collect all files with AI scores (including 0%)
	aiFiles := make(map[string]interface{})
	for filePath, analysis := range report.FileAnalysis {
		if analysis.AIScore != nil {
			aiFiles[filePath] = analysis
		}
	}

	if len(aiFiles) == 0 {
		return
	}

	sb.WriteString("🤖 AI-GENERATED CODE ANALYSIS\n\n")
	
	// Count files with actual AI-generated code (> 0%)
	aiGeneratedCount := 0
	for filePath, _ := range aiFiles {
		analysis := report.FileAnalysis[filePath]
		if analysis.AIScore != nil && analysis.AIScore.AIPercentage > 0 {
			aiGeneratedCount++
		}
	}
	
	// Show summary
	if aiGeneratedCount > 0 {
		sb.WriteString(fmt.Sprintf("Files with likely AI-generated code: %d\n", aiGeneratedCount))
		for filePath, _ := range aiFiles {
			analysis := report.FileAnalysis[filePath]
			aiScore := analysis.AIScore
			if aiScore.AIPercentage > 0 {
				sb.WriteString(fmt.Sprintf("  • %s (%.0f%%)\n", filePath, aiScore.AIPercentage))
			}
		}
	} else {
		sb.WriteString("No AI-generated code patterns detected (all files scored 0%)\n")
	}
	sb.WriteString("\n")

	// Then show detailed analysis (only for files with AI% > 0)
	for filePath, _ := range aiFiles {
		analysis := report.FileAnalysis[filePath]
		aiScore := analysis.AIScore
		
		// Skip files with 0% in detailed view
		if aiScore.AIPercentage == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("  📄 %s\n", filePath))
		sb.WriteString(fmt.Sprintf("     AI-Generated: %.1f%% (%d/%d lines)\n",
			aiScore.AIPercentage,
			aiScore.AIGeneratedLOC,
			aiScore.TotalLOC))
		sb.WriteString(fmt.Sprintf("     Confidence: %.0f%%\n",
			aiScore.OverallConfidence*100))

		if len(aiScore.FunctionScores) > 0 {
			sb.WriteString("     Functions:\n")
			for _, fnScore := range aiScore.FunctionScores {
				// Show top 2 indicators
				indicatorCount := len(fnScore.Indicators)
				if indicatorCount > 2 {
					indicatorCount = 2
				}
				indicators := strings.Join(fnScore.Indicators[:indicatorCount], ", ")

				sb.WriteString(fmt.Sprintf("       • %s (%.0f%% confidence) - %s",
					fnScore.FunctionName,
					fnScore.AIConfidence*100,
					indicators))

				if len(fnScore.Indicators) > 2 {
					sb.WriteString(fmt.Sprintf(" +%d more", len(fnScore.Indicators)-2))
				}
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	}
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

	// Separate static analysis from other issues
	criticalIssues := make([]ReviewIssue, 0)
	staticIssues := make([]ReviewIssue, 0)
	
	for _, issue := range report.Issues {
		if issue.Category == "STATIC_ANALYSIS" {
			staticIssues = append(staticIssues, issue)
		} else {
			criticalIssues = append(criticalIssues, issue)
		}
	}

	if len(criticalIssues) > 0 {
		sb.WriteString("## Issues\n")
		for _, issue := range criticalIssues {
			sb.WriteString(fmt.Sprintf("- **[%s]** %s: %s\n", issue.Severity, issue.Category, issue.Description))
			if issue.Location != "" {
				sb.WriteString(fmt.Sprintf("  - Location: `%s`\n", issue.Location))
			}
		}
		sb.WriteString("\n")
	}

	if len(report.Suggestions) > 0 {
		sb.WriteString("## Suggestions\n")
		for _, sug := range report.Suggestions {
			sb.WriteString(fmt.Sprintf("- %s\n", sug))
		}
		sb.WriteString("\n")
	}

	if len(staticIssues) > 0 {
		sb.WriteString("## Static Analysis\n")
		sb.WriteString("*Informational findings that do not affect the review score*\n\n")
		
		// Group by subcategory
		grouped := make(map[string][]ReviewIssue)
		for _, issue := range staticIssues {
			subcategory := issue.Subcategory
			if subcategory == "" {
				subcategory = "other"
			}
			grouped[subcategory] = append(grouped[subcategory], issue)
		}
		
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
		
		for subcategory, issues := range grouped {
			categoryName := categoryNames[subcategory]
			if categoryName == "" {
				categoryName = subcategory
			}
			
			sb.WriteString(fmt.Sprintf("### %s (%d)\n", categoryName, len(issues)))
			for _, issue := range issues {
				sb.WriteString(fmt.Sprintf("- %s", issue.Description))
				if issue.Location != "" {
					sb.WriteString(fmt.Sprintf(" - `%s`", issue.Location))
				}
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	// AI-Generated Code Analysis
	f.formatAIAnalysisMarkdown(&sb, report)

	return sb.String()
}

// formatAIAnalysisMarkdown formats AI analysis for markdown
func (f *Formatter) formatAIAnalysisMarkdown(sb *strings.Builder, report *ReviewReport) {
	if report.FileAnalysis == nil || len(report.FileAnalysis) == 0 {
		return
	}

	// Collect files with AI-generated code
	aiFiles := make(map[string]interface{})
	for filePath, analysis := range report.FileAnalysis {
		if analysis.AIScore != nil && analysis.AIScore.AIPercentage > 0 {
			aiFiles[filePath] = analysis
		}
	}

	if len(aiFiles) == 0 {
		return
	}

	sb.WriteString("## AI-Generated Code Analysis\n\n")
	sb.WriteString("*These files contain code patterns commonly found in AI-generated code. Review carefully for logic errors, edge cases, and domain-specific requirements.*\n\n")

	for filePath, _ := range aiFiles {
		analysis := report.FileAnalysis[filePath]
		aiScore := analysis.AIScore

		sb.WriteString(fmt.Sprintf("### `%s`\n\n", filePath))
		sb.WriteString(fmt.Sprintf("- **AI-Generated**: %.1f%% (%d/%d lines)\n",
			aiScore.AIPercentage,
			aiScore.AIGeneratedLOC,
			aiScore.TotalLOC))
		sb.WriteString(fmt.Sprintf("- **Confidence**: %.0f%%\n\n",
			aiScore.OverallConfidence*100))

		if len(aiScore.FunctionScores) > 0 {
			sb.WriteString("**Functions:**\n\n")
			for _, fnScore := range aiScore.FunctionScores {
				sb.WriteString(fmt.Sprintf("- `%s` (%.0f%% confidence)\n",
					fnScore.FunctionName,
					fnScore.AIConfidence*100))
				sb.WriteString(fmt.Sprintf("  - Indicators: %s\n",
					strings.Join(fnScore.Indicators, ", ")))
			}
			sb.WriteString("\n")
		}
	}
}
