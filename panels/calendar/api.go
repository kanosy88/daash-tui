package calendar

import (
	"context"
	"fmt"
	"sort"
	"time"

	"daash/config"
	gcal "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

// CalendarInfo holds the ID and name of a Google Calendar.
type CalendarInfo struct {
	ID   string
	Name string
}

// ListCalendars returns all calendars accessible with the current OAuth token.
func ListCalendars() ([]CalendarInfo, error) {
	ctx := context.Background()
	client, err := oauthClient(ctx)
	if err != nil {
		return nil, err
	}
	svc, err := gcal.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}
	list, err := svc.CalendarList.List().Do()
	if err != nil {
		return nil, err
	}
	var result []CalendarInfo
	for _, item := range list.Items {
		result = append(result, CalendarInfo{ID: item.Id, Name: item.Summary})
	}
	return result, nil
}

// PrintCalendars prints all accessible calendars to stdout.
func PrintCalendars() error {
	cals, err := ListCalendars()
	if err != nil {
		return err
	}
	fmt.Println("Available calendars:")
	fmt.Println()
	for _, c := range cals {
		fmt.Printf("  name: %q\n  id:   %q\n\n", c.Name, c.ID)
	}
	return nil
}

// fetchFromAPI fetches events from all calendars listed in the config.
func fetchFromAPI() ([]Event, error) {
	ctx := context.Background()

	client, err := oauthClient(ctx)
	if err != nil {
		return nil, err
	}

	svc, err := gcal.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, err
	}

	cfg := config.Load()
	showCalendarName := len(cfg.Calendars) > 1

	now := time.Now()
	tMin := now.Format(time.RFC3339)
	tMax := now.Add(30 * 24 * time.Hour).Format(time.RFC3339)

	var all []Event
	for _, cal := range cfg.Calendars {
		items, err := svc.Events.List(cal.ID).
			TimeMin(tMin).
			TimeMax(tMax).
			OrderBy("startTime").
			SingleEvents(true).
			MaxResults(50).
			Do()
		if err != nil {
			return nil, err
		}
		for _, item := range items.Items {
			ev, ok := parseEvent(item)
			if !ok {
				continue
			}
			if showCalendarName && cal.Name != "" {
				ev.CalendarName = cal.Name
			}
			all = append(all, ev)
		}
	}

	// Merge and sort by start time across all calendars.
	sort.Slice(all, func(i, j int) bool {
		return all[i].Time.Before(all[j].Time)
	})

	// Cap total results.
	if len(all) > 50 {
		all = all[:50]
	}

	return all, nil
}

func parseEvent(item *gcal.Event) (Event, bool) {
	var start time.Time

	switch {
	case item.Start.DateTime != "":
		t, err := time.Parse(time.RFC3339, item.Start.DateTime)
		if err != nil {
			return Event{}, false
		}
		start = t

	case item.Start.Date != "":
		// All-day event
		t, err := time.Parse("2006-01-02", item.Start.Date)
		if err != nil {
			return Event{}, false
		}
		start = t

	default:
		return Event{}, false
	}

	var duration time.Duration
	if item.End.DateTime != "" {
		end, err := time.Parse(time.RFC3339, item.End.DateTime)
		if err == nil {
			duration = end.Sub(start)
		}
	}

	return Event{
		Title:    item.Summary,
		Time:     start,
		Duration: duration,
		Location: item.Location,
	}, true
}
