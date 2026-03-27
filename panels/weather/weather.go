package weather

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"daash/config"
	"daash/panels"
	"daash/ui"
)

// ForecastDay holds a single day's forecast.
type ForecastDay struct {
	Day       string
	High, Low int
	Condition string
}

// weatherFetchedMsg is sent when the API call completes (success or error).
type weatherFetchedMsg struct {
	data *weatherData
	err  error
}

// WeatherModel implements panels.Panel.
type WeatherModel struct {
	city        string
	temperature int
	condition   string
	humidity    int
	forecast    []ForecastDay
	loading     bool
	fetchErr    string
	lastSync    time.Time
	width       int
	height      int
	focused     bool
}

func New() *WeatherModel {
	city := config.Load().Weather.City
	if city == "" {
		city = "Brussels"
	}
	return &WeatherModel{city: city, loading: true}
}

func (m *WeatherModel) Title() string { return "Weather" }

func (m *WeatherModel) SetSize(width, height int) panels.Panel {
	m.width = width
	m.height = height
	return m
}

func (m *WeatherModel) IsFocused() bool     { return m.focused }
func (m *WeatherModel) LastSync() time.Time { return m.lastSync }

func (m *WeatherModel) Focus() panels.Panel {
	m.focused = true
	return m
}

func (m *WeatherModel) Blur() panels.Panel {
	m.focused = false
	return m
}

// Init triggers the first weather fetch.
func (m *WeatherModel) Init() tea.Cmd {
	return fetchCmd(m.city)
}

func (m *WeatherModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case weatherFetchedMsg:
		if msg.err != nil {
			m.loading = false
			m.fetchErr = msg.err.Error()
		} else {
			m.loading = false
			m.fetchErr = ""
			m.city = msg.data.city
			m.temperature = msg.data.temperature
			m.condition = msg.data.condition
			m.humidity = msg.data.humidity
			m.forecast = msg.data.forecast
			m.lastSync = time.Now()
		}
	case tea.KeyMsg:
		if msg.String() == "r" {
			m.loading = true
			m.fetchErr = ""
			return m, fetchCmd(m.city)
		}
	}
	return m, nil
}

func (m *WeatherModel) View() string {
	contentW := m.width - 4
	if contentW < 1 {
		contentW = 1
	}

	var lines []string

	switch {
	case m.loading:
		lines = append(lines, "")
		lines = append(lines, "  "+ui.ItemDimmed.Render("Fetching weather…"))

	case m.fetchErr != "":
		lines = append(lines, "")
		lines = append(lines, "  "+ui.PriorityHigh.Render("Error:"))
		lines = append(lines, "  "+ui.ItemDimmed.Render(truncate(m.fetchErr, contentW-2)))
		lines = append(lines, "")
		lines = append(lines, "  "+ui.ItemDimmed.Render("Press r to retry"))

	default:
		glyph := conditionGlyph(m.condition)
		lines = append(lines, ui.ItemNormal.Render(fmt.Sprintf("%s  %d°C  %s", glyph, m.temperature, m.condition)))
		lines = append(lines, ui.ItemDimmed.Render(fmt.Sprintf("   Humidity: %d%%  —  %s", m.humidity, m.city)))
		lines = append(lines, "")
		lines = append(lines, ui.ItemDimmed.Render("5-Day Forecast"))
		lines = append(lines, strings.Repeat("─", contentW))

		for _, day := range m.forecast {
			dayGlyph := conditionGlyph(day.Condition)
			row := fmt.Sprintf("%-6s %s  %d° / %d°  %s", day.Day, dayGlyph, day.High, day.Low, day.Condition)
			lines = append(lines, ui.ItemNormal.Render(truncate(row, contentW)))
		}
	}

	return ui.RenderPanel("\ue312 "+m.Title(), strings.Join(lines, "\n"), m.width, m.height, m.focused, ui.AccentWeather, ui.AccentWeatherDim)
}

// fetchCmd returns a Bubble Tea command that geocodes the city and fetches weather.
func fetchCmd(city string) tea.Cmd {
	return func() tea.Msg {
		data, err := fetchFromAPI(city)
		return weatherFetchedMsg{data: data, err: err}
	}
}

func conditionGlyph(condition string) string {
	lower := strings.ToLower(condition)
	switch {
	case strings.Contains(lower, "clear"):
		return "\ue30d" // nf-weather-day_sunny
	case strings.Contains(lower, "partly"):
		return "\ue302" // nf-weather-day_cloudy
	case strings.Contains(lower, "fog"):
		return "\ue313" // nf-weather-day_fog
	case strings.Contains(lower, "rain") || strings.Contains(lower, "shower") || strings.Contains(lower, "drizzle"):
		return "\ue308" // nf-weather-rain
	case strings.Contains(lower, "snow"):
		return "\ue30a" // nf-weather-snow
	case strings.Contains(lower, "thunder") || strings.Contains(lower, "storm"):
		return "\ue31d" // nf-weather-thunderstorm
	default:
		return "\ue312" // nf-weather-cloudy
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
