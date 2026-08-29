package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/alexflint/go-arg"
)

var (
	Version   string
	BuildDate string
)

// Args defines all command-line flags supported by the program.
type Args struct {
	// Feed input
	FeedsFile string `arg:"-f,--feeds-file" help:"Path to file containing RSS feed URLs (one per line)"`
	Stdin     bool   `arg:"--stdin" help:"Read a single RSS/Atom feed from stdin (can be combined with --feeds-file)"`
	Prompt    string `arg:"-p,--prompt" help:"User prompt for summarisation/analysis (default: built-in summarisation prompt)"`

	// Feed aggregation options
	MaxArticlesPerFeed *int `arg:"-n,--max-articles-per-feed" help:"Maximum articles to fetch from each feed"`
	MaxDaysOld         *int `arg:"-d,--max-days-old" help:"Only include articles from the last N days"`
	MaxTotalArticles   *int `arg:"--max-total-articles" help:"Maximum total articles to process"`

	// Content filtering
	IncludeKeywords string `arg:"-i,--include-keywords" help:"Comma-separated list of keywords to include (case-insensitive)"`
	ExcludeKeywords string `arg:"-e,--exclude-keywords" help:"Comma-separated list of keywords to exclude (case-insensitive)"`

	// LLM API options
	APIKey      *string  `arg:"--api-key" help:"OpenAI-compatible API key (default: read from LLM_AGGREGATOR_API_KEY env var)"`
	BaseURL     *string  `arg:"--base-url" help:"API base URL"`
	Model       *string  `arg:"-m,--model" help:"LLM model to use"`
	MaxTokens   *int     `arg:"--max-tokens" help:"Maximum tokens in response"`
	Temperature *float64 `arg:"--temperature" help:"Sampling temperature (0.0 to 1.0)"`
	Timeout     *int     `arg:"--timeout" help:"LLM request timeout in seconds (default: 300)"`

	// Output options
	Output          string `arg:"-o,--output" help:"Output format" choice:"text,json,markdown"`
	OutputFile      string `arg:"--output-file" help:"Write output to file (default: stdout)"`
	IncludeArticles bool   `arg:"--include-articles" help:"Include original articles in JSON output"`
	Plain           bool   `arg:"-P,--plain" help:"Output only the raw LLM response without any formatting or metadata"`

	// System options
	SystemPrompt string `arg:"--system-prompt" help:"Custom system prompt for LLM"`
	TUI          bool   `arg:"-t,--tui" help:"Enable TUI interface with progress bar"`
	Verbose      bool   `arg:"-v,--verbose" help:"Show verbose output"`
	ShowVersion  bool   `arg:"--version" help:"Show version"`
	DryRun       bool   `arg:"-D,--dry-run" help:"Validate config, show article statistics, and exit without making LLM API calls"`
}

// Version returns the version string (e.g. "llm_aggregator v0.1.0 (built 2025-01-01)").
func (a *Args) Version() string {
	return fmt.Sprintf("llm_aggregator v%s (built %s)", Version, BuildDate)
}

// Description returns the program description.
func (a *Args) Description() string {
	return "LLM Aggregator - Aggregate RSS feeds and summarise with LLM API"
}

// ParseArgs parses command line arguments. The handled return is true when a
// meta-flag (-h/--help, --version) was processed: its output is already
// written and the caller should exit 0. It never calls os.Exit so callers
// keep control of the process lifetime.
func ParseArgs() (*Args, bool, error) {
	var args Args
	handled, err := parseArgs(&args, os.Args, os.Stdout)
	return &args, handled, err
}

func parseArgs(args *Args, argv []string, stdout io.Writer) (bool, error) {
	parser, err := arg.NewParser(arg.Config{
		Program: "llm_aggregator",
	}, args)
	if err != nil {
		return false, err
	}

	// Handle help and version flags before checking required fields
	if len(argv) > 1 {
		if argv[1] == "-h" || argv[1] == "--help" {
			WriteHelp(args, stdout)
			return true, nil
		}
		if argv[1] == "--version" {
			fmt.Fprintf(stdout, "llm_aggregator v%s (built %s)", Version, BuildDate)
			return true, nil
		}
	}

	err = parser.Parse(argv[1:])
	if err != nil {
		return false, err
	}

	return false, nil
}
