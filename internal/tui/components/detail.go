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
	CommentCursor int
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
			d.CommentCursor = 0
		case "j", "down":
			if d.focused && len(d.comments) > 0 {
				if d.CommentCursor < len(d.comments)-1 {
					d.CommentCursor++
				}
			}
		case "k", "up":
			if d.focused && len(d.comments) > 0 {
				if d.CommentCursor > 0 {
					d.CommentCursor--
				}
			}
		case "e":
			if d.focused && len(d.comments) > 0 {
				c := d.comments[d.CommentCursor]
				return d, func() tea.Msg {
					return EditCommentMsg{Comment: &c}
				}
			}
		case "d":
			if d.focused && len(d.comments) > 0 {
				c := d.comments[d.CommentCursor]
				return d, func() tea.Msg {
					return DeleteCommentMsg{CommentID: c.ID}
				}
			}
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
		// Calculate available width: width - 2 (border) - 2 (padding) - 2 (extra safety)
		descWidth := d.width - 6
		if descWidth < 10 {
			descWidth = 10
		}
		content.WriteString(styles.DetailDescription.Copy().Width(descWidth).Render(t.Description) + "\n")
	}

	// Comments
	if len(d.comments) > 0 {
		content.WriteString("\n" + styles.StatusBarKey.Render(fmt.Sprintf("Comments (%d):", len(d.comments))) + "\n")
		// Calculate wrap width for comments
		commentWidth := d.width - 6
		if commentWidth < 10 {
			commentWidth = 10
		}
		for _, c := range d.comments {
			content.WriteString("• " + styles.CommentContent.Copy().Width(commentWidth).Render(c.Content) + "\n")
		}
	}

	// Help
	content.WriteString("\n" + styles.HelpDesc.Render("Esc to close, C to comment"))

	return panelStyle.Render(content.String())
}

// renderPanel renders the full-width detail view.
func (d *DetailModel) renderPanel() string {
	if d.task == nil {
		return "No task selected"
	}

	t := d.task
	var b strings.Builder

	// Calculate content width
	// Dialog width: d.width - 4 (set in return)
	// Dialog padding: 2 left + 2 right = 4
	// Total chrome: 8 (approx)
	contentWidth := d.width - 10
	if contentWidth < 20 {
		contentWidth = 20
	}

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
	b.WriteString(styles.DetailSection.Render("  " + strings.Repeat("─", contentWidth)))
	b.WriteString("\n\n")

	// Description (if present)
	if t.Description != "" {
		b.WriteString(styles.DetailIcon.Render("  📝"))
		b.WriteString(styles.DetailLabel.Render("Description"))
		b.WriteString("\n")
		// Apply wrapping to description
		// DetailDescription has PaddingLeft(2), so subtract that from contentWidth
		b.WriteString(styles.DetailDescription.Copy().Width(contentWidth - 2).Render(t.Description))
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
		b.WriteString(styles.DetailSection.Render("  " + strings.Repeat("─", contentWidth)))
		b.WriteString("\n")
		b.WriteString(styles.Subtitle.Render("  Comments"))
		b.WriteString("\n\n")

		for i, c := range d.comments {
			cursor := "  "
			if d.focused && i == d.CommentCursor {
				cursor = "> "
			}

			// Parse and format timestamp
			b.WriteString(styles.CommentAuthor.Render(fmt.Sprintf("%s  %s", cursor, c.PostedAt)))
			b.WriteString("\n")

			// Apply wrapping to comment content
			b.WriteString(styles.CommentContent.Copy().Width(contentWidth - 2).Render(c.Content))

			// Render attachment if present
			if c.FileAttachment != nil {
				icon := "📎"
				if strings.HasPrefix(c.FileAttachment.FileType, "image/") {
					icon = "🖼️"
				}
				link := fmt.Sprintf("%s [%s](%s)", icon, c.FileAttachment.FileName, c.FileAttachment.FileURL)
				b.WriteString(styles.CommentContent.Render(fmt.Sprintf("    %s", link)))
				b.WriteString("\n")
			}
			b.WriteString("\n")
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
	b.WriteString(styles.HelpKey.Render("j/k"))
	b.WriteString(styles.HelpDesc.Render(" nav comments  "))
	b.WriteString(styles.HelpKey.Render("e"))
	b.WriteString(styles.HelpDesc.Render(" edit  "))
	b.WriteString(styles.HelpKey.Render("d"))
	b.WriteString(styles.HelpDesc.Render(" delete  "))
	b.WriteString(styles.HelpKey.Render("s"))
	b.WriteString(styles.HelpDesc.Render(" add subtask  "))
	b.WriteString(styles.HelpKey.Render("C"))
	b.WriteString(styles.HelpDesc.Render(" comment"))

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
	if d.CommentCursor >= len(d.comments) {
		d.CommentCursor = len(d.comments) - 1
		if d.CommentCursor < 0 {
			d.CommentCursor = 0
		}
	}
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
