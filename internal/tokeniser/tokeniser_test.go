package tokeniser

import (
	"strings"
	"testing"
)

func TestEncodingForModel(t *testing.T) {
	tests := []struct {
		name     string
		model    string
		expected string
	}{
		{"gpt-4o uses o200k", "gpt-4o", EncodingO200kBase},
		{"gpt-4o-mini uses o200k", "gpt-4o-mini", EncodingO200kBase},
		{"gpt-4.1 uses o200k", "gpt-4.1", EncodingO200kBase},
		{"gpt-4 uses cl100k", "gpt-4", EncodingCl100kBase},
		{"gpt-4-turbo uses cl100k", "gpt-4-turbo", EncodingCl100kBase},
		{"gpt-3.5-turbo uses cl100k", "gpt-3.5-turbo", EncodingCl100kBase},
		{"embedding models use cl100k", "text-embedding-3-small", EncodingCl100kBase},
		{"codex models use p50k", "code-davinci-002", EncodingP50kBase},
		{"davinci models use p50k", "text-davinci-003", EncodingP50kBase},
		{"unknown models fall back to cl100k", "deepseek-chat", EncodingCl100kBase},
		{"matching is case-insensitive", "GPT-4O-MINI", EncodingO200kBase},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EncodingForModel(tt.model)
			if err != nil {
				t.Fatalf("EncodingForModel(%q) returned unexpected error: %v", tt.model, err)
			}
			if got != tt.expected {
				t.Errorf("EncodingForModel(%q) = %q, want %q", tt.model, got, tt.expected)
			}
		})
	}
}

func TestGetEncoding(t *testing.T) {
	t.Run("valid encoding is returned", func(t *testing.T) {
		enc, err := GetEncoding(EncodingCl100kBase)
		if err != nil {
			t.Fatalf("GetEncoding returned unexpected error: %v", err)
		}
		if enc == nil {
			t.Fatal("GetEncoding returned nil encoding")
		}
	})

	t.Run("cached encoding is the same instance", func(t *testing.T) {
		first, err := GetEncoding(EncodingCl100kBase)
		if err != nil {
			t.Fatalf("first GetEncoding error: %v", err)
		}
		second, err := GetEncoding(EncodingCl100kBase)
		if err != nil {
			t.Fatalf("second GetEncoding error: %v", err)
		}
		if first != second {
			t.Error("cached encoding is a different instance")
		}
	})

	t.Run("invalid encoding returns an error", func(t *testing.T) {
		if _, err := GetEncoding("not-a-real-encoding"); err == nil {
			t.Error("expected error for invalid encoding name, got nil")
		}
	})
}

func TestCountTokens(t *testing.T) {
	t.Run("empty text counts zero tokens", func(t *testing.T) {
		n, err := CountTokens("", "deepseek-chat")
		if err != nil {
			t.Fatalf("CountTokens returned unexpected error: %v", err)
		}
		if n != 0 {
			t.Errorf("CountTokens(\"\") = %d, want 0", n)
		}
	})

	t.Run("known text counts deterministically", func(t *testing.T) {
		first, err := CountTokens("hello world", "deepseek-chat")
		if err != nil {
			t.Fatalf("CountTokens returned unexpected error: %v", err)
		}
		second, err := CountTokens("hello world", "deepseek-chat")
		if err != nil {
			t.Fatalf("repeat CountTokens returned unexpected error: %v", err)
		}
		if first == 0 {
			t.Fatal("token count is zero for non-empty text")
		}
		if first != second {
			t.Errorf("count not stable: %d then %d", first, second)
		}
	})

	t.Run("longer text counts more tokens", func(t *testing.T) {
		short, _ := CountTokens("hi", "gpt-4o")
		long, _ := CountTokens(strings.Repeat("supercalifragilistic ", 20), "gpt-4o")
		if long <= short {
			t.Errorf("long text (%d tokens) should outnumber short text (%d)", long, short)
		}
	})
}
