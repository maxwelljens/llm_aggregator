package runtime

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"llm_aggregator/internal/aggregator"
	"llm_aggregator/internal/defaults"
	"llm_aggregator/internal/llm"
	"llm_aggregator/internal/output"
	"llm_aggregator/internal/processor"
	"llm_aggregator/internal/progress"
)

// Runtime holds the execution context for the aggregator
type Runtime struct {
	// Configuration
	FeedsFile          string
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
	Prompt             string
	SystemPrompt       string
	Output             string
	OutputFile         string
	IncludeArticles    bool
	Verbose            bool

	// State
	Articles []map[string]any
	Summary  string
	Error    error

	// Logger for verbose output
	Progress progress.Progress
}

// NewRuntime creates a new runtime with default values
func NewRuntime() *Runtime {
	return &Runtime{
		MaxArticlesPerFeed: defaults.DefaultMaxArticlesPerFeed,
		MaxDaysOld:         defaults.DefaultMaxDaysOld,
		MaxTotalArticles:   defaults.DefaultMaxTotalArticles,
		Model:              defaults.DefaultModel,
		BaseURL:            defaults.DefaultBaseURL,
		MaxTokens:          defaults.DefaultMaxTokens,
		Temperature:        defaults.DefaultTemperature,
		Output:             defaults.DefaultOutput,
		Progress:           &progress.NoopLogger{},
	}
}

// Execute runs the full aggregation pipeline
func (r *Runtime) Execute(ctx context.Context) error {
	// The logger/progress handler is injected, so we don't create it here.
	// We wrap it in a context to pass to sub-modules that expect a *progress.Context.
	progCtx := progress.NewContext(r.Progress)

	// Step 1: Aggregate feeds
	r.Progress.SetStage("Aggregating feeds")
	r.Progress.SetSubStage(fmt.Sprintf("Parsing feeds from %s", r.FeedsFile))

	feedAgg := aggregator.NewFeedAggregatorWithProgress(
		r.MaxArticlesPerFeed,
		r.MaxDaysOld,
		5000,    // max content length
		progCtx, // Pass the new progress context
	)

	articles, err := feedAgg.ParseFeedsFromFile(r.FeedsFile)
	if err != nil {
		return fmt.Errorf("error aggregating feeds: %w", err)
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

	// Step 3: Initialise LLM client
	r.Progress.SetStage("Connecting to LLM")
	r.Progress.SetSubStage(fmt.Sprintf("Using model: %s", r.Model))

	llm, err := llm.NewLLMClient(
		r.APIKey,
		r.BaseURL,
		r.Model,
		r.MaxTokens,
		r.Temperature,
	)
	if err != nil {
		return fmt.Errorf("error initialising LLM client: %w", err)
	}
	llm.SetLogger(progCtx) // Pass the new progress context

	// Step 4: Get summary from LLM
	r.Progress.SetStage("Getting summary")
	r.Progress.SetSubStage(fmt.Sprintf("Sending %d articles to LLM", len(processedArticles)))

	summary, err := llm.SummariseArticles(
		processedArticles,
		r.Prompt,
		r.SystemPrompt,
	)
	if err != nil {
		return fmt.Errorf("error getting summary from LLM: %w", err)
	}

	r.Summary = summary
	r.Progress.SetArticleCount(len(processedArticles), len(processedArticles))
	return nil
}

// WriteOutput writes the formatted output to the specified writer
func (r *Runtime) WriteOutput(writer io.Writer) error {
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
