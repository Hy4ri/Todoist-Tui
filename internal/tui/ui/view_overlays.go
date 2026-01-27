package ui

import (
	"fmt"
	"strings"

	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// renderTaskForm renders the add/edit task form.
func (r *Renderer) renderTaskForm() string {
	if r.TaskForm == nil {
		return styles.Dialog.Width(r.Width - 4).Render("Form not initialized")
	}

	var b strings.Builder
	f := r.TaskForm

	// Header
	title := "Add Task"
	if f.Original != nil {
		title = "Edit Task"
	}
	b.WriteString(styles.Title.Render(title) + "\n\n")

	// Content
	b.WriteString(styles.InputLabel.Render("Content") + "\n")
	b.WriteString(f.Content.View() + "\n\n")

	// Description
	b.WriteString(styles.InputLabel.Render("Description") + "\n")
	b.WriteString(f.Description.View() + "\n\n")

	// Priority & Due Date Row
	b.WriteString(styles.InputLabel.Render("Priority (1-4)") + "   " + styles.InputLabel.Render("Due Date") + "\n")
	pStyle := styles.GetPriorityStyle(f.Priority)
	pLabel := fmt.Sprintf("P%d", 5-f.Priority) // Display as P1-P4
	b.WriteString(pStyle.Render(pLabel) + "           " + f.DueString.View() + "\n\n")

	// Project Selector (if showing)
	if f.ShowProjectList {
		b.WriteString(styles.InputLabel.Render("Select Project") + "\n")
		// In a real implementation we'd show a list here.
		// For now just show selected project name.
		b.WriteString(styles.TaskSelected.Render(f.ProjectName) + "\n\n")
	} else {
		b.WriteString(styles.InputLabel.Render("Project: ") + styles.TaskContent.Render(f.ProjectName) + "\n\n")
	}

	// Help
	b.WriteString(styles.HelpDesc.Render("Enter: save | Esc: cancel | Tab: next field"))

	return styles.Dialog.Width(r.Width - 4).Render(b.String())
}

// renderSearch renders the search view.
func (r *Renderer) renderSearch() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Search Tasks"))
	b.WriteString("\n\n")

	// Search input
	b.WriteString(styles.InputLabel.Render("Query"))
	b.WriteString("\n")
	b.WriteString(r.SearchInput.View())
	b.WriteString("\n\n")

	// Results
	if r.SearchQuery == "" {
		b.WriteString(styles.HelpDesc.Render("Type to search..."))
	} else if len(r.SearchResults) == 0 {
		b.WriteString(styles.StatusBarError.Render("No results found"))
	} else {
		b.WriteString(styles.Subtitle.Render(fmt.Sprintf("Found %d task(s)", len(r.SearchResults))))
		b.WriteString("\n\n")

		// Render search results
		for i, task := range r.SearchResults {
			cursor := "  "
			itemStyle := styles.TaskItem
			if i == r.TaskCursor {
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

	return styles.Dialog.Width(r.Width - 4).Render(b.String())
}
