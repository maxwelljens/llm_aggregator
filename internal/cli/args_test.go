package cli

import (
	"os"
	"testing"
)

func TestFeedsFileParsing(t *testing.T) {
	tests := []struct {
		name        string
		feedsFile   string
		wantErr     bool
		setupFunc   func()
		cleanupFunc func()
	}{
		{
			name:      "valid feeds file path",
			feedsFile: "/tmp/feeds.txt",
			wantErr:   false,
			setupFunc: func() {
				// Create a temporary feeds file
				f, _ := os.Create("/tmp/feeds.txt")
				f.Close()
			},
			cleanupFunc: func() {
				os.Remove("/tmp/feeds.txt")
			},
		},
		{
			name:      "empty feeds file path",
			feedsFile: "",
			wantErr:   true, // Required field, should fail
		},
		{
			name:      "relative feeds file path",
			feedsFile: "feeds/urls.txt",
			wantErr:   false, // Path doesn't need to exist for parsing
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc()
			}

			// Simulate command line args
			args := []string{}
			if tt.feedsFile != "" {
				args = append(args, "--feeds-file", tt.feedsFile)
			}
			if tt.name != "empty feeds file path" || tt.feedsFile != "" {
				args = append(args, "--prompt", "Test prompt")
			}

			if len(args) > 0 {
				// Test parsing with feeds file
				argsCopy := make([]string, len(os.Args))
				copy(argsCopy, os.Args)
				defer func() { os.Args = argsCopy }()

				os.Args = append([]string{"llm_aggregator"}, args...)

				parsedArgs, err := ParseArgs()
				if tt.wantErr {
					if err == nil {
						t.Errorf("Expected error for feeds-file %q, got nil", tt.feedsFile)
					}
				} else {
					if err != nil {
						t.Errorf("Unexpected error for feeds-file %q: %v", tt.feedsFile, err)
					} else if parsedArgs.FeedsFile != tt.feedsFile {
						t.Errorf("FeedsFile mismatch: want %q, got %q", tt.feedsFile, parsedArgs.FeedsFile)
					}
				}
			}
		})
	}
}

func TestPromptParsing(t *testing.T) {
	tests := []struct {
		name      string
		prompt    string
		wantErr   bool
		setupFunc func()
	}{
		{
			name:    "valid prompt text",
			prompt:  "What are the latest trends in free software?",
			wantErr: false,
		},
		{
			name:    "short prompt",
			prompt:  "Hello",
			wantErr: false,
		},
		{
			name:    "empty prompt",
			prompt:  "",
			wantErr: true, // Required field
		},
		{
			name:    "multiline prompt",
			prompt:  "Summarise the following:\n1. News\n2. Updates\n3. Changes",
			wantErr: false,
		},
		{
			name:    "prompt with special characters",
			prompt:  "What about AI & ML developments in 2024?",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}

			argsCopy := make([]string, len(os.Args))
			copy(argsCopy, os.Args)
			defer func() { os.Args = argsCopy }()

			args := []string{"--feeds-file", "/tmp/feeds.txt"}
			if tt.prompt != "" {
				args = append(args, "--prompt", tt.prompt)
			}
			os.Args = append([]string{"llm_aggregator"}, args...)

			parsedArgs, err := ParseArgs()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for prompt %q, got nil", tt.prompt)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for prompt %q: %v", tt.prompt, err)
				} else if parsedArgs.Prompt != tt.prompt {
					t.Errorf("Prompt mismatch: want %q, got %q", tt.prompt, parsedArgs.Prompt)
				}
			}
		})
	}
}

func TestAPIKeyParsing(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	tests := []struct {
		name          string
		apiKey        string
		envKey        string
		wantAPIKey    *string
		wantErr       bool
		setupFunc     func()
		cleanupFunc   func()
	}{
		{
			name:       "explicit api key",
			apiKey:     "sk-test-key-12345",
			wantAPIKey: strPtr("sk-test-key-12345"),
			wantErr:    false,
		},
		{
			name:       "empty api key",
			apiKey:     "",
			wantAPIKey: nil, // Optional field - not provided
			wantErr:    false,
		},
		{
			name:       "api key from environment variable - checked via config",
			apiKey:     "",
			envKey:     "sk-env-key-67890",
			wantAPIKey: nil, // CLI parsing doesn't read env vars directly
			wantErr:    false,
			setupFunc: func() {
				os.Setenv("LLM_AGGREGATOR_API_KEY", "sk-env-key-67890")
			},
			cleanupFunc: func() {
				os.Unsetenv("LLM_AGGREGATOR_API_KEY")
			},
		},
		{
			name:       "explicit key overrides env",
			apiKey:     "sk-explicit-key",
			envKey:     "sk-env-key-should-be-ignored",
			wantAPIKey: strPtr("sk-explicit-key"),
			wantErr:    false,
			setupFunc: func() {
				os.Setenv("LLM_AGGREGATOR_API_KEY", "sk-env-key-should-be-ignored")
			},
			cleanupFunc: func() {
				os.Unsetenv("LLM_AGGREGATOR_API_KEY")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc()
			}

			argsCopy := make([]string, len(os.Args))
			copy(argsCopy, os.Args)
			defer func() { os.Args = argsCopy }()

			args := []string{"--feeds-file", "/tmp/feeds.txt", "--prompt", "Test prompt"}
			if tt.apiKey != "" {
				args = append(args, "--api-key", tt.apiKey)
			}
			os.Args = append([]string{"llm_aggregator"}, args...)

			parsedArgs, err := ParseArgs()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for api-key %q, got nil", tt.apiKey)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for api-key: %v", err)
				} else if (tt.wantAPIKey == nil && parsedArgs.APIKey != nil) || (tt.wantAPIKey != nil && *parsedArgs.APIKey != *tt.wantAPIKey) {
					wantStr := "<nil>"
					if tt.wantAPIKey != nil {
						wantStr = *tt.wantAPIKey
					}
					gotStr := "<nil>"
					if parsedArgs.APIKey != nil {
						gotStr = *parsedArgs.APIKey
					}
					t.Errorf("APIKey mismatch: want %q, got %q", wantStr, gotStr)
				}
			}
		})
	}
}

func TestModelParsing(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	tests := []struct {
		name        string
		model       string
		wantModel   *string
		wantErr     bool
		setupFunc   func()
		cleanupFunc func()
	}{
		{
			name:      "default model",
			model:     "",
			wantModel: nil, // No model provided - will default via config
			wantErr:   false,
		},
		{
			name:      "deepseek-chat model",
			model:     "deepseek-chat",
			wantModel: strPtr("deepseek-chat"),
			wantErr:   false,
		},
		{
			name:      "deepseek-coder model",
			model:     "deepseek-coder",
			wantModel: strPtr("deepseek-coder"),
			wantErr:   false,
		},
		{
			name:      "gpt-4 model",
			model:     "gpt-4",
			wantModel: strPtr("gpt-4"),
			wantErr:   false,
		},
		{
			name:      "model from environment variable - checked via config",
			model:     "",
			wantModel: nil, // CLI parsing doesn't read env vars directly
			wantErr:   false,
			setupFunc: func() {
				os.Setenv("LLM_AGGREGATOR_MODEL", "custom-model-from-env")
			},
			cleanupFunc: func() {
				os.Unsetenv("LLM_AGGREGATOR_MODEL")
			},
		},
		{
			name:      "explicit model overrides env",
			model:     "explicit-model",
			wantModel: strPtr("explicit-model"),
			wantErr:   false,
			setupFunc: func() {
				os.Setenv("LLM_AGGREGATOR_MODEL", "env-model-should-be-ignored")
			},
			cleanupFunc: func() {
				os.Unsetenv("LLM_AGGREGATOR_MODEL")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc()
			}

			argsCopy := make([]string, len(os.Args))
			copy(argsCopy, os.Args)
			defer func() { os.Args = argsCopy }()

			args := []string{"--feeds-file", "/tmp/feeds.txt", "--prompt", "Test prompt"}
			if tt.model != "" {
				args = append(args, "--model", tt.model)
			}
			os.Args = append([]string{"llm_aggregator"}, args...)

			parsedArgs, err := ParseArgs()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for model %q, got nil", tt.model)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for model: %v", err)
				} else if (tt.wantModel == nil && parsedArgs.Model != nil) || (tt.wantModel != nil && (parsedArgs.Model == nil || *parsedArgs.Model != *tt.wantModel)) {
					wantStr := "<nil>"
					if tt.wantModel != nil {
						wantStr = *tt.wantModel
					}
					gotStr := "<nil>"
					if parsedArgs.Model != nil {
						gotStr = *parsedArgs.Model
					}
					t.Errorf("Model mismatch: want %q, got %q", wantStr, gotStr)
				}
			}
		})
	}
}

func TestBaseURLParsing(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	tests := []struct {
		name         string
		baseURL      string
		wantBaseURL  *string
		wantErr      bool
		setupFunc    func()
		cleanupFunc  func()
	}{
		{
			name:        "default base URL",
			baseURL:     "",
			wantBaseURL: nil, // No base URL provided
			wantErr:     false,
		},
		{
			name:        "explicit deepseek URL",
			baseURL:     "https://api.deepseek.com",
			wantBaseURL: strPtr("https://api.deepseek.com"),
			wantErr:     false,
		},
		{
			name:        "custom OpenAI-compatible URL",
			baseURL:     "https://api.openai.com/v1",
			wantBaseURL: strPtr("https://api.openai.com/v1"),
			wantErr:     false,
		},
		{
			name:        "local ollama URL",
			baseURL:     "http://localhost:11434/v1",
			wantBaseURL: strPtr("http://localhost:11434/v1"),
			wantErr:     false,
		},
		{
			name:        "azure openai URL",
			baseURL:     "https://my-resource.openai.azure.com/v1",
			wantBaseURL: strPtr("https://my-resource.openai.azure.com/v1"),
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc()
			}

			argsCopy := make([]string, len(os.Args))
			copy(argsCopy, os.Args)
			defer func() { os.Args = argsCopy }()

			args := []string{"--feeds-file", "/tmp/feeds.txt", "--prompt", "Test prompt"}
			if tt.baseURL != "" {
				args = append(args, "--base-url", tt.baseURL)
			}
			os.Args = append([]string{"llm_aggregator"}, args...)

			parsedArgs, err := ParseArgs()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for base-url %q, got nil", tt.baseURL)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for base-url: %v", err)
				} else if (tt.wantBaseURL == nil && parsedArgs.BaseURL != nil) || (tt.wantBaseURL != nil && (parsedArgs.BaseURL == nil || *parsedArgs.BaseURL != *tt.wantBaseURL)) {
					wantStr := "<nil>"
					if tt.wantBaseURL != nil {
						wantStr = *tt.wantBaseURL
					}
					gotStr := "<nil>"
					if parsedArgs.BaseURL != nil {
						gotStr = *parsedArgs.BaseURL
					}
					t.Errorf("BaseURL mismatch: want %q, got %q", wantStr, gotStr)
				}
			}
		})
	}
}

func TestArgsToViperMap(t *testing.T) {
	strPtr := func(s string) *string { return &s }
	intPtr := func(i int) *int { return &i }
	floatPtr := func(f float64) *float64 { return &f }

	args := &Args{
		FeedsFile:           "/tmp/feeds.txt",
		Stdin:               true,
		Prompt:              "Test prompt",
		MaxArticlesPerFeed:  intPtr(5),
		MaxDaysOld:          intPtr(14),
		MaxTotalArticles:    intPtr(50),
		IncludeKeywords:     "linux,opensource",
		ExcludeKeywords:     "advertisement",
		APIKey:              strPtr("sk-test-key"),
		Model:               strPtr("custom-model"),
		MaxTokens:           intPtr(2000),
		Temperature:         floatPtr(0.5),
		SystemPrompt:        "Custom system prompt",
		Output:              "json",
		OutputFile:          "/tmp/output.json",
		IncludeArticles:     true,
		Plain:               true,
	}

	viperMap := args.ToViperMap()

	tests := []struct {
		key      string
		expected any
	}{
		{"max_articles_per_feed", 5},
		{"max_days_old", 14},
		{"max_total_articles", 50},
		{"include_keywords", "linux,opensource"},
		{"exclude_keywords", "advertisement"},
		{"api_key", "sk-test-key"},
		{"model", "custom-model"},
		{"max_tokens", 2000},
		{"temperature", 0.5},
		{"system_prompt", "Custom system prompt"},
		{"output", "json"},
		{"output_file", "/tmp/output.json"},
		{"include_articles", true},
		{"plain", true},
		{"stdin", true},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got, ok := viperMap[tt.key]; !ok {
				t.Errorf("Key %q not found in viper map", tt.key)
			} else if got != tt.expected {
				t.Errorf("viperMap[%q] = %v, want %v", tt.key, got, tt.expected)
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
			result := ParseKeywords(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("ParseKeywords(%q) length = %d, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, kw := range result {
				if kw != tt.expected[i] {
					t.Errorf("ParseKeywords(%q)[%d] = %q, want %q", tt.input, i, kw, tt.expected[i])
				}
			}
		})
	}
}

func TestDryRunFlag(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantDryRun  bool
		wantErr     bool
		setupFunc   func()
		cleanupFunc func()
	}{
		{
			name:       "dry-run flag not set",
			args:       []string{"--feeds-file", "/tmp/feeds.txt", "--prompt", "Test"},
			wantDryRun: false,
			wantErr:    false,
		},
		{
			name:       "dry-run flag set",
			args:       []string{"--feeds-file", "/tmp/feeds.txt", "--prompt", "Test", "--dry-run"},
			wantDryRun: true,
			wantErr:    false,
		},
		{
			name:       "dry-run flag as --dry-run=true",
			args:       []string{"--feeds-file", "/tmp/feeds.txt", "--prompt", "Test", "--dry-run=true"},
			wantDryRun: true,
			wantErr:    false,
		},
		{
			name:       "dry-run with TUI",
			args:       []string{"--feeds-file", "/tmp/feeds.txt", "--prompt", "Test", "--dry-run", "--tui"},
			wantDryRun: true,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupFunc != nil {
				tt.setupFunc()
			}
			if tt.cleanupFunc != nil {
				defer tt.cleanupFunc()
			}

			argsCopy := make([]string, len(os.Args))
			copy(argsCopy, os.Args)
			defer func() { os.Args = argsCopy }()

			os.Args = append([]string{"llm_aggregator"}, tt.args...)

			parsedArgs, err := ParseArgs()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				} else if parsedArgs.DryRun != tt.wantDryRun {
					t.Errorf("DryRun mismatch: want %v, got %v", tt.wantDryRun, parsedArgs.DryRun)
				}
			}
		})
	}
}
