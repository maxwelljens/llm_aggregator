package aggregator

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"llm_aggregator/internal/progress"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

// FeedAggregator aggregates articles from multiple RSS feeds.
type FeedAggregator struct {
	maxArticlesPerFeed int
	maxDaysOld         int
	maxContentLength   int
	client             *http.Client
	userAgent          string
	progressCtx        *progress.Context
}

// NewFeedAggregator creates a new FeedAggregator with the specified options.
func NewFeedAggregator(maxArticlesPerFeed, maxDaysOld, maxContentLength int) *FeedAggregator {
	return NewFeedAggregatorWithProgress(maxArticlesPerFeed, maxDaysOld, maxContentLength, nil)
}

// NewFeedAggregatorWithProgress creates a new FeedAggregator with progress context.
func NewFeedAggregatorWithProgress(maxArticlesPerFeed, maxDaysOld, maxContentLength int, progressCtx *progress.Context) *FeedAggregator {
	return &FeedAggregator{
		maxArticlesPerFeed: maxArticlesPerFeed,
		maxDaysOld:         maxDaysOld,
		maxContentLength:   maxContentLength,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent:   "LLM-Aggregator/0.1.0 (+https://codeberg.org/maxwelljensen/llm-aggregator)",
		progressCtx: progressCtx,
	}
}

// ParseFeedsFromFile parses RSS feeds from a file containing one URL per line.
func (fa *FeedAggregator) ParseFeedsFromFile(filePath string) ([]*Article, error) {
	articles := []*Article{}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open feeds file: %w", err)
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read feeds file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	feedURLs := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			feedURLs = append(feedURLs, line)
		}
	}

	fa.progressCtx.Logf("Found %d feed URLs in %s", len(feedURLs), filePath)

	for i, feedURL := range feedURLs {
		fa.progressCtx.Logf("Processing feed %d/%d: %s", i+1, len(feedURLs), feedURL)
		feedArticles, err := fa.ParseFeed(feedURL)
		if err != nil {
			fa.progressCtx.Warningf("Failed to parse feed %s: %v", feedURL, err)
			continue
		}
		articles = append(articles, feedArticles...)
	}

	return articles, nil
}

// ParseFeed parses a single RSS feed and extracts articles.
func (fa *FeedAggregator) ParseFeed(feedURL string) ([]*Article, error) {
	articles := []*Article{}

	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed URL %s: %w", feedURL, err)
	}

	feedTitle := feed.Title
	if feedTitle == "" {
		feedTitle = feedURL
	}

	fa.progressCtx.Logf("Parsing feed: %s (%d entries)", feedTitle, len(feed.Items))

	var cutoffTime time.Time
	if fa.maxDaysOld > 0 {
		cutoffTime = time.Now().Add(-time.Duration(fa.maxDaysOld) * 24 * time.Hour)
	}

	maxItems := min(fa.maxArticlesPerFeed, len(feed.Items))

	for i, item := range feed.Items[:maxItems] {
		article, err := fa.extractArticle(item, feedTitle, cutoffTime)
		if err != nil {
			fa.progressCtx.Warningf("Failed to extract article %d from %s: %v", i, feedURL, err)
			continue
		}
		if article != nil {
			articles = append(articles, article)
		}
	}

	return articles, nil
}

func (fa *FeedAggregator) extractArticle(item *gofeed.Item, feedTitle string, cutoffTime time.Time) (*Article, error) {
	// Extract metadata
	title := item.Title
	if title == "" {
		title = "Untitled"
	}

	link := item.Link
	if link == "" {
		return nil, nil
	}

	// Parse publication date
	published := fa.parsePublishedDate(item)

	// Check if article is too old
	if !cutoffTime.IsZero() && !published.IsZero() && published.Before(cutoffTime) {
		fa.progressCtx.Debugf("Skipping old article: %s (%s)", title, published.Format("2006-01-02"))
		return nil, nil
	}

	// Extract author
	author := ""
	if item.Author != nil {
		author = item.Author.Name
		if author == "" {
			author = item.Author.Email
		}
	}
	if author == "" && item.DublinCoreExt != nil && len(item.DublinCoreExt.Creator) > 0 {
		author = item.DublinCoreExt.Creator[0]
	}

	// Get content
	content := fa.extractContent(item, link)

	// Truncate content if too long
	if len(content) > fa.maxContentLength {
		content = content[:fa.maxContentLength] + "... [truncated]"
	}

	return &Article{
		Title:      title,
		Link:       link,
		Content:    content,
		Published:  published,
		Author:     author,
		SourceFeed: feedTitle,
	}, nil
}

func (fa *FeedAggregator) parsePublishedDate(item *gofeed.Item) time.Time {
	if item.PublishedParsed != nil {
		return *item.PublishedParsed
	}
	if item.UpdatedParsed != nil {
		return *item.UpdatedParsed
	}
	// Try to parse from string
	if item.Published != "" {
		if t, err := time.Parse(time.RFC1123, item.Published); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC1123Z, item.Published); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC3339, item.Published); err == nil {
			return t
		}
	}
	return time.Time{}
}

func (fa *FeedAggregator) extractContent(item *gofeed.Item, link string) string {
	// Priority: Content -> Description -> Fetch webpage
	content := ""

	// Try to get content from item
	if item.Content != "" {
		content = item.Content
	} else if item.Description != "" {
		content = item.Description
	}

	// If still no content or very short, fetch webpage
	if content == "" || len(content) < 100 {
		fetchedContent, err := fa.fetchWebpageContent(link)
		if err == nil && fetchedContent != "" {
			content = fetchedContent
		}
	}

	return content
}

func (fa *FeedAggregator) fetchWebpageContent(url string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", fa.userAgent)

	resp, err := fa.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return "", err
	}

	// Remove script and style elements
	doc.Find("script, style, nav, footer, header").Remove()

	// Try to find article content
	articleSelectors := []string{
		"article",
		"main",
		".article-content",
		".post-content",
		".entry-content",
		"#content",
		".content",
	}

	for _, selector := range articleSelectors {
		articleElement := doc.Find(selector).First()
		if articleElement.Length() > 0 {
			text := articleElement.Text()
			text = strings.Join(strings.Fields(text), " ") // Normalise whitespace
			if len(text) > 200 {
				return text, nil
			}
		}
	}

	// Fallback: get all paragraphs
	paragraphs := doc.Find("p")
	if paragraphs.Length() > 0 {
		var texts []string
		paragraphs.Each(func(i int, s *goquery.Selection) {
			texts = append(texts, strings.TrimSpace(s.Text()))
		})
		text := strings.Join(texts, " ")
		if len(text) > 200 {
			return text, nil
		}
	}

	// Last resort: get all text
	text := doc.Text()
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 200 {
		return text, nil
	}

	return "", nil
}

