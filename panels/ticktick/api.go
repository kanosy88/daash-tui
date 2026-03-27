package ticktick

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/kanosy88/daash-tui/config"
)

const apiBase = "https://ticktick.com/open/v1"

// apiTask mirrors the TickTick Open API task object.
type apiTask struct {
	ID           string  `json:"id"`
	Title        string  `json:"title"`
	Status       int     `json:"status"`      // 0=active, 2=completed
	Priority     int     `json:"priority"`    // 0=none, 1=low, 3=medium, 5=high
	DueDate      string  `json:"dueDate"`     // RFC3339-like, may be empty
	IsAllDay     bool    `json:"isAllDay"`
	RepeatFlag   string  `json:"repeatFlag"`  // non-empty = recurring
	ProjectID    string  `json:"projectId"`
	CompletedTime string `json:"completedTime"`
}

type projectData struct {
	Tasks []apiTask `json:"tasks"`
}

// fetchFromAPI fetches tasks from configured TickTick projects.
// If all_projects: true, it first fetches the project list from the API.
func fetchFromAPI() ([]Task, error) {
	ctx := context.Background()
	client, err := httpClient(ctx)
	if err != nil {
		return nil, err
	}

	cfg := config.Load().TickTick

	var projects []config.TickTickProject
	if cfg.AllProjects {
		projects, err = listProjects(client)
		if err != nil {
			return nil, fmt.Errorf("fetching project list: %w", err)
		}
	} else {
		if len(cfg.Projects) == 0 {
			return nil, fmt.Errorf("no ticktick projects configured — set all_projects: true or list projects in config.yaml")
		}
		projects = cfg.Projects
	}

	showProject := len(projects) > 1

	var all []Task
	for _, proj := range projects {
		tasks, err := fetchProject(client, proj.ID)
		if err != nil {
			return nil, fmt.Errorf("project %s: %w", proj.Name, err)
		}
		for _, t := range tasks {
			task, ok := parseTask(t)
			if !ok {
				continue
			}
			if showProject && proj.Name != "" {
				task.ProjectName = proj.Name
			}
			all = append(all, task)
		}
	}

	return all, nil
}

// listProjects fetches all projects from the TickTick API.
func listProjects(client *http.Client) ([]config.TickTickProject, error) {
	resp, err := client.Get(apiBase + "/project")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Closed bool   `json:"closed"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var projects []config.TickTickProject
	for _, p := range raw {
		if !p.Closed {
			projects = append(projects, config.TickTickProject{ID: p.ID, Name: p.Name})
		}
	}
	return projects, nil
}

func fetchProject(client *http.Client, projectID string) ([]apiTask, error) {
	url := fmt.Sprintf("%s/project/%s/data", apiBase, projectID)
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var data projectData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data.Tasks, nil
}

// PrintProjects prints all configured project IDs and names.
// Used by --list-ticktick-projects CLI flag.
func PrintProjects() error {
	ctx := context.Background()
	client, err := httpClient(ctx)
	if err != nil {
		return err
	}

	resp, err := client.Get(apiBase + "/project")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var projects []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return err
	}

	fmt.Println("Available TickTick projects:")
	fmt.Println()
	for _, p := range projects {
		fmt.Printf("  name: %q\n  id:   %q\n\n", p.Name, p.ID)
	}
	return nil
}

func parseTask(t apiTask) (Task, bool) {
	// Skip completed tasks (status==2) and tasks with a completedTime
	// (past occurrences of recurring tasks).
	if t.Status == 2 || t.CompletedTime != "" {
		return Task{}, false
	}

	var due time.Time
	if t.DueDate != "" {
		// TickTick sends dates like "2026-03-25T23:00:00.000+0000"
		parsed, err := time.Parse("2006-01-02T15:04:05.000-0700", t.DueDate)
		if err != nil {
			// fallback: try without milliseconds
			parsed, err = time.Parse("2006-01-02T15:04:05-0700", t.DueDate)
		}
		if err == nil {
			due = parsed
		}
	}

	return Task{
		Title:       t.Title,
		Status:      t.Status,
		Priority:    t.Priority,
		DueDate:     due,
		IsAllDay:    t.IsAllDay,
		IsRecurring: t.RepeatFlag != "",
	}, true
}
