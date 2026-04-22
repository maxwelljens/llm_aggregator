package llm

import (
	"context"
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
	client      openai.Client
	model       string
	maxTokens   int
	temperature float64
	logger      *progress.Context
}

// NewLLMClient creates a new LLM client.
// apiKey: LLM API key (or read from LLM_AGGREGATOR_API_KEY env var)
// baseURL: API base URL (defaults to "https://api.deepseek.com")
// model: Model to use (defaults to "deepseek-chat")
// maxTokens: Maximum tokens in response (defaults to 4000)
// temperature: Sampling temperature (0.0 to 1.0, defaults to 0.7)
func NewLLMClient(apiKey, baseURL, model string, maxTokens int, temperature float64) (*LLMClient, error) {
	// Get API key from parameter or environment variable
	if apiKey == "" {
		apiKey = os.Getenv("LLM_AGGREGATOR_API_KEY")
	}
	if apiKey == "" || strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf(
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
	if temperature == 0 {
		temperature = defaults.DefaultTemperature
	}

	// Create OpenAI client configured for LLM
	clientOpts := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithBaseURL(baseURL),
	}

	client := openai.NewClient(clientOpts...)

	return &LLMClient{
		client:      client,
		model:       model,
		maxTokens:   maxTokens,
		temperature: temperature,
	}, nil
}

// SetLogger sets the logger for the LLM client
func (dc *LLMClient) SetLogger(logger *progress.Context) {
	dc.logger = logger
}

// SummariseArticles summarises a list of articles based on user prompt.
// articles: List of article maps
// userPrompt: User's query/summarisation request
// systemPrompt: Optional system prompt (defaults to helpful assistant)
func (dc *LLMClient) SummariseArticles(
	articles []map[string]any,
	userPrompt string,
	systemPrompt string,
) (string, error) {
	if len(articles) == 0 {
		return "No articles to summarise.", nil
	}

	// Prepare context from articles
	context := dc.prepareContext(articles)

	// Create messages for chat completion API
	messages := dc.createMessages(context, userPrompt, systemPrompt)

	// Call API with messages
	return dc.callAPIWithMessages(messages)
}

func (dc *LLMClient) prepareContext(articles []map[string]any) string {
	contextParts := []string{}

	for i, article := range articles {
		contextParts = append(contextParts, fmt.Sprintf("--- ARTICLE %d ---", i+1))
		contextParts = append(contextParts, fmt.Sprintf("Title: %s", article["title"]))

		if source, ok := article["source_feed"].(string); ok && source != "" {
			contextParts = append(contextParts, fmt.Sprintf("Source: %s", source))
		}

		if published, ok := article["published"]; ok {
			switch pub := published.(type) {
			case time.Time:
				if !pub.IsZero() {
					contextParts = append(contextParts, fmt.Sprintf("Published: %s", pub.Format(time.RFC3339)))
				}
			case string:
				contextParts = append(contextParts, fmt.Sprintf("Published: %s", pub))
			default:
				contextParts = append(contextParts, fmt.Sprintf("Published: %v", pub))
			}
		}

		if author, ok := article["author"].(string); ok && author != "" {
			contextParts = append(contextParts, fmt.Sprintf("Author: %s", author))
		}

		if link, ok := article["link"].(string); ok && link != "" {
			contextParts = append(contextParts, fmt.Sprintf("Link: %s", link))
		}

		if content, ok := article["content"].(string); ok && content != "" {
			// Truncate very long content
			maxContentLen := 3000
			if len(content) > maxContentLen {
				content = content[:maxContentLen] + "... [truncated]"
			}
			contextParts = append(contextParts, fmt.Sprintf("Content: %s", content))
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

func (dc *LLMClient) callAPIWithMessages(messages []openai.ChatCompletionMessageParamUnion) (string, error) {
	ctx := context.Background()

	if dc.logger != nil {
		dc.logger.Logf("Calling LLM API with model: %s", dc.model)
	}

	response, err := dc.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       dc.model,
		Messages:    messages,
		MaxTokens:   openai.Int(int64(dc.maxTokens)),
		Temperature: openai.Float(dc.temperature),
	})

	if err != nil {
		// Provide more specific error messages
		errStr := err.Error()
		if strings.Contains(errStr, "401") {
			return "", fmt.Errorf("invalid API key. Please check your LLM API key")
		} else if strings.Contains(errStr, "429") {
			return "", fmt.Errorf("rate limit exceeded. Please try again later")
		} else if strings.Contains(errStr, "500") {
			return "", fmt.Errorf("LLM API server error. Please try again later")
		} else if strings.Contains(errStr, "404") {
			return "", fmt.Errorf("API endpoint not found. Please check the base URL and endpoint. OpenAI API uses /chat/completions")
		}
		return "", fmt.Errorf("failed to connect to LLM API: %w", err)
	}

	// Extract text from response
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned from API")
	}

	outputText := response.Choices[0].Message.Content

	// Print token usage
	if dc.logger != nil {
		dc.logger.Logf(
			"LLM API response: %d prompt tokens, %d completion tokens",
			response.Usage.PromptTokens,
			response.Usage.CompletionTokens,
		)
	}

	return outputText, nil
}

// AnalyseWithCustomPrompt analyses articles with custom system and user prompts.
func (dc *LLMClient) AnalyseWithCustomPrompt(
	articles []map[string]any,
	systemPrompt string,
	userPrompt string,
) (string, error) {
	context := dc.prepareContext(articles)

	// Create messages for chat completion API
	messages := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage(systemPrompt),
		openai.UserMessage(fmt.Sprintf(`Here are articles from various RSS feeds:

%s

%s`, context, userPrompt)),
	}

	return dc.callAPIWithMessages(messages)
}
