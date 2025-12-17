package llm

import (
	"fmt"
	"strings"
)

// Token estimation constants
const (
	CHARS_PER_TOKEN = 4 // Rough estimate: 1 token ≈ 4 characters
)

// EstimateTokens estimates the number of tokens in a text string
// Uses a simple heuristic: ~4 characters per token
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}
	return len(text) / CHARS_PER_TOKEN
}

// TruncateToTokenLimit truncates text to fit within a token limit
func TruncateToTokenLimit(text string, maxTokens int) string {
	maxChars := maxTokens * CHARS_PER_TOKEN
	if len(text) <= maxChars {
		return text
	}
	
	truncated := text[:maxChars]
	return truncated + "\n... [truncated to fit token limit]"
}

// CalculateTokenBudget calculates how many tokens are available for a specific component
func CalculateTokenBudget(totalBudget int, usedTokens int, buffer int) int {
	remaining := totalBudget - usedTokens - buffer
	if remaining < 0 {
		return 0
	}
	return remaining
}

// TokenUsage tracks token usage across components
type TokenUsage struct {
	SystemPrompt int
	Context      int
	Diff         int
	Buffer       int
	Total        int
}

// Validate checks if token usage is within budget
func (t *TokenUsage) Validate(budget int) bool {
	t.Total = t.SystemPrompt + t.Context + t.Diff + t.Buffer
	return t.Total <= budget
}

// FormatUsage returns a human-readable token usage string
func (t *TokenUsage) FormatUsage() string {
	return strings.Join([]string{
		"System Prompt: " + formatTokenCount(t.SystemPrompt),
		"Context: " + formatTokenCount(t.Context),
		"Diff: " + formatTokenCount(t.Diff),
		"Buffer: " + formatTokenCount(t.Buffer),
		"Total: " + formatTokenCount(t.Total),
	}, ", ")
}

func formatTokenCount(count int) string {
	if count >= 1000 {
		k := float64(count) / 1000.0
		if k == float64(int(k)) {
			return fmt.Sprintf("%dk", int(k))
		}
		return fmt.Sprintf("%.1fk", k)
	}
	return fmt.Sprintf("%d", count)
}

