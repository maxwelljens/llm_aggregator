package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/aggregator"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/llm"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/output"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/processor"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/progress"
)

// Runtime holds the full execution context for the pipeline.
// It is constructed once in main() and passed through each stage.
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
	Temperature        *float64
	LLMTimeout         int // seconds; 0 means no timeout
	Prompt             string
	SystemPrompt       string
	Output             string
	OutputFile         string
	IncludeArticles    bool
	Plain              bool
	Verbose            bool

	// State
	Articles []map[string]any
	Summary  string

	// Progress is the logger/progress handler injected by the caller.
	// nil means no output (NoopLogger). SimpleLogger outputs to stdout.
	// TUIProgress bridges into the Bubbletea program for live updates.
	Progress progress.Progress
}

// Execute runs the four-stage pipeline: aggregate → process → LLM → (output is separate).
func (r *Runtime) Execute(ctx context.Context) error {
	// Guard against a nil Progress (bare Runtime values); NoopLogger produces no output.
	if r.Progress == nil {
		r.Progress = &progress.NoopLogger{}
	}

	feedAgg := aggregator.NewFeedAggregator(
		r.MaxArticlesPerFeed,
		r.MaxDaysOld,
		5000, // max content length
		r.Progress,
	)

	// Step 1: Aggregate feeds
	r.Progress.SetStage(progress.StageAggregating)

	var articles []*aggregator.Article
	var err error

	switch {
	case r.Stdin && r.FeedsFile != "":
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

		articles = append(stdinArticles, fileArticles...) //nolint:gocritic
	case r.Stdin:
		// Stdin only
		r.Progress.SetSubStage("Parsing stdin feed")
		if articles, err = feedAgg.ParseFeedFromStdin(); err != nil {
			return fmt.Errorf("error parsing stdin feed: %w", err)
		}
	case r.FeedsFile != "":
		// Feeds file only
		r.Progress.SetSubStage("Parsing feeds from " + r.FeedsFile)
		if articles, err = feedAgg.ParseFeedsFromFile(r.FeedsFile); err != nil {
			return fmt.Errorf("error aggregating feeds: %w", err)
		}
	default:
		return errors.New("no feed source specified: use --feeds-file or --stdin")
	}

	if len(articles) == 0 {
		return errors.New("no articles found after parsing feeds")
	}
	r.Progress.SetArticleCount(len(articles), 0)

	// Step 2: Process content
	r.Progress.SetStage(progress.StageProcessing)
	r.Progress.SetSubStage(fmt.Sprintf("Filtering and sorting %d articles", len(articles)))

	contentProc := processor.NewContentProcessor(
		r.MaxTotalArticles,
		3000, // max content per article
		r.IncludeKeywords,
		r.ExcludeKeywords,
	)
	contentProc.SetLogger(r.Progress)

	processedArticles := contentProc.ProcessArticles(articles, "date", true)

	if len(processedArticles) == 0 {
		return errors.New("no articles passed filtering")
	}
	r.Progress.SetArticleCount(len(articles), len(articles)-len(processedArticles))

	r.Articles = processedArticles

	// Estimate tokens for progress tracking
	tokenEst := contentProc.EstimateTokenCount(processedArticles, r.Model)
	r.Progress.SetTokenEstimate(tokenEst, tokenEst)

	// Step 3: Initialise LLM client
	r.Progress.SetStage(progress.StageConnecting)
	r.Progress.SetSubStage("Using model: " + r.Model)

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
	llm.SetLogger(r.Progress)

	// Step 4: Get summary from LLM
	r.Progress.SetStage(progress.StageGettingSummary)
	r.Progress.SetSubStage(fmt.Sprintf("Sending %d articles to LLM", len(processedArticles)))
	r.Progress.StartWaiting()

	// callCtx wraps the parent context with a timeout derived from LLMTimeout.
	// Both signal cancellation and timeout will abort the LLM call.
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

// WriteOutput writes the formatted summary to the given writer.
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
		"title":          "LLM Aggregator Summary - " + time.Now().Format("2006-01-02 15:04"),
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
	defer file.Close() //nolint:errcheck

	return r.WriteOutput(file)
}
