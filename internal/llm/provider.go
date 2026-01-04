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

// Provider-based token limits (simpler than per-model tracking)
const (
	OpenAITokenLimit    = 8192   // Conservative limit for OpenAI models
	AnthropicTokenLimit = 200000 // Claude models support 200K context window
	OllamaTokenLimit    = 8192   // Local models vary, use conservative default
)

// GetModelTokenLimit returns the token limit based on the provider
// Accepts the model name and returns the appropriate provider-based limit
func GetModelTokenLimit(model string) int {
	modelLower := strings.ToLower(model)
	
	// Anthropic Claude models
	if strings.Contains(modelLower, "claude") {
		return AnthropicTokenLimit
	}
	
	// OpenAI models (gpt-3.5, gpt-4, etc.)
	if strings.Contains(modelLower, "gpt") {
		return OpenAITokenLimit
	}
	
	// Default to OpenAI limit for unknown models
	return OpenAITokenLimit
}
