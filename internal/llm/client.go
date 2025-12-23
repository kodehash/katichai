package llm

import (
	"context"
	"fmt"

	"github.com/katichai/katich/internal/config"
)

// NewClient creates a new LLM client based on configuration
// If keyFetcher is provided and api_key is empty, it will fetch the key from the API server
func NewClient(cfg config.LLMConfig, projectName string, keyFetcher *KeyFetcher) (LLMProvider, error) {
	apiKey := cfg.APIKey
	
	// If API key is empty and keyFetcher is provided, fetch from API
	if apiKey == "" && keyFetcher != nil {
		ctx := context.Background()
		fetchedKey, err := keyFetcher.FetchLLMKey(ctx, projectName, cfg.Provider)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch API key: %w", err)
		}
		apiKey = fetchedKey
	}
	
	switch cfg.Provider {
	case "anthropic":
		if apiKey == "" {
			return nil, fmt.Errorf("API key is required for Anthropic provider")
		}
		return NewAnthropicProvider(apiKey, cfg.Model), nil
	
	case "openai":
		if apiKey == "" {
			return nil, fmt.Errorf("API key is required for OpenAI provider")
		}
		return NewOpenAIProvider(apiKey, cfg.Model), nil
	
	case "ollama", "local":
		return NewOllamaProvider(cfg.BaseURL, cfg.Model), nil
	
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}
