package progress

import (
	"fmt"
	"io"

	"codeberg.org/maxwelljensen/llm_aggregator/internal/style"
)

// Logger is the minimal interface for progress and verbose logging implementations.
type Logger interface {
	Logf(format string, args ...any)
	Warningf(format string, args ...any)
	Debugf(format string, args ...any)
}

// Stage identifies a pipeline step. Runtime sends stages and the TUI maps them
// to its step display; sharing the constants keeps the two sides in lockstep.
type Stage string

const (
	StageAggregating    Stage = "Aggregating feeds"
	StageProcessing     Stage = "Processing articles"
	StageConnecting     Stage = "Connecting to LLM"
	StageGettingSummary Stage = "Getting summary"
)

// StageReporter receives pipeline stage transitions. Only runtimes that
// display progress (the TUI) implement it meaningfully.
type StageReporter interface {
	SetStage(stage Stage)
	SetSubStage(status string)
	SetArticleCount(total, processed int)
	SetTokenEstimate(total, used int)
	StartWaiting()
}

// Progress is the full reporting interface the pipeline driver talks to:
// logging plus stage reporting. Logging-only components should accept
// Logger instead so adapters only implement what they use.
type Progress interface {
	Logger
	StageReporter
}

// noStages provides empty stage reporting for adapters that only log.
type noStages struct{}

func (noStages) SetStage(Stage)            {}
func (noStages) SetSubStage(string)        {}
func (noStages) SetArticleCount(int, int)  {}
func (noStages) SetTokenEstimate(int, int) {}
func (noStages) StartWaiting()             {}

// SimpleLogger writes formatted output to an io.Writer.
type SimpleLogger struct {
	noStages
	writer io.Writer
	debug  bool
}

// NewSimpleLogger creates a SimpleLogger; set debug=true to enable Debugf output.
func NewSimpleLogger(writer io.Writer, debug bool) *SimpleLogger {
	return &SimpleLogger{
		writer: writer,
		debug:  debug,
	}
}

func (sl *SimpleLogger) Logf(format string, args ...any) {
	_, _ = fmt.Fprintf(sl.writer, format+"\n", args...)
}

func (sl *SimpleLogger) Warningf(format string, args ...any) {
	// Use style.Warningf for orange bold WARNING prefix
	_, _ = fmt.Fprintf(sl.writer, "%s\n", style.Warningf(format, args...))
}

func (sl *SimpleLogger) Debugf(format string, args ...any) {
	if sl.debug {
		// Use style.Debugf for styled debug output
		_, _ = fmt.Fprintf(sl.writer, "%s\n", style.Debugf(format, args...))
	}
}

// NoopLogger discards all output. Used as the default when no progress is desired.
type NoopLogger struct{}

func (nl *NoopLogger) Logf(format string, args ...any)      {}
func (nl *NoopLogger) Warningf(format string, args ...any)  {}
func (nl *NoopLogger) Debugf(format string, args ...any)    {}
func (nl *NoopLogger) SetStage(stage Stage)                 {}
func (nl *NoopLogger) SetSubStage(status string)            {}
func (nl *NoopLogger) SetArticleCount(total, processed int) {}
func (nl *NoopLogger) SetTokenEstimate(total, used int)     {}
func (nl *NoopLogger) StartWaiting()                        {}
