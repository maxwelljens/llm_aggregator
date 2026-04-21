package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"llm_aggregator/internal/cli"
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
	// Parse command line arguments
	args, err := cli.ParseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	// Create runtime configuration
	rt := runtime.NewRuntime()
	rt.FeedsFile = args.FeedsFile
	rt.MaxArticlesPerFeed = args.MaxArticlesPerFeed
	rt.MaxDaysOld = args.MaxDaysOld
	rt.MaxTotalArticles = args.MaxTotalArticles
	rt.IncludeKeywords = cli.ParseKeywords(args.IncludeKeywords)
	rt.ExcludeKeywords = cli.ParseKeywords(args.ExcludeKeywords)
	rt.APIKey = args.APIKey
	if rt.APIKey == "" {
		rt.APIKey = os.Getenv("DEEPSEEK_API_KEY")
	}
	rt.Model = args.Model
	rt.MaxTokens = args.MaxTokens
	rt.Temperature = args.Temperature
	rt.Prompt = args.Prompt
	rt.SystemPrompt = args.SystemPrompt
	rt.Output = args.Output
	rt.OutputFile = args.OutputFile
	rt.IncludeArticles = args.IncludeArticles

	// Validate API key
	if rt.APIKey == "" {
		fmt.Fprintln(os.Stderr, "Error: DeepSeek API key is required. Set via --api-key or DEEPSEEK_API_KEY environment variable.")
		os.Exit(1)
	}

	// Run with TUI if requested
	if args.TUI {
		runWithTUI(rt)
	} else {
		runWithoutTUI(rt, args.Verbose)
	}
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
	if verbose {
		fmt.Printf("Aggregating feeds from: %s\n", rt.FeedsFile)
	}

	// Execute the runtime
	err := rt.Execute(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if verbose {
		fmt.Printf("Aggregated %d articles\n", len(rt.Articles))
		fmt.Printf("Requesting summary from DeepSeek with model: %s\n", rt.Model)
	}

	// Write output
	if rt.OutputFile != "" {
		if err := rt.WriteOutputToFile(); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
		if verbose {
			fmt.Printf("Output written to: %s\n", rt.OutputFile)
		}
	} else {
		if err := rt.WriteOutput(os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
			os.Exit(1)
		}
	}

	if verbose {
		fmt.Println("Processing completed successfully")
	}
}

