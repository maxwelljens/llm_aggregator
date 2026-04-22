package output

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"llm_aggregator/internal/aggregator"
)

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name      string
		format    string
		wantErr   bool
		wantNil   bool
	}{
		{"text format", "text", false, false},
		{"json format", "json", false, false},
		{"markdown format", "markdown", false, false},
		{"uppercase text", "TEXT", false, false},
		{"mixed case json", "Json", false, false},
		{"unsupported format", "xml", true, true},
		{"empty format", "", true, true},
		{"unknown format", "csv", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := NewFormatter(tt.format)

			if tt.wantErr && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if tt.wantNil && formatter != nil {
				t.Error("Expected nil formatter")
			}
			if !tt.wantNil && formatter == nil {
				t.Error("Expected non-nil formatter")
			}
		})
	}
}

func TestFormatDataJSON(t *testing.T) {
	formatter, err := NewFormatter("json")
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	data := map[string]any{
		"title":          "Test Summary",
		"prompt":         "Summarise the news",
		"model":          "deepseek-chat",
		"articles_count": 5,
		"timestamp":      "2024-01-15T10:00:00Z",
		"summary":        "This is a test summary.",
	}

	output, err := formatter.FormatData(data)
	if err != nil {
		t.Errorf("FormatData failed: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]any
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}

	// Verify expected fields
	if parsed["title"] != "Test Summary" {
		t.Errorf("Expected title 'Test Summary', got %v", parsed["title"])
	}
	if parsed["summary"] != "This is a test summary." {
		t.Errorf("Expected summary 'This is a test summary.', got %v", parsed["summary"])
	}
}

func TestFormatDataMarkdown(t *testing.T) {
	formatter, err := NewFormatter("markdown")
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	data := map[string]any{
		"title":          "Markdown Test",
		"prompt":         "Test prompt",
		"model":          "gpt-4",
		"articles_count": 3,
		"summary":        "Markdown summary content.",
	}

	output, err := formatter.FormatData(data)
	if err != nil {
		t.Errorf("FormatData failed: %v", err)
	}

	// Check for markdown structure
	if !strings.Contains(output, "# Markdown Test") {
		t.Error("Expected title in markdown format")
	}
	if !strings.Contains(output, "## Metadata") {
		t.Error("Expected metadata section")
	}
	if !strings.Contains(output, "## Summary") {
		t.Error("Expected summary section")
	}
	if !strings.Contains(output, "Markdown summary content.") {
		t.Error("Expected summary content")
	}
}

func TestFormatDataText(t *testing.T) {
	formatter, err := NewFormatter("text")
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	data := map[string]any{
		"title":          "Text Output Test",
		"prompt":         "Test prompt",
		"model":          "claude-3",
		"articles_count": 2,
		"summary":        "Plain text summary.",
	}

	output, err := formatter.FormatData(data)
	if err != nil {
		t.Errorf("FormatData failed: %v", err)
	}

	// Check for text formatting elements
	if !strings.Contains(output, "====") {
		t.Error("Expected separator lines")
	}
	if !strings.Contains(output, "Text Output Test") {
		t.Error("Expected title in output")
	}
	if !strings.Contains(output, "METADATA") {
		t.Error("Expected metadata header")
	}
	if !strings.Contains(output, "SUMMARY") {
		t.Error("Expected summary header")
	}
}

func TestFormatDataWithArticles(t *testing.T) {
	formatter, err := NewFormatter("text")
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	articles := []map[string]any{
		{
			"title":       "Article One",
			"source_feed": "News Feed",
			"author":      "John Doe",
			"link":        "https://example.com/1",
			"published":   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			"title":       "Article Two",
			"source_feed": "Tech Feed",
			"author":      "Jane Smith",
			"link":        "https://example.com/2",
		},
	}

	data := map[string]any{
		"title":          "With Articles",
		"summary":        "Summary with articles",
		"articles_count": 2,
		"articles":       articles,
	}

	output, err := formatter.FormatData(data)
	if err != nil {
		t.Errorf("FormatData failed: %v", err)
	}

	// Check articles are included
	if !strings.Contains(output, "Article One") {
		t.Error("Expected Article One in output")
	}
	if !strings.Contains(output, "Article Two") {
		t.Error("Expected Article Two in output")
	}
	if !strings.Contains(output, "News Feed") {
		t.Error("Expected feed name in output")
	}
}

func TestFormatDataMarkdownWithArticles(t *testing.T) {
	formatter, err := NewFormatter("markdown")
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	articles := []map[string]any{
		{
			"title":       "MD Article",
			"source_feed": "MD Feed",
			"published":   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			"link":        "https://example.com/md",
		},
	}

	data := map[string]any{
		"title":          "Markdown with Articles",
		"summary":        "Test summary",
		"articles_count": 1,
		"articles":       articles,
	}

	output, err := formatter.FormatData(data)
	if err != nil {
		t.Errorf("FormatData failed: %v", err)
	}

	// Check markdown article format
	if !strings.Contains(output, "### Article 1: MD Article") {
		t.Error("Expected markdown article header")
	}
	if !strings.Contains(output, "**Source**: MD Feed") {
		t.Error("Expected source field in markdown")
	}
	if !strings.Contains(output, "[https://example.com/md](https://example.com/md)") {
		t.Error("Expected link in markdown format")
	}
}

func TestFormatDataDefaultValues(t *testing.T) {
	formatter, err := NewFormatter("text")
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	// Data with missing fields - should use defaults
	data := map[string]any{}

	output, err := formatter.FormatData(data)
	if err != nil {
		t.Errorf("FormatData failed: %v", err)
	}

	// Check default values are used
	if !strings.Contains(output, "LLM Aggregator Summary") {
		t.Error("Expected default title")
	}
	if !strings.Contains(output, "No summary available.") {
		t.Error("Expected default summary message")
	}
}

func TestGetStringHelper(t *testing.T) {
	tests := []struct {
		name         string
		data         map[string]any
		key          string
		defaultValue string
		want         string
	}{
		{"existing string", map[string]any{"key": "value"}, "key", "default", "value"},
		{"missing key", map[string]any{}, "key", "default", "default"},
		{"nil value", map[string]any{"key": nil}, "key", "default", "default"},
		{"wrong type", map[string]any{"key": 123}, "key", "default", "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getString(tt.data, tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetIntHelper(t *testing.T) {
	tests := []struct {
		name         string
		data         map[string]any
		key          string
		defaultValue int
		want         int
	}{
		{"existing int", map[string]any{"count": 42}, "count", 0, 42},
		{"existing float64", map[string]any{"count": float64(42.5)}, "count", 0, 42},
		{"missing key", map[string]any{}, "count", 10, 10},
		{"nil value", map[string]any{"count": nil}, "count", 10, 10},
		{"wrong type", map[string]any{"count": "string"}, "count", 10, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getInt(tt.data, tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getInt() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestCenterText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		width  int
		expect string
	}{
		{"short text", "Hi", 10, "    Hi    "},  // 3 left, 2 right (floored)
		{"exact width", "Hello", 5, "Hello"},
		{"longer than width", "Hello World", 5, "Hello World"},
		{"even padding", "Test", 12, "    Test    "},  // 4 left, 3 right (floored)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := centerText(tt.text, tt.width)
			if got != tt.expect {
				t.Errorf("centerText(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.expect)
			}
		})
	}
}

func TestFormatArticleList(t *testing.T) {
	articles := []*aggregator.Article{
		{
			Title:      "Test Article",
			Link:       "https://example.com/test",
			Content:    "Article content here",
			SourceFeed: "Test Feed",
			Author:     "Test Author",
		},
	}

	t.Run("text format", func(t *testing.T) {
		output, err := FormatArticleList(articles, "text", false)
		if err != nil {
			t.Errorf("FormatArticleList failed: %v", err)
		}
		if !strings.Contains(output, "Test Article") {
			t.Error("Expected article title in output")
		}
	})

	t.Run("markdown format", func(t *testing.T) {
		output, err := FormatArticleList(articles, "markdown", false)
		if err != nil {
			t.Errorf("FormatArticleList failed: %v", err)
		}
		if !strings.Contains(output, "Articles List") {
			t.Error("Expected articles list header")
		}
	})

	t.Run("json format", func(t *testing.T) {
		output, err := FormatArticleList(articles, "json", false)
		if err != nil {
			t.Errorf("FormatArticleList failed: %v", err)
		}
		if !strings.Contains(output, "Test Article") {
			t.Error("Expected article title in JSON")
		}
	})

	t.Run("include content", func(t *testing.T) {
		output, err := FormatArticleList(articles, "text", true)
		if err != nil {
			t.Errorf("FormatArticleList failed: %v", err)
		}
		if !strings.Contains(output, "Article content here") {
			t.Error("Expected content in output when includeContent=true")
		}
	})
}

func TestJSONIndentation(t *testing.T) {
	formatter, err := NewFormatter("json")
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	data := map[string]any{
		"nested": map[string]any{
			"key1": "value1",
			"key2": 42,
		},
		"array": []int{1, 2, 3},
	}

	output, err := formatter.FormatData(data)
	if err != nil {
		t.Errorf("FormatData failed: %v", err)
	}

	// Check for proper indentation (two spaces)
	lines := strings.Split(output, "\n")
	if len(lines) < 3 {
		t.Error("Expected multiple lines in formatted JSON")
	}
	if !strings.HasPrefix(lines[1], "  ") {
		t.Error("Expected two-space indentation")
	}
}