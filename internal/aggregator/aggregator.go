package aggregator

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"llm_aggregator/internal/progress"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
	"golang.org/x/sync/errgroup"
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
// Feeds are fetched concurrently for improved performance.
func (fa *FeedAggregator) ParseFeedsFromFile(filePath string) ([]*Article, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open feeds file: %w", err)
	}
	defer file.Close() //nolint:errcheck

	return fa.parseFeedsFromReader(file, filePath)
}

// ParseFeedFromStdin parses a single RSS/Atom feed from stdin.
func (fa *FeedAggregator) ParseFeedFromStdin() ([]*Article, error) {
	return fa.ParseFeedFromReader(os.Stdin, "stdin")
}

// ParseFeedFromReader parses a single RSS/Atom feed from an io.Reader.
// The sourceName is used for progress/logging and to identify the feed source.
func (fa *FeedAggregator) ParseFeedFromReader(reader io.Reader, sourceName string) ([]*Article, error) {
	fp := gofeed.NewParser()
	feed, err := fp.Parse(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed from %s: %w", sourceName, err)
	}

	feedTitle := feed.Title
	if feedTitle == "" {
		feedTitle = sourceName
	}

	if fa.progressCtx != nil {
		fa.progressCtx.Logf("Parsing feed: %s (%d entries)", feedTitle, len(feed.Items))
	}

	var cutoffTime time.Time
	if fa.maxDaysOld > 0 {
		cutoffTime = time.Now().Add(-time.Duration(fa.maxDaysOld) * 24 * time.Hour)
	}

	maxItems := min(fa.maxArticlesPerFeed, len(feed.Items))
	articles := make([]*Article, 0, maxItems)

	for i := range feed.Items[:maxItems] {
		article := fa.extractArticle(feed.Items[i], feedTitle, cutoffTime)
		if article != nil {
			articles = append(articles, article)
		}
	}

	return articles, nil
}

// parseFeedsFromReader reads feed URLs from an io.Reader and fetches each one.
// Used by both file-based and stdin-based feed loading.
func (fa *FeedAggregator) parseFeedsFromReader(reader io.Reader, sourceName string) ([]*Article, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read feeds from %s: %w", sourceName, err)
	}

	lines := strings.Split(string(content), "\n")
	feedURLs := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			feedURLs = append(feedURLs, line)
		}
	}

	if fa.progressCtx != nil {
		fa.progressCtx.Logf("Found %d feed URLs in %s", len(feedURLs), sourceName)
	}

	// Use mutex to protect shared state
	var mu sync.Mutex
	var allArticles []*Article
	var feedErrors []string

	// Limit concurrency to avoid overwhelming servers
	// NOTE: maxConcurrency is hardcoded to 10. Changing this requires a code
	// modification.
	const maxConcurrency = 10
	sem := make(chan struct{}, maxConcurrency)

	g, _ := errgroup.WithContext(context.Background())
	for i, feedURL := range feedURLs {
		currentIndex := i
		sem <- struct{}{} // Acquire semaphore
		g.Go(func() error {
			defer func() { <-sem }() // Release semaphore
			if fa.progressCtx != nil {
				fa.progressCtx.Logf("Processing feed %d/%d: %s", currentIndex+1, len(feedURLs), feedURL)
			}
			feedArticles, err := fa.ParseFeed(feedURL)
			if err != nil {
				if fa.progressCtx != nil {
					fa.progressCtx.Warningf("Failed to parse feed %s: %v", feedURL, err)
				}
				mu.Lock()
				feedErrors = append(feedErrors, fmt.Sprintf("%s: %v", feedURL, err))
				mu.Unlock()
				return nil // Don't return error to continue processing other feeds
			}
			mu.Lock()
			allArticles = append(allArticles, feedArticles...)
			mu.Unlock()
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, fmt.Errorf("failed to process feeds: %w", err)
	}

	if len(feedErrors) > 0 {
		if fa.progressCtx != nil {
			fa.progressCtx.Warningf("Encountered %d feed errors: %v", len(feedErrors), feedErrors)
		}
	}

	return allArticles, nil
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

	if fa.progressCtx != nil {
		fa.progressCtx.Logf("Parsing feed: %s (%d entries)", feedTitle, len(feed.Items))
	}

	var cutoffTime time.Time
	if fa.maxDaysOld > 0 {
		cutoffTime = time.Now().Add(-time.Duration(fa.maxDaysOld) * 24 * time.Hour)
	}

	maxItems := min(fa.maxArticlesPerFeed, len(feed.Items))

	for _, item := range feed.Items[:maxItems] {
		article := fa.extractArticle(item, feedTitle, cutoffTime)
		if article != nil {
			articles = append(articles, article)
		}
	}

	return articles, nil
}

func (fa *FeedAggregator) extractArticle(item *gofeed.Item, feedTitle string, cutoffTime time.Time) *Article {
	// Extract metadata
	title := item.Title
	if title == "" {
		title = "Untitled"
	}

	link := item.Link
	if link == "" {
		return nil
	}

	// Parse publication date
	published := fa.parsePublishedDate(item)

	// Check if article is too old
	if !cutoffTime.IsZero() && !published.IsZero() && published.Before(cutoffTime) {
		if fa.progressCtx != nil {
			fa.progressCtx.Debugf("Skipping old article: %s (%s)", title, published.Format("2006-01-02"))
		}
		return nil
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
	}
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
	// NOTE: Webpage scraping is only attempted when content is empty or <100
	// chars. This balances content quality against latency (webpage fetch adds
	// ~1-2s per article).
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
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", fa.userAgent)

	resp, err := fa.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() //nolint:errcheck

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
