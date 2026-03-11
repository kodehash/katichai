package llm

import (
	"context"
	"strings"
)

// Role represents the role of a message sender
type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message represents a chat message
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// CompletionRequest represents a request for code completion/review
type CompletionRequest struct {
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
}

// CompletionResponse represents the response from the LLM
type CompletionResponse struct {
	Content string `json:"content"`
	Usage   Usage  `json:"usage"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMProvider defines the interface for LLM interaction
type LLMProvider interface {
	// GenerateCompletion generates a completion for the given messages
	GenerateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	
	// GetName returns the provider name
	GetName() string
	
	// GetModel returns the currently configured model
	GetModel() string
}

// Known context window sizes for common model families
const (
	DefaultTokenLimit = 8192 // Conservative fallback for unrecognised models

	MinOutputTokens = 1500 // Minimum output tokens needed for a useful review
)

// GetModelTokenLimit returns the total context window for a model.
func GetModelTokenLimit(model string) int {
	return GetModelTokenLimitWithConfig(model, 0)
}

// GetModelTokenLimitWithConfig returns the total context window for a model.
// If configLimit > 0, it takes precedence over auto-detection.
// Order matters: more-specific model names are checked before generic ones
// so that e.g. "gpt-4o" matches 128K before the plain "gpt-4" rule (8K).
func GetModelTokenLimitWithConfig(model string, configLimit int) int {
	if configLimit > 0 {
		return configLimit
	}

	m := strings.ToLower(model)

	// ── Anthropic Claude (all variants: 200K) ──
	if strings.Contains(m, "claude") {
		return 200_000
	}

	// ── Google Gemini ──
	if strings.Contains(m, "gemini") {
		if strings.Contains(m, "1.5") {
			return 128_000
		}
		return 1_048_576 // Gemini 2.0+: ~1M
	}

	// ── OpenAI o-series reasoning models (200K) ──
	if isOSeriesModel(m) {
		return 200_000
	}

	// ── OpenAI GPT-4.1 / GPT-4.1-mini / GPT-4.1-nano (1M) ──
	if strings.Contains(m, "gpt-4.1") || strings.Contains(m, "gpt-4-1") {
		return 1_000_000
	}

	// ── OpenAI GPT-4o / GPT-4o-mini (128K) ──
	if strings.Contains(m, "gpt-4o") {
		return 128_000
	}

	// ── OpenAI GPT-4 turbo (128K) — must precede plain gpt-4 ──
	if strings.Contains(m, "gpt-4-turbo") {
		return 128_000
	}

	// ── OpenAI GPT-4 base (8K) ──
	if strings.Contains(m, "gpt-4") {
		return 8_192
	}

	// ── OpenAI GPT-3.5 (modern variants default to 16K) ──
	if strings.Contains(m, "gpt-3.5") {
		return 16_384
	}

	// ── Meta Llama 4 (1M) ──
	if strings.Contains(m, "llama-4") || strings.Contains(m, "llama4") {
		return 1_000_000
	}

	// ── Meta Llama 3.1 / 3.2 / 3.3 (128K) — must precede plain llama3 ──
	if containsAny(m, "llama-3.1", "llama3.1", "llama-3.2", "llama3.2", "llama-3.3", "llama3.3") {
		return 128_000
	}

	// ── Meta Llama 3 base (8K) ──
	if strings.Contains(m, "llama3") || strings.Contains(m, "llama-3") {
		return 8_192
	}

	// ── Google Gemma 3 (128K for 4B+) ──
	if strings.Contains(m, "gemma-3") || strings.Contains(m, "gemma3") {
		return 128_000
	}

	// ── Google Gemma 2 and earlier (8K) ──
	if strings.Contains(m, "gemma") {
		return 8_192
	}

	// ── Mistral Codestral (256K) ──
	if strings.Contains(m, "codestral") {
		return 256_000
	}

	// ── Mixtral (32K) ──
	if strings.Contains(m, "mixtral") {
		return 32_768
	}

	// ── Mistral Large / Medium / Small / Pixtral (128K) ──
	if strings.Contains(m, "mistral") || strings.Contains(m, "pixtral") {
		return 128_000
	}

	// ── DeepSeek V3 / R1 / Coder (128K) ──
	if strings.Contains(m, "deepseek") {
		return 128_000
	}

	// ── Qwen 3 (256K) ──
	if strings.Contains(m, "qwen3") || strings.Contains(m, "qwen-3") {
		return 256_000
	}

	// ── Qwen 2 / 2.5 (128K) ──
	if strings.Contains(m, "qwen") {
		return 128_000
	}

	// ── Phi-3 / Phi-4 (128K) ──
	if strings.Contains(m, "phi-3") || strings.Contains(m, "phi-4") || strings.Contains(m, "phi3") || strings.Contains(m, "phi4") {
		return 128_000
	}

	// ── Yi / Yi-1.5 (200K) ──
	if strings.Contains(m, "yi-") || strings.Contains(m, "yi1") {
		return 200_000
	}

	// ── Command R / R+ (128K) ──
	if strings.Contains(m, "command-r") {
		return 128_000
	}

	return DefaultTokenLimit
}

// isOSeriesModel checks whether the model name refers to an OpenAI o-series
// reasoning model (o1, o3, o4-mini, etc.).
func isOSeriesModel(m string) bool {
	for _, prefix := range []string{"o1", "o3", "o4"} {
		if m == prefix || strings.HasPrefix(m, prefix+"-") || strings.HasPrefix(m, prefix+":") {
			return true
		}
	}
	return false
}

// containsAny returns true if s contains any of the given substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// CalculateMaxOutputTokens computes the maximum output tokens available
// given the model limit and estimated input size.
func CalculateMaxOutputTokens(modelLimit, estimatedInputTokens, buffer int) int {
	available := modelLimit - estimatedInputTokens - buffer
	if available < 512 {
		return 512
	}
	if available > 8192 {
		return 8192
	}
	return available
}
