package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"llm_aggregator/internal/aggregator"

	"llm_aggregator/internal/llm"
	"llm_aggregator/internal/output"
	"llm_aggregator/internal/processor"
	"llm_aggregator/internal/progress"
)

// Runtime holds the execution context for the aggregator
type Runtime struct {
	// Configuration
	FeedsFile          string
	Stdin              bool
	MaxArticlesPerFeed int
	MaxDaysOld         int
	MaxTotalArticles   int
	IncludeKeywords    []string
	ExcludeKeywords    []string
	APIKey             string
	BaseURL            string
	Model              string
	MaxTokens          int
	Temperature        float64
	LLMTimeout         int // seconds; 0 means no timeout
	Prompt             string
	SystemPrompt       string
	Output             string
	OutputFile         string
	IncludeArticles    bool
	Plain              bool
	Verbose            bool

	// State
	Articles    []map[string]any
	Summary     string
	Error       error
	Interrupted bool // Set to true when a termination signal is received

	// Logger for verbose output
	Progress progress.Progress
}

// Execute runs the full aggregation pipeline
func (r *Runtime) Execute(ctx context.Context) error {
	// The logger/progress handler is injected, so we don't create it here.
	// We wrap it in a context to pass to sub-modules that expect a *progress.Context.
	progCtx := progress.NewContext(r.Progress)

	feedAgg := aggregator.NewFeedAggregatorWithProgress(
		r.MaxArticlesPerFeed,
		r.MaxDaysOld,
		5000,    // max content length
		progCtx, // Pass the new progress context
	)

	// Step 1: Aggregate feeds
	r.Progress.SetStage("Aggregating feeds")

	var articles []*aggregator.Article
	var err error

	if r.Stdin && r.FeedsFile != "" {
		// Both stdin and feeds file: collate results
		r.Progress.SetSubStage("Parsing stdin feed and feeds file")

		var stdinArticles []*aggregator.Article
		var fileArticles []*aggregator.Article

		if stdinArticles, err = feedAgg.ParseFeedFromStdin(); err != nil {
			return fmt.Errorf("error parsing stdin feed: %w", err)
		}

		if fileArticles, err = feedAgg.ParseFeedsFromFile(r.FeedsFile); err != nil {
			return fmt.Errorf("error aggregating feeds: %w", err)
		}

		articles = append(stdinArticles, fileArticles...)
	} else if r.Stdin {
		// Stdin only
		r.Progress.SetSubStage("Parsing stdin feed")
		if articles, err = feedAgg.ParseFeedFromStdin(); err != nil {
			return fmt.Errorf("error parsing stdin feed: %w", err)
		}
	} else if r.FeedsFile != "" {
		// Feeds file only
		r.Progress.SetSubStage(fmt.Sprintf("Parsing feeds from %s", r.FeedsFile))
		if articles, err = feedAgg.ParseFeedsFromFile(r.FeedsFile); err != nil {
			return fmt.Errorf("error aggregating feeds: %w", err)
		}
	} else {
		return fmt.Errorf("no feed source specified: use --feeds-file or --stdin")
	}

	if len(articles) == 0 {
		return fmt.Errorf("no articles found after parsing feeds")
	}
	r.Progress.SetArticleCount(len(articles), 0)

	// Step 2: Process content
	r.Progress.SetStage("Processing articles")
	r.Progress.SetSubStage(fmt.Sprintf("Filtering and sorting %d articles", len(articles)))

	contentProc := processor.NewContentProcessor(
		r.MaxTotalArticles,
		3000, // max content per article
		r.IncludeKeywords,
		r.ExcludeKeywords,
	)
	contentProc.SetLogger(progCtx) // Pass the new progress context

	processedArticles := contentProc.ProcessArticles(articles, "date", true)

	if len(processedArticles) == 0 {
		return fmt.Errorf("no articles passed filtering")
	}
	r.Progress.SetArticleCount(len(articles), len(articles)-len(processedArticles))

	r.Articles = processedArticles

	// Estimate tokens for progress tracking
	tokenEst := contentProc.EstimateTokenCount(processedArticles, r.Model)
	r.Progress.SetTokenEstimate(tokenEst, tokenEst)

	// Step 3: Initialise LLM client
	r.Progress.SetStage("Connecting to LLM")
	r.Progress.SetSubStage(fmt.Sprintf("Using model: %s", r.Model))

	llm, err := llm.NewLLMClient(
		r.APIKey,
		r.BaseURL,
		r.Model,
		r.MaxTokens,
		r.Temperature,
		r.LLMTimeout,
	)
	if err != nil {
		return fmt.Errorf("error initialising LLM client: %w", err)
	}
	llm.SetLogger(progCtx) // Pass the new progress context

	// Step 4: Get summary from LLM
	r.Progress.SetStage("Getting summary")
	r.Progress.SetSubStage(fmt.Sprintf("Sending %d articles to LLM", len(processedArticles)))
	r.Progress.StartWaiting()

	// Derive a timeout context for the LLM call from the parent context.
	// The parent context carries signal cancellation, so both signal interrupts
	// and timeouts will abort the call.
	callCtx := ctx
	if r.LLMTimeout > 0 {
		var cancel context.CancelFunc
		callCtx, cancel = context.WithTimeout(ctx, time.Duration(r.LLMTimeout)*time.Second)
		defer cancel()
	}

	summary, usage, err := llm.SummariseArticles(
		processedArticles,
		r.Prompt,
		r.SystemPrompt,
		callCtx,
	)
	if err != nil {
		return fmt.Errorf("error getting summary from LLM: %w", err)
	}

	r.Summary = summary
	if usage != nil {
		r.Progress.SetTokenEstimate(tokenEst, tokenEst+usage.CompletionTokens)
	}
	return nil
}

// WasInterrupted returns true if a termination signal was received during execution.
func (r *Runtime) WasInterrupted() bool {
	return r.Interrupted
}
func (r *Runtime) WriteOutput(writer io.Writer) error {
	if r.Plain {
		if _, err := fmt.Fprint(writer, r.Summary); err != nil {
			return fmt.Errorf("error writing plain output: %w", err)
		}
		return nil
	}

	formatter, err := output.NewFormatter(r.Output)
	if err != nil {
		return fmt.Errorf("error creating formatter: %w", err)
	}

	outputData := map[string]any{
		"title":          fmt.Sprintf("LLM Aggregator Summary - %s", time.Now().Format("2006-01-02 15:04")),
		"prompt":         r.Prompt,
		"model":          r.Model,
		"articles_count": len(r.Articles),
		"summary":        r.Summary,
		"timestamp":      time.Now().Format(time.RFC3339),
	}

	if r.IncludeArticles {
		outputData["articles"] = r.Articles
	}

	formattedOutput, err := formatter.FormatData(outputData)
	if err != nil {
		return fmt.Errorf("error formatting output: %w", err)
	}

	if _, err := fmt.Fprint(writer, formattedOutput); err != nil {
		return fmt.Errorf("error writing output: %w", err)
	}

	return nil
}

// WriteOutputToFile writes the output to the specified file
func (r *Runtime) WriteOutputToFile() error {
	if r.OutputFile == "" {
		return r.WriteOutput(os.Stdout)
	}

	file, err := os.Create(r.OutputFile)
	if err != nil {
		return fmt.Errorf("error creating output file: %w", err)
	}
	defer file.Close()

	return r.WriteOutput(file)
}
