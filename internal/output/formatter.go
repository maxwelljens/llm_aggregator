package output

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"llm_aggregator/internal/aggregator"
)

// Formatter formats output in different formats.
type Formatter struct {
	formatType string
}

// NewFormatter creates a new formatter with the specified format.
func NewFormatter(formatType string) (*Formatter, error) {
	formatType = strings.ToLower(formatType)
	if formatType != "text" && formatType != "markdown" && formatType != "json" {
		return nil, fmt.Errorf("unsupported format: %s", formatType)
	}
	return &Formatter{formatType: formatType}, nil
}

// FormatData formats data according to the specified format.
func (f *Formatter) FormatData(data map[string]any) (string, error) {
	switch f.formatType {
	case "json":
		return f.formatJSON(data)
	case "markdown":
		return f.formatMarkdown(data)
	default: // text
		return f.formatText(data)
	}
}

func (f *Formatter) formatJSON(data map[string]any) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(bytes), nil
}

func (f *Formatter) formatMarkdown(data map[string]any) (string, error) {
	var lines []string

	// Title
	title := getString(data, "title", "LLM Aggregator Summary")
	lines = append(lines, fmt.Sprintf("# %s", title))
	lines = append(lines, "")

	// Metadata
	lines = append(lines, "## Metadata")
	lines = append(lines, "")

	prompt := getString(data, "prompt", "Unknown")
	model := getString(data, "model", "Unknown")
	articlesCount := getInt(data, "articles_count", 0)
	timestamp := getString(data, "timestamp", time.Now().Format(time.RFC3339))

	lines = append(lines, fmt.Sprintf("- **Prompt**: %s", prompt))
	lines = append(lines, fmt.Sprintf("- **Model**: %s", model))
	lines = append(lines, fmt.Sprintf("- **Articles Analysed**: %d", articlesCount))
	lines = append(lines, fmt.Sprintf("- **Generated**: %s", timestamp))
	lines = append(lines, "")

	// Summary
	lines = append(lines, "## Summary")
	lines = append(lines, "")
	summary := getString(data, "summary", "No summary available.")
	lines = append(lines, summary)
	lines = append(lines, "")

	// Articles (if included)
	if articles, ok := data["articles"].([]map[string]any); ok && len(articles) > 0 {
		lines = append(lines, "## Articles Analysed")
		lines = append(lines, "")

		for i, article := range articles {
			lines = append(lines, fmt.Sprintf("### Article %d: %s", i+1, article["title"]))
			lines = append(lines, "")

			if source, ok := article["source_feed"].(string); ok && source != "" {
				lines = append(lines, fmt.Sprintf("**Source**: %s", source))
			}

			if published, ok := article["published"]; ok {
				switch pub := published.(type) {
				case time.Time:
					if !pub.IsZero() {
						lines = append(lines, fmt.Sprintf("**Published**: %s", pub.Format("2006-01-02 15:04")))
					}
				case string:
					lines = append(lines, fmt.Sprintf("**Published**: %s", pub))
				}
			}

			if author, ok := article["author"].(string); ok && author != "" {
				lines = append(lines, fmt.Sprintf("**Author**: %s", author))
			}

			if link, ok := article["link"].(string); ok && link != "" {
				lines = append(lines, fmt.Sprintf("**Link**: [%s](%s)", link, link))
			}

			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n"), nil
}

func (f *Formatter) formatText(data map[string]any) (string, error) {
	var lines []string

	// Title/Header
	title := getString(data, "title", "LLM Aggregator Summary")
	lines = append(lines, strings.Repeat("=", 80))
	lines = append(lines, centerText(title, 80))
	lines = append(lines, strings.Repeat("=", 80))
	lines = append(lines, "")

	// Metadata
	lines = append(lines, "METADATA")
	lines = append(lines, strings.Repeat("-", 40))

	prompt := getString(data, "prompt", "Unknown")
	model := getString(data, "model", "Unknown")
	articlesCount := getInt(data, "articles_count", 0)
	timestamp := getString(data, "timestamp", time.Now().Format(time.RFC3339))

	lines = append(lines, fmt.Sprintf("Prompt: %s", prompt))
	lines = append(lines, fmt.Sprintf("Model: %s", model))
	lines = append(lines, fmt.Sprintf("Articles Analysed: %d", articlesCount))
	lines = append(lines, fmt.Sprintf("Generated: %s", timestamp))
	lines = append(lines, "")

	// Summary
	lines = append(lines, "SUMMARY")
	lines = append(lines, strings.Repeat("-", 40))
	lines = append(lines, "")
	summary := getString(data, "summary", "No summary available.")
	lines = append(lines, summary)
	lines = append(lines, "")

	// Articles (if included)
	if articles, ok := data["articles"].([]map[string]any); ok && len(articles) > 0 {
		lines = append(lines, "ARTICLES ANALYSED")
		lines = append(lines, strings.Repeat("-", 40))
		lines = append(lines, "")

		for i, article := range articles {
			lines = append(lines, fmt.Sprintf("[Article %d]", i+1))
			lines = append(lines, fmt.Sprintf("Title: %s", article["title"]))

			if source, ok := article["source_feed"].(string); ok && source != "" {
				lines = append(lines, fmt.Sprintf("Source: %s", source))
			}

			if published, ok := article["published"]; ok {
				switch pub := published.(type) {
				case time.Time:
					if !pub.IsZero() {
						lines = append(lines, fmt.Sprintf("Published: %s", pub.Format("2006-01-02 15:04")))
					}
				case string:
					lines = append(lines, fmt.Sprintf("Published: %s", pub))
				}
			}

			if author, ok := article["author"].(string); ok && author != "" {
				lines = append(lines, fmt.Sprintf("Author: %s", author))
			}

			if link, ok := article["link"].(string); ok && link != "" {
				lines = append(lines, fmt.Sprintf("Link: %s", link))
			}

			// Show preview of content
			if content, ok := article["content"].(string); ok && content != "" {
				preview := content
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				lines = append(lines, fmt.Sprintf("Preview: %s", preview))
			}

			lines = append(lines, "")
		}
	}

	// Footer
	lines = append(lines, strings.Repeat("=", 80))
	lines = append(lines, centerText("End of Summary", 80))
	lines = append(lines, strings.Repeat("=", 80))

	return strings.Join(lines, "\n"), nil
}

// Helper functions
func getString(data map[string]any, key, defaultValue string) string {
	if val, ok := data[key].(string); ok {
		return val
	}
	return defaultValue
}

func getInt(data map[string]any, key string, defaultValue int) int {
	if val, ok := data[key].(int); ok {
		return val
	}
	if val, ok := data[key].(float64); ok {
		return int(val)
	}
	return defaultValue
}

func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-padding-len(text))
}

// FormatArticleList formats a list of articles.
func FormatArticleList(articles []*aggregator.Article, formatType string, includeContent bool) (string, error) {
	// Convert articles to maps
	articleMaps := make([]map[string]any, len(articles))
	for i, article := range articles {
		articleMap := article.ToMap()
		if !includeContent {
			delete(articleMap, "content")
		}
		articleMaps[i] = articleMap
	}

	formatter, err := NewFormatter(formatType)
	if err != nil {
		return "", err
	}

	data := map[string]any{
		"title":          fmt.Sprintf("Articles List - %d articles", len(articles)),
		"articles":       articleMaps,
		"articles_count": len(articles),
	}

	return formatter.FormatData(data)
}
