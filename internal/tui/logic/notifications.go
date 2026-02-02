package logic

import (
	"log"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gen2brain/beeep"
)

// Debug logger
var debugLog *log.Logger

func init() {
	f, err := os.OpenFile("debug.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err == nil {
		debugLog = log.New(f, "NOTIF: ", log.Ltime|log.Lshortfile)
	}
}

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

	if debugLog != nil {
		debugLog.Printf("Checking notifications at %v. Task count: %d", t, len(h.AllTasks))
	}

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
				if debugLog != nil {
					debugLog.Printf("Failed to parse datetime %s for task %s: %v", *task.Due.Datetime, task.Content, err)
				}
				continue
			}
			dueTime = dueTime.Local()
		} else if task.Due.Date != "" {
			// Day-only task: default to 9:00 AM local time
			// We parse the date (YYYY-MM-DD) and add 9 hours
			parsedDate, err := time.ParseInLocation("2006-01-02", task.Due.Date[:10], time.Local)
			if err != nil {
				if debugLog != nil {
					debugLog.Printf("Failed to parse date %s for task %s: %v", task.Due.Date, task.Content, err)
				}
				continue
			}
			dueTime = parsedDate.Add(9 * time.Hour)
			isDayOnly = true
		} else {
			continue
		}

		if debugLog != nil {
			debugLog.Printf("Task '%s' due at %v (Now: %v)", task.Content, dueTime, t)
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
				if debugLog != nil {
					debugLog.Printf("Skipping old task '%s' (diff: %v)", task.Content, t.Sub(dueTime))
				}
				// Mark as notified silently so we don't check again
				h.NotifiedTasks[task.ID] = true
				continue
			}

			if debugLog != nil {
				debugLog.Printf("Notifying for task '%s'", task.Content)
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
				err := beeep.Notify(project, "Task Due: "+content, "")
				if err != nil && debugLog != nil {
					debugLog.Printf("Failed to send notification: %v", err)
				}
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
