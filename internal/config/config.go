package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/cli"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/defaults"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/progress"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/runtime"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/style"
)

// setting is one configurable knob: it registers its own default and
// environment variable, applies the CLI override, and loads itself into the
// Runtime. Adding a flag means adding one entry to viperSettings; defaults,
// env binding, CLI precedence and runtime assembly all follow from it.
type setting interface {
	bind(*viper.Viper)
	setCLI(*viper.Viper, *cli.Args)
	load(*viper.Viper, *runtime.Runtime)
}

type settingOf[T any] struct {
	key     string
	env     string // environment variable name; "" = not env-configurable
	def     T
	read    func(*viper.Viper, string) T
	apply   func(*runtime.Runtime, T)
	fromCLI func(*cli.Args) (T, bool) // nil = not CLI-settable
}

func (s settingOf[T]) bind(v *viper.Viper) {
	v.SetDefault(s.key, s.def)
	if s.env != "" {
		_ = v.BindEnv(s.key, s.env) //nolint:errcheck // viper.BindEnv always returns nil in practice
	}
}

func (s settingOf[T]) setCLI(v *viper.Viper, a *cli.Args) {
	if s.fromCLI == nil {
		return
	}
	if val, provided := s.fromCLI(a); provided {
		v.Set(s.key, val)
	}
}

func (s settingOf[T]) load(v *viper.Viper, rt *runtime.Runtime) {
	s.apply(rt, s.read(v, s.key))
}

// envName derives the LLM_AGGREGATOR_* variable name for a viper key.
func envName(key string) string {
	return "LLM_AGGREGATOR_" + strings.ToUpper(key)
}

// viperSettings is the single source of truth for every configurable value.
func viperSettings() []setting {
	str := func(key, def string, apply func(*runtime.Runtime, string), fromCLI func(*cli.Args) (string, bool)) setting {
		return settingOf[string]{key: key, env: envName(key), def: def, read: (*viper.Viper).GetString, apply: apply, fromCLI: fromCLI}
	}
	intS := func(key string, def int, apply func(*runtime.Runtime, int), fromCLI func(*cli.Args) (int, bool)) setting {
		return settingOf[int]{key: key, env: envName(key), def: def, read: (*viper.Viper).GetInt, apply: apply, fromCLI: fromCLI}
	}
	boolean := func(key string, def bool, apply func(*runtime.Runtime, bool), fromCLI func(*cli.Args) (bool, bool)) setting {
		return settingOf[bool]{key: key, env: envName(key), def: def, read: (*viper.Viper).GetBool, apply: apply, fromCLI: fromCLI}
	}
	providedStr := func(get func(*cli.Args) string) func(*cli.Args) (string, bool) {
		return func(a *cli.Args) (string, bool) {
			v := get(a)
			return v, v != ""
		}
	}

	return []setting{
		// Feed input
		str("feeds_file", "", func(rt *runtime.Runtime, s string) { rt.FeedsFile = s },
			providedStr(func(a *cli.Args) string { return a.FeedsFile })),
		str("prompt", defaults.DefaultPrompt, func(rt *runtime.Runtime, s string) { rt.Prompt = s },
			providedStr(func(a *cli.Args) string { return a.Prompt })),
		boolean("stdin", false, func(rt *runtime.Runtime, b bool) { rt.Stdin = b },
			func(a *cli.Args) (bool, bool) { return a.Stdin, a.Stdin }),

		// Feed aggregation
		intS("max_articles_per_feed", defaults.DefaultMaxArticlesPerFeed, func(rt *runtime.Runtime, i int) { rt.MaxArticlesPerFeed = i },
			func(a *cli.Args) (int, bool) { return derefInt(a.MaxArticlesPerFeed) }),
		intS("max_days_old", defaults.DefaultMaxDaysOld, func(rt *runtime.Runtime, i int) { rt.MaxDaysOld = i },
			func(a *cli.Args) (int, bool) { return derefInt(a.MaxDaysOld) }),
		intS("max_total_articles", defaults.DefaultMaxTotalArticles, func(rt *runtime.Runtime, i int) { rt.MaxTotalArticles = i },
			func(a *cli.Args) (int, bool) { return derefInt(a.MaxTotalArticles) }),

		// Content filtering
		str("include_keywords", "", func(rt *runtime.Runtime, s string) {
			if s != "" {
				rt.IncludeKeywords = parseKeywords(s)
			}
		}, providedStr(func(a *cli.Args) string { return a.IncludeKeywords })),
		str("exclude_keywords", "", func(rt *runtime.Runtime, s string) {
			if s != "" {
				rt.ExcludeKeywords = parseKeywords(s)
			}
		}, providedStr(func(a *cli.Args) string { return a.ExcludeKeywords })),

		// LLM API
		str("api_key", "", func(rt *runtime.Runtime, s string) { rt.APIKey = s },
			func(a *cli.Args) (string, bool) { return derefStr(a.APIKey) }),
		str("base_url", defaults.DefaultBaseURL, func(rt *runtime.Runtime, s string) { rt.BaseURL = s },
			func(a *cli.Args) (string, bool) { return derefStr(a.BaseURL) }),
		str("model", defaults.DefaultModel, func(rt *runtime.Runtime, s string) { rt.Model = s },
			func(a *cli.Args) (string, bool) { return derefStr(a.Model) }),
		intS("max_tokens", defaults.DefaultMaxTokens, func(rt *runtime.Runtime, i int) { rt.MaxTokens = i },
			func(a *cli.Args) (int, bool) { return derefInt(a.MaxTokens) }),
		settingOf[float64]{
			key: "temperature", env: envName("temperature"), def: defaults.DefaultTemperature,
			read:  (*viper.Viper).GetFloat64,
			apply: func(rt *runtime.Runtime, f float64) { t := f; rt.Temperature = &t },
			fromCLI: func(a *cli.Args) (float64, bool) {
				if a.Temperature == nil {
					return 0, false
				}
				return *a.Temperature, true
			},
		},
		intS("timeout", defaults.DefaultLLMTimeout, func(rt *runtime.Runtime, i int) { rt.LLMTimeout = i },
			func(a *cli.Args) (int, bool) { return derefInt(a.Timeout) }),
		str("system_prompt", defaults.DefaultSystemPrompt, func(rt *runtime.Runtime, s string) { rt.SystemPrompt = s },
			providedStr(func(a *cli.Args) string { return a.SystemPrompt })),

		// Output
		str("output", defaults.DefaultOutput, func(rt *runtime.Runtime, s string) { rt.Output = s },
			providedStr(func(a *cli.Args) string { return a.Output })),
		str("output_file", "", func(rt *runtime.Runtime, s string) { rt.OutputFile = s },
			providedStr(func(a *cli.Args) string { return a.OutputFile })),
		boolean("include_articles", defaults.DefaultIncludeArticles, func(rt *runtime.Runtime, b bool) { rt.IncludeArticles = b },
			func(a *cli.Args) (bool, bool) { return a.IncludeArticles, a.IncludeArticles }),
		boolean("plain", false, func(rt *runtime.Runtime, b bool) { rt.Plain = b },
			func(a *cli.Args) (bool, bool) { return a.Plain, a.Plain }),
	}
}

func derefStr(p *string) (string, bool) {
	if p == nil {
		return "", false
	}
	return *p, true
}

func derefInt(p *int) (int, bool) {
	if p == nil {
		return 0, false
	}
	return *p, true
}

// GetViper returns the global viper instance.
// Precedence (highest first): CLI flags → environment variables → config file → defaults.
//
// NOTE: Viper is a singleton. Subsequent calls return the SAME instance with all
// previously set values. Defaults, env bindings, and config-file reads are idempotent
// but the instance is NOT reset between calls. Use viper.New() directly for testing.
func GetViper() *viper.Viper {
	v := viper.GetViper()
	BindSettings(v)
	// Set environment variable prefix
	v.SetEnvPrefix("LLM_AGGREGATOR")
	v.AutomaticEnv()
	// Get config path from XDG
	configPath, err := GetConfigPath()
	if err == nil {
		// Set config file path
		v.SetConfigFile(configPath)
		// Try to read config file
		if err := v.ReadInConfig(); err != nil {
			// If config file doesn't exist, that's OK - we'll use defaults + env vars
			var notFoundError viper.ConfigFileNotFoundError
			if !errors.As(err, &notFoundError) {
				fmt.Fprintln(os.Stderr, style.Warningf("error reading config file: %v", err))
			}
		}
	}
	return v
}

// BindSettings registers defaults and environment bindings on a viper
// instance. GetViper calls it for production; tests call it on fresh
// instances.
func BindSettings(v *viper.Viper) {
	for _, s := range viperSettings() {
		s.bind(v)
	}
}

// ViperToRuntime assembles a Runtime from viper configuration and CLI args.
// CLI overrides are applied first so they take precedence over environment
// variables, config file, and defaults. FeedsFile and Prompt come from the
// positional CLI args.
func ViperToRuntime(v *viper.Viper, args *cli.Args) *runtime.Runtime {
	for _, s := range viperSettings() {
		s.setCLI(v, args)
	}

	rt := &runtime.Runtime{
		Progress: &progress.NoopLogger{},
	}
	for _, s := range viperSettings() {
		s.load(v, rt)
	}
	return rt
}

// parseKeywords splits a comma-separated keyword string into a trimmed list.
// Empty tokens resulting from malformed input are discarded.
func parseKeywords(keywordString string) []string {
	if keywordString == "" {
		return nil
	}
	keywords := strings.Split(keywordString, ",")
	result := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		if trimmed := strings.TrimSpace(kw); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// GetConfigPath returns the path to the config file.
func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(configDir, "llm_aggregator", "config.toml"), nil
}
