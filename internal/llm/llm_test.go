package llm

import (
	"time"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/aggregator"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/defaults"
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
				nil,
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
	client, err := NewLLMClient("test-api-key", "", "", 0, nil, 0)
	if err != nil {
		t.Fatalf("NewLLMClient returned unexpected error: %v", err)
	}

	if client.model != "deepseek-chat" {
		t.Errorf("model = %q, want %q", client.model, "deepseek-chat")
	}
	if client.maxTokens != 4000 {
		t.Errorf("maxTokens = %d, want %d", client.maxTokens, 4000)
	}
	if *client.temperature != 0.7 {
		t.Errorf("temperature = %f, want %f", *client.temperature, 0.7)
	}
	if client.llmTimeout != 300 {
		t.Errorf("llmTimeout = %d, want %d", client.llmTimeout, 300)
	}
}

func TestNewLLMClientRequiresAPIKey(t *testing.T) {
	_, err := NewLLMClient("", "", "", 0, nil, 0)
	if err == nil {
		t.Error("NewLLMClient expected error for missing API key, got nil")
	}
	if !strings.Contains(err.Error(), "API key is required") {
		t.Errorf("error message = %q, want to contain %q", err.Error(), "API key is required")
	}
}

func testArticle(title, content string) *aggregator.Article {
	return &aggregator.Article{
		Title:      title,
		Link:       "https://example.com/" + title,
		Content:    content,
		Author:     "Jane Doe",
		SourceFeed: "Test Feed",
		Published:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}
}

func TestPrepareContext(t *testing.T) {
	client := &LLMClient{}

	t.Run("renders article fields", func(t *testing.T) {
		ctx := client.prepareContext([]*aggregator.Article{testArticle("Alpha", "content here")})
		for _, want := range []string{
			"--- ARTICLE 1 ---",
			"Title: Alpha",
			"Source: Test Feed",
			"Published: 2024-01-15T10:00:00Z",
			"Author: Jane Doe",
			"Link: https://example.com/Alpha",
			"Content: content here",
		} {
			if !strings.Contains(ctx, want) {
				t.Errorf("prepareContext missing %q in:\n%s", want, ctx)
			}
		}
	})

	t.Run("omits empty fields", func(t *testing.T) {
		article := &aggregator.Article{Title: "Bare", Content: "x"}
		ctx := client.prepareContext([]*aggregator.Article{article})
		for _, unwanted := range []string{"Source:", "Published:", "Author:", "Link:"} {
			if strings.Contains(ctx, unwanted) {
				t.Errorf("prepareContext should omit empty %q:\n%s", unwanted, ctx)
			}
		}
	})

	t.Run("empty article list", func(t *testing.T) {
		if ctx := client.prepareContext(nil); ctx != "" {
			t.Errorf("expected empty context, got %q", ctx)
		}
	})
}

func TestCreateMessages(t *testing.T) {
	client := &LLMClient{}
	ctx := client.prepareContext([]*aggregator.Article{testArticle("Alpha", "content")})

	messages := client.createMessages(ctx, "summarise", "")
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	// System message falls back to the default prompt
	sys := messages[0].OfSystem
	if sys == nil {
		t.Fatalf("expected system message, got %+v", messages[0])
	}
	if sys.Content.OfString.Value != defaults.DefaultSystemPrompt {
		t.Errorf("system prompt = %q, want default", sys.Content.OfString.Value)
	}

	user := messages[1].OfUser
	if user == nil {
		t.Fatalf("expected user message, got %+v", messages[1])
	}
	content := user.Content.OfString.Value
	if !strings.Contains(content, "summarise") || !strings.Contains(content, "Alpha") {
		t.Errorf("user message should include prompt and context, got %q", content)
	}
}
