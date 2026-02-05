package state

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// QuickAddForm represents a lightweight form for quick task creation.
// Uses natural language parsing via the Todoist API's due_string parameter.
type QuickAddForm struct {
	Input textinput.Model

	// Context from current view (inherited when opening)
	ProjectID   string
	ProjectName string
	SectionID   string
	SectionName string

	// Status tracking
	LastStatus string
	TaskCount  int
}

// NewQuickAddForm creates a new quick add form.
func NewQuickAddForm() *QuickAddForm {
	input := textinput.New()
	input.Placeholder = "e.g. Buy milk tomorrow 3pm @errands #Shopping p1"
	input.Focus()
	input.CharLimit = 500
	input.Width = 60

	return &QuickAddForm{
		Input: input,
	}
}

// Update handles input events for the form.
func (f *QuickAddForm) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	f.Input, cmd = f.Input.Update(msg)
	return cmd
}

// Value returns the current input value.
func (f *QuickAddForm) Value() string {
	return strings.TrimSpace(f.Input.Value())
}

// IsValid returns true if there is content to submit.
func (f *QuickAddForm) IsValid() bool {
	return f.Value() != ""
}

// Clear resets the input field for the next task.
func (f *QuickAddForm) Clear() {
	f.Input.SetValue("")
}

// SetWidth sets the width of the input field.
func (f *QuickAddForm) SetWidth(width int) {
	// Account for dialog borders/padding
	inputWidth := width - 10
	if inputWidth < 40 {
		inputWidth = 40
	}
	if inputWidth > 80 {
		inputWidth = 80
	}
	f.Input.Width = inputWidth
}

// SetContext sets the project/section context from the current view.
func (f *QuickAddForm) SetContext(projectID, projectName, sectionID, sectionName string) {
	f.ProjectID = projectID
	f.ProjectName = projectName
	f.SectionID = sectionID
	f.SectionName = sectionName
}

// IncrementCount increments the task count and updates status.
func (f *QuickAddForm) IncrementCount() {
	f.TaskCount++
}

// ResetCount resets the task counter.
func (f *QuickAddForm) ResetCount() {
	f.TaskCount = 0
}
