package logic

import (
	"fmt"
	"strings"
	"time"

	"github.com/hy4ri/todoist-tui/internal/config"
	"github.com/hy4ri/todoist-tui/internal/tui/state"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
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
		sidebarWidth := 30
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

// switchToTab switches to a specific tab using the view coordinator.
func (h *Handler) switchToTab(tab state.Tab) tea.Cmd {
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

	// Comment editing (Modal) - Check BEFORE view switching to capture input
	if h.IsEditingComment {
		return h.handleCommentEditKeyMsg(msg)
	}
	if h.ConfirmDeleteComment {
		return h.handleDeleteCommentConfirmKeyMsg(msg)
	}

	// Route key messages based on current view - BEFORE tab switching
	// This allows forms to capture number keys for text input
	switch h.CurrentView {
	case state.ViewTaskForm:
		return h.handleFormKeyMsg(msg)
	case state.ViewQuickAdd:
		return h.handleQuickAddKeyMsg(msg)
	case state.ViewSearch:
		return h.handleSearchKeyMsg(msg)
	case state.ViewTaskDetail:
		return h.handleTaskDetailKeyMsg(msg)
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

	// state.Tab switching with number keys (1-5) - only when not in form/input modes
	// state.Tab switching with number keys (1-5) and letters - only when not in form/input modes
	switch msg.String() {
	case "1":
		return h.switchToTab(state.TabInbox)
	case "2":
		return h.switchToTab(state.TabToday)
	case "3":
		return h.switchToTab(state.TabUpcoming)
	case "4":
		return h.switchToTab(state.TabLabels)
	case "5":
		return h.switchToTab(state.TabCalendar)
	case "6":
		return h.switchToTab(state.TabProjects)
	case "D": // Shift+d
		return h.setDefaultView()
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
	action, consumed := h.KeyState.HandleKey(msg, h.Keymap)
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
		// Context-aware add for Labels tab
		if h.CurrentTab == state.TabLabels && h.CurrentLabel == nil {
			return h.handleNewLabel()
		}
		return h.handleAdd()
	case "edit":
		return h.handleEdit()
	case "search":
		return h.handleSearch()
	case "priority1", "priority2", "priority3", "priority4":
		return h.handlePriority(action)
	case "due_tomorrow":
		return h.handleDueTomorrow()
	case "move_task_prev_day":
		return h.handleMoveTaskDate(-1)
	case "move_task_next_day":
		return h.handleMoveTaskDate(1)
	case "new_project":
		// 'n' key creates project or label depending on current tab
		if h.CurrentTab == state.TabProjects {
			return h.handleNewProject()
		} else if h.CurrentTab == state.TabLabels {
			return h.handleNewLabel()
		}
	case "toggle_favorite":
		if h.CurrentTab == state.TabProjects && h.FocusedPane == state.PaneSidebar {
			return h.handleToggleFavorite()
		}
	// state.Tab shortcuts (Shift + letter)
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
				h.Sections = nil // Clear sections before loading new ones

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
		h.DetailComp.Hide()

		// If we are in ViewTaskDetail view (full screen detail), go back to previous view
		// This happens when opening from a list and then maximizing or similar flow?
		// Actually the user report says: "when i press esc on the task details the after the tab it shows a black screen"
		// If ShowDetailPanel is true, we might be in ViewTaskDetail or just split view.

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

// handleFormKeyMsg handles keyboard input when the form is active.
func (h *Handler) handleFormKeyMsg(msg tea.KeyMsg) tea.Cmd {
	if h.TaskForm == nil {
		return nil
	}

	switch msg.String() {
	case "esc":
		// Cancel form and go back
		h.CurrentView = h.PreviousView
		h.TaskForm = nil
		return nil

	case "ctrl+enter":
		if h.Loading {
			return nil
		}
		return h.submitForm()

	case "enter":
		if h.Loading {
			return nil
		}
		// If on task content or description input, submit immediately with defaults
		if h.TaskForm.FocusIndex == state.FormFieldContent || h.TaskForm.FocusIndex == state.FormFieldDescription {
			return h.submitForm()
		}

		// If on submit button, submit form
		if h.TaskForm.FocusIndex == state.FormFieldSubmit {
			return h.submitForm()
		}
		// Otherwise, let form handle enter (e.g. for opening project list)
	}

	// Forward to form
	return h.TaskForm.Update(msg)
}

// handleQuickAddKeyMsg handles keyboard input for the Quick Add popup.
func (h *Handler) handleQuickAddKeyMsg(msg tea.KeyMsg) tea.Cmd {
	if h.QuickAddForm == nil {
		return nil
	}

	switch msg.String() {
	case "esc":
		// Close Quick Add and return to previous view
		h.CurrentView = h.PreviousView
		h.QuickAddForm = nil
		return nil

	case "enter":
		// Submit task if there's content
		if !h.QuickAddForm.IsValid() {
			h.StatusMsg = "Enter task content"
			return nil
		}

		// Capture form values
		content := h.QuickAddForm.Value()
		projectName := h.QuickAddForm.ProjectName

		// Clear input and increment count (stays open)
		h.QuickAddForm.Clear()
		h.QuickAddForm.IncrementCount()
		h.StatusMsg = "Adding task..."

		// Create task in background using Quick Add API
		return func() tea.Msg {
			// Build the quick add text - append project if context exists
			text := content
			if projectName != "" {
				// Only append #project if user didn't already specify one
				if !strings.Contains(content, "#") {
					text = content + " #" + projectName
				}
			}

			_, err := h.Client.QuickAddTask(text)
			if err != nil {
				return errMsg{err}
			}
			return quickAddTaskCreatedMsg{}
		}
	}

	// Forward other keys to the form input
	return h.QuickAddForm.Update(msg)
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
	// sidebar click handling logic uses x < 25 for Projects tab
	isOverSidebar := h.CurrentTab == state.TabProjects && msg.X < 25

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

// handleTaskDetailKeyMsg handles keys for task detail view.
func (h *Handler) handleTaskDetailKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// If any modal state is active, let the specific handler deal with it
	if h.IsEditingComment || h.ConfirmDeleteComment {
		return nil
	}

	// Check for global actions relevant to detail view
	action, consumed := h.KeyState.HandleKey(msg, h.Keymap)

	// Handle explicit ESC if not consumed by keymap (or if keymap maps ESC to back)
	if msg.String() == "esc" || action == "back" {
		return h.handleBack()
	}

	if consumed {
		switch action {
		case "quit":
			return tea.Quit
		case "add_comment":
			if h.SelectedTask != nil {
				h.IsAddingComment = true
				h.CommentInput = textinput.New()
				h.CommentInput.Placeholder = "Write a comment..."
				h.CommentInput.Focus()
				h.CommentInput.Width = 50
				return nil
			}
		case "edit":
			return h.handleEdit()
		case "delete":
			return h.handleDelete()
		case "complete":
			return h.handleComplete()
		case "priority1", "priority2", "priority3", "priority4":
			return h.handlePriority(action)
		case "due_tomorrow":
			return h.handleDueTomorrow()
		case "move_task_prev_day":
			return h.handleMoveTaskDate(-1)
		case "move_task_next_day":
			return h.handleMoveTaskDate(1)
		}
	}

	// Delegate to component (handles j/k scrolling, etc.)
	_, cmd := h.DetailComp.Update(msg)
	return cmd
}

// handleCommentEditKeyMsg handles keys for comment editing.
func (h *Handler) handleCommentEditKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsEditingComment = false
		h.EditingComment = nil
		h.CommentInput.Reset()
		return nil
	case "enter":
		content := h.CommentInput.Value()
		if content == "" {
			return nil
		}
		h.IsEditingComment = false
		h.Loading = true
		h.StatusMsg = "Updating comment..."
		commentID := h.EditingComment.ID
		h.EditingComment = nil
		h.CommentInput.Reset()

		return func() tea.Msg {
			c, err := h.Client.UpdateComment(commentID, api.UpdateCommentRequest{Content: content})
			if err != nil {
				return errMsg{err}
			}
			return commentUpdatedMsg{comment: c}
		}
	}
	var cmd tea.Cmd
	h.CommentInput, cmd = h.CommentInput.Update(msg)
	return cmd
}

// handleDeleteCommentConfirmKeyMsg handles confirmation for comment deletion.
func (h *Handler) handleDeleteCommentConfirmKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		h.ConfirmDeleteComment = false
		h.Loading = true
		h.StatusMsg = "Deleting comment..."
		commentID := h.EditingComment.ID
		h.EditingComment = nil

		return func() tea.Msg {
			err := h.Client.DeleteComment(commentID)
			if err != nil {
				return errMsg{err}
			}
			return commentDeletedMsg{id: commentID}
		}
	case "n", "N", "esc":
		h.ConfirmDeleteComment = false
		h.EditingComment = nil
		return nil
	}
	return nil
}

// handleCommentInputKeyMsg handles keys for adding a comment.
func (h *Handler) handleCommentInputKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsAddingComment = false
		h.CommentInput.Reset()
		return nil

	case "enter":
		content := strings.TrimSpace(h.CommentInput.Value())
		if content == "" {
			return nil
		}

		// determine task ID (from selection or cursor)
		taskID := ""
		if h.SelectedTask != nil {
			taskID = h.SelectedTask.ID
		} else if len(h.Tasks) > 0 && h.TaskCursor < len(h.Tasks) {
			taskID = h.Tasks[h.TaskCursor].ID
		} else {
			h.IsAddingComment = false
			return nil
		}

		h.IsAddingComment = false
		h.CommentInput.Reset()
		h.Loading = true
		h.StatusMsg = "Adding comment..."

		return func() tea.Msg {
			comment, err := h.Client.CreateComment(api.CreateCommentRequest{
				TaskID:  taskID,
				Content: content,
			})
			if err != nil {
				return errMsg{err}
			}
			return commentCreatedMsg{comment: comment}
		}

	default:
		var cmd tea.Cmd
		h.CommentInput, cmd = h.CommentInput.Update(msg)
		return cmd
	}
}
