package output

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/aggregator"
)

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		wantErr bool
		wantNil bool
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

func testData() Data {
	return Data{
		Title:         "Test Summary",
		Prompt:        "Summarise the news",
		Model:         "deepseek-chat",
		ArticlesCount: 5,
		Timestamp:     "2024-01-15T10:00:00Z",
		Summary:       "This is a test summary.",
	}
}

func TestFormatDataJSON(t *testing.T) {
	formatter, err := NewFormatter("json")
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	output, err := formatter.FormatData(testData())
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
	if parsed["articles_count"] != float64(5) {
		t.Errorf("Expected articles_count 5, got %v", parsed["articles_count"])
	}
}

func TestFormatDataMarkdown(t *testing.T) {
	formatter, err := NewFormatter("markdown")
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	data := testData()
	data.Title = "Markdown Test"
	data.Model = "gpt-4"
	data.Summary = "Markdown summary content."

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

	data := testData()
	data.Title = "Text Output Test"
	data.Model = "claude-3"
	data.Summary = "Plain text summary."

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

	articles := []*aggregator.Article{
		{
			Title:      "Article One",
			SourceFeed: "News Feed",
			Author:     "John Doe",
			Link:       "https://example.com/1",
			Published:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			Title:      "Article Two",
			SourceFeed: "Tech Feed",
			Author:     "Jane Smith",
			Link:       "https://example.com/2",
		},
	}

	data := testData()
	data.Title = "With Articles"
	data.Summary = "Summary with articles"
	data.ArticlesCount = 2
	data.Articles = articles

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

	articles := []*aggregator.Article{
		{
			Title:      "MD Article",
			SourceFeed: "MD Feed",
			Published:  time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			Link:       "https://example.com/md",
		},
	}

	data := testData()
	data.Title = "Markdown with Articles"
	data.ArticlesCount = 1
	data.Articles = articles

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
	output, err := formatter.FormatData(Data{})
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

func TestCenterText(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		width  int
		expect string
	}{
		{"short text", "Hi", 10, "    Hi    "}, // 3 left, 2 right (floored)
		{"exact width", "Hello", 5, "Hello"},
		{"longer than width", "Hello World", 5, "Hello World"},
		{"even padding", "Test", 12, "    Test    "}, // 4 left, 3 right (floored)
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

func TestJSONIndentation(t *testing.T) {
	formatter, err := NewFormatter("json")
	if err != nil {
		t.Fatalf("Failed to create formatter: %v", err)
	}

	data := testData()
	data.Articles = []*aggregator.Article{
		{Title: "A", Link: "https://example.com/a"},
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
