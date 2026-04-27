package progress

import (
	"fmt"
	"io"

	"llm_aggregator/internal/style"
)

// Logger is the minimal interface for progress and verbose logging implementations.
type Logger interface {
	Logf(format string, args ...any)
	Warningf(format string, args ...any)
	Debugf(format string, args ...any)
}

// Progress extends Logger with stage/count/timing reporting.
// Implementations: NoopLogger (no output), SimpleLogger (stdout), TUIProgress (tea messages).
type Progress interface {
	Logger
	SetStage(stage string)
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

func (sl *SimpleLogger) SetStage(stage string)                      {}
func (sl *SimpleLogger) SetSubStage(stage string)                   {}
func (sl *SimpleLogger) SetArticleCount(total, processed int)       {}
func (sl *SimpleLogger) SetTokenEstimate(total, used int)           {}
func (sl *SimpleLogger) StartWaiting()                            {}

// NoopLogger discards all output. Used as the default when no progress is desired.
type NoopLogger struct{}

func (nl *NoopLogger) Logf(format string, args ...any)      {}
func (nl *NoopLogger) Warningf(format string, args ...any)  {}
func (nl *NoopLogger) Debugf(format string, args ...any)    {}
func (nl *NoopLogger) SetStage(stage string)                      {}
func (nl *NoopLogger) SetSubStage(status string)                  {}
func (nl *NoopLogger) SetArticleCount(total, processed int)        {}
func (nl *NoopLogger) SetTokenEstimate(total, used int)           {}
func (nl *NoopLogger) StartWaiting()                            {}

// Context wraps a Logger and guards against nil receiver.
type Context struct {
	logger Logger
}

// NewContext wraps a Logger. Passing nil produces a no-op context.
func NewContext(logger Logger) *Context {
	return &Context{
		logger: logger,
	}
}

// Logf logs a message
func (c *Context) Logf(format string, args ...any) {
	if c.logger != nil {
		c.logger.Logf(format, args...)
	}
}

// Warningf logs a warning
func (c *Context) Warningf(format string, args ...any) {
	if c.logger != nil {
		c.logger.Warningf(format, args...)
	}
}

// Debugf logs a debug message
func (c *Context) Debugf(format string, args ...any) {
	if c.logger != nil {
		c.logger.Debugf(format, args...)
	}
}
