# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a
Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to
[Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
- Environment variable support (`DEEPSEEK_API_KEY`) for authentication
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