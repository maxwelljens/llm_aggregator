package aggregator

import "time"

// Article represents an item extracted from an RSS feed.
// It is the single typed currency flowing through the pipeline:
// aggregator produces it, processor filters/sorts/truncates it,
// llm and the output formatter read it directly.
type Article struct {
	Title      string    `json:"title"`
	Link       string    `json:"link"`
	Content    string    `json:"content"`
	Published  time.Time `json:"published"`
	Author     string    `json:"author,omitempty"`
	SourceFeed string    `json:"source_feed,omitempty"`
	Summary    string    `json:"summary,omitempty"`
}
