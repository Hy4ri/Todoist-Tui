import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/config"
)

func (a *App) handleCalendarKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	firstOfMonth := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	daysInMonth := lastOfMonth.Day()

	switch msg.String() {
	case "q":
		return a, tea.Quit
	case "esc":
		return a.handleBack()
	case "tab":
		a.switchPane()
		return a, nil
	case "?":
		a.previousView = a.currentView
		a.currentView = ViewHelp
		return a, nil
	case "h", "left":
		// Previous day
		if a.calendarDay > 1 {
			a.calendarDay--
		} else {
			// Go to previous month's last day
			a.calendarDate = a.calendarDate.AddDate(0, -1, 0)
			prevMonth := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
			a.calendarDay = prevMonth.AddDate(0, 1, -1).Day()
		}
	case "l", "right":
		// Next day
		if a.calendarDay < daysInMonth {
			a.calendarDay++
		} else {
			// Go to next month's first day
			a.calendarDate = a.calendarDate.AddDate(0, 1, 0)
			a.calendarDay = 1
		}
	case "k", "up":
		// Previous week
		if a.calendarDay > 7 {
			a.calendarDay -= 7
		} else {
			// Go to previous month
			a.calendarDate = a.calendarDate.AddDate(0, -1, 0)
			prevMonth := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
			prevDays := prevMonth.AddDate(0, 1, -1).Day()
			newDay := a.calendarDay - 7 + prevDays
			if newDay > prevDays {
				newDay = prevDays
			}
			a.calendarDay = newDay
		}
	case "j", "down":
		// Next week
		if a.calendarDay+7 <= daysInMonth {
			a.calendarDay += 7
		} else {
			// Go to next month
			leftover := a.calendarDay + 7 - daysInMonth
			a.calendarDate = a.calendarDate.AddDate(0, 1, 0)
			nextMonth := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
			nextDays := nextMonth.AddDate(0, 1, -1).Day()
			if leftover > nextDays {
				leftover = nextDays
			}
			a.calendarDay = leftover
		}
	case "[":
		// Previous month
		a.calendarDate = a.calendarDate.AddDate(0, -1, 0)
		prevMonth := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
		prevDays := prevMonth.AddDate(0, 1, -1).Day()
		if a.calendarDay > prevDays {
			a.calendarDay = prevDays
		}
	case "]":
		// Next month
		a.calendarDate = a.calendarDate.AddDate(0, 1, 0)
		nextMonth := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
		nextDays := nextMonth.AddDate(0, 1, -1).Day()
		if a.calendarDay > nextDays {
			a.calendarDay = nextDays
		}
	case "t":
		// Go to today
		a.calendarDate = time.Now()
		a.calendarDay = time.Now().Day()
	case "v":
		// Toggle calendar view mode and save preference
		if a.calendarViewMode == CalendarViewCompact {
			a.calendarViewMode = CalendarViewExpanded
			a.config.UI.CalendarDefaultView = "expanded"
		} else {
			a.calendarViewMode = CalendarViewCompact
			a.config.UI.CalendarDefaultView = "compact"
		}
		// Save config in background (ignore errors)
		go func() {
			_ = config.Save(a.config)
		}()
	case "enter":
		// Open day detail view
		a.previousView = a.currentView
		a.currentView = ViewCalendarDay
		a.taskCursor = 0
		// Load tasks for this specific day
		return a, a.loadCalendarDayTasks()
	}

	return a, nil
}

// moveCursor moves the cursor by delta in the current list.
func (a *App) moveCursor(delta int) {
	if a.focusedPane == PaneSidebar && a.currentTab == TabProjects {
		newPos := a.sidebarCursor + delta
		// Skip separators
		for newPos >= 0 && newPos < len(a.sidebarItems) && a.sidebarItems[newPos].Type == "separator" {
			if delta > 0 {
				newPos++
			} else {
				newPos--
			}
		}
		if newPos < 0 {
			newPos = 0
		}
		if newPos >= len(a.sidebarItems) {
			newPos = len(a.sidebarItems) - 1
		}
		// Make sure we don't land on a separator
		for newPos >= 0 && newPos < len(a.sidebarItems) && a.sidebarItems[newPos].Type == "separator" {
			newPos--
		}
		if newPos >= 0 {
			a.sidebarCursor = newPos
		}
	} else {
		// Determine max items based on current view
		maxItems := len(a.tasks)

		// In Labels view without a selected label, navigate labels not tasks
		if a.currentView == ViewLabels && a.currentLabel == nil {
			labelsToUse := a.labels
			if len(labelsToUse) == 0 {
				labelsToUse = a.extractLabelsFromTasks()
			}
			maxItems = len(labelsToUse)
		}

		// In project view, use ordered indices (includes empty section headers)
		if a.currentView == ViewProject && len(a.taskOrderedIndices) > 0 {
			maxItems = len(a.taskOrderedIndices)
		}

		a.taskCursor += delta
		if a.taskCursor < 0 {
			a.taskCursor = 0
		}
		if maxItems > 0 && a.taskCursor >= maxItems {
			a.taskCursor = maxItems - 1
		}
		if a.taskCursor < 0 {
			a.taskCursor = 0
		}
	}
}

// moveCursorTo moves cursor to a specific position.
func (a *App) moveCursorTo(pos int) {
	if a.focusedPane == PaneSidebar {
		a.sidebarCursor = pos
	} else {
		a.taskCursor = pos
	}
}

// moveCursorToEnd moves cursor to the last item.
func (a *App) moveCursorToEnd() {
	if a.focusedPane == PaneSidebar {
		if len(a.sidebarItems) > 0 {
			a.sidebarCursor = len(a.sidebarItems) - 1
			// Skip separator
			for a.sidebarCursor > 0 && a.sidebarItems[a.sidebarCursor].Type == "separator" {
				a.sidebarCursor--
			}
		}
	} else {
		// Handle labels list view (when viewing label list, not label tasks)
		if a.currentView == ViewLabels && a.currentLabel == nil {
			labelsToShow := a.labels
			if len(labelsToShow) == 0 {
				labelsToShow = a.extractLabelsFromTasks()
			}
			if len(labelsToShow) > 0 {
				a.taskCursor = len(labelsToShow) - 1
			}
		} else if len(a.tasks) > 0 {
			a.taskCursor = len(a.tasks) - 1
		}
	}
}

// syncViewportToCursor ensures the viewport is scrolled to show the cursor line.
func (a *App) syncViewportToCursor(cursorLine int) {
	if !a.viewportReady {
		return
	}
	visibleStart := a.taskViewport.YOffset
	visibleEnd := visibleStart + a.taskViewport.Height

	if cursorLine < visibleStart {
		// Cursor above viewport - scroll up
		a.taskViewport.SetYOffset(cursorLine)
	} else if cursorLine >= visibleEnd {
		// Cursor below viewport - scroll down to show cursor at bottom
		a.taskViewport.SetYOffset(cursorLine - a.taskViewport.Height + 1)
	}
}

// switchPane toggles between sidebar and main pane (only in Projects tab).
func (a *App) switchPane() {
	// Only switch panes in Projects tab
	if a.currentTab != TabProjects {
		return
	}
	if a.focusedPane == PaneSidebar {
		a.focusedPane = PaneMain
	} else {
		a.focusedPane = PaneSidebar
	}
}

// sortTasks sorts tasks by due datetime (earliest first), then by priority (highest first).
// Tasks without due dates come after those with due dates.
