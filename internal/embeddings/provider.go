package embeddings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

// EmbeddingProvider generates embeddings for code
type EmbeddingProvider interface {
	GenerateEmbedding(text string) ([]float32, error)
	GenerateBatchEmbeddings(texts []string) ([][]float32, error)
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

// GenerateBatchEmbeddings generates embeddings for multiple texts
// Note: Ollama doesn't have native batch API, so we process sequentially
func (p *OllamaProvider) GenerateBatchEmbeddings(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}
	
	embeddings := make([][]float32, len(texts))
	for i, text := range texts {
		emb, err := p.GenerateEmbedding(text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding for text %d: %w", i, err)
		}
		embeddings[i] = emb
	}
	
	return embeddings, nil
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
			Timeout: 120 * time.Second,
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
		
		// Parse error response to provide better error messages
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		
		if err := json.Unmarshal(body, &errorResp); err == nil {
			// Check for quota errors
			if resp.StatusCode == 429 && (errorResp.Error.Code == "insufficient_quota" || errorResp.Error.Type == "insufficient_quota") {
				return nil, fmt.Errorf("openai quota exceeded: %s. Please check your OpenAI billing and plan. Consider using Ollama for local embeddings instead (see docs)", errorResp.Error.Message)
			}
			// Check for rate limit errors
			if resp.StatusCode == 429 {
				return nil, fmt.Errorf("openai rate limit exceeded: %s. Please wait a moment and try again, or use Ollama for local embeddings", errorResp.Error.Message)
			}
			// Return formatted error message
			return nil, fmt.Errorf("openai error (%s): %s", errorResp.Error.Code, errorResp.Error.Message)
		}
		
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

// GenerateBatchEmbeddings generates embeddings for multiple texts in batches.
// Uses small batches (100) to avoid timeouts on large repos and retries
// transient failures with exponential backoff.
func (p *OpenAIProvider) GenerateBatchEmbeddings(texts []string) ([][]float32, error) {
	const batchSize = 100
	const maxRetries = 3

	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	allEmbeddings := make([][]float32, 0, len(texts))
	totalBatches := (len(texts) + batchSize - 1) / batchSize

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batch := texts[i:end]
		batchNum := i/batchSize + 1

		var batchEmbeddings [][]float32
		var lastErr error

		for attempt := 0; attempt < maxRetries; attempt++ {
			if attempt > 0 {
				backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
				fmt.Printf("  ⏳ Retry %d/%d for batch %d/%d (waiting %s)...\n", attempt, maxRetries-1, batchNum, totalBatches, backoff)
				time.Sleep(backoff)
			}

			batchEmbeddings, lastErr = p.sendEmbeddingBatch(batch)
			if lastErr == nil {
				break
			}

			if isNonRetryableError(lastErr) {
				return nil, lastErr
			}
		}

		if lastErr != nil {
			return nil, fmt.Errorf("batch %d/%d failed after %d attempts: %w", batchNum, totalBatches, maxRetries, lastErr)
		}

		allEmbeddings = append(allEmbeddings, batchEmbeddings...)

		if totalBatches > 1 {
			fmt.Printf("  📦 Batch %d/%d complete (%d embeddings)\n", batchNum, totalBatches, len(batchEmbeddings))
		}
	}

	return allEmbeddings, nil
}

// sendEmbeddingBatch sends a single batch to the OpenAI embeddings API.
func (p *OpenAIProvider) sendEmbeddingBatch(batch []string) ([][]float32, error) {
	requestBody := map[string]interface{}{
		"input": batch,
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
		return nil, fmt.Errorf("openai batch request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}

		if err := json.Unmarshal(body, &errorResp); err == nil {
			if resp.StatusCode == 429 && (errorResp.Error.Code == "insufficient_quota" || errorResp.Error.Type == "insufficient_quota") {
				return nil, fmt.Errorf("openai quota exceeded: %s. Please check your OpenAI billing and plan. Consider using Ollama for local embeddings instead (see docs)", errorResp.Error.Message)
			}
			if resp.StatusCode == 429 {
				return nil, fmt.Errorf("openai rate limit exceeded: %s", errorResp.Error.Message)
			}
			return nil, fmt.Errorf("openai error (%s): %s", errorResp.Error.Code, errorResp.Error.Message)
		}

		return nil, fmt.Errorf("openai returned status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Data []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	embeddings := make([][]float32, len(batch))
	for _, item := range response.Data {
		if item.Index < len(embeddings) {
			embeddings[item.Index] = item.Embedding
		}
	}

	for idx, emb := range embeddings {
		if len(emb) == 0 {
			return nil, fmt.Errorf("missing embedding for index %d in batch", idx)
		}
	}

	return embeddings, nil
}

// isNonRetryableError returns true for errors that won't resolve with a retry
// (e.g. quota exceeded, auth errors).
func isNonRetryableError(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "quota exceeded") ||
		strings.Contains(msg, "insufficient_quota") ||
		strings.Contains(msg, "invalid_api_key") ||
		strings.Contains(msg, "authentication")
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

// GenerateBatchEmbeddings generates embeddings for multiple texts using the best available provider
func (p *HybridProvider) GenerateBatchEmbeddings(texts []string) ([][]float32, error) {
	// Try Ollama first if available
	if p.useOllama {
		embeddings, err := p.ollama.GenerateBatchEmbeddings(texts)
		if err == nil {
			return embeddings, nil
		}
		// If Ollama fails, mark as unavailable and try OpenAI
		p.useOllama = false
	}
	
	// Fall back to OpenAI
	if p.openai != nil {
		return p.openai.GenerateBatchEmbeddings(texts)
	}
	
	return nil, fmt.Errorf("no embedding provider available (Ollama not running, OpenAI key not configured)")
}
