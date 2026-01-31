package state

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
)

// FormField constants for focus management
const (
	FormFieldContent = iota
	FormFieldDescription
	FormFieldDue
	FormFieldDueTime // New field
	FormFieldPriority
	FormFieldShowProject
	FormFieldLabels
	FormFieldSubmit
)

const formFieldCount = 8

// TaskForm represents the state of the task creation/editing form.
type TaskForm struct {
	Content     textinput.Model
	Description textinput.Model
	Priority    int
	DueString   textinput.Model
	DueTime     textinput.Model // New field
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

	// Dropdown Cursors
	ProjectListCursor int
	LabelListCursor   int

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
			if !f.ShowProjectList {
				if msg.String() == "enter" || msg.String() == "space" {
					f.ShowProjectList = true
					// Initialize cursor to current project if found
					for i, p := range f.AvailableProjects {
						if p.ID == f.ProjectID {
							f.ProjectListCursor = i
							break
						}
					}
					return nil
				}
			} else {
				// Project list IS open
				switch msg.String() {
				case "esc":
					f.ShowProjectList = false
					return nil
				case "up", "k":
					if f.ProjectListCursor > 0 {
						f.ProjectListCursor--
					}
				case "down", "j":
					if f.ProjectListCursor < len(f.AvailableProjects)-1 {
						f.ProjectListCursor++
					}
				case "enter", "space":
					if len(f.AvailableProjects) > 0 && f.ProjectListCursor < len(f.AvailableProjects) {
						selectedProject := f.AvailableProjects[f.ProjectListCursor]
						f.ProjectID = selectedProject.ID
						f.ProjectName = selectedProject.Name
						f.ShowProjectList = false
						return nil
					}
				}
			}
		case FormFieldLabels:
			// If dropdown is NOT open, Enter/Space opens it
			if !f.ShowLabelList {
				if msg.String() == "enter" || msg.String() == "space" {
					f.ShowLabelList = true
					return nil
				}
			} else {
				// Dropdown IS open
				switch msg.String() {
				case "esc":
					f.ShowLabelList = false
					return nil
				case "up", "k":
					if f.LabelListCursor > 0 {
						f.LabelListCursor--
					}
				case "down", "j":
					if f.LabelListCursor < len(f.AvailableLabels)-1 {
						f.LabelListCursor++
					}
				case "enter", "space":
					if len(f.AvailableLabels) > 0 && f.LabelListCursor < len(f.AvailableLabels) {
						selectedLabel := f.AvailableLabels[f.LabelListCursor]

						// Toggle selection
						foundIndex := -1
						for i, name := range f.Labels {
							if name == selectedLabel.Name {
								foundIndex = i
								break
							}
						}

						if foundIndex >= 0 {
							// Remove
							f.Labels = append(f.Labels[:foundIndex], f.Labels[foundIndex+1:]...)
						} else {
							// Add
							f.Labels = append(f.Labels, selectedLabel.Name)
						}
					}
				}
			}

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
	if f.FocusIndex == FormFieldDueTime {
		f.DueTime, cmd = f.DueTime.Update(msg)
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
	due.Placeholder = "Due date (e.g. today)"
	due.Width = 25

	dueTime := textinput.New()
	dueTime.Placeholder = "Time (e.g. 10pm)"
	dueTime.Width = 15

	return &TaskForm{
		Content:           content,
		Description:       desc,
		Priority:          1, // Default to P4 (1)
		DueString:         due,
		DueTime:           dueTime,
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
		if t.Due.Datetime != nil {
			// Check for Datetime
			dtStr := *t.Due.Datetime
			var dt time.Time
			var err error

			// Try RFC3339 first
			dt, err = time.Parse(time.RFC3339, dtStr)
			if err != nil {
				// Try straight ISO
				dt, err = time.Parse("2006-01-02T15:04:05", dtStr)
			}

			if err == nil {
				// Format time as HH:MM (24h) or similar
				f.DueTime.SetValue(dt.Local().Format("15:04"))

				// Use the ISO date for the date field to avoid duplication
				f.DueString.SetValue(t.Due.Date)
			} else {
				// Fallback
				f.DueString.SetValue(t.Due.String)
			}
		} else {
			// No specific time, just use the string (e.g. "Tomorrow")
			f.DueString.SetValue(t.Due.String)
		}
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
	dueTime := strings.TrimSpace(f.DueTime.Value())

	if dueTime != "" {
		due = due + " " + dueTime
	}

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
	dueTime := strings.TrimSpace(f.DueTime.Value())
	priority := f.Priority

	if dueTime != "" {
		due = due + " " + dueTime
	}

	req := api.UpdateTaskRequest{
		Content:     &content,
		Description: &desc,
		Priority:    &priority,
		DueString:   &due,
		Labels:      f.Labels,
	}

	if f.ProjectID != "" {
		req.ProjectID = &f.ProjectID
	}
	if f.SectionID != "" {
		req.SectionID = &f.SectionID
	}

	return req
}

// FocusedField returns the index of the focused field.
// 0: Content, 1: Description, 2: Due, 3: DueTime, 4: Priority, 5: Project, 6: Labels, 7: Submit
func (f *TaskForm) FocusedField() int {
	return f.FocusIndex
}

// Helper to set focus
func (f *TaskForm) Focus(index int) {
	f.FocusIndex = index
	f.Content.Blur()
	f.Description.Blur()
	f.DueString.Blur()
	f.DueTime.Blur()

	switch index {
	case 0:
		f.Content.Focus()
	case 1:
		f.Description.Focus()
	case 2:
		f.DueString.Focus()
	case 3:
		f.DueTime.Focus()
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
	// Account for dialog borders/padding (~10 chars)
	inputWidth := width - 10
	if inputWidth < 10 {
		inputWidth = 10
	}

	f.Content.Width = inputWidth
	f.Description.Width = inputWidth
	f.DueString.Width = 25
	f.DueTime.Width = 15
}
