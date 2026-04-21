package progress

import (
	"fmt"
	"io"
)

// Logger provides logging interface for the aggregator
type Logger interface {
	Logf(format string, args ...any)
	Warningf(format string, args ...any)
	Debugf(format string, args ...any)
}

// Progress provides progress reporting interface for TUI
type Progress interface {
	Logger
	SetStage(stage string)
	SetSubStage(status string)
	SetArticleCount(total, processed int)
}

// SimpleLogger implements Logger with io.Writer output
type SimpleLogger struct {
	writer io.Writer
	debug  bool
}

// NewSimpleLogger creates a new SimpleLogger
func NewSimpleLogger(writer io.Writer, debug bool) *SimpleLogger {
	return &SimpleLogger{
		writer: writer,
		debug:  debug,
	}
}

func (sl *SimpleLogger) Logf(format string, args ...any) {
	fmt.Fprintf(sl.writer, format+"\n", args...)
}

func (sl *SimpleLogger) Warningf(format string, args ...any) {
	fmt.Fprintf(sl.writer, "Warning: "+format+"\n", args...)
}

func (sl *SimpleLogger) Debugf(format string, args ...any) {
	if sl.debug {
		fmt.Fprintf(sl.writer, "Debug: "+format+"\n", args...)
	}
}

// NoopLogger implements Logger with no output
type NoopLogger struct{}

func (nl *NoopLogger) Logf(format string, args ...any)     {}
func (nl *NoopLogger) Warningf(format string, args ...any) {}
func (nl *NoopLogger) Debugf(format string, args ...any)   {}

// Context provides progress context for components
type Context struct {
	logger Logger
}

// NewContext creates a new progress context
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

