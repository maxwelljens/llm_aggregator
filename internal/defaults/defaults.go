package defaults

// Default values used throughout the application.
// Kept in one place so they are consistent across CLI, config, and runtime.

const (
	// DefaultModel is the default LLM model.
	DefaultModel = "deepseek-chat"
	// DefaultBaseURL is the default API base URL.
	DefaultBaseURL = "https://api.deepseek.com"
	// DefaultMaxTokens is the default max tokens for LLM responses.
	DefaultMaxTokens = 4000
	// DefaultTemperature is the default sampling temperature.
	DefaultTemperature = 0.7
	// DefaultLLMTimeout is the default LLM request timeout in seconds.
	DefaultLLMTimeout = 300

	// DefaultMaxArticlesPerFeed is the default max articles per feed.
	DefaultMaxArticlesPerFeed = 10
	// DefaultMaxDaysOld is the default max age for articles in days.
	DefaultMaxDaysOld = 7
	// DefaultMaxTotalArticles is the default max total articles.
	DefaultMaxTotalArticles = 20

	// DefaultMaxContentLength caps raw article content extracted from feeds.
	DefaultMaxContentLength = 5000
	// DefaultMaxContentPerArticle caps per-article content sent to the LLM.
	DefaultMaxContentPerArticle = 3000

	// TruncatedSuffix is appended to content shortened to a maximum length.
	TruncatedSuffix = "... [truncated]"

	// DefaultOutput is the default output format.
	DefaultOutput = "text"
	// DefaultIncludeArticles is the default for including articles.
	DefaultIncludeArticles = false
)

// DefaultSystemPrompt is the default system prompt used for LLM summarisation.
const DefaultSystemPrompt = `You are an expert analyst and summariser.
You analyse content from multiple sources and provide
concise, insightful summaries based on user requests.
Focus on key points, trends, and important information.`
