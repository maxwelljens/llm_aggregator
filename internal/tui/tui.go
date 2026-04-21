package tui

import (
	"context"
	"fmt"
	"time"

	"llm_aggregator/internal/progress"
	"llm_aggregator/internal/runtime"

	tea "github.com/charmbracelet/bubbletea"
)

// TUIProgress implements the progress.Progress interface for TUI
type TUIProgress struct {
	program *tea.Program
	model   *Model
}

// NewTUIProgress creates a new TUIProgress
func NewTUIProgress() *TUIProgress {
	model := New()
	p := tea.NewProgram(model, tea.WithAltScreen())
	return &TUIProgress{
		program: p,
		model:   model,
	}
}

// Logf logs a message to the TUI
func (tp *TUIProgress) Logf(format string, args ...interface{}) {
	tp.program.Send(StepWithCounts(tp.model.currentStep, fmt.Sprintf(format, args...), 
		tp.model.articlesCount, tp.model.processedCount))
}

// Warningf logs a warning to the TUI
func (tp *TUIProgress) Warningf(format string, args ...interface{}) {
	tp.Logf("Warning: "+format, args...)
}

// Debugf logs a debug message to the TUI
func (tp *TUIProgress) Debugf(format string, args ...interface{}) {
	// Debug messages not shown in TUI by default
}

// SetStage sets the current stage
func (tp *TUIProgress) SetStage(stage string) {
	tp.program.Send(Step(tp.model.currentStep, stage))
}

// SetSubStage sets the sub-stage status
func (tp *TUIProgress) SetSubStage(status string) {
	tp.program.Send(Step(tp.model.currentStep, status))
}

// SetArticleCount sets the article counts
func (tp *TUIProgress) SetArticleCount(total, processed int) {
	tp.model.articlesCount = total
	tp.model.processedCount = processed
}

// Run runs the TUI
func (tp *TUIProgress) Run() error {
	_, err := tp.program.Run()
	return err
}

// Context returns a progress.Context for the TUI
func (tp *TUIProgress) Context() *progress.Context {
	return progress.NewContext(tp)
}

// RunWithRuntime runs the TUI with a runtime
func RunWithRuntime(rt *runtime.Runtime) error {
	// Create TUI progress
	tp := NewTUIProgress()
	
	// Run TUI in goroutine
	tuiDone := make(chan error)
	go func() {
		tuiDone <- tp.Run()
	}()

	// Run aggregation in another goroutine
	runDone := make(chan error)
	go func() {
		// Create context for cancellation
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Update TUI status
		tp.program.Send(Step(1, fmt.Sprintf("Aggregating feeds from: %s", rt.FeedsFile)))

		// TODO: Configure runtime with progress context
		
		// Execute the runtime
		err := rt.Execute(ctx)
		if err != nil {
			tp.program.Send(Error(err.Error()))
			runDone <- err
			return
		}

		// Update TUI status
		tp.program.Send(StepWithCounts(5, "Formatting output", len(rt.Articles), len(rt.Articles)))

		// Write output
		if rt.OutputFile != "" {
			if err := rt.WriteOutputToFile(); err != nil {
				tp.program.Send(Error(fmt.Sprintf("Error writing output: %v", err)))
				runDone <- err
				return
			}
			tp.program.Send(Step(6, fmt.Sprintf("Output written to: %s", rt.OutputFile)))
		} else {
			if err := rt.WriteOutputToFile(); err != nil {
				tp.program.Send(Error(fmt.Sprintf("Error writing output: %v", err)))
				runDone <- err
				return
			}
			tp.program.Send(Step(6, "Output complete"))
		}

		// Mark as done
		tp.program.Send(Step(7, "Processing completed successfully"))
		time.Sleep(2 * time.Second) // Show success message briefly
		tp.program.Send(tea.Quit())
		runDone <- nil
	}()

	// Wait for either TUI or run to complete
	select {
	case err := <-tuiDone:
		return err
	case err := <-runDone:
		if err != nil {
			return err
		}
		// Wait for TUI to quit gracefully
		select {
		case err := <-tuiDone:
			return err
		case <-time.After(5 * time.Second):
			return fmt.Errorf("TUI did not exit gracefully")
		}
	}
}