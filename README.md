<h1 align="center">llm_aggregator</h1>

<p align="center">
    <img src="assets/logo.svg" alt="LLM Aggregator logo" width=500>
    <br>
    <strong>Aggregate RSS feeds and summarise them with LLMs</strong>
</p>

![Codeberg Release](https://img.shields.io/gitea/v/release/maxwelljensen/llm_aggregator?gitea_url=https%3A%2F%2Fcodeberg.org&style=for-the-badge)
![Codeberg License](assets/eupl-12-badge.svg)

---

## What is it?

`llm_aggregator` fetches articles from multiple RSS/Atom feeds, filters and
processes the content, and sends it to any OpenAI-compatible LLM to produce a
concise summary, without you needing to read dozens or hundreds of posts.

**Supports**: RSS 2.0, Atom, JSON Feed | OpenAI-compatible APIs | Local LLMs
(Ollama, etc.) | TUI with live progress | Text/Markdown/JSON output

---

## Quick start

```bash
# Create a feeds file
cat > feeds.txt << 'EOF'
https://news.ycombinator.com/rss
https://lwn.net/headlines/newrss
EOF

# Run
llm_aggregator --api-key <YOUR_KEY> --base-url <URL> \
-f feeds.txt -p "What are the top tech stories today?"
```

**First run?** See [docs/USAGE.md](docs/USAGE.md) for installation, configuration,
and all available options.

---

## Key features

| | |
|--|--|
| 🚀 | Concurrent feed fetching with rate limiting |
| 🔍 | Keyword filtering (include/exclude, case‑insensitive) |
| 📅 | Date‑based filtering and sorting (date/title/source) |
| 🤖 | Any OpenAI-compatible API (Deepseek, Ollama, OpenRouter, …) |
| 🖥️ | Interactive TUI with progress bar and mouse scrolling |
| 📦 | Config file, environment variables, and CLI flags |
| 📄 | Read feeds directly from stdin via `--stdin` |
| 🔧 | Dry‑run mode to validate config without API calls |

---

## TUI mode

Enable the TUI with `-t` for a colourful progress bar, live article counters,
and elapsed time. The TUI renders LLM output as styled Markdown (headers,
bold, code blocks, lists) and supports keyboard navigation (j/k, arrows, b,
g/G) and mouse wheel scrolling.

![LLM Aggregator action in GIF](./assets/demo.gif)

---

## Architecture

```
Feeds file / stdin
        ↓
  ┌─────────────┐
  │  aggregator │   RSS/Atom/JSON Feed parsing, concurrent fetching
  └─────────────┘
        ↓
  ┌─────────────┐
  │  processor  │   Filter by keywords/age, sort, truncate
  └─────────────┘
        ↓
  ┌─────────────┐
  │     llm     │   OpenAI-compatible API call
  └─────────────┘
        ↓
  ┌─────────────┐
  │   output    │   Text / Markdown / JSON
  └─────────────┘
```

## Configuration precedence

```
CLI flags  >  Environment variables  >  Config file  >  Defaults
```

Create `~/.config/llm_aggregator/config.toml` (see
[docs/USAGE.md](docs/USAGE.md#configuration) for the full reference).

## Building

Detailed build instructions (standard build, goreleaser, cross-compilation,
man page installation, tests, and linting) are in
[docs/BUILD.md](docs/BUILD.md).

---

## Licence

This project is licensed under [European Union Public Licence
1.2](https://joinup.ec.europa.eu/collection/eupl/eupl-text-eupl-12).
