package runtime

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/aggregator"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/llm"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/progress"
)

const testAtomFeed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Stdin Feed</title>
  <link href="https://example.com/atom"/>
  <entry>
    <title>Stdin Article</title>
    <link href="https://example.com/atom1"/>
    <content>Content from stdin feed.</content>
    <updated>2024-01-15T10:00:00Z</updated>
  </entry>
</feed>`

// fakeSummariser records calls and returns a canned response, letting tests
// drive the whole Execute pipeline without a network call.
type fakeSummariser struct {
	mu      sync.Mutex
	calls   int
	summary string
	err     error
}

func (f *fakeSummariser) SummariseArticles(articles []*aggregator.Article, userPrompt, systemPrompt string, ctx context.Context) (string, *llm.TokenUsage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.summary, &llm.TokenUsage{PromptTokens: 10, CompletionTokens: 5}, f.err
}

func (f *fakeSummariser) SetLogger(progress.Progress) {}

func (f *fakeSummariser) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func testFeedsFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "feeds.txt")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp feeds file: %v", err)
	}
	return tmpFile
}

// serveFeed spins up an HTTP server that serves testAtomFeed for any request.
func serveFeed(t *testing.T) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/atom+xml")
		_, _ = fmt.Fprint(w, testAtomFeed)
	}))
	t.Cleanup(server.Close)
	return server
}

// withStdinFeed replaces os.Stdin with a pipe carrying the test feed.
func withStdinFeed(t *testing.T) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.WriteString(testAtomFeed); err != nil {
		t.Fatalf("write pipe: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe: %v", err)
	}
	oldStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })
}

func baseRuntime(summariser Summariser) *Runtime {
	return &Runtime{
		MaxArticlesPerFeed: 10,
		MaxDaysOld:         0,
		MaxTotalArticles:   20,
		Progress:           &progress.NoopLogger{},
		Summariser:         summariser,
	}
}

// TestExecuteBranchStdinOnly verifies Execute parses stdin and summarises.
func TestExecuteBranchStdinOnly(t *testing.T) {
	withStdinFeed(t)
	sum := &fakeSummariser{summary: "stdin summary"}

	rt := baseRuntime(sum)
	rt.Stdin = true

	result, err := rt.Execute(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.Articles) != 1 {
		t.Fatalf("Expected 1 article, got %d", len(result.Articles))
	}
	if result.Articles[0].Title != "Stdin Article" {
		t.Errorf("Expected title 'Stdin Article', got %q", result.Articles[0].Title)
	}
	if result.Articles[0].SourceFeed != "Stdin Feed" {
		t.Errorf("Expected source 'Stdin Feed', got %q", result.Articles[0].SourceFeed)
	}
	if result.Summary != "stdin summary" {
		t.Errorf("Expected summary %q, got %q", "stdin summary", result.Summary)
	}
	if sum.callCount() != 1 {
		t.Errorf("Expected 1 summariser call, got %d", sum.callCount())
	}
}

// TestExecuteBranchFeedsFileOnly verifies Execute fetches a feeds file and summarises.
func TestExecuteBranchFeedsFileOnly(t *testing.T) {
	server := serveFeed(t)
	feedsFile := testFeedsFile(t, server.URL+"\n")
	sum := &fakeSummariser{summary: "file summary"}

	rt := baseRuntime(sum)
	rt.FeedsFile = feedsFile

	result, err := rt.Execute(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.Articles) != 1 {
		t.Fatalf("Expected 1 article, got %d", len(result.Articles))
	}
	if result.Articles[0].Title != "Stdin Article" {
		t.Errorf("Expected article from feed, got %q", result.Articles[0].Title)
	}
	if result.Summary != "file summary" {
		t.Errorf("Expected summary %q, got %q", "file summary", result.Summary)
	}
}

// TestExecuteBranchCollated verifies stdin + feeds file are collated before summarising.
func TestExecuteBranchCollated(t *testing.T) {
	withStdinFeed(t)
	server := serveFeed(t)
	feedsFile := testFeedsFile(t, server.URL+"\n")
	sum := &fakeSummariser{summary: "collated summary"}

	rt := baseRuntime(sum)
	rt.Stdin = true
	rt.FeedsFile = feedsFile

	result, err := rt.Execute(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(result.Articles) != 2 {
		t.Fatalf("Expected 2 articles after collation, got %d", len(result.Articles))
	}
	if result.ArticlesFetched != 2 {
		t.Errorf("Expected ArticlesFetched 2, got %d", result.ArticlesFetched)
	}
	if result.Summary != "collated summary" {
		t.Errorf("Expected summary %q, got %q", "collated summary", result.Summary)
	}
}

// TestExecuteBranchNoSource verifies Execute errors when no feed source is set.
func TestExecuteBranchNoSource(t *testing.T) {
	sum := &fakeSummariser{summary: "should not be called"}
	rt := baseRuntime(sum)

	_, err := rt.Execute(context.Background())
	if !errors.Is(err, ErrNoFeedSource) {
		t.Fatalf("Expected ErrNoFeedSource, got %v", err)
	}
	if sum.callCount() != 0 {
		t.Errorf("Summariser should not be called without a feed source")
	}
}

// TestExecuteDryRun verifies dry-run skips the LLM but returns processed articles.
func TestExecuteDryRun(t *testing.T) {
	withStdinFeed(t)
	sum := &fakeSummariser{summary: "should not be called"}

	rt := baseRuntime(sum)
	rt.Stdin = true
	rt.DryRun = true

	result, err := rt.Execute(context.Background())
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.Summary != "" {
		t.Errorf("Expected empty summary in dry-run, got %q", result.Summary)
	}
	if len(result.Articles) != 1 {
		t.Fatalf("Expected 1 article, got %d", len(result.Articles))
	}
	if result.TokenEstimate <= 0 {
		t.Errorf("Expected positive token estimate, got %d", result.TokenEstimate)
	}
	if sum.callCount() != 0 {
		t.Errorf("Summariser must not be called in dry-run, got %d calls", sum.callCount())
	}
}

// TestExecuteSummariserError verifies LLM errors propagate from Execute.
func TestExecuteSummariserError(t *testing.T) {
	withStdinFeed(t)
	sum := &fakeSummariser{summary: "", err: errors.New("boom")}

	rt := baseRuntime(sum)
	rt.Stdin = true

	_, err := rt.Execute(context.Background())
	if err == nil {
		t.Fatal("Expected error from summariser")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("Expected wrapped summariser error, got %v", err)
	}
}

// TestWriteOutputFormatted verifies WriteOutput renders the result in the chosen format.
func TestWriteOutputFormatted(t *testing.T) {
	rt := &Runtime{
		Output:          "json",
		Prompt:          "test prompt",
		Model:           "test-model",
		IncludeArticles: true,
	}
	result := Result{
		Articles: []*aggregator.Article{
			{Title: "Art One", Link: "https://example.com/1", SourceFeed: "Feed"},
		},
		Summary: "the summary",
	}

	var buf strings.Builder
	if err := rt.WriteOutput(&buf, result); err != nil {
		t.Fatalf("WriteOutput failed: %v", err)
	}
	for _, want := range []string{`"summary": "the summary"`, `"title": "Art One"`, `"articles_count": 1`} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("JSON output missing %s: %s", want, buf.String())
		}
	}
}

// TestExecuteRequiresSummariser verifies a non-dry-run Execute without an
// injected Summariser fails instead of constructing a real LLM client.
func TestExecuteRequiresSummariser(t *testing.T) {
	withStdinFeed(t)
	rt := baseRuntime(nil)
	rt.Stdin = true

	_, err := rt.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error for missing summariser, got nil")
	}
	if !strings.Contains(err.Error(), "summariser") {
		t.Errorf("error %q does not mention the summariser", err.Error())
	}
}
