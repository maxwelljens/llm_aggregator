package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/cli"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/config"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/progress"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/runtime"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/signals"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/style"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

var (
	version   string
	buildDate string
)

func main() {
	cli.BuildDate = buildDate
	cli.Version = version

	args, handled, err := cli.ParseArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, style.Errorf("parsing arguments: %v", err))
		os.Exit(1)
	}
	if handled {
		os.Exit(0)
	}

	sh := signals.New()
	sh.Watch()

	v := config.GetViper()

	config.BindCLIArgs(v, args.ToViperMap())

	// FeedsFile and Prompt come from positional CLI args; everything else from viper
	rt := config.ViperToRuntime(v, args.FeedsFile, args.Prompt)

	os.Exit(run(rt, args, sh, os.Stdout, os.Stderr))
}

// run owns path selection and exit-code policy for the whole program.
// Every run path returns the process exit code; only main calls os.Exit.
func run(rt *runtime.Runtime, args *cli.Args, sh *signals.SignalHandler, stdout, stderr io.Writer) int {
	if args.DryRun {
		sh.Stop()
		rt.DryRun = true
		return runDryRun(rt, args.Verbose, stdout, stderr)
	}

	if rt.APIKey == "" {
		sh.Stop()
		fmt.Fprintln(stderr, style.Errorf("OpenAI-compatible API key is required. Set via --api-key, %s environment variable, or config file.", "LLM_AGGREGATOR_API_KEY"))
		return 1
	}

	if args.TUI {
		return runWithTUI(rt, sh, stdout, stderr)
	}
	return runWithoutTUI(rt, args.Verbose, sh, stdout, stderr)
}

func runWithTUI(rt *runtime.Runtime, sh *signals.SignalHandler, stdout, stderr io.Writer) int {
	// Cancel the pipeline context on signal, exactly like the non-TUI path,
	// so SIGINT/SIGTERM abort the in-flight LLM call instead of being ignored.
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sh.Done()
		cancel()
	}()

	// Build the model around the pipeline and inject progress bridge so
	// runtime can send messages into the TUI
	model := tui.New(rt.Execute, ctx)
	p := tea.NewProgram(model, tea.WithAltScreen())
	tp := tui.NewTUIProgress(p)
	rt.Progress = tp

	// Blocking call; returns on quit
	if _, err := p.Run(); err != nil {
		sh.Stop()
		fmt.Fprintln(stderr, style.Errorf("TUI error: %v", err))
		return 1
	}

	result, err, finished := model.FinalResult()
	if !finished {
		// User quit before the pipeline returned; nothing to write.
		sh.Stop()
		return 0
	}
	if sh.IsExiting() {
		sh.Stop()
		if result.Summary != "" {
			_ = rt.WriteOutput(stdout, result) //nolint:errcheck // stdout write failure is unrecoverable
		}
		fmt.Fprintln(stderr, style.Errorf("interrupted by signal"))
		return 130
	}
	if err != nil {
		sh.Stop()
		fmt.Fprintln(stderr, style.Errorf("execution failed: %v", err))
		return 1
	}

	if err := writeResult(rt, result, stdout); err != nil {
		sh.Stop()
		fmt.Fprintln(stderr, style.Errorf("%v", err))
		return 1
	}

	sh.Stop()
	return 0
}

func runWithoutTUI(rt *runtime.Runtime, verbose bool, sh *signals.SignalHandler, stdout, stderr io.Writer) int {
	// SimpleLogger outputs to stdout; nil uses NoopLogger
	if verbose {
		rt.Progress = progress.NewSimpleLogger(stdout, true)
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
	result, err := rt.Execute(ctx)
	if sh.IsExiting() {
		// Signal arrived during execution — output partial result if available
		sh.Stop()
		if result.Summary != "" {
			_ = rt.WriteOutput(stdout, result) //nolint:errcheck // stdout write failure is unrecoverable
		}
		fmt.Fprintln(stderr, style.Errorf("interrupted by signal"))
		return 130
	}
	if err != nil {
		sh.Stop()
		fmt.Fprintln(stderr, style.Errorf("execution failed: %v", err))
		return 1
	}

	if err := writeResult(rt, result, stdout); err != nil {
		sh.Stop()
		fmt.Fprintln(stderr, style.Errorf("%v", err))
		return 1
	}

	sh.Stop()
	return 0
}

// writeResult sends the pipeline result to the configured output destination.
func writeResult(rt *runtime.Runtime, result runtime.Result, stdout io.Writer) error {
	if rt.OutputFile != "" {
		if err := rt.WriteOutputToFile(result); err != nil {
			return fmt.Errorf("writing output to file: %v", err)
		}
		return nil
	}
	if err := rt.WriteOutput(stdout, result); err != nil {
		return fmt.Errorf("writing output: %v", err)
	}
	return nil
}

func runDryRun(rt *runtime.Runtime, verbose bool, stdout, stderr io.Writer) int {
	// Setup logger based on verbose flag
	if verbose {
		rt.Progress = progress.NewSimpleLogger(stdout, true)
	} else {
		rt.Progress = &progress.NoopLogger{}
	}

	// Print dry-run header
	fmt.Fprintln(stdout, style.Heading("========================================"))
	fmt.Fprintln(stdout, style.Heading("       llm_aggregator --dry-run"))
	fmt.Fprintln(stdout, style.Heading("========================================"))
	fmt.Fprintln(stdout)

	// Validate feed source exists
	if rt.FeedsFile != "" {
		fmt.Fprintf(stdout, "%s Feeds file: %s\n", style.Success(""), style.Filepath(rt.FeedsFile))
	} else {
		fmt.Fprintf(stdout, "%s Feed source: stdin\n", style.Success(""))
	}

	// Print configuration summary
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, style.Label("Configuration:"))
	fmt.Fprintf(stdout, "  Max articles per feed: %s\n", style.Value(strconv.Itoa(rt.MaxArticlesPerFeed)))
	fmt.Fprintf(stdout, "  Max days old: %s\n", style.Value(strconv.Itoa(rt.MaxDaysOld)))
	fmt.Fprintf(stdout, "  Max total articles: %s\n", style.Value(strconv.Itoa(rt.MaxTotalArticles)))
	fmt.Fprintf(stdout, "  Include keywords: %s\n", style.Value(fmt.Sprintf("%v", rt.IncludeKeywords)))
	fmt.Fprintf(stdout, "  Exclude keywords: %s\n", style.Value(fmt.Sprintf("%v", rt.ExcludeKeywords)))
	fmt.Fprintf(stdout, "  Output format: %s\n", style.Value(rt.Output))
	fmt.Fprintf(stdout, "  Model: %s\n", style.Value(rt.Model))
	fmt.Fprintf(stdout, "  LLM timeout: %s\n", style.Value(fmt.Sprintf("%d seconds", rt.LLMTimeout)))
	fmt.Fprintln(stdout)

	// Fetch and process feeds through the shared pipeline (no LLM call)
	fmt.Fprintln(stdout, style.Info("Fetching feeds..."))

	result, err := rt.Execute(context.Background())
	if err != nil {
		if errors.Is(err, runtime.ErrNoArticles) || errors.Is(err, runtime.ErrNoArticlesPassedFiltering) {
			fmt.Fprintln(stderr, style.Warning("No articles found. Check your feeds file or network connectivity."))
			fmt.Fprintln(stdout)
			fmt.Fprintf(stdout, "%s %s\n", style.Success("Dry-run complete"), "(no LLM API calls made).")
			return 0
		}
		fmt.Fprintln(stderr, style.Errorf("dry-run failed: %v", err))
		return 1
	}

	totalArticles := result.ArticlesFetched
	fmt.Fprintf(stdout, "%s Fetched %d articles from feeds\n", style.Success(""), totalArticles)

	filteredCount := totalArticles - len(result.Articles)

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, style.Label("Article statistics:"))
	fmt.Fprintf(stdout, "  Total fetched: %s\n", style.Value(strconv.Itoa(totalArticles)))
	fmt.Fprintf(stdout, "  After filtering: %s\n", style.Value(strconv.Itoa(len(result.Articles))))
	if filteredCount > 0 {
		fmt.Fprintf(stdout, "  Filtered out: %s\n", style.Value(strconv.Itoa(filteredCount)))
	}

	// Group articles by source for summary
	sourceCounts := make(map[string]int)
	for _, article := range result.Articles {
		if article.SourceFeed != "" {
			sourceCounts[article.SourceFeed]++
		}
	}

	if len(sourceCounts) > 0 {
		fmt.Fprintln(stdout)
		fmt.Fprintln(stdout, style.Label("Articles by source:"))
		for source, count := range sourceCounts {
			fmt.Fprintf(stdout, "  %s: %s\n", style.Filepath(source), style.Value(strconv.Itoa(count)))
		}
	}

	fmt.Fprintln(stdout)
	fmt.Fprintf(stdout, "Estimated token count: %s (for model: %s)\n", style.Value(fmt.Sprintf("~%d", result.TokenEstimate)), style.Value(rt.Model))
	fmt.Fprintf(stdout, "Max tokens for response: %s\n", style.Value(strconv.Itoa(rt.MaxTokens)))

	// Prompt preview
	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, style.Label("Prompt:"))
	fmt.Fprintf(stdout, "  %s\n", style.Italic(rt.Prompt))
	if rt.SystemPrompt != "" {
		fmt.Fprintln(stdout, style.Label("System prompt:"))
		fmt.Fprintf(stdout, "  %s\n", style.Italic(rt.SystemPrompt))
	}

	fmt.Fprintln(stdout)
	fmt.Fprintln(stdout, style.Heading("========================================"))
	fmt.Fprintf(stdout, " %s %s\n", style.Success("Dry-run complete"), "(no LLM API calls made).")
	fmt.Fprintln(stdout, style.Heading("========================================"))

	return 0
}
