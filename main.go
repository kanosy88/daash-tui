package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kanosy88/daash-tui/model"
	"github.com/kanosy88/daash-tui/panels/calendar"
	"github.com/kanosy88/daash-tui/panels/ticktick"
	"github.com/kanosy88/daash-tui/panels/weather"
)

func main() {
	if err := calendar.EnsureAuth(); err != nil {
		fmt.Fprintln(os.Stderr, "Google Calendar auth:", err)
		os.Exit(1)
	}

	if err := ticktick.EnsureAuth(); err != nil {
		fmt.Fprintln(os.Stderr, "TickTick auth:", err)
		os.Exit(1)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--help", "-h":
			fmt.Println(`daash — personal life dashboard

USAGE:
  daash [command]

COMMANDS:
  (no command)               Launch the TUI dashboard
  --list-calendars           List all Google Calendar IDs available to your account
  --list-ticktick-projects   List all TickTick project IDs available to your account
  --help, -h                 Show this help message

KEYBINDINGS (in TUI):
  Tab / Shift+Tab     Cycle focus between panels
  ↑ / k               Scroll up
  ↓ / j               Scroll down
  v                   Cycle TickTick view (Today / 7 Days / All)  [TickTick panel]
  r                   Force refresh                               [Calendar / TickTick]
  q / Ctrl+C          Quit

CONFIG:
  ~/.config/daash/config.yaml   Calendars and TickTick projects (see --list-* commands)`)
			return

		case "--list-calendars":
			if err := calendar.PrintCalendars(); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			return

		case "--list-ticktick-projects":
			if err := ticktick.PrintProjects(); err != nil {
				fmt.Fprintln(os.Stderr, "Error:", err)
				os.Exit(1)
			}
			return

		default:
			fmt.Fprintf(os.Stderr, "Unknown command: %s\nRun 'daash --help' for usage.\n", os.Args[1])
			os.Exit(1)
		}
	}

	app := model.New(
		calendar.New(),
		ticktick.New(),
		weather.New(),
	)

	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
