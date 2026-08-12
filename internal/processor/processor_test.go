package processor

import (
	"strings"
	"testing"
	"time"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/aggregator"
)

func TestNewContentProcessor(t *testing.T) {
	tests := []struct {
		name                 string
		maxTotalArticles     int
		maxContentPerArticle int
		filterKeywords       []string
		excludeKeywords      []string
	}{
		{"standard config", 20, 5000, []string{"tech"}, []string{"spam"}},
		{"zero limits", 0, 0, nil, nil},
		{"empty keywords", 10, 1000, []string{}, []string{}},
		{"no keywords", 15, 3000, nil, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cp := NewContentProcessor(
				tt.maxTotalArticles,
				tt.maxContentPerArticle,
				tt.filterKeywords,
				tt.excludeKeywords,
			)
			if cp == nil {
				t.Error("Expected non-nil ContentProcessor")
			}
		})
	}
}

func TestProcessArticles(t *testing.T) {
	articles := createTestArticles()

	t.Run("process all articles", func(t *testing.T) {
		cp := NewContentProcessor(100, 5000, nil, nil)
		result := cp.ProcessArticles(articles, "date", false)

		if len(result) != len(articles) {
			t.Errorf("Expected %d articles, got %d", len(articles), len(result))
		}
	})

	t.Run("limit articles", func(t *testing.T) {
		cp := NewContentProcessor(3, 5000, nil, nil)
		result := cp.ProcessArticles(articles, "date", false)

		if len(result) != 3 {
			t.Errorf("Expected 3 articles, got %d", len(result))
		}
	})

	t.Run("sort by date ascending", func(t *testing.T) {
		cp := NewContentProcessor(100, 5000, nil, nil)
		result := cp.ProcessArticles(articles, "date", false)

		// Check first article has earliest date
		firstTitle := result[0].Title
		if !strings.Contains(firstTitle, "Old") {
			t.Errorf("Expected first article to be oldest, got %s", firstTitle)
		}
	})

	t.Run("sort by date descending", func(t *testing.T) {
		cp := NewContentProcessor(100, 5000, nil, nil)
		result := cp.ProcessArticles(articles, "date", true)

		// Check first article has latest date
		firstTitle := result[0].Title
		if !strings.Contains(firstTitle, "New") {
			t.Errorf("Expected first article to be newest, got %s", firstTitle)
		}
	})

	t.Run("empty articles slice", func(t *testing.T) {
		cp := NewContentProcessor(10, 5000, nil, nil)
		result := cp.ProcessArticles([]*aggregator.Article{}, "date", false)

		if len(result) != 0 {
			t.Errorf("Expected 0 articles, got %d", len(result))
		}
	})

	t.Run("nil articles slice", func(t *testing.T) {
		cp := NewContentProcessor(10, 5000, nil, nil)
		result := cp.ProcessArticles(nil, "date", false)

		if len(result) != 0 {
			t.Errorf("Expected 0 articles, got %d", len(result))
		}
	})
}

func TestFilterArticlesByIncludeKeywords(t *testing.T) {
	articles := []*aggregator.Article{
		{Title: "Tech Article", Content: "Tech news content"},
		{Title: "Sports Article", Content: "Sports news content"},
		{Title: "AI Technology", Content: "Artificial intelligence content"},
	}

	cp := NewContentProcessor(100, 5000, []string{"tech", "ai"}, nil)
	result := cp.ProcessArticles(articles, "date", false)

	if len(result) != 2 {
		t.Errorf("Expected 2 articles after filtering, got %d", len(result))
	}

	titles := make([]string, len(result))
	for i, a := range result {
		titles[i] = a.Title
	}

	if !containsString(titles, "Tech Article") {
		t.Error("Expected Tech Article to be included")
	}
	if !containsString(titles, "AI Technology") {
		t.Error("Expected AI Technology to be included")
	}
	if containsString(titles, "Sports Article") {
		t.Error("Sports Article should be excluded")
	}
}

func TestFilterArticlesByExcludeKeywords(t *testing.T) {
	articles := []*aggregator.Article{
		{Title: "Interesting Tech", Content: "Tech content"},
		{Title: "Advertisement", Content: "Buy now! Buy now!"},
		{Title: "More Tech", Content: "More tech content"},
	}

	cp := NewContentProcessor(100, 5000, nil, []string{"advertisement", "buy now"})
	result := cp.ProcessArticles(articles, "date", false)

	if len(result) != 2 {
		t.Errorf("Expected 2 articles after filtering, got %d", len(result))
	}

	titles := make([]string, len(result))
	for i, a := range result {
		titles[i] = a.Title
	}

	if containsString(titles, "Advertisement") {
		t.Error("Advertisement should be excluded")
	}
}

func TestFilterArticlesCaseInsensitive(t *testing.T) {
	articles := []*aggregator.Article{
		{Title: "TECH NEWS", Content: "TECH CONTENT"},
		{Title: "Tech News", Content: "Tech Content"},
		{Title: "tech news", Content: "tech content"},
	}

	cp := NewContentProcessor(100, 5000, []string{"tech"}, nil)
	result := cp.ProcessArticles(articles, "date", false)

	if len(result) != 3 {
		t.Errorf("Expected all 3 articles (case insensitive), got %d", len(result))
	}
}

func TestSortArticlesByTitle(t *testing.T) {
	articles := []*aggregator.Article{
		{Title: "Zebra Article", Content: "Content"},
		{Title: "Alpha Article", Content: "Content"},
		{Title: "Middle Article", Content: "Content"},
	}

	t.Run("ascending", func(t *testing.T) {
		cp := NewContentProcessor(100, 5000, nil, nil)
		result := cp.ProcessArticles(articles, "title", false)

		if result[0].Title != "Alpha Article" {
			t.Errorf("Expected Alpha first, got %s", result[0].Title)
		}
		if result[2].Title != "Zebra Article" {
			t.Errorf("Expected Zebra last, got %s", result[2].Title)
		}
	})

	t.Run("descending", func(t *testing.T) {
		cp := NewContentProcessor(100, 5000, nil, nil)
		result := cp.ProcessArticles(articles, "title", true)

		if result[0].Title != "Zebra Article" {
			t.Errorf("Expected Zebra first, got %s", result[0].Title)
		}
	})
}

func TestSortArticlesBySource(t *testing.T) {
	articles := []*aggregator.Article{
		{Title: "A", SourceFeed: "Zulu Feed"},
		{Title: "B", SourceFeed: "Alpha Feed"},
		{Title: "C", SourceFeed: "Middle Feed"},
	}

	t.Run("ascending", func(t *testing.T) {
		cp := NewContentProcessor(100, 5000, nil, nil)
		result := cp.ProcessArticles(articles, "source", false)

		if result[0].SourceFeed != "Alpha Feed" {
			t.Errorf("Expected Alpha Feed first, got %s", result[0].SourceFeed)
		}
	})

	t.Run("descending", func(t *testing.T) {
		cp := NewContentProcessor(100, 5000, nil, nil)
		result := cp.ProcessArticles(articles, "source", true)

		if result[0].SourceFeed != "Zulu Feed" {
			t.Errorf("Expected Zulu Feed first, got %s", result[0].SourceFeed)
		}
	})
}

func TestPrepareForLLM(t *testing.T) {
	now := time.Now()
	article := &aggregator.Article{
		Title:      "Test Title",
		Link:       "https://example.com/test",
		Content:    "This is the article content",
		Author:     "Test Author",
		SourceFeed: "Test Feed",
		Summary:    "Test Summary",
		Published:  now,
	}

	cp := NewContentProcessor(10, 5000, nil, nil)
	result := cp.ProcessArticles([]*aggregator.Article{article}, "date", false)

	if len(result) != 1 {
		t.Fatalf("Expected 1 article, got %d", len(result))
	}

	// Check all expected fields are present
	articleMap := result[0]
	if articleMap.Title != "Test Title" {
		t.Errorf("Expected title 'Test Title', got %v", articleMap.Title)
	}
	if articleMap.Link != "https://example.com/test" {
		t.Errorf("Expected link, got %v", articleMap.Link)
	}
	if articleMap.Author != "Test Author" {
		t.Errorf("Expected author, got %v", articleMap.Author)
	}
	if articleMap.SourceFeed != "Test Feed" {
		t.Errorf("Expected source_feed, got %v", articleMap.SourceFeed)
	}
	if articleMap.Summary != "Test Summary" {
		t.Errorf("Expected summary, got %v", articleMap.Summary)
	}
}

func TestContentTruncation(t *testing.T) {
	article := &aggregator.Article{
		Title:   "Long Article",
		Link:    "https://example.com/long",
		Content: strings.Repeat("x", 6000),
	}

	t.Run("truncate at max content length", func(t *testing.T) {
		cp := NewContentProcessor(10, 1000, nil, nil)
		result := cp.ProcessArticles([]*aggregator.Article{article}, "date", false)

		content := result[0].Content
		if !strings.HasSuffix(content, "... [truncated]") {
			t.Error("Expected content to be truncated")
		}
		if len(content) <= 1000 {
			t.Errorf("Expected truncated content around 1000 chars, got %d", len(content))
		}
	})

	t.Run("no truncation for short content", func(t *testing.T) {
		shortArticle := &aggregator.Article{
			Title:   "Short Article",
			Link:    "https://example.com/short",
			Content: "This is a short article content.",
		}
		cp := NewContentProcessor(10, 5000, nil, nil)
		result := cp.ProcessArticles([]*aggregator.Article{shortArticle}, "date", false)

		content := result[0].Content
		if strings.Contains(content, "[truncated]") {
			t.Error("Short content should not be truncated")
		}
	})
}

func TestSetLogger(t *testing.T) {
	cp := NewContentProcessor(10, 5000, nil, nil)

	t.Run("nil logger is handled", func(t *testing.T) {
		cp.SetLogger(nil)
		// Should not panic
		result := cp.ProcessArticles(nil, "date", false)
		if len(result) != 0 {
			t.Error("Expected empty result")
		}
	})
}

func TestUnknownSortField(t *testing.T) {
	articles := createTestArticles()
	cp := NewContentProcessor(100, 5000, nil, nil)

	// Unknown sort field should default to date sorting
	result := cp.ProcessArticles(articles, "unknown_field", false)

	// Should not panic and return results
	if len(result) == 0 {
		t.Error("Expected some results for unknown sort field")
	}
}

func TestMixedIncludeExclude(t *testing.T) {
	articles := []*aggregator.Article{
		{Title: "Tech Advertisement", Content: "Tech content with ads"},
		{Title: "Tech News", Content: "Interesting tech content"},
		{Title: "Sports Tech", Content: "Tech in sports"},
		{Title: "Advertisement Only", Content: "Just ads"},
	}

	// Include "tech" but exclude "advertisement"
	cp := NewContentProcessor(100, 5000, []string{"tech"}, []string{"advertisement"})
	result := cp.ProcessArticles(articles, "date", false)

	// Should only include "Tech News" and "Sports Tech"
	if len(result) != 2 {
		t.Errorf("Expected 2 articles, got %d", len(result))
	}

	titles := make([]string, len(result))
	for i, a := range result {
		titles[i] = a.Title
	}

	if !containsString(titles, "Tech News") {
		t.Error("Expected Tech News")
	}
	if !containsString(titles, "Sports Tech") {
		t.Error("Expected Sports Tech")
	}
}

func TestEmptyKeywordLists(t *testing.T) {
	articles := createTestArticles()
	cp := NewContentProcessor(100, 5000, []string{}, []string{})

	result := cp.ProcessArticles(articles, "date", false)

	// Should include all articles when no keywords specified
	if len(result) != len(articles) {
		t.Errorf("Expected all %d articles, got %d", len(articles), len(result))
	}
}

// Helper functions

func createTestArticles() []*aggregator.Article {
	now := time.Now()
	return []*aggregator.Article{
		{
			Title:      "New Article",
			Link:       "https://example.com/new",
			Content:    "New content",
			Published:  now,
			SourceFeed: "News Feed",
		},
		{
			Title:      "Middle Article",
			Link:       "https://example.com/middle",
			Content:    "Middle content",
			Published:  now.Add(-24 * time.Hour),
			SourceFeed: "News Feed",
		},
		{
			Title:      "Old Article",
			Link:       "https://example.com/old",
			Content:    "Old content",
			Published:  now.Add(-48 * time.Hour),
			SourceFeed: "News Feed",
		},
	}
}

func containsString(list []string, s string) bool {
	for _, item := range list {
		if item == s {
			return true
		}
	}
	return false
}
