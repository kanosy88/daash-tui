package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderHelpOverlay draws a centered help panel over a dimmed full-screen background.
func RenderHelpOverlay(termW, termH int) string {
	const (
		boxW   = 54
		keyCol = 17
	)

	accent    := lipgloss.Color("#bb9af7") // lavender
	accentDim := lipgloss.Color("#4a3570")
	innerW    := boxW - 4 // 2 border chars + 2 padding spaces

	keyStyle  := lipgloss.NewStyle().Foreground(accent).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(ColorText)
	headStyle := lipgloss.NewStyle().Foreground(accent).Bold(true)
	dimStyle  := lipgloss.NewStyle().Foreground(ColorSubtle)
	sepStyle  := lipgloss.NewStyle().Foreground(accentDim)

	sep := sepStyle.Render(strings.Repeat("─", innerW))

	row := func(key, desc string) string {
		return keyStyle.Render(fmt.Sprintf("%-*s", keyCol, key)) + descStyle.Render(desc)
	}

	footer := lipgloss.NewStyle().
		Width(innerW).
		Align(lipgloss.Center).
		Foreground(ColorSubtle).
		Render("press ? or esc to close")

	lines := []string{
		headStyle.Render("NAVIGATION"),
		sep,
		row("tab / shift+tab", "next / previous panel"),
		row("1  2  3", "focus panel directly"),
		"",
		headStyle.Render("IN-PANEL"),
		sep,
		row("↑ k  /  ↓ j", "scroll"),
		row("v", "cycle view  (ticktick only)"),
		row("r", "refresh"),
		"",
		headStyle.Render("GENERAL"),
		sep,
		row("?", "toggle this help"),
		row("q  /  ctrl+c", "quit"),
		"",
		dimStyle.Render(footer),
	}

	content := strings.Join(lines, "\n")
	boxH := len(lines) + 2

	box := RenderPanel("HELP", content, boxW, boxH, true, accent, accentDim)

	return lipgloss.Place(termW, termH, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceBackground(lipgloss.Color("#16161e")))
}
