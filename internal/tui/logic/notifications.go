package logic

import (
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
			// Try other formats if needed, or skip
			continue
		}

		// Adjust to local time if needed
		dueTime = dueTime.Local()

		// Check if due time has passed
		if t.After(dueTime) || t.Equal(dueTime) {
			// Only notify if it was due recently (e.g. within last 2 minutes)
			// This prevents a flood of notifications for old overdue tasks on startup
			if t.Sub(dueTime) > 2*time.Minute {
				// Mark as notified silently so we don't check again
				h.NotifiedTasks[task.ID] = true
				continue
			}

			// Mark as notified
			h.NotifiedTasks[task.ID] = true

			// Capture task content for closure
			content := task.Content

			// Add notification command
			cmds = append(cmds, func() tea.Msg {
				_ = exec.Command("notify-send", "Todoist", "Task Due: "+content).Run()
				return nil
			})
		}
	}

	return tea.Batch(cmds...)
}
