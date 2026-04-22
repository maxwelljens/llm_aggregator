package tokeniser

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pkoukk/tiktoken-go"
)

// Encoding names for different model families
const (
	EncodingCl100kBase = "cl100k_base" // GPT-4, GPT-3.5-turbo, embeddings
	EncodingO200kBase  = "o200k_base"  // GPT-4o, GPT-4.1, GPT-4.5
	EncodingP50kBase   = "p50k_base"   // Codex models, davinci-002, davinci-003
	EncodingR50kBase   = "r50k_base"   // GPT-3 models
)

// tokenizerCache caches encodings to avoid repeated initialization overhead
type tokenizerCache struct {
	mu        sync.RWMutex
	encodings map[string]*tiktoken.Tiktoken
}

var (
	cache     = &tokenizerCache{encodings: make(map[string]*tiktoken.Tiktoken)}
	modelLock sync.RWMutex
)

// GetEncoding returns a Tiktoken encoding for the given encoding name.
func GetEncoding(encodingName string) (*tiktoken.Tiktoken, error) {
	// Check cache first
	cache.mu.RLock()
	if enc, ok := cache.encodings[encodingName]; ok {
		cache.mu.RUnlock()
		return enc, nil
	}
	cache.mu.RUnlock()

	// Initialise encoding
	enc, err := tiktoken.GetEncoding(encodingName)
	if err != nil {
		return nil, fmt.Errorf("failed to get encoding %s: %w", encodingName, err)
	}

	// Cache it
	cache.mu.Lock()
	cache.encodings[encodingName] = enc
	cache.mu.Unlock()

	return enc, nil
}

// EncodingForModel returns the appropriate encoding name for a given model.
func EncodingForModel(model string) (string, error) {
	model = strings.ToLower(model)

	// GPT-4o and newer use o200k_base
	if strings.Contains(model, "gpt-4o") ||
		strings.Contains(model, "gpt-4.1") ||
		strings.Contains(model, "gpt-4.5") ||
		strings.HasPrefix(model, "gpt-4o") ||
		strings.HasPrefix(model, "gpt-4.1") ||
		strings.HasPrefix(model, "gpt-4.5") {
		return EncodingO200kBase, nil
	}

	// GPT-4 family uses cl100k_base
	if strings.HasPrefix(model, "gpt-4") {
		return EncodingCl100kBase, nil
	}

	// GPT-3.5-turbo uses cl100k_base
	if strings.Contains(model, "gpt-3.5-turbo") {
		return EncodingCl100kBase, nil
	}

	// Embedding models use cl100k_base
	if strings.Contains(model, "text-embedding") {
		return EncodingCl100kBase, nil
	}

	// Codex and davinci models use p50k_base
	if strings.Contains(model, "code-") ||
		strings.Contains(model, "davinci") {
		return EncodingP50kBase, nil
	}

	// Default to cl100k_base (most common for modern models)
	return EncodingCl100kBase, nil
}

// CountTokens counts the tokens in a text string using the appropriate encoding.
func CountTokens(text string, model string) (int, error) {
	encodingName, err := EncodingForModel(model)
	if err != nil {
		return 0, err
	}

	enc, err := GetEncoding(encodingName)
	if err != nil {
		return 0, fmt.Errorf("failed to get encoding: %w", err)
	}

	tokens := enc.Encode(text, nil, nil)
	return len(tokens), nil
}

// CountMessagesTokens counts tokens for chat API messages.
// Based on OpenAI's official token counting method.
func CountMessagesTokens(messages []Message, model string) (int, error) {
	encodingName, err := EncodingForModel(model)
	if err != nil {
		return 0, err
	}

	enc, err := GetEncoding(encodingName)
	if err != nil {
		return 0, fmt.Errorf("failed to get encoding: %w", err)
	}

	var tokensPerMessage, tokensPerName int
	switch {
	case strings.Contains(model, "gpt-3.5-turbo-0613"),
		strings.Contains(model, "gpt-3.5-turbo-16k-0613"),
		strings.Contains(model, "gpt-4-0314"),
		strings.Contains(model, "gpt-4-32k-0314"),
		strings.Contains(model, "gpt-4-0613"),
		strings.Contains(model, "gpt-4-32k-0613"):
		tokensPerMessage = 3
		tokensPerName = 1
	case strings.Contains(model, "gpt-3.5-turbo-0301"):
		tokensPerMessage = 4
		tokensPerName = -1
	default:
		// Use default values for other models
		tokensPerMessage = 3
		tokensPerName = 1
	}

	numTokens := 0
	for _, msg := range messages {
		numTokens += tokensPerMessage
		numTokens += len(enc.Encode(msg.Content, nil, nil))
		numTokens += len(enc.Encode(msg.Role, nil, nil))
		numTokens += len(enc.Encode(msg.Name, nil, nil))
		if msg.Name != "" {
			numTokens += tokensPerName
		}
	}
	numTokens += 3 // Every reply is primed with <|start|>assistant<|message|>

	return numTokens, nil
}

// Message represents a chat message for token counting.
type Message struct {
	Role    string
	Content string
	Name    string
}
