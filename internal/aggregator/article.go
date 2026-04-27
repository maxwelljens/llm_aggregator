package aggregator

import "time"

// Article represents an item extracted from an RSS feed.
type Article struct {
	Title      string    `json:"title"`
	Link       string    `json:"link"`
	Content    string    `json:"content"`
	Published  time.Time `json:"published"`
	Author     string    `json:"author,omitempty"`
	SourceFeed string    `json:"source_feed,omitempty"`
	Summary    string    `json:"summary,omitempty"`
}

// ToMap serialises the article to a map for LLM input or JSON output.
// Content is truncated to 500 chars to avoid oversized prompts.
func (a *Article) ToMap() map[string]any {
	content := a.Content
	if len(content) > 500 {
		content = content[:500] + "... [truncated]"
	}

	result := map[string]any{
		"title":       a.Title,
		"link":        a.Link,
		"content":     content,
		"author":      a.Author,
		"source_feed": a.SourceFeed,
		"summary":     a.Summary,
	}

	if !a.Published.IsZero() {
		result["published"] = a.Published.Format(time.RFC3339)
	}

	return result
}

