package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// KeyCacheEntry represents a cached API key with expiration
type KeyCacheEntry struct {
	Key       string
	ExpiresAt time.Time
}

// KeyFetcher handles fetching LLM API keys from a centralized server
type KeyFetcher struct {
	apiServerURL string
	authToken    string
	client       *http.Client
	cache        map[string]*KeyCacheEntry
	cacheMutex   sync.RWMutex
	cacheTTL     time.Duration
}

// NewKeyFetcher creates a new KeyFetcher instance
func NewKeyFetcher(apiServerURL, authToken string) *KeyFetcher {
	return &KeyFetcher{
		apiServerURL: apiServerURL,
		authToken:    authToken,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:    make(map[string]*KeyCacheEntry),
		cacheTTL: 1 * time.Hour, // Cache for 1 hour
	}
}

// FetchLLMKey fetches an LLM API key from the centralized server
// It uses caching to avoid repeated API calls
func (kf *KeyFetcher) FetchLLMKey(ctx context.Context, projectName, provider string) (string, error) {
	// Create cache key from project name and provider
	cacheKey := fmt.Sprintf("%s:%s", projectName, provider)
	
	// Check cache first
	kf.cacheMutex.RLock()
	if entry, exists := kf.cache[cacheKey]; exists {
		// Check if cache is still valid
		if time.Now().Before(entry.ExpiresAt) {
			key := entry.Key
			kf.cacheMutex.RUnlock()
			return key, nil
		}
		// Cache expired, remove it
		delete(kf.cache, cacheKey)
	}
	kf.cacheMutex.RUnlock()
	
	// Fetch from API
	key, expiresAt, err := kf.fetchFromAPI(ctx, projectName, provider)
	if err != nil {
		return "", fmt.Errorf("failed to fetch LLM key from API: %w", err)
	}
	
	// Store in cache
	kf.cacheMutex.Lock()
	kf.cache[cacheKey] = &KeyCacheEntry{
		Key:       key,
		ExpiresAt: expiresAt,
	}
	kf.cacheMutex.Unlock()
	
	return key, nil
}

// fetchFromAPI makes the actual API call to fetch the LLM key
func (kf *KeyFetcher) fetchFromAPI(ctx context.Context, projectName, provider string) (string, time.Time, error) {
	// Prepare request body
	requestBody := map[string]string{
		"project_name": projectName,
		"provider":     provider,
	}
	
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	// Use the full URL from config (includes the complete API path)
	url := kf.apiServerURL
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", kf.authToken))
	
	// Log API call details
	fmt.Printf("🔑 Fetching LLM API key from server:\n")
	fmt.Printf("   URL: %s\n", url)
	fmt.Printf("   Project: %s\n", projectName)
	fmt.Printf("   Provider: %s\n", provider)
	
	// Send request
	resp, err := kf.client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	
	// Check status code
	if resp.StatusCode != http.StatusOK {
		// Read response body for error details
		body, _ := json.Marshal(requestBody)
		fmt.Printf("   Request Body: %s\n", string(body))
		fmt.Printf("   Response Status: %d\n", resp.StatusCode)
		
		// Try to read error response body
		respBody, err := io.ReadAll(resp.Body)
		if err == nil && len(respBody) > 0 {
			fmt.Printf("   Error Response: %s\n", string(respBody))
		}
		
		return "", time.Time{}, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	
	// Parse response
	var response struct {
		APIKey    string `json:"api_key"`
		ExpiresAt string `json:"expires_at,omitempty"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", time.Time{}, fmt.Errorf("failed to decode response: %w", err)
	}
	
	if response.APIKey == "" {
		return "", time.Time{}, fmt.Errorf("API returned empty api_key")
	}
	
	// Parse expiration time if provided, otherwise use cache TTL
	expiresAt := time.Now().Add(kf.cacheTTL)
	if response.ExpiresAt != "" {
		if parsed, err := time.Parse(time.RFC3339, response.ExpiresAt); err == nil {
			expiresAt = parsed
		}
	}
	
	fmt.Printf("   ✅ Successfully fetched API key (expires: %s)\n", expiresAt.Format(time.RFC3339))
	
	return response.APIKey, expiresAt, nil
}

