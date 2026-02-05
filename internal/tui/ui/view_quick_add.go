package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// renderQuickAdd renders the Quick Add overlay dialog.
func (r *Renderer) renderQuickAdd() string {
	if r.QuickAddForm == nil {
		return ""
	}

	// Dialog dimensions
	dialogWidth := 70
	if r.Width < 80 {
		dialogWidth = r.Width - 10
	}
	if dialogWidth < 50 {
		dialogWidth = 50
	}

	r.QuickAddForm.SetWidth(dialogWidth)

	// Build dialog content
	var content strings.Builder

	// Title
	title := styles.DialogTitle.Render("⚡ Quick Add")
	content.WriteString(title + "\n\n")

	// Context hint (if in a project/section)
	if r.QuickAddForm.ProjectName != "" {
		ctx := fmt.Sprintf("📂 %s", r.QuickAddForm.ProjectName)
		if r.QuickAddForm.SectionName != "" {
			ctx += fmt.Sprintf(" / %s", r.QuickAddForm.SectionName)
		}
		contextStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
		content.WriteString(contextStyle.Render(ctx) + "\n\n")
	}

	// Input field
	content.WriteString(r.QuickAddForm.Input.View() + "\n\n")

	// Help text
	helpStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	helpText := helpStyle.Render("Natural language: 'Buy milk tomorrow 3pm @errands p1'")
	content.WriteString(helpText + "\n")

	// Status (if tasks were added)
	if r.QuickAddForm.TaskCount > 0 {
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
		status := statusStyle.Render(fmt.Sprintf("✓ %d task(s) added", r.QuickAddForm.TaskCount))
		content.WriteString(status + "\n")
	}

	// Footer
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).MarginTop(1)
	footer := footerStyle.Render("Enter: Add task  •  Esc: Close")
	content.WriteString("\n" + footer)

	// Create dialog box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("5")).
		Padding(1, 2).
		Width(dialogWidth)

	dialog := dialogStyle.Render(content.String())

	// Center the dialog on screen
	return lipgloss.Place(
		r.Width,
		r.Height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}
