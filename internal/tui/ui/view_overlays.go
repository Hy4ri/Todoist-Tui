package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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

// renderCenteredDialog renders a dialog box centered on the screen.
func (r *Renderer) renderCenteredDialog(content string, width int) string {
	dialogStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Highlight).
		Padding(1, 2).
		Width(width)

	// Special case for error dialogs (red border)
	if strings.Contains(content, "⚠️") {
		dialogStyle = dialogStyle.BorderForeground(styles.ErrorColor)
	}

	dialog := dialogStyle.Render(content)
	return dialog
}

// Helper to overlay a dialog on existing content
func (r *Renderer) overlayContent(background string, dialog string) string {
	contentLines := strings.Split(background, "\n")
	dialogLines := strings.Split(dialog, "\n")

	dialogWidth := lipgloss.Width(dialogLines[0]) // Approximate width
	dialogHeight := len(dialogLines)

	// Center vertically
	startLine := (len(contentLines) - dialogHeight) / 2
	if startLine < 0 {
		startLine = 0
	}

	// Center horizontally padding
	leftPad := (r.Width - dialogWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	centeredDialogLines := make([]string, len(dialogLines))
	for i, line := range dialogLines {
		centeredDialogLines[i] = strings.Repeat(" ", leftPad) + line
	}

	// Overlay
	for i := 0; i < len(centeredDialogLines) && startLine+i < len(contentLines); i++ {
		contentLines[startLine+i] = centeredDialogLines[i]
	}

	return strings.Join(contentLines, "\n")
}

// renderProjectDialog renders the new project dialog.
func (r *Renderer) renderProjectDialog() string {
	content := styles.Title.Render("📁 New Project") + "\n\n" +
		r.ProjectInput.View() + "\n\n" +
		styles.HelpDesc.Render("Enter: create • Esc: cancel")

	return r.renderCenteredDialog(content, 50)
}

// renderProjectEditDialog renders the edit project dialog.
func (r *Renderer) renderProjectEditDialog() string {
	content := styles.Title.Render("✏️ Edit Project") + "\n\n" +
		r.ProjectInput.View() + "\n\n" +
		styles.HelpDesc.Render("Enter: save • Esc: cancel")

	return r.renderCenteredDialog(content, 50)
}

// renderProjectDeleteDialog renders the project delete confirmation.
func (r *Renderer) renderProjectDeleteDialog() string {
	if r.EditingProject == nil {
		return ""
	}
	content := styles.StatusBarError.Render("⚠️ Delete Project?") + "\n\n" +
		fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", r.EditingProject.Name) +
		styles.HelpDesc.Render("This will delete all tasks in this project.") + "\n\n" +
		styles.HelpDesc.Render("y: confirm • n/Esc: cancel")

	return r.renderCenteredDialog(content, 50)
}

// renderLabelDialog renders the new label dialog.
func (r *Renderer) renderLabelDialog() string {
	content := styles.Title.Render("🏷️ New Label") + "\n\n" +
		r.LabelInput.View() + "\n\n" +
		styles.HelpDesc.Render("Enter: create • Esc: cancel")

	return r.renderCenteredDialog(content, 50)
}

// renderLabelEditDialog renders the edit label dialog.
func (r *Renderer) renderLabelEditDialog() string {
	content := styles.Title.Render("✏️ Edit Label") + "\n\n" +
		r.LabelInput.View() + "\n\n" +
		styles.HelpDesc.Render("Enter: save • Esc: cancel")

	return r.renderCenteredDialog(content, 50)
}

// renderLabelDeleteDialog renders the label delete confirmation.
func (r *Renderer) renderLabelDeleteDialog() string {
	if r.EditingLabel == nil {
		return ""
	}
	content := styles.StatusBarError.Render("⚠️ Delete Label?") + "\n\n" +
		fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", r.EditingLabel.Name) +
		styles.HelpDesc.Render("y: confirm • n/Esc: cancel")

	return r.renderCenteredDialog(content, 50)
}

// renderSubtaskDialog renders the add subtask dialog.
func (r *Renderer) renderSubtaskDialog() string {
	content := styles.Title.Render("➕ Add Subtask") + "\n\n" +
		r.SubtaskInput.View() + "\n\n" +
		styles.HelpDesc.Render("Enter: create • Esc: cancel")

	return r.renderCenteredDialog(content, 60)
}

// renderSectionDialog renders the new section dialog.
func (r *Renderer) renderSectionDialog() string {
	content := styles.Title.Render("📂 New Section") + "\n\n" +
		r.SectionInput.View() + "\n\n" +
		styles.HelpDesc.Render("Enter: create • Esc: cancel")

	return r.renderCenteredDialog(content, 50)
}

// renderSectionEditDialog renders the edit section dialog.
func (r *Renderer) renderSectionEditDialog() string {
	content := styles.Title.Render("✏️ Edit Section") + "\n\n" +
		r.SectionInput.View() + "\n\n" +
		styles.HelpDesc.Render("Enter: save • Esc: cancel")

	return r.renderCenteredDialog(content, 50)
}

// renderSectionDeleteDialog renders the section delete confirmation.
func (r *Renderer) renderSectionDeleteDialog() string {
	if r.EditingSection == nil {
		return ""
	}
	content := styles.StatusBarError.Render("⚠️ Delete Section?") + "\n\n" +
		fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", r.EditingSection.Name) +
		styles.HelpDesc.Render("This will likely delete/move tasks inside.") + "\n\n" +
		styles.HelpDesc.Render("y: confirm • n/Esc: cancel")

	return r.renderCenteredDialog(content, 50)
}

// renderMoveTaskDialog renders the move task to section dialog.
func (r *Renderer) renderMoveTaskDialog() string {
	var b strings.Builder
	b.WriteString(styles.Title.Render("➡️ Move Task to Section") + "\n\n")

	if len(r.Sections) == 0 {
		b.WriteString(styles.HelpDesc.Render("No sections in this project."))
	} else {
		for i, section := range r.Sections {
			cursor := "  "
			style := lipgloss.NewStyle()
			if i == r.MoveSectionCursor {
				cursor = "> "
				style = lipgloss.NewStyle().Foreground(styles.Highlight)
			}
			b.WriteString(cursor + style.Render(section.Name) + "\n")
		}
	}

	b.WriteString("\n" + styles.HelpDesc.Render("j/k: select • Enter: move • Esc: cancel"))

	return r.renderCenteredDialog(b.String(), 50)
}

// renderCommentDialog renders the add comment dialog.
func (r *Renderer) renderCommentDialog() string {
	content := styles.Title.Render("💬 Add Comment") + "\n\n" +
		r.CommentInput.View() + "\n\n" +
		styles.HelpDesc.Render("Enter: submit • Esc: cancel")

	return r.renderCenteredDialog(content, 60)
}
