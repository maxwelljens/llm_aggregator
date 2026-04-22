package defaults

import (
	"testing"
)

func TestDefaultValues(t *testing.T) {
	tests := []struct {
		name     string
		got      any
		expected any
	}{
		{"DefaultModel", DefaultModel, "deepseek-chat"},
		{"DefaultBaseURL", DefaultBaseURL, "https://api.deepseek.com"},
		{"DefaultMaxTokens", DefaultMaxTokens, 4000},
		{"DefaultTemperature", DefaultTemperature, 0.7},
		{"DefaultMaxArticlesPerFeed", DefaultMaxArticlesPerFeed, 10},
		{"DefaultMaxDaysOld", DefaultMaxDaysOld, 7},
		{"DefaultMaxTotalArticles", DefaultMaxTotalArticles, 20},
		{"DefaultOutput", DefaultOutput, "text"},
		{"DefaultIncludeArticles", DefaultIncludeArticles, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("%s = %v, want %v", tt.name, tt.got, tt.expected)
			}
		})
	}
}

func TestDefaultSystemPrompt(t *testing.T) {
	expected := `You are an expert analyst and summariser.
You analyse content from multiple sources and provide
concise, insightful summaries based on user requests.
Focus on key points, trends, and important information.`

	if DefaultSystemPrompt != expected {
		t.Errorf("DefaultSystemPrompt doesn't match expected value")
	}
}

func TestDefaultValuesAreSensible(t *testing.T) {
	// Verify temperature is in valid range
	if DefaultTemperature < 0 || DefaultTemperature > 1 {
		t.Errorf("DefaultTemperature = %f, should be between 0 and 1", DefaultTemperature)
	}

	// Verify max tokens is reasonable
	if DefaultMaxTokens < 100 || DefaultMaxTokens > 100000 {
		t.Errorf("DefaultMaxTokens = %d, should be between 100 and 100000", DefaultMaxTokens)
	}

	// Verify article limits are reasonable
	if DefaultMaxArticlesPerFeed < 1 {
		t.Errorf("DefaultMaxArticlesPerFeed = %d, should be at least 1", DefaultMaxArticlesPerFeed)
	}
	if DefaultMaxTotalArticles < 1 {
		t.Errorf("DefaultMaxTotalArticles = %d, should be at least 1", DefaultMaxTotalArticles)
	}

	// Verify days old is non-negative
	if DefaultMaxDaysOld < 0 {
		t.Errorf("DefaultMaxDaysOld = %d, should be non-negative", DefaultMaxDaysOld)
	}
}

func TestDefaultBaseURLIsValid(t *testing.T) {
	// Base URL should be a valid URL starting with http
	if len(DefaultBaseURL) < 4 {
		t.Errorf("DefaultBaseURL = %q, too short", DefaultBaseURL)
	}
	if DefaultBaseURL[:4] != "http" {
		t.Errorf("DefaultBaseURL = %q, should start with http", DefaultBaseURL)
	}
}

func TestOutputFormatChoices(t *testing.T) {
	validOutputs := map[string]bool{
		"text":     true,
		"json":     true,
		"markdown": true,
	}

	if !validOutputs[DefaultOutput] {
		t.Errorf("DefaultOutput = %q, should be one of text, json, markdown", DefaultOutput)
	}
}
