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

	// Feed input
	Stdin bool `mapstructure:"stdin"`

	// LLM API options
	APIKey      string  `mapstructure:"api_key"`
	BaseURL     string  `mapstructure:"base_url"`
	Model       string  `mapstructure:"model"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
	Timeout     int     `mapstructure:"timeout"`

	// System prompt
	SystemPrompt string `mapstructure:"system_prompt"`

	// Output options
	Output          string `mapstructure:"output"`
	OutputFile      string `mapstructure:"output_file"`
	IncludeArticles bool   `mapstructure:"include_articles"`
	Plain           bool   `mapstructure:"plain"`
}

// GetViper returns the global viper instance with all configuration sources set up.
// The precedence order is: CLI flags > Environment variables > Config file > Defaults.
//
// NOTE: Viper uses a singleton pattern. Subsequent calls to GetViper() return the SAME
// instance with all previously set values. This means defaults are only set once,
// environment vars are bound once, and the config file is read once. Do NOT call this
// expecting a fresh state — if you need a fresh instance for testing, use viper.New() directly.
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
		// Debug: show what config file is being used
		// Try to read config file
		if err := v.ReadInConfig(); err != nil {
			// If config file doesn't exist, that's OK - we'll use defaults + env vars
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				fmt.Fprintln(os.Stderr, style.Warningf("error reading config file: %v", err))
			} else {
			}
		} else {
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
	v.SetDefault("timeout", defaults.DefaultLLMTimeout)
	// System prompt default
	v.SetDefault("system_prompt", defaults.DefaultSystemPrompt)
	// Output defaults
	v.SetDefault("output", defaults.DefaultOutput)
	v.SetDefault("include_articles", defaults.DefaultIncludeArticles)
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
//
// NOTE: This function is critical for CLI argument handling. Pointer types
// (*string, *int, etc.) return true only when nil (not provided), NOT when
// pointing to a zero value. This allows distinguishing between "flag not
// provided" (nil) vs "flag explicitly set to zero" (e.g., --temperature 0
// should override default, but no flag means preserve default).
func isZero(v any) bool {
	// Handle nil interface values first (before type switch)
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case int:
		return val == 0
	case int8:
		return val == 0
	case int16:
		return val == 0
	case int32:
		return val == 0
	case int64:
		return val == 0
	case float32:
		return val == 0.0
	case float64:
		return val == 0.0
	case bool:
		return !val
	case *string:
		return val == nil
	case *int:
		return val == nil
	case *int8:
		return val == nil
	case *int16:
		return val == nil
	case *int32:
		return val == nil
	case *int64:
		return val == nil
	case *float32:
		return val == nil
	case *float64:
		return val == nil
	case *bool:
		return val == nil
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
	rt.LLMTimeout = v.GetInt("timeout")
	// System prompt
	rt.SystemPrompt = v.GetString("system_prompt")
	// Output options
	rt.Output = v.GetString("output")
	rt.OutputFile = v.GetString("output_file")
	rt.IncludeArticles = v.GetBool("include_articles")
	rt.Plain = v.GetBool("plain")
	rt.Stdin = v.GetBool("stdin")
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
	v.BindEnv("timeout", "LLM_AGGREGATOR_TIMEOUT")
	// System prompt
	v.BindEnv("system_prompt", "LLM_AGGREGATOR_SYSTEM_PROMPT")
	// Output options
	v.BindEnv("output", "LLM_AGGREGATOR_OUTPUT")
	v.BindEnv("output_file", "LLM_AGGREGATOR_OUTPUT_FILE")
	v.BindEnv("include_articles", "LLM_AGGREGATOR_INCLUDE_ARTICLES")
	v.BindEnv("plain", "LLM_AGGREGATOR_PLAIN")
	// Feed input
	v.BindEnv("stdin", "LLM_AGGREGATOR_STDIN")
}

// GetConfigPath returns the path to the config file.
func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "llm_aggregator", "config.toml"), nil
}
