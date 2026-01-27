package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// renderSections renders the sections management view.
func (r *Renderer) renderSections() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Manage Sections"))
	b.WriteString("\n\n")

	if len(r.Sections) == 0 {
		b.WriteString(styles.HelpDesc.Render("No sections found. Press 'a' to add one."))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpDesc.Render("Esc: back"))
		return b.String()
	}

	// Render list
	for i, section := range r.Sections {
		cursor := "  "
		style := lipgloss.NewStyle()

		if i == r.TaskCursor {
			cursor = "> "
			style = lipgloss.NewStyle().Foreground(styles.Highlight)
		}

		b.WriteString(cursor + style.Render(section.Name) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("j/k: nav • a: add • e: edit • d: delete • Esc: back"))

	return b.String()
}
