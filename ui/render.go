package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderPanel draws a btop-style panel: title embedded in the top border line,
// content padded to fill the inner area.
//
// w and h are the outer (total) dimensions including the border characters.
// accent is the full-brightness color (focused); accentDim is the darker shade (blurred).
func RenderPanel(title, content string, w, h int, focused bool, accent, accentDim lipgloss.Color) string {
	activeColor := accentDim
	if focused {
		activeColor = accent
	}

	borderSt := lipgloss.NewStyle().Foreground(activeColor)
	titleSt := lipgloss.NewStyle().Foreground(activeColor).Bold(focused)

	// ── Top border: ╭─ TITLE ─────────╮ ──────────────────────────────────────
	// Visible layout: ╭ + ─ + space + title + space + ─×n + ╮  = w
	//                 1   1    1      len(t)    1      n    1
	titleVisLen := lipgloss.Width(title)
	dashCount := w - titleVisLen - 5
	if dashCount < 0 {
		dashCount = 0
	}
	topBorder := borderSt.Render("╭─") +
		" " + titleSt.Render(title) + " " +
		borderSt.Render(strings.Repeat("─", dashCount)+"╮")

	// ── Bottom border ─────────────────────────────────────────────────────────
	bottomBorder := borderSt.Render("╰" + strings.Repeat("─", w-2) + "╯")

	// ── Content rows ──────────────────────────────────────────────────────────
	innerH := h - 2
	if innerH < 0 {
		innerH = 0
	}
	// contentW = w - │ - space - space - │ = w - 4
	contentW := w - 4
	if contentW < 0 {
		contentW = 0
	}

	lines := strings.Split(content, "\n")
	for len(lines) < innerH {
		lines = append(lines, "")
	}
	lines = lines[:innerH]

	lBorder := borderSt.Render("│")
	rBorder := borderSt.Render("│")

	rows := make([]string, 0, h)
	rows = append(rows, topBorder)
	for _, line := range lines {
		vis := lipgloss.Width(line)
		pad := contentW - vis
		if pad < 0 {
			pad = 0
		}
		rows = append(rows, lBorder+" "+line+strings.Repeat(" ", pad)+" "+rBorder)
	}
	rows = append(rows, bottomBorder)

	return strings.Join(rows, "\n")
}
