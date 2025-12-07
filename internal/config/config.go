package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	LLM        LLMConfig        `yaml:"llm"`
	Embeddings EmbeddingsConfig `yaml:"embeddings"`
	Analysis   AnalysisConfig   `yaml:"analysis"`
}

// LLMConfig contains LLM provider settings
type LLMConfig struct {
	Provider string `yaml:"provider"` // openai, anthropic, local
	APIKey   string `yaml:"api_key"`
	Model    string `yaml:"model"`
	BaseURL  string `yaml:"base_url,omitempty"` // for local LLMs
}

// EmbeddingsConfig contains embedding model settings
type EmbeddingsConfig struct {
	Model    string `yaml:"model"`     // jina-code-v2, bge-code, nomic-embed, snowflake-arctic
	Provider string `yaml:"provider"`  // local, api
	APIKey   string `yaml:"api_key,omitempty"`
}

// AnalysisConfig contains code analysis thresholds
type AnalysisConfig struct {
	MaxFunctionLength   int     `yaml:"max_function_length"`
	ComplexityThreshold int     `yaml:"complexity_threshold"`
	SimilarityThreshold float64 `yaml:"similarity_threshold"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			Provider: "openai",
			Model:    "gpt-4",
		},
		Embeddings: EmbeddingsConfig{
			Model:    "jina-code-v2",
			Provider: "local",
		},
		Analysis: AnalysisConfig{
			MaxFunctionLength:   50,
			ComplexityThreshold: 10,
			SimilarityThreshold: 0.85,
		},
	}
}

// Load loads configuration from a file
func Load(path string) (*Config, error) {
	// If no path specified, try default location
	if path == "" {
		path = ".katich/config.yaml"
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return DefaultConfig(), nil
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Override with environment variables if set
	config.overrideFromEnv()

	return config, nil
}

// overrideFromEnv overrides config values with environment variables
func (c *Config) overrideFromEnv() {
	if apiKey := os.Getenv("KATICH_LLM_API_KEY"); apiKey != "" {
		c.LLM.APIKey = apiKey
	}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" && c.LLM.Provider == "openai" {
		c.LLM.APIKey = apiKey
	}
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" && c.LLM.Provider == "anthropic" {
		c.LLM.APIKey = apiKey
	}
}

// Save saves the configuration to a file
func (c *Config) Save(path string) error {
	// If no path specified, use default location
	if path == "" {
		path = ".katich/config.yaml"
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Check LLM configuration
	if c.LLM.Provider == "" {
		return fmt.Errorf("LLM provider is required")
	}
	if c.LLM.Provider != "local" && c.LLM.APIKey == "" {
		return fmt.Errorf("LLM API key is required for provider: %s", c.LLM.Provider)
	}

	// Check embeddings configuration
	if c.Embeddings.Model == "" {
		return fmt.Errorf("embeddings model is required")
	}

	// Check analysis thresholds
	if c.Analysis.MaxFunctionLength <= 0 {
		return fmt.Errorf("max_function_length must be positive")
	}
	if c.Analysis.ComplexityThreshold <= 0 {
		return fmt.Errorf("complexity_threshold must be positive")
	}
	if c.Analysis.SimilarityThreshold < 0 || c.Analysis.SimilarityThreshold > 1 {
		return fmt.Errorf("similarity_threshold must be between 0 and 1")
	}

	return nil
}
