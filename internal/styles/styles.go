// Package styles provides terminal color and formatting utilities for foxi output.
// It includes functions for success, error, warning, info, and other styled text output.
package styles

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Color palette for foxi
var (
	// Primary colors
	Primary   = lipgloss.Color("#7D56F4") // Purple
	Secondary = lipgloss.Color("#04B575") // Green
	Accent    = lipgloss.Color("#F25D94") // Pink

	// Status colors
	SuccessColor = lipgloss.Color("#04B575") // Green
	WarningColor = lipgloss.Color("#FFB347") // Orange
	ErrorColor   = lipgloss.Color("#FF6B6B") // Red
	InfoColor    = lipgloss.Color("#54A6FF") // Blue

	// Text colors
	Text     = lipgloss.Color("#FAFAFA") // Light
	TextDim  = lipgloss.Color("#A8A8A8") // Dim
	TextDark = lipgloss.Color("#383838") // Dark

	// Background colors
	Background    = lipgloss.Color("#1A1A1A") // Dark background
	BackgroundAlt = lipgloss.Color("#2D2D2D") // Alternate background
)

// Base styles for common UI elements
var (
	// Headers and titles
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary).
			PaddingTop(1).
			PaddingBottom(1)

	SubHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(InfoColor)

	// Status indicators
	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(SuccessColor)

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ErrorColor)

	WarningStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(WarningColor)

	InfoStyle = lipgloss.NewStyle().
			Foreground(InfoColor)

	// Text styles
	BoldStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Text)

	DimStyle = lipgloss.NewStyle().
			Foreground(TextDim)

	CodeStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Background(BackgroundAlt).
			PaddingLeft(1).
			PaddingRight(1)

	// UI components
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(1, 2)

	ListItemStyle = lipgloss.NewStyle().
			PaddingLeft(2)
)

// Convenience functions for commonly used styled text
func Success(text string) string {
	return SuccessStyle.Render("✓ " + text)
}

func Error(text string) string {
	return ErrorStyle.Render("✗ " + text)
}

func Warning(text string) string {
	return WarningStyle.Render("⚠ " + text)
}

func Info(text string) string {
	return InfoStyle.Render("ℹ " + text)
}

func Header(text string) string {
	return HeaderStyle.Render("🦊 " + text)
}

func SubHeader(text string) string {
	return SubHeaderStyle.Render(text)
}

func Bold(text string) string {
	return BoldStyle.Render(text)
}

func Dim(text string) string {
	return DimStyle.Render(text)
}

func Code(text string) string {
	return CodeStyle.Render(text)
}

// DBF operation specific styles
func DBFSuccess(filepath, message string) string {
	return Success("Successfully read " + Bold(filepath) + ": " + message)
}

func DBFInfo(message string) string {
	return Info(message)
}

func DBFProgress(message string) string {
	return InfoStyle.Render("⏳ " + message)
}

// Action-specific styles
func ActionHeader(action string) string {
	return SubHeaderStyle.Render("📋 " + action)
}

func ActionSuccess(action string) string {
	return Success("Completed " + action)
}

func ActionProgress(action, message string) string {
	return InfoStyle.Render("🔄 [" + action + "] " + message)
}

// File operation styles
func FileOperation(op, file string) string {
	return DimStyle.Render("  " + op + ": " + file)
}

func FileCount(count int, operation string) string {
	return InfoStyle.Render(fmt.Sprintf("📁 %d files %s", count, operation))
}

// Progress and status styles
func ProgressBar(current, total int, message string) string {
	percentage := float64(current) / float64(total) * 100
	bar := progressBar(int(percentage))
	return InfoStyle.Render(fmt.Sprintf("%s %.1f%% %s", bar, percentage, message))
}

func progressBar(percentage int) string {
	const barWidth = 20
	filled := percentage * barWidth / 100
	bar := ""
	for i := 0; i < barWidth; i++ {
		if i < filled {
			bar += "█"
		} else {
			bar += "░"
		}
	}
	return "[" + bar + "]"
}

// Error formatting for detailed error messages
func ErrorDetails(err error, context map[string]string) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.ThickBorder()).
		BorderForeground(ErrorColor).
		Padding(1).
		Margin(1)

	content := ErrorStyle.Render("Error: ") + err.Error() + "\n"

	if len(context) > 0 {
		content += "\n" + DimStyle.Render("Context:") + "\n"
		for key, value := range context {
			content += DimStyle.Render("  "+key+": ") + value + "\n"
		}
	}

	return style.Render(content)
}

// Help and documentation styles
func HelpSection(title, content string) string {
	return HeaderStyle.Render(title) + "\n" + content + "\n"
}

func Example(command, description string) string {
	return "  " + Code(command) + " - " + Dim(description)
}

// Prompt styling functions
func PromptTitle(title string) string {
	return BoldStyle.Render(title)
}

func PromptHint(hint string) string {
	return DimStyle.Render(hint)
}
