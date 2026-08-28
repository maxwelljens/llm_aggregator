package style

import (
	"strings"
	"testing"
)

func TestNoColor(t *testing.T) {
	t.Run("NO_COLOR set means no colour", func(t *testing.T) {
		t.Setenv("NO_COLOR", "1")
		if !NoColor() {
			t.Error("NoColor() = false with NO_COLOR set, want true")
		}
	})

	t.Run("NO_COLOR unset enables colour", func(t *testing.T) {
		t.Setenv("NO_COLOR", "")
		if NoColor() {
			t.Error("NoColor() = true with NO_COLOR empty, want false")
		}
	})
}

// TestPlainOutputUnderNoColor verifies the no-colour fallbacks keep their
// human-readable prefixes and the original text.
func TestPlainOutputUnderNoColor(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	tests := []struct {
		name    string
		got     string
		contain string
	}{
		{"warning keeps prefix", Warning("check your feeds"), "WARNING: check your feeds"},
		{"warningf formats", Warningf("feed %s failed", "x"), "WARNING: feed x failed"},
		{"error keeps prefix", Error("bad key"), "ERROR: bad key"},
		{"errorf formats", Errorf("code %d", 401), "ERROR: code 401"},
		{"success keeps checkmark", Success("done"), "✓ done"},
		{"info keeps symbol", Info("working"), "ℹ working"},
		{"debug keeps prefix", Debug("detail"), "Debug: detail"},
		{"debugf formats", Debugf("n=%d", 5), "Debug: n=5"},
		{"filepath returned as-is", Filepath("/tmp/feeds.txt"), "/tmp/feeds.txt"},
		{"value returned as-is", Value("42"), "42"},
		{"label keeps text", Label("Configuration:"), "Configuration:"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !strings.Contains(tt.got, tt.contain) {
				t.Errorf("output %q missing %q", tt.got, tt.contain)
			}
		})
	}
}

func TestItalicAndHeading(t *testing.T) {
	t.Setenv("NO_COLOR", "1")

	if got := Italic("text"); !strings.Contains(got, "text") {
		t.Errorf("Italic output %q missing text", got)
	}
	if got := Heading("Title"); !strings.Contains(got, "Title") {
		t.Errorf("Heading output %q missing text", got)
	}
}
