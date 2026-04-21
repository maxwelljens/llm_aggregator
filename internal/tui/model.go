package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
			Foreground(lipgloss.Color("#FFF7DB")).
			SetString("LLM Aggregator")

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
		"Connecting to DeepSeek",
		"Getting summary",
		"Formatting output",
		"Complete",
	}
)

type progressMsg float64

type Model struct {
	progress       progress.Model
	currentStep    int
	totalSteps     int
	status         string
	subStatus      string
	startTime      time.Time
	width          int
	height         int
	done           bool
	errorMsg       string
	articlesCount  int
	processedCount int
}

func New() *Model {
	prog := progress.New(
		progress.WithScaledGradient("#FF7CCB", "#FDFF8C"),
		progress.WithWidth(40),
		progress.WithoutPercentage(),
	)

	return &Model{
		progress:    prog,
		currentStep: 0,
		totalSteps:  len(stepNames) - 1, // Last step is "Complete"
		status:      "Initialising...",
		subStatus:   "Loading configuration",
		startTime:   time.Now(),
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
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
		m.progress.Width = msg.Width - padding*2 - 4
		if m.progress.Width > maxWidth {
			m.progress.Width = maxWidth
		}
	case progressMsg:
		progress := float64(msg)
		if progress >= 1.0 {
			m.done = true
			return m, tea.Quit
		}
		cmd := m.progress.SetPercent(float64(progress))
		return m, cmd
	case StepMsg:
		if msg.Step >= 0 && msg.Step < len(stepNames) {
			m.currentStep = msg.Step
			m.status = stepNames[msg.Step]
			if msg.Status != "" {
				m.subStatus = msg.Status
			}
			if msg.ArticlesCount > 0 {
				m.articlesCount = msg.ArticlesCount
			}
			if msg.ProcessedCount > 0 {
				m.processedCount = msg.ProcessedCount
			}
		}
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
	sb.WriteString(titleStyle.Render("🤖 LLM RSS Aggregator"))
	sb.WriteString("\n\n")

	// Progress bar
	progressPercent := float64(m.currentStep) / float64(m.totalSteps)
	progressView := m.progress.ViewAs(progressPercent)
	sb.WriteString(progressBarStyle.Render(progressView))
	sb.WriteString("\n\n")

	// Status
	sb.WriteString(statusStyle.Render("Status: "))
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
	elapsed := time.Since(m.startTime).Round(time.Second)
	sb.WriteString(infoStyle.Render(fmt.Sprintf("Elapsed: %v", elapsed)))
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

	// Help text
	if !m.done && m.errorMsg == "" {
		sb.WriteString(infoStyle.Render("Press 'q' or Ctrl+C to quit"))
	} else if m.done {
		completeStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#43BF6D")).
			Bold(true)
		sb.WriteString(completeStyle.Render("✓ Processing complete!"))
		sb.WriteString("\n")
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