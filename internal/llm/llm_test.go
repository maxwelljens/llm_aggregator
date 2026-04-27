package llm

import (
	"strings"
	"testing"
)

func TestNewLLMClientTimeout(t *testing.T) {
	tests := []struct {
		name          string
		timeout       int
		expectTimeout int
	}{
		{
			name:          "zero timeout uses default 300",
			timeout:       0,
			expectTimeout: 300,
		},
		{
			name:          "custom timeout is stored",
			timeout:       60,
			expectTimeout: 60,
		},
		{
			name:          "large timeout is stored",
			timeout:       600,
			expectTimeout: 600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// API key is required; use a placeholder
			client, err := NewLLMClient(
				"test-api-key",
				"",
				"",
				0,
				0,
				tt.timeout,
			)
			if err != nil {
				t.Fatalf("NewLLMClient returned unexpected error: %v", err)
			}
			if client.llmTimeout != tt.expectTimeout {
				t.Errorf("llmTimeout = %d, want %d", client.llmTimeout, tt.expectTimeout)
			}
		})
	}
}

func TestNewLLMClientDefaults(t *testing.T) {
	// Verify all defaults are applied when only API key is provided
	client, err := NewLLMClient("test-api-key", "", "", 0, 0, 0)
	if err != nil {
		t.Fatalf("NewLLMClient returned unexpected error: %v", err)
	}

	if client.model != "deepseek-chat" {
		t.Errorf("model = %q, want %q", client.model, "deepseek-chat")
	}
	if client.maxTokens != 4000 {
		t.Errorf("maxTokens = %d, want %d", client.maxTokens, 4000)
	}
	if client.temperature != 0.7 {
		t.Errorf("temperature = %f, want %f", client.temperature, 0.7)
	}
	if client.llmTimeout != 300 {
		t.Errorf("llmTimeout = %d, want %d", client.llmTimeout, 300)
	}
}

func TestNewLLMClientRequiresAPIKey(t *testing.T) {
	_, err := NewLLMClient("", "", "", 0, 0, 0)
	if err == nil {
		t.Error("NewLLMClient expected error for missing API key, got nil")
	}
	if !strings.Contains(err.Error(), "API key is required") {
		t.Errorf("error message = %q, want to contain %q", err.Error(), "API key is required")
	}
}
