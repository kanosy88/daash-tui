package model

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"daash/panels"
	"daash/ui"
)

const (
	statusBarHeight = 1
	minWidth        = 80
	minHeight       = 20
)

// AppModel is the root Bubble Tea model. It owns all panels and orchestrates
// layout, focus routing, and the composed view.
type AppModel struct {
	panels       []panels.Panel
	focusedIndex int
	width        int
	height       int
	ready        bool
}

// New creates an AppModel with the given panels. The first panel starts focused.
func New(ps ...panels.Panel) AppModel {
	m := AppModel{panels: make([]panels.Panel, len(ps))}
	copy(m.panels, ps)
	// Focus the first panel
	if len(m.panels) > 0 {
		m.panels[0] = m.panels[0].Focus()
	}
	return m
}

func (m AppModel) Init() tea.Cmd {
	cmds := make([]tea.Cmd, len(m.panels))
	for i, p := range m.panels {
		cmds[i] = p.Init()
	}
	return tea.Batch(cmds...)
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m = m.recalculateLayout()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case ui.KeyQuit, ui.KeyQuitAlt:
			return m, tea.Quit
		case ui.KeyNextPanel:
			m = m.focusPanel((m.focusedIndex + 1) % len(m.panels))
			return m, nil
		case ui.KeyPrevPanel:
			prev := (m.focusedIndex - 1 + len(m.panels)) % len(m.panels)
			m = m.focusPanel(prev)
			return m, nil
		case "1", "2", "3":
			idx := int(msg.Runes[0] - '1')
			if idx >= 0 && idx < len(m.panels) {
				m = m.focusPanel(idx)
			}
			return m, nil
		}
	}

	// Key messages go only to the focused panel.
	// All other messages (data fetches, ticks, etc.) are broadcast to every panel.
	if _, isKey := msg.(tea.KeyMsg); isKey {
		var cmd tea.Cmd
		updated, cmd := m.panels[m.focusedIndex].Update(msg)
		m.panels[m.focusedIndex] = updated.(panels.Panel)
		return m, cmd
	}

	var cmds []tea.Cmd
	for i, p := range m.panels {
		updated, cmd := p.Update(msg)
		m.panels[i] = updated.(panels.Panel)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return m, tea.Batch(cmds...)
}

func (m AppModel) View() string {
	if !m.ready {
		return "Initializing…"
	}
	if m.width < minWidth || m.height < minHeight {
		return fmt.Sprintf("Terminal too small (min %dx%d, got %dx%d)",
			minWidth, minHeight, m.width, m.height)
	}

	// Layout: Calendar (left, full height) | TicTic + Weather (right, stacked)
	rightCol := lipgloss.JoinVertical(lipgloss.Left,
		m.panels[1].View(),
		m.panels[2].View(),
	)
	grid := lipgloss.JoinHorizontal(lipgloss.Top, m.panels[0].View(), rightCol)

	statusBar := ui.StatusBarStyle.Width(m.width).Render(m.statusBarContent())

	return lipgloss.JoinVertical(lipgloss.Left, grid, statusBar)
}

// recalculateLayout distributes terminal space across the 3 panels.
// Layout: Calendar (left, full height) | TicTic (right top) + Weather (right bottom)
func (m AppModel) recalculateLayout() AppModel {
	availH := m.height - statusBarHeight
	panelW := m.width / 2

	ticticH := availH / 2
	weatherH := availH - ticticH // absorbs odd-height remainder

	m.panels[0] = m.panels[0].SetSize(panelW, availH)  // Calendar: full height
	m.panels[1] = m.panels[1].SetSize(panelW, ticticH)  // TicTic: top half
	m.panels[2] = m.panels[2].SetSize(panelW, weatherH) // Weather: bottom half
	return m
}

// focusPanel blurs all panels and focuses the one at index i.
func (m AppModel) focusPanel(i int) AppModel {
	m.focusedIndex = i
	for j, p := range m.panels {
		if j == i {
			m.panels[j] = p.Focus()
		} else {
			m.panels[j] = p.Blur()
		}
	}
	return m
}

func (m AppModel) statusBarContent() string {
	now := time.Now()

	// Left: current time
	left := now.Format("15:04")

	// Center: focused panel + shortcuts
	center := fmt.Sprintf("[ %s ]  tab · ↑↓ · v · q", m.panels[m.focusedIndex].Title())

	// Right: last sync times for each panel
	labels := []string{"cal", "tick", "wx"}
	right := ""
	for i, p := range m.panels {
		right += labels[i] + ": " + formatSyncAge(p.LastSync(), now)
		if i < len(m.panels)-1 {
			right += "  "
		}
	}

	// Pad center between left and right
	totalSideW := lipgloss.Width(left) + lipgloss.Width(right)
	centerW := m.width - totalSideW - 4 // 4 for padding
	if centerW < 0 {
		centerW = 0
	}
	centerPadded := fmt.Sprintf("%-*s", centerW, center)

	return left + "  " + centerPadded + right
}

func formatSyncAge(last time.Time, now time.Time) string {
	if last.IsZero() {
		return "—"
	}
	age := now.Sub(last)
	switch {
	case age < time.Minute:
		return "just now"
	case age < time.Hour:
		return fmt.Sprintf("%dm ago", int(age.Minutes()))
	default:
		return last.Format("15:04")
	}
}
