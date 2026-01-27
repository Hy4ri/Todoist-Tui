package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// renderDetailPanel renders task details in the right panel for split view.
func (a *App) renderDetailPanel(width, height int) string {
	if a.selectedTask == nil {
		return ""
	}

	t := a.selectedTask

	// Create border style
	panelStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Highlight).
		Padding(0, 1).
		Width(width - 2).
		Height(height - 2)

	// Build content
	var content strings.Builder

	// Title - use Width to ensure Lipgloss handles wrapping and style continuation
	content.WriteString(styles.Title.Width(width-6).Render(t.Content) + "\n\n")

	// Due date
	if t.Due != nil {
		content.WriteString(styles.StatusBarKey.Render("Due: "))
		content.WriteString(t.Due.String + "\n")
	}

	// Priority
	priorityStyle := styles.GetPriorityStyle(t.Priority)
	priorityLabel := fmt.Sprintf("P%d", 5-t.Priority)
	content.WriteString(styles.StatusBarKey.Render("Priority: "))
	content.WriteString(priorityStyle.Render(priorityLabel) + "\n")

	// Description
	if t.Description != "" {
		content.WriteString("\n" + styles.StatusBarKey.Render("Description:") + "\n")
		content.WriteString(t.Description + "\n")
	}

	// Comments
	if len(a.comments) > 0 {
		content.WriteString("\n" + styles.StatusBarKey.Render(fmt.Sprintf("Comments (%d):", len(a.comments))) + "\n")
		for _, c := range a.comments {
			content.WriteString("• " + c.Content + "\n")
		}
	}

	// Help
	content.WriteString("\n" + styles.HelpDesc.Render("Esc to close"))

	return panelStyle.Render(content.String())
}

// renderTaskDetail renders the task detail view.
func (a *App) renderTaskDetail() string {
	if a.selectedTask == nil {
		return "No task selected"
	}

	t := a.selectedTask
	var b strings.Builder

	// Title with checkbox status
	checkbox := "[ ]"
	if t.Checked {
		checkbox = "[x]"
	}
	b.WriteString(styles.Title.Render("Task Details"))
	b.WriteString("\n\n")

	// Task content (main title)
	priorityStyle := styles.GetPriorityStyle(t.Priority)
	b.WriteString(fmt.Sprintf("  %s %s\n\n", checkbox, priorityStyle.Render(t.Content)))

	// Horizontal divider
	b.WriteString(styles.DetailSection.Render("  " + strings.Repeat("─", 40)))
	b.WriteString("\n\n")

	// Description (if present)
	if t.Description != "" {
		b.WriteString(styles.DetailIcon.Render("  📝"))
		b.WriteString(styles.DetailLabel.Render("Description"))
		b.WriteString("\n")
		b.WriteString(styles.DetailDescription.Render(t.Description))
		b.WriteString("\n\n")
	}

	// Due date
	if t.Due != nil {
		dueIcon := "📅"
		dueStyle := styles.DetailValue
		if t.IsOverdue() {
			dueIcon = "🔴"
			dueStyle = styles.TaskDueOverdue
		} else if t.IsDueToday() {
			dueIcon = "🟢"
			dueStyle = styles.TaskDueToday
		}
		b.WriteString(styles.DetailIcon.Render("  " + dueIcon))
		b.WriteString(styles.DetailLabel.Render("Due"))
		b.WriteString(dueStyle.Render(t.Due.String))
		if t.Due.IsRecurring {
			b.WriteString(styles.HelpDesc.Render(" (recurring)"))
		}
		b.WriteString("\n")
	}

	// Priority
	priorityIcon := "⚪"
	priorityLabel := "P4 (Low)"
	switch t.Priority {
	case 4:
		priorityIcon = "🔴"
		priorityLabel = "P1 (Urgent)"
	case 3:
		priorityIcon = "🟠"
		priorityLabel = "P2 (High)"
	case 2:
		priorityIcon = "🟡"
		priorityLabel = "P3 (Medium)"
	}
	b.WriteString(styles.DetailIcon.Render("  " + priorityIcon))
	b.WriteString(styles.DetailLabel.Render("Priority"))
	b.WriteString(priorityStyle.Render(priorityLabel))
	b.WriteString("\n")

	// Labels
	if len(t.Labels) > 0 {
		b.WriteString(styles.DetailIcon.Render("  🏷️"))
		b.WriteString(styles.DetailLabel.Render("Labels"))
		for i, l := range t.Labels {
			if i > 0 {
				b.WriteString(" ")
			}
			b.WriteString(styles.TaskLabel.Render("@" + l))
		}
		b.WriteString("\n")
	}

	// Project (find name)
	if t.ProjectID != "" {
		projectName := t.ProjectID
		for _, p := range a.projects {
			if p.ID == t.ProjectID {
				projectName = p.Name
				break
			}
		}
		b.WriteString(styles.DetailIcon.Render("  📁"))
		b.WriteString(styles.DetailLabel.Render("Project"))
		b.WriteString(styles.DetailValue.Render(projectName))
		b.WriteString("\n")
	}

	// Comment count
	if t.NoteCount > 0 {
		b.WriteString(styles.DetailIcon.Render("  💬"))
		b.WriteString(styles.DetailLabel.Render("Comments"))
		b.WriteString(styles.DetailValue.Render(fmt.Sprintf("%d", t.NoteCount)))
		b.WriteString("\n")
	}

	// Comments section
	if len(a.comments) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.DetailSection.Render("  " + strings.Repeat("─", 40)))
		b.WriteString("\n")
		b.WriteString(styles.Subtitle.Render("  Comments"))
		b.WriteString("\n\n")

		for _, c := range a.comments {
			// Parse and format timestamp
			timestamp := c.PostedAt
			if t, err := time.Parse(time.RFC3339, c.PostedAt); err == nil {
				timestamp = t.Format("Jan 2, 2006 3:04 PM")
			}
			b.WriteString(styles.CommentAuthor.Render(fmt.Sprintf("    %s", timestamp)))
			b.WriteString("\n")
			b.WriteString(styles.CommentContent.Render(fmt.Sprintf("    %s", c.Content)))
			b.WriteString("\n\n")
		}
	}

	// Divider before help
	b.WriteString(styles.DetailSection.Render("  " + strings.Repeat("─", 40)))
	b.WriteString("\n\n")

	// Help section
	b.WriteString(styles.HelpDesc.Render("  Shortcuts: "))
	b.WriteString(styles.HelpKey.Render("ESC"))
	b.WriteString(styles.HelpDesc.Render(" back  "))
	b.WriteString(styles.HelpKey.Render("x"))
	b.WriteString(styles.HelpDesc.Render(" complete  "))
	b.WriteString(styles.HelpKey.Render("e"))
	b.WriteString(styles.HelpDesc.Render(" edit  "))
	b.WriteString(styles.HelpKey.Render("s"))
	b.WriteString(styles.HelpDesc.Render(" add subtask"))

	return styles.Dialog.Width(a.width - 4).Render(b.String())
}
