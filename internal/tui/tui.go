package tui

import (
	"fmt"

	"llm_aggregator/internal/progress"

	tea "github.com/charmbracelet/bubbletea"
)

// TUIProgress implements the progress.Progress interface for TUI
type TUIProgress struct {
	program *tea.Program
}

// NewTUIProgress creates a new TUIProgress.
// It now accepts the program instance instead of creating one.
func NewTUIProgress(p *tea.Program) *TUIProgress {
	return &TUIProgress{
		program: p,
	}
}

// SetStage sends a message to update the main stage.
func (tp *TUIProgress) SetStage(stage string) {
	tp.program.Send(StageMsg(stage))
}

// SetSubStage sends a message to update the sub-status.
func (tp *TUIProgress) SetSubStage(status string) {
	tp.program.Send(SubStageMsg(status))
}

// SetArticleCount sends a message to update the article counts.
func (tp *TUIProgress) SetArticleCount(total, processed int) {
	tp.program.Send(ArticleCountMsg{Total: total, Processed: processed})
}

// SetTokenEstimate sends a message to update the token estimates.
func (tp *TUIProgress) SetTokenEstimate(total, used int) {
	tp.program.Send(TokenEstimateMsg{Total: total, Used: used})
}

// StartWaiting signals that we're waiting for the LLM response.
func (tp *TUIProgress) StartWaiting() {
	tp.program.Send(WaitingMsg{})
}

// Logf sends a generic log message (can be displayed as a sub-status).
func (tp *TUIProgress) Logf(format string, args ...any) {
	tp.program.Send(SubStageMsg(fmt.Sprintf(format, args...)))
}

// Warningf logs a warning.
func (tp *TUIProgress) Warningf(format string, args ...any) {
	tp.Logf("⚠️ Warning: "+format, args...)
}

// Debugf does nothing in the TUI.
func (tp *TUIProgress) Debugf(format string, args ...any) {}

// Run runs the TUI
func (tp *TUIProgress) Run() error {
	_, err := tp.program.Run()
	return err
}

// Context returns a progress.Context for the TUI
func (tp *TUIProgress) Context() *progress.Context {
	return progress.NewContext(tp)
}

