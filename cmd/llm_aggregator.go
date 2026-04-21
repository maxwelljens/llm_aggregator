package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"llm_aggregator/internal/cli"
	"llm_aggregator/internal/config"
	"llm_aggregator/internal/progress"
	"llm_aggregator/internal/runtime"
	"llm_aggregator/internal/tui"
)

var (
	version   string
	buildDate string
)

func main() {
	cli.BuildDate = buildDate
	cli.Version = version

	// Load configuration from file and environment variables
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load configuration: %v\n", err)
		// Continue with defaults
		cfg = config.DefaultConfig()
	}

	// Parse command line arguments
	args, err := cli.ParseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	// Create runtime configuration
	rt := runtime.NewRuntime()
	rt.FeedsFile = args.FeedsFile
	rt.Prompt = args.Prompt

	// Apply configuration with precedence: CLI args > Environment vars > Config file > Defaults
	applyConfiguration(rt, cfg, args)

	// Validate API key
	if rt.APIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: OpenAI-compatible API key is required. Set via --api-key, LLM_AGGREGATOR_API_KEY environment variable, or config file.")
		os.Exit(1)
	}

	// Run with TUI if requested
	if args.TUI {
		runWithTUI(rt)
	} else {
		runWithoutTUI(rt, args.Verbose)
	}
}

// applyConfiguration applies configuration with proper precedence
func applyConfiguration(rt *runtime.Runtime, cfg *config.Config, args *cli.Args) {
	// Feed aggregation options - CLI args override config
	if args.MaxArticlesPerFeed != 0 {
		rt.MaxArticlesPerFeed = args.MaxArticlesPerFeed
	} else if cfg.MaxArticlesPerFeed != 0 {
		rt.MaxArticlesPerFeed = cfg.MaxArticlesPerFeed
	}

	if args.MaxDaysOld != 0 {
		rt.MaxDaysOld = args.MaxDaysOld
	} else if cfg.MaxDaysOld != 0 {
		rt.MaxDaysOld = cfg.MaxDaysOld
	}

	if args.MaxTotalArticles != 0 {
		rt.MaxTotalArticles = args.MaxTotalArticles
	} else if cfg.MaxTotalArticles != 0 {
		rt.MaxTotalArticles = cfg.MaxTotalArticles
	}

	// Content filtering - CLI args override config
	if args.IncludeKeywords != "" {
		rt.IncludeKeywords = cli.ParseKeywords(args.IncludeKeywords)
	} else if cfg.IncludeKeywords != "" {
		rt.IncludeKeywords = cli.ParseKeywords(cfg.IncludeKeywords)
	}

	if args.ExcludeKeywords != "" {
		rt.ExcludeKeywords = cli.ParseKeywords(args.ExcludeKeywords)
	} else if cfg.ExcludeKeywords != "" {
		rt.ExcludeKeywords = cli.ParseKeywords(cfg.ExcludeKeywords)
	}

	// Deepseek API options - CLI args override config
	if args.APIKey != "" {
		rt.APIKey = args.APIKey
	} else if cfg.APIKey != "" {
		rt.APIKey = cfg.APIKey
	} else {
		// Fall back to environment variable
		rt.APIKey = os.Getenv("LLM_AGGREGATOR_API_KEY")
	}

	if args.Model != "" {
		rt.Model = args.Model
	} else if cfg.Model != "" {
		rt.Model = cfg.Model
	}

	if args.MaxTokens != 0 {
		rt.MaxTokens = args.MaxTokens
	} else if cfg.MaxTokens != 0 {
		rt.MaxTokens = cfg.MaxTokens
	}

	if args.Temperature != 0 {
		rt.Temperature = args.Temperature
	} else if cfg.Temperature != 0 {
		rt.Temperature = cfg.Temperature
	}

	// System prompt - CLI args override config
	if args.SystemPrompt != "" {
		rt.SystemPrompt = args.SystemPrompt
	} else if cfg.SystemPrompt != "" {
		rt.SystemPrompt = cfg.SystemPrompt
	}

	// Output options - CLI args override config
	if args.Output != "" {
		rt.Output = args.Output
	} else if cfg.Output != "" {
		rt.Output = cfg.Output
	}

	if args.OutputFile != "" {
		rt.OutputFile = args.OutputFile
	} else if cfg.OutputFile != "" {
		rt.OutputFile = cfg.OutputFile
	}

	rt.IncludeArticles = args.IncludeArticles || cfg.IncludeArticles
	rt.Verbose = args.Verbose
}

func runWithTUI(rt *runtime.Runtime) {
	// 1. Create the model and pass the runtime to it.
	model := tui.New(rt)

	// 2. Create the tea.Program.
	p := tea.NewProgram(model, tea.WithAltScreen())

	// 3. Create the TUIProgress handler and give it the program instance.
	//    This is the bridge that allows the runtime to send messages to the TUI.
	tp := tui.NewTUIProgress(p)

	// 4. Inject the progress handler into the runtime.
	rt.Progress = tp

	// 5. Run the program. This is a blocking call.
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI Error: %v\n", err)
		os.Exit(1)
	}
}

func runWithoutTUI(rt *runtime.Runtime, verbose bool) {
	// Setup logger based on verbose flag and inject it into the runtime
	if verbose {
		rt.Progress = progress.NewSimpleLogger(os.Stdout, true)
	} else {
		// Default is already NoopLogger, but we can be explicit
		rt.Progress = &progress.NoopLogger{}
	}

	// Execute the runtime
	err := rt.Execute(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Write output
	if rt.OutputFile != "" {
		if err := rt.WriteOutputToFile(); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := rt.WriteOutput(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	}
}
