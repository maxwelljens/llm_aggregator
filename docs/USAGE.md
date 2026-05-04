# Usage guide

## Installation

```bash
# From source
go build .
```

Or grab the latest binary in the
[Releases](https://codeberg.org/maxwelljensen/llm_aggregator/releases) section.

## Man pages

`man`` pages ship with the project in `docs/`:

| File | Section | Description |
|------|---------|-------------|
| `docs/llm_aggregator.1` | 1 | Full command reference |
| `docs/llm_aggregator.5` | 5 | Configuration file reference |

Install to your `$MANPATH`, then run:

```bash
man llm_aggregator    # section 1: command options
man 5 llm_aggregator  # section 5: config file format
```

## Basic usage

```bash
llm_aggregator --feeds-file FEEDS-FILE --prompt PROMPT [OPTIONS]...
```

`llm_aggregator` reads a list of RSS feed URLs, fetches the articles, filters
them by date and keywords, and sends a summary request to the LLM. The
result is printed to the terminal in your chosen format (text, markdown, or
JSON).

---

## Feed sources

### Feeds file

Create a plain text file with one RSS/Atom feed URL per line. Lines starting
with `#` are comments; empty lines are skipped.

```bash
# Example feeds.txt
https://news.ycombinator.com/rss
https://lwn.net/headlines/newrss

# Linux news
https://opensource.com/feed
https://www.phoronix.com/rss.php
```

### stdin

Read a single RSS/Atom feed directly from stdin:

```bash
curl -s https://example.com/feed.xml | llm_aggregator --stdin -p "Summarise"
```

### Combining both

```bash
llm_aggregator -f feeds.txt --stdin -p "Summarise all"
```

---

## Command reference

### Required

| Flag | Description |
|------|-------------|
| `-f, --feeds-file FILE` | Path to file containing RSS feed URLs (one per line) |
| `-p, --prompt PROMPT` | User prompt for summarisation/analysis |

> [!NOTE]
> `-f/--feeds-file` is required only if not using `--stdin`

### Feed & filtering options

| Flag | Default | Description |
|------|---------|-------------|
| `-f, --feeds-file FILE` | — | Path to file containing RSS feed URLs (one per line) |
| `--stdin` | `false` | Read a single RSS/Atom feed from stdin (can be combined with `-f`) |
| `-n, --max-articles-per-feed N` | `10` | Maximum articles to fetch from each feed |
| `-d, --max-days-old N` | `7` | Only include articles from the last N days (0 for all) |
| `--max-total-articles N` | `20` | Maximum total articles to process |
| `-i, --include-keywords LIST` | — | Comma-separated keywords to include (case-insensitive) |
| `-e, --exclude-keywords LIST` | — | Comma-separated keywords to exclude (case-insensitive) |

### LLM API options

| Flag | Default | Description |
|------|---------|-------------|
| `--api-key KEY` | `$LLM_AGGREGATOR_API_KEY` | API key |
| `-m, --model MODEL` | `deepseek-chat` | Model name |
| `--base-url URL` | `https://api.deepseek.com` | API base URL |
| `--max-tokens N` | `4000` | Maximum tokens in response |
| `--temperature VALUE` | `0.7` | Sampling temperature (0.0 to 1.0) |
| `--timeout N` | `300` | LLM request timeout in seconds |
| `--system-prompt TEXT` | — | Custom system prompt for LLM |

### Output options

| Flag | Default | Description |
|------|---------|-------------|
| `-o, --output FORMAT` | `text` | Output format: `text`, `markdown`, or `json` |
| `--output-file FILE` | — | Write output to FILE instead of STDOUT |
| `--include-articles` | `false` | Include original articles in JSON output |
| `-P, --plain` | `false` | Output only the raw LLM response without formatting or metadata |

### Interface options

| Flag | Description |
|------|-------------|
| `-t, --tui` | Enable TUI interface with progress bar |
| `-v, --verbose` | Enable verbose logging |
| `-D, --dry-run` | Validate config, show article statistics, and exit without making LLM API calls |

### Other

| Flag | Description |
|------|-------------|
| `-h, --help` | Show help message and exit |
| `--version` | Show version information and exit |

---

## Examples

### Basic summarisation

```bash
llm_aggregator -f feeds.txt -p "What are the latest AI-related trends in free software?"
```

### Custom model and higher token limit

```bash
llm_aggregator -f feeds.txt -p "Code analysis" \
    -m deepseek-reasoner --max-tokens 8000
```

### Local Ollama instance

```bash
llm_aggregator -f feeds.txt -p "Summarise news" \
    --base-url "http://localhost:11434/v1" -m llama3
```

### Keyword filtering

Include only articles about Linux or open source:

```bash
llm_aggregator -f feeds.txt -p "Linux news" \
    -i linux,opensource -d 3
```

Exclude articles about Windows:

```bash
llm_aggregator -f feeds.txt -p "Tech news" \
    -e windows,microsoft
```

### JSON output with articles

```bash
llm_aggregator -f feeds.txt -p "Analyse AI developments" \
    -o json --output-file analysis.json --include-articles
```

### TUI mode

```bash
llm_aggregator -f feeds.txt -p "Summarise tech news" -t
```

---

## Configuration

`llm_aggregator` supports multiple configuration sources with the following
precedence order (highest to lowest):

1. **Command-line arguments** — Override everything
2. **Environment variables** — Start with `LLM_AGGREGATOR_` prefix
3. **Configuration file** — `~/.config/llm_aggregator/config.toml`
4. **Built-in defaults**

### Configuration file

Create a TOML file at `~/.config/llm_aggregator/config.toml`:

```toml
# Feed aggregation options
max_articles_per_feed = 10
max_days_old = 7
max_total_articles = 20

# Content filtering (comma-separated keywords)
# include_keywords = "linux,opensource"
# exclude_keywords = "windows,microsoft"

# LLM API options
# api_key = "your_api_key_here"  # Can also be set via LLM_AGGREGATOR_API_KEY env var
# base_url = "https://api.deepseek.com"  # Optional custom API endpoint
model = "deepseek-chat"
max_tokens = 4000
temperature = 0.7

# System prompt for LLM API
system_prompt = """You are an expert analyst and summariser.
You analyse content from multiple sources and provide
concise, insightful summaries based on user requests.
Focus on key points, trends, and important information."""

# Output options
output = "text"  # Options: text, json, markdown
# output_file = ""  # Optional output file path
include_articles = false
```

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LLM_AGGREGATOR_API_KEY` | — | LLM API key |
| `LLM_AGGREGATOR_BASE_URL` | `https://api.deepseek.com` | API base URL |
| `LLM_AGGREGATOR_MODEL` | `deepseek-chat` | Model name |
| `LLM_AGGREGATOR_MAX_TOKENS` | `4000` | Maximum tokens in response |
| `LLM_AGGREGATOR_TEMPERATURE` | `0.7` | Sampling temperature |
| `LLM_AGGREGATOR_TIMEOUT` | `300` | LLM request timeout in seconds |
| `LLM_AGGREGATOR_SYSTEM_PROMPT` | — | Custom system prompt |
| `LLM_AGGREGATOR_MAX_ARTICLES_PER_FEED` | `10` | Maximum articles per feed |
| `LLM_AGGREGATOR_MAX_DAYS_OLD` | `7` | Maximum article age in days |
| `LLM_AGGREGATOR_MAX_TOTAL_ARTICLES` | `20` | Maximum total articles |
| `LLM_AGGREGATOR_INCLUDE_KEYWORDS` | — | Comma-separated include keywords |
| `LLM_AGGREGATOR_EXCLUDE_KEYWORDS` | — | Comma-separated exclude keywords |
| `LLM_AGGREGATOR_OUTPUT` | `text` | Output format |
| `LLM_AGGREGATOR_OUTPUT_FILE` | — | Output file path |
| `LLM_AGGREGATOR_INCLUDE_ARTICLES` | `false` | Include articles in JSON output |
| `LLM_AGGREGATOR_PLAIN` | `false` | Plain output without metadata |
| `LLM_AGGREGATOR_STDIN` | `false` | Read RSS/Atom feed from stdin |
| `LLM_AGGREGATOR_FEEDS_FILE` | — | Feeds file path |
| `LLM_AGGREGATOR_TUI` | `false` | Enable TUI mode |
| `LLM_AGGREGATOR_DRY_RUN` | `false` | Dry-run mode |
| `LLM_AGGREGATOR_VERBOSE` | `false` | Verbose logging |
