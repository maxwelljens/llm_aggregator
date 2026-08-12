package aggregator

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestArticleJSON verifies the Article JSON shape used for the "json" output format.
func TestArticleJSON(t *testing.T) {
	published := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)
	article := &Article{
		Title:      "Test Article Title",
		Link:       "https://example.com/article",
		Content:    "This is the article content",
		Published:  published,
		Author:     "John Doe",
		SourceFeed: "Test Feed",
		Summary:    "This is a summary",
	}

	bytes, err := json.Marshal(article)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	raw := string(bytes)
	for _, want := range []string{
		`"title":"Test Article Title"`,
		`"link":"https://example.com/article"`,
		`"content":"This is the article content"`,
		`"author":"John Doe"`,
		`"source_feed":"Test Feed"`,
		`"summary":"This is a summary"`,
		published.Format(time.RFC3339),
	} {
		if !strings.Contains(raw, want) {
			t.Errorf("JSON output missing %s: %s", want, raw)
		}
	}
}

// TestArticleOmitEmpty verifies empty optional fields are omitted from JSON.
func TestArticleOmitEmpty(t *testing.T) {
	article := &Article{
		Title: "Minimal Article",
		Link:  "https://example.com/minimal",
	}

	bytes, err := json.Marshal(article)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	raw := string(bytes)
	if strings.Contains(raw, `"author"`) {
		t.Errorf("Empty author should be omitted: %s", raw)
	}
	if strings.Contains(raw, `"source_feed"`) {
		t.Errorf("Empty source_feed should be omitted: %s", raw)
	}
	if strings.Contains(raw, `"summary"`) {
		t.Errorf("Empty summary should be omitted: %s", raw)
	}
}
