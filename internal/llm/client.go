package llm

import (
	"fmt"

	"github.com/katichai/katich/internal/config"
)

// NewClient creates a new LLM client based on configuration
func NewClient(cfg config.LLMConfig) (LLMProvider, error) {
	switch cfg.Provider {
	case "anthropic":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("API key is required for Anthropic provider")
		}
		return NewAnthropicProvider(cfg.APIKey, cfg.Model), nil
	
	case "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("API key is required for OpenAI provider")
		}
		return NewOpenAIProvider(cfg.APIKey, cfg.Model), nil
	
	case "ollama", "local":
		return NewOllamaProvider(cfg.BaseURL, cfg.Model), nil
	
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", cfg.Provider)
	}
}
