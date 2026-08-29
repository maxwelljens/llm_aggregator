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
    cli.ParseArgs() → (*Args, handled bool, err)   ← go-arg; help/version handled here
        ↓
    signals.New() → Watch()        ← signal setup before any blocking call
        ↓
    config.GetViper() → ViperToRuntime(v, args)    ← settings table, config precedence
        ↓
    run(rt, args, sh, stdout, stderr) int          ← path selection + exit-code policy
        ├── dry-run: rt.DryRun = true → Runtime.Execute(ctx) skips the LLM call
        ├── otherwise: run() injects llm.NewLLMClient(llm.Options{...}) as rt.Summariser
        └── TUI → runWithTUI   |   non-TUI → runWithoutTUI
                ↓
    Runtime.Execute(ctx) → (Result, error):
        ├── aggregator.ParseFeedsFromFile(ctx)   ← concurrent, context-aware fetching
        ├── processor.ProcessArticles()          ← filter, sort, limit, truncate
        ├── summariser.SummariseArticles(ctx)    ← injected Summariser, ctx propagated
        └── Result{Articles, Summary, TokenEstimate}
              ↓
    rt.WriteOutput(stdout, result) → output.Formatter.FormatData(Data)
```

Only `main` calls `os.Exit`; `run()` and every `run*` path return an exit code (0 success, 1 error, 130 signal). The TUI path runs the same pipeline through a `tui.ExecuteFunc` and writes output after the program exits, with identical exit-code semantics. The context `ctx` passed through `Execute()` carries the cancellation signal; when a signal arrives, `cancel()` is called and the LLM HTTP request aborts mid-flight, with any partial summary still written to stdout before exit 130. Tests inject a `runtime.Summariser` fake to exercise `Execute` end-to-end without a network call.

## Key Packages

| Package | Purpose |
|---------|---------|
| `internal/aggregator` | RSS/Atom/JSON feed parsing via gofeed; context-aware fetching through one injected HTTP client (`ParseFeed(ctx, url)`), concurrency capped by errgroup `SetLimit(maxFeedConcurrency=10)` |
| `internal/processor` | Article filtering (keyword include/exclude), sorting (date/title/source), truncation |
| `internal/llm` | OpenAI-compatible API client; constructed via `NewLLMClient(Options)` — no env fallback inside the package (config layer owns that) |
| `internal/runtime` | Orchestrates the pipeline via `Execute(ctx) (Result, error)`; `Summariser` is injected (nil + non-dry-run is a hard error) |
| `internal/tui` | Bubbletea TUI with markdown rendering via glamour; `New(execute ExecuteFunc, ctx)` and `FinalResult()` hand the pipeline outcome back to `main` |
| `internal/config` | Viper-based config with precedence: CLI > env vars > config file > defaults |
| `internal/cli` | go-arg struct-based CLI parsing; `ParseArgs()` returns `(*Args, handled bool, error)` and `WriteHelp` takes an `io.Writer` |
| `internal/defaults` | Central default constants (model, baseURL, tokens, content caps, truncated suffix, etc.) |
| `internal/output` | Formatter for text/markdown/json output |
| `internal/progress` | `Logger` (Logf/Warningf/Debugf) + `StageReporter` (typed stage updates); `Progress` = both. Implementations: NoopLogger, SimpleLogger |
| `internal/tokeniser` | tiktoken-based token counting for accurate estimation |
| `internal/signals` | Signal handling for graceful shutdown (SIGINT, SIGTERM, SIGHUP) |

## Configuration Precedence

1. CLI arguments (only when explicitly provided, not zero values)
2. Environment variables (`LLM_AGGREGATOR_*`)
3. Config file (`~/.config/llm_aggregator/config.toml`)
4. Defaults (from `internal/defaults`)

**Critical**: CLI flags are only applied when explicitly provided. The config package keeps a single settings table (`viperSettings()`) where every knob is declared once (key, env name derived as `LLM_AGGREGATOR_` + upper key, default, Runtime applier, CLI extractor). The CLI extractor uses pointer-presence semantics: `*int`/`*string` flags apply only when non-nil, so `--temperature 0` overrides but an absent flag does not.

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
`ParseFeedsFromFile(ctx)` continues processing other feeds if one fails. Per-feed errors are collected and logged as a warning, and a cancelled context short-circuits the loop. If **all** feeds fail, the call returns an error (`"none of the %d feeds could be fetched: ..."`) instead of silently producing zero articles. Only fatal errors (like inability to open the feeds file) fail immediately.

### 4. Article Data Flow
`aggregator.Article` is the single typed currency for the whole pipeline: aggregator produces it, processor filters/sorts/truncates it in place, and the LLM and output formatter read its fields directly. The output formatter takes a typed `output.Data` envelope. No `map[string]any` crosses package seams.

### 5. Progress Interfaces
Progress is split into two seams: `progress.Logger` (Logf/Warningf/Debugf) and `progress.StageReporter` (SetStage/SetSubStage/SetArticleCount/SetTokenEstimate with typed `Stage` constants shared with the TUI). Components take `progress.Logger` directly; nil defaults to `NoopLogger` in constructors, and `SimpleLogger` embeds a `noStages` adapter so log-only loggers need no stage stubs:
- `NoopLogger` (default) — no output
- `SimpleLogger` — stdout logging (verbose mode)
- `TUIProgress` — tea messages for TUI updates (implements full `Progress`)

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

### 10. Signal Handling and Exit-Code Ownership
`main.go` sets up `signals.New()` before any blocking call. `SignalHandler.Watch()` installs a handler for SIGINT, SIGTERM, SIGHUP via `signal.Notify` — disabling Go's default exit behaviour for those signals. The non-TUI path and the TUI path both derive a cancellable context from `sh.Done()` so signals abort the in-flight LLM request; after a `tea.Program` exits, `runWithTUI` pulls the outcome via `model.FinalResult()` and applies the same output/exit-code rules as the non-TUI path.

The signal path (non-TUI):
```
SIGINT/SIGTERM/SIGHUP → sh.notify channel → Watch() goroutine: close(sh.Done())
  → runWithoutTUI goroutine: <-sh.Done() → cancel()
    → HTTP request aborted (context cancelled)
    → rt.Execute(ctx) returns context.Canceled
    → runWithoutTUI: partial result.Summary written → exit 130
```

`llm.SummariseArticles()` receives the context from `Runtime.Execute()`, and `callAPIWithMessages()` passes it to the openai-go completion call. The HTTP client aborts on context cancellation.


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
- Feed errors are collected and logged; only when every feed fails does `Execute` error out
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
