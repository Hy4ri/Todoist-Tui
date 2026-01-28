package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
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
	if f.Context != "" {
		title += fmt.Sprintf(" (%s)", f.Context)
	}
	b.WriteString(styles.Title.Render(title) + "\n\n")

	// Helper for metadata bar items
	renderMetaItem := func(field int, icon, value, label string) string {
		style := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Subtle).
			Padding(0, 1).
			MarginRight(1)

		if f.FocusIndex == field {
			style = style.BorderForeground(styles.Highlight).Foreground(styles.Highlight)
		}

		content := fmt.Sprintf("%s %s", icon, value)
		if value == "" {
			content = fmt.Sprintf("%s %s", icon, label)
		}

		return style.Render(content)
	}

	// Content
	b.WriteString(styles.InputLabel.Copy().Foreground(styles.Highlight).Underline(true).Render("TASK CONTENT") + "\n")
	b.WriteString(f.Content.View() + "\n\n")

	// Description
	b.WriteString(styles.InputLabel.Render("description") + "\n")
	b.WriteString(f.Description.View() + "\n\n")

	// 1. Due Date (On its own line as requested)
	b.WriteString(styles.InputLabel.Render("due date") + "\n")
	dueStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Subtle).
		Padding(0, 1).
		Width(20) // Give it some width

	if f.FocusIndex == state.FormFieldDue {
		dueStyle = dueStyle.BorderForeground(styles.Highlight)
	}
	b.WriteString(dueStyle.Render("📅 "+f.DueString.View()) + "\n\n")

	// Metadata Bar (Priority, Project, Labels)

	// Priority
	pLabel := fmt.Sprintf("P%d", 5-f.Priority)
	pIcon := "🚩"
	pStyle := styles.GetPriorityStyle(f.Priority)
	pItemContent := pStyle.Render(fmt.Sprintf("%s %s", pIcon, pLabel))

	pContainerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Subtle).
		Padding(0, 1).
		MarginRight(1)

	if f.FocusIndex == state.FormFieldPriority {
		pContainerStyle = pContainerStyle.BorderForeground(styles.Highlight)
		pItemContent = pStyle.Render(fmt.Sprintf("◀ %s %s ▶", pIcon, pLabel))
	}
	priorityItem := pContainerStyle.Render(pItemContent)

	// Project
	projName := f.ProjectName
	if projName == "" {
		projName = "Inbox"
	}
	projectItem := renderMetaItem(state.FormFieldShowProject, "📁", projName, "Project")

	// Labels
	// Calculate label string
	labelStr := "Labels"
	if len(f.Labels) > 0 {
		labelStr = strings.Join(f.Labels, ", ")
		// If too long, truncate?
		if len(labelStr) > 20 {
			labelStr = fmt.Sprintf("%d Labels", len(f.Labels))
		}
	}
	labelItem := renderMetaItem(state.FormFieldLabels, "🏷️", labelStr, "Labels")

	// Construct the bar
	bar := lipgloss.JoinHorizontal(lipgloss.Top, priorityItem, projectItem, labelItem)
	b.WriteString(bar + "\n\n")

	// Project Selector Dropdown
	if f.ShowProjectList {
		b.WriteString(styles.Subtitle.Render("Select Project:") + "\n")
		var lines []string
		count := 0
		for _, p := range f.AvailableProjects {
			if count > 5 {
				break
			}
			cursor := "  "
			style := styles.ProjectItem
			if p.ID == f.ProjectID {
				cursor = "✓ "
				style = styles.ProjectSelected
			}
			lines = append(lines, style.Render(cursor+p.Name))
			count++
		}
		if len(f.AvailableProjects) > 5 {
			remaining := len(f.AvailableProjects) - 5
			lines = append(lines, styles.HelpDesc.Render(fmt.Sprintf("...and %d more", remaining)))
		}
		list := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(0, 1).
			Render(strings.Join(lines, "\n"))
		b.WriteString(list + "\n\n")
	}

	// Label Selector Dropdown
	if f.ShowLabelList {
		b.WriteString(styles.Subtitle.Render("Select Labels:") + "\n")
		var lines []string
		count := 0

		if len(f.AvailableLabels) == 0 {
			lines = append(lines, styles.HelpDesc.Render("No labels available"))
		} else {
			for _, l := range f.AvailableLabels {
				if count > 5 {
					break
				}
				cursor := "  "
				style := styles.LabelItem

				// Check if label is selected in f.Labels
				isSelected := false
				for _, selectedLabelName := range f.Labels {
					if selectedLabelName == l.Name {
						isSelected = true
						break
					}
				}

				if isSelected {
					cursor = "✓ "
					style = styles.LabelSelected
				}
				lines = append(lines, style.Render(cursor+l.Name))
				count++
			}
			if len(f.AvailableLabels) > 5 {
				remaining := len(f.AvailableLabels) - 5
				lines = append(lines, styles.HelpDesc.Render(fmt.Sprintf("...and %d more", remaining)))
			}
		}

		list := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(0, 1).
			Render(strings.Join(lines, "\n"))
		b.WriteString(list + "\n\n")
	}

	// Submit Button
	submitStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(styles.Highlight).
		Padding(0, 2).
		MarginTop(1)

	if f.FocusIndex == state.FormFieldSubmit {
		// Invert colors for highlight effect if we could, but highlight is adaptive.
		// Let's just use consistent highlight background.
		// If focused, maybe add a border?
		submitStyle = submitStyle.Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#FFFFFF"))
		b.WriteString(submitStyle.Render("[ Submit Task ]"))
	} else {
		// Dimmed/Different style if not focused
		b.WriteString(styles.HelpDesc.Render("[ Submit Task ]"))
	}
	b.WriteString("\n")

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
	if r.IsSelectingColor {
		return r.renderColorPicker("Project Color")
	}

	content := styles.Title.Render("📁 New Project") + "\n\n" +
		styles.InputLabel.Copy().Foreground(styles.Highlight).Underline(true).Render("PROJECT NAME") + "\n" +
		r.ProjectInput.View() + "\n\n" +
		styles.HelpDesc.Render("Enter: next • Esc: cancel")

	return r.renderCenteredDialog(content, 50)
}

// renderProjectEditDialog renders the edit project dialog.
func (r *Renderer) renderProjectEditDialog() string {
	content := styles.Title.Render("✏️ Edit Project") + "\n\n" +
		styles.InputLabel.Copy().Foreground(styles.Highlight).Underline(true).Render("PROJECT NAME") + "\n" +
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
	if r.IsSelectingColor {
		return r.renderColorPicker("Label Color")
	}

	content := styles.Title.Render("🏷️ New Label") + "\n\n" +
		r.LabelInput.View() + "\n\n" +
		styles.HelpDesc.Render("Enter: next • Esc: cancel")

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
		styles.InputLabel.Copy().Foreground(styles.Highlight).Underline(true).Render("SECTION NAME") + "\n" +
		r.SectionInput.View() + "\n\n" +
		styles.HelpDesc.Render("Enter: create • Esc: cancel")

	return r.renderCenteredDialog(content, 50)
}

// renderSectionEditDialog renders the edit section dialog.
func (r *Renderer) renderSectionEditDialog() string {
	content := styles.Title.Render("✏️ Edit Section") + "\n\n" +
		styles.InputLabel.Copy().Foreground(styles.Highlight).Underline(true).Render("SECTION NAME") + "\n" +
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

	b.WriteString(styles.InputLabel.Copy().Foreground(styles.Highlight).Underline(true).Render("SELECT DESTINATION") + "\n\n")

	if len(r.Sections) == 0 {
		b.WriteString(styles.HelpDesc.Render("No sections in this project."))
	} else {
		for i, section := range r.Sections {
			cursor := "  "
			style := lipgloss.NewStyle()
			if i == r.MoveSectionCursor {
				cursor = "✓ "
				style = lipgloss.NewStyle().Foreground(styles.Highlight)
			}
			b.WriteString(style.Render(cursor+section.Name) + "\n")
		}
	}

	b.WriteString("\n\n" + styles.HelpDesc.Render("j/k: select • Enter: move • Esc: cancel"))

	return r.renderCenteredDialog(b.String(), 50)
}

// renderCommentDialog renders the add comment dialog.
func (r *Renderer) renderCommentDialog() string {
	content := styles.Title.Render("💬 Add Comment") + "\n\n" +
		r.CommentInput.View() + "\n\n" +
		styles.HelpDesc.Render("Enter: submit • Esc: cancel")

	return r.renderCenteredDialog(content, 60)
}

// renderColorPicker renders the color selection dialog.
func (r *Renderer) renderColorPicker(title string) string {
	var b strings.Builder
	b.WriteString(styles.Title.Render(title) + "\n\n")

	// Calculate visible window
	height := 8
	start := 0
	if r.ColorCursor > height/2 {
		start = r.ColorCursor - height/2
	}
	end := start + height
	if end > len(r.AvailableColors) {
		end = len(r.AvailableColors)
		start = end - height
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		colorName := r.AvailableColors[i]
		cursor := "  "
		style := lipgloss.NewStyle()

		if i == r.ColorCursor {
			cursor = "> "
			style = lipgloss.NewStyle().Foreground(styles.Highlight).Bold(true)
		}

		// Show a preview block of the color
		hex := styles.GetColor(colorName)
		preview := lipgloss.NewStyle().Background(hex).SetString("  ").String()

		line := fmt.Sprintf("%s%s %s", cursor, preview, colorName)
		b.WriteString(style.Render(line) + "\n")
	}

	b.WriteString("\n" + styles.HelpDesc.Render("j/k: select • Enter: confirm • Esc: cancel"))

	return r.renderCenteredDialog(b.String(), 40)
}
