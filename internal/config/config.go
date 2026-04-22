package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

// Config holds the application configuration loaded from TOML file, environment variables, and CLI arguments.
type Config struct {
	// Feed aggregation options
	MaxArticlesPerFeed int    `mapstructure:"max_articles_per_feed"`
	MaxDaysOld         int    `mapstructure:"max_days_old"`
	MaxTotalArticles   int    `mapstructure:"max_total_articles"`
	IncludeKeywords    string `mapstructure:"include_keywords"`
	ExcludeKeywords    string `mapstructure:"exclude_keywords"`

	// LLM API options
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`

	// System prompt
	SystemPrompt string `mapstructure:"system_prompt"`

	// Output options
	Output          string `mapstructure:"output"`
	OutputFile      string `mapstructure:"output_file"`
	IncludeArticles bool   `mapstructure:"include_articles"`

	// Internal state
	loaded bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		MaxArticlesPerFeed: 10,
		MaxDaysOld:         7,
		MaxTotalArticles:   20,
		Model:              "deepseek-chat",
		MaxTokens:          4000,
		Temperature:        0.7,
		SystemPrompt: `You are an expert analyst and summariser.
You analyse content from multiple sources and provide
concise, insightful summaries based on user requests.
Focus on key points, trends, and important information.`,
		Output:          "text",
		IncludeArticles: false,
	}
}

// Load loads configuration from TOML file, environment variables, and sets defaults.
// The precedence order is: CLI args > Environment variables > Config file > Defaults.
func Load() (*Config, error) {
	config := DefaultConfig()

	// Initialise Viper
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("toml")

	// Set environment variable prefix first
	v.SetEnvPrefix("LLM_AGGREGATOR")
	v.AutomaticEnv() // Read from environment variables

	// Bind environment variables to config fields
	bindEnvVars(v)

	// Get config path from XDG
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	// Set config file path
	v.AddConfigPath(filepath.Dir(configPath))

	// Try to read config file
	if err := v.ReadInConfig(); err != nil {
		// If config file doesn't exist, that's OK - we'll use defaults + env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		// Config file was found
		config.loaded = true
	}

	// Unmarshal config into struct
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return config, nil
}

// bindEnvVars binds environment variables to viper keys.
func bindEnvVars(v *viper.Viper) {
	// Feed aggregation options
	v.BindEnv("max_articles_per_feed", "LLM_AGGREGATOR_MAX_ARTICLES_PER_FEED")
	v.BindEnv("max_days_old", "LLM_AGGREGATOR_MAX_DAYS_OLD")
	v.BindEnv("max_total_articles", "LLM_AGGREGATOR_MAX_TOTAL_ARTICLES")
	v.BindEnv("include_keywords", "LLM_AGGREGATOR_INCLUDE_KEYWORDS")
	v.BindEnv("exclude_keywords", "LLM_AGGREGATOR_EXCLUDE_KEYWORDS")

	// LLM API options
	v.BindEnv("api_key", "LLM_AGGREGATOR_API_KEY")
	v.BindEnv("model", "LLM_AGGREGATOR_MODEL")
	v.BindEnv("max_tokens", "LLM_AGGREGATOR_MAX_TOKENS")
	v.BindEnv("temperature", "LLM_AGGREGATOR_TEMPERATURE")

	// System prompt
	v.BindEnv("system_prompt", "LLM_AGGREGATOR_SYSTEM_PROMPT")

	// Output options
	v.BindEnv("output", "LLM_AGGREGATOR_OUTPUT")
	v.BindEnv("output_file", "LLM_AGGREGATOR_OUTPUT_FILE")
	v.BindEnv("include_articles", "LLM_AGGREGATOR_INCLUDE_ARTICLES")
}

// GetConfigPath returns the path to the config file.
func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "llm_aggregator", "config.toml"), nil
}

// ConfigExists returns true if a config file exists.
func ConfigExists() (bool, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return false, err
	}
	_, err = os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// Save saves the configuration to the TOML file.
func (c *Config) Save() error {
	v := viper.New()
	v.SetConfigType("toml")

	// Set values from config struct
	v.Set("max_articles_per_feed", c.MaxArticlesPerFeed)
	v.Set("max_days_old", c.MaxDaysOld)
	v.Set("max_total_articles", c.MaxTotalArticles)
	v.Set("include_keywords", c.IncludeKeywords)
	v.Set("exclude_keywords", c.ExcludeKeywords)
	v.Set("api_key", c.APIKey)
	v.Set("model", c.Model)
	v.Set("max_tokens", c.MaxTokens)
	v.Set("temperature", c.Temperature)
	v.Set("system_prompt", c.SystemPrompt)
	v.Set("output", c.Output)
	v.Set("output_file", c.OutputFile)
	v.Set("include_articles", c.IncludeArticles)

	// Get config path
	configPath, err := GetConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write config file
	if err := v.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// IsLoaded returns true if the configuration was loaded from a file.
func (c *Config) IsLoaded() bool {
	return c.loaded
}