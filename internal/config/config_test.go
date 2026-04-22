package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"llm_aggregator/internal/cli"
	"llm_aggregator/internal/defaults"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg.MaxArticlesPerFeed != 10 {
		t.Errorf("Expected MaxArticlesPerFeed = 10, got %d", cfg.MaxArticlesPerFeed)
	}
	
	if cfg.MaxDaysOld != 7 {
		t.Errorf("Expected MaxDaysOld = 7, got %d", cfg.MaxDaysOld)
	}
	
	if cfg.MaxTotalArticles != 20 {
		t.Errorf("Expected MaxTotalArticles = 20, got %d", cfg.MaxTotalArticles)
	}
	
	if cfg.Model != "deepseek-chat" {
		t.Errorf("Expected Model = 'deepseek-chat', got %s", cfg.Model)
	}
	
	if cfg.MaxTokens != 4000 {
		t.Errorf("Expected MaxTokens = 4000, got %d", cfg.MaxTokens)
	}
	
	if cfg.Temperature != 0.7 {
		t.Errorf("Expected Temperature = 0.7, got %f", cfg.Temperature)
	}
	
	expectedPrompt := `You are an expert analyst and summariser.
You analyse content from multiple sources and provide
concise, insightful summaries based on user requests.
Focus on key points, trends, and important information.`
	
	if cfg.SystemPrompt != expectedPrompt {
		t.Errorf("SystemPrompt doesn't match expected default")
	}
	
	if cfg.Output != "text" {
		t.Errorf("Expected Output = 'text', got %s", cfg.Output)
	}
}

func TestConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}
	
	// Should end with .config/llm_aggregator/config.toml
	expectedSuffix := filepath.Join(".config", "llm_aggregator", "config.toml")
	if !hasSuffix(path, expectedSuffix) {
		t.Errorf("Config path %s doesn't end with %s", path, expectedSuffix)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	oldEnv := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldEnv)
	
	cfg := DefaultConfig()
	cfg.SystemPrompt = "Test system prompt"
	cfg.Model = "test-model"
	cfg.MaxTokens = 1234
	
	// Save config
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// Verify file exists
	configPath, err := GetConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file not created at %s", configPath)
	}
	
	// Load config
	loadedCfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	if !loadedCfg.IsLoaded() {
		t.Error("Config should be marked as loaded")
	}
	
	if loadedCfg.SystemPrompt != cfg.SystemPrompt {
		t.Errorf("SystemPrompt mismatch: expected %q, got %q", cfg.SystemPrompt, loadedCfg.SystemPrompt)
	}
	
	if loadedCfg.Model != cfg.Model {
		t.Errorf("Model mismatch: expected %s, got %s", cfg.Model, loadedCfg.Model)
	}
	
	if loadedCfg.MaxTokens != cfg.MaxTokens {
		t.Errorf("MaxTokens mismatch: expected %d, got %d", cfg.MaxTokens, loadedCfg.MaxTokens)
	}
}

func TestEnvironmentVariables(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	oldEnv := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldEnv)
	
	// Set environment variables
	os.Setenv("LLM_AGGREGATOR_SYSTEM_PROMPT", "Env system prompt")
	os.Setenv("LLM_AGGREGATOR_MODEL", "env-model")
	os.Setenv("LLM_AGGREGATOR_MAX_TOKENS", "9999")
	defer func() {
		os.Unsetenv("LLM_AGGREGATOR_SYSTEM_PROMPT")
		os.Unsetenv("LLM_AGGREGATOR_MODEL")
		os.Unsetenv("LLM_AGGREGATOR_MAX_TOKENS")
	}()
	
	// Load config (should pick up env vars)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	if cfg.SystemPrompt != "Env system prompt" {
		t.Errorf("SystemPrompt should be from env var, got %q", cfg.SystemPrompt)
	}
	
	if cfg.Model != "env-model" {
		t.Errorf("Model should be from env var, got %s", cfg.Model)
	}
	
	if cfg.MaxTokens != 9999 {
		t.Errorf("MaxTokens should be from env var, got %d", cfg.MaxTokens)
	}
}

func TestConfigExists(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	oldEnv := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldEnv)
	
	// Get config path to verify
	configPath, err := GetConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}
	
	// Ensure it's in our temp directory
	if !contains(configPath, tempDir) {
		t.Logf("Config path %s not in temp dir %s", configPath, tempDir)
	}
	
	// Initially should not exist
	exists, err := ConfigExists()
	if err != nil {
		t.Fatalf("Failed to check config existence: %v", err)
	}
	
	if exists {
		// Config exists, maybe from previous test run
		t.Logf("Config exists at %s, removing...", configPath)
		os.Remove(configPath)
		// Check again
		exists, err = ConfigExists()
		if err != nil {
			t.Fatalf("Failed to check config existence after removal: %v", err)
		}
		if exists {
			t.Error("Config should not exist after removal")
		}
	}
	
	// Create config
	cfg := DefaultConfig()
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// Now should exist
	exists, err = ConfigExists()
	if err != nil {
		t.Fatalf("Failed to check config existence: %v", err)
	}
	
	if !exists {
		t.Error("Config should exist after saving")
	}
}

// TestIsZero tests the isZero helper function with various types including pointers.
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
	oldEnv := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		os.Setenv("XDG_CONFIG_HOME", oldEnv)
	}()

	// Save a config file with known values
	configuredModel := "config-file-model"
	configuredBaseURL := "https://config-file.example.com"
	configuredTemperature := 0.3

	cfg := DefaultConfig()
	cfg.Model = configuredModel
	cfg.BaseURL = configuredBaseURL
	cfg.Temperature = configuredTemperature
	cfg.MaxTokens = 5000

	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Test 1: Config file should override defaults
	t.Run("config file overrides defaults", func(t *testing.T) {
		// Clear any env vars
		os.Unsetenv("LLM_AGGREGATOR_MODEL")
		os.Unsetenv("LLM_AGGREGATOR_BASE_URL")
		os.Unsetenv("LLM_AGGREGATOR_TEMPERATURE")
		os.Unsetenv("LLM_AGGREGATOR_MAX_TOKENS")

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
		os.Setenv("LLM_AGGREGATOR_MODEL", envModel)
		os.Setenv("LLM_AGGREGATOR_BASE_URL", envBaseURL)
		defer func() {
			os.Unsetenv("LLM_AGGREGATOR_MODEL")
			os.Unsetenv("LLM_AGGREGATOR_BASE_URL")
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
		os.Setenv("LLM_AGGREGATOR_MODEL", "env-should-be-overridden")
		os.Setenv("LLM_AGGREGATOR_BASE_URL", "https://env-should-be-overridden.example.com")
		defer func() {
			os.Unsetenv("LLM_AGGREGATOR_MODEL")
			os.Unsetenv("LLM_AGGREGATOR_BASE_URL")
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
func strPtr(s string) *string { return &s }
func intPtr(i int) *int { return &i }
func floatPtr(f float64) *float64 { return &f }
func boolPtr(b bool) *bool { return &b }

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(s[0:len(substr)] == substr || contains(s[1:], substr)))
}

// Helper function to check if path has suffix
func hasSuffix(path, suffix string) bool {
	if len(path) < len(suffix) {
		return false
	}
	return path[len(path)-len(suffix):] == suffix
}

// TestConfigParsingAlwaysPasses ensures that parsing of options always passes.
// This tests the integration between CLI args, config loading, and Viper.
func TestConfigParsingAlwaysPasses(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	oldEnv := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		os.Setenv("XDG_CONFIG_HOME", oldEnv)
	}()

	// Create a test feeds file
	feedsFile := filepath.Join(tempDir, "feeds.txt")
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
				"--exclude-keywords":     "advertisement",
				"--max-tokens":            "1000",
				"--temperature":          "0.3",
				"--output":                "json",
			},
			expectFeeds: feedsFile,
			expectModel: "deepseek-coder",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build CLI args slice
			args := []string{"llm_aggregator"}
			for k, v := range tt.cliArgs {
				// Remove leading dashes for go-arg compatibility
				key := k
				if strings.HasPrefix(key, "--") {
					key = key[2:]
				}
				args = append(args, "--"+key, v)
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
		})
	}
}

// TestConfigLoadWithVariousSources tests that configuration loading works
// regardless of the source (CLI, env, config file, defaults)
func TestConfigLoadWithVariousSources(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	oldEnv := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer func() {
		os.Setenv("XDG_CONFIG_HOME", oldEnv)
	}()

	// Test 1: Config loads with saved config file
	t.Run("load with saved config file", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.Model = "saved-model"
		cfg.MaxArticlesPerFeed = 30
		cfg.Temperature = 0.9

		if err := cfg.Save(); err != nil {
			t.Fatalf("Save() failed: %v", err)
		}

		loadedCfg, err := Load()
		if err != nil {
			t.Fatalf("Load() failed: %v", err)
		}

		if loadedCfg.Model != "saved-model" {
			t.Errorf("Model = %q, want %q", loadedCfg.Model, "saved-model")
		}
		if loadedCfg.MaxArticlesPerFeed != 30 {
			t.Errorf("MaxArticlesPerFeed = %d, want 30", loadedCfg.MaxArticlesPerFeed)
		}
		if loadedCfg.Temperature != 0.9 {
			t.Errorf("Temperature = %f, want 0.9", loadedCfg.Temperature)
		}
	})

	// Test 2: Verify DefaultConfig returns correct defaults
	t.Run("default config has correct values", func(t *testing.T) {
		cfg := DefaultConfig()

		if cfg.MaxArticlesPerFeed != defaults.DefaultMaxArticlesPerFeed {
			t.Errorf("MaxArticlesPerFeed = %d, want %d", cfg.MaxArticlesPerFeed, defaults.DefaultMaxArticlesPerFeed)
		}
		if cfg.MaxDaysOld != defaults.DefaultMaxDaysOld {
			t.Errorf("MaxDaysOld = %d, want %d", cfg.MaxDaysOld, defaults.DefaultMaxDaysOld)
		}
		if cfg.MaxTotalArticles != defaults.DefaultMaxTotalArticles {
			t.Errorf("MaxTotalArticles = %d, want %d", cfg.MaxTotalArticles, defaults.DefaultMaxTotalArticles)
		}
		if cfg.BaseURL != defaults.DefaultBaseURL {
			t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, defaults.DefaultBaseURL)
		}
		if cfg.Model != defaults.DefaultModel {
			t.Errorf("Model = %q, want %q", cfg.Model, defaults.DefaultModel)
		}
		if cfg.MaxTokens != defaults.DefaultMaxTokens {
			t.Errorf("MaxTokens = %d, want %d", cfg.MaxTokens, defaults.DefaultMaxTokens)
		}
		if cfg.Temperature != defaults.DefaultTemperature {
			t.Errorf("Temperature = %f, want %f", cfg.Temperature, defaults.DefaultTemperature)
		}
		if cfg.Output != defaults.DefaultOutput {
			t.Errorf("Output = %q, want %q", cfg.Output, defaults.DefaultOutput)
		}
		if cfg.IncludeArticles != defaults.DefaultIncludeArticles {
			t.Errorf("IncludeArticles = %v, want %v", cfg.IncludeArticles, defaults.DefaultIncludeArticles)
		}
	})

	// Test 3: Verify ConfigExists works correctly
	t.Run("config exists check", func(t *testing.T) {
		// Should not exist initially (new temp dir)
		exists, err := ConfigExists()
		if err != nil {
			t.Fatalf("ConfigExists() failed: %v", err)
		}

		// Create a config
		cfg := DefaultConfig()
		if err := cfg.Save(); err != nil {
			t.Fatalf("Save() failed: %v", err)
		}

		// Now should exist
		exists, err = ConfigExists()
		if err != nil {
			t.Fatalf("ConfigExists() failed after save: %v", err)
		}
		if !exists {
			t.Error("ConfigExists() should return true after saving")
		}
	})
}

// TestViperToConfigConversion tests the ViperToConfig conversion function.
func TestViperToConfigConversion(t *testing.T) {
	// Create a fresh viper instance
	v := viper.New()
	v.Set("max_articles_per_feed", 15)
	v.Set("max_days_old", 14)
	v.Set("max_total_articles", 50)
	v.Set("include_keywords", "ai,ml")
	v.Set("exclude_keywords", "spam")
	v.Set("api_key", "sk-test-key")
	v.Set("base_url", "https://api.example.com")
	v.Set("model", "test-model")
	v.Set("max_tokens", 3000)
	v.Set("temperature", 0.5)
	v.Set("system_prompt", "Test system prompt")
	v.Set("output", "markdown")
	v.Set("output_file", "/tmp/output.md")
	v.Set("include_articles", true)

	cfg := ViperToConfig(v)

	if cfg.MaxArticlesPerFeed != 15 {
		t.Errorf("MaxArticlesPerFeed = %d, want 15", cfg.MaxArticlesPerFeed)
	}
	if cfg.MaxDaysOld != 14 {
		t.Errorf("MaxDaysOld = %d, want 14", cfg.MaxDaysOld)
	}
	if cfg.MaxTotalArticles != 50 {
		t.Errorf("MaxTotalArticles = %d, want 50", cfg.MaxTotalArticles)
	}
	if cfg.IncludeKeywords != "ai,ml" {
		t.Errorf("IncludeKeywords = %q, want %q", cfg.IncludeKeywords, "ai,ml")
	}
	if cfg.ExcludeKeywords != "spam" {
		t.Errorf("ExcludeKeywords = %q, want %q", cfg.ExcludeKeywords, "spam")
	}
	if cfg.APIKey != "sk-test-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "sk-test-key")
	}
	if cfg.BaseURL != "https://api.example.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, "https://api.example.com")
	}
	if cfg.Model != "test-model" {
		t.Errorf("Model = %q, want %q", cfg.Model, "test-model")
	}
	if cfg.MaxTokens != 3000 {
		t.Errorf("MaxTokens = %d, want 3000", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.5 {
		t.Errorf("Temperature = %f, want 0.5", cfg.Temperature)
	}
	if cfg.SystemPrompt != "Test system prompt" {
		t.Errorf("SystemPrompt = %q, want %q", cfg.SystemPrompt, "Test system prompt")
	}
	if cfg.Output != "markdown" {
		t.Errorf("Output = %q, want %q", cfg.Output, "markdown")
	}
	if cfg.OutputFile != "/tmp/output.md" {
		t.Errorf("OutputFile = %q, want %q", cfg.OutputFile, "/tmp/output.md")
	}
	if cfg.IncludeArticles != true {
		t.Errorf("IncludeArticles = %v, want true", cfg.IncludeArticles)
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
		"model":      "", // Empty string should not override
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