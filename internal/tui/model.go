package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"llm_aggregator/internal/runtime"
)

const (
	padding  = 2
	maxWidth = 80
)

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	titleStyle = lipgloss.NewStyle().
			MarginLeft(2).
			MarginRight(2).
			Padding(0, 1).
			Italic(true).
			Foreground(lipgloss.Color("#FFF7DB"))

	infoStyle = lipgloss.NewStyle().
			Foreground(subtle).
			Italic(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(highlight).
			Bold(true)

	progressBarStyle = lipgloss.NewStyle().
				Foreground(special)

	stepNames = []string{
		"Initialising",
		"Aggregating feeds",
		"Processing articles",
		"Connecting to LLM",
		"Getting summary",
		"Formatting output",
		"Complete",
	}
)

type StageMsg string
type SubStageMsg string
type ArticleCountMsg struct {
	Total, Processed int
}

type processingDoneMsg struct {
	err error
}

type progressMsg float64

type Model struct {
	progress       progress.Model
	spinner        spinner.Model
	runtime        *runtime.Runtime
	currentStep    int
	totalSteps     int
	status         string
	subStatus      string
	startTime      time.Time
	elapsed        time.Duration
	width          int
	height         int
	done           bool
	errorMsg       string
	articlesCount  int
	processedCount int
}

func New(rt *runtime.Runtime) *Model {
	prog := progress.New(
		progress.WithScaledGradient("#FF7CCB", "#FDFF8C"),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF7CCB"))

	return &Model{
		progress:    prog,
		spinner:     sp,
		runtime:     rt,
		currentStep: 0,
		totalSteps:  len(stepNames) - 1, // Last step is "Complete"
		status:      "Initialising...",
		subStatus:   "Loading configuration",
		startTime:   time.Now(),
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startProcessing)
}

// MODIFIED: This is now the standard bubbletea command pattern.
// The bubbletea runtime will execute this function in a goroutine.
// We no longer need to manage the goroutine or use program.Send().
func (m *Model) startProcessing() tea.Msg {
	err := m.runtime.Execute(context.Background())
	return processingDoneMsg{err: err}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = min(msg.Width-padding*2-4, maxWidth)
	case progressMsg:
		// This message type seems unused, but leaving it in case.
		progress := float64(msg)
		if progress >= 1.0 {
			m.done = true
			return m, nil
		}
		cmd := m.progress.SetPercent(float64(progress))
		return m, cmd
	case spinner.TickMsg:
		// MODIFIED: Stop updating the spinner once we're done.
		if m.done {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case StageMsg:
		stageName := string(msg)
		for i, name := range stepNames {
			if name == stageName {
				m.currentStep = i
				m.status = name
				m.subStatus = "" // Clear sub-status when stage changes
				break
			}
		}

	case SubStageMsg:
		m.subStatus = string(msg)

	case ArticleCountMsg:
		m.articlesCount = msg.Total
		m.processedCount = msg.Processed

	case processingDoneMsg:
		if msg.err != nil {
			m.errorMsg = msg.err.Error()
			m.done = true // Also mark as done on error
		} else {
			m.done = true
			m.currentStep = m.totalSteps // Go to "Complete" step
			m.status = stepNames[m.totalSteps]
			m.subStatus = "Summary generated successfully!"
		}
		// MODIFIED: Capture the final elapsed time.
		m.elapsed = time.Since(m.startTime)
		// We are done, so we return a command to quit the program automatically.
		// The user can see the final summary. If we want the user to quit manually,
		// just return 'm, nil'.
		return m, nil
	case ErrorMsg:
		m.errorMsg = msg.Error
		m.status = "Error"
		return m, tea.Quit
	}

	return m, nil
}

func (m *Model) View() string {
	if m.width == 0 {
		return "Initialising..."
	}

	var sb strings.Builder

	// Title
	sb.WriteString(titleStyle.Render("🤖 LLM Aggregator"))
	sb.WriteString("\n\n")

	// MODIFIED: If done, show 100% progress.
	var progressPercent float64
	if m.done && m.errorMsg == "" {
		progressPercent = 1.0
	} else if m.articlesCount > 0 && m.currentStep >= 4 {
		progressPercent = float64(m.processedCount) / float64(m.articlesCount)
	} else {
		progressPercent = float64(m.currentStep) / float64(m.totalSteps)
	}

	progressView := m.progress.ViewAs(progressPercent)
	sb.WriteString(progressBarStyle.Render(progressView))
	sb.WriteString("\n\n")

	// Status with spinner
	sb.WriteString(statusStyle.Render("Status: "))

	// MODIFIED: Replace spinner with a checkmark when done.
	if m.done && m.errorMsg == "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(special).Render("✓"))
	} else if m.done && m.errorMsg != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render("✗"))
	} else {
		sb.WriteString(m.spinner.View())
	}

	sb.WriteString(" ")
	sb.WriteString(m.status)
	sb.WriteString("\n")

	// Sub-status
	if m.subStatus != "" {
		sb.WriteString(infoStyle.Render("  → "))
		sb.WriteString(m.subStatus)
		sb.WriteString("\n")
	}

	// Statistics
	if m.articlesCount > 0 {
		sb.WriteString(infoStyle.Render(fmt.Sprintf("Articles: %d", m.articlesCount)))
		sb.WriteString("  ")
	}
	if m.processedCount > 0 {
		sb.WriteString(infoStyle.Render(fmt.Sprintf("Processed: %d", m.processedCount)))
		sb.WriteString("\n")
	}

	// Elapsed time
	// MODIFIED: Use the stored elapsed time if done, otherwise calculate it live.
	elapsed := m.elapsed
	if !m.done {
		elapsed = time.Since(m.startTime)
	}
	sb.WriteString(infoStyle.Render(fmt.Sprintf("Elapsed: %v", elapsed.Round(time.Second))))
	sb.WriteString("\n\n")

	// Error message (if any)
	if m.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)
		sb.WriteString(errorStyle.Render("Error: "))
		sb.WriteString(m.errorMsg)
		sb.WriteString("\n\n")
	}

	// MODIFIED: Simplified help text logic
	if m.done {
		// The summary is not part of the TUI, it's printed to stdout *after* the TUI exits.
		// The TUI's job is just to show the progress.
		// The `tea.Quit` in the `processingDoneMsg` case will cause the program to exit
		// and the rest of the `main` function to proceed.
	} else {
		sb.WriteString(infoStyle.Render("Press 'q' or Ctrl+C to quit"))
	}

	// If the runtime has a summary and we are done, we can show it here.
	// However, the typical flow is for the TUI to finish, and then the main
	// function prints the final output. If we want to show it *in* the TUI,
	// we would add this block:
	if m.done && m.runtime.Summary != "" {
		summaryTitleStyle := lipgloss.NewStyle().Bold(true).Underline(true)
		sb.WriteString(summaryTitleStyle.Render("Summary"))
		sb.WriteString("\n")

		// Calculate a safe width for wrapping (terminal width minus some margin)
		wrapWidth := max(m.width-4,
			// Sane minimum fallback
			20)

		// Create a style that enforces word-wrapping
		summaryStyle := lipgloss.NewStyle().
			Width(wrapWidth)

		// Render the summary through the style to apply the wrapping
		sb.WriteString(summaryStyle.Render(m.runtime.Summary))
		sb.WriteString("\n\n")

		sb.WriteString(infoStyle.Render("Press 'q' to exit."))
	} else if m.done {
		sb.WriteString(infoStyle.Render("Press 'q' to exit."))
	}

	return sb.String()
}

// Messages for updating the TUI
type StepMsg struct {
	Step           int
	Status         string
	ArticlesCount  int
	ProcessedCount int
}

type ErrorMsg struct {
	Error string
}

func Step(step int, status string) tea.Cmd {
	return func() tea.Msg {
		return StepMsg{Step: step, Status: status}
	}
}

func StepWithCounts(step int, status string, articles, processed int) tea.Cmd {
	return func() tea.Msg {
		return StepMsg{
			Step:           step,
			Status:         status,
			ArticlesCount:  articles,
			ProcessedCount: processed,
		}
	}
}

func Error(err string) tea.Cmd {
	return func() tea.Msg {
		return ErrorMsg{Error: err}
	}
}
