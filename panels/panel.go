package panels

import (
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

// Panel is the contract every dashboard panel must satisfy.
// It extends tea.Model so panels can be composed directly.
type Panel interface {
	tea.Model

	// Title returns the panel's display title.
	Title() string

	// SetSize informs the panel of its allocated outer dimensions (border included).
	// Called by AppModel on WindowSizeMsg and initial layout.
	SetSize(width, height int) Panel

	// IsFocused returns true when this panel has keyboard focus.
	IsFocused() bool

	// Focus grants keyboard focus to this panel.
	Focus() Panel

	// Blur removes keyboard focus from this panel.
	Blur() Panel

	// LastSync returns the time of the last successful data fetch.
	// Returns zero time if no fetch has completed yet.
	LastSync() time.Time
}
