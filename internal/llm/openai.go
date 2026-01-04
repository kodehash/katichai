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

// OpenAIProvider implements LLMProvider for OpenAI
type OpenAIProvider struct {
	apiKey  string
	model   string
	client  *http.Client
	baseURL string
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string, model string) *OpenAIProvider {
	if model == "" {
		model = "gpt-4o"
	}

	return &OpenAIProvider{
		apiKey:  apiKey,
		model:   model,
		client:  &http.Client{Timeout: 120 * time.Second}, // 2 minutes for reviews
		baseURL: "https://api.openai.com/v1/chat/completions",
	}
}

// GenerateCompletion generates a completion using OpenAI API
func (p *OpenAIProvider) GenerateCompletion(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	// Prepare OpenAI request
	openAIReq := map[string]interface{}{
		"model":       p.model,
		"messages":    req.Messages,
		"temperature": req.Temperature,
	}

	if req.MaxTokens > 0 {
		openAIReq["max_tokens"] = req.MaxTokens
	}

	// Marshal request body
	jsonData, err := json.Marshal(openAIReq)
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
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

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
		return nil, fmt.Errorf("openai API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response
	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract content
	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("empty response choices")
	}

	return &CompletionResponse{
		Content: openAIResp.Choices[0].Message.Content,
		Usage: Usage{
			PromptTokens:     openAIResp.Usage.PromptTokens,
			CompletionTokens: openAIResp.Usage.CompletionTokens,
			TotalTokens:      openAIResp.Usage.TotalTokens,
		},
	}, nil
}

// GetName returns the provider name
func (p *OpenAIProvider) GetName() string {
	return "OpenAI"
}

// GetModel returns the currently configured model
func (p *OpenAIProvider) GetModel() string {
	return p.model
}
