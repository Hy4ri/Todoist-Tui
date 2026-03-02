package logic

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gen2brain/beeep"
	"github.com/hy4ri/todoist-tui/internal/api"
)


type checkDueMsg time.Time

func checkDueCmd() tea.Cmd {
	return tea.Tick(time.Minute, func(t time.Time) tea.Msg {
		return checkDueMsg(t)
	})
}

func (h *Handler) handleCheckDue(t time.Time) tea.Cmd {
	var cmds []tea.Cmd

	// Always schedule the next check
	cmds = append(cmds, checkDueCmd())


	// Check for due tasks
	for _, task := range h.AllTasks {
		// Skip if already notified, checked, or deleted
		if h.NotifiedTasks[task.ID] || task.Checked || task.IsDeleted {
			continue
		}

		if task.Due == nil {
			continue
		}

		// Determine the effective due time
		var dueTime time.Time
		var err error
		var isDayOnly bool

		if task.Due.Datetime != nil && *task.Due.Datetime != "" {
			// Time-specific task
			dueTime, err = time.Parse(time.RFC3339, *task.Due.Datetime)
			if err != nil {
				continue
			}
			dueTime = dueTime.Local()
		} else if task.Due.Date != "" {
			// Day-only task: default to 9:00 AM local time
			// We parse the date (YYYY-MM-DD) and add 9 hours
			parsedDate, err := time.ParseInLocation("2006-01-02", task.Due.Date[:10], time.Local)
			if err != nil {
				continue
			}
			dueTime = parsedDate.Add(9 * time.Hour)
			isDayOnly = true
		} else {
			continue
		}


		// Check if due time has passed
		if t.After(dueTime) || t.Equal(dueTime) {
			// For notifications, we want to be reasonably timely.
			// Ideally fewer than X minutes past the due time to avoid spamming on startup.
			// However, for day-only tasks (09:00 AM), user might open app at 09:05 and expect it.
			threshold := 5 * time.Minute
			if isDayOnly {
				threshold = 60 * time.Minute // Wider window for day tasks on startup
			}

			if t.Sub(dueTime) > threshold {
				// Mark as notified silently so we don't check again
				h.NotifiedTasks[task.ID] = true
				continue
			}


			// Mark as notified
			h.NotifiedTasks[task.ID] = true

			// Capture task content for closure
			content := task.Content
			project := "Todoist"
			if p, ok := h.getProjectName(task.ProjectID); ok {
				project = p
			}

			// Add notification command
			cmds = append(cmds, func() tea.Msg {
				_ = beeep.Notify(project, "Task Due: "+content, "")
				return nil
			})
		}
	}

	// Check for reminders
	for _, rem := range h.Reminders {
		// Skip if already notified
		if h.NotifiedTasks[rem.ID] {
			continue
		}

		// Calculate trigger time
		var triggerTime time.Time
		var content string

		if rem.Type == "relative" {
			// Find associated task
			var task *api.Task
			for i := range h.AllTasks {
				if h.AllTasks[i].ID == rem.ItemID {
					task = &h.AllTasks[i]
					break
				}
			}
			if task == nil || task.Checked || task.IsDeleted || task.Due == nil {
				continue
			}

			// Calculate based on task due time
			var taskDue time.Time
			var err error
			if task.Due.Datetime != nil && *task.Due.Datetime != "" {
				taskDue, err = time.Parse(time.RFC3339, *task.Due.Datetime)
			} else if task.Due.Date != "" {
				taskDue, err = time.ParseInLocation("2006-01-02", task.Due.Date[:10], time.Local)
				if err == nil {
					taskDue = taskDue.Add(9 * time.Hour) // Same default as tasks
				}
			} else {
				continue
			}

			if err != nil {
				continue
			}

			// Apply offset (minutes before)
			triggerTime = taskDue.Add(-time.Duration(rem.MinuteOffset) * time.Minute)
			content = fmt.Sprintf("Reminder: %s (%d min before)", task.Content, rem.MinuteOffset)

		} else if rem.Type == "absolute" && rem.Due != nil {
			// Absolute reminder
			// Todoist returns YYYY-MM-DDTHH:MM:SS (ISO8601) usually for absolute reminders
			// Let's try parsing
			var err error
			triggerTime, err = time.Parse("2006-01-02T15:04:05", rem.Due.Date)
			if err != nil {
				// Try RFC3339
				triggerTime, err = time.Parse(time.RFC3339, rem.Due.Date)
			}
			if err != nil {
				continue
			}

			// Find task for context (optional, but good for message)
			for i := range h.AllTasks {
				if h.AllTasks[i].ID == rem.ItemID {
					content = fmt.Sprintf("Reminder: %s", h.AllTasks[i].Content)
					break
				}
			}
			if content == "" {
				content = "Reminder for task"
			}
		} else {
			continue
		}

		triggerTime = triggerTime.Local()

		// Check if time passed
		if t.After(triggerTime) || t.Equal(triggerTime) {
			// Max delay threshold (e.g. 5 mins) to avoid spamming old reminders
			if t.Sub(triggerTime) > 5*time.Minute {
				h.NotifiedTasks[rem.ID] = true
				continue
			}


			h.NotifiedTasks[rem.ID] = true

			cmds = append(cmds, func() tea.Msg {
				_ = beeep.Notify("Todoist", content, "")
				return nil
			})
		}
	}

	return tea.Batch(cmds...)
}

func (h *Handler) getProjectName(id string) (string, bool) {
	for _, p := range h.Projects {
		if p.ID == id {
			return p.Name, true
		}
	}
	return "", false
}
