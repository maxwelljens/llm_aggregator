package progress

import (
	"bytes"
	"strings"
	"testing"
)

// Compile-time assertions: SimpleLogger satisfies the full Progress
// interface (logging plus stage reporting) even though it only logs.
var (
	_ Logger   = (*SimpleLogger)(nil)
	_ Progress = (*SimpleLogger)(nil)
	_ Logger   = (*NoopLogger)(nil)
	_ Progress = (*NoopLogger)(nil)
)

func TestSimpleLogger_Logf(t *testing.T) {
	var buf bytes.Buffer
	sl := NewSimpleLogger(&buf, false)

	sl.Logf("found %d items", 3)

	if got := buf.String(); got != "found 3 items\n" {
		t.Errorf("output = %q, want %q", got, "found 3 items\n")
	}
}

func TestSimpleLogger_Warningf(t *testing.T) {
	var buf bytes.Buffer
	sl := NewSimpleLogger(&buf, false)

	sl.Warningf("feed %s failed", "http://x")

	if !strings.Contains(buf.String(), "WARNING:") {
		t.Errorf("output %q missing WARNING prefix", buf.String())
	}
	if !strings.Contains(buf.String(), "feed http://x failed") {
		t.Errorf("output %q missing formatted message", buf.String())
	}
}

func TestSimpleLogger_Debugf(t *testing.T) {
	tests := []struct {
		name      string
		debug     bool
		wantEmpty bool
	}{
		{"debug disabled discards output", false, true},
		{"debug enabled writes output", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			sl := NewSimpleLogger(&buf, tt.debug)

			sl.Debugf("detail %d", 7)

			if tt.wantEmpty && buf.Len() != 0 {
				t.Errorf("output = %q, want empty", buf.String())
			}
			if !tt.wantEmpty && !strings.Contains(buf.String(), "detail 7") {
				t.Errorf("output %q missing message", buf.String())
			}
		})
	}
}

// TestSimpleLogger_StageReportingIsSilent verifies the stage half of the
// interface is accepted but produces no output for a plain logger.
func TestSimpleLogger_StageReportingIsSilent(t *testing.T) {
	var buf bytes.Buffer
	sl := NewSimpleLogger(&buf, true)

	var p Progress = sl
	p.SetStage(StageAggregating)
	p.SetSubStage("sub")
	p.SetArticleCount(10, 2)
	p.SetTokenEstimate(100, 50)
	p.StartWaiting()

	if buf.Len() != 0 {
		t.Errorf("stage reporting wrote %q, want no output", buf.String())
	}
}

// TestNoopLogger_DiscardsEverything is a smoke test: every method runs
// without panicking and there is no output to observe.
func TestNoopLogger_DiscardsEverything(t *testing.T) {
	var nl Progress = &NoopLogger{}
	nl.Logf("log")
	nl.Warningf("warn")
	nl.Debugf("debug")
	nl.SetStage(StageProcessing)
	nl.SetSubStage("sub")
	nl.SetArticleCount(1, 0)
	nl.SetTokenEstimate(1, 1)
	nl.StartWaiting()
}
