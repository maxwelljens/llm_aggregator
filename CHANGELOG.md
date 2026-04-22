# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a
Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
- **Command‑line argument precedence** implemented in `cmd/llm_aggregator.go`
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
