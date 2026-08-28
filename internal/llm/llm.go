package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/aggregator"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/defaults"
	"codeberg.org/maxwelljensen/llm_aggregator/internal/progress"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// Options configures an LLMClient. APIKey is required; the config layer
// resolves it (CLI flag, LLM_AGGREGATOR_API_KEY env var, config file) before
// constructing the client, so this package never reads the environment.
type Options struct {
	APIKey         string
	BaseURL        string
	Model          string
	MaxTokens      int
	Temperature    *float64
	TimeoutSeconds int
}

// LLMClient is a client for interacting with LLM API.
type LLMClient struct {
	client      openai.Client
	model       string
	maxTokens   int
	temperature *float64
	llmTimeout  int // seconds; 0 means no timeout
	logger      progress.Progress
}

// NewLLMClient creates an LLM API client.
func NewLLMClient(opts Options) (*LLMClient, error) {
	if opts.APIKey == "" || strings.TrimSpace(opts.APIKey) == "" {
		return nil, errors.New(
			"LLM API key is required. " +
				"Set via --api-key, LLM_AGGREGATOR_API_KEY environment variable, or config file",
		)
	}

	// Set defaults using central constants
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = defaults.DefaultBaseURL
	}
	model := opts.Model
	if model == "" {
		model = defaults.DefaultModel
	}
	maxTokens := opts.MaxTokens
	if maxTokens == 0 {
		maxTokens = defaults.DefaultMaxTokens
	}
	temperature := opts.Temperature
	if temperature == nil {
		t := defaults.DefaultTemperature
		temperature = &t
	}
	timeoutSeconds := opts.TimeoutSeconds
	if timeoutSeconds == 0 {
		timeoutSeconds = defaults.DefaultLLMTimeout
	}

	// Create OpenAI client configured for LLM
	clientOpts := []option.RequestOption{
		option.WithAPIKey(opts.APIKey),
		option.WithBaseURL(baseURL),
	}

	client := openai.NewClient(clientOpts...)

	return &LLMClient{
		client:      client,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
		llmTimeout:  timeoutSeconds,
		logger:      &progress.NoopLogger{},
	}, nil
}

// SetLogger sets the logger for the LLM client; nil means no output.
func (dc *LLMClient) SetLogger(logger progress.Progress) {
	if logger == nil {
		logger = &progress.NoopLogger{}
	}
	dc.logger = logger
}

// TokenUsage holds token usage information from the API response.
type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
}

// SummariseArticles sends articles to the LLM for summarisation.
// Returns the LLM response text, token usage, and any error.
// ctx must carry signal cancellation so that SIGINT/SIGTERM aborts the call.
func (dc *LLMClient) SummariseArticles(
	articles []*aggregator.Article,
	userPrompt string,
	systemPrompt string,
	ctx context.Context,
) (string, *TokenUsage, error) {
	if len(articles) == 0 {
		return "No articles to summarise.", nil, nil
	}

	// Prepare context from articles
	context := dc.prepareContext(articles)

	// Create messages for chat completion API
	messages := dc.createMessages(context, userPrompt, systemPrompt)

	// Call API with messages
	return dc.callAPIWithMessages(ctx, messages)
}

func (dc *LLMClient) prepareContext(articles []*aggregator.Article) string {
	contextParts := []string{}

	for i, article := range articles {
		contextParts = append(contextParts, fmt.Sprintf("--- ARTICLE %d ---", i+1))
		contextParts = append(contextParts, "Title: "+article.Title)

		if article.SourceFeed != "" {
			contextParts = append(contextParts, "Source: "+article.SourceFeed)
		}

		if !article.Published.IsZero() {
			contextParts = append(contextParts, "Published: "+article.Published.Format(time.RFC3339))
		}

		if article.Author != "" {
			contextParts = append(contextParts, "Author: "+article.Author)
		}

		if article.Link != "" {
			contextParts = append(contextParts, "Link: "+article.Link)
		}

		if article.Content != "" {
			// Content is truncated by the processor; render as-is.
			contextParts = append(contextParts, "Content: "+article.Content)
		}

		contextParts = append(contextParts, "") // Empty line between articles
	}

	return strings.Join(contextParts, "\n")
}

func (dc *LLMClient) createMessages(context, userPrompt, systemPrompt string) []openai.ChatCompletionMessageParamUnion {
	if systemPrompt == "" {
		systemPrompt = defaults.DefaultSystemPrompt
	}

	// Combine context with user prompt
	fullUserContent := fmt.Sprintf(`Here are articles from various RSS feeds:

%s

User request: %s

Please provide a comprehensive summary/analysis addressing the user's request.
Focus on key insights, trends, and important information from the articles.
If relevant, note any patterns, contradictions, or notable developments.`,
		context, userPrompt)

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
		openai.UserMessage(fullUserContent),
	}

	return messages
}

func (dc *LLMClient) callAPIWithMessages(ctx context.Context, messages []openai.ChatCompletionMessageParamUnion) (string, *TokenUsage, error) {
	dc.logger.Logf("Calling LLM API with model: %s", dc.model)
	dc.logger.Logf("Max tokens: %d, Temperature: %.2f", dc.maxTokens, *dc.temperature)
	dc.logger.Logf("Messages count: %d", len(messages))

	response, err := dc.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       dc.model,
		Messages:    messages,
		MaxTokens:   openai.Int(int64(dc.maxTokens)),
		Temperature: openai.Float(*dc.temperature),
	})

	if err != nil {
		errStr := err.Error()

		dc.logger.Logf("API error: %s", errStr)

		switch {
		case strings.Contains(errStr, "401"):
			return "", nil, errors.New("invalid API key. Please check your LLM API key")
		case strings.Contains(errStr, "429"):
			return "", nil, errors.New("rate limit exceeded. Please try again later")
		case strings.Contains(errStr, "500"):
			return "", nil, errors.New("LLM API server error. Please try again later")
		case strings.Contains(errStr, "404"):
			return "", nil, errors.New("API endpoint not found. Please check the base URL and endpoint. OpenAI API uses /chat/completions")
		case strings.Contains(errStr, "context deadline exceeded") || strings.Contains(errStr, "context canceled"):
			return "", nil, fmt.Errorf("LLM request timed out after %d seconds", dc.llmTimeout)
		}
		return "", nil, fmt.Errorf("failed to connect to LLM API: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", nil, errors.New("no response choices returned from API")
	}

	outputText := response.Choices[0].Message.Content

	usage := &TokenUsage{
		PromptTokens:     int(response.Usage.PromptTokens),
		CompletionTokens: int(response.Usage.CompletionTokens),
	}

	dc.logger.Logf(
		"LLM API response: %d prompt tokens, %d completion tokens",
		usage.PromptTokens,
		usage.CompletionTokens,
	)

	return outputText, usage, nil
}
