package defaults

// Default values for the llm_aggregator application.
//
// These constants are used throughout the codebase to ensure
// consistent default values across CLI arguments, configuration,
// and runtime settings.

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
