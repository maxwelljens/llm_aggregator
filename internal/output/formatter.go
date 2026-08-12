package output

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/aggregator"
)

// Data is the typed envelope passed to the formatter.
// It replaces the previous map[string]any contract so key names and
// value types are checked by the compiler instead of asserted at runtime.
type Data struct {
	Title         string                `json:"title"`
	Prompt        string                `json:"prompt"`
	Model         string                `json:"model"`
	ArticlesCount int                   `json:"articles_count"`
	Summary       string                `json:"summary"`
	Timestamp     string                `json:"timestamp"`
	Articles      []*aggregator.Article `json:"articles,omitempty"`
}

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
func (f *Formatter) FormatData(data Data) (string, error) {
	switch f.formatType {
	case "json":
		return f.formatJSON(data)
	case "markdown":
		return f.formatMarkdown(data)
	default: // text
		return f.formatText(data)
	}
}

func (f *Formatter) formatJSON(data Data) (string, error) {
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(bytes), nil
}

func (f *Formatter) formatMarkdown(data Data) (string, error) {
	var lines []string

	// Title
	title := data.Title
	if title == "" {
		title = "LLM Aggregator Summary"
	}
	lines = append(lines, "# "+title)
	lines = append(lines, "")

	// Metadata
	lines = append(lines, "## Metadata")
	lines = append(lines, "")

	prompt := data.Prompt
	if prompt == "" {
		prompt = "Unknown"
	}
	model := data.Model
	if model == "" {
		model = "Unknown"
	}
	timestamp := data.Timestamp
	if timestamp == "" {
		timestamp = time.Now().Format(time.RFC3339)
	}

	lines = append(lines, "- **Prompt**: "+prompt)
	lines = append(lines, "- **Model**: "+model)
	lines = append(lines, fmt.Sprintf("- **Articles Analysed**: %d", data.ArticlesCount))
	lines = append(lines, "- **Generated**: "+timestamp)
	lines = append(lines, "")

	// Summary
	lines = append(lines, "## Summary")
	lines = append(lines, "")
	summary := data.Summary
	if summary == "" {
		summary = "No summary available."
	}
	lines = append(lines, summary)
	lines = append(lines, "")

	// Articles (if included)
	if len(data.Articles) > 0 {
		lines = append(lines, "## Articles Analysed")
		lines = append(lines, "")

		for i, article := range data.Articles {
			lines = append(lines, fmt.Sprintf("### Article %d: %s", i+1, article.Title))
			lines = append(lines, "")

			if article.SourceFeed != "" {
				lines = append(lines, "**Source**: "+article.SourceFeed)
			}

			if !article.Published.IsZero() {
				lines = append(lines, "**Published**: "+article.Published.Format("2006-01-02 15:04"))
			}

			if article.Author != "" {
				lines = append(lines, "**Author**: "+article.Author)
			}

			if article.Link != "" {
				lines = append(lines, fmt.Sprintf("**Link**: [%s](%s)", article.Link, article.Link))
			}

			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n"), nil
}

func (f *Formatter) formatText(data Data) (string, error) {
	var lines []string

	// Title/Header
	title := data.Title
	if title == "" {
		title = "LLM Aggregator Summary"
	}
	lines = append(lines, strings.Repeat("=", 80))
	lines = append(lines, centerText(title, 80))
	lines = append(lines, strings.Repeat("=", 80))
	lines = append(lines, "")

	// Metadata
	lines = append(lines, "METADATA")
	lines = append(lines, strings.Repeat("-", 40))

	prompt := data.Prompt
	if prompt == "" {
		prompt = "Unknown"
	}
	model := data.Model
	if model == "" {
		model = "Unknown"
	}
	timestamp := data.Timestamp
	if timestamp == "" {
		timestamp = time.Now().Format(time.RFC3339)
	}

	lines = append(lines, "Prompt: "+prompt)
	lines = append(lines, "Model: "+model)
	lines = append(lines, fmt.Sprintf("Articles Analysed: %d", data.ArticlesCount))
	lines = append(lines, "Generated: "+timestamp)
	lines = append(lines, "")

	// Summary
	lines = append(lines, "SUMMARY")
	lines = append(lines, strings.Repeat("-", 40))
	lines = append(lines, "")
	summary := data.Summary
	if summary == "" {
		summary = "No summary available."
	}
	lines = append(lines, summary)
	lines = append(lines, "")

	// Articles (if included)
	if len(data.Articles) > 0 {
		lines = append(lines, "ARTICLES ANALYSED")
		lines = append(lines, strings.Repeat("-", 40))
		lines = append(lines, "")

		for i, article := range data.Articles {
			lines = append(lines, fmt.Sprintf("[Article %d]", i+1))
			lines = append(lines, "Title: "+article.Title)

			if article.SourceFeed != "" {
				lines = append(lines, "Source: "+article.SourceFeed)
			}

			if !article.Published.IsZero() {
				lines = append(lines, "Published: "+article.Published.Format("2006-01-02 15:04"))
			}

			if article.Author != "" {
				lines = append(lines, "Author: "+article.Author)
			}

			if article.Link != "" {
				lines = append(lines, "Link: "+article.Link)
			}

			// Show preview of content
			if article.Content != "" {
				preview := article.Content
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				lines = append(lines, "Preview: "+preview)
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

func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text + strings.Repeat(" ", width-padding-len(text))
}
