package logic

import (
	"log"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
		// Skip if already notified, completed, or deleted
		if h.NotifiedTasks[task.ID] || task.Checked || task.IsDeleted {
			continue
		}

		// Skip if no due date
		if task.Due == nil || task.Due.Datetime == nil {
			continue
		}

		// Parse due datetime
		// Format is usually RFC3339: "2023-10-25T14:30:00Z" or similar
		dueTime, err := time.Parse(time.RFC3339, *task.Due.Datetime)
		if err != nil {
			// Try other formats if needed
			if debugLog != nil {
				debugLog.Printf("Failed to parse datetime %s for task %s: %v", *task.Due.Datetime, task.Content, err)
			}
			continue
		}

		// Adjust to local time if needed
		dueTime = dueTime.Local()

		if debugLog != nil {
			debugLog.Printf("Task '%s' due at %v (Now: %v)", task.Content, dueTime, t)
		}

		// Check if due time has passed
		if t.After(dueTime) || t.Equal(dueTime) {
			// Only notify if it was due recently (e.g. within last 2 minutes)
			// This prevents a flood of notifications for old overdue tasks on startup
			if t.Sub(dueTime) > 2*time.Minute {
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

			// Add notification command
			cmds = append(cmds, func() tea.Msg {
				err := exec.Command("notify-send", "Todoist", "Task Due: "+content).Run()
				if err != nil && debugLog != nil {
					debugLog.Printf("Failed to send notification: %v", err)
				}
				return nil
			})
		}
	}

	return tea.Batch(cmds...)
}
