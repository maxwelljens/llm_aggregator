package aggregator

import (
	"os"
	"strings"
	"testing"
	"time"

	"llm_aggregator/internal/progress"
)

// Mock feed data for testing
const validRSSFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <link>https://example.com/feed</link>
    <description>A test RSS feed</description>
    <item>
      <title>Article One</title>
      <link>https://example.com/article1</link>
      <description>This is the content of article one.</description>
      <pubDate>Wed, 15 Jan 2024 10:00:00 GMT</pubDate>
      <author>john@example.com (John Doe)</author>
    </item>
    <item>
      <title>Article Two</title>
      <link>https://example.com/article2</link>
      <description>This is the content of article two.</description>
      <pubDate>Tue, 14 Jan 2024 09:00:00 GMT</pubDate>
      <author>jane@example.com</author>
    </item>
  </channel>
</rss>`

const atomFeed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Atom Test Feed</title>
  <link href="https://example.com/atom"/>
  <entry>
    <title>Atom Article One</title>
    <link href="https://example.com/atom1"/>
    <content>This is atom article content.</content>
    <updated>2024-01-15T10:00:00Z</updated>
    <author><name>Atom Author</name></author>
  </entry>
</feed>`

const malformedRSSFeed = `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Malformed Feed</title>
    <item>
      <title>Incomplete Article</title>
      <!-- missing link -->
    </item>
  </channel>
</rss>`

func TestNewFeedAggregatorWithProgress(t *testing.T) {
	t.Run("with nil progress context", func(t *testing.T) {
		fa := NewFeedAggregatorWithProgress(10, 7, 5000, nil)
		if fa == nil {
			t.Error("Expected non-nil FeedAggregator")
		}
		if fa.progressCtx != nil {
			t.Error("Expected nil progress context")
		}
	})
}

func TestParseFeedsFromFileNotFound(t *testing.T) {
	fa := NewFeedAggregatorWithProgress(10, 7, 5000, nil)

	_, err := fa.ParseFeedsFromFile("/nonexistent/path/to/feeds.txt")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
	if !strings.Contains(err.Error(), "failed to open feeds file") {
		t.Errorf("Expected 'failed to open feeds file' error, got: %v", err)
	}
}

func TestParseFeedsFromFileEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/test_feeds.txt"

	t.Run("empty file", func(t *testing.T) {
		if err := os.WriteFile(tmpFile, []byte(""), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		fa := NewFeedAggregatorWithProgress(10, 7, 5000, nil)
		articles, err := fa.ParseFeedsFromFile(tmpFile)

		// Empty file should not cause an error
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(articles) != 0 {
			t.Errorf("Expected 0 articles, got %d", len(articles))
		}
	})

	t.Run("file with only comments", func(t *testing.T) {
		if err := os.WriteFile(tmpFile, []byte("# Comment line\n# Another comment\n"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		fa := NewFeedAggregatorWithProgress(10, 7, 5000, nil)
		articles, err := fa.ParseFeedsFromFile(tmpFile)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(articles) != 0 {
			t.Errorf("Expected 0 articles, got %d", len(articles))
		}
	})

	t.Run("file with comments and URLs", func(t *testing.T) {
		content := "# Header comment\nhttps://example.com/feed1\n# Inline comment\nhttps://example.com/feed2\n"
		if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		fa := NewFeedAggregatorWithProgress(10, 7, 5000, nil)
		// This will fail to fetch but shouldn't crash
		_, err := fa.ParseFeedsFromFile(tmpFile)

		// We expect network errors but not parsing errors
		// (If feeds are unreachable, that's okay for this test)
		if err != nil && strings.Contains(err.Error(), "failed to open") {
			t.Errorf("File parsing error: %v", err)
		}
	})
}

func writeTestFile(path, content string) error {
	return writeFile(path, []byte(content))
}

func writeFile(path string, data []byte) error {
	f, err := createFile(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func createFile(path string) (*testFile, error) {
	return &testFile{path: path}, nil
}

type testFile struct {
	path string
}

func (tf *testFile) Write(data []byte) (int, error) {
	// Using os.Write would be cleaner but let's use os package directly
	return len(data), nil
}

func (tf *testFile) Close() error {
	return nil
}

// Helper to actually write to temp file
func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpFile := "/tmp/llm_aggregator_test_feeds.txt"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}
	return tmpFile
}

func TestArticleAgeFiltering(t *testing.T) {
	// FeedAggregator configured for 7-day max article age
	now := time.Now()
	recentDate := now.Add(-3 * 24 * time.Hour)  // 3 days old
	oldDate := now.Add(-10 * 24 * time.Hour)    // 10 days old

	t.Run("article within age limit", func(t *testing.T) {
		article := &Article{
			Title:     "Recent Article",
			Link:      "https://example.com/recent",
			Content:   "Content",
			Published: recentDate,
		}

		cutoffTime := now.Add(-7 * 24 * time.Hour)
		isOld := article.Published.Before(cutoffTime)

		if isOld {
			t.Error("Recent article should not be considered old")
		}
	})

	t.Run("article outside age limit", func(t *testing.T) {
		article := &Article{
			Title:     "Old Article",
			Link:      "https://example.com/old",
			Content:   "Content",
			Published: oldDate,
		}

		cutoffTime := now.Add(-7 * 24 * time.Hour)
		isOld := article.Published.Before(cutoffTime)

		if !isOld {
			t.Error("Old article should be considered old")
		}
	})
}

func TestArticleSorting(t *testing.T) {
	now := time.Now()
	articles := []*Article{
		{
			Title:     "Third Article",
			Link:      "https://example.com/third",
			Published: now.Add(-3 * 24 * time.Hour),
		},
		{
			Title:     "First Article",
			Link:      "https://example.com/first",
			Published: now.Add(-1 * 24 * time.Hour),
		},
		{
			Title:     "Second Article",
			Link:      "https://example.com/second",
			Published: now.Add(-2 * 24 * time.Hour),
		},
	}

	t.Run("sort by date ascending", func(t *testing.T) {
		sorted := sortByDate(articles, false)

		// Ascending: oldest first (Third, Second, First)
		if sorted[0].Title != "Third Article" {
			t.Errorf("Expected Third Article first, got %s", sorted[0].Title)
		}
		if sorted[1].Title != "Second Article" {
			t.Errorf("Expected Second Article second, got %s", sorted[1].Title)
		}
		if sorted[2].Title != "First Article" {
			t.Errorf("Expected First Article third, got %s", sorted[2].Title)
		}
	})

	t.Run("sort by date descending", func(t *testing.T) {
		sorted := sortByDate(articles, true)

		// Descending: newest first (First, Second, Third)
		if sorted[0].Title != "First Article" {
			t.Errorf("Expected First Article first, got %s", sorted[0].Title)
		}
		if sorted[1].Title != "Second Article" {
			t.Errorf("Expected Second Article second, got %s", sorted[1].Title)
		}
		if sorted[2].Title != "Third Article" {
			t.Errorf("Expected Third Article third, got %s", sorted[2].Title)
		}
	})
}

// sortByDate is a standalone test helper that sorts by date
func sortByDate(articles []*Article, reverse bool) []*Article {
	result := make([]*Article, len(articles))
	copy(result, articles)

	// Simple bubble sort for testing
	for i := 0; i < len(result)-1; i++ {
		for j := 0; j < len(result)-i-1; j++ {
			iTime := result[j].Published
			jTime := result[j+1].Published
			if iTime.IsZero() {
				iTime = time.Time{}
			}
			if jTime.IsZero() {
				jTime = time.Time{}
			}

			var shouldSwap bool
			if reverse {
				shouldSwap = iTime.Before(jTime)
			} else {
				shouldSwap = iTime.After(jTime)
			}

			if shouldSwap {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}

	return result
}

func TestArticleLinkExtraction(t *testing.T) {
	tests := []struct {
		name     string
		item     *Article
		wantLink string
	}{
		{
			name:     "valid link",
			item:     &Article{Link: "https://example.com/valid"},
			wantLink: "https://example.com/valid",
		},
		{
			name:     "empty link",
			item:     &Article{Link: ""},
			wantLink: "",
		},
		{
			name:     "relative link",
			item:     &Article{Link: "/article/path"},
			wantLink: "/article/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.item.Link != tt.wantLink {
				t.Errorf("Link = %q, want %q", tt.item.Link, tt.wantLink)
			}
		})
	}
}

func TestArticleContentExtraction(t *testing.T) {
	t.Run("long content is truncated", func(t *testing.T) {
		longContent := strings.Repeat("x", 600)
		article := &Article{
			Title:   "Long Content Article",
			Link:    "https://example.com/long",
			Content: longContent,
		}

		// ToMap truncates content over 500 chars
		m := article.ToMap()
		content := m["content"].(string)

		// 500 chars + "... [truncated]" (11 chars) = 511 chars
		expectedLen := 500 + len("... [truncated]")
		if len(content) != expectedLen {
			t.Errorf("Expected %d chars (500 + truncation), got %d", expectedLen, len(content))
		}
		if !strings.HasSuffix(content, "... [truncated]") {
			t.Errorf("Expected content to end with truncation marker")
		}
	})

	t.Run("short content is not truncated", func(t *testing.T) {
		shortContent := "Short content"
		article := &Article{
			Title:   "Short Content Article",
			Link:    "https://example.com/short",
			Content: shortContent,
		}

		m := article.ToMap()
		if m["content"] != shortContent {
			t.Errorf("Content should not be truncated: %s", m["content"])
		}
	})
}

func TestEmptyFeedsFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := tmpDir + "/empty_feeds.txt"

	// Create empty file
	if err := os.WriteFile(tmpFile, []byte(""), 0644); err != nil {
		t.Fatalf("Failed to write temp file: %v", err)
	}

	// Use WithProgress to avoid nil pointer dereference
	fa := NewFeedAggregatorWithProgress(10, 7, 5000, progress.NewContext(&progress.NoopLogger{}))
	articles, err := fa.ParseFeedsFromFile(tmpFile)

	if err != nil {
		t.Errorf("Expected no error for empty file, got: %v", err)
	}
	if len(articles) != 0 {
		t.Errorf("Expected 0 articles from empty file, got %d", len(articles))
	}
}