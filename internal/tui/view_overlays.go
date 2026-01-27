package tui

import (
	"fmt"
	"strings"

	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// renderTaskForm renders the add/edit task form.
func (a *App) renderTaskForm() string {
	if a.taskForm == nil {
		return styles.Dialog.Width(a.width - 4).Render("Form not initialized")
	}

	return styles.Dialog.Width(a.width - 4).Render(a.taskForm.View())
}

// renderSearch renders the search view.
func (a *App) renderSearch() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Search Tasks"))
	b.WriteString("\n\n")

	// Search input
	b.WriteString(styles.InputLabel.Render("Query"))
	b.WriteString("\n")
	b.WriteString(a.searchInput.View())
	b.WriteString("\n\n")

	// Results
	if a.searchQuery == "" {
		b.WriteString(styles.HelpDesc.Render("Type to search..."))
	} else if len(a.searchResults) == 0 {
		b.WriteString(styles.StatusBarError.Render("No results found"))
	} else {
		b.WriteString(styles.Subtitle.Render(fmt.Sprintf("Found %d task(s)", len(a.searchResults))))
		b.WriteString("\n\n")

		// Render search results
		for i, task := range a.searchResults {
			cursor := "  "
			itemStyle := styles.TaskItem
			if i == a.taskCursor {
				cursor = "> "
				itemStyle = styles.TaskSelected
			}

			checkbox := styles.CheckboxUnchecked
			if task.Checked {
				checkbox = styles.CheckboxChecked
			}

			content := task.Content
			priorityStyle := styles.GetPriorityStyle(task.Priority)
			content = priorityStyle.Render(content)

			// Due date
			due := ""
			if task.Due != nil {
				dueStr := task.DueDisplay()
				if task.IsOverdue() {
					due = styles.TaskDueOverdue.Render(" | " + dueStr)
				} else if task.IsDueToday() {
					due = styles.TaskDueToday.Render(" | " + dueStr)
				} else {
					due = styles.TaskDue.Render(" | " + dueStr)
				}
			}

			line := fmt.Sprintf("%s%s %s%s", cursor, checkbox, content, due)
			b.WriteString(itemStyle.Render(line))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("j/k: navigate | Enter: view | x: complete | Esc: back"))

	return styles.Dialog.Width(a.width - 4).Render(b.String())
}
