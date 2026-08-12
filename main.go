package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/aggregator"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/cli"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/config"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/processor"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/progress"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/runtime"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/signals"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/style"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/tui"
)

var (
	version   string
	buildDate string
)

func main() {
	cli.BuildDate = buildDate
	cli.Version = version

	args, err := cli.ParseArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, style.Errorf("parsing arguments: %v", err))
		os.Exit(1)
	}

	sh := signals.New()
	sh.Watch()

	v := config.GetViper()

	config.BindCLIArgs(v, args.ToViperMap())

	// FeedsFile and Prompt come from positional CLI args; everything else from viper
	rt := config.ViperToRuntime(v, args.FeedsFile, args.Prompt)


	if args.DryRun {
		sh.Stop()
		runDryRun(rt, args.Verbose)
		os.Exit(0)
	}

	if v.GetString("api_key") == "" {
		sh.Stop()
		fmt.Fprintln(os.Stderr, style.Errorf("OpenAI-compatible API key is required. Set via --api-key, %s environment variable, or config file.", "LLM_AGGREGATOR_API_KEY"))
		os.Exit(1)
	}

	if args.TUI {
		sh.Stop()
		runWithTUI(rt)
	} else {
		runWithoutTUI(rt, args.Verbose, sh)
	}
}

func runWithTUI(rt *runtime.Runtime) {
	// Build the model and inject progress bridge so runtime can send messages into the TUI
	model := tui.New(rt)
	p := tea.NewProgram(model, tea.WithAltScreen())
	tp := tui.NewTUIProgress(p)
	rt.Progress = tp

	// Blocking call; returns on quit
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, style.Errorf("TUI error: %v", err))
		os.Exit(1)
	}
}

func runWithoutTUI(rt *runtime.Runtime, verbose bool, sh *signals.SignalHandler) {
	// SimpleLogger outputs to stdout; nil uses NoopLogger
	if verbose {
		rt.Progress = progress.NewSimpleLogger(os.Stdout, true)
	} else {
		// Default is already NoopLogger, but we can be explicit
		rt.Progress = &progress.NoopLogger{}
	}

	// Create a context that is cancelled when a signal arrives.
	// signal.Notify disables default exit behaviour for SIGINT/SIGTERM/SIGHUP,
	// so the program stays alive long enough to handle them.
	ctx, cancel := context.WithCancel(context.Background())

	// Propagate signal reception into context cancellation. The channel closes
	// exactly once, when the first signal arrives; on clean completion os.Exit
	// terminates this goroutine.
	go func() {
		<-sh.Done()
		cancel()
	}()

	// Execute the runtime
	err := rt.Execute(ctx)
	if sh.IsExiting() {
		// Signal arrived during execution — output partial result if available
		sh.Stop()
		if rt.Summary != "" {
			_ = rt.WriteOutput(os.Stdout) //nolint:errcheck // stdout write failure is unrecoverable
		}
		fmt.Fprintln(os.Stderr, style.Errorf("interrupted by signal"))
		os.Exit(130)
	}
	if err != nil {
		sh.Stop()
		fmt.Fprintln(os.Stderr, style.Errorf("execution failed: %v", err))
		os.Exit(1)
	}

	// Write output
	if rt.OutputFile != "" {
		if err := rt.WriteOutputToFile(); err != nil {
			fmt.Fprintln(os.Stderr, style.Errorf("writing output to file: %v", err))
			os.Exit(1)
		}
	} else {
		if err := rt.WriteOutput(os.Stdout); err != nil {
			fmt.Fprintln(os.Stderr, style.Errorf("writing output: %v", err))
			os.Exit(1)
		}
	}

	sh.Stop()
}

func runDryRun(rt *runtime.Runtime, verbose bool) {
	// Setup logger based on verbose flag
	if verbose {
		rt.Progress = progress.NewSimpleLogger(os.Stdout, true)
	} else {
		rt.Progress = &progress.NoopLogger{}
	}

	// Print dry-run header
	fmt.Println(style.Heading("========================================"))
	fmt.Println(style.Heading("       llm_aggregator --dry-run"))
	fmt.Println(style.Heading("========================================"))
	fmt.Println()

	// Validate feed source exists
	if rt.FeedsFile != "" {
		fmt.Printf("%s Feeds file: %s\n", style.Success(""), style.Filepath(rt.FeedsFile))
	} else {
		fmt.Printf("%s Feed source: stdin\n", style.Success(""))
	}

	// Print configuration summary
	fmt.Println()
	fmt.Println(style.Label("Configuration:"))
	fmt.Printf("  Max articles per feed: %s\n", style.Value(strconv.Itoa(rt.MaxArticlesPerFeed)))
	fmt.Printf("  Max days old: %s\n", style.Value(strconv.Itoa(rt.MaxDaysOld)))
	fmt.Printf("  Max total articles: %s\n", style.Value(strconv.Itoa(rt.MaxTotalArticles)))
	fmt.Printf("  Include keywords: %s\n", style.Value(fmt.Sprintf("%v", rt.IncludeKeywords)))
	fmt.Printf("  Exclude keywords: %s\n", style.Value(fmt.Sprintf("%v", rt.ExcludeKeywords)))
	fmt.Printf("  Output format: %s\n", style.Value(rt.Output))
	fmt.Printf("  Model: %s\n", style.Value(rt.Model))
	fmt.Printf("  LLM timeout: %s\n", style.Value(fmt.Sprintf("%d seconds", rt.LLMTimeout)))
	fmt.Println()

	// Fetch and process feeds (but don't call LLM)
	fmt.Println(style.Info("Fetching feeds..."))

	feedAgg := aggregator.NewFeedAggregatorWithProgress(
		rt.MaxArticlesPerFeed,
		rt.MaxDaysOld,
		5000, // max content length
		nil,  // No progress context for cleaner dry-run output
	)

	var articles []*aggregator.Article
	var err error

	switch {
	case rt.Stdin && rt.FeedsFile != "":
		var stdinArticles []*aggregator.Article
		var fileArticles []*aggregator.Article

		if stdinArticles, err = feedAgg.ParseFeedFromStdin(); err != nil {
			fmt.Fprintln(os.Stderr, style.Errorf("Failed to parse stdin feed: %v", err))
			os.Exit(1)
		}
		if fileArticles, err = feedAgg.ParseFeedsFromFile(rt.FeedsFile); err != nil {
			fmt.Fprintln(os.Stderr, style.Errorf("Failed to parse feeds file: %v", err))
			os.Exit(1)
		}
		articles = append(stdinArticles, fileArticles...) //nolint:gocritic
	case rt.Stdin:
		if articles, err = feedAgg.ParseFeedFromStdin(); err != nil {
			fmt.Fprintln(os.Stderr, style.Errorf("Failed to parse stdin feed: %v", err))
			os.Exit(1)
		}
	default:
		if articles, err = feedAgg.ParseFeedsFromFile(rt.FeedsFile); err != nil {
			fmt.Fprintln(os.Stderr, style.Errorf("Failed to parse feeds: %v", err))
			os.Exit(1)
		}
	}

	totalArticles := len(articles)
	fmt.Printf("%s Fetched %d articles from feeds\n", style.Success(""), totalArticles)

	if totalArticles == 0 {
		fmt.Fprintln(os.Stderr, style.Warning("No articles found. Check your feeds file or network connectivity."))
		fmt.Println()
		fmt.Printf("%s %s\n", style.Success("Dry-run complete"), "(no LLM API calls made).")
		os.Exit(0)
	}

	// Process articles (filter and sort)
	contentProc := processor.NewContentProcessor(
		rt.MaxTotalArticles,
		3000, // max content per article
		rt.IncludeKeywords,
		rt.ExcludeKeywords,
	)

	processedArticles := contentProc.ProcessArticles(articles, "date", true)
	filteredCount := totalArticles - len(processedArticles)

	fmt.Println()
	fmt.Println(style.Label("Article statistics:"))
	fmt.Printf("  Total fetched: %s\n", style.Value(strconv.Itoa(totalArticles)))
	fmt.Printf("  After filtering: %s\n", style.Value(strconv.Itoa(len(processedArticles))))
	if filteredCount > 0 {
		fmt.Printf("  Filtered out: %s\n", style.Value(strconv.Itoa(filteredCount)))
	}

	// Group articles by source for summary
	sourceCounts := make(map[string]int)
	for _, article := range processedArticles {
		if title, ok := article["source_feed"].(string); ok {
			sourceCounts[title]++
		}
	}

	if len(sourceCounts) > 0 {
		fmt.Println()
		fmt.Println(style.Label("Articles by source:"))
		for source, count := range sourceCounts {
			fmt.Printf("  %s: %s\n", style.Filepath(source), style.Value(strconv.Itoa(count)))
		}
	}

	// Estimate token count
	estimatedTokens := contentProc.EstimateTokenCount(processedArticles, rt.Model)
	fmt.Println()
	fmt.Printf("Estimated token count: %s (for model: %s)\n", style.Value(fmt.Sprintf("~%d", estimatedTokens)), style.Value(rt.Model))
	fmt.Printf("Max tokens for response: %s\n", style.Value(strconv.Itoa(rt.MaxTokens)))

	// Prompt preview
	fmt.Println()
	fmt.Println(style.Label("Prompt:"))
	fmt.Printf("  %s\n", style.Italic(rt.Prompt))
	if rt.SystemPrompt != "" {
		fmt.Println(style.Label("System prompt:"))
		fmt.Printf("  %s\n", style.Italic(rt.SystemPrompt))
	}

	fmt.Println()
	fmt.Println(style.Heading("========================================"))
	fmt.Printf(" %s %s\n", style.Success("Dry-run complete"), "(no LLM API calls made).")
	fmt.Println(style.Heading("========================================"))
}
