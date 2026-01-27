import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

func (a *App) handleMouseMsg(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Only handle clicks
	if msg.Action != tea.MouseActionPress || msg.Button != tea.MouseButtonLeft {
		return a, nil
	}

	// Skip if in modal views
	if a.currentView == ViewHelp || a.currentView == ViewTaskForm || a.currentView == ViewSearch || a.currentView == ViewTaskDetail {
		return a, nil
	}

	x, y := msg.X, msg.Y

	// Check if click is on tab bar (first 3 lines: border + tabs + border)
	if y <= 2 {
		return a.handleTabClick(x)
	}

	// Handle clicks in main content area
	contentStartY := 3 // After tab bar
	if y >= contentStartY {
		return a.handleContentClick(x, y-contentStartY)
	}

	return a, nil
}

// handleTabClick handles mouse clicks on the tab bar.
func (a *App) handleTabClick(x int) (tea.Model, tea.Cmd) {
	tabs := getTabDefinitions()

	// Determine label style based on available width (same logic as renderTabBar)
	useShortLabels := a.width < 80
	useMinimalLabels := a.width < 50

	// Calculate actual rendered positions for each tab
	currentPos := 2 // Start after TabBar left padding

	for _, t := range tabs {
		var label string
		if useMinimalLabels {
			label = t.icon
		} else if useShortLabels {
			label = fmt.Sprintf("%s %s", t.icon, t.shortName)
		} else {
			label = fmt.Sprintf("%s %s", t.icon, t.name)
		}

		// Render the tab to get its actual width (includes padding from Tab/TabActive style)
		var renderedTab string
		if a.currentTab == t.tab {
			renderedTab = styles.TabActive.Render(label)
		} else {
			renderedTab = styles.Tab.Render(label)
		}

		tabWidth := lipgloss.Width(renderedTab)
		endPos := currentPos + tabWidth

		// Check if click is within this tab
		if x >= currentPos && x < endPos {
			a.currentTab = t.tab
			a.taskCursor = 0
			a.currentLabel = nil

			switch t.tab {
			case TabToday:
				a.currentView = ViewToday
				a.currentProject = nil
				return a, a.filterTodayTasks()
			case TabUpcoming:
				a.currentView = ViewUpcoming
				a.currentProject = nil
				return a, a.filterUpcomingTasks()
			case TabLabels:
				a.currentView = ViewLabels
				a.currentProject = nil
				return a, nil
			case TabCalendar:
				a.currentView = ViewCalendar
				a.currentProject = nil
				a.calendarDate = time.Now()
				a.calendarDay = time.Now().Day()
				return a, a.filterCalendarTasks()
			case TabProjects:
				a.currentView = ViewProject
				a.focusedPane = PaneSidebar
				a.sidebarCursor = 0
				return a, nil
			}
		}

		// Move to next tab position (+1 for space separator between tabs)
		currentPos = endPos + 1
	}

	return a, nil
}

// handleContentClick handles mouse clicks in the content area.
func (a *App) handleContentClick(x, y int) (tea.Model, tea.Cmd) {
	// In Projects tab, check if click is in sidebar
	if a.currentTab == TabProjects {
		sidebarWidth := 25
		if x < sidebarWidth {
			// Click in sidebar
			a.focusedPane = PaneSidebar
			// Calculate which item was clicked (accounting for title + blank line)
			itemIdx := y - 2
			if itemIdx >= 0 && itemIdx < len(a.sidebarItems) {
				// Skip separators
				if a.sidebarItems[itemIdx].Type != "separator" {
					a.sidebarCursor = itemIdx
					// Select the item
					return a.handleSelect()
				}
			}
		} else {
			// Click in main content
			a.focusedPane = PaneMain
			return a.handleTaskClick(y)
		}
	} else {
		// Other tabs - click directly on tasks
		return a.handleTaskClick(y)
	}

	return a, nil
}

// handleTaskClick handles clicking on a task in the task list.
func (a *App) handleTaskClick(y int) (tea.Model, tea.Cmd) {
	// Calculate header offset based on view
	// Default: title (1 line) + blank line (1 line) = 2 lines
	headerOffset := 2

	// Account for scroll indicator if there's content above
	if a.scrollOffset > 0 {
		headerOffset++ // "▲ N more above" takes 1 line
	}

	// In Labels view, might be clicking on a label
	if a.currentView == ViewLabels && a.currentLabel == nil {
		labelsToUse := a.labels
		if len(labelsToUse) == 0 {
			labelsToUse = a.extractLabelsFromTasks()
		}
		// Adjust for scroll offset
		clickedIdx := y - headerOffset + a.scrollOffset
		if clickedIdx >= 0 && clickedIdx < len(labelsToUse) {
			a.taskCursor = clickedIdx
			return a.handleSelect()
		}
		return a, nil
	}

	// For task lists - use viewportLines mapping if available
	// viewportLines maps viewport line number to task index (-1 for headers)
	viewportLine := y - headerOffset + a.scrollOffset

	if len(a.viewportLines) > 0 {
		// Use the viewport line mapping for accurate click handling
		if viewportLine >= 0 && viewportLine < len(a.viewportLines) {
			taskIndex := a.viewportLines[viewportLine]
			if taskIndex >= 0 {
				// Find the display position (cursor) for this task index
				for displayPos, idx := range a.taskOrderedIndices {
					if idx == taskIndex {
						a.taskCursor = displayPos
						return a, nil
					}
				}
			}
		}
	} else {
		// Fallback for simple lists without section headers
		if viewportLine >= 0 && viewportLine < len(a.tasks) {
			a.taskCursor = viewportLine
			return a, nil
		}
	}

	return a, nil
}

// switchToTab switches to a specific tab.
func (a *App) switchToTab(tab Tab) (tea.Model, tea.Cmd) {
	// Reset detail panel state when switching tabs
	if a.showDetailPanel {
		a.showDetailPanel = false
		a.selectedTask = nil
		a.comments = nil
		a.detailComp.Hide()
	}

	// Don't switch if in modal views

	if a.currentView == ViewHelp || a.currentView == ViewTaskForm || a.currentView == ViewSearch || a.currentView == ViewTaskDetail {
		return a, nil
	}

	a.currentTab = tab
	a.taskCursor = 0
	a.currentLabel = nil

	switch tab {
	case TabToday:
		a.currentView = ViewToday
		a.currentProject = nil
		a.focusedPane = PaneMain
		return a, a.filterTodayTasks()
	case TabUpcoming:
		a.currentView = ViewUpcoming
		a.currentProject = nil
		a.focusedPane = PaneMain
		return a, a.filterUpcomingTasks()
	case TabLabels:
		a.currentView = ViewLabels
		a.currentProject = nil
		a.focusedPane = PaneMain
		return a, nil
	case TabCalendar:
		a.currentView = ViewCalendar
		a.currentProject = nil
		a.focusedPane = PaneMain
		a.calendarDate = time.Now()
		a.calendarDay = time.Now().Day()
		return a, a.filterCalendarTasks()
	case TabProjects:
		a.currentView = ViewProject
		a.focusedPane = PaneSidebar
		a.sidebarCursor = 0
		return a, nil
	}

	return a, nil
}

// handleKeyMsg processes keyboard input.
func (a *App) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Only ctrl+c is truly global
	if msg.String() == "ctrl+c" {
		return a, tea.Quit
	}

	// If we're in help view, any key goes back
	if a.currentView == ViewHelp {
		a.currentView = a.previousView
		return a, nil
	}

	// Route key messages based on current view - BEFORE tab switching
	// This allows forms to capture number keys for text input
	switch a.currentView {
	case ViewTaskForm:
		return a.handleFormKeyMsg(msg)
	case ViewSearch:
		return a.handleSearchKeyMsg(msg)
	}

	// Handle all input/dialog states BEFORE tab switching
	// This prevents number keys from switching views during text entry

	// Project state handling
	if a.isCreatingProject {
		return a.handleProjectInputKeyMsg(msg)
	}
	if a.isEditingProject {
		return a.handleProjectEditKeyMsg(msg)
	}
	if a.confirmDeleteProject {
		return a.handleDeleteConfirmKeyMsg(msg)
	}

	// Label state handling
	if a.isCreatingLabel {
		return a.handleLabelInputKeyMsg(msg)
	}
	if a.isEditingLabel {
		return a.handleLabelEditKeyMsg(msg)
	}
	if a.confirmDeleteLabel {
		return a.handleLabelDeleteConfirmKeyMsg(msg)
	}

	// Section state handling
	if a.isCreatingSection {
		return a.handleSectionInputKeyMsg(msg)
	}
	if a.isEditingSection {
		return a.handleSectionEditKeyMsg(msg)
	}
	if a.confirmDeleteSection {
		return a.handleSectionDeleteConfirmKeyMsg(msg)
	}

	// Subtask creation handling
	if a.isCreatingSubtask {
		return a.handleSubtaskInputKeyMsg(msg)
	}

	// Comment input handling
	if a.isAddingComment {
		return a.handleCommentInputKeyMsg(msg)
	}

	// Move task handling
	if a.isMovingTask {
		return a.handleMoveTaskKeyMsg(msg)
	}

	// Tab switching with number keys (1-5) - only when not in form/input modes
	// Tab switching with number keys (1-5) and letters - only when not in form/input modes
	switch msg.String() {
	case "1":
		return a.switchToTab(TabToday)
	case "2":
		return a.switchToTab(TabUpcoming)
	case "3":
		return a.switchToTab(TabLabels)
	case "4":
		return a.switchToTab(TabCalendar)
	case "5":
		return a.switchToTab(TabProjects)
	case "t", "T":
		return a.switchToTab(TabToday)
	case "u", "U":
		return a.switchToTab(TabUpcoming)
	case "p", "P":
		return a.switchToTab(TabProjects)
	case "c", "C":
		return a.switchToTab(TabCalendar)
	// 'l' is excluded here to preserve navigation in Calendar/Projects, handled in "right" action
	case "L":
		return a.switchToTab(TabLabels)
	}

	// Sections view routing
	if a.currentView == ViewSections {
		return a.handleSectionsKeyMsg(msg)
	}

	// If we're in calendar view, handle calendar-specific keys
	if a.currentView == ViewCalendar && a.focusedPane == PaneMain {
		return a.handleCalendarKeyMsg(msg)
	}

	// Process key through keymap
	action, consumed := a.keyState.HandleKey(msg, a.keymap)
	if !consumed {
		return a, nil
	}

	// Handle actions
	switch action {
	case "quit":
		return a, tea.Quit
	case "help":
		a.previousView = a.currentView
		a.currentView = ViewHelp
		return a, nil
	case "refresh":
		return a, func() tea.Msg { return refreshMsg{} }
	case "up":
		a.moveCursor(-1)
	case "down":
		a.moveCursor(1)
	case "top":
		a.moveCursorTo(0)
	case "bottom":
		a.moveCursorToEnd()
	case "half_up":
		a.moveCursor(-10)
	case "half_down":
		a.moveCursor(10)
	case "left":
		// h key - move to sidebar in Projects tab
		if a.currentTab == TabProjects && a.focusedPane == PaneMain {
			a.focusedPane = PaneSidebar
		}
	case "right":
		// l key - move to main pane in Projects tab
		if a.currentTab == TabProjects && a.focusedPane == PaneSidebar {
			a.focusedPane = PaneMain
		} else if a.currentView != ViewCalendar && msg.String() == "l" {
			// If not navigating projects or calendar, 'l' switches to Labels
			return a.switchToTab(TabLabels)
		}
	case "switch_pane":
		a.switchPane()
	case "select":
		return a.handleSelect()
	case "back":
		return a.handleBack()
	case "complete":
		return a.handleComplete()
	case "delete":
		return a.handleDelete()
	case "add":
		return a.handleAdd()
	case "edit":
		return a.handleEdit()
	case "search":
		return a.handleSearch()
	case "priority1", "priority2", "priority3", "priority4":
		return a.handlePriority(action)
	case "due_today":
		return a.handleDueToday()
	case "due_tomorrow":
		return a.handleDueTomorrow()
	case "new_project":
		// 'n' key creates project or label depending on current tab
		if a.currentTab == TabProjects {
			return a.handleNewProject()
		} else if a.currentTab == TabLabels {
			return a.handleNewLabel()
		}
	// Tab shortcuts (Shift + letter)
	case "tab_today":
		return a.switchToTab(TabToday)
	case "tab_upcoming":
		return a.switchToTab(TabUpcoming)
	case "tab_projects":
		return a.switchToTab(TabProjects)
	case "tab_labels":
		return a.switchToTab(TabLabels)
	case "tab_calendar":
		return a.switchToTab(TabCalendar)
	case "toggle_hints":
		a.showHints = !a.showHints
	case "add_subtask":
		return a.handleAddSubtask()
	case "undo":
		return a.handleUndo()
	case "manage_sections":
		if a.currentTab == TabProjects && len(a.projects) > 0 {
			a.previousView = a.currentView
			a.currentView = ViewSections
			a.taskCursor = 0 // use cursor for sections
			return a, nil
		}
	case "move_task":
		if a.focusedPane == PaneMain && len(a.tasks) > 0 && len(a.sections) > 0 {
			a.isMovingTask = true
			a.moveSectionCursor = 0
			return a, nil
		}
	case "new_section":
		// Allow creating sections in project view when a project is selected
		if a.currentTab == TabProjects && a.currentProject != nil {
			a.sectionInput = textinput.New()
			a.sectionInput.Placeholder = "Enter section name..."
			a.sectionInput.CharLimit = 100
			a.sectionInput.Width = 40
			a.sectionInput.Focus()
			a.isCreatingSection = true
			return a, nil
		}
	case "move_section":
		// Redirect to section management view
		if a.currentTab == TabProjects && a.currentProject != nil && len(a.sections) > 1 {
			a.statusMsg = "Use 'S' to manage sections - select with Space, reorder with Shift+j/k"
			return a, nil
		}
	// Note: 'C' key is not in keymap yet, handling manually or adding to keymap
	// Actually, I should add 'C' to keymap or handle via raw key manually if I want.
	// But let's assume I added 'C' -> 'add_comment' in keymap (I didn't yet).
	// I'll add the case here assuming I will update keymap.go next.
	case "add_comment":
		if a.selectedTask != nil {
			a.isAddingComment = true
			a.commentInput = textinput.New()
			a.commentInput.Placeholder = "Write a comment..."
			a.commentInput.Focus()
			a.commentInput.Width = 50
			return a, nil
		}
	case "toggle_select":
		return a.handleToggleSelect()
	case "copy":
		return a.handleCopy()
	}

	return a, nil
}

// handleCalendarKeyMsg handles keyboard input when calendar view is active.
func (a *App) handleSelect() (tea.Model, tea.Cmd) {
	// Handle sidebar selection (only in Projects tab)
	if a.currentTab == TabProjects && a.focusedPane == PaneSidebar {
		if a.sidebarCursor >= len(a.sidebarItems) {
			return a, nil
		}

		item := a.sidebarItems[a.sidebarCursor]
		if item.Type == "separator" {
			return a, nil
		}

		a.focusedPane = PaneMain
		a.taskCursor = 0

		// Find project by ID
		for i := range a.projects {
			if a.projects[i].ID == item.ID {
				a.currentProject = &a.projects[i]
				return a, a.filterProjectTasks(a.currentProject.ID)
			}
		}
		return a, nil
	}

	// Handle main pane selection based on current view
	switch a.currentView {
	case ViewLabels:
		// Select label to filter tasks
		if a.currentLabel == nil {
			labelsToUse := a.labels
			if len(labelsToUse) == 0 {
				labelsToUse = a.extractLabelsFromTasks()
			}
			if a.taskCursor < len(labelsToUse) {
				a.currentLabel = &labelsToUse[a.taskCursor]
				a.taskCursor = 0 // Reset cursor for task list
				return a, a.filterLabelTasks(a.currentLabel.Name)
			}
		} else {
			// Viewing label tasks - select task for detail
			if a.taskCursor < len(a.tasks) {
				a.selectedTask = &a.tasks[a.taskCursor]
				a.showDetailPanel = true
				return a, a.loadTaskComments()
			}
		}
	case ViewCalendar:
		// Selection handled by calendar navigation
		return a, nil
	default:
		// Select task for detail view
		// First check if we have tasks at all
		if len(a.tasks) == 0 {
			return a, nil
		}

		// Try taskOrderedIndices first (for views with sections/groups)
		if len(a.taskOrderedIndices) > 0 && a.taskCursor >= 0 && a.taskCursor < len(a.taskOrderedIndices) {
			taskIndex := a.taskOrderedIndices[a.taskCursor]

			// Skip if cursor is on empty section header (taskIndex <= -100)
			if taskIndex <= -100 {
				return a, nil
			}

			if taskIndex >= 0 && taskIndex < len(a.tasks) {
				taskCopy := new(api.Task)
				*taskCopy = a.tasks[taskIndex]
				a.selectedTask = taskCopy
				a.showDetailPanel = true
				return a, a.loadTaskComments()
			}
		}

		// Fallback: use taskCursor directly to index a.tasks
		// This works for views that don't pre-populate taskOrderedIndices
		if a.taskCursor >= 0 && a.taskCursor < len(a.tasks) {
			taskCopy := new(api.Task)
			*taskCopy = a.tasks[a.taskCursor]
			a.selectedTask = taskCopy
			a.showDetailPanel = true
			return a, a.loadTaskComments()
		}
	}
	return a, nil
}

// handleBack handles the Escape key.
func (a *App) handleBack() (tea.Model, tea.Cmd) {
	// Close detail panel if open
	if a.showDetailPanel {
		a.showDetailPanel = false
		a.selectedTask = nil
		a.comments = nil
		return a, nil
	}

	switch a.currentView {
	case ViewTaskDetail:
		a.currentView = a.previousView
		a.selectedTask = nil
		a.comments = nil
	case ViewCalendarDay:
		// Go back to calendar view
		a.currentView = ViewCalendar
		a.taskCursor = 0
		// Reload all tasks for calendar display
		return a, a.loadAllTasks()
	case ViewProject:
		// In Projects tab, just clear selection
		if a.currentTab == TabProjects {
			a.currentProject = nil
			a.tasks = nil
			a.focusedPane = PaneSidebar
		}
	case ViewLabels:
		if a.currentLabel != nil {
			// Go back to labels list
			a.currentLabel = nil
			a.tasks = nil
			a.taskCursor = 0
		}
	case ViewTaskForm:
		a.currentView = a.previousView
	}
	return a, nil
}

// handleComplete toggles task completion.
