package processor

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"llm_aggregator/internal/aggregator"
	"llm_aggregator/internal/progress"
)

// ContentProcessor processes and prepares aggregated content for LLM analysis.
type ContentProcessor struct {
	maxTotalArticles     int
	maxContentPerArticle int
	filterKeywords       []string
	excludeKeywords      []string
	logger               *progress.Context
}

// NewContentProcessor creates a new ContentProcessor with the specified options.
func NewContentProcessor(maxTotalArticles, maxContentPerArticle int, filterKeywords, excludeKeywords []string) *ContentProcessor {
	// Convert keywords to lowercase for case-insensitive matching
	filterLower := make([]string, len(filterKeywords))
	for i, kw := range filterKeywords {
		filterLower[i] = strings.ToLower(kw)
	}

	excludeLower := make([]string, len(excludeKeywords))
	for i, kw := range excludeKeywords {
		excludeLower[i] = strings.ToLower(kw)
	}

	return &ContentProcessor{
		maxTotalArticles:     maxTotalArticles,
		maxContentPerArticle: maxContentPerArticle,
		filterKeywords:       filterLower,
		excludeKeywords:      excludeLower,
	}
}

// SetLogger sets the logger for the processor
func (cp *ContentProcessor) SetLogger(logger *progress.Context) {
	cp.logger = logger
}

// ProcessArticles processes articles: filter, sort, and prepare for LLM.
func (cp *ContentProcessor) ProcessArticles(articles []*aggregator.Article, sortBy string, reverse bool) []map[string]any {
	if len(articles) == 0 {
		if cp.logger != nil {
			cp.logger.Logf("Warning: No articles to process")
		}
		return []map[string]any{}
	}

	// Filter articles
	filteredArticles := cp.filterArticles(articles)

	// Sort articles
	sortedArticles := cp.sortArticles(filteredArticles, sortBy, reverse)

	// Limit total articles
	if len(sortedArticles) > cp.maxTotalArticles {
		if cp.logger != nil {
			cp.logger.Logf("Limiting articles from %d to %d", len(sortedArticles), cp.maxTotalArticles)
		}
		sortedArticles = sortedArticles[:cp.maxTotalArticles]
	}

	// Prepare articles for LLM
	processed := cp.prepareForLLM(sortedArticles)

	if cp.logger != nil {
		cp.logger.Logf("Processed %d articles (from %d original)", len(processed), len(articles))
	}

	return processed
}

func (cp *ContentProcessor) filterArticles(articles []*aggregator.Article) []*aggregator.Article {
	if len(cp.filterKeywords) == 0 && len(cp.excludeKeywords) == 0 {
		return articles
	}

	filtered := []*aggregator.Article{}

	for _, article := range articles {
		include := true

		// Check if article should be excluded
		if len(cp.excludeKeywords) > 0 {
			articleText := strings.ToLower(article.Title + " " + article.Content)
			for _, keyword := range cp.excludeKeywords {
				if strings.Contains(articleText, keyword) {
					if cp.logger != nil {
						cp.logger.Logf("Excluding article due to keyword '%s': %s", keyword, article.Title)
					}
					include = false
					break
				}
			}
		}

		// Check if article should be included (only if we have inclusion filters)
		if include && len(cp.filterKeywords) > 0 {
			articleText := strings.ToLower(article.Title + " " + article.Content)
			include = false
			for _, keyword := range cp.filterKeywords {
				if strings.Contains(articleText, keyword) {
					include = true
					break
				}
			}
		}

		if include {
			filtered = append(filtered, article)
		}
	}

	if cp.logger != nil {
		cp.logger.Logf(
			"Filtered %d articles to %d (inclusion: %v, exclusion: %v)",
			len(articles), len(filtered), cp.filterKeywords, cp.excludeKeywords,
		)
	}

	return filtered
}

func (cp *ContentProcessor) sortArticles(articles []*aggregator.Article, sortBy string, reverse bool) []*aggregator.Article {
	if len(articles) == 0 {
		return articles
	}

	// Create a copy to avoid modifying the original slice
	sortedArticles := make([]*aggregator.Article, len(articles))
	copy(sortedArticles, articles)

	// Define sort functions
	switch strings.ToLower(sortBy) {
	case "date":
		sort.Slice(sortedArticles, func(i, j int) bool {
			iTime := sortedArticles[i].Published
			jTime := sortedArticles[j].Published
			if iTime.IsZero() {
				iTime = time.Time{}
			}
			if jTime.IsZero() {
				jTime = time.Time{}
			}
			if reverse {
				return iTime.After(jTime)
			}
			return iTime.Before(jTime)
		})
	case "title":
		sort.Slice(sortedArticles, func(i, j int) bool {
			iTitle := strings.ToLower(sortedArticles[i].Title)
			jTitle := strings.ToLower(sortedArticles[j].Title)
			if reverse {
				return iTitle > jTitle
			}
			return iTitle < jTitle
		})
	case "source":
		sort.Slice(sortedArticles, func(i, j int) bool {
			iSource := strings.ToLower(sortedArticles[i].SourceFeed)
			jSource := strings.ToLower(sortedArticles[j].SourceFeed)
			if reverse {
				return iSource > jSource
			}
			return iSource < jSource
		})
	default:
		// Default to date sorting
		sort.Slice(sortedArticles, func(i, j int) bool {
			iTime := sortedArticles[i].Published
			jTime := sortedArticles[j].Published
			if iTime.IsZero() {
				iTime = time.Time{}
			}
			if jTime.IsZero() {
				jTime = time.Time{}
			}
			if reverse {
				return iTime.After(jTime)
			}
			return iTime.Before(jTime)
		})
	}

	return sortedArticles
}

func (cp *ContentProcessor) prepareForLLM(articles []*aggregator.Article) []map[string]any {
	processed := make([]map[string]any, len(articles))

	for i, article := range articles {
		// Process content
		content := article.Content
		if len(content) > cp.maxContentPerArticle {
			content = content[:cp.maxContentPerArticle] + "... [truncated]"
		}

		// Create map
		articleMap := map[string]any{
			"title":       article.Title,
			"link":        article.Link,
			"content":     content,
			"author":      article.Author,
			"source_feed": article.SourceFeed,
			"summary":     article.Summary,
		}

		if !article.Published.IsZero() {
			articleMap["published"] = article.Published
		}

		processed[i] = articleMap
	}

	return processed
}

// EstimateTokenCount estimates token count for articles (rough approximation).
// Note: This is a rough estimate (1 token ≈ 4 characters for English).
// Actual tokenisation may vary.
func (cp *ContentProcessor) EstimateTokenCount(articles []map[string]any) int {
	totalChars := 0

	for _, article := range articles {
		for _, field := range []string{"title", "content", "author", "source_feed"} {
			if val, ok := article[field]; ok && val != nil {
				if str, ok := val.(string); ok {
					totalChars += len(str)
				}
			}
		}
	}

	// Rough estimate: 1 token ≈ 4 characters for English
	estimatedTokens := totalChars / 4

	if cp.logger != nil {
		cp.logger.Logf("Estimated tokens: %d (~%d chars)", estimatedTokens, totalChars)
	}

	return estimatedTokens
}

// CreateConciseContext creates a concise context from articles, respecting token limits.
func (cp *ContentProcessor) CreateConciseContext(articles []map[string]any, maxTotalTokens int) string {
	if len(articles) == 0 {
		return "No articles available."
	}

	contextParts := []string{}
	currentTokens := 0

	for i, article := range articles {
		// Create article summary
		articleText := fmt.Sprintf("Article %d: %s\n", i+1, article["title"])

		// Add source if available
		if source, ok := article["source_feed"].(string); ok && source != "" {
			articleText += fmt.Sprintf("Source: %s\n", source)
		}

		// Add publication date if available
		if published, ok := article["published"]; ok {
			switch pub := published.(type) {
			case time.Time:
				if !pub.IsZero() {
					articleText += fmt.Sprintf("Published: %s\n", pub.Format("2006-01-02"))
				}
			case string:
				articleText += fmt.Sprintf("Published: %s\n", pub)
			}
		}

		// Add content (truncated if needed)
		content := ""
		if c, ok := article["content"].(string); ok {
			content = c
		}

		maxContentTokens := maxTotalTokens / len(articles) // Rough allocation
		maxContentChars := maxContentTokens * 4

		if len(content) > maxContentChars {
			content = content[:maxContentChars] + "... [truncated]"
		}

		articleText += fmt.Sprintf("Content: %s\n", content)

		// Estimate tokens for this article
		articleTokens := len(articleText) / 4

		// Check if adding this article would exceed limit
		if currentTokens+articleTokens > maxTotalTokens {
			if cp.logger != nil {
				cp.logger.Logf(
					"Reached token limit (%d). Included %d of %d articles.",
					maxTotalTokens, i, len(articles),
				)
			}
			break
		}

		contextParts = append(contextParts, articleText)
		currentTokens += articleTokens
	}

	context := strings.Join(contextParts, "\n---\n")

	if cp.logger != nil {
		cp.logger.Logf("Created context with %d articles, ~%d tokens", len(contextParts), currentTokens)
	}

	return context
}

