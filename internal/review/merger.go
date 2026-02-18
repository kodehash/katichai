package review

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/katichai/katich/internal/llm"
)

// mergeChunkResults merges multiple chunk review responses into a single unified review
func (e *ReviewEngine) mergeChunkResults(responses []string) string {
	if len(responses) == 0 {
		return ""
	}
	
	if len(responses) == 1 {
		return responses[0]
	}
	
	// Parse each response into structured sections
	parsedChunks := make([]parsedReview, 0, len(responses))
	for _, response := range responses {
		parsed := parseReviewMarkdown(response)
		parsedChunks = append(parsedChunks, parsed)
	}
	
	// Collect raw summaries for AI normalization
	rawSummaries := make([]string, 0, len(parsedChunks))
	for _, pc := range parsedChunks {
		if pc.Summary != "" {
			rawSummaries = append(rawSummaries, pc.Summary)
		}
	}

	// Merge sections — summary is AI-normalized; everything else is deduplicated deterministically
	merged := parsedReview{
		Summary:               e.normalizeSummaryWithAI(rawSummaries),
		CriticalIssues:        mergeCriticalIssues(parsedChunks),
		Suggestions:           mergeSuggestions(parsedChunks),
		UnnecessaryComplexity: mergeComplexity(parsedChunks),
		Score:                 calculateMergedScore(parsedChunks),
	}
	
	// Format back to markdown
	return formatReviewMarkdown(merged)
}

// parsedReview represents a structured review
type parsedReview struct {
	Summary               string
	CriticalIssues        []string
	Suggestions           []string
	UnnecessaryComplexity []string
	Score                 string
}

// parseReviewMarkdown parses a markdown review into structured sections
func parseReviewMarkdown(content string) parsedReview {
	parsed := parsedReview{
		CriticalIssues:        make([]string, 0),
		Suggestions:           make([]string, 0),
		UnnecessaryComplexity: make([]string, 0),
	}
	
	// Split into sections
	summaryRe := regexp.MustCompile(`(?s)## Summary\s*\n(.*?)(?:\n##|$)`)
	criticalRe := regexp.MustCompile(`(?s)## Critical Issues.*?\n(.*?)(?:\n##|$)`)
	suggestionsRe := regexp.MustCompile(`(?s)## Suggestions\s*\n(.*?)(?:\n##|$)`)
	complexityRe := regexp.MustCompile(`(?s)## Unnecessary Complexity\s*\n(.*?)(?:\n##|$)`)
	scoreRe := regexp.MustCompile(`(?s)## Score\s*\n(.*?)(?:\n##|$)`)
	
	// Extract summary
	if match := summaryRe.FindStringSubmatch(content); len(match) > 1 {
		parsed.Summary = strings.TrimSpace(match[1])
	}
	
	// Extract critical issues
	if match := criticalRe.FindStringSubmatch(content); len(match) > 1 {
		issues := strings.TrimSpace(match[1])
		parsed.CriticalIssues = extractBulletPoints(issues)
	}
	
	// Extract suggestions
	if match := suggestionsRe.FindStringSubmatch(content); len(match) > 1 {
		suggestions := strings.TrimSpace(match[1])
		parsed.Suggestions = extractBulletPoints(suggestions)
	}
	
	// Extract complexity issues
	if match := complexityRe.FindStringSubmatch(content); len(match) > 1 {
		complexity := strings.TrimSpace(match[1])
		parsed.UnnecessaryComplexity = extractBulletPoints(complexity)
	}
	
	// Extract score
	if match := scoreRe.FindStringSubmatch(content); len(match) > 1 {
		parsed.Score = strings.TrimSpace(match[1])
	}
	
	return parsed
}

// extractBulletPoints extracts bullet points from markdown text
func extractBulletPoints(text string) []string {
	lines := strings.Split(text, "\n")
	points := make([]string, 0)
	
	currentPoint := ""
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		
		// Check if it's a bullet point
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
			// Save previous point if exists
			if currentPoint != "" {
				points = append(points, currentPoint)
			}
			// Start new point
			currentPoint = trimmed
		} else if currentPoint != "" {
			// Continuation of previous point
			currentPoint += " " + trimmed
		}
	}
	
	// Add last point
	if currentPoint != "" {
		points = append(points, currentPoint)
	}
	
	return points
}

// mergeSummaries combines summaries from multiple chunks
func mergeSummaries(chunks []parsedReview) string {
	summaries := make([]string, 0)
	for _, chunk := range chunks {
		if chunk.Summary != "" {
			summaries = append(summaries, chunk.Summary)
		}
	}
	
	if len(summaries) == 0 {
		return "No summary available."
	}
	
	if len(summaries) == 1 {
		return summaries[0]
	}
	
	// Combine summaries with a unifying intro
	combined := "This review covers multiple areas of the codebase. Key findings:\n\n"
	for i, summary := range summaries {
		combined += fmt.Sprintf("**Area %d**: %s\n\n", i+1, summary)
	}
	
	return strings.TrimSpace(combined)
}

// mergeCriticalIssues deduplicates and combines critical issues
func mergeCriticalIssues(chunks []parsedReview) []string {
	issueMap := make(map[string]bool)
	issues := make([]string, 0)
	
	for _, chunk := range chunks {
		for _, issue := range chunk.CriticalIssues {
			// Extract file:line reference for deduplication
			key := extractFileLineKey(issue)
			if key == "" {
				key = issue // Use full text as key if no file:line found
			}
			
			if !issueMap[key] {
				issueMap[key] = true
				issues = append(issues, issue)
			}
		}
	}
	
	return issues
}

// mergeSuggestions deduplicates and combines suggestions
func mergeSuggestions(chunks []parsedReview) []string {
	suggestionMap := make(map[string]bool)
	suggestions := make([]string, 0)
	
	for _, chunk := range chunks {
		for _, suggestion := range chunk.Suggestions {
			// Use normalized text as key for deduplication
			key := normalizeText(suggestion)
			
			if !suggestionMap[key] {
				suggestionMap[key] = true
				suggestions = append(suggestions, suggestion)
			}
		}
	}
	
	return suggestions
}

// mergeComplexity combines complexity findings
func mergeComplexity(chunks []parsedReview) []string {
	complexityMap := make(map[string]bool)
	complexity := make([]string, 0)
	
	for _, chunk := range chunks {
		for _, item := range chunk.UnnecessaryComplexity {
			// Extract file:line reference for deduplication
			key := extractFileLineKey(item)
			if key == "" {
				key = item // Use full text as key if no file:line found
			}
			
			if !complexityMap[key] {
				complexityMap[key] = true
				complexity = append(complexity, item)
			}
		}
	}
	
	return complexity
}

// calculateMergedScore calculates an overall score from chunk scores
func calculateMergedScore(chunks []parsedReview) string {
	scores := make([]int, 0)
	scoreRe := regexp.MustCompile(`(\d+)`)
	
	for _, chunk := range chunks {
		if match := scoreRe.FindStringSubmatch(chunk.Score); len(match) > 1 {
			var score int
			fmt.Sscanf(match[1], "%d", &score)
			scores = append(scores, score)
		}
	}
	
	if len(scores) == 0 {
		return "N/A"
	}
	
	// Calculate average score
	sum := 0
	for _, score := range scores {
		sum += score
	}
	avgScore := sum / len(scores)
	
	// Determine readiness message
	readiness := "Approve"
	if avgScore < 70 {
		readiness = "Request Changes"
	} else if avgScore < 85 {
		readiness = "Approve with Minor Changes"
	}
	
	return fmt.Sprintf("%d/100 (%s)", avgScore, readiness)
}

// extractFileLineKey extracts file:line reference from issue text
func extractFileLineKey(text string) string {
	// Look for patterns like "file.go:123" or "path/to/file.go:45"
	re := regexp.MustCompile(`([a-zA-Z0-9_/\-\.]+\.(go|java|py|ts|tsx|js|jsx|cs|cpp|c|h)):(\d+)`)
	if match := re.FindStringSubmatch(text); len(match) > 0 {
		return match[0]
	}
	return ""
}

// normalizeText normalizes text for comparison
func normalizeText(text string) string {
	// Convert to lowercase and remove extra whitespace
	normalized := strings.ToLower(text)
	normalized = strings.TrimSpace(normalized)
	normalized = regexp.MustCompile(`\s+`).ReplaceAllString(normalized, " ")
	return normalized
}

// normalizeSummaryWithAI uses the LLM to collate N scattered chunk summaries into a
// single coherent summary. It falls back to the naive concatenation if the LLM call
// fails so that the rest of the review is never blocked.
func (e *ReviewEngine) normalizeSummaryWithAI(summaries []string) string {
	if len(summaries) == 0 {
		return "No summary available."
	}
	if len(summaries) == 1 {
		return summaries[0]
	}

	// Build the user message
	var sb strings.Builder
	sb.WriteString("The following are summary sections from a code review that was split into multiple chunks.\n")
	sb.WriteString("Each section covers a different portion of the same codebase changes.\n\n")
	sb.WriteString("Merge them into ONE cohesive summary. Rules:\n")
	sb.WriteString("- Do NOT add new issues or findings that are not present in the input\n")
	sb.WriteString("- Do NOT remove or omit any key finding\n")
	sb.WriteString("- Do NOT reference chunks, areas, or numbered sections\n")
	sb.WriteString("- Write as a single unified review summary\n")
	sb.WriteString("- Be concise (under 250 words)\n")
	sb.WriteString("- Output ONLY the summary text, with no markdown headers\n\n")
	sb.WriteString("--- Chunk Summaries ---\n\n")
	for i, s := range summaries {
		sb.WriteString(fmt.Sprintf("Chunk %d:\n%s\n\n", i+1, strings.TrimSpace(s)))
		if i < len(summaries)-1 {
			sb.WriteString("---\n\n")
		}
	}

	userMsg := sb.String()

	fmt.Printf("🔀 Normalizing %d chunk summaries into a unified summary...\n", len(summaries))

	resp, err := e.llmClient.GenerateCompletion(context.Background(), llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Content: llm.SystemPrompt},
			{Role: llm.RoleUser, Content: userMsg},
		},
		Temperature: 0.2,
		MaxTokens:   1024,
	})
	if err != nil {
		fmt.Printf("⚠️  Failed to normalize summary with AI (falling back to concatenation): %v\n", err)
		return mergeSummaries(make([]parsedReview, 0)) // trigger the fallback path below
	}

	normalized := strings.TrimSpace(resp.Content)
	if normalized == "" {
		fmt.Println("⚠️  AI returned empty summary, falling back to concatenation")
		// Re-build parsedReview slice just for the fallback
		fakeChunks := make([]parsedReview, len(summaries))
		for i, s := range summaries {
			fakeChunks[i] = parsedReview{Summary: s}
		}
		return mergeSummaries(fakeChunks)
	}

	return normalized
}

// formatReviewMarkdown formats a parsed review back to markdown
func formatReviewMarkdown(review parsedReview) string {
	var sb strings.Builder
	
	// Summary
	sb.WriteString("## Summary\n")
	sb.WriteString(review.Summary)
	sb.WriteString("\n\n")
	
	// Critical Issues
	if len(review.CriticalIssues) > 0 {
		sb.WriteString("## Critical Issues (Blockers)\n")
		for _, issue := range review.CriticalIssues {
			sb.WriteString(issue)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	
	// Suggestions
	if len(review.Suggestions) > 0 {
		sb.WriteString("## Suggestions\n")
		for _, suggestion := range review.Suggestions {
			sb.WriteString(suggestion)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	
	// Unnecessary Complexity
	if len(review.UnnecessaryComplexity) > 0 {
		sb.WriteString("## Unnecessary Complexity\n")
		for _, item := range review.UnnecessaryComplexity {
			sb.WriteString(item)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	
	// Score
	sb.WriteString("## Score\n")
	sb.WriteString(review.Score)
	sb.WriteString("\n")
	
	return sb.String()
}




