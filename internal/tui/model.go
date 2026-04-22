package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"llm_aggregator/internal/runtime"
)

const (
	padding  = 2
	maxWidth = 80
)

var (
	colorSubtle        = lipgloss.Color("7")   // Gray (dim text)
	colorHighlight     = lipgloss.Color("13")  // Magenta (status)
	colorSuccess       = lipgloss.Color("2")   // Green (completion)
	colorError         = lipgloss.Color("1")   // Red (errors)
	colorGradientStart = lipgloss.Color("205") // Pink
	colorGradientEnd   = lipgloss.Color("226") // Yellow

	titleStyle = lipgloss.NewStyle().
			MarginLeft(2).
			MarginRight(2).
			Padding(0, 1).
			Italic(true).
			Foreground(lipgloss.Color("15"))

	infoStyle = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Italic(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(colorHighlight).
			Bold(true)

	progressBarStyle = lipgloss.NewStyle().
				Foreground(colorSuccess)

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
	viewport       viewport.Model
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
	showSummary    bool
}

func New(rt *runtime.Runtime) *Model {
	prog := progress.New(
		progress.WithSolidFill("colorSuccess"),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colorHighlight)

	return &Model{
		progress:    prog,
		spinner:     sp,
		runtime:     rt,
		currentStep: 0,
		totalSteps:  len(stepNames) - 1, // Last step is "Complete"
		status:      "Initialising...",
		subStatus:   "Loading configuration",
		startTime:   time.Now(),
		showSummary: false,
	}
}

func (m *Model) Init() tea.Cmd {
	// Initialise the viewport with default size (will be resized on WindowSizeMsg)
	m.viewport = viewport.New(80, 20)
	m.viewport.MouseWheelEnabled = true

	return tea.Batch(m.spinner.Tick, m.startProcessing)
}

// This is now the standard bubbletea command pattern.
// The bubbletea runtime will execute this function in a goroutine.
// We no longer need to manage the goroutine or use program.Send().
func (m *Model) startProcessing() tea.Msg {
	err := m.runtime.Execute(context.Background())
	return processingDoneMsg{err: err}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Let the viewport handle navigation keys if showing summary
		if m.showSummary {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			if cmd != nil {
				return m, cmd
			}
			// If viewport handled a key, we're done
			switch msg.String() {
			case "j", "down", "k", "up", "g", "G", " ", "b":
				return m, nil
			}
		}
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.progress.Width = min(msg.Width-padding*2-4, maxWidth)
		// Resize viewport for summary display
		if m.showSummary {
			m.viewport.Width = msg.Width - padding*2
			m.viewport.Height = msg.Height - 10 // Leave space for header/footer
		}
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
			// Enable summary viewport if there's content to display
			if m.runtime.Summary != "" {
				m.showSummary = true
				// Resize viewport to fit available space
				m.viewport.Width = m.width - padding*2
				m.viewport.Height = m.height - 15 // Leave space for header/footer
				// Wrap content to viewport width before setting
				contentWidth := m.viewport.Width
				summaryContent := "📝 Summary\n\n" + wrapText(m.runtime.Summary, contentWidth)
				m.viewport.SetContent(summaryContent)
			}
		}
		// Capture the final elapsed time.
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

	// If we're showing the summary with scrolling, use a different layout
	if m.showSummary {
		return m.viewSummary()
	}

	var sb strings.Builder

	// Title
	sb.WriteString(titleStyle.Render("🤖 LLM Aggregator"))
	sb.WriteString("\n\n")

	// If done, show 100% progress.
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

	// Replace spinner with a checkmark when done.
	if m.done && m.errorMsg == "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorSuccess).Render("✓"))
	} else if m.done && m.errorMsg != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorError).Render("✗"))
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
			Foreground(colorError).
			Bold(true)
		sb.WriteString(errorStyle.Render("Error: "))
		sb.WriteString(m.errorMsg)
		sb.WriteString("\n\n")
	}

	// Simplified help text logic
	if m.done {
		// The summary is not part of the TUI, it's printed to stdout *after* the TUI exits.
		// The TUI's job is just to show the progress.
		// The `tea.Quit` in the `processingDoneMsg` case will cause the program to exit
		// and the rest of the `main` function to proceed.
	} else {
		sb.WriteString(infoStyle.Render("Press 'q' or Ctrl+C to quit"))
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

// viewSummary renders the summary view with scrolling support.
func (m *Model) viewSummary() string {
	var sb strings.Builder

	// Header
	sb.WriteString(titleStyle.Render("🤖 LLM Aggregator"))
	sb.WriteString("\n\n")

	// Show completion status
	sb.WriteString(statusStyle.Render("✓ Complete"))
	sb.WriteString(" ")
	sb.WriteString(m.subStatus)
	sb.WriteString("\n\n")

	// Elapsed time and article stats
	sb.WriteString(infoStyle.Render(fmt.Sprintf("Articles: %d | Processed: %d | Elapsed: %v",
		m.articlesCount, m.processedCount, m.elapsed.Round(time.Second))))
	sb.WriteString("\n\n")

	// Scroll progress indicator
	scrollPercent := int(m.viewport.ScrollPercent() * 100)
	if m.viewport.TotalLineCount() > m.viewport.VisibleLineCount() {
		sb.WriteString(infoStyle.Render(fmt.Sprintf("Scroll: %d%% (%d/%d lines)",
			scrollPercent, m.viewport.TotalLineCount()-m.viewport.VisibleLineCount()-m.viewport.YOffset,
			m.viewport.TotalLineCount())))
		sb.WriteString("\n\n")
	}

	// Viewport (scrollable summary)
	sb.WriteString(m.viewport.View())
	sb.WriteString("\n")

	// Navigation help
	sb.WriteString(infoStyle.Render("Navigate: ↑/↓ or j/k (scroll) | Space/B (page) | g/G (start/end) | q (quit)"))

	return sb.String()
}

// wrapText wraps text to a given width using lipgloss.
func wrapText(text string, width int) string {
	style := lipgloss.NewStyle().Width(width)
	return style.Render(text)
}
