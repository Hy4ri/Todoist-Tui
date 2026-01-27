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

	return styles.Dialog.Width(r.Width - 4).Render(r.TaskForm.View())
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
