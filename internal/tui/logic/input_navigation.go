package logic

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/config"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// switchToTab switches to a specific tab using the view coordinator.
func (h *Handler) switchToTab(tab state.Tab) tea.Cmd {
	// Capture the current task before switching
	if h.SelectedTask != nil {
		h.LastSelectedTask = h.SelectedTask
	} else if task := h.getSelectedTask(); task != nil {
		h.LastSelectedTask = task
	}

	// Selections are view-local: clear them whenever we switch tabs.
	h.clearSelection()

	// Reset detail panel state when switching tabs
	if h.ShowDetailPanel {
		h.ShowDetailPanel = false
		h.SelectedTask = nil
		h.Comments = nil
		h.DetailComp.Hide()
	}

	// Don't switch if in modal views
	if h.CurrentView == state.ViewHelp || h.CurrentView == state.ViewTaskForm || h.CurrentView == state.ViewQuickAdd || h.CurrentView == state.ViewSearch || h.CurrentView == state.ViewTaskDetail {
		return nil
	}

	h.TaskCursor = 0
	h.CurrentLabel = nil

	// Update state via coordinator (sets CurrentTab, CurrentView, FocusedPane)
	h.coordinator.SwitchToTab(tab)
	h.CurrentTab = tab
	switch tab {
	case state.TabInbox:
		h.CurrentView = state.ViewInbox
		h.CurrentProject = nil
		h.Sections = nil
		h.FocusedPane = state.PaneMain
		return h.loadInboxTasks()
	case state.TabToday:
		h.CurrentView = state.ViewToday
		h.CurrentProject = nil
		h.FocusedPane = state.PaneMain
		return h.filterTodayTasks()
	case state.TabUpcoming:
		h.CurrentView = state.ViewUpcoming
		h.CurrentProject = nil
		h.FocusedPane = state.PaneMain
		return h.filterUpcomingTasks()
	case state.TabLabels:
		h.CurrentView = state.ViewLabels
		h.CurrentProject = nil
		h.FocusedPane = state.PaneMain
		return nil
	case state.TabCalendar:
		h.CurrentView = state.ViewCalendar
		h.CurrentProject = nil
		h.FocusedPane = state.PaneMain
		h.CalendarDate = time.Now()
		h.CalendarDay = time.Now().Day()
		return h.filterCalendarTasks()
	case state.TabProjects:
		h.CurrentView = state.ViewProject
		h.FocusedPane = state.PaneSidebar
		h.SidebarCursor = 0
		return nil
	case state.TabFilters:
		h.CurrentView = state.ViewFilters
		h.CurrentProject = nil
		h.CurrentFilter = nil // Clear any previous filter
		h.Tasks = nil         // Clear tasks until user selects a filter
		h.FocusedPane = state.PaneSidebar
		h.FilterCursor = 0
		h.FilterInput = textinput.New()
		h.FilterInput.Placeholder = "Search filters..."
		h.IsFilterSearch = false
		h.FilterSearchQuery = ""
		return h.loadFilters()
	case state.TabCompleted:
		h.CurrentView = state.ViewCompleted
		h.CurrentProject = nil
		h.FocusedPane = state.PaneMain
		h.CompletedPage = 0
		h.Tasks = nil // Clear tasks
		h.TaskCursor = 0
		return h.loadCompletedTasks()
	case state.TabPomodoro:
		h.CurrentView = state.ViewPomodoro
		h.FocusedPane = state.PaneMain
		return nil
	}

	return nil
}

// handleSelect handles the Enter key for selecting items.
func (h *Handler) handleSelect() tea.Cmd {
	// Handle sidebar selection (only in Projects tab)
	if h.CurrentTab == state.TabProjects && h.FocusedPane == state.PaneSidebar {
		if h.SidebarCursor >= len(h.SidebarItems) {
			return nil
		}

		item := h.SidebarItems[h.SidebarCursor]
		if item.Type == "separator" {
			return nil
		}

		h.FocusedPane = state.PaneMain
		h.TaskCursor = 0

		// Find project by ID
		for i := range h.Projects {
			if h.Projects[i].ID == item.ID {
				h.CurrentProject = &h.Projects[i]
				h.Sections = nil // Clear sections before loading new ones

				// Selections are project-local: clear them when switching projects.
				h.clearSelection()

				// Close detail panel when switching projects
				if h.ShowDetailPanel {
					h.ShowDetailPanel = false
					h.SelectedTask = nil
					h.Comments = nil
					h.DetailComp.Hide()
				}

				// Use cached data if fresh (within 30s) for instant project switching
				dataIsFresh := len(h.AllTasks) > 0 && time.Since(h.LastDataFetch) < 30*time.Second
				if dataIsFresh {
					h.TasksSorted = false // Reset sorted flag
					return h.filterProjectTasks(h.CurrentProject.ID)
				}

				// Otherwise fetch from API
				h.Loading = true
				return h.loadProjectTasks(h.CurrentProject.ID)
			}
		}
		return nil
	}

	// Handle main pane selection based on current view
	switch h.CurrentView {
	case state.ViewLabels:
		// Select label to filter tasks
		if h.CurrentLabel == nil {
			labelsToUse := h.Labels
			if len(labelsToUse) == 0 {
				labelsToUse = h.extractLabelsFromTasks()
			}
			if h.TaskCursor < len(labelsToUse) {
				h.CurrentLabel = &labelsToUse[h.TaskCursor]
				h.TaskCursor = 0 // Reset cursor for task list
				return h.filterLabelTasks(h.CurrentLabel.Name)
			}
		} else {
			// Viewing label tasks - select task for detail
			if h.TaskCursor < len(h.Tasks) {
				h.SelectedTask = &h.Tasks[h.TaskCursor]
				h.ShowDetailPanel = true
				return h.loadTaskComments()
			}
		}
	case state.ViewCalendar:
		// Selection handled by calendar navigation
		return nil
	default:
		// Select task for detail view
		task := h.getSelectedTask()
		if task == nil {
			return nil
		}
		taskCopy := new(api.Task)
		*taskCopy = *task
		h.SelectedTask = taskCopy
		h.ShowDetailPanel = true
		return h.loadTaskComments()
	}
	return nil
}

// handleBack handles the Escape key.
func (h *Handler) handleBack() tea.Cmd {
	// Close detail panel if open
	if h.ShowDetailPanel {
		h.ShowDetailPanel = false
		h.SelectedTask = nil
		h.Comments = nil
		h.DetailComp.Hide()

		// If we are in ViewTaskDetail view (full screen detail), go back to previous view
		if h.CurrentView == state.ViewTaskDetail {
			// Restore previous view
			if h.PreviousView != state.ViewTaskDetail {
				h.CurrentView = h.PreviousView
			} else {
				// Fallback if previous view is invalid
				h.CurrentView = state.ViewInbox
			}
		}
		return nil
	}

	// If tasks are multi-selected, Escape cancels the selection before doing
	// anything else (so users don't accidentally navigate away mid-selection).
	if len(h.SelectedTaskIDs) > 0 {
		h.clearSelection()
		h.StatusMsg = "Selection cleared"
		return nil
	}

	switch h.CurrentView {
	case state.ViewTaskDetail:
		h.CurrentView = h.PreviousView
		h.SelectedTask = nil
		h.Comments = nil
	case state.ViewCalendarDay:
		// Go back to calendar view
		h.CurrentView = state.ViewCalendar
		h.TaskCursor = 0
		// Reload all tasks for calendar display
		return h.loadAllTasks()
	case state.ViewProject:
		// In Projects tab, just clear selection
		if h.CurrentTab == state.TabProjects {
			h.CurrentProject = nil
			h.Tasks = nil
			h.FocusedPane = state.PaneSidebar
		}
	case state.ViewLabels:
		if h.CurrentLabel != nil {
			// Go back to labels list
			h.CurrentLabel = nil
			h.Tasks = nil
			h.TaskCursor = 0
		}
	case state.ViewTaskForm:
		h.CurrentView = h.PreviousView
	}
	return nil
}

// setDefaultView saves the current view as the default.
func (h *Handler) setDefaultView() tea.Cmd {
	var viewName string
	switch h.CurrentTab {
	case state.TabInbox:
		viewName = "inbox"
	case state.TabToday:
		viewName = "today"
	case state.TabUpcoming:
		viewName = "upcoming"
	case state.TabLabels:
		viewName = "labels"
	case state.TabCalendar:
		viewName = "calendar"
	case state.TabProjects:
		viewName = "projects"
	}

	if viewName != "" {
		// Update in memory
		h.Config.UI.DefaultView = viewName

		// Update on disk (preserving comments)
		if err := config.UpdateDefaultView(viewName); err != nil {
			h.StatusMsg = fmt.Sprintf("Failed to save config: %v", err)
		} else {
			h.StatusMsg = fmt.Sprintf("Default view set to: %s", viewName)
		}
	}
	return nil
}

// handleSendTaskToPomodoro copies the currently selected task to the Pomodoro state.
func (h *Handler) handleSendTaskToPomodoro() tea.Cmd {
	task := h.getSelectedTask()

	// If in Pomodoro view, always try to use the last selected task from another view
	if h.CurrentView == state.ViewPomodoro && h.LastSelectedTask != nil {
		task = h.LastSelectedTask
	}

	if task == nil {
		h.StatusMsg = "No task selected to send to Pomodoro"
		return nil
	}

	taskCopy := new(api.Task)
	*taskCopy = *task
	h.PomodoroTask = taskCopy

	// Find project name for display
	for _, p := range h.Projects {
		if p.ID == task.ProjectID {
			h.PomodoroProject = p.Name
			break
		}
	}

	h.StatusMsg = "Task sent to Pomodoro 🍅"
	return nil
}
