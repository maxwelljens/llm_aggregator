package cli

import (
	"os"
	"strings"
	"testing"

	"llm_aggregator/internal/style"
)

func TestShouldStyle(t *testing.T) {
	tests := []struct {
		name       string
		noColorVal string
		want       bool
	}{
		{"color enabled", "", true},
		{"color disabled via NO_COLOR", "1", false},
		{"color disabled via arbitrary value", "true", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old := os.Getenv("NO_COLOR")
			os.Setenv("NO_COLOR", tt.noColorVal)
			defer os.Setenv("NO_COLOR", old)

			got := shouldStyle()
			if got != tt.want {
				t.Errorf("shouldStyle() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderFlag(t *testing.T) {
	tests := []struct {
		name  string
		short string
		long  string
		want  string
	}{
		{
			name:  "with short and long flag",
			short: "-f",
			long:  "--feeds-file",
			want:  "-f, --feeds-file",
		},
		{
			name:  "long flag only",
			short: "",
			long:  "--version",
			want:  "--version",
		},
		{
			name:  "short and long with multiple dashes",
			short: "-n",
			long:  "--max-articles-per-feed",
			want:  "-n, --max-articles-per-feed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := renderFlag(tt.short, tt.long)
			if got != tt.want {
				t.Errorf("renderFlag(%q, %q) = %q, want %q", tt.short, tt.long, got, tt.want)
			}
		})
	}
}

func TestGetOptionValue(t *testing.T) {
	tests := []struct {
		name string
		opt  HelpOption
		want string
	}{
		{
			name: "type with default",
			opt:  HelpOption{Type: "int", Default: "10"},
			want: "int (default: 10)",
		},
		{
			name: "type without default",
			opt:  HelpOption{Type: "string"},
			want: "string",
		},
		{
			name: "default without type",
			opt:  HelpOption{Default: "deepseek-chat"},
			want: "deepseek-chat",
		},
		{
			name: "neither type nor default",
			opt:  HelpOption{},
			want: "",
		},
		{
			name: "choice type with default",
			opt:  HelpOption{Type: "text|markdown|json", Default: "text"},
			want: "text|markdown|json (default: text)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getOptionValue(tt.opt)
			if got != tt.want {
				t.Errorf("getOptionValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMaxFlagWidth(t *testing.T) {
	old := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer os.Setenv("NO_COLOR", old)

	sections := []HelpSection{
		{
			Title: "Required",
			Options: []HelpOption{
				{Short: "-f", Name: "--feeds-file"},
				{Short: "-p", Name: "--prompt"},
			},
		},
		{
			Title: "Optional",
			Options: []HelpOption{
				{Name: "--max-articles-per-feed"},
				{Name: "--max-total-articles"},
			},
		},
	}

	got := maxFlagWidth(sections)
	// "--max-articles-per-feed" is 23 chars
	if got != 23 {
		t.Errorf("maxFlagWidth() = %d, want 24", got)
	}
}

func TestFormatSection(t *testing.T) {
	// Test with NO_COLOR to avoid lipgloss dependency in assertions
	old := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer os.Setenv("NO_COLOR", old)

	section := HelpSection{
		Title: "Test Section",
		Options: []HelpOption{
			{
				Short:       "-f",
				Name:        "--feeds-file",
				Description: "Path to feeds file",
				Default:     "feeds.txt",
				Required:    true,
			},
			{
				Short:       "-v",
				Name:        "--verbose",
				Description: "Enable verbose output",
			},
		},
	}

	output := formatSection(section, 16, 0)

	if !strings.Contains(output, "Test Section") {
		t.Error("formatSection missing section title")
	}
	if !strings.Contains(output, "-f, --feeds-file") {
		t.Error("formatSection missing short+long flag")
	}
	if !strings.Contains(output, "feeds.txt") {
		t.Error("formatSection missing default value")
	}
	if !strings.Contains(output, "[required]") {
		t.Error("formatSection missing required marker")
	}
	if !strings.Contains(output, "Path to feeds file") {
		t.Error("formatSection missing description")
	}
	if !strings.Contains(output, "--verbose") {
		t.Error("formatSection missing long-only flag")
	}
}

func TestFormatSectionWithStyling(t *testing.T) {
	// Test with color enabled
	old := os.Getenv("NO_COLOR")
	os.Unsetenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", old)

	// Suppress the lipgloss warning during test
	_ = style.NoColor

	section := HelpSection{
		Title: "Styled Section",
		Options: []HelpOption{
			{
				Short:       "-o",
				Name:        "--output",
				Description: "Output format",
				Type:        "text|json|markdown",
				Default:     "text",
			},
		},
	}

	output := formatSection(section, 20, 0)

	if !strings.Contains(output, "Styled Section") {
		t.Error("formatSection missing section title")
	}
	if !strings.Contains(output, "text|json|markdown") {
		t.Error("formatSection missing type with default")
	}
}

func TestBuildStyledHelp(t *testing.T) {
	// Test without color
	old := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer os.Setenv("NO_COLOR", old)

	args := &Args{FeedsFile: "/tmp/feeds.txt", Prompt: "Summarise"}
	output := BuildStyledHelp(args)

	if !strings.Contains(output, "llm_aggregator") {
		t.Error("BuildStyledHelp missing program name")
	}
	if !strings.Contains(output, "Required Arguments") {
		t.Error("BuildStyledHelp missing Required Arguments section")
	}
	if !strings.Contains(output, "--feeds-file") {
		t.Error("BuildStyledHelp missing feeds-file flag")
	}
	if !strings.Contains(output, "--prompt") {
		t.Error("BuildStyledHelp missing prompt flag")
	}
	if !strings.Contains(output, "Examples:") {
		t.Error("BuildStyledHelp missing Examples section")
	}
}

func TestBuildStyledHelpWithColor(t *testing.T) {
	old := os.Getenv("NO_COLOR")
	os.Unsetenv("NO_COLOR")
	defer os.Setenv("NO_COLOR", old)

	args := &Args{FeedsFile: "/tmp/feeds.txt", Prompt: "Summarise"}
	output := BuildStyledHelp(args)

	// Should contain all expected sections even with styling
	if !strings.Contains(output, "Required Arguments") {
		t.Error("BuildStyledHelp missing Required Arguments")
	}
	if !strings.Contains(output, "LLM API Options") {
		t.Error("BuildStyledHelp missing LLM API Options")
	}
	if !strings.Contains(output, "Output Options") {
		t.Error("BuildStyledHelp missing Output Options")
	}
}

func TestWriteHelp(t *testing.T) {
	args := &Args{FeedsFile: "/tmp/feeds.txt", Prompt: "Summarise"}

	old := os.Getenv("NO_COLOR")
	os.Setenv("NO_COLOR", "1")
	defer os.Setenv("NO_COLOR", old)

	tmpFile, err := os.CreateTemp("", "help_test_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	w, err := os.OpenFile(tmpFile.Name(), os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("Failed to open temp file for write: %v", err)
	}
	defer w.Close()

	WriteHelp(args, w)

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if !strings.Contains(string(content), "llm_aggregator") {
		t.Error("WriteHelp did not write expected content")
	}
}

// Note: ParseArgs help/version branches (lines 86-95) call os.Exit
// and cannot be tested without refactoring to use an interface or
// callback for exit behavior.
