package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// renderProjectsTabContent renders content for the Projects tab (sidebar + tasks).
func (r *Renderer) renderProjectsTabContent(width, height int) string {
	sidebarWidth := 30 // Wider sidebar for full project names
	if width < 70 {
		sidebarWidth = 20
	}
	if width < 50 {
		sidebarWidth = 15
	}
	mainWidth := width - sidebarWidth - 4
	if mainWidth < 20 {
		mainWidth = 20
	}

	// Render sidebar (project list) - using component
	r.SidebarComp.SetSize(sidebarWidth, height)
	r.SidebarComp.SetItems(r.SidebarItems)
	r.SidebarComp.SetCursor(r.SidebarCursor) // Sync cursor from App state
	if r.FocusedPane == state.PaneSidebar {
		r.SidebarComp.Focus()
	} else {
		r.SidebarComp.Blur()
	}
	if r.CurrentProject != nil {
		r.SidebarComp.SetActiveProject(r.CurrentProject.ID)
	}
	sidebar := r.SidebarComp.View()

	// Render main content (tasks for selected project)
	main := r.renderProjectTaskList(mainWidth, height)

	// Enforce strict dimensions
	sidebar = lipgloss.Place(sidebarWidth, height, lipgloss.Left, lipgloss.Top, sidebar)
	main = lipgloss.Place(mainWidth, height, lipgloss.Left, lipgloss.Top, main)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", main)
}

// renderProjectTaskList renders the task list for the selected project.
func (r *Renderer) renderProjectTaskList(width, height int) string {
	var content string

	// Reserve space for borders (top + bottom = 2 lines)
	innerHeight := height - 2
	if innerHeight < 5 {
		innerHeight = 5
	}

	// Calculate inner width for the content
	innerWidth := width - styles.MainContent.GetHorizontalFrameSize()

	if r.CurrentProject == nil {
		content = styles.HelpDesc.Render("Select a project from the sidebar")
	} else {
		content = r.renderDefaultTaskList(innerWidth, innerHeight)
	}

	containerStyle := styles.MainContent
	if r.FocusedPane == state.PaneMain {
		containerStyle = styles.MainContentFocused
	}

	return containerStyle.Width(width).Height(innerHeight).Render(content)
}
