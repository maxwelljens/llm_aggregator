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

// Progress extends Logger with stage/count/timing reporting.
// Implementations: NoopLogger (no output), SimpleLogger (stdout), TUIProgress (tea messages).
type Progress interface {
	Logger
	SetStage(stage Stage)
	SetSubStage(status string)
	SetArticleCount(total, processed int)
	SetTokenEstimate(total, used int)
	StartWaiting()
}

// SimpleLogger writes formatted output to an io.Writer.
type SimpleLogger struct {
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

func (sl *SimpleLogger) SetStage(stage Stage)                 {}
func (sl *SimpleLogger) SetSubStage(stage string)             {}
func (sl *SimpleLogger) SetArticleCount(total, processed int) {}
func (sl *SimpleLogger) SetTokenEstimate(total, used int)     {}
func (sl *SimpleLogger) StartWaiting()                        {}

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
