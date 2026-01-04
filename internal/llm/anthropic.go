package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AnthropicProvider implements LLMProvider for Anthropic
type AnthropicProvider struct {
	apiKey  string
	model   string
	client  *http.Client
	baseURL string
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey string, model string) *AnthropicProvider {
	if model == "" {
		model = "claude-3-5-sonnet-20240620"
	}
	
	return &AnthropicProvider{
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 180 * time.Second}, // 3 minutes for large context reviews
		baseURL: "https://api.anthropic.com/v1/messages",
	}
}

// GenerateCompletion generates a completion using Anthropic API
func (p *AnthropicProvider) GenerateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// Prepare Anthropic request
	anthropicReq := map[string]interface{}{
		"model":      p.model,
		"messages":   req.Messages,
		"max_tokens": req.MaxTokens,
	}

	if anthropicReq["max_tokens"] == 0 {
		// MaxTokens = 0 means no limit requested (typically for full repository reviews)
		// Use a very high limit to ensure no truncation
		// Claude 3.5 Sonnet supports up to 8192, but we use 32768 for comprehensive reviews
		anthropicReq["max_tokens"] = 32768
	}

	// Extract system message if present (Anthropic handles system differently)
	var filteredMessages []Message
	for _, msg := range req.Messages {
		if msg.Role == RoleSystem {
			anthropicReq["system"] = msg.Content
		} else {
			filteredMessages = append(filteredMessages, msg)
		}
	}
	anthropicReq["messages"] = filteredMessages

	// Marshal request body
	jsonData, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01") // Compatible with all Claude 3 models including 3.5

	// Send request
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var anthropicResp struct {
		Content []struct {
			Text string `json:"text"`
			Type string `json:"type"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &anthropicResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract content
	if len(anthropicResp.Content) == 0 {
		return nil, fmt.Errorf("empty response content")
	}

	return &CompletionResponse{
		Content: anthropicResp.Content[0].Text,
		Usage: Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}, nil
}

// GetName returns the provider name
func (p *AnthropicProvider) GetName() string {
	return "Anthropic"
}

// GetModel returns the currently configured model
func (p *AnthropicProvider) GetModel() string {
	return p.model
}
