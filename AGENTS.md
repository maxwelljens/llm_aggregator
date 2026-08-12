# AGENTS.md — LLM Aggregator

## Project Overview

LLM Aggregator is a Go CLI tool that fetches RSS feeds, filters/processes
articles, and sends them to an LLM (OpenAI-compatible API) for summarization.
It supports text, markdown, and JSON output, with an optional TUI for progress
visualization.

## Essential Commands

```bash
# Build
go build .

# Run tests
go test ./...

# Test with verbose output
go test ./... -v

# Test with coverage
go test ./... -cover

# Race detector
go test ./... -race

# Build release binaries (goreleaser)
goreleaser build --clean
```

## Architecture

```
main.go (entry point)
        ↓
    signals.New() → Watch()        ← signal setup before any blocking call
        ↓
    cli.ParseArgs()               ← go-arg for CLI parsing
        ↓
    config.GetViper()             ← Viper singleton (config precedence)
        ↓
    config.ViperToRuntime()       ← Converts config → Runtime struct
        ↓
    Runtime.Execute(ctx) → (Result, error):
        ├── aggregator.ParseFeedsFromFile()  ← Concurrent feed fetching
        ├── processor.ProcessArticles()      ← Filter, sort, limit, truncate
        ├── llm.SummariseArticles(ctx)       ← LLM API call (ctx propagated)
        └── Result{Articles, Summary, TokenEstimate}
              ↓
    output.Formatter.FormatData(Data)    ← Format output
```

The context `ctx` passed through `Execute()` carries the cancellation signal. When a signal arrives, `cancel()` is called and the LLM HTTP request aborts mid-flight. `Result.Summary` is written to stdout before `os.Exit(130)`. Dry-run is `Runtime.DryRun = true` — the same pipeline skips the LLM call and returns the processed articles. Tests inject a `runtime.Summariser` fake to exercise `Execute` end-to-end without a network call.

## Key Packages

| Package | Purpose |
|---------|---------|
| `internal/aggregator` | RSS/Atom/JSON feed parsing via gofeed, concurrent fetching with semaphore (maxConcurrency=10) |
| `internal/processor` | Article filtering (keyword include/exclude), sorting (date/title/source), truncation |
| `internal/llm` | OpenAI-compatible API client using openai-go, system/user message construction |
| `internal/runtime` | Orchestrates the pipeline via `Execute(ctx) (Result, error)`; `Summariser` seam for testability |
| `internal/tui` | Bubbletea-based TUI with markdown rendering via glamour |
| `internal/config` | Viper-based config with precedence: CLI > env vars > config file > defaults |
| `internal/cli` | go-arg struct-based CLI parsing |
| `internal/defaults` | Central default constants (model, baseURL, tokens, etc.) |
| `internal/output` | Formatter for text/markdown/json output |
| `internal/progress` | Progress interface with implementations: NoopLogger, SimpleLogger, TUIProgress |
| `internal/tokeniser` | tiktoken-based token counting for accurate estimation |
| `internal/signals` | Signal handling for graceful shutdown (SIGINT, SIGTERM, SIGHUP) |

## Configuration Precedence

1. CLI arguments (only when explicitly provided, not zero values)
2. Environment variables (`LLM_AGGREGATOR_*`)
3. Config file (`~/.config/llm_aggregator/config.toml`)
4. Defaults (from `internal/defaults`)

**Critical**: CLI args use `isZero()` check — if a flag isn't passed, its value doesn't override config file/env values. Pointer types (`*int`, `*string`) are used for CLI flags to detect explicit provision vs. default.

## Gotchas and Non-Obvious Patterns

### 1. Viper is a Singleton
`config.GetViper()` returns the same global instance. Subsequent calls return the same Viper with all previously set values. This means:
- Defaults are set once
- Environment vars are bound once
- Config file is read once
Don't call `GetViper()` multiple times expecting fresh state.

### 2. Feed File Format
Lines starting with `#` are treated as comments. Empty lines are skipped:
```
https://example.com/feed1
# This is a comment
https://example.com/feed2

https://example.com/feed3
```

### 3. FeedAggregator Error Handling
`ParseFeedsFromFile()` continues processing other feeds if one fails. Errors are collected in a `feedErrors` slice and logged as a warning. Only fatal errors (like inability to open the feeds file) cause immediate failure.

### 4. Article Data Flow
`aggregator.Article` is the single typed currency for the whole pipeline: aggregator produces it, processor filters/sorts/truncates it in place, and the LLM and output formatter read its fields directly. The output formatter takes a typed `output.Data` envelope. No `map[string]any` crosses package seams.

### 5. Progress Interface
Components receive `progress.Progress` directly; nil defaults to `NoopLogger` in constructors. Pipeline stages are typed constants (`progress.StageAggregating`, …) shared with the TUI:
- `NoopLogger` (default) — no output
- `SimpleLogger` — stdout logging (verbose mode)
- `TUIProgress` — tea messages for TUI updates

### 6. Token Estimation
`processor.EstimateTokenCount()` uses tiktoken for accurate counting. Models are mapped to encoding names via `tokeniser.EncodingForModel()`. Unknown models fall back to rough char-based estimation.

### 7. Default LLM Settings
- Model: `deepseek-chat`
- Base URL: `https://api.deepseek.com`
- Max Tokens: `4000`
- Temperature: `0.7`

### 8. TUI Rendering
The TUI uses `charm.glamour` for markdown rendering in the terminal. It supports keyboard navigation (j/k, arrows, space, b, g/G) and mouse wheel scrolling.

### 9. Content Extraction Fallback
When feed items lack full content, `aggregator.fetchWebpageContent()` scrapes the article URL using goquery with article/main/.article-content selectors as fallbacks.

### 10. Signal Handling
`main.go` sets up `signals.New()` before any blocking call. `SignalHandler.Watch()` installs a handler for SIGINT, SIGTERM, SIGHUP via `signal.Notify` — disabling Go's default exit behaviour for those signals.

The signal path:
```
SIGINT/SIGTERM/SIGHUP → sh.notify channel → Watch() goroutine: close(sh.Done())
  → runWithoutTUI goroutine: <-sh.Done() → cancel()
    → HTTP request aborted (context cancelled)
    → rt.Execute(ctx) returns context.Canceled
    → runWithoutTUI: partial result.Summary written → exit 130
```

`llm.SummariseArticles()` receives the context from `Runtime.Execute()`, and `callAPIWithMessages()` passes it to `dc.client.Chat.Completions.New()`. The HTTP client aborts on context cancellation.


Exit codes: 0 = success, 1 = error, 130 = signal termination (128 + signal number, per UNIX convention).

## Code Conventions

### Test Naming
```
func Test<Component>_<Action>(t *testing.T)
```
Examples: `TestNewFeedAggregator`, `TestFilterArticlesByExcludeKeywords`, `TestFormatDataJSON`

### Test Structure
Prefer table-driven tests:
```go
tests := []struct {
    name     string
    input    string
    expected string
}{
    {"empty string", "", ""},
    {"single word", "hello", "hello"},
}
for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) { ... })
}
```

### Error Handling Patterns
- Feed errors are collected and logged, not fatal
- Fatal errors (bad config, missing files) exit with code 1
- API errors have specific messages (401 = bad key, 429 = rate limit, etc.)

## Dependencies

| Library | Purpose |
|---------|---------|
| `github.com/mmcdole/gofeed` | RSS/Atom/JSON feed parsing |
| `github.com/openai/openai-go/v3` | OpenAI-compatible API calls |
| `github.com/charmbracelet/bubbletea` | TUI framework |
| `github.com/charmbracelet/lipgloss` | Terminal styling |
| `github.com/charmbracelet/glamour` | Markdown rendering (TUI) |
| `github.com/alexflint/go-arg` | CLI argument parsing |
| `github.com/spf13/viper` | Configuration management |
| `github.com/PuerkitoBio/goquery` | HTML scraping for article content |
| `github.com/pkoukk/tiktoken-go` | BPE token counting |
| `golang.org/x/sync/errgroup` | Concurrent processing with context |

## File Locations

- Config file: `~/.config/llm_aggregator/config.toml` (XDG compliant)
- Entry point: `main.go`
- Version info: `main.buildDate` and `main.version` via ldflags (goreleaser)
- Build: `.goreleaser.yaml`

## Build and Release

Uses goreleaser for cross-platform builds (Linux, Windows, Darwin; arm64, amd64, 386). Build command:
```bash
goreleaser build --clean
```

Version and build date are injected via ldflags:
```
-s -w -X main.buildDate={{.Date}} -X main.version={{.Version}}
```
