package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"llm_aggregator/internal/cli"
	"llm_aggregator/internal/config"
	"llm_aggregator/internal/runtime"
	"llm_aggregator/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
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
	// Create TUI model
	m := tui.New()
	p := tea.NewProgram(m, tea.WithAltScreen())

	// Run TUI in goroutine
	done := make(chan error)
	go func() {
		if _, err := p.Run(); err != nil {
			done <- err
			return
		}
		done <- nil
	}()

	// Run aggregation in another goroutine
	go func() {
		// Create context for cancellation
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Set up signal handling
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		// Handle signals
		go func() {
			<-sigCh
			p.Send(tui.Error("Operation cancelled"))
			cancel()
		}()

		// Update TUI status
		p.Send(tui.Step(1, fmt.Sprintf("Aggregating feeds from: %s", rt.FeedsFile)))

		// Execute the runtime
		err := rt.Execute(ctx)
		if err != nil {
			p.Send(tui.Error(err.Error()))
			return
		}

		// Update TUI status
		p.Send(tui.StepWithCounts(5, "Formatting output", len(rt.Articles), len(rt.Articles)))

		// Write output
		if rt.OutputFile != "" {
			if err := rt.WriteOutputToFile(); err != nil {
				p.Send(tui.Error(fmt.Sprintf("Error writing output: %v", err)))
				return
			}
			p.Send(tui.Step(6, fmt.Sprintf("Output written to: %s", rt.OutputFile)))
		} else {
			if err := rt.WriteOutput(os.Stdout); err != nil {
				p.Send(tui.Error(fmt.Sprintf("Error writing output: %v", err)))
				return
			}
			p.Send(tui.Step(6, "Output complete"))
		}

		// Mark as done
		p.Send(tui.Step(7, "Processing completed successfully"))
	}()

	// Wait for TUI to complete
	if err := <-done; err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}

func runWithoutTUI(rt *runtime.Runtime, verbose bool) {
	// Set verbose flag on runtime
	rt.Verbose = verbose

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
