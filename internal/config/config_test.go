package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/cli"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/defaults"
)

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
		rt := ViperToRuntime(v, &cli.Args{FeedsFile: "/tmp/feeds.txt", Prompt: "test prompt"})

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
		rt := ViperToRuntime(v, &cli.Args{FeedsFile: "/tmp/feeds.txt", Prompt: "test prompt"})

		if rt.Model != envModel {
			t.Errorf("Model = %q, want %q (env var should override config file)", rt.Model, envModel)
		}
		if rt.BaseURL != envBaseURL {
			t.Errorf("BaseURL = %q, want %q", rt.BaseURL, envBaseURL)
		}
	})

}

// Helper functions for creating pointers
func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

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
			parsedArgs, _, err := cli.ParseArgs()
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

// newTestViper returns a fresh viper instance with all settings bound
// (defaults + env bindings), mirroring what GetViper does for production.
func newTestViper() *viper.Viper {
	v := viper.New()
	BindSettings(v)
	return v
}

func TestViperToRuntimeFromArgs(t *testing.T) {
	t.Run("nil flags keep defaults", func(t *testing.T) {
		rt := ViperToRuntime(newTestViper(), &cli.Args{})
		if rt.MaxTokens != defaults.DefaultMaxTokens {
			t.Errorf("MaxTokens = %d, want default %d", rt.MaxTokens, defaults.DefaultMaxTokens)
		}
		if rt.Temperature == nil || *rt.Temperature != defaults.DefaultTemperature {
			t.Errorf("Temperature = %v, want default", rt.Temperature)
		}
		if rt.Output != defaults.DefaultOutput {
			t.Errorf("Output = %q, want default", rt.Output)
		}
	})

	t.Run("explicit flag overrides default", func(t *testing.T) {
		rt := ViperToRuntime(newTestViper(), &cli.Args{MaxTokens: intPtr(42)})
		if rt.MaxTokens != 42 {
			t.Errorf("MaxTokens = %d, want 42", rt.MaxTokens)
		}
	})

	t.Run("explicit zero temperature overrides default", func(t *testing.T) {
		zero := 0.0
		rt := ViperToRuntime(newTestViper(), &cli.Args{Temperature: &zero})
		if rt.Temperature == nil || *rt.Temperature != 0.0 {
			t.Errorf("Temperature = %v, want 0.0", rt.Temperature)
		}
	})

	t.Run("positional args reach the runtime", func(t *testing.T) {
		rt := ViperToRuntime(newTestViper(), &cli.Args{FeedsFile: "feeds.txt", Prompt: "sum this"})
		if rt.FeedsFile != "feeds.txt" || rt.Prompt != "sum this" {
			t.Errorf("FeedsFile = %q, Prompt = %q", rt.FeedsFile, rt.Prompt)
		}
	})

	t.Run("keywords are parsed from the comma string", func(t *testing.T) {
		rt := ViperToRuntime(newTestViper(), &cli.Args{IncludeKeywords: "go, rust"})
		if len(rt.IncludeKeywords) != 2 || rt.IncludeKeywords[0] != "go" || rt.IncludeKeywords[1] != "rust" {
			t.Errorf("IncludeKeywords = %v", rt.IncludeKeywords)
		}
	})

	t.Run("environment variable is honoured", func(t *testing.T) {
		t.Setenv("LLM_AGGREGATOR_MAX_TOKENS", "99")
		rt := ViperToRuntime(newTestViper(), &cli.Args{})
		if rt.MaxTokens != 99 {
			t.Errorf("MaxTokens = %d, want 99 from env", rt.MaxTokens)
		}
	})

	t.Run("explicit flag overrides environment variable", func(t *testing.T) {
		t.Setenv("LLM_AGGREGATOR_MAX_TOKENS", "99")
		rt := ViperToRuntime(newTestViper(), &cli.Args{MaxTokens: intPtr(5)})
		if rt.MaxTokens != 5 {
			t.Errorf("MaxTokens = %d, want 5 from CLI", rt.MaxTokens)
		}
	})
}
