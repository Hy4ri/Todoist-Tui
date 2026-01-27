import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

func (h *Handler) handleMouseMsg(msg tea.MouseMsg) tea.Cmd {
	// Only handle clicks
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return nil
	}

	// Skip if in modal views
	if h.CurrentView == state.ViewHelp || h.CurrentView == state.ViewTaskForm || h.CurrentView == state.ViewSearch || h.CurrentView == state.ViewTaskDetail {
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
	tabs := getTabDefinitions()

	// Determine label style based on available width (same logic as renderTabBar)
	useShortLabels := h.Width < 80
	useMinimalLabels := h.Width < 50

	// Calculate actual rendered positions for each tab
	currentPos := 2 // Start after state.TabBar left padding

	for _, t := range tabs {
		var label string
		if useMinimalLabels {
			label = t.icon
		} else if useShortLabels {
			label = fmt.Sprintf("%s %s", t.icon, t.shortName)
		} else {
			label = fmt.Sprintf("%s %s", t.icon, t.name)
		}

		// Render the tab to get its actual width (includes padding from Tab/state.TabActive style)
		var renderedTab string
		if h.CurrentTab == t.tab {
			renderedTab = styles.state.TabActive.Render(label)
		} else {
			renderedTab = styles.Tab.Render(label)
		}

		tabWidth := lipgloss.Width(renderedTab)
		endPos := currentPos + tabWidth

		// Check if click is within this tab
		if x >= currentPos && x < endPos {
			h.CurrentTab = t.tab
			h.TaskCursor = 0
			h.CurrentLabel = nil

			switch t.tab {
			case state.TabToday:
				h.CurrentView = state.ViewToday
				h.CurrentProject = nil
				return h.filterTodayTasks()
			case state.TabUpcoming:
				h.CurrentView = state.ViewUpcoming
				h.CurrentProject = nil
				return h.filterUpcomingTasks()
			case state.TabLabels:
				h.CurrentView = state.ViewLabels
				h.CurrentProject = nil
				return nil
			case state.TabCalendar:
				h.CurrentView = state.ViewCalendar
				h.CurrentProject = nil
				h.CalendarDate = time.Now()
				h.CalendarDay = time.Now().Day()
				return h.filterCalendarTasks()
			case state.TabProjects:
				h.CurrentView = state.ViewProject
				h.FocusedPane = state.PaneSidebar
				h.SidebarCursor = 0
				return nil
			}
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
		sidebarWidth := 25
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

	if len(h.state.ViewportLines) > 0 {
		// Use the viewport line mapping for accurate click handling
		if viewportLine >= 0 && viewportLine < len(h.state.ViewportLines) {
			taskIndex := h.state.ViewportLines[viewportLine]
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

// switchToTab switches to a specific tab.
func (h *Handler) switchToTab(tab Tab) tea.Cmd {
	// Reset detail panel state when switching tabs
	if h.ShowDetailPanel {
		h.ShowDetailPanel = false
		h.SelectedTask = nil
		h.Comments = nil
		h.DetailComp.Hide()
	}

	// Don't switch if in modal views

	if h.CurrentView == state.ViewHelp || h.CurrentView == state.ViewTaskForm || h.CurrentView == state.ViewSearch || h.CurrentView == state.ViewTaskDetail {
		return nil
	}

	h.CurrentTab = tab
	h.TaskCursor = 0
	h.CurrentLabel = nil

	switch tab {
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
	}

	return nil
}

// handleKeyMsg processes keyboard input.
func (h *Handler) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// Only ctrl+c is truly global
	if msg.String() == "ctrl+c" {
		return tea.Quit
	}

	// If we're in help view, any key goes back
	if h.CurrentView == state.ViewHelp {
		h.CurrentView = h.PreviousView
		return nil
	}

	// Route key messages based on current view - BEFORE tab switching
	// This allows forms to capture number keys for text input
	switch h.CurrentView {
	case state.ViewTaskForm:
		return h.handleFormKeyMsg(msg)
	case state.ViewSearch:
		return h.handleSearchKeyMsg(msg)
	}

	// Handle all input/dialog states BEFORE tab switching
	// This prevents number keys from switching views during text entry

	// Project state handling
	if h.IsCreatingProject {
		return h.handleProjectInputKeyMsg(msg)
	}
	if h.IsEditingProject {
		return h.handleProjectEditKeyMsg(msg)
	}
	if h.ConfirmDeleteProject {
		return h.handleDeleteConfirmKeyMsg(msg)
	}

	// Label state handling
	if h.IsCreatingLabel {
		return h.handleLabelInputKeyMsg(msg)
	}
	if h.IsEditingLabel {
		return h.handleLabelEditKeyMsg(msg)
	}
	if h.ConfirmDeleteLabel {
		return h.handleLabelDeleteConfirmKeyMsg(msg)
	}

	// Section state handling
	if h.IsCreatingSection {
		return h.handleSectionInputKeyMsg(msg)
	}
	if h.IsEditingSection {
		return h.handleSectionEditKeyMsg(msg)
	}
	if h.ConfirmDeleteSection {
		return h.handleSectionDeleteConfirmKeyMsg(msg)
	}

	// Subtask creation handling
	if h.IsCreatingSubtask {
		return h.handleSubtaskInputKeyMsg(msg)
	}

	// Comment input handling
	if h.IsAddingComment {
		return h.handleCommentInputKeyMsg(msg)
	}

	// Move task handling
	if h.IsMovingTask {
		return h.handleMoveTaskKeyMsg(msg)
	}

	// Tab switching with number keys (1-5) - only when not in form/input modes
	// Tab switching with number keys (1-5) and letters - only when not in form/input modes
	switch msg.String() {
	case "1":
		return h.switchToTab(state.TabToday)
	case "2":
		return h.switchToTab(state.TabUpcoming)
	case "3":
		return h.switchToTab(state.TabLabels)
	case "4":
		return h.switchToTab(state.TabCalendar)
	case "5":
		return h.switchToTab(state.TabProjects)
	case "t", "T":
		return h.switchToTab(state.TabToday)
	case "u", "U":
		return h.switchToTab(state.TabUpcoming)
	case "p", "P":
		return h.switchToTab(state.TabProjects)
	case "c", "C":
		return h.switchToTab(state.TabCalendar)
	// 'l' is excluded here to preserve navigation in Calendar/Projects, handled in "right" action
	case "L":
		return h.switchToTab(state.TabLabels)
	}

	// Sections view routing
	if h.CurrentView == state.ViewSections {
		return h.handleSectionsKeyMsg(msg)
	}

	// If we're in calendar view, handle calendar-specific keys
	if h.CurrentView == state.ViewCalendar && h.FocusedPane == state.PaneMain {
		return h.handleCalendarKeyMsg(msg)
	}

	// Process key through keymap
	action, consumed := h.keyState.HandleKey(msg, h.Keymap)
	if !consumed {
		return nil
	}

	// Handle actions
	switch action {
	case "quit":
		return tea.Quit
	case "help":
		h.PreviousView = h.CurrentView
		h.CurrentView = state.ViewHelp
		return nil
	case "refresh":
		return func() tea.Msg { return refreshMsg{} }
	case "up":
		h.moveCursor(-1)
	case "down":
		h.moveCursor(1)
	case "top":
		h.moveCursorTo(0)
	case "bottom":
		h.moveCursorToEnd()
	case "half_up":
		h.moveCursor(-10)
	case "half_down":
		h.moveCursor(10)
	case "left":
		// h key - move to sidebar in Projects tab
		if h.CurrentTab == state.TabProjects && h.FocusedPane == state.PaneMain {
			h.FocusedPane = state.PaneSidebar
		}
	case "right":
		// l key - move to main pane in Projects tab
		if h.CurrentTab == state.TabProjects && h.FocusedPane == state.PaneSidebar {
			h.FocusedPane = state.PaneMain
		} else if h.CurrentView != state.ViewCalendar && msg.String() == "l" {
			// If not navigating projects or calendar, 'l' switches to Labels
			return h.switchToTab(state.TabLabels)
		}
	case "switch_pane":
		h.switchPane()
	case "select":
		return h.handleSelect()
	case "back":
		return h.handleBack()
	case "complete":
		return h.handleComplete()
	case "delete":
		return h.handleDelete()
	case "add":
		return h.handleAdd()
	case "edit":
		return h.handleEdit()
	case "search":
		return h.handleSearch()
	case "priority1", "priority2", "priority3", "priority4":
		return h.handlePriority(action)
	case "due_today":
		return h.handleDueToday()
	case "due_tomorrow":
		return h.handleDueTomorrow()
	case "new_project":
		// 'n' key creates project or label depending on current tab
		if h.CurrentTab == state.TabProjects {
			return h.handleNewProject()
		} else if h.CurrentTab == state.TabLabels {
			return h.handleNewLabel()
		}
	// Tab shortcuts (Shift + letter)
	case "tab_today":
		return h.switchToTab(state.TabToday)
	case "tab_upcoming":
		return h.switchToTab(state.TabUpcoming)
	case "tab_projects":
		return h.switchToTab(state.TabProjects)
	case "tab_labels":
		return h.switchToTab(state.TabLabels)
	case "tab_calendar":
		return h.switchToTab(state.TabCalendar)
	case "toggle_hints":
		h.ShowHints = !h.ShowHints
	case "add_subtask":
		return h.handleAddSubtask()
	case "undo":
		return h.handleUndo()
	case "manage_sections":
		if h.CurrentTab == state.TabProjects && len(h.Projects) > 0 {
			h.PreviousView = h.CurrentView
			h.CurrentView = state.ViewSections
			h.TaskCursor = 0 // use cursor for sections
			return nil
		}
	case "move_task":
		if h.FocusedPane == state.PaneMain && len(h.Tasks) > 0 && len(h.Sections) > 0 {
			h.IsMovingTask = true
			h.MoveSectionCursor = 0
			return nil
		}
	case "new_section":
		// Allow creating sections in project view when a project is selected
		if h.CurrentTab == state.TabProjects && h.CurrentProject != nil {
			h.SectionInput = textinput.New()
			h.SectionInput.Placeholder = "Enter section name..."
			h.SectionInput.CharLimit = 100
			h.SectionInput.Width = 40
			h.SectionInput.Focus()
			h.IsCreatingSection = true
			return nil
		}
	case "move_section":
		// Redirect to section management view
		if h.CurrentTab == state.TabProjects && h.CurrentProject != nil && len(h.Sections) > 1 {
			h.StatusMsg = "Use 'S' to manage sections - select with Space, reorder with Shift+j/k"
			return nil
		}
	// Note: 'C' key is not in keymap yet, handling manually or adding to keymap
	// Actually, I should add 'C' to keymap or handle via raw key manually if I want.
	// But let's assume I added 'C' -> 'add_comment' in keymap (I didn't yet).
	// I'll add the case here assuming I will update keymap.go next.
	case "add_comment":
		if h.SelectedTask != nil {
			h.IsAddingComment = true
			h.CommentInput = textinput.New()
			h.CommentInput.Placeholder = "Write a comment..."
			h.CommentInput.Focus()
			h.CommentInput.Width = 50
			return nil
		}
	case "toggle_select":
		return h.handleToggleSelect()
	case "copy":
		return h.handleCopy()
	}

	return nil
}

// handleCalendarKeyMsg handles keyboard input when calendar view is active.
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
				return h.filterProjectTasks(h.CurrentProject.ID)
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
			// state.Viewing label tasks - select task for detail
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
		// First check if we have tasks at all
		if len(h.Tasks) == 0 {
			return nil
		}

		// Try taskOrderedIndices first (for views with sections/groups)
		if len(h.TaskOrderedIndices) > 0 && h.TaskCursor >= 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
			taskIndex := h.TaskOrderedIndices[h.TaskCursor]

			// Skip if cursor is on empty section header (taskIndex <= -100)
			if taskIndex <= -100 {
				return nil
			}

			if taskIndex >= 0 && taskIndex < len(h.Tasks) {
				taskCopy := new(api.Task)
				*taskCopy = h.Tasks[taskIndex]
				h.SelectedTask = taskCopy
				h.ShowDetailPanel = true
				return h.loadTaskComments()
			}
		}

		// Fallback: use taskCursor directly to index h.Tasks
		// This works for views that don't pre-populate taskOrderedIndices
		if h.TaskCursor >= 0 && h.TaskCursor < len(h.Tasks) {
			taskCopy := new(api.Task)
			*taskCopy = h.Tasks[h.TaskCursor]
			h.SelectedTask = taskCopy
			h.ShowDetailPanel = true
			return h.loadTaskComments()
		}
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

// handleComplete toggles task completion.
