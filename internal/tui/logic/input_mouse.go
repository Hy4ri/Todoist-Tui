package logic

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

func (h *Handler) handleMouseMsg(msg tea.MouseMsg) tea.Cmd {
	// Handle wheel scroll
	if msg.Type == tea.MouseWheelUp || msg.Type == tea.MouseWheelDown {
		return h.handleMouseScroll(msg)
	}

	// Only handle left clicks for other actions
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}

	// Skip if in modal views
	if h.CurrentView == state.ViewHelp || h.CurrentView == state.ViewTaskForm || h.CurrentView == state.ViewQuickAdd || h.CurrentView == state.ViewSearch || h.CurrentView == state.ViewTaskDetail {
		return nil
	}

	// If command line is active, check if click is outside (to dismiss)
	if h.CommandLine != nil && h.CommandLine.Active {
		// For now, any click dismisses command line
		h.CommandLine.Active = false
		return nil
	}

	x, y := msg.X, msg.Y

	// Check if click is on tab bar (first 3 lines: border + tabs + border)
	if y <= 2 {
		return h.handleTabClick(x)
	}

	// Handle clicks in main content area
	contentStartY := 3 // After tab bar
	if y >= contentStartY {
		return h.handleContentClick(x, y-contentStartY)
	}

	return nil
}

// handleTabClick handles mouse clicks on the tab bar.
func (h *Handler) handleTabClick(x int) tea.Cmd {
	tabs := state.GetTabDefinitions()

	// Determine label style based on available width (same logic as renderTabBar)
	useShortLabels := h.Width < 80
	useMinimalLabels := h.Width < 50

	// Calculate actual rendered positions for each tab
	currentPos := 1 // Start after styles.TabBar left padding

	for _, t := range tabs {
		var label string
		if useMinimalLabels {
			label = t.Icon
		} else if useShortLabels {
			label = fmt.Sprintf("%s %s", t.Icon, t.ShortName)
		} else {
			label = fmt.Sprintf("%s %s", t.Icon, t.Name)
		}

		// Render the tab to get its actual width (includes padding from styles.Tab/styles.TabActive style)
		var renderedTab string
		if h.CurrentTab == t.Tab {
			renderedTab = styles.TabActive.Render(label)
		} else {
			renderedTab = styles.Tab.Render(label)
		}

		tabWidth := lipgloss.Width(renderedTab)
		endPos := currentPos + tabWidth

		// Check if click is within this tab
		if x >= currentPos && x < endPos {
			return h.switchToTab(t.Tab)
		}

		// Move to next tab position (+1 for space separator between tabs)
		currentPos = endPos + 1
	}

	return nil
}

// handleContentClick handles mouse clicks in the content area.
func (h *Handler) handleContentClick(x, y int) tea.Cmd {
	// In Projects tab, check if click is in sidebar
	if h.CurrentTab == state.TabProjects {
		sidebarWidth := styles.SidebarWidth(h.Width)
		if x < sidebarWidth {
			// Click in sidebar
			h.FocusedPane = state.PaneSidebar
			// Calculate which item was clicked (accounting for title + blank line)
			itemIdx := y - 2
			if itemIdx >= 0 && itemIdx < len(h.SidebarItems) {
				// Skip separators
				if h.SidebarItems[itemIdx].Type != "separator" {
					h.SidebarCursor = itemIdx
					// Select the item
					return h.handleSelect()
				}
			}
		} else {
			// Click in main content
			h.FocusedPane = state.PaneMain
			return h.handleTaskClick(y)
		}
	} else if h.CurrentTab == state.TabFilters {
		sidebarWidth := styles.SidebarWidth(h.Width)

		if x < sidebarWidth {
			// Click in sidebar
			h.FocusedPane = state.PaneSidebar
			itemIdx := y - 2 // Header + Separator

			visible := h.getVisibleFilters()
			if itemIdx >= 0 && itemIdx < len(visible) {
				h.FilterCursor = itemIdx
				return h.handleFilterSelect()
			}
		} else {
			// Click in main content
			h.FocusedPane = state.PaneMain
			return h.handleTaskClick(y)
		}
	} else if h.CurrentView == state.ViewCalendar {
		return h.handleCalendarClick(x, y)
	} else {
		// Other tabs - click directly on tasks
		return h.handleTaskClick(y)
	}

	return nil
}

// handleTaskClick handles clicking on a task in the task list.
func (h *Handler) handleTaskClick(y int) tea.Cmd {
	// Calculate header offset based on view
	// Default: title (1 line) + blank line (1 line) = 2 lines
	headerOffset := 2

	// Account for scroll indicator if there's content above
	if h.ScrollOffset > 0 {
		headerOffset++ // "▲ N more above" takes 1 line
	}

	// In Labels view, might be clicking on a label
	if h.CurrentView == state.ViewLabels && h.CurrentLabel == nil {
		labelsToUse := h.Labels
		if len(labelsToUse) == 0 {
			labelsToUse = h.extractLabelsFromTasks()
		}
		// Adjust for scroll offset
		clickedIdx := y - headerOffset + h.ScrollOffset
		if clickedIdx >= 0 && clickedIdx < len(labelsToUse) {
			h.TaskCursor = clickedIdx
			return h.handleSelect()
		}
		return nil
	}

	// For task lists - use viewportLines mapping if available
	// viewportLines maps viewport line number to task index (-1 for headers)
	viewportLine := y - headerOffset + h.ScrollOffset

	if len(h.State.ViewportLines) > 0 {
		// Use the viewport line mapping for accurate click handling
		if viewportLine >= 0 && viewportLine < len(h.State.ViewportLines) {
			taskIndex := h.State.ViewportLines[viewportLine]
			if taskIndex >= 0 {
				// Find the display position (cursor) for this task index
				for displayPos, idx := range h.TaskOrderedIndices {
					if idx == taskIndex {
						h.TaskCursor = displayPos
						return nil
					}
				}
			}
		}
	} else {
		// Fallback for simple lists without section headers
		if viewportLine >= 0 && viewportLine < len(h.Tasks) {
			h.TaskCursor = viewportLine
			return nil
		}
	}

	return nil
}

// handleCalendarClick handles clicks in the calendar view.
func (h *Handler) handleCalendarClick(x, y int) tea.Cmd {
	if h.State.CalendarViewMode == state.CalendarViewCompact {
		return h.handleCalendarCompactClick(x, y)
	}
	return h.handleCalendarExpandedClick(x, y)
}

func (h *Handler) handleCalendarCompactClick(x, y int) tea.Cmd {
	// Compact view layout:
	// y=0: Title
	// y=1: Empty
	// y=2: Help
	// y=3: Empty
	// y=4: Weekdays
	// y=5+: Grid

	if y < 5 {
		return nil
	}

	// Grid calculations
	firstOfMonth := time.Date(h.CalendarDate.Year(), h.CalendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
	startWeekday := int(firstOfMonth.Weekday())
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	daysInMonth := lastOfMonth.Day()

	row := y - 5
	col := x / 5 // Cell width is 5 in compact view

	if col < 0 || col > 6 {
		return nil
	}

	// Check if click is on the task list below the calendar
	weeksNeeded := (daysInMonth + startWeekday + 6) / 7
	if row >= weeksNeeded {
		// Offset for tasks: Grid rows + 2 context lines (date title + blank)
		return h.handleTaskClick(y)
	}

	day := row*7 + col - startWeekday + 1
	if day >= 1 && day <= daysInMonth {
		h.CalendarDay = day
		return nil
	}

	return nil
}

func (h *Handler) handleCalendarExpandedClick(x, y int) tea.Cmd {
	// Expanded view layout:
	// y=0: Title
	// y=1: Empty
	// y=2: Help
	// y=3: Empty
	// y=4: Weekdays
	// y=5: Top Border
	// y=6+: Rows

	if y < 6 {
		return nil
	}

	// Calculate cell dimensions exactly as in renderCalendarExpanded
	availableWidth := h.Width - 8
	if availableWidth < 35 {
		availableWidth = 35
	}
	cellWidth := availableWidth / 7
	if cellWidth < 5 {
		cellWidth = 5
	}
	if cellWidth > 20 {
		cellWidth = 20
	}

	// Row height calculations
	firstOfMonth := time.Date(h.CalendarDate.Year(), h.CalendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	startWeekday := int(firstOfMonth.Weekday())
	daysInMonth := lastOfMonth.Day()
	weeksNeeded := (daysInMonth + startWeekday + 6) / 7

	maxHeight := h.Height - 7 // Matches renderer innerHeight (r.Height - 5 - 2)
	availableForWeeks := maxHeight - 6
	if availableForWeeks < weeksNeeded*3 {
		availableForWeeks = weeksNeeded * 3
	}
	maxTasksPerCell := (availableForWeeks+1)/weeksNeeded - 2
	if maxTasksPerCell < 2 {
		maxTasksPerCell = 2
	}
	if maxTasksPerCell > 6 {
		maxTasksPerCell = 6
	}

	rowHeight := maxTasksPerCell + 2 // 1 (day) + maxTasksPerCell + 1 (separator/border)

	// Determine which row and column
	gridY := y - 6
	row := gridY / rowHeight
	// Column with borders: │ Col0 │ Col1 ...
	// Cell: 1 (│) + cellWidth chars
	col := (x - 1) / (cellWidth + 1)

	if col < 0 || col > 6 || row < 0 || row >= weeksNeeded {
		return nil
	}

	// Calculate day
	day := row*7 + col - startWeekday + 1
	if day >= 1 && day <= daysInMonth {
		h.CalendarDay = day
		return nil
	}

	return nil
}

// handleMouseScroll handles mouse wheel scrolling.
func (h *Handler) handleMouseScroll(msg tea.MouseMsg) tea.Cmd {
	// Check if scroll is over the sidebar for tabs that have one
	sidebarWidth := styles.SidebarWidth(h.Width)
	isOverSidebar := (h.CurrentTab == state.TabProjects || h.CurrentTab == state.TabFilters) && msg.X < sidebarWidth

	if isOverSidebar {
		h.FocusedPane = state.PaneSidebar
		if msg.Type == tea.MouseWheelUp {
			h.moveSidebarCursor(-1)
		} else {
			h.moveSidebarCursor(1)
		}
	} else {
		// Main content scroll
		h.FocusedPane = state.PaneMain
		if msg.Type == tea.MouseWheelUp {
			h.moveCursor(-1)
		} else {
			h.moveCursor(1)
		}
	}

	return nil
}

// moveSidebarCursor moves the sidebar cursor and selects the current item.
func (h *Handler) moveSidebarCursor(delta int) {
	newIdx := h.SidebarCursor + delta
	if newIdx < 0 {
		newIdx = 0
	}
	if newIdx >= len(h.SidebarItems) {
		newIdx = len(h.SidebarItems) - 1
	}

	// Skip separators when scrolling
	if newIdx != h.SidebarCursor && h.SidebarItems[newIdx].Type == "separator" {
		if delta > 0 {
			newIdx++
		} else {
			newIdx--
		}
		// Bound check again after skipping
		if newIdx < 0 {
			newIdx = 0
		}
		if newIdx >= len(h.SidebarItems) {
			newIdx = len(h.SidebarItems) - 1
		}
	}

	h.SidebarCursor = newIdx
}
