package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"llm_aggregator/internal/style"
)

type HelpOption struct {
	Short       string
	Name        string
	Description string
	Default     string
	Type        string
	Required    bool
}

type HelpSection struct {
	Title   string
	Options []HelpOption
}

var (
	headerStyle = lipgloss.NewStyle().Bold(true)
	flagStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color(style.Cyan))
)

func shouldStyle() bool {
	return !style.NoColor()
}

func renderFlag(short, long string) string {
	if short != "" {
		return fmt.Sprintf("%s, %s", short, long)
	}
	return long
}

func getOptionValue(opt HelpOption) string {
	if opt.Type != "" {
		val := opt.Type
		if opt.Default != "" {
			val = fmt.Sprintf("%s (default: %s)", opt.Type, opt.Default)
		}
		return val
	}
	if opt.Default != "" {
		return opt.Default
	}
	return ""
}

func maxFlagWidth(sections []HelpSection) int {
	maxWidth := 0
	for _, section := range sections {
		for _, opt := range section.Options {
			flagStr := renderFlag(opt.Short, opt.Name)
			width := lipgloss.Width(flagStr)
			maxWidth = max(maxWidth, width)
		}
	}
	return maxWidth
}

func formatSection(section HelpSection, flagWidth int, _ int) string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(headerStyle.Render(section.Title))
	b.WriteString("\n")

	for _, opt := range section.Options {
		flagStr := renderFlag(opt.Short, opt.Name)
		padding := max(flagWidth-lipgloss.Width(flagStr), 0)

		if shouldStyle() {
			b.WriteString(flagStyle.Render(flagStr))
		} else {
			b.WriteString(flagStr)
		}
		b.WriteString(strings.Repeat(" ", padding+2))

		value := getOptionValue(opt)
		if value != "" {
			if shouldStyle() {
				b.WriteString(flagStyle.Render(value))
			} else {
				b.WriteString(value)
			}
			b.WriteString("  ")
		}

		if opt.Required {
			b.WriteString("[required] ")
		}

		b.WriteString(opt.Description)
		b.WriteString("\n")
	}

	return b.String()
}

func BuildStyledHelp(args *Args) string {
	var b strings.Builder

	b.WriteString(headerStyle.Render("llm_aggregator"))
	b.WriteString("\n\n")

	b.WriteString(args.Description())
	b.WriteString("\n\n")

	b.WriteString(headerStyle.Render("FLAGS"))
	b.WriteString("\n")

	sections := []HelpSection{
		{
			Title: "Required Arguments",
			Options: []HelpOption{
				{
					Short:       "-p",
					Name:        "--prompt PROMPT",
					Description: "User prompt for summarisation/analysis",
					Required:    true,
				},
			},
		},
		{
			Title: "Feed Input",
			Options: []HelpOption{
				{
					Short:       "-f",
					Name:        "--feeds-file FILE",
					Description: "Path to file containing RSS feed URLs (one per line)",
				},
				{
					Name:        "--stdin",
					Description: "Read a single RSS/Atom feed from stdin (can be combined with --feeds-file)",
				},
			},
		},
		{
			Title: "Feed Aggregation",
			Options: []HelpOption{
				{
					Short:       "-n",
					Name:        "--max-articles-per-feed N",
					Description: "Maximum articles to fetch from each feed",
					Default:     "10",
				},
				{
					Short:       "-d",
					Name:        "--max-days-old N",
					Description: "Only include articles from the last N days (0 for all)",
					Default:     "7",
				},
				{
					Name:        "--max-total-articles N",
					Description: "Maximum total articles to process across all feeds",
					Default:     "20",
				},
			},
		},
		{
			Title: "Content Filtering",
			Options: []HelpOption{
				{
					Short:       "-i",
					Name:        "--include-keywords LIST",
					Description: "Comma-separated keywords to include (case-insensitive)",
				},
				{
					Short:       "-e",
					Name:        "--exclude-keywords LIST",
					Description: "Comma-separated keywords to exclude (case-insensitive)",
				},
			},
		},
		{
			Title: "LLM API Options",
			Options: []HelpOption{
				{
					Name:        "--api-key KEY",
					Description: "OpenAI-compatible API key (default: read from $LLM_AGGREGATOR_API_KEY)",
				},
				{
					Name:        "--base-url URL",
					Description: "API base URL",
					Default:     "https://api.deepseek.com",
				},
				{
					Short:       "-m",
					Name:        "--model MODEL",
					Description: "LLM model to use",
					Default:     "deepseek-chat",
				},
				{
					Name:        "--max-tokens N",
					Description: "Maximum tokens in LLM response",
					Default:     "4000",
				},
				{
					Name:        "--temperature VALUE",
					Description: "Sampling temperature (0.0 to 1.0)",
					Default:     "0.7",
				},
				{
					Name:        "--timeout N",
					Description: "LLM request timeout in seconds (default: 300)",
					Default:     "300",
				},
				{
					Name:        "--system-prompt TEXT",
					Description: "Custom system prompt for LLM",
				},
			},
		},
		{
			Title: "Output Options",
			Options: []HelpOption{
				{
					Short:       "-o",
					Name:        "--output FORMAT",
					Description: "Output format",
					Type:        "text|markdown|json",
					Default:     "text",
				},
				{
					Name:        "--output-file FILE",
					Description: "Write output to FILE instead of stdout",
				},
				{
					Name:        "--include-articles",
					Description: "Include original articles in JSON output",
				},
				{
					Short:       "-P",
					Name:        "--plain",
					Description: "Output only the raw LLM response without any formatting or metadata",
				},
			},
		},
		{
			Title: "Interface Options",
			Options: []HelpOption{
				{
					Short:       "-t",
					Name:        "--tui",
					Description: "Enable TUI interface with progress bar, live counters, and elapsed time",
				},
				{
					Short:       "-D",
					Name:        "--dry-run",
					Description: "Validate config, show article statistics, and exit without making LLM API calls",
				},
				{
					Short:       "-v",
					Name:        "--verbose",
					Description: "Enable verbose logging",
				},
			},
		},
		{
			Title: "General Options",
			Options: []HelpOption{
				{
					Name:        "--version",
					Description: "Show version information and exit",
				},
				{
					Short:       "-h",
					Name:        "--help",
					Description: "Show this help message and exit",
				},
			},
		},
	}

	flagWidth := maxFlagWidth(sections)

	for _, section := range sections {
		b.WriteString(formatSection(section, flagWidth, 0))
	}

	b.WriteString("\n")
	b.WriteString(headerStyle.Render("Examples:"))
	b.WriteString("\n\n")

	examples := []struct {
		cmd  string
		desc string
	}{
		{"llm_aggregator -f feeds.txt -p \"Summarise news\"", "Basic usage with feeds file"},
		{"curl -s https://example.com/feed.xml | llm_aggregator --stdin -p \"Summarise\"", "RSS from stdin"},
		{"llm_aggregator -f feeds.txt --stdin -p \"Summarise all\"", "Combine feeds file with stdin feed"},
		{"llm_aggregator -f feeds.txt -p \"Linux news\" -i linux,opensource -d 3", "Filter by keywords and age"},
		{"llm_aggregator -f feeds.txt -p \"Summarise tech news\" -t", "With TUI progress bar"},
		{"llm_aggregator -f feeds.txt -p \"Summarise news\" -D", "Dry run (no LLM API calls)"},
		{"llm_aggregator -f feeds.txt -p \"Analyse AI\" -o json --output-file output.json", "JSON output to file"},
		{"llm_aggregator -f feeds.txt -p \"Summarise news\" -P", "Plain output (no metadata)"},
	}

	for _, ex := range examples {
		if shouldStyle() {
			fmt.Fprintf(&b, "  %s\n    %s\n\n", flagStyle.Render(ex.cmd), ex.desc)
		} else {
			fmt.Fprintf(&b, "  %s\n    %s\n\n", ex.cmd, ex.desc)
		}
	}

	return b.String()
}

func WriteHelp(args *Args, w *os.File) {
	output := BuildStyledHelp(args)
	w.WriteString(output) //nolint:errcheck
}
