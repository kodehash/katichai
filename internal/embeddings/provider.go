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

// OpenAI embeddings API limits.
const (
	maxPerInputTokens   = 7500    // conservative cap per string (API limit: 8192)
	maxBatchTotalTokens = 250_000 // margin under OpenAI's 300k per-request sum
	maxBatchInputs      = 2048    // API array cap
	maxSplitDepth       = 3       // recursion limit for adaptive batch splitting
	maxRetries          = 3       // retries for transient (non-token) errors
)

func embEstimateTokens(s string) int {
	if s == "" {
		return 0
	}
	return len(s) / 4
}

func embTruncate(text string, maxTokens int) string {
	maxChars := maxTokens * 4
	if len(text) <= maxChars {
		return text
	}
	return text[:maxChars] + "\n... [truncated for embedding]"
}

// GenerateBatchEmbeddings generates embeddings for multiple texts.
// Batches are formed by estimated token sum (250k cap, 2048 inputs) rather
// than a fixed item count. Each batch is sent through sendBatchWithSplit
// which adaptively splits on token-limit errors (up to 3 recursions).
func (p *OpenAIProvider) GenerateBatchEmbeddings(texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return [][]float32{}, nil
	}

	allEmbeddings := make([][]float32, 0, len(texts))

	var currentBatch []string
	batchTokens := 0
	batchNum := 0
	truncated := 0

	flush := func() error {
		if len(currentBatch) == 0 {
			return nil
		}
		batchNum++
		result, err := p.sendBatchWithSplit(currentBatch, 0)
		if err != nil {
			return fmt.Errorf("batch %d failed: %w", batchNum, err)
		}
		allEmbeddings = append(allEmbeddings, result...)
		fmt.Printf("  📦 Batch %d complete (%d embeddings)\n", batchNum, len(result))
		currentBatch = nil
		batchTokens = 0
		return nil
	}

	for _, text := range texts {
		t := embTruncate(text, maxPerInputTokens)
		if len(t) < len(text) {
			truncated++
		}
		tok := embEstimateTokens(t)

		if len(currentBatch) > 0 && (batchTokens+tok > maxBatchTotalTokens || len(currentBatch) >= maxBatchInputs) {
			if err := flush(); err != nil {
				return nil, err
			}
		}

		currentBatch = append(currentBatch, t)
		batchTokens += tok
	}

	if err := flush(); err != nil {
		return nil, err
	}

	if truncated > 0 {
		fmt.Printf("  ⚠️  %d snippet(s) truncated to fit embedding token limit\n", truncated)
	}

	return allEmbeddings, nil
}

// sendBatchWithSplit sends a batch to the OpenAI API. On token-limit errors
// it splits the batch in half and retries each half recursively (max depth 3).
// Transient errors (timeout, 429 rate limit, 5xx) are retried with backoff.
// Non-retryable errors (quota, auth) fail immediately.
func (p *OpenAIProvider) sendBatchWithSplit(batch []string, depth int) ([][]float32, error) {
	result, err := p.sendEmbeddingBatch(batch)
	if err == nil {
		return result, nil
	}

	// 1. Non-retryable (quota, auth) — fail immediately.
	if isNonRetryableError(err) {
		return nil, err
	}

	// 2. Token-limit error — split rather than retry same payload.
	if isTokenLimitError(err) {
		if depth >= maxSplitDepth {
			return nil, fmt.Errorf("token limit exceeded after %d split attempts (%d items remaining): %w", maxSplitDepth, len(batch), err)
		}

		if len(batch) == 1 {
			emergency := embTruncate(batch[0], maxPerInputTokens/2)
			fmt.Printf("  ⚠️  Single item too large; emergency truncating to ~%d tokens\n", maxPerInputTokens/2)
			result, retryErr := p.sendEmbeddingBatch([]string{emergency})
			if retryErr != nil {
				return nil, fmt.Errorf("single item still exceeds token limit after emergency truncation: %w", retryErr)
			}
			return result, nil
		}

		mid := len(batch) / 2
		fmt.Printf("  🔀 Token limit hit for batch of %d items (depth %d), splitting...\n", len(batch), depth)

		left, leftErr := p.sendBatchWithSplit(batch[:mid], depth+1)
		if leftErr != nil {
			return nil, leftErr
		}
		right, rightErr := p.sendBatchWithSplit(batch[mid:], depth+1)
		if rightErr != nil {
			return nil, rightErr
		}
		return append(left, right...), nil
	}

	// 3. Transient error — retry with exponential backoff.
	var lastErr error = err
	for attempt := 1; attempt < maxRetries; attempt++ {
		backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		fmt.Printf("  ⏳ Retry %d/%d (waiting %s)...\n", attempt, maxRetries-1, backoff)
		time.Sleep(backoff)

		result, retryErr := p.sendEmbeddingBatch(batch)
		if retryErr == nil {
			return result, nil
		}
		if isNonRetryableError(retryErr) {
			return nil, retryErr
		}
		if isTokenLimitError(retryErr) {
			return p.sendBatchWithSplit(batch, depth)
		}
		lastErr = retryErr
	}

	return nil, fmt.Errorf("batch failed after %d retries: %w", maxRetries, lastErr)
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

// isTokenLimitError returns true when the API rejects a request because of
// per-input or per-request token limits (HTTP 400). Must NOT match 429
// rate-limit errors which mention "tokens per minute".
func isTokenLimitError(err error) bool {
	msg := strings.ToLower(err.Error())
	indicators := []string{
		"maximum context length",
		"max_tokens",
		"too many tokens",
		"token limit",
		"total tokens",
		"8192 tokens",
		"300000",
		"exceeds the model",
		"string too long",
		"input too long",
	}
	for _, ind := range indicators {
		if strings.Contains(msg, ind) {
			return true
		}
	}
	return false
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
