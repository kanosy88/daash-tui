package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// GoogleConfig holds Google OAuth credentials.
type GoogleConfig struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

// CalendarEntry is one agenda entry in the config file.
type CalendarEntry struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// TickTickProject is one TickTick project to display.
type TickTickProject struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

// TickTickConfig holds TickTick OAuth credentials and project list.
type TickTickConfig struct {
	ClientID     string            `yaml:"client_id"`
	ClientSecret string            `yaml:"client_secret"`
	AllProjects  bool              `yaml:"all_projects"` // if true, fetch all projects automatically
	Projects     []TickTickProject `yaml:"projects"`      // ignored when all_projects: true
}

// WeatherConfig holds weather location settings.
type WeatherConfig struct {
	City string `yaml:"city"` // e.g. "Brussels", "Paris", "Tokyo"
}

// Config holds user preferences loaded from ~/.config/daash/config.yaml.
type Config struct {
	Google    GoogleConfig    `yaml:"google"`
	Calendars []CalendarEntry `yaml:"calendars"`
	TickTick  TickTickConfig  `yaml:"ticktick"`
	Weather   WeatherConfig   `yaml:"weather"`
}

// Load reads ~/.config/daash/config.yaml.
// Missing sections fall back to sensible defaults.
func Load() Config {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig()
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultConfig()
	}
	if len(cfg.Calendars) == 0 {
		cfg.Calendars = defaultConfig().Calendars
	}
	return cfg
}

func defaultConfig() Config {
	return Config{
		Calendars: []CalendarEntry{
			{ID: "primary", Name: ""},
		},
	}
}

func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(home, ".config", "daash", "config.yaml")
}

// ConfigDir returns the daash config directory path.
func ConfigDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config", "daash")
}
