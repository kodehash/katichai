package review

import (
	"fmt"
	"strings"
	"time"
)

// FormatGFM generates a GitHub Flavored Markdown report
func (f *Formatter) FormatGFM(report *ReviewReport, fileContents map[string]string, diffInfo *DiffInfo) string {
	var sb strings.Builder

	// Header Section
	sb.WriteString("# Katich AI Code Review Report\n\n")
	
	if diffInfo != nil && diffInfo.ProjectName != "" {
		sb.WriteString(fmt.Sprintf("**Project:** %s\n\n", diffInfo.ProjectName))
	}

	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	if diffInfo != nil {
		if diffInfo.IsFullRepository {
			sb.WriteString("**Review Type:** Full Repository Review\n\n")
		} else if diffInfo.Range != "" {
			sb.WriteString(fmt.Sprintf("**Range:** %s", diffInfo.Range))
			if diffInfo.FromCommit != "" && diffInfo.ToCommit != "" {
				sb.WriteString(fmt.Sprintf(" (%s..%s)", diffInfo.FromCommit, diffInfo.ToCommit))
			}
			sb.WriteString("\n\n")
		} else if diffInfo.ToCommit != "" {
			sb.WriteString("**Commits:** ")
			if diffInfo.FromCommit != "" {
				sb.WriteString(fmt.Sprintf("%s..%s", diffInfo.FromCommit, diffInfo.ToCommit))
			} else {
				sb.WriteString(diffInfo.ToCommit)
			}
			sb.WriteString("\n\n")
		}
	}

	// Similarity Check Status
	if report.SimilarityCheckLimited {
		sb.WriteString("⚠️ **Similarity Check:** ")
		if strings.Contains(report.SimilarityCheckReason, "no context") {
			sb.WriteString(fmt.Sprintf("Disabled (%s)\n\n", report.SimilarityCheckReason))
		} else {
			sb.WriteString(fmt.Sprintf("%s\n\n", report.SimilarityCheckReason))
		}
	}

	// File Sampling
	if report.SamplingInfo != nil && report.SamplingInfo.TotalFiles > 0 {
		sb.WriteString("## 📁 File Sampling\n\n")
		sb.WriteString(fmt.Sprintf("*%d reviewed, %d ignored (%d total)*\n\n", 
			report.SamplingInfo.ReviewedCount, 
			report.SamplingInfo.IgnoredCount,
			report.SamplingInfo.TotalFiles))

		// Create table for reviewed and ignored files
		maxRows := len(report.SamplingInfo.ReviewedFiles)
		if len(report.SamplingInfo.IgnoredFiles) > maxRows {
			maxRows = len(report.SamplingInfo.IgnoredFiles)
		}

		if maxRows > 0 {
			headers := []string{"Reviewed Files", "Ignored Files"}
			rows := make([][]string, maxRows)
			for i := 0; i < maxRows; i++ {
				row := make([]string, 2)
				if i < len(report.SamplingInfo.ReviewedFiles) {
					row[0] = report.SamplingInfo.ReviewedFiles[i]
				} else {
					row[0] = ""
				}
				if i < len(report.SamplingInfo.IgnoredFiles) {
					ignored := report.SamplingInfo.IgnoredFiles[i]
					row[1] = fmt.Sprintf("%s (%s)", ignored.Path, ignored.Reason)
				} else {
					row[1] = ""
				}
				rows[i] = row
			}
			sb.WriteString(f.formatTable(headers, rows))
			sb.WriteString("\n")
		}
	}

	// Suggestions
	if len(report.Suggestions) > 0 {
		sb.WriteString("## 💡 Suggestions\n\n")
		for _, sug := range report.Suggestions {
			sb.WriteString(fmt.Sprintf("- %s\n", sug))
		}
		sb.WriteString("\n")
	}

	// Critical Issues (exclude STATIC_ANALYSIS and duplicate-related)
	criticalIssues := []ReviewIssue{}
	for _, issue := range report.Issues {
		if issue.Category == "STATIC_ANALYSIS" {
			continue
		}
		if strings.Contains(strings.ToLower(issue.Description), "duplicate") {
			continue
		}
		criticalIssues = append(criticalIssues, issue)
	}

	if len(criticalIssues) > 0 {
		sb.WriteString("## ⚠️ Critical Issues\n\n")
		headers := []string{"Severity", "Category", "Description", "Location"}
		rows := make([][]string, len(criticalIssues))
		for i, issue := range criticalIssues {
			location := ""
			if issue.Location != "" {
				location = f.formatFileLink(issue.Location, 0, 0)
			}
			rows[i] = []string{
				fmt.Sprintf("[%s]", issue.Severity),
				issue.Category,
				issue.Description,
				location,
			}
		}
		sb.WriteString(f.formatTable(headers, rows))
		sb.WriteString("\n")
	}

	// Unnecessary Complexity
	if len(report.ComplexityIssues) > 0 {
		sb.WriteString("## 🔧 Unnecessary Complexity\n\n")
		for _, issue := range report.ComplexityIssues {
			severity := "LOW"
			if issue.Score > 0.6 {
				severity = "HIGH"
			} else if issue.Score > 0.3 {
				severity = "MEDIUM"
			}

			sb.WriteString(fmt.Sprintf("### [%s] %s\n\n", severity, issue.Type))
			sb.WriteString(fmt.Sprintf("**Score:** %.0f%%\n\n", issue.Score*100))
			sb.WriteString(fmt.Sprintf("**Description:** %s\n\n", issue.Description))
			
			location := f.formatFileLink(issue.File, issue.Line, issue.Line)
			sb.WriteString(fmt.Sprintf("**Location:** %s", location))
			if issue.Function != "" {
				sb.WriteString(fmt.Sprintf(" - Function: `%s`", issue.Function))
			}
			sb.WriteString("\n\n")

			if issue.Reasoning != "" {
				sb.WriteString(fmt.Sprintf("**Reasoning:** %s\n\n", issue.Reasoning))
			}
			if issue.Suggestion != "" {
				sb.WriteString(fmt.Sprintf("**Suggestion:** %s\n\n", issue.Suggestion))
			}
		}
	}

	// DB/ORM Query Review
	if len(report.DBQueryReviews) > 0 {
		sb.WriteString("## 🗄️ DB/ORM Query Review\n\n")
		headers := []string{"File", "Line", "Function", "Query Type", "Efficiency", "Issues"}
		rows := make([][]string, len(report.DBQueryReviews))
		for i, review := range report.DBQueryReviews {
			fileLink := f.formatFileLink(review.File, review.Line, review.Line)
			issuesStr := ""
			if len(review.Issues) > 0 {
				issueTypes := make([]string, len(review.Issues))
				for j, issue := range review.Issues {
					issueTypes[j] = fmt.Sprintf("%s (%s)", issue.Type, issue.Severity)
				}
				issuesStr = strings.Join(issueTypes, ", ")
			} else {
				issuesStr = "None"
			}
			rows[i] = []string{
				fileLink,
				fmt.Sprintf("%d", review.Line),
				review.Function,
				review.QueryType,
				fmt.Sprintf("%.0f%%", review.EfficiencyScore*100),
				issuesStr,
			}
		}
		sb.WriteString(f.formatTable(headers, rows))
		sb.WriteString("\n")

		// Add detailed issues for each query
		for _, review := range report.DBQueryReviews {
			if len(review.Issues) > 0 {
				fileLink := f.formatFileLink(review.File, review.Line, review.Line)
				sb.WriteString(fmt.Sprintf("### %s\n\n", fileLink))
				for _, issue := range review.Issues {
					sb.WriteString(fmt.Sprintf("- **[%s] %s:** %s", issue.Severity, issue.Type, issue.Description))
					if issue.Suggestion != "" {
						sb.WriteString(fmt.Sprintf(" 💡 *Suggestion: %s*", issue.Suggestion))
					}
					sb.WriteString("\n")
				}
				if review.Recommendation != "" {
					sb.WriteString(fmt.Sprintf("\n**Recommendation:** %s\n\n", review.Recommendation))
				}
			}
		}
	}

	// Duplicate Code
	if len(report.DuplicateBlocks) > 0 {
		sb.WriteString("## 🔄 Duplicate Code\n\n")
		headers := []string{"Similarity", "Lines", "Original", "Duplicate"}
		rows := make([][]string, len(report.DuplicateBlocks))
		for i, dup := range report.DuplicateBlocks {
			originalLink := f.formatFileLink(dup.OriginalFile, dup.OriginalStart, dup.OriginalEnd)
			duplicateLink := f.formatFileLink(dup.DuplicateFile, dup.DuplicateStart, dup.DuplicateEnd)
			rows[i] = []string{
				fmt.Sprintf("%.1f%%", dup.Similarity*100),
				fmt.Sprintf("%d", dup.Lines),
				originalLink,
				duplicateLink,
			}
		}
		sb.WriteString(f.formatTable(headers, rows))
		sb.WriteString("\n")
	}

	// AI-Generated Code Analysis
	aiFiles := []struct {
		path      string
		percentage float64
		confidence float64
	}{}
	for filePath, analysis := range report.FileAnalysis {
		if analysis.AIScore != nil && analysis.AIScore.AIPercentage > 0 && analysis.AIScore.OverallConfidence >= 0.6 {
			aiFiles = append(aiFiles, struct {
				path      string
				percentage float64
				confidence float64
			}{
				path:      filePath,
				percentage: analysis.AIScore.AIPercentage,
				confidence: analysis.AIScore.OverallConfidence,
			})
		}
	}

	if len(aiFiles) > 0 {
		sb.WriteString("## 🤖 AI-Generated Code Analysis\n\n")
		headers := []string{"File", "AI Percentage", "Overall Confidence"}
		rows := make([][]string, len(aiFiles))
		for i, file := range aiFiles {
			rows[i] = []string{
				file.path,
				fmt.Sprintf("%.0f%%", file.percentage),
				fmt.Sprintf("%.0f%%", file.confidence*100),
			}
		}
		sb.WriteString(f.formatTable(headers, rows))
		sb.WriteString("\n")
	}

	// Static Analysis
	staticIssues := []ReviewIssue{}
	for _, issue := range report.Issues {
		if issue.Category == "STATIC_ANALYSIS" {
			staticIssues = append(staticIssues, issue)
		}
	}

	if len(staticIssues) > 0 {
		sb.WriteString("## 📊 Static Analysis\n\n")
		
		// Group by category
		categoryMap := make(map[string][]ReviewIssue)
		for _, issue := range staticIssues {
			category := issue.Subcategory
			if category == "" {
				category = "Other"
			}
			categoryMap[category] = append(categoryMap[category], issue)
		}

		for category, issues := range categoryMap {
			sb.WriteString(fmt.Sprintf("### %s (%d issues)\n\n", category, len(issues)))
			headers := []string{"Severity", "Description", "Location"}
			rows := make([][]string, len(issues))
			for i, issue := range issues {
				location := ""
				if issue.Location != "" {
					location = f.formatFileLink(issue.Location, 0, 0)
				}
				rows[i] = []string{
					fmt.Sprintf("[%s]", issue.Severity),
					issue.Description,
					location,
				}
			}
			sb.WriteString(f.formatTable(headers, rows))
			sb.WriteString("\n")
		}
	}

	// Truncate to 65,000 characters if needed
	content := sb.String()
	return f.truncateToLimit(content, 65000)
}

// formatFileLink generates a GitHub file link
// Format: [file.go:123-145](path/to/file.go#L123-L145)
func (f *Formatter) formatFileLink(filePath string, startLine, endLine int) string {
	if filePath == "" {
		return ""
	}

	// Extract line number from filePath if it contains ":"
	path := filePath
	lineNum := 0
	if idx := strings.LastIndex(filePath, ":"); idx != -1 {
		path = filePath[:idx]
		if _, err := fmt.Sscanf(filePath[idx+1:], "%d", &lineNum); err == nil {
			if startLine == 0 {
				startLine = lineNum
			}
			if endLine == 0 {
				endLine = lineNum
			}
		}
	}

	// Build the link text
	var linkText string
	if startLine > 0 && endLine > 0 {
		if startLine == endLine {
			linkText = fmt.Sprintf("%s:%d", path, startLine)
		} else {
			linkText = fmt.Sprintf("%s:%d-%d", path, startLine, endLine)
		}
	} else {
		linkText = path
	}

	// Build the URL fragment
	var fragment string
	if startLine > 0 && endLine > 0 {
		if startLine == endLine {
			fragment = fmt.Sprintf("#L%d", startLine)
		} else {
			fragment = fmt.Sprintf("#L%d-L%d", startLine, endLine)
		}
	}

	return fmt.Sprintf("[%s](%s%s)", linkText, path, fragment)
}

// formatTable generates a markdown table
func (f *Formatter) formatTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	var sb strings.Builder

	// Write headers
	sb.WriteString("|")
	for _, header := range headers {
		sb.WriteString(fmt.Sprintf(" %s |", f.escapeTableCell(header)))
	}
	sb.WriteString("\n")

	// Write separator
	sb.WriteString("|")
	for range headers {
		sb.WriteString(" --- |")
	}
	sb.WriteString("\n")

	// Write rows
	for _, row := range rows {
		sb.WriteString("|")
		for i := 0; i < len(headers); i++ {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			sb.WriteString(fmt.Sprintf(" %s |", f.escapeTableCell(cell)))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// escapeTableCell escapes special characters in table cells
func (f *Formatter) escapeTableCell(cell string) string {
	// Replace pipe characters and newlines
	cell = strings.ReplaceAll(cell, "|", "\\|")
	cell = strings.ReplaceAll(cell, "\n", " ")
	return cell
}

// truncateToLimit truncates content to maxLen characters, preserving important sections
func (f *Formatter) truncateToLimit(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}

	// Find section boundaries
	sections := []struct {
		start int
		end   int
		name  string
		priority int // Lower = more important
	}{
		{0, 0, "Header", 0},
		{0, 0, "File Sampling", 1},
		{0, 0, "Suggestions", 1},
		{0, 0, "Critical Issues", 0},
		{0, 0, "Unnecessary Complexity", 3},
		{0, 0, "DB/ORM Query Review", 2},
		{0, 0, "Duplicate Code", 3},
		{0, 0, "AI-Generated Code Analysis", 3},
		{0, 0, "Static Analysis", 4},
	}

	// Find section positions
	sectionMarkers := []string{
		"# Katich AI Code Review Report",
		"## 📁 File Sampling",
		"## 💡 Suggestions",
		"## ⚠️ Critical Issues",
		"## 🔧 Unnecessary Complexity",
		"## 🗄️ DB/ORM Query Review",
		"## 🔄 Duplicate Code",
		"## 🤖 AI-Generated Code Analysis",
		"## 📊 Static Analysis",
	}

	for i, marker := range sectionMarkers {
		if idx := strings.Index(content, marker); idx != -1 {
			sections[i].start = idx
			if i > 0 {
				sections[i-1].end = idx
			}
		}
	}
	// Set last section end
	if len(sections) > 0 {
		sections[len(sections)-1].end = len(content)
	}

	// Truncate from least important sections
	truncationNotice := "\n\n---\n\n*Report truncated to fit GitHub comment limit (65,000 characters)*"
	noticeLen := len(truncationNotice)

	// Start with full content and remove sections by priority
	result := content
	for priority := 4; priority >= 0 && len(result) > maxLen-noticeLen; priority-- {
		for i := len(sections) - 1; i >= 0; i-- {
			if sections[i].priority == priority && sections[i].start > 0 && sections[i].end > sections[i].start {
				// Remove this section
				if sections[i].start < len(result) {
					// Keep everything before this section
					if i > 0 && sections[i-1].end > 0 {
						result = result[:sections[i-1].end]
					} else {
						result = result[:sections[i].start]
					}
					if len(result) <= maxLen-noticeLen {
						break
					}
				}
			}
		}
	}

	result += truncationNotice
	return result
}

