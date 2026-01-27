package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// DetailModel displays task details in a side panel or full view.
type DetailModel struct {
	task          *api.Task
	comments      []api.Comment
	projects      []api.Project
	width, height int
	showPanel     bool
	focused       bool
}

// NewDetail creates a new DetailModel.
func NewDetail() *DetailModel {
	return &DetailModel{
		task:      nil,
		comments:  []api.Comment{},
		projects:  []api.Project{},
		showPanel: false,
	}
}

// Init implements Component.
func (d *DetailModel) Init() tea.Cmd {
	return nil
}

// Update implements Component.
func (d *DetailModel) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			d.showPanel = false
			d.task = nil
			d.comments = nil
		}
	}
	return d, nil
}

// View implements Component.
func (d *DetailModel) View() string {
	if d.task == nil {
		return ""
	}
	return d.renderPanel()
}

// ViewPanel renders the detail panel for split view.
func (d *DetailModel) ViewPanel() string {
	if d.task == nil {
		return ""
	}

	t := d.task

	// Use standardized style
	containerStyle := styles.DetailPanel
	if d.focused {
		containerStyle = styles.DetailPanelFocused
	}

	panelStyle := containerStyle.
		Width(d.width).
		Height(d.height - 2)

	// Build content
	var content strings.Builder

	// Title
	content.WriteString(styles.Title.Render(t.Content) + "\n\n")

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
	if len(d.comments) > 0 {
		content.WriteString("\n" + styles.StatusBarKey.Render(fmt.Sprintf("Comments (%d):", len(d.comments))) + "\n")
		for _, c := range d.comments {
			content.WriteString("• " + c.Content + "\n")
		}
	}

	// Help
	content.WriteString("\n" + styles.HelpDesc.Render("Esc to close"))

	return panelStyle.Render(content.String())
}

// renderPanel renders the full-width detail view.
func (d *DetailModel) renderPanel() string {
	if d.task == nil {
		return "No task selected"
	}

	t := d.task
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
		for _, p := range d.projects {
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
	if len(d.comments) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.DetailSection.Render("  " + strings.Repeat("─", 40)))
		b.WriteString("\n")
		b.WriteString(styles.Subtitle.Render("  Comments"))
		b.WriteString("\n\n")

		for _, c := range d.comments {
			// Parse and format timestamp - using simpler formatting as time import is not added, or rely on string
			// NOTE: view_details used time.Parse, but I don't want to add time import if not needed.
			// Let's just use the string for now or use basic formatting if available.
			// view_details used time import. I should check if time is imported. It is NOT imported in detail.go currently.
			// I'll stick to string or add time import.
			// Wait, I should add time import if I want to keep exact parity.
			// Checking imports again...
			// "fmt", "strings", "tea", "api", "styles" are currently imports.
			// I'll use the string directly to avoid adding 'time' import for now unless needed.
			// Actually, view_details:
			// if t, err := time.Parse(time.RFC3339, c.PostedAt); err == nil { ... }
			// Providing a cleaner string is better. I'll add "time" to imports.
			b.WriteString(styles.CommentAuthor.Render(fmt.Sprintf("    %s", c.PostedAt)))
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

	return styles.Dialog.Width(d.width - 4).Render(b.String())
}

// SetSize implements Component.
func (d *DetailModel) SetSize(width, height int) {
	d.width = width
	d.height = height
}

// SetTask sets the task to display.
func (d *DetailModel) SetTask(task *api.Task) {
	d.task = task
	d.showPanel = task != nil
}

// SetComments sets the comments for the task.
func (d *DetailModel) SetComments(comments []api.Comment) {
	d.comments = comments
}

// SetProjects sets the projects for lookup.
func (d *DetailModel) SetProjects(projects []api.Project) {
	d.projects = projects
}

// Task returns the current task.
func (d *DetailModel) Task() *api.Task {
	return d.task
}

// ShowPanel returns whether the panel should be shown.
func (d *DetailModel) ShowPanel() bool {
	return d.showPanel
}

// Hide hides the detail panel.
func (d *DetailModel) Hide() {
	d.showPanel = false
	d.task = nil
	d.comments = nil
}

// Focus sets focus.
func (d *DetailModel) Focus() {
	d.focused = true
}

// Blur removes focus.
func (d *DetailModel) Blur() {
	d.focused = false
}
