package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"

	"llm_aggregator/internal/cli"
	"llm_aggregator/internal/defaults"
	"llm_aggregator/internal/progress"
	"llm_aggregator/internal/runtime"
	"llm_aggregator/internal/style"
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
	BaseURL     string  `mapstructure:"base_url"`
	Model       string  `mapstructure:"model"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`

	// System prompt
	SystemPrompt string `mapstructure:"system_prompt"`

	// Output options
	Output          string `mapstructure:"output"`
	OutputFile      string `mapstructure:"output_file"`
	IncludeArticles bool   `mapstructure:"include_articles"`
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		MaxArticlesPerFeed: defaults.DefaultMaxArticlesPerFeed,
		MaxDaysOld:         defaults.DefaultMaxDaysOld,
		MaxTotalArticles:   defaults.DefaultMaxTotalArticles,
		Model:              defaults.DefaultModel,
		BaseURL:            defaults.DefaultBaseURL,
		MaxTokens:          defaults.DefaultMaxTokens,
		Temperature:        defaults.DefaultTemperature,
		SystemPrompt:       defaults.DefaultSystemPrompt,
		Output:             defaults.DefaultOutput,
		IncludeArticles:    defaults.DefaultIncludeArticles,
	}
}

// Load loads configuration from TOML file, environment variables, and sets defaults.
// The precedence order is: CLI args > Environment variables > Config file > Defaults.
func Load() (*Config, error) {
	// Get the global viper instance which handles precedence automatically
	v := GetViper()

	// Return the current configuration as a Config struct
	return ViperToConfig(v), nil
}

// GetViper returns the global viper instance with all configuration sources set up.
// The precedence order is: CLI flags > Environment variables > Config file > Defaults.
func GetViper() *viper.Viper {
	// Set defaults first (lowest priority)
	setDefaults()

	// Initialise or get existing Viper instance
	v := viper.GetViper()

	// Ensure defaults are set (for fresh instances)
	setDefaultsOn(v)

	// Set environment variable prefix
	v.SetEnvPrefix("LLM_AGGREGATOR")
	v.AutomaticEnv()

	// Bind environment variables to config fields
	bindEnvVars(v)

	// Get config path from XDG
	configPath, err := GetConfigPath()
	if err == nil {
		// Set config file path
		v.SetConfigFile(configPath)

		// Try to read config file
		if err := v.ReadInConfig(); err != nil {
			// If config file doesn't exist, that's OK - we'll use defaults + env vars
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				fmt.Fprintln(os.Stderr, style.Warningf("error reading config file: %v", err))
			}
		}
	}

	return v
}

// setDefaults sets default values on the global viper instance.
func setDefaults() {
	v := viper.GetViper()
	setDefaultsOn(v)
}

// setDefaultsOn sets default values on the given viper instance.
func setDefaultsOn(v *viper.Viper) {
	// Feed aggregation defaults
	v.SetDefault("max_articles_per_feed", defaults.DefaultMaxArticlesPerFeed)
	v.SetDefault("max_days_old", defaults.DefaultMaxDaysOld)
	v.SetDefault("max_total_articles", defaults.DefaultMaxTotalArticles)

	// LLM API defaults
	v.SetDefault("base_url", defaults.DefaultBaseURL)
	v.SetDefault("model", defaults.DefaultModel)
	v.SetDefault("max_tokens", defaults.DefaultMaxTokens)
	v.SetDefault("temperature", defaults.DefaultTemperature)

	// System prompt default
	v.SetDefault("system_prompt", defaults.DefaultSystemPrompt)

	// Output defaults
	v.SetDefault("output", defaults.DefaultOutput)
	v.SetDefault("include_articles", defaults.DefaultIncludeArticles)
}

// ViperToConfig converts viper configuration to a Config struct.
func ViperToConfig(v *viper.Viper) *Config {
	return &Config{
		MaxArticlesPerFeed: v.GetInt("max_articles_per_feed"),
		MaxDaysOld:         v.GetInt("max_days_old"),
		MaxTotalArticles:   v.GetInt("max_total_articles"),
		IncludeKeywords:    v.GetString("include_keywords"),
		ExcludeKeywords:    v.GetString("exclude_keywords"),
		APIKey:             v.GetString("api_key"),
		BaseURL:            v.GetString("base_url"),
		Model:              v.GetString("model"),
		MaxTokens:          v.GetInt("max_tokens"),
		Temperature:        v.GetFloat64("temperature"),
		SystemPrompt:       v.GetString("system_prompt"),
		Output:             v.GetString("output"),
		OutputFile:         v.GetString("output_file"),
		IncludeArticles:    v.GetBool("include_articles"),
	}
}

// BindCLIArgs binds CLI argument values to viper with highest precedence.
// This should be called after parsing CLI arguments to override config file/env defaults.
func BindCLIArgs(v *viper.Viper, args map[string]any) {
	for key, value := range args {
		if value != nil && !isZero(value) {
			v.Set(key, value)
		}
	}
}

// isZero checks if a value is the zero value for its type.
func isZero(v any) bool {
	switch val := v.(type) {
	case string:
		return val == ""
	case int, int8, int16, int32, int64:
		return val == 0
	case float32, float64:
		return val == 0
	case bool:
		return !val
	default:
		return false
	}
}

// ViperToRuntime converts viper configuration directly to runtime.Runtime.
func ViperToRuntime(v *viper.Viper, feedsFile, prompt string) *runtime.Runtime {
	rt := &runtime.Runtime{
		FeedsFile: feedsFile,
		Prompt:    prompt,
		Progress:  &progress.NoopLogger{},
	}

	// Feed aggregation options
	rt.MaxArticlesPerFeed = v.GetInt("max_articles_per_feed")
	rt.MaxDaysOld = v.GetInt("max_days_old")
	rt.MaxTotalArticles = v.GetInt("max_total_articles")

	// Keywords
	includeKeywords := v.GetString("include_keywords")
	excludeKeywords := v.GetString("exclude_keywords")
	if includeKeywords != "" {
		rt.IncludeKeywords = cli.ParseKeywords(includeKeywords)
	}
	if excludeKeywords != "" {
		rt.ExcludeKeywords = cli.ParseKeywords(excludeKeywords)
	}

	// LLM API options
	rt.APIKey = v.GetString("api_key")
	rt.BaseURL = v.GetString("base_url")
	rt.Model = v.GetString("model")
	rt.MaxTokens = v.GetInt("max_tokens")
	rt.Temperature = v.GetFloat64("temperature")

	// System prompt
	rt.SystemPrompt = v.GetString("system_prompt")

	// Output options
	rt.Output = v.GetString("output")
	rt.OutputFile = v.GetString("output_file")
	rt.IncludeArticles = v.GetBool("include_articles")

	return rt
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
	v.BindEnv("base_url", "LLM_AGGREGATOR_BASE_URL")
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
	v.Set("base_url", c.BaseURL)
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
	// Check if config file exists - if it does, values would have been loaded from it
	configPath, err := GetConfigPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(configPath)
	return err == nil
}