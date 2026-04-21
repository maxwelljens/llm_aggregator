package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	
	if cfg.MaxArticlesPerFeed != 10 {
		t.Errorf("Expected MaxArticlesPerFeed = 10, got %d", cfg.MaxArticlesPerFeed)
	}
	
	if cfg.MaxDaysOld != 7 {
		t.Errorf("Expected MaxDaysOld = 7, got %d", cfg.MaxDaysOld)
	}
	
	if cfg.MaxTotalArticles != 20 {
		t.Errorf("Expected MaxTotalArticles = 20, got %d", cfg.MaxTotalArticles)
	}
	
	if cfg.Model != "deepseek-chat" {
		t.Errorf("Expected Model = 'deepseek-chat', got %s", cfg.Model)
	}
	
	if cfg.MaxTokens != 4000 {
		t.Errorf("Expected MaxTokens = 4000, got %d", cfg.MaxTokens)
	}
	
	if cfg.Temperature != 0.7 {
		t.Errorf("Expected Temperature = 0.7, got %f", cfg.Temperature)
	}
	
	expectedPrompt := `You are an expert analyst and summariser.
You analyse content from multiple sources and provide
concise, insightful summaries based on user requests.
Focus on key points, trends, and important information.`
	
	if cfg.SystemPrompt != expectedPrompt {
		t.Errorf("SystemPrompt doesn't match expected default")
	}
	
	if cfg.Output != "text" {
		t.Errorf("Expected Output = 'text', got %s", cfg.Output)
	}
}

func TestConfigPath(t *testing.T) {
	path, err := GetConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}
	
	// Should end with .config/llm_aggregator/config.toml
	expectedSuffix := filepath.Join(".config", "llm_aggregator", "config.toml")
	if !hasSuffix(path, expectedSuffix) {
		t.Errorf("Config path %s doesn't end with %s", path, expectedSuffix)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	oldEnv := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldEnv)
	
	cfg := DefaultConfig()
	cfg.SystemPrompt = "Test system prompt"
	cfg.Model = "test-model"
	cfg.MaxTokens = 1234
	
	// Save config
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// Verify file exists
	configPath, err := GetConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}
	
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file not created at %s", configPath)
	}
	
	// Load config
	loadedCfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	if !loadedCfg.IsLoaded() {
		t.Error("Config should be marked as loaded")
	}
	
	if loadedCfg.SystemPrompt != cfg.SystemPrompt {
		t.Errorf("SystemPrompt mismatch: expected %q, got %q", cfg.SystemPrompt, loadedCfg.SystemPrompt)
	}
	
	if loadedCfg.Model != cfg.Model {
		t.Errorf("Model mismatch: expected %s, got %s", cfg.Model, loadedCfg.Model)
	}
	
	if loadedCfg.MaxTokens != cfg.MaxTokens {
		t.Errorf("MaxTokens mismatch: expected %d, got %d", cfg.MaxTokens, loadedCfg.MaxTokens)
	}
}

func TestEnvironmentVariables(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	oldEnv := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldEnv)
	
	// Set environment variables
	os.Setenv("LLM_AGGREGATOR_SYSTEM_PROMPT", "Env system prompt")
	os.Setenv("LLM_AGGREGATOR_MODEL", "env-model")
	os.Setenv("LLM_AGGREGATOR_MAX_TOKENS", "9999")
	defer func() {
		os.Unsetenv("LLM_AGGREGATOR_SYSTEM_PROMPT")
		os.Unsetenv("LLM_AGGREGATOR_MODEL")
		os.Unsetenv("LLM_AGGREGATOR_MAX_TOKENS")
	}()
	
	// Load config (should pick up env vars)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	
	if cfg.SystemPrompt != "Env system prompt" {
		t.Errorf("SystemPrompt should be from env var, got %q", cfg.SystemPrompt)
	}
	
	if cfg.Model != "env-model" {
		t.Errorf("Model should be from env var, got %s", cfg.Model)
	}
	
	if cfg.MaxTokens != 9999 {
		t.Errorf("MaxTokens should be from env var, got %d", cfg.MaxTokens)
	}
}

func TestConfigExists(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	oldEnv := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", oldEnv)
	
	// Get config path to verify
	configPath, err := GetConfigPath()
	if err != nil {
		t.Fatalf("Failed to get config path: %v", err)
	}
	
	// Ensure it's in our temp directory
	if !contains(configPath, tempDir) {
		t.Logf("Config path %s not in temp dir %s", configPath, tempDir)
	}
	
	// Initially should not exist
	exists, err := ConfigExists()
	if err != nil {
		t.Fatalf("Failed to check config existence: %v", err)
	}
	
	if exists {
		// Config exists, maybe from previous test run
		t.Logf("Config exists at %s, removing...", configPath)
		os.Remove(configPath)
		// Check again
		exists, err = ConfigExists()
		if err != nil {
			t.Fatalf("Failed to check config existence after removal: %v", err)
		}
		if exists {
			t.Error("Config should not exist after removal")
		}
	}
	
	// Create config
	cfg := DefaultConfig()
	if err := cfg.Save(); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}
	
	// Now should exist
	exists, err = ConfigExists()
	if err != nil {
		t.Fatalf("Failed to check config existence: %v", err)
	}
	
	if !exists {
		t.Error("Config should exist after saving")
	}
}

// Helper function to check if path has suffix
func hasSuffix(path, suffix string) bool {
	if len(path) < len(suffix) {
		return false
	}
	return path[len(path)-len(suffix):] == suffix
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(s[0:len(substr)] == substr || contains(s[1:], substr)))
}