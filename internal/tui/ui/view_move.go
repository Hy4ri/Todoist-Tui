package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// renderMoveToProject renders the move to project modal.
func (r *Renderer) renderMoveToProject(width, height int) string {
	// Modal styling
	modalWidth := 60
	if width < modalWidth {
		modalWidth = width - 4
	}
	modalHeight := 20
	if height < modalHeight {
		modalHeight = height - 4
	}

	style := lipgloss.NewStyle().
		Width(modalWidth).
		Height(modalHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Highlight).
		Padding(1, 2)

	var b strings.Builder

	// Header
	b.WriteString(styles.Title.Render("Move to Project"))
	b.WriteString("\n\n")

	// Search Input
	b.WriteString(r.MoveProjectInput.View())
	b.WriteString("\n\n")

	// Target List
	listHeight := modalHeight - 6 // Title(2) + Input(2) + Padding(2)
	startIdx := 0
	if r.MoveTargetCursor >= listHeight {
		startIdx = r.MoveTargetCursor - listHeight + 1
	}
	endIdx := startIdx + listHeight
	if endIdx > len(r.MoveTargetList) {
		endIdx = len(r.MoveTargetList)
	}

	if len(r.MoveTargetList) == 0 {
		b.WriteString(styles.HelpDesc.Render("No matching projects found"))
	} else {
		for i := startIdx; i < endIdx; i++ {
			target := r.MoveTargetList[i]
			cursor := "  "
			itemStyle := lipgloss.NewStyle()

			if i == r.MoveTargetCursor {
				cursor = "> "
				itemStyle = itemStyle.Foreground(styles.Highlight)
			}

			prefix := ""
			if target.IsSection {
				prefix = "  └ "
			} else {
				prefix = "# "
			}

			line := fmt.Sprintf("%s%s%s", cursor, prefix, target.Name)
			b.WriteString(itemStyle.Render(line))
			b.WriteString("\n")
		}
	}

	// Center modal on screen?
	// The View() method in view.go typically handles whole screen.
	// We can use Place() to center it.
	content := style.Render(b.String())
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)
}
