package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	ProjectName string          `yaml:"project_name,omitempty"`
	LLM         LLMConfig       `yaml:"llm"`
	Embeddings  EmbeddingsConfig `yaml:"embeddings"`
	Analysis    AnalysisConfig  `yaml:"analysis"`
	APIServer   APIServerConfig `yaml:"api_server,omitempty"`
	Review      ReviewConfig    `yaml:"review,omitempty"`
	Context     ContextConfig   `yaml:"context,omitempty"`
}

// ContextConfig controls where context and embeddings are loaded from
type ContextConfig struct {
	Source string             `yaml:"source"` // "local" (default) or "remote"
	Remote RemoteContextConfig `yaml:"remote,omitempty"`
}

// RemoteContextConfig is used when source is "remote" (origin branch/directory)
type RemoteContextConfig struct {
	Branch    string `yaml:"branch"`    // default: "main"
	Directory string `yaml:"directory"` // default: "katich-ai-context"
}

// LLMConfig contains LLM provider settings
type LLMConfig struct {
	Provider        string `yaml:"provider"`                    // openai, anthropic, deepseek, qwen, local, ollama
	APIKey          string `yaml:"api_key"`
	Model           string `yaml:"model"`
	BaseURL         string `yaml:"base_url,omitempty"`          // override endpoint (supports OpenAI-compatible APIs)
	MaxInputTokens  int    `yaml:"max_input_tokens,omitempty"`  // max tokens for input (default: 20000)
	TokensPerMinute int    `yaml:"tokens_per_minute,omitempty"` // TPM rate limit (0 = no rate limiting)
}

// EmbeddingsConfig contains embedding model settings
type EmbeddingsConfig struct {
	Model    string `yaml:"model"`     // jina-code-v2, bge-code, nomic-embed, snowflake-arctic
	Provider string `yaml:"provider"`  // local, api
	APIKey   string `yaml:"api_key,omitempty"`
}

// AnalysisConfig contains code analysis thresholds
type AnalysisConfig struct {
	MaxFunctionLength     int            `yaml:"max_function_length"`
	ComplexityThreshold   int            `yaml:"complexity_threshold"`
	SimilarityThreshold   float64        `yaml:"similarity_threshold"`
	Sampling              SamplingConfig `yaml:"sampling,omitempty"`
	
	// Similarity Detection Settings
	MinFunctionLines      int     `yaml:"min_function_lines"`       // Minimum lines to check (default: 5)
	DuplicateThreshold    float64 `yaml:"duplicate_threshold"`      // Duplicate threshold (default: 0.85)
	IgnoreTrivialPatterns bool    `yaml:"ignore_trivial_patterns"`  // Skip getters/setters/CRUD (default: true)
}

// SamplingConfig contains diff sampling settings
type SamplingConfig struct {
	Enabled        bool `yaml:"enabled"`          // enable smart sampling (default: true)
	MaxFiles       int  `yaml:"max_files"`        // max files to review (default: 20)
	ContextLines   int  `yaml:"context_lines"`    // context lines to keep (default: 2)
	SkipGenerated  bool `yaml:"skip_generated"`   // skip generated files (default: true)
	SkipTests      bool `yaml:"skip_tests"`       // skip test files (default: false)
	AdaptiveBudget bool `yaml:"adaptive_budget"`  // dynamically adjust based on context (default: true)
}

// APIServerConfig contains API server settings for fetching LLM keys
type APIServerConfig struct {
	Enabled bool   `yaml:"enabled"` // whether to use API server
	URL     string `yaml:"url,omitempty"` // API server URL (from env var KATICH_API_SERVER_URL)
	Token   string `yaml:"token,omitempty"` // authentication token (from config.yaml or env var KATICH_API_TOKEN)
}

// ReviewConfig contains review output settings
type ReviewConfig struct {
	GenerateHTML      bool   `yaml:"generate_html"`               // whether to generate HTML reports
	HTMLOutputPath    string `yaml:"html_output_path,omitempty"`   // output path for HTML reports (default: .katich/reports)
	GenerateGFM       bool   `yaml:"generate_gfm"`                // whether to generate GFM reports
	GFMOutputPath     string `yaml:"gfm_output_path,omitempty"`   // output path for GFM reports (default: .katich/reports)
	DetectAICode      bool   `yaml:"detect_ai_code"`              // run AI-generated code detection (default: false)
	GenerateFixPrompt bool   `yaml:"generate_fix_prompt"`         // generate a fix prompt for AI assistants (default: true)
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		LLM: LLMConfig{
			Provider:        "openai",
			Model:           "gpt-4",
			MaxInputTokens:  20000,
			TokensPerMinute: 90000, // Conservative default; adjust to match your API tier
		},
		Embeddings: EmbeddingsConfig{
			Model:    "jina-code-v2",
			Provider: "local",
		},
		Analysis: AnalysisConfig{
			MaxFunctionLength:     50,
			ComplexityThreshold:   10,
			SimilarityThreshold:   0.70,
			MinFunctionLines:      5,
			DuplicateThreshold:    0.85,
			IgnoreTrivialPatterns: true,
			Sampling: SamplingConfig{
				Enabled:        true,
				MaxFiles:       50,
				ContextLines:   2,
				SkipGenerated:  true,
				SkipTests:      false,
				AdaptiveBudget: true,
			},
		},
		APIServer: APIServerConfig{
			Enabled: false,
		},
		Review: ReviewConfig{
			GenerateHTML:      true,  // Default to true for HTML reports
			HTMLOutputPath:    ".katich/reports",
			GenerateGFM:       false, // Default to false for GFM reports
			GFMOutputPath:     ".katich/reports",
			DetectAICode:      false, // AI detection is opt-in
			GenerateFixPrompt: true,  // Fix prompt is on by default
		},
		Context: ContextConfig{
			Source: "local",
			Remote: RemoteContextConfig{
				Branch:    "main",
				Directory: "katich-ai-context",
			},
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
	if apiKey := os.Getenv("DEEPSEEK_API_KEY"); apiKey != "" && c.LLM.Provider == "deepseek" {
		c.LLM.APIKey = apiKey
	}
	// Alibaba DashScope key used by Qwen models
	if apiKey := os.Getenv("DASHSCOPE_API_KEY"); apiKey != "" && c.LLM.Provider == "qwen" {
		c.LLM.APIKey = apiKey
	}

	// Embeddings always use OpenAI (text-embedding-3-small), regardless of the LLM provider.
	// Check a dedicated env var first, then fall back to OPENAI_API_KEY directly.
	if apiKey := os.Getenv("KATICH_EMBEDDINGS_API_KEY"); apiKey != "" {
		c.Embeddings.APIKey = apiKey
	} else if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" && c.Embeddings.APIKey == "" {
		// Always pick up OPENAI_API_KEY for embeddings, even if LLM provider is not OpenAI
		c.Embeddings.APIKey = apiKey
	}
	
	// Override API server URL from environment variable
	if apiServerURL := os.Getenv("KATICH_API_SERVER_URL"); apiServerURL != "" {
		c.APIServer.URL = apiServerURL
	}
	
	// Override API server token from environment variable
	if apiToken := os.Getenv("KATICH_API_TOKEN"); apiToken != "" {
		c.APIServer.Token = apiToken
	}
	
	// Override review HTML generation from environment variable
	if generateHTML := os.Getenv("KATICH_GENERATE_HTML"); generateHTML == "true" || generateHTML == "1" {
		c.Review.GenerateHTML = true
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
	
	// If API server is enabled, validate its configuration
	if c.APIServer.Enabled {
		if c.APIServer.URL == "" {
			return fmt.Errorf("API server URL is required when API server is enabled (set KATICH_API_SERVER_URL)")
		}
		if c.APIServer.Token == "" {
			return fmt.Errorf("API server token is required when API server is enabled (set KATICH_API_TOKEN or add to config)")
		}
		// If API server is enabled, api_key can be empty (will be fetched)
	} else if c.LLM.Provider != "local" && c.LLM.Provider != "ollama" && c.LLM.APIKey == "" {
		// If API server is disabled, api_key is required for non-local providers
		return fmt.Errorf("LLM API key is required for provider: %s (or enable API server)", c.LLM.Provider)
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

// GetProjectName returns the project name from config, or extracts from Git if not set
func (c *Config) GetProjectName() (string, error) {
	if c.ProjectName != "" {
		return c.ProjectName, nil
	}
	
	// Try to extract from Git repository
	// Import git package would create circular dependency, so we'll handle this at call site
	// For now, return empty string and let caller handle Git extraction
	return "", fmt.Errorf("project name not set in config")
}
