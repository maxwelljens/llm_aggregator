# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a
Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- Signal handling no longer polls in a busy loop: `SignalHandler.Done()` is a
  channel closed on the first signal, which cancels the execution context.
- The `progress.Context` pass-through wrapper is gone; components receive
  `progress.Progress` directly and default to a no-op logger. Pipeline stages
  are typed constants shared with the TUI, so a stage-name mismatch is a
  compile error instead of a frozen progress bar.
- `Article` is now the typed currency of the pipeline: the processor no longer
  degrades articles to `map[string]any`, the LLM renders typed fields without
  type-switching, and the output formatter takes a typed `Data` envelope.
  Content truncation happens once, in the processor.
- `Runtime.Execute` returns a `Result` (articles, summary, token estimate)
  instead of mutating shared state, and accepts an injectable `Summariser`
  seam. Dry-run is a flag on the same pipeline rather than a reimplementation.
- Dead code removed: `Article.ToMap`, the unused `Config` struct,
  `Runtime.Interrupted`/`WasInterrupted`/`Error`, unused style variables, and
  duplicated default values in help text and the LLM system prompt.

### Fixed

- Deprecated `gofeed` `item.Author` replaced with `item.Authors`.
- The duplicated date-sort comparator in the processor is a single helper.

### Added

- `Runtime.Execute` is now tested end-to-end: all four feed-source branches,
  dry-run, and summariser error propagation run against a fake summariser.
- Coverage for LLM context rendering and message construction.

## [1.0.2] - 2026-05-04

### Changed

- Module path updated to `codeberg.org/maxwelljensen/llm_aggregator` — all
  internal imports updated accordingly.

## [1.0.0] - 2026-04-27

### Added

- Linting and code quality tooling via `golangci-lint`.

### Fixed

- Tight CPU-spinning polling loop in signal handler (added `time.Sleep` to
  avoid busy-waiting).
- `--temperature 0` was silently overwritten by the default (0.7); temperature
  is now a pointer so `nil` means "not provided" and a non-nil value — including
  `0.0` — means "explicitly set".
- `WasInterrupted` godoc comment capitalised to satisfy `golint`.

## [0.15.0] - 2026-04-27

### Added

- **`--timeout N` flag**: LLM request timeout in seconds (default: 300). If the
  LLM API call takes longer than this, the request is aborted and a descriptive
  error is returned. The timeout is implemented as a `context.WithTimeout`
  derived from the signal-handling context, so both signal interrupts and
  timeouts abort the call.
- **`DefaultLLMTimeout` constant**: Centralises the default timeout value
  (300 seconds) in `internal/defaults`.
- **LLM client unit tests**: Three tests for the constructor: timeout default,
  all-parameter defaults, and API key validation.

### Changed

- **`LLMClient.llmTimeout` field**: Renamed from `timeoutSeconds` for clarity.
  The field stores the timeout in seconds; 0 means no timeout.

## [0.14.0] - 2026-04-27

### Added

- **Signal handling for graceful shutdown**: `llm_aggregator` now handles
  `SIGINT`, `SIGTERM`, and `SIGHUP`. When a signal arrives during execution:
  - The context is cancelled, stopping in-progress operations as soon as possible
  - Any partial LLM summary already received is written to stdout (or the output file)
  - The process exits with code 130 (128 + signal number) instead of a generic error
  - In TUI mode, `q`/`Ctrl+C` also triggers clean shutdown via `RequestExit()`
- **`Runtime.Interrupted` and `Runtime.WasInterrupted()`**: New state field and
  query method allow callers to detect signal-caused termination.

## [0.13.0] - 2026-04-26

### Added

- **`--stdin` flag**: Read raw RSS/Atom XML directly from standard input,
  composable with `-f/--feeds-file` to collate articles from both sources into
  a single LLM summary. Stdin articles appear before feeds-file articles in the
  collated output.
  - `curl -s $URL | llm_aggregator --stdin -p "Summarise"`
  - `llm_aggregator -f feeds.txt --stdin -p "Summarise all"`
  - Environment variable: `LLM_AGGREGATOR_STDIN`
- **`--feeds-file` is no longer required**: Either `--feeds-file` or `--stdin`
  (or both) must be provided. A clear error is returned if neither is specified.

### Changed

- Refactored `ParseFeedsFromFile` to delegate to a new `ParseFeedFromReader`
  method that accepts any `io.Reader`, enabling both stdin and unit test usage
  with `strings.Reader`.

## [0.12.0] - 2026-04-26

### Added

- **`-P/--plain` flag**: Output the raw LLM summary without any formatting or
  metadata. Useful when the summary is to be consumed by downstream scripts.

## [0.11.0] - 2026-04-23

- Progress bar now tracks estimated token count (via tiktoken) before sending
  to LLM, then updates with actual completion tokens from the API response.
  Gives a more accurate picture of how much content is being sent.

## [0.10.0] - 2026-04-23

### Added

- **Short command-line flags**: Single-character shortcuts for the most
  commonly used options:
  - `-f` / `--feeds-file` — feeds file path
  - `-p` / `--prompt` — summarisation prompt
  - `-n` / `--max-articles-per-feed` — articles per feed limit
  - `-d` / `--max-days-old` — article age filter
  - `-i` / `--include-keywords` — keyword include filter
  - `-e` / `--exclude-keywords` — keyword exclude filter
  - `-m` / `--model` — LLM model selection
  - `-o` / `--output` — output format
  - `-t` / `--tui` — enable TUI
  - `-D` / `--dry-run` — dry run mode
- **Styled help output**: Custom lipgloss-styled help text at
  `internal/cli/help.go` replaces go-arg's default plain-text output. Flags are
  rendered in cyan, section headers in bold, with consistent column alignment.
  Respects `$NO_COLOR`.

## [0.9.0] - 2026-04-23

### Added

- **Markdown rendering in TUI**: Implemented `charmbracelet/glamour` to
  render LLM summary output with proper markdown styling. Headers, bold,
  italic, code blocks, lists, and other markdown elements are now styled
  for better readability in the terminal.
- **Formalised infobar keybinds**: TUI keybinds are now defined as
  dedicated style variables at the top of the model for easier maintenance.
  Keybinds are bold while separators and action text are subtle but
  distinguishable.
- **Thinking indicator as separate element**: The `[💭 Thinking: ON/OFF]`
  toggle indicator is now on its own line above the keybinds, using a
  distinct style (magenta+bold when ON, subtle+italic when OFF).
- **Dynamic text wrapping on resize**: TUI summary text now re-wraps when
  the terminal window size or font changes, with proper markdown re-rendering.

## [0.8.0] - 2026-04-23

### Added

- **Thinking tags toggle in TUI**: LLMs that output `<think>`/`</think>`
  thinking tags now have those sections automatically removed from the summary
  display by default. Users can press `t` to toggle the thinking section back
  on, displayed in gray at the head of the output with XML tags stripped for
  cleaner presentation. A status indicator shows whether thinking is currently
  visible.

## [0.7.1] - 2026-04-22

### Fixed

- Environment variables were correctly overriding config file values, but CLI
  arguments without explicit flags were also overriding config file values
  unintentionally.
  - Root cause: `go-arg` was populating struct fields with default values
    from `default:` tags even when those CLI flags weren't provided
  - Fix: Removed `default:` tags from Args struct fields for LLM options
    (`model`, `base-url`, `api-key`, `max-tokens`, `temperature`);
    Viper's own default handling is sufficient
  - Fix: Changed Args struct fields from value types to pointer types
    (`*string`, `*int`, `*float64`) so that "not provided on CLI" can be
    distinguished from "provided but zero/empty"
  - Fix: Updated `BindCLIArgs()` and `isZero()` to properly handle pointer
    types and distinguish nil pointers from zero values
- `isZero()` with compound type switch cases: Compound type cases like `case
  *int, *int8, *int16, *int32, *int64` don't properly handle nil comparison when
  stored in an interface variable. Fixed by splitting into individual type cases.
- `isZero()` missing nil interface handling: When a nil pointer is stored
  in an `any` interface, `value != nil` returns `true` (the interface itself
  is not nil). Added early `if v == nil { return true }` check.

## [0.7.0] - 2026-04-22

### Added

- **`--dry-run` option**: Validate configuration and preview what would happen
  without making any LLM API calls. Useful for validating feeds file and
  configuration, checking article counts and filtering results and anything
  else that can be done "offline".
- **Styled terminal output using `lipgloss`**: All CLI output now uses ANSI
  colour codes for better readability:
- **`$NO_COLOR` environment variable support**: Respects the ["no
color"](https://no-color.org/) standard. When `NO_COLOR` is set, all styling is
disabled and plain text is output instead.

### Changed

- API key validation now only occurs when not using `--dry-run` mode

## [0.6.0] - 2026-04-22

### Added

- **Comprehensive unit test suite**, including CLI argument tests
  (`--feeds-file`, `--prompt`, `--api-key`, `--model`, `--base-url`),
  configuration tests, aggregator tests, output formatter tests, processor
  tests and defaults tests.
- **`--base-url` CLI option**: configure custom API endpoints for different LLM
  providers. Should support any OpenAI-compatible provider.
  - Environment variable: `LLM_AGGREGATOR_BASE_URL`
- `internal/defaults` package
- `docs/TESTING.md` which contains a comprehensive testing guide with test
  descriptions

### Changed

- **Configuration defaults**: All packages now use `defaults.Default*` constants
  - `internal/config`: Uses `defaults.Default*` for default values
  - `internal/runtime`: Uses `defaults.Default*` for runtime defaults
  - `internal/llm`: Uses `defaults.Default*` for LLM client defaults

### Fixed

- `nil` pointer safety in aggregator
  - Added nil checks for `progressCtx` across all log statements
  - `FeedAggregator` now works correctly when progress context is nil
  - Tests use `NewFeedAggregatorWithProgress` with explicit progress context

## [0.5.0] - 2026-04-22

### Added

- **Accurate token counting with tiktoken**
  - New `internal/tokeniser` package using OpenAI's tiktoken-go library
  - BPE tokenisation for accurate token counting vs rough character estimation
  - Encoding caching to avoid repeated initialisation overhead
  - Model-aware encoding selection (cl100k_base, o200k_base, etc.)
  - `CountTokens()` and `CountMessagesTokens()` functions
  - Fallback to rough estimation when tiktoken initialisation fails
- **Concurrent feed fetching**: Feeds are now fetched concurrently using
  `golang.org/x/sync/errgroup`
  - Semaphore limits concurrent requests to 10 to avoid server overload
  - Mutex-protected shared state for thread-safe article collection
  - Partial failures don't stop other feeds from being processed
- **Scrollable summary in TUI**
  - Bubble Tea viewport component for browsing long summaries
  - Keyboard navigation: j/k or arrows (scroll), Space/B (page), g/G
  (start/end)
  - Mouse wheel scrolling support
  - Scroll progress indicator showing position and total lines

### Changed

- **Streamlined configuration management**: Viper now uses global instance with
automatic precedence handling. Simplified configuration flow: Parse → GetViper
→ BindCLIArgs → ViperToRuntime
- **TUI colour scheme**: All hex colour codes replaced with ANSI 256-colour
  palette

### Fixed

- Viewport now properly handles scroll events via `Update()`. Summary content
  pre-wrapped to viewport width before display
- Fixed string literal colour names to use actual variables

## [0.4.0] - 2026-04-21

### Changed 

- **TUI is now fully functional and production-ready**: Replaced the
work-in-progress TUI implementation with a robust Bubble Tea command pattern:
  - Runtime execution now uses native `tea.Cmd` instead of manual goroutine
  orchestration.
  - Added real-time progress updates via `progress.Progress` interface with
  stage, substage, and article count messages.
  - TUI displays a spinner, elapsed time, and final summary inline.
  - Removed signal handling and channel synchronisation in favour of Bubble
  Tea's built-in lifecycle.
  - `TUIProgress` now only sends messages to an existing program, simplifying
  integration.
  - Both verbose mode (`SimpleLogger`) and TUI mode share the same
  `progress.Progress` interface.

## [0.3.0] - 2026-04-21

### Added

- **Configuration file support via TOML** (XDG-compliant and cross-platform)
  - Load settings from `~/.config/llm_aggregator/config.toml`
  - Supports all aggregation, API, and output options
  - Example configuration file at `configs/config.example.toml`
- **Environment variable support with `LLM_AGGREGATOR_` prefix**
  - All configuration options can be set via environment variables
  - Precedence order: CLI arguments > environment variables > config file >
  built‑in defaults
- New `internal/config` package with Viper integration
  - `Load()` and `Save()` methods for configuration management
  - `GetConfigPath()` for XDG‑compliant config file location
  - `ConfigExists()` to check for existing configuration
- Comprehensive unit tests for configuration loading, saving, and environment
  variable precedence (`internal/config/config_test.go`)

### Changed

- **Environment variable renamed:** `DEEPSEEK_API_KEY` →
  `LLM_AGGREGATOR_API_KEY`
- **CLI help text** now reflects the new API key environment variable and
  general configuration options
- **Command‑line argument precedence** implemented in `cmd/llm_aggregator/main.go`
via `applyConfiguration()`
- Updated all help text, documentation, and code references
- `README.md` completely revised with a new "Configuration" section

### Removed

- Global `QuietMode` and `VerboseMode` variables from
`internal/config/config.go`. Replaced by runtime‑specific configuration in
`runtime.Runtime` and CLI arguments
- Hardcoded system prompt in `runtime.NewRuntime()`. Now left empty, allowing
configuration or client default to be used
- CLI flags now correctly override config file and environment variables

## [0.2.0] - 2026-04-21

### Changed

- Moved all progress and status messages behind `-v/--verbose` flag
- Default CLI mode is now silent except for final output and errors
- Components (aggregator, processor, LLM client) use logger interface
controlled by verbose flag

## [0.1.0] - 2026-04-21

### Added

- Complete translation from Python prototype to Go implementation
- RSS/Atom feed parsing using `gofeed` library
- DeepSeek API integration via `openai-go` client, configured for `/chat/completions` endpoint
- Content filtering and processing with keyword‑based include/exclude, date filtering, and sorting
- Command‑line interface using `go‑arg` with automatic help and version flags
- Multiple output formats: plain text, GitHub‑flavoured markdown, and JSON
- Terminal user interface (TUI) built with `bubbletea` featuring:
  - Animated progress bar with gradient colours (`#FF7CCB` to `#FDFF8C`)
  - Live article counters (aggregated/processed)
  - Elapsed time display
  - Coloured status indicators using `lipgloss` styling
  - Keyboard controls (q/Ctrl+C to quit)
- Web content extraction fallback using `goquery` when feed descriptions are minimal
- Configurable limits: articles per feed, maximum age, total articles
- Token estimation and API usage logging
- Environment variable support (`LLM_AGGREGATOR_API_KEY`) for authentication
- Example feeds file with technology, programming, and free software sources

### Changed

- Program structure organised into standard Go layout: `cmd/`, `internal/`, `pkg/`
- English spelling conventions maintained throughout (colour, initialise, summarise)
- Error handling improved with specific messages for common API failures (invalid key, rate limits, etc.)
- Progress reporting unified through a `ProgressContext` interface for both TUI and CLI modes

### Fixed

- Initial API endpoint mismatch (using `/responses` instead of `/chat/completions`)
- Nil pointer dereferences when handling optional feed metadata (author, dates)
- String repetition syntax errors in Go (replaced `"="*80` with `strings.Repeat`)
- CLI help/version flag handling to show information without requiring other arguments
- Type compatibility issues with `openai-go` v3 API (message parameters, token usage fields)
