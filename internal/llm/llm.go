package llm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"llm_aggregator/internal/defaults"
	"llm_aggregator/internal/progress"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
)

// LLMClient is a client for interacting with LLM API.
type LLMClient struct {
	client     openai.Client
	model      string
	maxTokens  int
	temperature *float64
	llmTimeout int // seconds; 0 means no timeout
	logger     *progress.Context
}

// NewLLMClient creates an LLM API client.
// Set apiKey to "" to read from LLM_AGGREGATOR_API_KEY.
func NewLLMClient(apiKey, baseURL, model string, maxTokens int, temperature *float64, timeoutSeconds int) (*LLMClient, error) {
	// Get API key from parameter or environment variable
	if apiKey == "" {
		apiKey = os.Getenv("LLM_AGGREGATOR_API_KEY")
	}
	if apiKey == "" || strings.TrimSpace(apiKey) == "" {
		return nil, errors.New(
			"LLM API key is required. " +
				"Set LLM_AGGREGATOR_API_KEY environment variable or pass apiKey parameter",
		)
	}

	// Set defaults using central constants
	if baseURL == "" {
		baseURL = defaults.DefaultBaseURL
	}
	if model == "" {
		model = defaults.DefaultModel
	}
	if maxTokens == 0 {
		maxTokens = defaults.DefaultMaxTokens
	}
	if temperature == nil {
		t := defaults.DefaultTemperature
		temperature = &t
	}
	if timeoutSeconds == 0 {
		timeoutSeconds = defaults.DefaultLLMTimeout
	}

	// Create OpenAI client configured for LLM
	clientOpts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}

	client := openai.NewClient(clientOpts...)

	return &LLMClient{
		client:     client,
		model:      model,
		maxTokens:  maxTokens,
		temperature: temperature,
		llmTimeout: timeoutSeconds,
	}, nil
}

// SetLogger sets the logger for the LLM client
func (dc *LLMClient) SetLogger(logger *progress.Context) {
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
	articles []map[string]any,
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

func (dc *LLMClient) prepareContext(articles []map[string]any) string {
	contextParts := []string{}

	for i, article := range articles {
		contextParts = append(contextParts, fmt.Sprintf("--- ARTICLE %d ---", i+1))
		contextParts = append(contextParts, fmt.Sprintf("Title: %s", article["title"]))

		if source, ok := article["source_feed"].(string); ok && source != "" {
			contextParts = append(contextParts, "Source: "+source)
		}

		if published, ok := article["published"]; ok {
			switch pub := published.(type) {
			case time.Time:
				if !pub.IsZero() {
					contextParts = append(contextParts, "Published: "+pub.Format(time.RFC3339))
				}
			case string:
				contextParts = append(contextParts, "Published: "+pub)
			default:
				contextParts = append(contextParts, fmt.Sprintf("Published: %v", pub))
			}
		}

		if author, ok := article["author"].(string); ok && author != "" {
			contextParts = append(contextParts, "Author: "+author)
		}

		if link, ok := article["link"].(string); ok && link != "" {
			contextParts = append(contextParts, "Link: "+link)
		}

		if content, ok := article["content"].(string); ok && content != "" {
			// Truncate very long content
			maxContentLen := 3000
			if len(content) > maxContentLen {
				content = content[:maxContentLen] + "... [truncated]"
			}
			contextParts = append(contextParts, "Content: "+content)
		}

		contextParts = append(contextParts, "") // Empty line between articles
	}

	return strings.Join(contextParts, "\n")
}

func (dc *LLMClient) createMessages(context, userPrompt, systemPrompt string) []openai.ChatCompletionMessageParamUnion {
	if systemPrompt == "" {
		systemPrompt = `You are an expert analyst and summariser.
You analyse content from multiple sources and provide
concise, insightful summaries based on user requests.
Focus on key points, trends, and important information.`
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
	if dc.logger != nil {
		dc.logger.Logf("Calling LLM API with model: %s", dc.model)
		dc.logger.Logf("Max tokens: %d, Temperature: %.2f", dc.maxTokens, *dc.temperature)
		dc.logger.Logf("Messages count: %d", len(messages))
	}

	response, err := dc.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       dc.model,
		Messages:    messages,
		MaxTokens:   openai.Int(int64(dc.maxTokens)),
		Temperature: openai.Float(*dc.temperature),
	})

	if err != nil {
		errStr := err.Error()

		if dc.logger != nil {
			dc.logger.Logf("API error: %s", errStr)
		}

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

	if dc.logger != nil {
		dc.logger.Logf(
			"LLM API response: %d prompt tokens, %d completion tokens",
			usage.PromptTokens,
			usage.CompletionTokens,
		)
	}

	return outputText, usage, nil
}
