package aggregator

import (
	"strings"
	"testing"
	"time"
)

func TestArticleToMap(t *testing.T) {
	tests := []struct {
		name     string
		article  *Article
		checkFn  func(map[string]any) bool
	}{
		{
			name: "full article with all fields",
			article: &Article{
				Title:      "Test Article Title",
				Link:       "https://example.com/article",
				Content:    "This is the article content",
				Published:  time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
				Author:     "John Doe",
				SourceFeed: "Test Feed",
				Summary:    "This is a summary",
			},
			checkFn: func(m map[string]any) bool {
				return m["title"] == "Test Article Title" &&
					m["link"] == "https://example.com/article" &&
					m["author"] == "John Doe" &&
					m["source_feed"] == "Test Feed" &&
					m["summary"] == "This is a summary" &&
					strings.Contains(m["content"].(string), "article content")
			},
		},
		{
			name: "article with truncated content",
			article: &Article{
				Title:   "Long Article",
				Link:    "https://example.com/long",
				Content: strings.Repeat("x", 600), // More than 500 chars
			},
			checkFn: func(m map[string]any) bool {
				content := m["content"].(string)
				// 600 chars > 500, so truncated to 500 + "... [truncated]" (15 chars) = 515
				return len(content) == 515 &&
					strings.HasSuffix(content, "... [truncated]")
			},
		},
		{
			name: "article with zero time",
			article: &Article{
				Title:     "No Date Article",
				Link:      "https://example.com/nodate",
				Content:   "Content",
				Published: time.Time{}, // Zero time
			},
			checkFn: func(m map[string]any) bool {
				_, hasPublished := m["published"]
				return !hasPublished
			},
		},
		{
			name: "article without author",
			article: &Article{
				Title:     "Anonymous Article",
				Link:      "https://example.com/anon",
				Content:   "Content",
				Author:    "",
				SourceFeed: "Feed",
			},
			checkFn: func(m map[string]any) bool {
				return m["author"] == ""
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.article.ToMap()
			if !tt.checkFn(result) {
				t.Errorf("ToMap() produced unexpected result: %v", result)
			}
		})
	}
}

func TestArticleStruct(t *testing.T) {
	t.Run("empty article", func(t *testing.T) {
		article := &Article{}
		m := article.ToMap()

		if m["title"] != "" {
			t.Errorf("Expected empty title, got %q", m["title"])
		}
		if m["link"] != "" {
			t.Errorf("Expected empty link, got %q", m["link"])
		}
	})

	t.Run("article with only title and link", func(t *testing.T) {
		article := &Article{
			Title: "Minimal Article",
			Link:  "https://example.com/minimal",
		}
		m := article.ToMap()

		if m["title"] != "Minimal Article" {
			t.Errorf("Expected title 'Minimal Article', got %q", m["title"])
		}
		if m["link"] != "https://example.com/minimal" {
			t.Errorf("Expected link 'https://example.com/minimal', got %q", m["link"])
		}
	})

	t.Run("article with future date", func(t *testing.T) {
		futureDate := time.Now().Add(24 * time.Hour)
		article := &Article{
			Title:     "Future Article",
			Link:      "https://example.com/future",
			Content:   "Content",
			Published: futureDate,
		}
		m := article.ToMap()

		if published, ok := m["published"].(string); !ok {
			t.Error("Expected published field to be set")
		} else if !strings.Contains(published, futureDate.Format("2006-01-02")) {
			t.Errorf("Expected published date %s, got %s", futureDate.Format(time.RFC3339), published)
		}
	})
}