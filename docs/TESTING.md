# Testing guide

This document explains the testing strategy and test coverage for the LLM
Aggregator project.

## Running tests

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test ./... -v

# Run tests for a specific package
go test ./internal/aggregator/...

# Run tests with coverage
go test ./... -cover
```

## Linting

The project uses [golangci-lint](https://golangci-lint.run/) for static analysis.
Configuration is in `.golangci.yml`.

```bash
# Run the linter
golangci-lint run

# Auto-fix formatting issues
gofmt -w .
```

## Test packages

### `internal/cli`: CLI parsing

Tests for command-line argument handling using the `go-arg` library.

| Test | Description |
|------|-------------|
| `TestFeedsFileParsing` | Verifies `--feeds-file` accepts valid file paths and rejects non-existent files |
| `TestPromptParsing` | Tests `--prompt` handling including empty strings and special characters |
| `TestAPIKeyParsing` | Verifies `--api-key` parsing and environment variable fallback |
| `TestModelParsing` | Tests `--model` with default value (`deepseek-chat`) and custom models |
| `TestBaseURLParsing` | Tests `--base-url` with various API endpoints |
| `TestArgsToViperMap` | Ensures CLI arguments convert correctly to Viper configuration |
| `TestParseKeywords` | Tests keyword parsing from comma-separated strings |

Key features tested are argument validation and default value handling.

---

### `internal/config`: Configuration management

Tests for the Viper-based configuration system.

| Test | Description |
|------|-------------|
| `TestConfigParsingAlwaysPasses` | Ensures CLI argument parsing succeeds for valid inputs |
| `TestConfigLoadWithVariousSources` | Tests loading from config files and environment variables |
| `TestViperToConfigConversion` | Verifies conversion from Viper to Config struct |
| `TestBindCLIArgs` | Tests binding CLI arguments to Viper |
| `TestDefaultConfig` | Verifies default configuration values |
| `TestConfigPath` | Tests XDG-compliant config path resolution |
| `TestConfigSaveAndLoad` | Tests config file save/load round-trip |
| `TestConfigExists` | Tests config file existence checking |
| `TestEnvironmentVariables` | Tests environment variable loading |
| `TestIsZero` | Tests `isZero()` helper with all types including pointer types |
| `TestViperToRuntimePrecedence` | Tests configuration precedence: CLI > env vars > config file > defaults |
| `TestBindCLIArgsWithPointers` | Tests that nil/zero CLI args don't override existing values |

**Configuration precedence** (highest to lowest):
1. CLI arguments
2. Environment variables (`LLM_AGGREGATOR_*`)
3. Config file (`~/.config/llm_aggregator/config.toml`)
4. Default values

> [!IMPORTANT]
> CLI arguments only override config file values when explicitly provided. If
> `--model` is not passed on the command line, the config file or environment
> variable value is used. This prevents CLI defaults from overriding
> configuration file values.

---

### `internal/defaults`: Default values

Tests for central default constants.

| Test | Description |
|------|-------------|
| `TestDefaultValues` | Verifies all default constants are set correctly |
| `TestDefaultSystemPrompt` | Tests the default summarisation prompt |
| `TestDefaultValuesAreSensible` | Validates ranges (e.g., temperature 0-2, max articles > 0) |
| `TestDefaultBaseURLIsValid` | Ensures URL format is valid |
| `TestOutputFormatChoices` | Verifies supported output formats |

**Default Values:**
| Setting | Default | Description |
|---------|---------|-------------|
| `DefaultModel` | `deepseek-chat` | LLM model to use |
| `DefaultBaseURL` | `https://api.deepseek.com` | API endpoint |
| `DefaultMaxTokens` | `4000` | Maximum tokens per request |
| `DefaultTemperature` | `0.7` | Sampling temperature (0-2) |
| `DefaultMaxArticlesPerFeed` | `10` | Articles per feed |
| `DefaultMaxDaysOld` | `7` | Maximum article age in days |
| `DefaultMaxTotalArticles` | `20` | Total articles limit |
| `DefaultOutput` | `text` | Output format |
| `DefaultIncludeArticles` | `false` | Include articles in output |
| `DefaultLLMTimeout` | `300` | LLM request timeout (seconds) |

---

### `internal/llm`: LLM client

Tests for the LLM client constructor, defaults, and configuration.

| Test | Description |
|------|-------------|
| `TestNewLLMClientTimeout` | Verifies `timeoutSeconds=0` defaults to 300; custom values are stored |
| `TestNewLLMClientDefaults` | Verifies model, maxTokens, temperature, and llmTimeout all get defaults |
| `TestNewLLMClientRequiresAPIKey` | Verifies missing API key returns a descriptive error |

**Client field naming:** The `LLMClient` struct uses `llmTimeout int` (seconds; 0 means no timeout) to avoid confusion with `http.Client.Timeout` which operates at the transport layer.

---

### `internal/aggregator`: Feed aggregation

Tests for feed parsing, article extraction, and filtering.

| Test | Description |
|------|-------------|
| `TestNewFeedAggregator` | Tests aggregator creation with various configurations |
| `TestNewFeedAggregatorWithProgress` | Tests with progress context |
| `TestParseFeedsFromFile` | Verifies parsing feeds file with comments |
| `TestParseFeedsFromFileEmpty` | Tests empty and comment-only files |
| `TestParseFeedsFromFileNotFound` | Verifies error handling for missing files |
| `TestParseFeedFromReader` | Tests parsing RSS XML from any `io.Reader` (used by stdin) |
| `TestParseFeedFromReaderAtom` | Tests parsing Atom XML from `io.Reader` |
| `TestParseFeedFromReaderMalformed` | Tests error handling for malformed XML |
| `TestParseFeedFromStdin` | Tests reading a single RSS/Atom feed from stdin via `os.Pipe()` |
| `TestArticleAgeFiltering` | Tests date-based article filtering |
| `TestArticleSorting` | Tests date sorting (ascending/descending) |
| `TestArticleLinkExtraction` | Verifies link extraction |
| `TestArticleContentExtraction` | Tests content truncation |
| `TestArticleToMap` | Tests article-to-map conversion |
| `TestArticleStruct` | Tests Article struct methods |
| `TestEmptyFeedsFile` | Tests handling of empty feeds file |

**Mock data**

```xml
<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>Test Feed</title>
    <item>
      <title>Article One</title>
      <link>https://example.com/article1</link>
      <pubDate>Wed, 15 Jan 2024 10:00:00 GMT</pubDate>
    </item>
  </channel>
</rss>
```

---

### `internal/output`: Summary formatting

Tests for JSON, Markdown, and Text output formatting.

| Test | Description |
|------|-------------|
| `TestNewFormatter` | Tests formatter creation with valid/invalid formats |
| `TestFormatDataJSON` | Verifies JSON output structure |
| `TestFormatDataMarkdown` | Tests Markdown formatting |
| `TestFormatDataText` | Tests plain text formatting |
| `TestFormatDataWithArticles` | Verifies articles appear in output |
| `TestFormatDataMarkdownWithArticles` | Tests article formatting in Markdown |
| `TestFormatDataDefaultValues` | Tests default values for missing fields |
| `TestGetStringHelper` | Tests string extraction from map |
| `TestGetIntHelper` | Tests integer extraction from map |
| `TestCenterText` | Tests text centering |
| `TestFormatArticleList` | Tests article list formatting |
| `TestJSONIndentation` | Verifies proper JSON indentation |

**Output Formats:**

**JSON:**
```json
{
  "title": "LLM Aggregator Summary",
  "prompt": "Summarise the news",
  "model": "deepseek-chat",
  "articles_count": 5,
  "timestamp": "2024-01-15T10:00:00Z",
  "summary": "...",
  "articles": [...]
}
```

**Markdown:**
```markdown
# LLM Aggregator Summary

## Metadata
- **Prompt**: Summarise the news
- **Model**: deepseek-chat
- **Articles Analysed**: 5
- **Generated**: 2024-01-15T10:00:00Z

## Summary
...

## Articles Analysed
### Article 1: Title
**Source**: News Feed
**Link**: [URL](URL)
```

**Text:**
```
================================================================
                          Title
================================================================
METADATA
----------------------------------------
Prompt: Summarise the news
Model: deepseek-chat
...
================================================================
                       End of Summary
================================================================
```

---

### `internal/processor`: Content processing

Tests for article filtering, sorting, and preparation for LLM summarisation.

| Test | Description |
|------|-------------|
| `TestNewContentProcessor` | Tests processor creation |
| `TestProcessArticles` | Tests overall article processing pipeline |
| `TestFilterArticlesByIncludeKeywords` | Tests keyword inclusion filtering |
| `TestFilterArticlesByExcludeKeywords` | Tests keyword exclusion filtering |
| `TestFilterArticlesCaseInsensitive` | Verifies case-insensitive matching |
| `TestSortArticlesByTitle` | Tests title sorting (A-Z, Z-A) |
| `TestSortArticlesBySource` | Tests feed name sorting |
| `TestPrepareForLLM` | Verifies article preparation for LLM |
| `TestContentTruncation` | Tests content truncation |
| `TestSetLogger` | Tests logger configuration |
| `TestUnknownSortField` | Tests default sorting for unknown fields |
| `TestMixedIncludeExclude` | Tests combined include/exclude filtering |
| `TestEmptyKeywordLists` | Tests handling of empty keyword lists |

**Filtering Logic:**

```
Original Articles
      ↓
┌─────────────────────────────┐
│  1. Check exclude keywords  │
│     - If match → EXCLUDE    │
│     - If no match → NEXT    │
└─────────────────────────────┘
      ↓
┌─────────────────────────────┐
│  2. Check include keywords  │
│     - If no filter → INCLUDE│
│     - If match → INCLUDE    │
│     - If no match → EXCLUDE │
└─────────────────────────────┘
      ↓
Filtered Articles
      ↓
┌─────────────────────────────┐
│  3. Sort (date/title/source) │
└─────────────────────────────┘
      ↓
┌─────────────────────────────┐
│  4. Limit to maxTotalArticles│
└─────────────────────────────┘
      ↓
┌─────────────────────────────┐
│  5. Truncate content        │
└─────────────────────────────┘
      ↓
Prepared Articles
```

**Sorting Options:**
| Value | Behaviour |
|-------|----------|
| `date` | Sort by publication date (default) |
| `title` | Sort alphabetically by title |
| `source` | Sort by feed name |

Keyword matching is case-insensitive.

---

### `internal/runtime`: Execution pipeline

Tests for the full pipeline execution flow.

| Test | Description |
|------|-------------|
| `TestExecuteBranchStdinOnly` | Tests `Execute()` with stdin feed only |
| `TestExecuteBranchFeedsFileOnly` | Tests `Execute()` with feeds file only |
| `TestExecuteBranchCollated` | Tests `Execute()` with both stdin and feeds file |
| `TestExecuteBranchNoSource` | Tests `Execute()` returns error when no source specified |

**Branch logic in `Execute()`:**
```
if Stdin && FeedsFile != "" → collate stdin then file
else if Stdin           → stdin only
else if FeedsFile != "" → file only
else                   → error: no feed source
```

---

### `internal/signals`: Signal handling

Tests for graceful shutdown on `SIGINT`, `SIGTERM`, and `SIGHUP`.

| Test | Description |
|------|-------------|
| `TestSignalHandler_Watch` | Tests `Watch()` starts and `Stop()` cleans up |
| `TestSignalHandler_IsExiting` | Tests `IsExiting()` is false after clean `Stop()` |
| `TestSignalHandler_Stop_Idempotent` | Tests `Stop()` is safe to call multiple times |

**Signal handling flow:**
```
SIGINT/SIGTERM/SIGHUP
  → signal.Notify intercepts (no default exit)
  → sh.notify channel receives signal
  → Watch() goroutine: sh.exiting.Store(true)
  → polling goroutine: cancel() called
  → HTTP request aborted via context cancellation
  → rt.Execute returns → partial output → exit 130
```

---

## Test utilities

### Helper functions

| Package | Function | Purpose |
|---------|----------|---------|
| `aggregator` | `createTestArticles()` | Creates sample articles with dates |
| `aggregator` | `writeTempFile()` | Creates temporary test files |
| `processor` | `containsString()` | Checks if string exists in slice |
| `output` | `getString()` | Extracts string from map with default |
| `output` | `getInt()` | Extracts int from map with default |
| `output` | `centerText()` | Centres text within width |

### Mock data

| Package | Data | Purpose |
|---------|------|---------|
| `aggregator` | `validRSSFeed` | Valid RSS XML for parsing tests |
| `aggregator` | `atomFeed` | Atom XML for compatibility tests |
| `aggregator` | `malformedRSSFeed` | Malformed XML for error handling |

---

## Coverage goals

Current coverage status:

| Package | Coverage | Notes |
|---------|----------|-------|
| `cli` | 95% | All argument types and help formatting tested |
| `config` | 82% | Most config operations tested |
| `aggregator` | 57% | Full parsing pipeline tested; network calls require advanced mocks |
| `output` | 93% | All formatters tested |
| `processor` | 76% | All filtering/sorting tested |
| `runtime` | 0% | Integration-only; context propagation not exercised by unit tests |
| `signals` | 89% | Handler lifecycle and concurrency tested |
| `llm` | 22% | Constructor defaults, API key validation, and timeout field configuration |
| `tui` | 0% | Requires terminal interaction |
| `progress` | 0% | Interface only |
| `tokeniser` | 0% | Depends on library internals |
| `cmd` | 0% | Entry point |

> [!NOTE]
> `defaults` and `style` have no testable statements; they contain only
> constant declarations or subjective parameters.

---

## Writing new tests

### Test naming convention

```go
func Test<Component>_<Action>(t *testing.T) { ... }
```

Examples:
- `TestNewFeedAggregator`: Test constructor
- `TestFilterArticlesByExcludeKeywords`: Test filter with exclude
- `TestFormatDataJSON`: Test JSON formatting

### Table-driven tests

Prefer table-driven tests for multiple test cases:

```go
func TestExample(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"empty string", "", ""},
        {"single word", "hello", "hello"},
        {"multiple words", "hello world", "hello world"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Function(tt.input)
            if result != tt.expected {
                t.Errorf("got %q, want %q", result, tt.expected)
            }
        })
    }
}
```

### Using temporary files

```go
func TestWithTempFile(t *testing.T) {
    tmpDir := t.TempDir()  // Go 1.15+
    tmpFile := tmpDir + "/test.txt"
    
    if err := os.WriteFile(tmpFile, []byte("content"), 0644); err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }
    
    // Test using tmpFile
}
```

### Testing `nil` safety

Always test `nil`/empty cases:

```go
t.Run("nil input", func(t *testing.T) {
    result := ProcessArticles(nil, "date", false)
    if len(result) != 0 {
        t.Error("Expected empty result for nil input")
    }
})

t.Run("empty slice", func(t *testing.T) {
    result := ProcessArticles([]*Article{}, "date", false)
    if len(result) != 0 {
        t.Error("Expected empty result for empty slice")
    }
})
```

---

## Continuous integration

Tests should pass before committing:

```bash
# Verify all tests pass
go test ./...

# Verify with race detector
go test ./... -race

# Verify with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```
