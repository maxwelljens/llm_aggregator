<h1 align="center">llm_aggregator</h1>

<p align="center">
    <img src="assets/logo.svg" alt="LLM Aggregator logo" width=500>
    <br>
    <strong>A CLI tool to aggregate RSS feeds and summarise them with LLMs</strong>
</p>

![Codeberg Release](https://img.shields.io/gitea/v/release/maxwelljensen/llm_aggregator?gitea_url=https%3A%2F%2Fcodeberg.org&style=for-the-badge)
![Codeberg License](assets/eupl-12-badge.svg)

---

## What is `llm_aggregator`?

`llm_aggregator` is a command‑line utility that fetches articles from multiple
RSS feeds, filters and processes the content, and sends it to an LLM through
OpenAI-compatible API to generate a concise summary or analysis. It’s designed
for keeping up with news and articles from your favourite sources without
having to read dozens or hundreds of individual posts.

## How do I use `llm_aggregator`?

    llm_aggregator --feeds-file FEEDS-FILE --prompt PROMPT [OPTIONS]...

By default, `llm_aggregator` reads a list of RSS feed URLs, fetches the
articles, filters them by date and keywords, and sends a summary request to
the LLM. The resulting output is printed to the terminal in your chosen format
(text, markdown, or JSON).

### Basic Options

    -f, --feeds-file FILE    Path to file containing RSS feed URLs (one per line)   [required]
    -p, --prompt PROMPT      User prompt for summarisation/analysis                 [required]
    --api-key KEY            API key (default: read from $LLM_AGGREGATOR_API_KEY)
    -m, --model MODEL        Model to use (default: deepseek-chat)
    --base-url URL           API base URL (default: https://api.deepseek.com)
    --max-tokens N           Maximum tokens in response (default: 4000)
    -o, --output FORMAT      Output format: text, markdown, or json (default: text)
    --output-file FILE       Write output to FILE instead of STDOUT
    -t, --tui                Enable TUI interface with progress bar
    -D, --dry-run            Validate config, show article statistics, and exit without making LLM API calls
    -v, --verbose            Enable verbose logging
    -h, --help               Show this help message and exit
    --version                Show version information and exit

### Filtering & Processing

    -n, --max-articles-per-feed N  Maximum articles to fetch from each feed (default: 10)
    -d, --max-days-old N           Only include articles from the last N days (0 for all) (default: 7)
    --max-total-articles N         Maximum total articles to process (default: 20)
    -i, --include-keywords LIST    Comma-separated list of keywords to include (case‑insensitive)
    -e, --exclude-keywords LIST    Comma-separated list of keywords to exclude (case‑insensitive)
    --include-articles             Include original articles in JSON output

### LLM Configuration

    --temperature VALUE       Sampling temperature (0.0 to 1.0) (default: 0.7)
    --system-prompt TEXT      Custom system prompt for LLM

### Examples

```bash
# Basic usage: summarise tech news from a list of feeds
llm_aggregator -f feeds.txt -p "What are the latest AI-related trends in free software?"

# With TUI progress bar
llm_aggregator -f feeds.txt -p "Summarise tech news" -t

# Output to a JSON file with included articles
llm_aggregator -f feeds.txt -p "Analyse AI developments" \
    -o json --output-file analysis.json --include-articles


# Filter by keywords (only include articles about Linux or open source)
llm_aggregator -f feeds.txt -p "Linux news" \
    -i linux,opensource -d 3

# Use a custom model and higher token limit
llm_aggregator -f feeds.txt -p "Code analysis" \
    -m deepseek-reasoner --max-tokens 8000

# Use a custom API endpoint (e.g., local Ollama)
llm_aggregator -f feeds.txt -p "Summarise news" \
    --base-url "http://localhost:11434/v1" -m llama3

# Show version information
llm_aggregator --version

# Show help message
llm_aggregator --help
```

## How does `llm_aggregator` work?

`llm_aggregator` performs the following steps for each run:

1. Parse command‑line arguments
2. Read the feeds file: a plain text file containing one RSS/Atom feed URL per
   line.
3. **Fetch and parse feeds concurrently**: RSS, Atom, and JSON Feed formats are
   supported. Feeds are fetched in parallel with rate limiting to maximise
   throughput while avoiding server overload.
4. **Extract article content**: for each feed entry, the tool extracts the
   title, link, publication date, author, and description. If the feed provides
   only a snippet, it can optionally fetch the full webpage using `goquery` to
   extract the main content.
5. **Filter and sort articles**: articles are filtered by age (configurable
   with `--max-days-old`), optionally filtered by keywords (include/exclude),
   and sorted by date, title, or source.
6. Prepare the prompt with selected articles, formatted into a context
   string that is sent to the LLM along with the user’s custom prompt.
7. Call the OpenAI API via the `openai‑go` client.
8. **Format and output the result**: the AI’s response is printed in the chosen
   format (plain text, GitHub‑flavoured markdown, or JSON). If JSON output is
   selected, the original articles can be included alongside the summary.

![LLM Aggregator action in GIF](./assets/demo.gif)

When the `--tui` flag is used, the entire process is wrapped in a `bubbletea`
TUI that shows a colourful progress bar, live article counters, and elapsed
time. The TUI renders Markdown content from the LLM with proper styling for
headers, bold, italic, code blocks, and lists. The TUI supports keyboard
navigation (j/k, arrows, space, b, g/G) and mouse wheel scrolling for
browsing long summaries.

Feeds are fetched concurrently for optimal performance, with rate limiting to
avoid overwhelming feed servers.

## Configuration

`llm_aggregator` supports multiple configuration sources with the following precedence order (highest to lowest):

1. **Command‑line arguments** – Override everything
2. **Environment variables** – Start with `LLM_AGGREGATOR_` prefix
3. **Configuration file** – `~/.config/llm_aggregator/config.toml`
4. **Built‑in defaults**

### Configuration file

Create a TOML file at `~/.config/llm_aggregator/config.toml` with the following structure:

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

All configuration options can also be set via environment variables with the `LLM_AGGREGATOR_` prefix:

| Variable | Description |
|----------|-------------|
| `LLM_AGGREGATOR_API_KEY` | LLM API key |
| `LLM_AGGREGATOR_BASE_URL` | API base URL (default: "https://api.deepseek.com") |
| `LLM_AGGREGATOR_MODEL` | Model name (default: "deepseek-chat") |
| `LLM_AGGREGATOR_MAX_TOKENS` | Maximum tokens in response (default: 4000) |
| `LLM_AGGREGATOR_TEMPERATURE` | Sampling temperature (default: 0.7) |
| `LLM_AGGREGATOR_SYSTEM_PROMPT` | Custom system prompt |
| `LLM_AGGREGATOR_MAX_ARTICLES_PER_FEED` | Maximum articles per feed (default: 10) |
| `LLM_AGGREGATOR_MAX_DAYS_OLD` | Maximum article age in days (default: 7) |
| `LLM_AGGREGATOR_MAX_TOTAL_ARTICLES` | Maximum total articles (default: 20) |
| `LLM_AGGREGATOR_INCLUDE_KEYWORDS` | Comma‑separated include keywords |
| `LLM_AGGREGATOR_EXCLUDE_KEYWORDS` | Comma‑separated exclude keywords |
| `LLM_AGGREGATOR_OUTPUT` | Output format (default: "text") |
| `LLM_AGGREGATOR_OUTPUT_FILE` | Output file path |
| `LLM_AGGREGATOR_INCLUDE_ARTICLES` | Include articles in JSON output (true/false) |

The API key can be provided via `--api‑key`, `LLM_AGGREGATOR_API_KEY` environment variable, or in the configuration file.

## Example feeds file

Create a file named `feeds.txt` with your favourite RSS feeds, one per line.
For example:

    https://news.ycombinator.com/rss
    https://lwn.net/headlines/newrss
    https://opensource.com/feed
    https://www.phoronix.com/rss.php

Then run:

    llm_aggregator -f feeds.txt -p "Summarise the top tech stories"

## Dependencies

`llm_aggregator` is written in Go and uses the following libraries:

| Library | Description |
|---------|-------------|
| [`gofeed`](https://github.com/mmcdole/gofeed) | Robust RSS/Atom/JSON feed parser |
| [`openai‑go`](https://github.com/openai/openai-go) | Official OpenAI API library for Go |
| [`bubbletea`](https://github.com/charmbracelet/bubbletea) | TUI framework for terminal applications |
| [`lipgloss`](https://github.com/charmbracelet/lipgloss) | Library for styling terminal output |
| [`glamour`](https://github.com/charmbracelet/glamour) | Markdown rendering for terminal (used in TUI mode) |
| [`go‑arg`](https://github.com/alexflint/go-arg) | Struct‑based argument parsing |
| [`tiktoken-go`](https://github.com/pkoukk/tiktoken-go) | OpenAI's tiktoken BPE tokeniser |
| [`viper`](https://github.com/spf13/viper) | Configuration management |
| [`goquery`](https://github.com/PuerkitoBio/goquery) | jQuery‑like HTML scraping |

## How do I build `llm_aggregator`?

`llm_aggregator` can be built with a standard Go toolchain:

    go build ./cmd/llm_aggregator.go

For information about the project's test suite, see
[docs/TESTING.md](docs/TESTING.md).

## Licence

This project is licensed under [European Union Public Licence
1.2](https://joinup.ec.europa.eu/collection/eupl/eupl-text-eupl-12).
