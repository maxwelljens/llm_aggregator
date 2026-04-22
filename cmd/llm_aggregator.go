package main

import (
	"context"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"llm_aggregator/internal/aggregator"
	"llm_aggregator/internal/cli"
	"llm_aggregator/internal/config"
	"llm_aggregator/internal/processor"
	"llm_aggregator/internal/progress"
	"llm_aggregator/internal/runtime"
	"llm_aggregator/internal/style"
	"llm_aggregator/internal/tui"
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
		fmt.Fprintln(os.Stderr, style.Errorf("parsing arguments: %v", err))
		os.Exit(1)
	}

	// Get viper instance with config file + environment variables + defaults
	v := config.GetViper()

	// Bind CLI args to viper (highest precedence)
	config.BindCLIArgs(v, args.ToViperMap())

	// Create runtime directly from viper configuration
	rt := config.ViperToRuntime(v, args.FeedsFile, args.Prompt)

	// Run dry-run mode if requested (validates config, shows statistics, no LLM calls)
	if args.DryRun {
		runDryRun(rt, args.Verbose)
		os.Exit(0)
	}

	// Validate API key (required for actual execution)
	if v.GetString("api_key") == "" {
		fmt.Fprintln(os.Stderr, style.Errorf("OpenAI-compatible API key is required. Set via --api-key, %s environment variable, or config file.", "LLM_AGGREGATOR_API_KEY"))
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
		fmt.Fprintln(os.Stderr, style.Errorf("TUI error: %v", err))
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

	// Validate feeds file exists
	if _, err := os.Stat(rt.FeedsFile); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, style.Errorf("Feeds file not found: %s", rt.FeedsFile))
		os.Exit(1)
	}
	fmt.Printf("%s Feeds file: %s\n", style.Success(""), style.Filepath(rt.FeedsFile))

	// Print configuration summary
	fmt.Println()
	fmt.Println(style.Label("Configuration:"))
	fmt.Printf("  Max articles per feed: %s\n", style.Value(fmt.Sprintf("%d", rt.MaxArticlesPerFeed)))
	fmt.Printf("  Max days old: %s\n", style.Value(fmt.Sprintf("%d", rt.MaxDaysOld)))
	fmt.Printf("  Max total articles: %s\n", style.Value(fmt.Sprintf("%d", rt.MaxTotalArticles)))
	fmt.Printf("  Include keywords: %s\n", style.Value(fmt.Sprintf("%v", rt.IncludeKeywords)))
	fmt.Printf("  Exclude keywords: %s\n", style.Value(fmt.Sprintf("%v", rt.ExcludeKeywords)))
	fmt.Printf("  Output format: %s\n", style.Value(rt.Output))
	fmt.Printf("  Model: %s\n", style.Value(rt.Model))
	fmt.Println()

	// Fetch and process feeds (but don't call LLM)
	fmt.Println(style.Info("Fetching feeds..."))

	feedAgg := aggregator.NewFeedAggregatorWithProgress(
		rt.MaxArticlesPerFeed,
		rt.MaxDaysOld,
		5000, // max content length
		nil,  // No progress context for cleaner dry-run output
	)

	articles, err := feedAgg.ParseFeedsFromFile(rt.FeedsFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, style.Errorf("Failed to parse feeds: %v", err))
		os.Exit(1)
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
	fmt.Printf("  Total fetched: %s\n", style.Value(fmt.Sprintf("%d", totalArticles)))
	fmt.Printf("  After filtering: %s\n", style.Value(fmt.Sprintf("%d", len(processedArticles))))
	if filteredCount > 0 {
		fmt.Printf("  Filtered out: %s\n", style.Value(fmt.Sprintf("%d", filteredCount)))
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
			fmt.Printf("  %s: %s\n", style.Filepath(source), style.Value(fmt.Sprintf("%d", count)))
		}
	}

	// Estimate token count
	estimatedTokens := contentProc.EstimateTokenCount(processedArticles, rt.Model)
	fmt.Println()
	fmt.Printf("Estimated token count: %s (for model: %s)\n", style.Value(fmt.Sprintf("~%d", estimatedTokens)), style.Value(rt.Model))
	fmt.Printf("Max tokens for response: %s\n", style.Value(fmt.Sprintf("%d", rt.MaxTokens)))

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
