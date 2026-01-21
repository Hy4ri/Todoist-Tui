// Package tui provides the terminal user interface for Todoist.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// FormField represents which field is currently focused in the form.
type FormField int

const (
	FormFieldContent FormField = iota
	FormFieldDescription
	FormFieldDue
	FormFieldPriority
	FormFieldProject
	FormFieldSubmit
)

const formFieldCount = 6

// TaskForm manages the state of the add/edit task form.
type TaskForm struct {
	// Mode
	Mode   string // "add" or "edit"
	TaskID string // For edit mode

	// Inputs
	ContentInput     textinput.Model
	DescriptionInput textinput.Model
	DueInput         textinput.Model

	// Selections
	Priority        int    // 1-4 (Todoist priority, 4=highest)
	ProjectID       string // Selected project ID
	ProjectName     string // Selected project name (for display)
	projectCursor   int    // For project selection dropdown
	showProjectList bool   // Whether project dropdown is open

	// Form state
	FocusedField FormField
	Projects     []api.Project
	width        int
}

// NewTaskForm creates a new task form for adding a task.
func NewTaskForm(projects []api.Project) *TaskForm {
	contentInput := textinput.New()
	contentInput.Placeholder = "Task name"
	contentInput.Focus()
	contentInput.CharLimit = 500
	contentInput.Width = 50

	descInput := textinput.New()
	descInput.Placeholder = "Description (optional)"
	descInput.CharLimit = 1000
	descInput.Width = 50

	dueInput := textinput.New()
	dueInput.Placeholder = "Due date (e.g., tomorrow, next monday)"
	dueInput.CharLimit = 100
	dueInput.Width = 50

	// Find inbox project as default
	var inboxID, inboxName string
	for _, p := range projects {
		if p.IsInboxProject {
			inboxID = p.ID
			inboxName = p.Name
			break
		}
	}

	return &TaskForm{
		Mode:             "add",
		ContentInput:     contentInput,
		DescriptionInput: descInput,
		DueInput:         dueInput,
		Priority:         1, // Default to P4 (lowest)
		ProjectID:        inboxID,
		ProjectName:      inboxName,
		FocusedField:     FormFieldContent,
		Projects:         projects,
	}
}

// NewEditTaskForm creates a new task form pre-populated for editing.
func NewEditTaskForm(task *api.Task, projects []api.Project) *TaskForm {
	form := NewTaskForm(projects)
	form.Mode = "edit"
	form.TaskID = task.ID

	// Pre-populate fields
	form.ContentInput.SetValue(task.Content)
	form.DescriptionInput.SetValue(task.Description)
	if task.Due != nil {
		form.DueInput.SetValue(task.Due.String)
	}
	form.Priority = task.Priority

	// Set project
	form.ProjectID = task.ProjectID
	for _, p := range projects {
		if p.ID == task.ProjectID {
			form.ProjectName = p.Name
			break
		}
	}

	return form
}

// SetWidth sets the form width for responsive layout.
func (f *TaskForm) SetWidth(width int) {
	f.width = width
	inputWidth := width - 10
	if inputWidth < 30 {
		inputWidth = 30
	}
	if inputWidth > 60 {
		inputWidth = 60
	}
	f.ContentInput.Width = inputWidth
	f.DescriptionInput.Width = inputWidth
	f.DueInput.Width = inputWidth
}

// Update handles input for the form.
func (f *TaskForm) Update(msg tea.Msg) (*TaskForm, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle project dropdown if open
		if f.showProjectList {
			return f.handleProjectListKey(msg)
		}

		switch msg.String() {
		case "tab":
			f.nextField()
			return f, nil
		case "shift+tab":
			f.prevField()
			return f, nil
		case "ctrl+p":
			// Toggle project dropdown
			if f.FocusedField == FormFieldProject {
				f.showProjectList = !f.showProjectList
			}
			return f, nil
		case "enter":
			if f.FocusedField == FormFieldProject {
				f.showProjectList = true
				return f, nil
			}
			// Submit is handled by the parent
		}

		// Handle priority keys when on priority field
		if f.FocusedField == FormFieldPriority {
			switch msg.String() {
			case "1":
				f.Priority = 4 // P1 = highest = Todoist 4
				return f, nil
			case "2":
				f.Priority = 3
				return f, nil
			case "3":
				f.Priority = 2
				return f, nil
			case "4":
				f.Priority = 1
				return f, nil
			case "h", "left":
				if f.Priority < 4 {
					f.Priority++
				}
				return f, nil
			case "l", "right":
				if f.Priority > 1 {
					f.Priority--
				}
				return f, nil
			}
		}
	}

	// Update focused text input
	switch f.FocusedField {
	case FormFieldContent:
		var cmd tea.Cmd
		f.ContentInput, cmd = f.ContentInput.Update(msg)
		cmds = append(cmds, cmd)
	case FormFieldDescription:
		var cmd tea.Cmd
		f.DescriptionInput, cmd = f.DescriptionInput.Update(msg)
		cmds = append(cmds, cmd)
	case FormFieldDue:
		var cmd tea.Cmd
		f.DueInput, cmd = f.DueInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return f, tea.Batch(cmds...)
}

// handleProjectListKey handles key input when project dropdown is open.
func (f *TaskForm) handleProjectListKey(msg tea.KeyMsg) (*TaskForm, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if f.projectCursor < len(f.Projects)-1 {
			f.projectCursor++
		}
	case "k", "up":
		if f.projectCursor > 0 {
			f.projectCursor--
		}
	case "enter":
		if f.projectCursor < len(f.Projects) {
			p := f.Projects[f.projectCursor]
			f.ProjectID = p.ID
			f.ProjectName = p.Name
		}
		f.showProjectList = false
	case "esc", "q":
		f.showProjectList = false
	}
	return f, nil
}

// nextField moves focus to the next field.
func (f *TaskForm) nextField() {
	f.blurCurrent()
	f.FocusedField = (f.FocusedField + 1) % formFieldCount
	f.focusCurrent()
}

// prevField moves focus to the previous field.
func (f *TaskForm) prevField() {
	f.blurCurrent()
	f.FocusedField = (f.FocusedField - 1 + formFieldCount) % formFieldCount
	f.focusCurrent()
}

// blurCurrent blurs the currently focused input.
func (f *TaskForm) blurCurrent() {
	switch f.FocusedField {
	case FormFieldContent:
		f.ContentInput.Blur()
	case FormFieldDescription:
		f.DescriptionInput.Blur()
	case FormFieldDue:
		f.DueInput.Blur()
	}
}

// focusCurrent focuses the current field.
func (f *TaskForm) focusCurrent() {
	switch f.FocusedField {
	case FormFieldContent:
		f.ContentInput.Focus()
	case FormFieldDescription:
		f.DescriptionInput.Focus()
	case FormFieldDue:
		f.DueInput.Focus()
	}
}

// IsValid returns true if the form has valid data for submission.
func (f *TaskForm) IsValid() bool {
	return strings.TrimSpace(f.ContentInput.Value()) != ""
}

// ToCreateRequest converts the form to a CreateTaskRequest.
func (f *TaskForm) ToCreateRequest() api.CreateTaskRequest {
	req := api.CreateTaskRequest{
		Content:   strings.TrimSpace(f.ContentInput.Value()),
		ProjectID: f.ProjectID,
		Priority:  f.Priority,
	}

	if desc := strings.TrimSpace(f.DescriptionInput.Value()); desc != "" {
		req.Description = desc
	}

	if due := strings.TrimSpace(f.DueInput.Value()); due != "" {
		req.DueString = due
	}

	return req
}

// ToUpdateRequest converts the form to an UpdateTaskRequest.
func (f *TaskForm) ToUpdateRequest() api.UpdateTaskRequest {
	content := strings.TrimSpace(f.ContentInput.Value())
	desc := strings.TrimSpace(f.DescriptionInput.Value())
	due := strings.TrimSpace(f.DueInput.Value())
	priority := f.Priority

	req := api.UpdateTaskRequest{
		Content:     &content,
		Description: &desc,
		Priority:    &priority,
	}

	if due != "" {
		req.DueString = &due
	}

	return req
}

// View renders the form.
func (f *TaskForm) View() string {
	var b strings.Builder

	// Title
	title := "Add Task"
	if f.Mode == "edit" {
		title = "Edit Task"
	}
	b.WriteString(styles.DialogTitle.Render(title))
	b.WriteString("\n\n")

	// Content field
	b.WriteString(f.renderField("Task Name", f.ContentInput.View(), FormFieldContent))
	b.WriteString("\n")

	// Description field
	b.WriteString(f.renderField("Description", f.DescriptionInput.View(), FormFieldDescription))
	b.WriteString("\n")

	// Due date field
	b.WriteString(f.renderField("Due Date", f.DueInput.View(), FormFieldDue))
	b.WriteString("\n")

	// Priority field
	b.WriteString(f.renderPriorityField())
	b.WriteString("\n")

	// Project field
	b.WriteString(f.renderProjectField())
	b.WriteString("\n\n")

	// Submit button
	submitStyle := styles.HelpDesc
	if f.FocusedField == FormFieldSubmit {
		submitStyle = styles.HelpKey
	}
	submitText := "[ Submit ]"
	if f.Mode == "edit" {
		submitText = "[ Save Changes ]"
	}
	b.WriteString(submitStyle.Render(submitText))
	b.WriteString("\n\n")

	// Help
	helpText := "Tab: next field | Shift+Tab: previous | Enter: submit | Esc: cancel"
	if f.FocusedField == FormFieldPriority {
		helpText = "1-4: set priority | h/l: adjust | Tab: next field"
	} else if f.FocusedField == FormFieldProject {
		helpText = "Enter: open list | Tab: next field"
	}
	b.WriteString(styles.HelpDesc.Render(helpText))

	return b.String()
}

// renderField renders a form field with label.
func (f *TaskForm) renderField(label, input string, field FormField) string {
	labelStyle := styles.InputLabel
	if f.FocusedField == field {
		labelStyle = labelStyle.Foreground(styles.Highlight)
	}

	return fmt.Sprintf("%s\n%s", labelStyle.Render(label), input)
}

// renderPriorityField renders the priority selector.
func (f *TaskForm) renderPriorityField() string {
	labelStyle := styles.InputLabel
	if f.FocusedField == FormFieldPriority {
		labelStyle = labelStyle.Foreground(styles.Highlight)
	}

	var priorities []string
	for i := 4; i >= 1; i-- {
		pLabel := fmt.Sprintf("P%d", 5-i) // Display as P1-P4
		style := styles.GetPriorityStyle(i)

		if i == f.Priority {
			// Selected priority
			style = style.Bold(true).Underline(true)
		}

		priorities = append(priorities, style.Render(pLabel))
	}

	selector := strings.Join(priorities, "  ")
	if f.FocusedField == FormFieldPriority {
		selector = "[ " + selector + " ]"
	}

	return fmt.Sprintf("%s\n%s", labelStyle.Render("Priority"), selector)
}

// renderProjectField renders the project selector.
func (f *TaskForm) renderProjectField() string {
	labelStyle := styles.InputLabel
	if f.FocusedField == FormFieldProject {
		labelStyle = labelStyle.Foreground(styles.Highlight)
	}

	var content string
	if f.showProjectList {
		// Show project dropdown
		var lines []string
		for i, p := range f.Projects {
			cursor := "  "
			style := styles.ProjectItem
			if i == f.projectCursor {
				cursor = "> "
				style = styles.ProjectSelected
			}

			name := p.Name
			if p.IsInboxProject {
				name = "Inbox"
			}
			lines = append(lines, style.Render(cursor+name))
		}
		content = strings.Join(lines, "\n")
		content = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(0, 1).
			Render(content)
	} else {
		projectDisplay := f.ProjectName
		if projectDisplay == "" {
			projectDisplay = "Select project..."
		}
		if f.FocusedField == FormFieldProject {
			content = "[ " + projectDisplay + " ] (press Enter to change)"
		} else {
			content = projectDisplay
		}
	}

	return fmt.Sprintf("%s\n%s", labelStyle.Render("Project"), content)
}
