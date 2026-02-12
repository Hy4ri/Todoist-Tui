package logic

import (
	"time"

	"github.com/hy4ri/todoist-tui/internal/tui/state"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/config"
)

func (h *Handler) handleCalendarKeyMsg(msg tea.KeyMsg) tea.Cmd {
	firstOfMonth := time.Date(h.CalendarDate.Year(), h.CalendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	daysInMonth := lastOfMonth.Day()

	switch msg.String() {
	case "q":
		return tea.Quit
	case "esc":
		return h.handleBack()
	case "tab":
		h.switchPane()
		return nil
	case "?":
		h.PreviousView = h.CurrentView
		h.CurrentView = state.ViewHelp
		return nil
	case "h", "left":
		// Previous day
		if h.CalendarDay > 1 {
			h.CalendarDay--
		} else {
			// Go to previous month's last day
			h.CalendarDate = h.CalendarDate.AddDate(0, -1, 0)
			prevMonth := time.Date(h.CalendarDate.Year(), h.CalendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
			h.CalendarDay = prevMonth.AddDate(0, 1, -1).Day()
		}
	case "l", "right":
		// Next day
		if h.CalendarDay < daysInMonth {
			h.CalendarDay++
		} else {
			// Go to next month's first day
			h.CalendarDate = h.CalendarDate.AddDate(0, 1, 0)
			h.CalendarDay = 1
		}
	case "k", "up":
		// Previous week
		if h.CalendarDay > 7 {
			h.CalendarDay -= 7
		} else {
			// Go to previous month
			h.CalendarDate = h.CalendarDate.AddDate(0, -1, 0)
			prevMonth := time.Date(h.CalendarDate.Year(), h.CalendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
			prevDays := prevMonth.AddDate(0, 1, -1).Day()
			newDay := h.CalendarDay - 7 + prevDays
			if newDay > prevDays {
				newDay = prevDays
			}
			h.CalendarDay = newDay
		}
	case "j", "down":
		// Next week
		if h.CalendarDay+7 <= daysInMonth {
			h.CalendarDay += 7
		} else {
			// Go to next month
			leftover := h.CalendarDay + 7 - daysInMonth
			h.CalendarDate = h.CalendarDate.AddDate(0, 1, 0)
			nextMonth := time.Date(h.CalendarDate.Year(), h.CalendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
			nextDays := nextMonth.AddDate(0, 1, -1).Day()
			if leftover > nextDays {
				leftover = nextDays
			}
			h.CalendarDay = leftover
		}
	case "[":
		// Previous month
		h.CalendarDate = h.CalendarDate.AddDate(0, -1, 0)
		prevMonth := time.Date(h.CalendarDate.Year(), h.CalendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
		prevDays := prevMonth.AddDate(0, 1, -1).Day()
		if h.CalendarDay > prevDays {
			h.CalendarDay = prevDays
		}
	case "]":
		// Next month
		h.CalendarDate = h.CalendarDate.AddDate(0, 1, 0)
		nextMonth := time.Date(h.CalendarDate.Year(), h.CalendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
		nextDays := nextMonth.AddDate(0, 1, -1).Day()
		if h.CalendarDay > nextDays {
			h.CalendarDay = nextDays
		}
	case "t":
		// Go to today
		h.CalendarDate = time.Now()
		h.CalendarDay = time.Now().Day()
	case "v":
		// Toggle calendar view mode and save preference
		if h.State.CalendarViewMode == state.CalendarViewCompact {
			h.State.CalendarViewMode = state.CalendarViewExpanded
			h.Config.UI.CalendarDefaultView = "expanded"
		} else {
			h.State.CalendarViewMode = state.CalendarViewCompact
			h.Config.UI.CalendarDefaultView = "compact"
		}
		// Save config in background (ignore errors)
		go func() {
			_ = config.Save(h.Config)
		}()
	case "enter":
		// Open day detail view
		h.PreviousView = h.CurrentView
		h.CurrentView = state.ViewCalendarDay
		h.TaskCursor = 0
		// Load tasks for this specific day
		return h.loadCalendarDayTasks()
	}

	return nil
}

// moveCursor moves the cursor by delta in the current list.
func (h *Handler) moveCursor(delta int) {
	if h.FocusedPane == state.PaneSidebar && h.CurrentTab == state.TabProjects {
		newPos := h.SidebarCursor + delta
		// Skip separators
		for newPos >= 0 && newPos < len(h.SidebarItems) && h.SidebarItems[newPos].Type == "separator" {
			if delta > 0 {
				newPos++
			} else {
				newPos--
			}
		}
		if newPos < 0 {
			newPos = 0
		}
		if newPos >= len(h.SidebarItems) {
			newPos = len(h.SidebarItems) - 1
		}
		// Make sure we don't land on a separator
		for newPos >= 0 && newPos < len(h.SidebarItems) && h.SidebarItems[newPos].Type == "separator" {
			newPos--
		}
		if newPos >= 0 {
			h.SidebarCursor = newPos
		}
	} else {
		// Determine max items based on current view
		maxItems := len(h.Tasks)

		// In Labels view without a selected label, navigate labels not tasks
		if h.CurrentView == state.ViewLabels && h.CurrentLabel == nil {
			labelsToUse := h.Labels
			if len(labelsToUse) == 0 {
				labelsToUse = h.extractLabelsFromTasks()
			}
			maxItems = len(labelsToUse)
		}

		// In project view, use ordered indices (includes empty section headers)
		if h.CurrentView == state.ViewProject && len(h.TaskOrderedIndices) > 0 {
			maxItems = len(h.TaskOrderedIndices)
		}

		h.TaskCursor += delta
		if h.TaskCursor < 0 {
			h.TaskCursor = 0
		}
		if maxItems > 0 && h.TaskCursor >= maxItems {
			h.TaskCursor = maxItems - 1
		}
		if h.TaskCursor < 0 {
			h.TaskCursor = 0
		}
	}
}

// moveCursorTo moves cursor to a specific position.
func (h *Handler) moveCursorTo(pos int) {
	if h.FocusedPane == state.PaneSidebar {
		h.SidebarCursor = pos
	} else {
		h.TaskCursor = pos
	}
}

// moveCursorToEnd moves cursor to the last item.
func (h *Handler) moveCursorToEnd() {
	if h.FocusedPane == state.PaneSidebar {
		if len(h.SidebarItems) > 0 {
			h.SidebarCursor = len(h.SidebarItems) - 1
			// Skip separator
			for h.SidebarCursor > 0 && h.SidebarItems[h.SidebarCursor].Type == "separator" {
				h.SidebarCursor--
			}
		}
	} else {
		// Handle labels list view (when viewing label list, not label tasks)
		if h.CurrentView == state.ViewLabels && h.CurrentLabel == nil {
			labelsToShow := h.Labels
			if len(labelsToShow) == 0 {
				labelsToShow = h.extractLabelsFromTasks()
			}
			if len(labelsToShow) > 0 {
				h.TaskCursor = len(labelsToShow) - 1
			}
		} else if len(h.Tasks) > 0 {
			h.TaskCursor = len(h.Tasks) - 1
		}
	}
}

// switchPane toggles between panes.
func (h *Handler) switchPane() {
	// If detail panel is open, toggle between detail view and main view
	if h.ShowDetailPanel {
		if h.CurrentView == state.ViewTaskDetail {
			// Switch back to list view
			switch h.CurrentTab {
			case state.TabInbox:
				h.CurrentView = state.ViewInbox
			case state.TabToday:
				h.CurrentView = state.ViewToday
			case state.TabUpcoming:
				h.CurrentView = state.ViewUpcoming
			case state.TabLabels:
				h.CurrentView = state.ViewLabels
			case state.TabCalendar:
				h.CurrentView = state.ViewCalendar
			case state.TabProjects:
				h.CurrentView = state.ViewProject
			}
		} else {
			// Switch to detail view
			h.CurrentView = state.ViewTaskDetail
		}
		return
	}

	// Only switch panes in Projects tab (Sidebar <-> Main)
	if h.CurrentTab != state.TabProjects {
		return
	}
	if h.FocusedPane == state.PaneSidebar {
		h.FocusedPane = state.PaneMain
	} else {
		h.FocusedPane = state.PaneSidebar
	}
}
