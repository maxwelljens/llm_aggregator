// Package style provides consistent text styling using lipgloss.
// It respects the NO_COLOR environment variable (https://no-color.org/).
package style

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Italic returns an italic style.
func Italic(text string) string {
	return lipgloss.NewStyle().Italic(true).Render(text)
}

// Colour represents a lipgloss colour that can be applied.
type Colour lipgloss.Color

// ANSI colour codes (standard 8-colour palette).
// Using ANSI escape codes for portability across terminals.
var (
	// Red for errors and warnings
	Red = Colour("1")

	// Green for success messages
	Green = Colour("2")

	// Blue for URLs
	Blue = Colour("12")

	// Magenta for labels and info
	Magenta = Colour("5")

	// Cyan for filepaths and values
	Cyan = Colour("6")

	// White for headings
	White = Colour("15")

	// Bright black for muted/debug text
	BrightBlack = Colour("8")
)

// NoColor returns true if the NO_COLOR environment variable is set.
// This respects https://no-color.org/ standard.
func NoColor() bool {
	return os.Getenv("NO_COLOR") != ""
}

// Warning returns an orange and bold warning message.
// Note: ANSI doesn't have orange, so we use red with bold styling.
func Warning(text string) string {
	if NoColor() {
		return "WARNING: " + text
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Red)).
		Bold(true).
		Render("WARNING: " + text)
}

// Warningf returns a formatted warning message.
func Warningf(format string, args ...any) string {
	return Warning(fmt.Sprintf(format, args...))
}

// Error returns a red and bold error message.
func Error(text string) string {
	if NoColor() {
		return "ERROR: " + text
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Red)).
		Bold(true).
		Render("ERROR: " + text)
}

// Errorf returns a formatted error message.
func Errorf(format string, args ...any) string {
	return Error(fmt.Sprintf(format, args...))
}

// Success returns a green success message.
func Success(text string) string {
	if NoColor() {
		return "✓ " + text
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Green)).
		Bold(true).
		Render("✓ " + text)
}

// Filepath returns a styled filepath in cyan.
func Filepath(filepath string) string {
	if NoColor() {
		return filepath
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(Cyan)).Render(filepath)
}

// FilepathStyled returns a lipgloss style for filepaths that can be applied to custom text.
var FilepathStyled = lipgloss.NewStyle().Foreground(lipgloss.Color(Cyan))

// URLStyled returns a lipgloss style for URLs that can be applied to custom text.
var URLStyled = lipgloss.NewStyle().Foreground(lipgloss.Color(Blue)).Underline(true)

// Label returns a styled label/key in magenta and bold.
func Label(text string) string {
	if NoColor() {
		return lipgloss.NewStyle().Bold(true).Render(text)
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Magenta)).
		Bold(true).
		Render(text)
}

// LabelStyled returns a lipgloss style for labels.
var LabelStyled = lipgloss.NewStyle().Foreground(lipgloss.Color(Magenta)).Bold(true)

// Heading returns a styled heading.
func Heading(text string) string {
	return lipgloss.NewStyle().Bold(true).Render(text)
}

// Value returns styled value text.
func Value(text string) string {
	if NoColor() {
		return text
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(Cyan)).Render(text)
}

// ValueStyled returns a lipgloss style for values.
var ValueStyled = lipgloss.NewStyle().Foreground(lipgloss.Color(Cyan))

// Info returns a styled info message in cyan.
func Info(text string) string {
	if NoColor() {
		return "ℹ " + text
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(Cyan)).Render("ℹ " + text)
}

// Debug returns a styled debug message in grey.
func Debug(text string) string {
	if NoColor() {
		return lipgloss.NewStyle().Italic(true).Render("Debug: " + text)
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(BrightBlack)).
		Italic(true).
		Render("Debug: " + text)
}

// Debugf returns a formatted debug message.
func Debugf(format string, args ...any) string {
	return Debug(fmt.Sprintf(format, args...))
}
