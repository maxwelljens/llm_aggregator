package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/cli"
)

func TestIsZero(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		// String tests
		{"empty string is zero", "", true},
		{"non-empty string is not zero", "hello", false},

		// Int tests
		{"zero int is zero", 0, true},
		{"non-zero int is not zero", 42, false},

		// Float tests
		{"zero float is zero", 0.0, true},
		{"non-zero float is not zero", 0.7, false},

		// Bool tests
		{"false bool is zero", false, true},
		{"true bool is not zero", true, false},

		// Pointer tests - the bug we fixed
		{"nil *string is zero", (*string)(nil), true},
		{"non-nil *string is not zero", strPtr("hello"), false},
		{"nil *int is zero", (*int)(nil), true},
		{"non-nil *int is not zero", intPtr(42), false},
		{"nil *float64 is zero", (*float64)(nil), true},
		{"non-nil *float64 is not zero", floatPtr(0.7), false},
		{"nil *bool is zero", (*bool)(nil), true},
		{"non-nil *bool is not zero", boolPtr(true), false},

		// Unknown type
		{"nil interface is zero", nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isZero(tt.value)
			if result != tt.expected {
				t.Errorf("isZero(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

// TestViperToRuntimePrecedence tests that ViperToRuntime correctly respects precedence.
// Order from highest to lowest: CLI args > Environment variables > Config file > Defaults
func TestViperToRuntimePrecedence(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// Save a config file with known values (manually write TOML since DefaultConfig/Save don't exist)
	configuredModel := "config-file-model"
	configuredBaseURL := "https://config-file.example.com"
	configuredTemperature := 0.3
	configuredMaxTokens := 5000

	configContent := fmt.Sprintf(`model = "%s"
base_url = "%s"
temperature = %f
max_tokens = %d
`, configuredModel, configuredBaseURL, configuredTemperature, configuredMaxTokens)

	configPath := filepath.Join(tempDir, "llm_aggregator", "config.toml")
	_ = os.MkdirAll(filepath.Dir(configPath), 0755) //nolint:errcheck
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Test 1: Config file should override defaults
	t.Run("config file overrides defaults", func(t *testing.T) {
		// Clear any env vars
		_ = os.Unsetenv("LLM_AGGREGATOR_MODEL")
		_ = os.Unsetenv("LLM_AGGREGATOR_BASE_URL")
		_ = os.Unsetenv("LLM_AGGREGATOR_TEMPERATURE")
		_ = os.Unsetenv("LLM_AGGREGATOR_MAX_TOKENS")

		v := GetViper()
		rt := ViperToRuntime(v, "/tmp/feeds.txt", "test prompt")

		if rt.Model != configuredModel {
			t.Errorf("Model = %q, want %q (config file should override default)", rt.Model, configuredModel)
		}
		if rt.BaseURL != configuredBaseURL {
			t.Errorf("BaseURL = %q, want %q", rt.BaseURL, configuredBaseURL)
		}
	})

	// Test 2: Environment variables should override config file
	t.Run("environment variables override config file", func(t *testing.T) {
		envModel := "env-override-model"
		envBaseURL := "https://env-override.example.com"
		t.Setenv("LLM_AGGREGATOR_MODEL", envModel)
		t.Setenv("LLM_AGGREGATOR_BASE_URL", envBaseURL)
		defer func() {
			_ = os.Unsetenv("LLM_AGGREGATOR_MODEL")
			_ = os.Unsetenv("LLM_AGGREGATOR_BASE_URL")
		}()

		v := GetViper()
		rt := ViperToRuntime(v, "/tmp/feeds.txt", "test prompt")

		if rt.Model != envModel {
			t.Errorf("Model = %q, want %q (env var should override config file)", rt.Model, envModel)
		}
		if rt.BaseURL != envBaseURL {
			t.Errorf("BaseURL = %q, want %q", rt.BaseURL, envBaseURL)
		}
	})

	// Test 3: CLI args (via BindCLIArgs) should override env vars and config file
	t.Run("CLI args override environment variables and config file", func(t *testing.T) {
		// Create a fresh viper instance to avoid polluting global state
		v := viper.New()
		v.SetDefault("model", "default-model")
		v.SetDefault("base_url", "https://default.example.com")
		v.SetDefault("temperature", 0.7)

		// Set env vars that should be overridden
		t.Setenv("LLM_AGGREGATOR_MODEL", "env-should-be-overridden")
		t.Setenv("LLM_AGGREGATOR_BASE_URL", "https://env-should-be-overridden.example.com")
		defer func() {
			_ = os.Unsetenv("LLM_AGGREGATOR_MODEL")
			_ = os.Unsetenv("LLM_AGGREGATOR_BASE_URL")
		}()

		// Simulate CLI args binding - using pointer types like the actual Args struct
		cliModel := "cli-override-model"
		cliBaseURL := "https://cli-override.example.com"
		cliTemperature := 0.9

		cliArgs := map[string]any{
			"model":       &cliModel,
			"base_url":    &cliBaseURL,
			"temperature": &cliTemperature,
		}
		BindCLIArgs(v, cliArgs)

		if v.GetString("model") != cliModel {
			t.Errorf("model = %q, want %q (CLI should override env var)", v.GetString("model"), cliModel)
		}
		if v.GetString("base_url") != cliBaseURL {
			t.Errorf("base_url = %q, want %q", v.GetString("base_url"), cliBaseURL)
		}
		if v.GetFloat64("temperature") != cliTemperature {
			t.Errorf("temperature = %f, want %f", v.GetFloat64("temperature"), cliTemperature)
		}
	})

	// Test 4: Nil/unset CLI args should NOT override existing values
	// Note: This test verifies that when a CLI arg is nil (not provided), the existing
	// value in viper is preserved. However, since viper doesn't support unsetting keys,
	// and the test uses a fresh viper without BindEnv, we can't properly test the
	// env var persistence here. This is actually a test design limitation.
	t.Run("nil CLI args do not override viper values", func(t *testing.T) {
		// Create a fresh viper instance to avoid polluting global state
		v := viper.New()
		v.SetDefault("model", "default-model")

		// Simulate CLI args where some values were not provided (nil pointers)
		var nilModel *string = nil
		cliArgs := map[string]any{
			"model": nilModel, // Not provided on CLI
		}
		BindCLIArgs(v, cliArgs)

		// Since we didn't call BindEnv or Set any value, model should keep its default
		if v.GetString("model") != "default-model" {
			t.Errorf("model = %q, want %q (nil CLI arg should preserve viper default)", v.GetString("model"), "default-model")
		}
	})

	// Test 5: Empty string CLI args should NOT override existing values
	t.Run("empty string CLI args do not override viper values", func(t *testing.T) {
		// Create a fresh viper instance to avoid polluting global state
		v := viper.New()
		v.SetDefault("model", "default-model")

		// Simulate CLI args where value was provided as empty string
		cliArgs := map[string]any{
			"model": "", // Empty string should not override
		}
		BindCLIArgs(v, cliArgs)

		// Empty string is a "zero" value for string type, so it should not override
		if v.GetString("model") != "default-model" {
			t.Errorf("model = %q, want %q (empty CLI arg should preserve viper default)", v.GetString("model"), "default-model")
		}
	})
}

// TestBindCLIArgsWithPointers tests BindCLIArgs specifically with pointer types.
func TestBindCLIArgsWithPointers(t *testing.T) {
	// Test that pointer types are handled correctly by isZero
	t.Run("BindCLIArgs with pointer types", func(t *testing.T) {
		v := viper.New()
		v.SetDefault("model", "default-model")
		v.SetDefault("max_tokens", 1000)
		v.SetDefault("temperature", 0.7)

		// Simulate CLI args with pointer types (as Args struct now uses)
		model := "cli-model"
		maxTokens := 2000
		temperature := 0.5

		args := map[string]any{
			"model":       &model,
			"max_tokens":  &maxTokens,
			"temperature": &temperature,
		}
		BindCLIArgs(v, args)

		if v.GetString("model") != "cli-model" {
			t.Errorf("model = %q, want %q", v.GetString("model"), "cli-model")
		}
		if v.GetInt("max_tokens") != 2000 {
			t.Errorf("max_tokens = %d, want %d", v.GetInt("max_tokens"), 2000)
		}
		if v.GetFloat64("temperature") != 0.5 {
			t.Errorf("temperature = %f, want %f", v.GetFloat64("temperature"), 0.5)
		}
	})

	t.Run("BindCLIArgs with nil pointers does not override", func(t *testing.T) {
		v := viper.New()
		v.SetDefault("model", "default-model")
		v.SetDefault("max_tokens", 1000)

		// Simulate CLI args where nothing was provided (all nil)
		var nilStr *string = nil
		var nilInt *int = nil

		args := map[string]any{
			"model":      nilStr,
			"max_tokens": nilInt,
		}
		BindCLIArgs(v, args)

		// Should keep defaults
		if v.GetString("model") != "default-model" {
			t.Errorf("model = %q, want %q (nil should not override)", v.GetString("model"), "default-model")
		}
		if v.GetInt("max_tokens") != 1000 {
			t.Errorf("max_tokens = %d, want %d", v.GetInt("max_tokens"), 1000)
		}
	})

	t.Run("BindCLIArgs with zero int pointer DOES override (user explicitly passed --max-tokens 0)", func(t *testing.T) {
		v := viper.New()
		v.SetDefault("max_tokens", 1000)

		zero := 0
		args := map[string]any{
			"max_tokens": &zero, // User explicitly passed --max-tokens 0, so 0 SHOULD override
		}
		BindCLIArgs(v, args)

		if v.GetInt("max_tokens") != 0 {
			t.Errorf("max_tokens = %d, want %d (explicit 0 should override default)", v.GetInt("max_tokens"), 0)
		}
	})

	t.Run("BindCLIArgs with zero float64 pointer DOES override (user explicitly passed --temperature 0)", func(t *testing.T) {
		v := viper.New()
		v.SetDefault("temperature", 0.7)

		zero := 0.0
		args := map[string]any{
			"temperature": &zero, // User explicitly passed --temperature 0, so 0 SHOULD override
		}
		BindCLIArgs(v, args)

		if v.GetFloat64("temperature") != 0.0 {
			t.Errorf("temperature = %f, want %f (explicit 0 should override default)", v.GetFloat64("temperature"), 0.0)
		}
	})
}

// Helper functions for creating pointers
func strPtr(s string) *string     { return &s }
func intPtr(i int) *int           { return &i }
func floatPtr(f float64) *float64 { return &f }
func boolPtr(b bool) *bool        { return &b }

// TestConfigParsingAlwaysPasses ensures that parsing of options always passes.
// This tests the integration between CLI args, config loading, and Viper.
func TestConfigParsingAlwaysPasses(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)

	// Create a test feeds file
	feedsFile := tempDir + "/feeds.txt"
	if err := os.WriteFile(feedsFile, []byte("https://example.com/feed.xml\nhttps://example.org/feed.xml"), 0644); err != nil {
		t.Fatalf("Failed to create test feeds file: %v", err)
	}

	tests := []struct {
		name        string
		cliArgs     map[string]string
		expectFeeds string
		expectModel string
	}{
		{
			name: "default configuration",
			cliArgs: map[string]string{
				"--feeds-file": feedsFile,
				"--prompt":     "Test prompt",
			},
			expectFeeds: feedsFile,
			expectModel: "deepseek-chat",
		},
		{
			name: "custom model via CLI",
			cliArgs: map[string]string{
				"--feeds-file": feedsFile,
				"--prompt":     "Test prompt",
				"--model":      "gpt-4",
			},
			expectFeeds: feedsFile,
			expectModel: "gpt-4",
		},
		{
			name: "custom api key via CLI",
			cliArgs: map[string]string{
				"--feeds-file": feedsFile,
				"--prompt":     "Test prompt",
				"--api-key":    "sk-test-key-12345",
			},
			expectFeeds: feedsFile,
			expectModel: "deepseek-chat",
		},
		{
			name: "all CLI options",
			cliArgs: map[string]string{
				"--feeds-file":            feedsFile,
				"--prompt":                "Summarise tech news",
				"--api-key":               "sk-test-key",
				"--model":                 "deepseek-coder",
				"--max-articles-per-feed": "5",
				"--max-days-old":          "3",
				"--max-total-articles":    "15",
				"--include-keywords":      "ai,ml",
				"--exclude-keywords":      "advertisement",
				"--max-tokens":            "1000",
				"--temperature":           "0.3",
				"--output":                "json",
				"--stdin":                 "",
				"--plain":                 "",
			},
			expectFeeds: feedsFile,
			expectModel: "deepseek-coder",
		},
		{
			name: "stdin only",
			cliArgs: map[string]string{
				"--prompt": "Summarise from stdin",
				"--stdin":  "",
			},
			expectFeeds: "",
			expectModel: "deepseek-chat",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build CLI args slice
			args := []string{"llm_aggregator"}
			for k, v := range tt.cliArgs {
				// Remove leading dashes for go-arg compatibility
				key := strings.TrimPrefix(k, "--")
				args = append(args, "--"+key)
				if v != "" {
					args = append(args, v)
				}
			}

			// Save original os.Args
			origArgs := os.Args
			defer func() { os.Args = origArgs }()
			os.Args = args

			// Parse arguments - this should never fail for valid inputs
			parsedArgs, err := cli.ParseArgs()
			if err != nil {
				t.Fatalf("Failed to parse CLI args: %v", err)
			}

			// Verify feeds file path is set correctly
			if parsedArgs.FeedsFile != tt.expectFeeds {
				t.Errorf("FeedsFile = %q, want %q", parsedArgs.FeedsFile, tt.expectFeeds)
			}

			// Verify prompt is set
			if parsedArgs.Prompt == "" {
				t.Error("Prompt should not be empty")
			}

			// Verify model matches expected
			if parsedArgs.Model != nil && *parsedArgs.Model != tt.expectModel {
				t.Errorf("Model = %q, want %q", *parsedArgs.Model, tt.expectModel)
			}

			// Verify API key if provided
			if apiKey, ok := tt.cliArgs["--api-key"]; ok {
				if parsedArgs.APIKey != nil && *parsedArgs.APIKey != apiKey {
					t.Errorf("APIKey = %q, want %q", *parsedArgs.APIKey, apiKey)
				}
			}

			// Verify plain flag
			if tt.name == "all CLI options" && !parsedArgs.Plain {
				t.Error("Plain should be true when --plain is provided")
			}

			// Verify stdin flag
			if tt.name == "all CLI options" && !parsedArgs.Stdin {
				t.Error("Stdin should be true when --stdin is provided")
			}
			if tt.name == "stdin only" && !parsedArgs.Stdin {
				t.Error("Stdin should be true when --stdin is provided")
			}
		})
	}
}

// TestBindCLIArgs tests the BindCLIArgs function.
func TestBindCLIArgs(t *testing.T) {
	// Create a fresh viper instance with defaults
	v := viper.New()
	v.SetDefault("model", "default-model")
	v.SetDefault("max_tokens", 1000)

	// Bind CLI args with override values
	args := map[string]any{
		"model":      "cli-model",
		"max_tokens": 2000,
	}
	BindCLIArgs(v, args)

	if v.GetString("model") != "cli-model" {
		t.Errorf("model = %q, want %q", v.GetString("model"), "cli-model")
	}
	if v.GetInt("max_tokens") != 2000 {
		t.Errorf("max_tokens = %d, want %d", v.GetInt("max_tokens"), 2000)
	}

	// Test that zero/empty values don't override
	v2 := viper.New()
	v2.SetDefault("model", "default-model")
	v2.SetDefault("temperature", 0.7)

	args2 := map[string]any{
		"model":       "", // Empty string should not override
		"temperature": 0,  // Zero should not override
	}
	BindCLIArgs(v2, args2)

	if v2.GetString("model") != "default-model" {
		t.Errorf("model = %q, want %q (empty should not override)", v2.GetString("model"), "default-model")
	}
	if v2.GetFloat64("temperature") != 0.7 {
		t.Errorf("temperature = %f, want %f (zero should not override)", v2.GetFloat64("temperature"), 0.7)
	}
}
func TestParseKeywords(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single keyword",
			input:    "linux",
			expected: []string{"linux"},
		},
		{
			name:     "multiple keywords",
			input:    "linux,opensource,free-software",
			expected: []string{"linux", "opensource", "free-software"},
		},
		{
			name:     "keywords with spaces",
			input:    "linux, open source, free software",
			expected: []string{"linux", "open source", "free software"},
		},
		{
			name:     "keywords with extra spaces",
			input:    "  linux  ,  open source  ,  free software  ",
			expected: []string{"linux", "open source", "free software"},
		},
		{
			name:     "keywords with empty items",
			input:    "linux,,opensource",
			expected: []string{"linux", "opensource"},
		},
		{
			name:     "keywords with only empty items",
			input:    ",,",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseKeywords(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("parseKeywords(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, kw := range result {
				if kw != tt.expected[i] {
					t.Errorf("parseKeywords(%q)[%d] = %q, want %q", tt.input, i, kw, tt.expected[i])
				}
			}
		})
	}
}
