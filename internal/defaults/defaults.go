package defaults

// Default values for the llm_aggregator application.
//
// These constants are used throughout the codebase to ensure
// consistent default values across CLI arguments, configuration,
// and runtime settings.

const (
	// LLM API defaults
	DefaultModel       = "deepseek-chat"
	DefaultBaseURL     = "https://api.deepseek.com"
	DefaultMaxTokens   = 4000
	DefaultTemperature = 0.7

	// Feed aggregation defaults
	DefaultMaxArticlesPerFeed = 10
	DefaultMaxDaysOld         = 7
	DefaultMaxTotalArticles   = 20

	// Output defaults
	DefaultOutput         = "text"
	DefaultIncludeArticles = false
)

// DefaultSystemPrompt is the default system prompt used for LLM summarisation.
const DefaultSystemPrompt = `You are an expert analyst and summariser.
You analyse content from multiple sources and provide
concise, insightful summaries based on user requests.
Focus on key points, trends, and important information.`
