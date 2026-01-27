package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

type Renderer struct {
	*state.State
}

func NewRenderer(s *state.State) *Renderer {
	return &Renderer{State: s}
}

func (r *Renderer) View() string {
	if r.Width == 0 {
		return "Loading..."
	}

	var content string
	switch r.CurrentView {
	case state.ViewHelp:
		content = r.HelpComp.View()
	case state.ViewTaskDetail:
		// Ensure component has latest data
		r.DetailComp.SetSize(r.Width, r.Height)
		r.DetailComp.SetTask(r.SelectedTask)
		r.DetailComp.SetComments(r.Comments)
		r.DetailComp.SetProjects(r.Projects) // Ensure projects are set
		r.DetailComp.Focus()
		content = r.DetailComp.View()
	case state.ViewTaskForm:
		content = r.renderTaskForm()
	case state.ViewSearch:
		content = r.renderSearch()
	case state.ViewCalendarDay:
		content = r.renderCalendarDay()
	case state.ViewSections:
		content = r.renderSections()
	default:
		// Update DetailComp projects here too just in case split view needs it
		r.DetailComp.SetProjects(r.Projects)
		content = r.renderMainView()
	}

	// Overlay content checks
	type overlaySpec struct {
		active bool
		render func() string
	}

	overlays := []overlaySpec{
		{r.IsCreatingProject, r.renderProjectDialog},
		{r.IsEditingProject, r.renderProjectEditDialog},
		{r.ConfirmDeleteProject && r.EditingProject != nil, r.renderProjectDeleteDialog},
		{r.IsCreatingLabel, r.renderLabelDialog},
		{r.IsEditingLabel, r.renderLabelEditDialog},
		{r.ConfirmDeleteLabel && r.EditingLabel != nil, r.renderLabelDeleteDialog},
		{r.IsCreatingSubtask, r.renderSubtaskDialog},
		{r.IsCreatingSection, r.renderSectionDialog},
		{r.IsEditingSection, r.renderSectionEditDialog},
		{r.ConfirmDeleteSection && r.EditingSection != nil, r.renderSectionDeleteDialog},
		{r.IsMovingTask, r.renderMoveTaskDialog},
		{r.IsAddingComment, r.renderCommentDialog},
	}

	for _, o := range overlays {
		if o.active {
			content = r.overlayContent(content, o.render())
		}
	}

	return content
}

// renderMainView renders the main layout with tab bar and content.
func (r *Renderer) renderMainView() string {
	// Render tab bar
	tabBar := r.renderTabBar()

	// Calculate content height dynamically (total - tab bar - status bar)
	tabBarHeight := lipgloss.Height(tabBar)
	statusBarHeight := 2
	contentHeight := r.Height - tabBarHeight - statusBarHeight

	var mainContent string

	// If detail panel is shown, split the view
	if r.ShowDetailPanel && r.SelectedTask != nil {
		// Split layout
		detailWidth := r.Width / 2
		remainingWidth := r.Width - detailWidth - 3 // -3 for border/spacing

		if r.CurrentTab == state.TabProjects {
			// Three-pane layout: Sidebar | Tasks | Detail
			sidebarWidth := 30
			if remainingWidth < 70 {
				sidebarWidth = 20
			}
			if remainingWidth < 50 {
				sidebarWidth = 15
			}

			// We need 2 spaces for joins
			taskListWidth := remainingWidth - sidebarWidth - 2

			// Sizing validation
			if taskListWidth < 20 {
				taskListWidth = 20
			}

			// Render Sidebar
			r.SidebarComp.SetSize(sidebarWidth, contentHeight)
			r.SidebarComp.SetCursor(r.SidebarCursor)
			if r.FocusedPane == state.PaneSidebar {
				r.SidebarComp.Focus()
			} else {
				r.SidebarComp.Blur()
			}
			if r.CurrentProject != nil {
				r.SidebarComp.SetActiveProject(r.CurrentProject.ID)
			}
			sidebarPane := r.SidebarComp.View()

			// Render Task List
			taskListPane := r.renderProjectTaskList(taskListWidth, contentHeight)

			// Render Detail
			r.DetailComp.SetSize(detailWidth, contentHeight)
			r.DetailComp.SetTask(r.SelectedTask)
			r.DetailComp.SetComments(r.Comments)
			if r.CurrentView == state.ViewTaskDetail {
				r.DetailComp.Focus()
			} else {
				r.DetailComp.Blur()
			}
			rightPane := r.DetailComp.ViewPanel()

			// Enforce strict dimensions for top alignment and stable layout
			sidebarPane = lipgloss.Place(sidebarWidth, contentHeight, lipgloss.Left, lipgloss.Top, sidebarPane)
			taskListPane = lipgloss.Place(taskListWidth, contentHeight, lipgloss.Left, lipgloss.Top, taskListPane)
			rightPane = lipgloss.Place(detailWidth, contentHeight, lipgloss.Left, lipgloss.Top, rightPane)

			mainContent = lipgloss.JoinHorizontal(lipgloss.Top, sidebarPane, " ", taskListPane, " ", rightPane)
		} else {
			// Two-pane layout: Tasks | Detail
			// Need 1 space for join
			listWidth := remainingWidth - 1
			leftPane := r.renderTaskList(listWidth, contentHeight)

			// Render Detail
			r.DetailComp.SetSize(detailWidth, contentHeight)
			r.DetailComp.SetTask(r.SelectedTask)
			r.DetailComp.SetComments(r.Comments)
			if r.CurrentView == state.ViewTaskDetail {
				r.DetailComp.Focus()
			} else {
				r.DetailComp.Blur()
			}
			rightPane := r.DetailComp.ViewPanel()

			// Enforce strict dimensions
			leftPane = lipgloss.Place(listWidth, contentHeight, lipgloss.Left, lipgloss.Top, leftPane)
			rightPane = lipgloss.Place(detailWidth, contentHeight, lipgloss.Left, lipgloss.Top, rightPane)

			mainContent = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " ", rightPane)
		}

	} else {
		if r.CurrentTab == state.TabProjects {
			// Projects tab shows sidebar + content
			mainContent = r.renderProjectsTabContent(r.Width, contentHeight)
		} else {
			// Other tabs show content only (full width)
			mainContent = r.renderTaskList(r.Width-2, contentHeight)
		}
	}

	// Add status bar
	statusBar := r.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, mainContent, statusBar)
}

// renderTabBar renders the top tab bar.
func (r *Renderer) renderTabBar() string {
	tabs := state.GetTabDefinitions()

	// Determine label style based on available width
	// Full: "T Today" (~9 chars rendered), Short: "T Tdy" (~7 chars), Minimal: "T" (~3 chars)
	// Each tab with padding(2+2) + separator(1) = +5 chars overhead
	// 5 tabs * 14 chars (full with padding) = ~70 chars minimum for full labels
	useShortLabels := r.Width < 80
	useMinimalLabels := r.Width < 50

	var tabStrs []string
	for _, t := range tabs {
		var label string
		if useMinimalLabels {
			label = t.Icon
		} else if useShortLabels {
			label = fmt.Sprintf("%s %s", t.Icon, t.ShortName)
		} else {
			label = fmt.Sprintf("%s %s", t.Icon, t.Name)
		}

		if r.CurrentTab == t.Tab {
			tabStrs = append(tabStrs, styles.TabActive.Render(label))
		} else {
			tabStrs = append(tabStrs, styles.Tab.Render(label))
		}
	}

	tabLine := strings.Join(tabStrs, " ")

	// Truncate if still too wide
	maxWidth := r.Width - 4 // Account for state.TabBar padding
	if lipgloss.Width(tabLine) > maxWidth && maxWidth > 0 {
		tabLine = lipgloss.NewStyle().MaxWidth(maxWidth).Render(tabLine)
	}

	return styles.TabBar.Width(r.Width).Render(tabLine)
}
