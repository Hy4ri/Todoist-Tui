package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// DetailModel displays task details in a side panel or full view.
type DetailModel struct {
	task          *api.Task
	comments      []api.Comment
	width, height int
	showPanel     bool
}

// NewDetail creates a new DetailModel.
func NewDetail() *DetailModel {
	return &DetailModel{
		task:      nil,
		comments:  []api.Comment{},
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

	// Create border style
	panelStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Highlight).
		Padding(0, 1).
		Width(d.width - 2).
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

	// Title with checkbox
	checkbox := "[ ]"
	if t.Checked {
		checkbox = "[x]"
	}
	b.WriteString(styles.Title.Render(fmt.Sprintf("%s %s", checkbox, t.Content)))
	b.WriteString("\n\n")

	// Due date
	if t.Due != nil {
		b.WriteString(styles.StatusBarKey.Render("📅 Due: "))
		b.WriteString(t.Due.String)
		b.WriteString("\n")
	}

	// Priority
	priorityStyle := styles.GetPriorityStyle(t.Priority)
	priorityLabel := fmt.Sprintf("P%d", 5-t.Priority)
	b.WriteString(styles.StatusBarKey.Render("🔴 Priority: "))
	b.WriteString(priorityStyle.Render(priorityLabel))
	b.WriteString("\n")

	// Labels
	if len(t.Labels) > 0 {
		b.WriteString(styles.StatusBarKey.Render("🏷️ Labels: "))
		for i, l := range t.Labels {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(styles.TaskLabel.Render("@" + l))
		}
		b.WriteString("\n")
	}

	// Description
	if t.Description != "" {
		b.WriteString("\n")
		b.WriteString(styles.StatusBarKey.Render("📝 Description"))
		b.WriteString("\n")
		b.WriteString(t.Description)
		b.WriteString("\n")
	}

	// Comments
	if len(d.comments) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.StatusBarKey.Render(fmt.Sprintf("💬 Comments (%d)", len(d.comments))))
		b.WriteString("\n")
		for _, c := range d.comments {
			b.WriteString("  • ")
			b.WriteString(c.Content)
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("ESC: back | e: edit | x: complete | dd: delete | c: comment"))

	return b.String()
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
