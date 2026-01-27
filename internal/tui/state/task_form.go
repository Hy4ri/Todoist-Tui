package state

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
)

// FormField constants for focus management
const (
	FormFieldContent = iota
	FormFieldDescription
	FormFieldDue
	FormFieldPriority
	FormFieldShowProject
	FormFieldLabels
	FormFieldSubmit
)

const formFieldCount = 7

// TaskForm represents the state of the task creation/editing form.
type TaskForm struct {
	Content     textinput.Model
	Description textinput.Model
	Priority    int
	DueString   textinput.Model
	ProjectID   string
	SectionID   string
	Labels      []string
	Original    *api.Task

	// Helpers for logic/ui
	ShowProjectList bool
	ShowLabelList   bool
	FocusIndex      int
	ProjectName     string
	SectionName     string
	Context         string

	// Mode tracking
	Mode   string // "create" or "edit"
	TaskID string // ID of task being edited

	// Data for completion/selection
	AvailableProjects []api.Project
	AvailableLabels   []api.Label
}

// Update updates the form models.
func (f *TaskForm) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "down":
			f.NextField()
			return nil
		case "shift+tab", "up":
			f.PrevField()
			return nil
		}

		// Handle specific field input
		switch f.FocusIndex {
		case FormFieldPriority:
			switch msg.String() {
			case "1":
				f.Priority = 4 // P1
			case "2":
				f.Priority = 3
			case "3":
				f.Priority = 2
			case "4":
				f.Priority = 1
			case "h", "left":
				if f.Priority < 4 {
					f.Priority++
				}
			case "l", "right":
				if f.Priority > 1 {
					f.Priority--
				}
			}
			return nil
		case FormFieldShowProject:
			if f.ShowProjectList {
				// Handle project list navigation if implemented here
				// For now, project list navigation is handled by parent or logic
				// We just handle opening/closing
				if msg.String() == "enter" || msg.String() == "space" {
					f.ShowProjectList = !f.ShowProjectList
				}
			} else {
				if msg.String() == "enter" || msg.String() == "space" {
					f.ShowProjectList = true
				}
			}
			// Don't return here, let parent handle project selection if needed
		case FormFieldLabels:
			// Toggle dropdown on Enter/Space
			if msg.String() == "enter" || msg.String() == "space" {
				f.ShowLabelList = !f.ShowLabelList
				return nil
			}

			// If dropdown is open, handle selection with keys?
			// Actually, navigation happens in wrapper logic?
			// Wait, f.Labels is a list of strings. We need a way to SELECT one.
			// The current state doesn't have a "Cursor" for the label list.
			// We need to add `LabelListCursor int` to struct to support selection.
			// For now, let's just allow opening/closing.
			// Real selection logic requires tracking a cursor inside the dropdown.

		}
	}

	// Only update text inputs if focused
	if f.FocusIndex == FormFieldContent {
		f.Content, cmd = f.Content.Update(msg)
		cmds = append(cmds, cmd)
	}
	if f.FocusIndex == FormFieldDescription {
		f.Description, cmd = f.Description.Update(msg)
		cmds = append(cmds, cmd)
	}
	if f.FocusIndex == FormFieldDue {
		f.DueString, cmd = f.DueString.Update(msg)
		cmds = append(cmds, cmd)
	}

	return tea.Batch(cmds...)
}

// NextField moves focus to the next field.
func (f *TaskForm) NextField() {
	f.Focus((f.FocusIndex + 1) % formFieldCount)
}

// PrevField moves focus to the previous field.
func (f *TaskForm) PrevField() {
	f.Focus((f.FocusIndex - 1 + formFieldCount) % formFieldCount)
}

// NewTaskForm creates a new task form.
func NewTaskForm(projects []api.Project, labels []api.Label) *TaskForm {
	content := textinput.New()
	content.Placeholder = "Task content"
	content.Focus()
	content.CharLimit = 500
	content.Width = 50

	desc := textinput.New()
	desc.Placeholder = "Description"
	desc.CharLimit = 1000
	desc.Width = 50

	due := textinput.New()
	due.Placeholder = "Due date (e.g. today, tomorrow)"
	due.Width = 20

	return &TaskForm{
		Content:           content,
		Description:       desc,
		Priority:          1, // Default to P4 (1)
		DueString:         due,
		Labels:            []string{},
		ShowProjectList:   false,
		AvailableProjects: projects,
		AvailableLabels:   labels,
	}
}

func NewEditTaskForm(t *api.Task, projects []api.Project, labels []api.Label) *TaskForm {
	f := NewTaskForm(projects, labels)
	f.Original = t
	f.Content.SetValue(t.Content)
	f.Description.SetValue(t.Description)
	f.Priority = t.Priority
	if t.Due != nil {
		f.DueString.SetValue(t.Due.String)
	}
	f.ProjectID = t.ProjectID
	f.SectionID = "" // need to lookup
	if t.SectionID != nil {
		f.SectionID = *t.SectionID
	}
	f.Labels = t.Labels

	// Find project name
	for _, p := range projects {
		if p.ID == t.ProjectID {
			f.ProjectName = p.Name
			break
		}
	}

	f.Mode = "edit"
	f.TaskID = t.ID

	return f
}

// IsValid checks if the form is valid.
func (f *TaskForm) IsValid() bool {
	return strings.TrimSpace(f.Content.Value()) != ""
}

// ToCreateRequest converts form to create request.
func (f *TaskForm) ToCreateRequest() api.CreateTaskRequest {
	content := strings.TrimSpace(f.Content.Value())
	desc := strings.TrimSpace(f.Description.Value())
	due := strings.TrimSpace(f.DueString.Value())

	// Priority mapping: UI 1-4 -> API 4-1
	// UI P1 (Red) = 4, UI P2 (Orange) = 3, UI P3 (Blue) = 2, UI P4 (Grey) = 1
	// Our internal Priority seems to match Todoist API directly?
	// In view.go: priorityLabel := fmt.Sprintf("P%d", 5-t.Priority).
	// If t.Priority is 4 (High), label is P1.
	// So internal priority 4 = High.
	// API AddParams uses Priority.

	return api.CreateTaskRequest{
		Content:     content,
		Description: desc,
		Priority:    f.Priority,
		DueString:   due,
		ProjectID:   f.ProjectID,
		SectionID:   f.SectionID,
		Labels:      f.Labels,
	}
}

// ToUpdateRequest converts form to update request.
func (f *TaskForm) ToUpdateRequest() api.UpdateTaskRequest {
	content := strings.TrimSpace(f.Content.Value())
	desc := strings.TrimSpace(f.Description.Value())
	due := strings.TrimSpace(f.DueString.Value())
	priority := f.Priority

	return api.UpdateTaskRequest{
		Content:     &content,
		Description: &desc,
		Priority:    &priority,
		DueString:   &due,
		Labels:      f.Labels,
	}
}

// FocusedField returns the index of the focused field.
// 0: Content, 1: Description, 2: Due, 3: Project, 4: Labels, 5: Submit
// This logic was likely in Update but we can expose a helper.
// Actually, `update_projects.go` uses `FocusedField`.
// It likely expects an int. I need to implement logic to determine which is focused.
// But `TaskForm` in state doesn't track "FocusedFieldIndex".
// It tracks it via textinput.Focused()?
func (f *TaskForm) FocusedField() int {
	if f.Content.Focused() {
		return 0
	}
	if f.Description.Focused() {
		return 1
	}
	if f.DueString.Focused() {
		return 2
	}
	// How do we track Project/Labels/Submit focus if they aren't textinputs?
	// We need a `FocusIndex` field in TaskForm struct!
	return f.FocusIndex
}

// Helper to set focus
func (f *TaskForm) Focus(index int) {
	f.FocusIndex = index
	f.Content.Blur()
	f.Description.Blur()
	f.DueString.Blur()

	switch index {
	case 0:
		f.Content.Focus()
	case 1:
		f.Description.Focus()
	case 2:
		f.DueString.Focus()
	}
}

// SetDue sets the due date string.
func (f *TaskForm) SetDue(due string) {
	f.DueString.SetValue(due)
}

// SetProjectSection sets the project and section IDs.
func (f *TaskForm) SetProjectSection(projectID, sectionID string) {
	f.ProjectID = projectID
	f.SectionID = sectionID
}

// SetContext sets the context label (e.g. "Today").
func (f *TaskForm) SetContext(context string) {
	f.Context = context
}

// SetWidth sets width of inputs
func (f *TaskForm) SetWidth(width int) {
	f.Content.Width = width
	f.Description.Width = width
	f.DueString.Width = width
}
