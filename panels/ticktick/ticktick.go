package ticktick

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/kanosy88/daash-tui/panels"
	"github.com/kanosy88/daash-tui/ui"
)

// ViewMode controls which tasks are shown.
type ViewMode int

const (
	ViewToday ViewMode = iota // tasks due today (or overdue)
	ViewWeek                  // tasks due within the next 7 days
	ViewAll                   // all tasks
)

var viewModeLabels = map[ViewMode]string{
	ViewToday: "Today",
	ViewWeek:  "7 Days",
	ViewAll:   "All",
}

// Task represents a TickTick todo item.
type Task struct {
	Title       string
	Status      int  // 0=active, 2=completed
	Priority    int  // 0=none, 1=low, 3=medium, 5=high
	DueDate     time.Time
	IsAllDay    bool
	IsRecurring bool
	ProjectName string // empty when only one project or no name set
}

// ticktickFetchedMsg is sent when the API call completes.
type ticktickFetchedMsg struct {
	tasks []Task
	err   error
}

// periodicRefreshMsg fires on a timer to refresh tasks.
type periodicRefreshMsg struct{}

// TickTickModel implements panels.Panel.
type TickTickModel struct {
	tasks    []Task
	cursor   int
	viewMode ViewMode
	loading  bool
	fetchErr string
	lastSync time.Time
	width    int
	height   int
	focused  bool
}

func New() *TickTickModel {
	return &TickTickModel{loading: true, viewMode: ViewToday}
}

func (m *TickTickModel) Title() string { return "TickTick" }

func (m *TickTickModel) SetSize(width, height int) panels.Panel {
	m.width = width
	m.height = height
	return m
}

func (m *TickTickModel) IsFocused() bool     { return m.focused }
func (m *TickTickModel) LastSync() time.Time { return m.lastSync }

func (m *TickTickModel) Focus() panels.Panel {
	m.focused = true
	return m
}

func (m *TickTickModel) Blur() panels.Panel {
	m.focused = false
	return m
}

func (m *TickTickModel) Init() tea.Cmd {
	return tea.Batch(fetchCmd(), periodicRefreshCmd())
}

func (m *TickTickModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case ticktickFetchedMsg:
		m.loading = false
		if msg.err != nil {
			m.fetchErr = msg.err.Error()
			return m, nil
		}
		m.fetchErr = ""
		m.tasks = msg.tasks
		m.cursor = 0
		m.lastSync = time.Now()
		return m, nil

	case periodicRefreshMsg:
		m.loading = true
		return m, tea.Batch(fetchCmd(), periodicRefreshCmd())

	case tea.KeyMsg:
		switch msg.String() {
		case ui.KeyScrollUp, ui.KeyScrollUpK:
			if m.cursor > 0 {
				m.cursor--
			}
		case ui.KeyScrollDown, ui.KeyScrollDownJ:
			visible := m.visibleTasks()
			if m.cursor < len(visible)-1 {
				m.cursor++
			}
		case "v":
			m.viewMode = (m.viewMode + 1) % 3
			m.cursor = 0
		case "r":
			m.loading = true
			m.fetchErr = ""
			return m, fetchCmd()
		}
	}
	return m, nil
}

func (m *TickTickModel) View() string {
	contentW := m.width - 4
	if contentW < 1 {
		contentW = 1
	}

	var lines []string

	switch {
	case m.loading:
		lines = append(lines, "")
		lines = append(lines, "  "+ui.ItemDimmed.Render("Fetching tasks…"))

	case m.fetchErr != "":
		lines = append(lines, "")
		lines = append(lines, "  "+ui.PriorityHigh.Render("Error:"))
		lines = append(lines, "  "+ui.ItemDimmed.Render(truncate(m.fetchErr, contentW-2)))
		lines = append(lines, "")
		lines = append(lines, "  "+ui.ItemDimmed.Render("Press r to retry"))

	default:
		visible := m.visibleTasks()
		if len(visible) == 0 {
			lines = append(lines, "  "+ui.ItemDimmed.Render("No tasks."))
		} else if m.viewMode == ViewAll {
			lines = append(lines, m.renderGrouped(visible, contentW)...)
		} else {
			lines = append(lines, m.renderFlat(visible, contentW)...)
		}
	}

	label := viewModeLabels[m.viewMode]
	title := "󰄨 " + m.Title() + " ─ " + label

	return ui.RenderPanel(title, strings.Join(lines, "\n"), m.width, m.height, m.focused, ui.AccentTickTick, ui.AccentTickTickDim)
}

// renderFlat renders tasks as a flat list with [ProjectName] tag below each.
// Used in Today and 7 Days modes.
func (m *TickTickModel) renderFlat(tasks []Task, contentW int) []string {
	var lines []string
	for i, task := range tasks {
		lines = append(lines, m.renderTask(task, i, contentW)...)
		if task.ProjectName != "" {
			lines = append(lines, "    "+ui.ItemDimmed.Render("["+task.ProjectName+"]"))
		}
	}
	return lines
}

// renderGrouped renders tasks grouped by project with section separators.
// Used in All mode.
func (m *TickTickModel) renderGrouped(tasks []Task, contentW int) []string {
	type group struct {
		name  string
		tasks []Task
		start int // index of first task in the flat list (for cursor)
	}

	// Build ordered groups preserving first-appearance order.
	var groups []group
	seen := map[string]int{} // project name → group index
	for _, t := range tasks {
		name := t.ProjectName
		if idx, ok := seen[name]; ok {
			groups[idx].tasks = append(groups[idx].tasks, t)
		} else {
			seen[name] = len(groups)
			groups = append(groups, group{name: name, tasks: []Task{t}})
		}
	}

	// Assign flat cursor indices.
	idx := 0
	for i := range groups {
		groups[i].start = idx
		idx += len(groups[i].tasks)
	}

	var lines []string
	for _, g := range groups {
		if g.name != "" {
			lines = append(lines, taskSectionSep(g.name, contentW))
		}
		for j, task := range g.tasks {
			lines = append(lines, m.renderTask(task, g.start+j, contentW)...)
		}
	}
	return lines
}

// renderTask renders a single task row. cursorIdx is its position in the flat visible list.
func (m *TickTickModel) renderTask(task Task, cursorIdx int, contentW int) []string {
	checkbox := "[ ]"
	taskStyle := ui.ItemNormal
	if task.Status == 2 {
		checkbox = "[x]"
		taskStyle = ui.ItemDimmed
	}

	prioIndicator := priorityStyle(task.Priority).Render("●")
	prefix := "  "
	if cursorIdx == m.cursor && m.focused {
		taskStyle = ui.ItemSelected
		prefix = "▸ "
	}

	suffix := ""
	if task.IsRecurring {
		suffix += " " + ui.ItemDimmed.Render("↻")
	}
	if !task.DueDate.IsZero() && task.Status != 2 {
		suffix += "  " + ui.ItemDimmed.Render(formatDue(task.DueDate))
	}

	titleWidth := contentW - 10
	if task.IsRecurring {
		titleWidth -= 2
	}
	line := prefix + checkbox + " " + prioIndicator + " " + taskStyle.Render(truncate(task.Title, titleWidth)) + suffix
	return []string{line}
}

// taskSectionSep renders a dimmed section separator for a project group.
func taskSectionSep(label string, contentW int) string {
	inner := "── " + label + " "
	runeCount := len([]rune(inner)) + 2
	dashCount := contentW - runeCount
	if dashCount < 2 {
		dashCount = 2
	}
	return "  " + ui.ItemDimmed.Render(inner+strings.Repeat("─", dashCount))
}

// visibleTasks returns tasks filtered by the current ViewMode.
func (m *TickTickModel) visibleTasks() []Task {
	todayStart := startOfDay(time.Now())
	in7daysStart := todayStart.Add(7 * 24 * time.Hour)

	var result []Task
	for _, t := range m.tasks {
		switch m.viewMode {
		case ViewToday:
			if !t.DueDate.IsZero() && !startOfDay(t.DueDate).After(todayStart) {
				result = append(result, t)
			}
		case ViewWeek:
			if !t.DueDate.IsZero() && startOfDay(t.DueDate).Before(in7daysStart) {
				result = append(result, t)
			}
		case ViewAll:
			result = append(result, t)
		}
	}
	return result
}

func fetchCmd() tea.Cmd {
	return func() tea.Msg {
		tasks, err := fetchFromAPI()
		return ticktickFetchedMsg{tasks: tasks, err: err}
	}
}

func periodicRefreshCmd() tea.Cmd {
	return tea.Tick(10*time.Minute, func(t time.Time) tea.Msg {
		return periodicRefreshMsg{}
	})
}

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func priorityStyle(p int) lipgloss.Style {
	switch p {
	case 5:
		return ui.PriorityHigh
	case 3:
		return ui.PriorityMedium
	default:
		return ui.PriorityLow
	}
}

func formatDue(t time.Time) string {
	diff := time.Until(t)
	switch {
	case diff < 0:
		return "overdue"
	case diff < 24*time.Hour:
		return "today"
	case diff < 48*time.Hour:
		return "tomorrow"
	default:
		return t.Format("Jan 2")
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
