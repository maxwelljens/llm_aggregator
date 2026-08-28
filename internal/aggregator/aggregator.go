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

	"codeberg.org/maxwelljensen/llm_aggregator/internal/progress"

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
	progress           progress.Logger
}

// NewFeedAggregator creates a FeedAggregator with progress reporting.
// prog may be nil; nil means no output (NoopLogger).
func NewFeedAggregator(maxArticlesPerFeed, maxDaysOld, maxContentLength int, prog progress.Logger) *FeedAggregator {
	if prog == nil {
		prog = &progress.NoopLogger{}
	}
	return &FeedAggregator{
		maxArticlesPerFeed: maxArticlesPerFeed,
		maxDaysOld:         maxDaysOld,
		maxContentLength:   maxContentLength,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "LLM-Aggregator/0.1.0 (+https://codeberg.org/maxwelljensen/llm-aggregator)",
		progress:  prog,
	}
}

// ParseFeedsFromFile parses RSS feeds from a file containing one URL per line.
// Feeds are fetched concurrently for improved performance; ctx cancellation
// (e.g. SIGINT) aborts in-flight fetches. If the file lists feeds but none of
// them can be fetched, the collected errors are returned as an error instead
// of an empty article list.
func (fa *FeedAggregator) ParseFeedsFromFile(ctx context.Context, filePath string) ([]*Article, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open feeds file: %w", err)
	}
	defer file.Close() //nolint:errcheck

	return fa.parseFeedsFromReader(ctx, file, filePath)
}

// ParseFeedFromStdin parses a single RSS/Atom feed from stdin.
func (fa *FeedAggregator) ParseFeedFromStdin(ctx context.Context) ([]*Article, error) {
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

	return fa.articlesFromFeed(feed, feedTitle), nil
}

// parseFeedsFromReader reads feed URLs line-by-line from any io.Reader,
// then fetches each feed concurrently. Used by both file and stdin paths.
func (fa *FeedAggregator) parseFeedsFromReader(ctx context.Context, reader io.Reader, sourceName string) ([]*Article, error) {
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

	if fa.progress != nil {
		fa.progress.Logf("Found %d feed URLs in %s", len(feedURLs), sourceName)
	}

	// errgroup both bounds concurrency and propagates ctx cancellation to
	// in-flight fetches.
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(maxFeedConcurrency)

	var mu sync.Mutex
	var allArticles []*Article
	var feedErrors []string

	for i, feedURL := range feedURLs {
		currentIndex := i
		g.Go(func() error {
			if fa.progress != nil {
				fa.progress.Logf("Processing feed %d/%d: %s", currentIndex+1, len(feedURLs), feedURL)
			}
			feedArticles, err := fa.ParseFeed(gctx, feedURL)
			if err != nil {
				if gctx.Err() != nil {
					return gctx.Err() // cancelled: stop instead of collecting errors
				}
				if fa.progress != nil {
					fa.progress.Warningf("Failed to parse feed %s: %v", feedURL, err)
				}
				mu.Lock()
				feedErrors = append(feedErrors, fmt.Sprintf("%s: %v", feedURL, err))
				mu.Unlock()
				return nil // keep processing the other feeds
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
		if fa.progress != nil {
			fa.progress.Warningf("Encountered %d feed errors: %v", len(feedErrors), feedErrors)
		}
		if len(allArticles) == 0 {
			return nil, fmt.Errorf("none of the %d feeds could be fetched: %s",
				len(feedURLs), strings.Join(feedErrors, "; "))
		}
	}

	return allArticles, nil
}

// maxFeedConcurrency bounds how many feeds are fetched at once to avoid
// overwhelming servers.
const maxFeedConcurrency = 10

// fetchURL fetches url through the aggregator's HTTP client with the
// configured User-Agent, returning the response body for the caller to close.
func (fa *FeedAggregator) fetchURL(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", fa.userAgent)

	resp, err := fa.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close() //nolint:errcheck
		return nil, fmt.Errorf("HTTP status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

// ParseFeed fetches a single RSS/Atom feed over HTTP (through the injected
// client, honouring ctx) and extracts its articles.
func (fa *FeedAggregator) ParseFeed(ctx context.Context, feedURL string) ([]*Article, error) {
	body, err := fa.fetchURL(ctx, feedURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch feed URL %s: %w", feedURL, err)
	}
	defer body.Close() //nolint:errcheck

	fp := gofeed.NewParser()
	feed, err := fp.Parse(body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed URL %s: %w", feedURL, err)
	}

	feedTitle := feed.Title
	if feedTitle == "" {
		feedTitle = feedURL
	}

	return fa.articlesFromFeed(feed, feedTitle), nil
}

// articlesFromFeed converts feed items into Articles, applying the per-feed
// limit and the max-days-old cutoff.
func (fa *FeedAggregator) articlesFromFeed(feed *gofeed.Feed, feedTitle string) []*Article {
	if fa.progress != nil {
		fa.progress.Logf("Parsing feed: %s (%d entries)", feedTitle, len(feed.Items))
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

	return articles
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
		if fa.progress != nil {
			fa.progress.Debugf("Skipping old article: %s (%s)", title, published.Format("2006-01-02"))
		}
		return nil
	}

	// Extract author
	author := ""
	if len(item.Authors) > 0 {
		author = item.Authors[0].Name
		if author == "" {
			author = item.Authors[0].Email
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
	content := ""

	// Feed item content fields are tried in order; gofeed unifies atom Content/Description
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

	// Remove noise elements before extracting text
	doc.Find("script, style, nav, footer, header").Remove()

	// Try article-shaped selectors first, then fall back to paragraphs, then body
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

	// Fallback: collect all paragraphs
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

	// Last resort: raw body text
	text := doc.Text()
	text = strings.Join(strings.Fields(text), " ")
	if len(text) > 200 {
		return text, nil
	}

	return "", nil
}
