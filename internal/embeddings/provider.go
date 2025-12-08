package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EmbeddingProvider generates embeddings for code
type EmbeddingProvider interface {
	GenerateEmbedding(text string) ([]float32, error)
	GetDimension() int
	GetName() string
}

// OllamaProvider uses Ollama for local embeddings
type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	if model == "" {
		model = "nomic-embed-text"
	}

	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateEmbedding generates an embedding using Ollama
func (p *OllamaProvider) GenerateEmbedding(text string) ([]float32, error) {
	requestBody := map[string]interface{}{
		"model":  p.model,
		"prompt": text,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/embeddings", p.baseURL)
	resp, err := p.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Embedding []float32 `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Embedding) == 0 {
		return nil, fmt.Errorf("ollama returned empty embedding")
	}

	return response.Embedding, nil
}

// GetDimension returns the embedding dimension
func (p *OllamaProvider) GetDimension() int {
	return 768 // nomic-embed-text dimension
}

// GetName returns the provider name
func (p *OllamaProvider) GetName() string {
	return "Ollama"
}

// IsAvailable checks if Ollama is available
func (p *OllamaProvider) IsAvailable() bool {
	url := fmt.Sprintf("%s/api/tags", p.baseURL)
	resp, err := p.client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// OpenAIProvider uses OpenAI API for embeddings
type OpenAIProvider struct {
	apiKey string
	model  string
	client *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	if model == "" {
		model = "text-embedding-3-small"
	}

	return &OpenAIProvider{
		apiKey: apiKey,
		model:  model,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GenerateEmbedding generates an embedding using OpenAI
func (p *OpenAIProvider) GenerateEmbedding(text string) ([]float32, error) {
	requestBody := map[string]interface{}{
		"input": text,
		"model": p.model,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", p.apiKey))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Data) == 0 || len(response.Data[0].Embedding) == 0 {
		return nil, fmt.Errorf("openai returned empty embedding")
	}

	return response.Data[0].Embedding, nil
}

// GetDimension returns the embedding dimension
func (p *OpenAIProvider) GetDimension() int {
	return 1536 // text-embedding-3-small dimension
}

// GetName returns the provider name
func (p *OpenAIProvider) GetName() string {
	return "OpenAI"
}

// HybridProvider tries Ollama first, falls back to OpenAI
type HybridProvider struct {
	ollama *OllamaProvider
	openai *OpenAIProvider
	useOllama bool
}

// NewHybridProvider creates a new hybrid provider
func NewHybridProvider(ollamaURL, ollamaModel, openaiKey, openaiModel string) *HybridProvider {
	ollama := NewOllamaProvider(ollamaURL, ollamaModel)
	
	var openai *OpenAIProvider
	if openaiKey != "" {
		openai = NewOpenAIProvider(openaiKey, openaiModel)
	}

	// Check if Ollama is available
	useOllama := ollama.IsAvailable()

	return &HybridProvider{
		ollama:    ollama,
		openai:    openai,
		useOllama: useOllama,
	}
}

// GenerateEmbedding generates an embedding using the best available provider
func (p *HybridProvider) GenerateEmbedding(text string) ([]float32, error) {
	// Try Ollama first if available
	if p.useOllama {
		embedding, err := p.ollama.GenerateEmbedding(text)
		if err == nil {
			return embedding, nil
		}
		// If Ollama fails, mark as unavailable and try OpenAI
		p.useOllama = false
	}

	// Fall back to OpenAI
	if p.openai != nil {
		return p.openai.GenerateEmbedding(text)
	}

	return nil, fmt.Errorf("no embedding provider available (Ollama not running, OpenAI key not configured)")
}

// GetDimension returns the embedding dimension
func (p *HybridProvider) GetDimension() int {
	if p.useOllama {
		return p.ollama.GetDimension()
	}
	if p.openai != nil {
		return p.openai.GetDimension()
	}
	return 768 // Default to Ollama dimension
}

// GetName returns the active provider name
func (p *HybridProvider) GetName() string {
	if p.useOllama {
		return p.ollama.GetName()
	}
	if p.openai != nil {
		return p.openai.GetName()
	}
	return "None"
}

// GetActiveProvider returns which provider is being used
func (p *HybridProvider) GetActiveProvider() string {
	if p.useOllama {
		return "Ollama (local)"
	}
	if p.openai != nil {
		return "OpenAI (API)"
	}
	return "None"
}
