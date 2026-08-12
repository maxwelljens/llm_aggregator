package processor

import (
	"sort"
	"strings"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/aggregator"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/progress"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/tokeniser"
)

// ContentProcessor processes and prepares aggregated content for LLM analysis.
type ContentProcessor struct {
	maxTotalArticles     int
	maxContentPerArticle int
	filterKeywords       []string
	excludeKeywords      []string
	logger               progress.Progress
}

// NewContentProcessor creates a processor that filters and truncates articles.
// Keyword comparison is case-insensitive.
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
		logger:               &progress.NoopLogger{},
	}
}

// SetLogger sets the logger for the processor; nil means no output.
func (cp *ContentProcessor) SetLogger(logger progress.Progress) {
	if logger == nil {
		logger = &progress.NoopLogger{}
	}
	cp.logger = logger
}

// ProcessArticles applies keyword filtering, sorting, a ceiling on total count,
// and content truncation, returning the surviving articles for the LLM.
func (cp *ContentProcessor) ProcessArticles(articles []*aggregator.Article, sortBy string, reverse bool) []*aggregator.Article {
	if len(articles) == 0 {
		cp.logger.Logf("Warning: No articles to process")
		return []*aggregator.Article{}
	}

	// Filter articles
	filteredArticles := cp.filterArticles(articles)

	// Sort articles
	sortedArticles := cp.sortArticles(filteredArticles, sortBy, reverse)

	// Limit total articles
	if len(sortedArticles) > cp.maxTotalArticles {
		cp.logger.Logf("Limiting articles from %d to %d", len(sortedArticles), cp.maxTotalArticles)
		sortedArticles = sortedArticles[:cp.maxTotalArticles]
	}

	// Truncate content for the LLM
	cp.truncateContent(sortedArticles)

	cp.logger.Logf("Processed %d articles (from %d original)", len(sortedArticles), len(articles))

	return sortedArticles
}

// truncateContent caps each article's content at maxContentPerArticle.
// This is the single truncation policy for LLM-bound content.
func (cp *ContentProcessor) truncateContent(articles []*aggregator.Article) {
	for _, article := range articles {
		if len(article.Content) > cp.maxContentPerArticle {
			article.Content = article.Content[:cp.maxContentPerArticle] + "... [truncated]"
		}
	}
}

// filterArticles applies include/exclude keyword filters to the article list.
// Exclusions take priority over inclusions. If no filters are set, all articles pass.
func (cp *ContentProcessor) filterArticles(articles []*aggregator.Article) []*aggregator.Article {
	if len(cp.filterKeywords) == 0 && len(cp.excludeKeywords) == 0 {
		return articles
	}

	filtered := []*aggregator.Article{}

	for _, article := range articles {
		include := true

		if len(cp.excludeKeywords) > 0 {
			articleText := strings.ToLower(article.Title + " " + article.Content)
			for _, keyword := range cp.excludeKeywords {
				if strings.Contains(articleText, keyword) {
					cp.logger.Logf("Excluding article due to keyword '%s': %s", keyword, article.Title)
					include = false
					break
				}
			}
		}

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

	cp.logger.Logf(
		"Filtered %d articles to %d (inclusion: %v, exclusion: %v)",
		len(articles), len(filtered), cp.filterKeywords, cp.excludeKeywords,
	)
	return filtered
}

func (cp *ContentProcessor) sortArticles(articles []*aggregator.Article, sortBy string, reverse bool) []*aggregator.Article {
	if len(articles) == 0 {
		return articles
	}

	sortedArticles := make([]*aggregator.Article, len(articles))
	copy(sortedArticles, articles)

	switch strings.ToLower(sortBy) {
	case "date":
		sortByDate(sortedArticles, reverse)
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
		// Unknown sort key: fall back to date order
		sortByDate(sortedArticles, reverse)
	}

	return sortedArticles
}

// sortByDate orders articles by publication date. Zero times sort last
// regardless of direction since they are neither Before nor After anything.
func sortByDate(articles []*aggregator.Article, reverse bool) {
	sort.Slice(articles, func(i, j int) bool {
		iTime := articles[i].Published
		jTime := articles[j].Published
		if reverse {
			return iTime.After(jTime)
		}
		return iTime.Before(jTime)
	})
}

// EstimateTokenCount uses tiktoken to estimate total token count across all articles.
// Falls back to char÷4 on encoding errors (logged as warnings).
func (cp *ContentProcessor) EstimateTokenCount(articles []*aggregator.Article, model string) int {
	totalTokens := 0

	for _, article := range articles {
		fields := []struct {
			name  string
			value string
		}{
			{"title", article.Title},
			{"content", article.Content},
			{"author", article.Author},
			{"source_feed", article.SourceFeed},
		}

		for _, field := range fields {
			if field.value == "" {
				continue
			}
			tokens, err := tokeniser.CountTokens(field.value, model)
			if err != nil {
				// Fallback to rough estimate on error
				cp.logger.Logf("Warning: token count error for %s: %v", field.name, err)
				totalTokens += len(field.value) / 4
				continue
			}
			totalTokens += tokens
		}
	}

	cp.logger.Logf("Estimated tokens: %d", totalTokens)

	return totalTokens
}
