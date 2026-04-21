<h1 align="center">llm_aggregator</h1>

<p align="center">
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

The tool includes a WIP terminal user interface (TUI) built with
`bubbletea` and `lipgloss` that shows a live progress bar, article counts, and
elapsed time while the aggregation runs.

## How do I use `llm_aggregator`?

    llm_aggregator --feeds-file FEEDS-FILE --prompt PROMPT [OPTIONS]...

By default, `llm_aggregator` reads a list of RSS feed URLs, fetches the
articles, filters them by date and keywords, and sends a summary request to
Deepseek. The resulting output is printed to the terminal in your chosen format
(text, markdown, or JSON).

### Basic Options

    --feeds-file FILE         Path to file containing RSS feed URLs (one per line)
    --prompt PROMPT           User prompt for summarisation/analysis
    --api-key KEY             Deepseek API key (default: read from DEEPSEEK_API_KEY env var)
    --model MODEL             Deepseek model to use (default: deepseek-chat)
    --max-tokens N            Maximum tokens in response (default: 4000)
    --output FORMAT           Output format: text, markdown, or json (default: text)
    --output-file FILE        Write output to FILE instead of stdout
    --tui                     Enable TUI interface with progress bar
    --verbose, -v             Enable verbose logging
    --help, -h                Show this help message and exit
    --version                 Show version information and exit

### Filtering & Processing

    --max-articles-per-feed N Maximum articles to fetch from each feed (default: 10)
    --max-days-old N          Only include articles from the last N days (0 for all) (default: 7)
    --max-total-articles N    Maximum total articles to process (default: 20)
    --include-keywords LIST   Comma-separated list of keywords to include (case‑insensitive)
    --exclude-keywords LIST   Comma-separated list of keywords to exclude (case‑insensitive)
    --include-articles        Include original articles in JSON output

### Deepseek Configuration

    --temperature VALUE       Sampling temperature (0.0 to 1.0) (default: 0.7)
    --system-prompt TEXT      Custom system prompt for Deepseek

### Examples

```bash
# Basic usage: summarise tech news from a list of feeds
llm_aggregator --feeds-file feeds.txt \
--prompt "What are the latest AI-related trends in free software?"

# With TUI progress bar
llm_aggregator --feeds-file feeds.txt --prompt "Summarise tech news" --tui

# Output to a JSON file with included articles
llm_aggregator --feeds-file feeds.txt --prompt "Analyse AI developments" \
    --output json --output-file analysis.json --include-articles

# Filter by keywords (only include articles about Linux or open source)
llm_aggregator --feeds-file feeds.txt --prompt "Linux news" \
    --include-keywords linux,opensource --max-days-old 3

# Use a custom Deepseek model and higher token limit
llm_aggregator --feeds-file feeds.txt --prompt "Code analysis" \
    --model deepseek-coder --max-tokens 8000

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
3. Fetch and parse each feed. RSS, Atom, and JSON Feed formats are supported.
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

When the `--tui` flag is used, the entire process is wrapped in a `bubbletea`
TUI that shows a colourful progress bar, live article counters, and elapsed
time (WIP).

## Dependencies

`llm_aggregator` is written in Go and uses the following libraries:

* [`gofeed`](https://github.com/mmcdole/gofeed): a robust RSS/Atom/JSON feed parser
* [`openai‑go`](https://github.com/openai/openai-go): the official OpenAI Go
  SDK, configured to work with Deepseek’s compatible API
* [`bubbletea`](https://github.com/charmbracelet/bubbletea): a TUI framework
  for building terminal applications
* [`lipgloss`](https://github.com/charmbracelet/lipgloss): a library for
  styling terminal output (colours, borders, alignment)
* [`go‑arg`](https://github.com/alexflint/go-arg): struct‑based argument
  parsing with automatic help and version flags
* [`goquery`](https://github.com/PuerkitoBio/goquery): a jQuery‑like HTML
  scraping library (used as a fallback when feed content is minimal)

## How do I build `llm_aggregator`?

`llm_aggregator` can be built with a standard Go toolchain:

    go build ./cmd/llm_aggregator.go

## Configuration file

A configuration file is not yet implemented; all options are passed via
command‑line arguments or environment variables. The API key can be provided
either with `--api-key` or by setting the `DEEPSEEK_API_KEY` environment
variable.

## Example feeds file

Create a file named `feeds.txt` with your favourite RSS feeds, one per line.
For example:

    https://news.ycombinator.com/rss
    https://lwn.net/headlines/newrss
    https://opensource.com/feed
    https://www.phoronix.com/rss.php

Then run:

    llm_aggregator --feeds-file feeds.txt --prompt "Summarise the top tech stories"

## Licence

This project is licensed under [European Union Public Licence
1.2](https://joinup.ec.europa.eu/collection/eupl/eupl-text-eupl-12).
