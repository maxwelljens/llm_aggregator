# Building from source

## Prerequisites

- Go 1.21 or later
- `git` (for fetching dependencies)

## Standard build

```bash
# Clone the repository
git clone https://codeberg.org/maxwelljensen/llm_aggregator.git
cd llm_aggregator

# Download dependencies
go mod tidy

# Build the binary
go build ./cmd/llm_aggregator.go

# Run
./llm_aggregator --help
```

The binary is placed in the repository root as `llm_aggregator`.

## Development build with version info

During development builds, version information is injected via ldflags:

```bash
go build -ldflags "
  -s -w
  -X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')
  -X main.buildDate=$(date -u +%Y-%m-%d)
" ./cmd/llm_aggregator.go
```

The `--version` flag will then report the current git tag + commit hash.

## Release builds (goreleaser)

The project uses [goreleaser](https://goreleaser.com/) for cross-platform
release builds. The `.goreleaser.yaml` configuration builds for:

| OS | Architectures |
|----|---------------|
| Linux | `386`, `amd64`, `arm64` |
| Windows | `386`, `amd64`, `arm64` |
| Darwin (macOS) | `amd64`, `arm64` |

```bash
# Install goreleaser (once)
go install github.com/goreleaser/goreleaser/v2@latest

# Build release binaries locally
goreleaser build --clean
```

Release artifacts are written to `dist/`.

## Cross-compilation

Go makes cross-compilation straightforward:

```bash
# Linux on macOS (or vice versa)
GOOS=linux GOARCH=amd64 go build ./cmd/llm_aggregator.go

# Windows
GOOS=windows GOARCH=amd64 go build -o llm_aggregator.exe ./cmd/llm_aggregator.go

# ARM64 Linux
GOOS=linux GOARCH=arm64 go build ./cmd/llm_aggregator.go
```

## Running tests

```bash
# All tests
go test ./...

# With race detector
go test ./... -race

# With coverage
go test ./... -cover
```

## Linting

The project uses [golangci-lint](https://golangci-lint.run/):

```bash
# Run linter
golangci-lint run

# Auto-fix formatting issues
gofmt -w .
```

## Man pages

`man` pages are in `docs/`:

- `docs/llm_aggregator.1` — command reference
- `docs/llm_aggregator.5` — configuration file reference

To install them system-wide:

```bash
# Copy man pages to your man directory
cp docs/llm_aggregator.1 /usr/local/share/man/man1/
cp docs/llm_aggregator.5 /usr/local/share/man/man5/

# Rebuild the whatis database (if needed)
mandb

# View the man page
man llm_aggregator
```

To add them to your personal `$MANPATH`:

```bash
# Add to shell profile (~/.bashrc, ~/.zshrc, etc.)
export MANPATH="$MANPATH:$HOME/.local/share/man"

# Create the directories and copy man pages
mkdir -p ~/.local/share/man/man1
mkdir -p ~/.local/share/man/man5
cp docs/llm_aggregator.1 ~/.local/share/man/man1/
cp docs/llm_aggregator.5 ~/.local/share/man/man5/

# Now you can run:
man llm_aggregator     # section 1 (commands)
man 5 llm_aggregator   # section 5 (config file)
```

## Verifying the build

After building, verify the binary works:

```bash
# Show version and default config
./llm_aggregator --version

# Run a dry-run (no API key needed)
./llm_aggregator -f feeds.txt -p "test" -D --verbose
```

Where `feeds.txt` is a real feeds file to test end-to-end.

## Dependencies

| Library | Purpose |
|---------|---------|
| [`gofeed`](https://github.com/mmcdole/gofeed) | RSS/Atom/JSON feed parsing |
| [`openai-go`](https://github.com/openai/openai-go) | OpenAI-compatible API calls |
| [`bubbletea`](https://github.com/charmbracelet/bubbletea) | TUI framework |
| [`lipgloss`](https://github.com/charmbracelet/lipgloss) | Terminal styling |
| [`glamour`](https://github.com/charmbracelet/glamour) | Markdown rendering (TUI) |
| [`go-arg`](https://github.com/alexflint/go-arg) | CLI argument parsing |
| [`tiktoken-go`](https://github.com/pkoukk/tiktoken-go) | BPE token counting |
| [`viper`](https://github.com/spf13/viper) | Configuration management |
| [`goquery`](https://github.com/PuerkitoBio/goquery) | HTML scraping for article content |
