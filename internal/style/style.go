// Package style provides consistent text styling using lipgloss.
// It respects the NO_COLOR environment variable (https://no-color.org/).
package style

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Bold returns a bold style.
func Bold(text string) string {
	return lipgloss.NewStyle().Bold(true).Render(text)
}

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

	// Yellow for emphasis
	Yellow = Colour("3")

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
		return lipgloss.NewStyle().Bold(true).Render("WARNING: " + text)
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
		return lipgloss.NewStyle().Bold(true).Render("ERROR: " + text)
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
		return lipgloss.NewStyle().Bold(true).Render("✓ " + text)
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Green)).
		Bold(true).
		Render("✓ " + text)
}

// Successf returns a formatted success message.
func Successf(format string, args ...any) string {
	return Success(fmt.Sprintf(format, args...))
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

// URL returns a styled URL in blue with underline.
func URL(url string) string {
	if NoColor() {
		return lipgloss.NewStyle().Underline(true).Render(url)
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Blue)).
		Underline(true).
		Render(url)
}

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

// Section returns a styled section header.
func Section(text string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		Render(text)
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

// Infof returns a formatted info message.
func Infof(format string, args ...any) string {
	return Info(fmt.Sprintf(format, args...))
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

// Highlight returns a highlighted text (yellow bold).
func Highlight(text string) string {
	if NoColor() {
		return lipgloss.NewStyle().Bold(true).Render(text)
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(Yellow)).
		Bold(true).
		Render(text)
}

// Highlightf returns a formatted highlighted message.
func Highlightf(format string, args ...any) string {
	return Highlight(fmt.Sprintf(format, args...))
}

// ParseAndStyleURLs takes a string and styles any URLs found within it.
// This is useful for processing text that may contain URLs mixed with other content.
func ParseAndStyleURLs(text string) string {
	// Simple URL pattern matching - matches http:// and https:// URLs
	// This is a basic implementation; for production use, consider a regex library
	parts := strings.Split(text, " ")

	for i, part := range parts {
		if strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://") {
			parts[i] = URL(part)
		}
	}

	return strings.Join(parts, " ")
}

// ParseAndStyleFilepaths takes a string and styles any filepaths found within it.
// This looks for paths that start with / or are relative paths containing /.
func ParseAndStyleFilepaths(text string) string {
	// Simple detection for filepaths - words containing / but not just URLs
	parts := strings.Split(text, " ")

	for i, part := range parts {
		// Skip if it's already a URL (starts with http)
		if strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://") {
			continue
		}

		// Check if it looks like a filepath
		if strings.Contains(part, "/") && !strings.Contains(part, "://") {
			parts[i] = Filepath(part)
		}
	}

	return strings.Join(parts, " ")
}

