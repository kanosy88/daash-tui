package calendar

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kanosy88/daash-tui/panels"
	"github.com/kanosy88/daash-tui/ui"
)

// Event represents a calendar event.
type Event struct {
	Title        string
	Time         time.Time
	Duration     time.Duration
	Location     string
	CalendarName string // empty when only one calendar is configured
}

// calendarFetchedMsg is sent when the API call completes.
type calendarFetchedMsg struct {
	events []Event
	err    error
}

// minuteTickMsg fires every minute — triggers a local re-render (no API call).
type minuteTickMsg time.Time

// periodicRefreshMsg fires every 5 minutes as a safety-net API fetch.
type periodicRefreshMsg struct{}

// CalendarModel implements panels.Panel.
type CalendarModel struct {
	events   []Event
	cursor   int
	loading  bool
	fetchErr string
	lastSync time.Time
	width    int
	height   int
	focused  bool
}

func New() *CalendarModel {
	return &CalendarModel{loading: true}
}

func (m *CalendarModel) Title() string { return "Calendar" }

func (m *CalendarModel) SetSize(width, height int) panels.Panel {
	m.width = width
	m.height = height
	return m
}

func (m *CalendarModel) IsFocused() bool      { return m.focused }
func (m *CalendarModel) LastSync() time.Time  { return m.lastSync }

func (m *CalendarModel) Focus() panels.Panel {
	m.focused = true
	return m
}

func (m *CalendarModel) Blur() panels.Panel {
	m.focused = false
	return m
}

// Init starts the first API fetch plus the two background tickers.
func (m *CalendarModel) Init() tea.Cmd {
	return tea.Batch(fetchCmd(), minuteTickCmd(), periodicRefreshCmd())
}

func (m *CalendarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case calendarFetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.fetchErr = msg.err.Error()
			return m, nil
		}
		m.fetchErr = ""
		m.events = msg.events
		m.cursor = 0
		m.lastSync = time.Now()
		return m, nil

	case minuteTickMsg:
		// Re-arm the minute ticker and trigger a re-render (no API call).
		return m, minuteTickCmd()

	case periodicRefreshMsg:
		// Re-arm the periodic ticker and fetch fresh data.
		m.loading = true
		return m, tea.Batch(fetchCmd(), periodicRefreshCmd())

	case tea.KeyMsg:
		switch msg.String() {
		case ui.KeyScrollUp, ui.KeyScrollUpK:
			if m.cursor > 0 {
				m.cursor--
			}
		case ui.KeyScrollDown, ui.KeyScrollDownJ:
			if m.cursor < len(m.events)-1 {
				m.cursor++
			}
		case "r":
			m.loading = true
			m.fetchErr = ""
			return m, fetchCmd()
		}
	}
	return m, nil
}

func (m *CalendarModel) View() string {
	contentW := m.width - 4
	if contentW < 1 {
		contentW = 1
	}

	var lines []string

	switch {
	case m.loading:
		lines = append(lines, "")
		lines = append(lines, "  "+ui.ItemDimmed.Render("Fetching calendar…"))

	case m.fetchErr != "":
		lines = append(lines, "")
		lines = append(lines, "  "+ui.PriorityHigh.Render("Error:"))
		lines = append(lines, "  "+ui.ItemDimmed.Render(truncate(m.fetchErr, contentW-2)))
		lines = append(lines, "")
		lines = append(lines, "  "+ui.ItemDimmed.Render("Press r to retry"))

	case len(m.events) == 0:
		lines = append(lines, "")
		lines = append(lines, "  "+ui.ItemDimmed.Render("No upcoming events."))

	default:
		now := time.Now()

		type indexedEvent struct {
			ev  Event
			idx int
		}
		var ongoing, upcoming []indexedEvent
		for i, ev := range m.events {
			if isOngoing(ev, now) {
				ongoing = append(ongoing, indexedEvent{ev, i})
			} else {
				upcoming = append(upcoming, indexedEvent{ev, i})
			}
		}

		renderEvent := func(ie indexedEvent, timeLabel string) {
			itemStyle := ui.ItemNormal
			prefix := "  "
			if ie.idx == m.cursor && m.focused {
				itemStyle = ui.ItemSelected
				prefix = "▸ "
			}
			lines = append(lines, prefix+itemStyle.Render(truncate(ie.ev.Title, contentW-14))+"  "+ui.ItemDimmed.Render(timeLabel))
			if ie.ev.Location != "" {
				lines = append(lines, "    "+ui.ItemDimmed.Render("@ "+truncate(ie.ev.Location, contentW-6)))
			}
			if ie.ev.CalendarName != "" {
				lines = append(lines, "    "+ui.ItemDimmed.Render("["+ie.ev.CalendarName+"]"))
			}
		}

		if len(ongoing) > 0 {
			lines = append(lines, sectionSep("En cours", contentW))
			nowStyle := lipgloss.NewStyle().Foreground(ui.AccentCalendar)
			for _, ie := range ongoing {
				itemStyle := ui.ItemNormal
				prefix := "  "
				if ie.idx == m.cursor && m.focused {
					itemStyle = ui.ItemSelected
					prefix = "▸ "
				}
				lines = append(lines, prefix+itemStyle.Render(truncate(ie.ev.Title, contentW-14))+"  "+nowStyle.Render(formatTimeLeft(ie.ev)))
				if ie.ev.Location != "" {
					lines = append(lines, "    "+ui.ItemDimmed.Render("@ "+truncate(ie.ev.Location, contentW-6)))
				}
				if ie.ev.CalendarName != "" {
					lines = append(lines, "    "+ui.ItemDimmed.Render("["+ie.ev.CalendarName+"]"))
				}
			}
		}

		if len(upcoming) > 0 {
			if len(ongoing) > 0 {
				lines = append(lines, sectionSep("À venir", contentW))
			}
			for _, ie := range upcoming {
				renderEvent(ie, formatRelativeTime(ie.ev.Time))
			}
		}
	}

	return ui.RenderPanel("󰃭 "+m.Title(), strings.Join(lines, "\n"), m.width, m.height, m.focused, ui.AccentCalendar, ui.AccentCalendarDim)
}

// isOngoing returns true if the event has started but not yet ended.
func isOngoing(ev Event, now time.Time) bool {
	if ev.Duration == 0 {
		// All-day event: ongoing if start date is today.
		sy, sm, sd := ev.Time.Date()
		ny, nm, nd := now.Date()
		return sy == ny && sm == nm && sd == nd
	}
	return !ev.Time.After(now) && now.Before(ev.Time.Add(ev.Duration))
}

// formatTimeLeft returns a human-readable string for time remaining in an event.
func formatTimeLeft(ev Event) string {
	remaining := time.Until(ev.Time.Add(ev.Duration))
	if remaining < time.Minute {
		return "ending now"
	}
	if remaining < time.Hour {
		return fmt.Sprintf("%dmin left", int(remaining.Minutes()))
	}
	h := int(remaining.Hours())
	m := int(remaining.Minutes()) % 60
	if m == 0 {
		return fmt.Sprintf("%dh left", h)
	}
	return fmt.Sprintf("%dh%dmin left", h, m)
}

// sectionSep renders a dimmed section separator line filling contentW.
func sectionSep(label string, contentW int) string {
	inner := "── " + label + " "
	runeCount := len([]rune(inner)) + 2 // +2 for "  " prefix
	dashCount := contentW - runeCount
	if dashCount < 2 {
		dashCount = 2
	}
	return "  " + ui.ItemDimmed.Render(inner+strings.Repeat("─", dashCount))
}

// fetchCmd calls the Google Calendar API in the background.
func fetchCmd() tea.Cmd {
	return func() tea.Msg {
		events, err := fetchFromAPI()
		return calendarFetchedMsg{events: events, err: err}
	}
}

// minuteTickCmd fires every minute to keep relative time labels fresh.
// Must be re-returned from Update() to keep ticking.
func minuteTickCmd() tea.Cmd {
	return tea.Tick(time.Minute, func(t time.Time) tea.Msg {
		return minuteTickMsg(t)
	})
}

// periodicRefreshCmd fires every 15 minutes as a safety-net API refresh.
func periodicRefreshCmd() tea.Cmd {
	return tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
		return periodicRefreshMsg{}
	})
}

func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := t.Sub(now)
	switch {
	case diff < 2*time.Hour:
		return fmt.Sprintf("in %dh%dm", int(diff.Hours()), int(diff.Minutes())%60)
	case diff < 24*time.Hour:
		return "Today " + t.Format("3:04 PM")
	case diff < 48*time.Hour:
		return "Tomorrow " + t.Format("3:04 PM")
	default:
		return t.Format("Mon Jan 2")
	}
}

func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
