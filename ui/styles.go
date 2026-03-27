package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Base palette — Tokyo Night
	ColorText   = lipgloss.Color("#c0caf5")
	ColorSubtle = lipgloss.Color("#737aa2")

	// Per-panel accent colors — focused (full brightness)
	AccentCalendar = lipgloss.Color("#7aa2f7") // blue
	AccentTickTick   = lipgloss.Color("#9ece6a") // green
	AccentWeather  = lipgloss.Color("#e0af68") // amber
	AccentObsidian = lipgloss.Color("#bb9af7") // lavender (unused, kept for reference)

	// Per-panel accent colors — blurred (~40% brightness, same hue)
	AccentCalendarDim = lipgloss.Color("#3a5070")
	AccentTickTickDim   = lipgloss.Color("#405a28")
	AccentWeatherDim  = lipgloss.Color("#6b5028")
	AccentObsidianDim = lipgloss.Color("#4a3570")

	// Item styles
	ItemSelected   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7aa2f7")).Bold(true)
	ItemNormal     = lipgloss.NewStyle().Foreground(ColorText)
	ItemDimmed     = lipgloss.NewStyle().Foreground(ColorSubtle)
	TagStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("#9ece6a"))
	PriorityHigh   = lipgloss.NewStyle().Foreground(lipgloss.Color("#f7768e"))
	PriorityMedium = lipgloss.NewStyle().Foreground(lipgloss.Color("#e0af68"))
	PriorityLow    = lipgloss.NewStyle().Foreground(ColorSubtle)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle).
			Background(lipgloss.Color("#1a1b26")).
			PaddingLeft(1).PaddingRight(1)
)
