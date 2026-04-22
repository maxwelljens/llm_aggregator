package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/alexflint/go-arg"
)

var (
	Version   string
	BuildDate string
)

// Args represents command-line arguments.
type Args struct {
	FeedsFile string `arg:"--feeds-file,required" help:"Path to file containing RSS feed URLs (one per line)"`
	Prompt    string `arg:"--prompt,required" help:"User prompt for summarisation/analysis"`

	// Feed aggregation options
	MaxArticlesPerFeed *int `arg:"--max-articles-per-feed" help:"Maximum articles to fetch from each feed"`
	MaxDaysOld         *int `arg:"--max-days-old" help:"Only include articles from the last N days"`
	MaxTotalArticles   *int `arg:"--max-total-articles" help:"Maximum total articles to process"`

	// Content filtering
	IncludeKeywords string `arg:"--include-keywords" help:"Comma-separated list of keywords to include (case-insensitive)"`
	ExcludeKeywords string `arg:"--exclude-keywords" help:"Comma-separated list of keywords to exclude (case-insensitive)"`

	// LLM API options
	APIKey      *string  `arg:"--api-key" help:"OpenAI-compatible API key (default: read from LLM_AGGREGATOR_API_KEY env var)"`
	BaseURL     *string  `arg:"--base-url" help:"API base URL"`
	Model       *string  `arg:"--model" help:"LLM model to use"`
	MaxTokens   *int     `arg:"--max-tokens" help:"Maximum tokens in response"`
	Temperature *float64 `arg:"--temperature" help:"Sampling temperature (0.0 to 1.0)"`

	// Output options
	Output          string `arg:"--output" help:"Output format" choice:"text,json,markdown"`
	OutputFile      string `arg:"--output-file" help:"Write output to file (default: stdout)"`
	IncludeArticles bool   `arg:"--include-articles" help:"Include original articles in JSON output"`

	// System options
	SystemPrompt string `arg:"--system-prompt" help:"Custom system prompt for LLM"`
	TUI          bool   `arg:"--tui" help:"Enable TUI interface with progress bar"`
	Verbose      bool   `arg:"-v,--verbose" help:"Show verbose output"`
	ShowVersion  bool   `arg:"--version" help:"Show version"`
	DryRun       bool   `arg:"--dry-run" help:"Validate config, show article statistics, and exit without making LLM API calls"`
}

// Version returns the version string.
func (Args) Version() string {
	return fmt.Sprintf("llm_aggregator v%s (built %s)", Version, BuildDate)
}

// Description returns the program description.
func (Args) Description() string {
	return `LLM Aggregator - Aggregate RSS feeds and summarise with LLM API

Examples:
  # Basic usage with prompts
  llm_aggregator --feeds-file feeds.txt --prompt "What are the latest trends in free software?"
  
  # Validate configuration and preview articles (no LLM API calls)
  llm_aggregator --feeds-file feeds.txt --prompt "Summarise news" --dry-run
  
  # With custom LLM model
  llm_aggregator --feeds-file feeds.txt --prompt "Summarise tech news" --model deepseek-coder
  
  # Output to JSON file
  llm_aggregator --feeds-file feeds.txt --prompt "Analyse AI developments" --output json --output-file analysis.json
  
  # Filter by keywords
  llm_aggregator --feeds-file feeds.txt --prompt "Linux news" --include-keywords linux,opensource

Environment Variables:
  LLM_AGGREGATOR_API_KEY: Your LLM API key (required if not provided via --api-key)`
}

// ParseKeywords parses comma-separated keywords string into list.
func ParseKeywords(keywordString string) []string {
	if keywordString == "" {
		return nil
	}
	keywords := strings.Split(keywordString, ",")
	result := make([]string, 0, len(keywords))
	for _, kw := range keywords {
		if trimmed := strings.TrimSpace(kw); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ParseArgs parses command line arguments.
func ParseArgs() (*Args, error) {
	var args Args
	parser, err := arg.NewParser(arg.Config{
		Program: "llm_aggregator",
	}, &args)
	if err != nil {
		return nil, err
	}

	// Handle help and version flags before checking required fields
	if len(os.Args) > 1 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			parser.WriteHelp(os.Stdout)
			os.Exit(0)
		}
		if os.Args[1] == "--version" {
			fmt.Printf("llm_aggregator v%s (built %s)", Version, BuildDate)
			os.Exit(0)
		}
	}

	err = parser.Parse(os.Args[1:])
	if err != nil {
		return nil, err
	}

	return &args, nil
}

// ToViperMap converts Args to a map for binding to Viper.
// Only non-nil values (explicitly provided CLI flags) are included.
func (a *Args) ToViperMap() map[string]any {
	m := map[string]any{}
	if a.FeedsFile != "" {
		m["feeds_file"] = a.FeedsFile
	}
	if a.MaxArticlesPerFeed != nil {
		m["max_articles_per_feed"] = *a.MaxArticlesPerFeed
	}
	if a.MaxDaysOld != nil {
		m["max_days_old"] = *a.MaxDaysOld
	}
	if a.MaxTotalArticles != nil {
		m["max_total_articles"] = *a.MaxTotalArticles
	}
	if a.IncludeKeywords != "" {
		m["include_keywords"] = a.IncludeKeywords
	}
	if a.ExcludeKeywords != "" {
		m["exclude_keywords"] = a.ExcludeKeywords
	}
	if a.APIKey != nil {
		m["api_key"] = *a.APIKey
	}
	if a.BaseURL != nil {
		m["base_url"] = *a.BaseURL
	}
	if a.Model != nil {
		m["model"] = *a.Model
	}
	if a.MaxTokens != nil {
		m["max_tokens"] = *a.MaxTokens
	}
	if a.Temperature != nil {
		m["temperature"] = *a.Temperature
	}
	if a.Prompt != "" {
		m["prompt"] = a.Prompt
	}
	if a.SystemPrompt != "" {
		m["system_prompt"] = a.SystemPrompt
	}
	if a.Output != "" {
		m["output"] = a.Output
	}
	if a.OutputFile != "" {
		m["output_file"] = a.OutputFile
	}
	if a.IncludeArticles {
		m["include_articles"] = a.IncludeArticles
	}
	return m
}
