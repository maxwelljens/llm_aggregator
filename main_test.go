package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/aggregator"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/cli"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/llm"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/progress"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/runtime"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/signals"
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

const emptyAtomFeed = `<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Empty Feed</title>
</feed>`

// withStdin replaces os.Stdin with a pipe carrying content.
func withStdin(t *testing.T, content string) {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.WriteString(content); err != nil {
		t.Fatalf("write pipe: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close pipe: %v", err)
	}
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old })
}

type fakeSummariser struct {
	mu      sync.Mutex
	summary string
	err     error
	calls   int
}

func (f *fakeSummariser) SummariseArticles(_ []*aggregator.Article, _, _ string, _ context.Context) (string, *llm.TokenUsage, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	return f.summary, &llm.TokenUsage{PromptTokens: 10, CompletionTokens: 5}, f.err
}

func (f *fakeSummariser) SetLogger(_ progress.Progress) {}

func stdinArgs() *cli.Args {
	return &cli.Args{Stdin: true, Prompt: "test prompt"}
}

func stdinRuntime(sum *fakeSummariser) *runtime.Runtime {
	return &runtime.Runtime{
		Stdin:              true,
		Prompt:             "test prompt",
		APIKey:             "test-key",
		Output:             "text",
		MaxArticlesPerFeed: 10,
		MaxTotalArticles:   20,
		Summariser:         sum,
	}
}

func TestRun_MissingAPIKey(t *testing.T) {
	rt := &runtime.Runtime{Stdin: true, Prompt: "p"}
	var stdout, stderr bytes.Buffer

	code := run(rt, stdinArgs(), signals.New(), &stdout, &stderr)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "API key") {
		t.Errorf("stderr %q does not mention the API key", stderr.String())
	}
}

func TestRun_DryRunNoFeedSource(t *testing.T) {
	rt := &runtime.Runtime{DryRun: true, Prompt: "p"}
	args := &cli.Args{DryRun: true, Prompt: "p"}
	var stdout, stderr bytes.Buffer

	code := run(rt, args, signals.New(), &stdout, &stderr)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "dry-run failed") {
		t.Errorf("stderr %q does not report the dry-run failure", stderr.String())
	}
}

func TestRun_DryRunNoArticles(t *testing.T) {
	withStdin(t, emptyAtomFeed)
	rt := stdinRuntime(nil)
	rt.DryRun = true
	args := stdinArgs()
	args.DryRun = true
	var stdout, stderr bytes.Buffer

	code := run(rt, args, signals.New(), &stdout, &stderr)

	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), "No articles found") {
		t.Errorf("stderr %q does not carry the no-articles message", stderr.String())
	}
}

func TestRun_NonTUISuccess(t *testing.T) {
	withStdin(t, testAtomFeed)
	sum := &fakeSummariser{summary: "hello summary"}
	rt := stdinRuntime(sum)
	var stdout, stderr bytes.Buffer

	code := run(rt, stdinArgs(), signals.New(), &stdout, &stderr)

	if code != 0 {
		t.Errorf("exit code = %d, want 1 (stderr: %s)", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "hello summary") {
		t.Errorf("stdout %q does not contain the summary", stdout.String())
	}
	if sum.calls != 1 {
		t.Errorf("summariser calls = %d, want 1", sum.calls)
	}
}

func TestRun_SummariserError(t *testing.T) {
	withStdin(t, testAtomFeed)
	sum := &fakeSummariser{err: errors.New("llm exploded")}
	rt := stdinRuntime(sum)
	var stdout, stderr bytes.Buffer

	code := run(rt, stdinArgs(), signals.New(), &stdout, &stderr)

	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "llm exploded") {
		t.Errorf("stderr %q does not report the summariser error", stderr.String())
	}
}
