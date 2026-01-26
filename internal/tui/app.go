// Package tui provides the terminal user interface for Todoist.
package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/config"
	"github.com/hy4ri/todoist-tui/internal/tui/components"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// View represents the current view/screen.
type View int

const (
	ViewToday View = iota
	ViewUpcoming
	ViewLabels
	ViewCalendar
	ViewCalendarDay // Day detail view from calendar
	ViewProject
	ViewTaskDetail
	ViewTaskForm
	ViewSearch
	ViewHelp
	ViewSections
)

// Tab represents a top-level tab.
type Tab int

const (
	TabToday Tab = iota
	TabUpcoming
	TabLabels
	TabCalendar
	TabProjects
)

// SidebarItem represents an item in the sidebar (special views or projects).
type SidebarItem struct {
	Type       string // "special", "separator", "project"
	ID         string // View name for special, project ID for projects
	Name       string
	Icon       string
	IsFavorite bool
	ParentID   *string
}

// Pane represents which pane is currently focused (only used in Projects tab).
type Pane int

const (
	PaneSidebar Pane = iota
	PaneMain
)

// CalendarViewMode represents the calendar display mode.
type CalendarViewMode int

const (
	CalendarViewCompact  CalendarViewMode = iota // Small grid view
	CalendarViewExpanded                         // Grid with task names in cells
)

// LastAction represents an undoable action.
type LastAction struct {
	Type   string // "complete", "uncomplete"
	TaskID string
}

// App is the main Bubble Tea model for the application.
type App struct {
	// Dependencies
	client *api.Client
	config *config.Config

	// View state
	currentView  View
	previousView View
	currentTab   Tab
	focusedPane  Pane // Only used in Projects tab

	// Data
	projects       []api.Project
	tasks          []api.Task
	allTasks       []api.Task // All tasks for upcoming/calendar views
	sections       []api.Section
	labels         []api.Label
	comments       []api.Comment // Comments for selected task
	selectedTask   *api.Task
	currentProject *api.Project
	currentLabel   *api.Label

	// Sidebar items (only shown in Projects tab)
	sidebarItems  []SidebarItem
	sidebarCursor int

	// List state
	projectCursor int
	taskCursor    int
	scrollOffset  int // Track scroll position for click handling

	// Calendar state
	calendarDate     time.Time        // Currently viewed month
	calendarDay      int              // Selected day (1-31)
	calendarViewMode CalendarViewMode // Compact or Expanded view

	// UI state
	loading         bool
	err             error
	statusMsg       string
	width           int
	height          int
	showHints       bool        // Toggle visibility of keyboard shortcuts
	lastAction      *LastAction // Last undoable action
	showDetailPanel bool        // Show task detail panel on right

	// Components
	spinner  spinner.Model
	keyState KeyState
	keymap   Keymap

	// UI Components (new)
	sidebarComp *components.SidebarModel
	detailComp  *components.DetailModel
	helpComp    *components.HelpModel

	// Form state (for add/edit)
	taskForm *TaskForm

	// Search state
	searchQuery   string
	searchInput   textinput.Model
	searchResults []api.Task
	isSearching   bool

	// New project state
	projectInput         textinput.Model
	isCreatingProject    bool
	isEditingProject     bool
	editingProject       *api.Project
	confirmDeleteProject bool

	// New label state
	labelInput         textinput.Model
	isCreatingLabel    bool
	isEditingLabel     bool
	editingLabel       *api.Label
	confirmDeleteLabel bool

	// Section state
	sectionInput         textinput.Model
	isCreatingSection    bool
	isEditingSection     bool
	editingSection       *api.Section
	confirmDeleteSection bool

	// Subtask creation state
	subtaskInput      textinput.Model
	isCreatingSubtask bool
	parentTaskID      string

	// Viewport for scrollable task lists
	taskViewport  viewport.Model
	viewportReady bool

	// Move task state
	isMovingTask      bool
	moveSectionCursor int

	// Comment state
	commentInput    textinput.Model
	isAddingComment bool

	viewportContent    string   // Current content in viewport
	viewportLines      []int    // Maps viewport line number to task index (-1 for headers)
	taskOrderedIndices []int    // Maps display order (cursor position) to a.tasks index
	viewportSections   []string // Maps viewport line number to section ID

	// Selection state
	selectedTaskIDs map[string]bool // Task IDs that are selected for bulk operations

	// Cursor restoration state
	restoreCursorToTaskID string // Task ID to restore cursor to after refresh
}

// NewApp creates a new App instance.
// NewApp creates a new App instance.
func NewApp(client *api.Client, cfg *config.Config, initialView string) *App {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = styles.Spinner

	searchInput := textinput.New()
	searchInput.Placeholder = "Search tasks..."
	searchInput.CharLimit = 100
	searchInput.Width = 40

	// Set calendar view mode from config (default to expanded)
	calViewMode := CalendarViewExpanded
	if cfg.UI.CalendarDefaultView == "compact" {
		calViewMode = CalendarViewCompact
	}

	app := &App{
		client:           client,
		config:           cfg,
		currentView:      ViewToday,
		currentTab:       TabToday,
		focusedPane:      PaneMain,
		spinner:          s,
		keymap:           DefaultKeymap(),
		searchInput:      searchInput,
		calendarDate:     time.Now(),
		calendarDay:      time.Now().Day(),
		calendarViewMode: calViewMode,
		loading:          true,
		showHints:        true,
		selectedTaskIDs:  make(map[string]bool),
		// Initialize UI components
		sidebarComp: components.NewSidebar(),
		detailComp:  components.NewDetail(),
		helpComp:    components.NewHelp(),
	}

	// Handle initial view override
	switch initialView {
	case "upcoming":
		app.currentView = ViewUpcoming
		app.currentTab = TabUpcoming
	case "projects":
		app.currentView = ViewProject
		app.currentTab = TabProjects
		app.focusedPane = PaneSidebar
	case "calendar":
		app.currentView = ViewCalendar
		app.currentTab = TabCalendar
	case "labels":
		app.currentView = ViewLabels
		app.currentTab = TabLabels
	}

	return app
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.spinner.Tick,
		a.loadInitialData(),
	)
}

// loadInitialData loads projects and today's tasks.
func (a *App) loadInitialData() tea.Cmd {
	return func() tea.Msg {
		// Load projects
		projects, err := a.client.GetProjects()
		if err != nil {
			return errMsg{err}
		}

		// Load today's tasks (including overdue) using filter endpoint
		tasks, err := a.client.GetTasksByFilter("today | overdue")
		if err != nil {
			return errMsg{err}
		}

		// Load all tasks for upcoming/calendar views
		allTasks, err := a.client.GetTasks(api.TaskFilter{})
		if err != nil {
			return errMsg{err}
		}

		// Load labels
		labels, err := a.client.GetLabels()
		if err != nil {
			return errMsg{err}
		}

		return dataLoadedMsg{
			projects: projects,
			tasks:    tasks,
			allTasks: allTasks,
			labels:   labels,
		}
	}
}

// Message types
type errMsg struct{ err error }
type statusMsg struct{ msg string }
type dataLoadedMsg struct {
	projects []api.Project
	tasks    []api.Task
	allTasks []api.Task
	sections []api.Section
	labels   []api.Label
}
type taskUpdatedMsg struct{ task *api.Task }
type taskDeletedMsg struct{ id string }
type taskCompletedMsg struct{ id string }
type taskCreatedMsg struct{}
type projectCreatedMsg struct{ project *api.Project }
type projectUpdatedMsg struct{ project *api.Project }
type projectDeletedMsg struct{ id string }
type labelCreatedMsg struct{ label *api.Label }
type labelUpdatedMsg struct{ label *api.Label }
type labelDeletedMsg struct{ id string }
type sectionCreatedMsg struct{ section *api.Section }
type sectionUpdatedMsg struct{ section *api.Section }
type sectionDeletedMsg struct{ id string }
type commentCreatedMsg struct{ comment *api.Comment }
type subtaskCreatedMsg struct{}
type undoCompletedMsg struct{}
type searchRefreshMsg struct{}
type refreshMsg struct{}
type commentsLoadedMsg struct{ comments []api.Comment }

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKeyMsg(msg)

	case tea.MouseMsg:
		return a.handleMouseMsg(msg)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Initialize or update viewport dimensions
		// Reserve space for tab bar (~3 lines), status bar (2 lines), borders (2 lines), title (2 lines)
		vpHeight := msg.Height - 9
		if vpHeight < 5 {
			vpHeight = 5
		}
		vpWidth := msg.Width - 4
		if vpWidth < 20 {
			vpWidth = 20
		}
		if !a.viewportReady {
			a.taskViewport = viewport.New(vpWidth, vpHeight)
			a.taskViewport.Style = lipgloss.NewStyle()
			a.taskViewport.MouseWheelEnabled = true
			a.viewportReady = true
		} else {
			a.taskViewport.Width = vpWidth
			a.taskViewport.Height = vpHeight
		}
		return a, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	case errMsg:
		a.loading = false
		a.err = msg.err
		return a, nil

	case statusMsg:
		a.statusMsg = msg.msg
		return a, nil

	case dataLoadedMsg:
		a.loading = false
		// Only update fields that are non-nil/non-empty to avoid overwriting
		if len(msg.projects) > 0 {
			a.projects = msg.projects
			a.buildSidebarItems()
			// Sync sidebar component
			a.sidebarComp.SetProjects(msg.projects)
		}
		if msg.tasks != nil {
			a.tasks = msg.tasks
			a.sortTasks()
		}
		if len(msg.labels) > 0 {
			a.labels = msg.labels
		}
		if len(msg.allTasks) > 0 {
			a.allTasks = msg.allTasks
		}
		if len(msg.sections) > 0 {
			a.sections = msg.sections
			// Sort sections by SectionOrder
			sort.Slice(a.sections, func(i, j int) bool {
				return a.sections[i].SectionOrder < a.sections[j].SectionOrder
			})
		}

		// Restore cursor position if we have a task ID to restore to
		if a.restoreCursorToTaskID != "" {
			for i, task := range a.tasks {
				if task.ID == a.restoreCursorToTaskID {
					a.taskCursor = i
					break
				}
			}
			a.restoreCursorToTaskID = "" // Clear after restoring
		}

		return a, nil

	case taskUpdatedMsg:
		a.loading = false
		// Refresh the task list
		return a, a.refreshTasks()

	case taskDeletedMsg:
		a.loading = false
		a.statusMsg = "Task deleted"
		// Keep selections after delete (remaining tasks stay visible)
		return a, a.refreshTasks()

	case taskCompletedMsg:
		a.loading = false
		// Keep selections after complete (tasks remain visible)
		return a, a.refreshTasks()

	case taskCreatedMsg:
		a.loading = false
		a.statusMsg = "Task saved"
		a.currentView = a.previousView
		a.taskForm = nil
		return a, a.refreshTasks()

	case projectCreatedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Created project: %s", msg.project.Name)
		// Reload projects
		return a, a.loadProjects()

	case projectUpdatedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Updated project: %s", msg.project.Name)
		// Reload projects
		return a, a.loadProjects()

	case projectDeletedMsg:
		a.loading = false
		a.statusMsg = "Project deleted"
		a.sidebarCursor = 0
		// Reload projects and switch to first project
		return a, a.loadProjects()

	case labelCreatedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Created label: %s", msg.label.Name)
		// Reload labels
		return a, a.loadLabels()

	case labelUpdatedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Updated label: %s", msg.label.Name)
		return a, a.loadLabels()

	case labelDeletedMsg:
		a.loading = false
		a.statusMsg = "Label deleted"
		a.taskCursor = 0
		return a, a.loadLabels()

	case sectionCreatedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Created section: %s", msg.section.Name)
		// Reload current project
		if a.sidebarCursor < len(a.sidebarItems) {
			return a, a.loadProjectTasks(a.sidebarItems[a.sidebarCursor].ID)
		}
		return a, nil

	case sectionUpdatedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Updated section: %s", msg.section.Name)
		if a.sidebarCursor < len(a.sidebarItems) {
			return a, a.loadProjectTasks(a.sidebarItems[a.sidebarCursor].ID)
		}
		return a, nil

	case sectionDeletedMsg:
		a.loading = false
		a.statusMsg = "Section deleted"
		if a.sidebarCursor < len(a.sidebarItems) {
			return a, a.loadProjectTasks(a.sidebarItems[a.sidebarCursor].ID)
		}
		return a, nil

	case commentCreatedMsg:
		a.loading = false
		a.statusMsg = "Comment added"
		return a, a.loadTaskComments()

	case subtaskCreatedMsg:
		a.loading = false
		a.statusMsg = "Subtask created"
		// Reload current view to show subtask
		return a, func() tea.Msg { return refreshMsg{} }

	case undoCompletedMsg:
		a.loading = false
		a.statusMsg = "Undo successful"
		// Reload current view
		return a, func() tea.Msg { return refreshMsg{} }

	case searchRefreshMsg:
		a.loading = false
		a.statusMsg = "Task updated"
		return a, a.refreshSearchResults()

	case refreshMsg:
		// Store current task ID to restore cursor position after reload
		if len(a.tasks) > 0 && a.taskCursor >= 0 && a.taskCursor < len(a.tasks) {
			a.restoreCursorToTaskID = a.tasks[a.taskCursor].ID
		}
		a.loading = true

		if a.currentView == ViewSearch {
			return a, a.refreshSearchResults()
		}

		switch a.currentTab {
		case TabProjects:
			if a.currentProject != nil {
				// Reload current project
				return a, a.loadProjectTasks(a.currentProject.ID)
			}
			// Just reload project list
			return a, a.loadProjects()
		case TabUpcoming:
			return a, a.loadUpcomingTasks()
		case TabCalendar:
			// Preserve calendar date and reload all tasks
			return a, a.loadAllTasks()
		case TabLabels:
			if a.currentLabel != nil {
				return a, a.loadLabelTasks(a.currentLabel.Name)
			}
			return a, a.loadLabels()
		default:
			// TabToday or fallback
			return a, a.loadTodayTasks()
		}

	case commentsLoadedMsg:
		a.comments = msg.comments
		return a, nil
	}

	return a, nil
}

// handleMouseMsg processes mouse input.
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
				return a, a.loadTodayTasks()
			case TabUpcoming:
				a.currentView = ViewUpcoming
				a.currentProject = nil
				return a, a.loadUpcomingTasks()
			case TabLabels:
				a.currentView = ViewLabels
				a.currentProject = nil
				return a, nil
			case TabCalendar:
				a.currentView = ViewCalendar
				a.currentProject = nil
				a.calendarDate = time.Now()
				a.calendarDay = time.Now().Day()
				return a, a.loadAllTasks()
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
		return a, a.loadTodayTasks()
	case TabUpcoming:
		a.currentView = ViewUpcoming
		a.currentProject = nil
		a.focusedPane = PaneMain
		return a, a.loadUpcomingTasks()
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
		return a, a.loadAllTasks()
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
		} else if a.currentView != ViewCalendar {
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
func (a *App) sortTasks() {
	sort.Slice(a.tasks, func(i, j int) bool {
		ti, tj := a.tasks[i], a.tasks[j]

		// Get due dates (tasks without due dates sort to end)
		hasDueI := ti.Due != nil
		hasDueJ := tj.Due != nil

		if hasDueI && !hasDueJ {
			return true // i has due date, j doesn't -> i first
		}
		if !hasDueI && hasDueJ {
			return false // j has due date, i doesn't -> j first
		}

		// Both have due dates - compare by datetime/date
		if hasDueI && hasDueJ {
			// Use datetime if available, else use date
			dateI := ti.Due.Date
			dateJ := tj.Due.Date
			if ti.Due.Datetime != nil {
				dateI = *ti.Due.Datetime
			}
			if tj.Due.Datetime != nil {
				dateJ = *tj.Due.Datetime
			}

			if dateI != dateJ {
				return dateI < dateJ // Earlier dates first
			}
		}

		// Same due date or both no due date - sort by priority (higher = P1 = 4)
		return ti.Priority > tj.Priority
	})
}

// handleSelect handles the Enter key.
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
				return a, a.loadProjectTasks(a.currentProject.ID)
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
				return a, a.loadLabelTasks(a.currentLabel.Name)
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
func (a *App) handleComplete() (tea.Model, tea.Cmd) {
	// In Projects tab, only allow in main pane
	if a.currentTab == TabProjects && a.focusedPane != PaneMain {
		return a, nil
	}

	if len(a.tasks) == 0 {
		return a, nil
	}

	// In Labels view without a selected label, we're showing label list
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	// If there are selected tasks, complete/uncomplete all of them
	if len(a.selectedTaskIDs) > 0 {
		a.loading = true
		tasksToComplete := make([]api.Task, 0)
		for _, task := range a.tasks {
			if a.selectedTaskIDs[task.ID] {
				tasksToComplete = append(tasksToComplete, task)
			}
		}

		return a, func() tea.Msg {
			// Use channels and goroutines for concurrent API calls
			type result struct {
				success bool
			}

			results := make(chan result, len(tasksToComplete))

			// Launch concurrent API calls
			for _, task := range tasksToComplete {
				go func(t api.Task) {
					var err error
					if t.Checked {
						err = a.client.ReopenTask(t.ID)
					} else {
						err = a.client.CloseTask(t.ID)
					}
					results <- result{success: err == nil}
				}(task)
			}

			// Collect results
			completed := 0
			failed := 0
			for i := 0; i < len(tasksToComplete); i++ {
				res := <-results
				if res.success {
					completed++
				} else {
					failed++
				}
			}

			if failed > 0 {
				return statusMsg{msg: fmt.Sprintf("Completed %d tasks, %d failed", completed, failed)}
			}
			return statusMsg{msg: fmt.Sprintf("Completed %d tasks", completed)}
		}
	}

	// Get the correct task using ordered indices mapping
	var task *api.Task
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex := a.taskOrderedIndices[a.taskCursor]
		// Skip empty section headers (negative indices <= -100)
		if taskIndex >= 0 && taskIndex < len(a.tasks) {
			task = &a.tasks[taskIndex]
		}
	} else if a.taskCursor < len(a.tasks) {
		// Fallback for views that don't use ordered indices
		task = &a.tasks[a.taskCursor]
	}

	if task == nil {
		return a, nil
	}

	// Store last action for undo
	if task.Checked {
		a.lastAction = &LastAction{Type: "uncomplete", TaskID: task.ID}
	} else {
		a.lastAction = &LastAction{Type: "complete", TaskID: task.ID}
	}

	a.loading = true

	return a, func() tea.Msg {
		var err error
		if task.Checked {
			err = a.client.ReopenTask(task.ID)
		} else {
			err = a.client.CloseTask(task.ID)
		}
		if err != nil {
			return errMsg{err}
		}
		return taskCompletedMsg{id: task.ID}
	}
}

// handleUndo reverses the last undoable action.
func (a *App) handleUndo() (tea.Model, tea.Cmd) {
	if a.lastAction == nil {
		a.statusMsg = "Nothing to undo"
		return a, nil
	}

	action := a.lastAction
	a.lastAction = nil
	a.loading = true

	return a, func() tea.Msg {
		var err error
		switch action.Type {
		case "complete":
			// Was completed, so reopen it
			err = a.client.ReopenTask(action.TaskID)
		case "uncomplete":
			// Was reopened, so close it
			err = a.client.CloseTask(action.TaskID)
		default:
			return statusMsg{msg: "Unknown action"}
		}
		if err != nil {
			return errMsg{err}
		}
		return undoCompletedMsg{}
	}
}

// handleToggleSelect toggles selection of the task under the cursor.
func (a *App) handleToggleSelect() (tea.Model, tea.Cmd) {
	// Only allow in main pane with tasks
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		return a, nil
	}

	// Don't allow selection in label list view
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	// Get the task at cursor
	var task *api.Task
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex := a.taskOrderedIndices[a.taskCursor]
		// Skip placeholders (negative indices < -100)
		if taskIndex >= 0 && taskIndex < len(a.tasks) {
			task = &a.tasks[taskIndex]
		}
	} else if a.taskCursor < len(a.tasks) {
		task = &a.tasks[a.taskCursor]
	}

	if task == nil {
		return a, nil
	}

	// Toggle selection
	if a.selectedTaskIDs[task.ID] {
		delete(a.selectedTaskIDs, task.ID)
		a.statusMsg = fmt.Sprintf("Deselected (%d selected)", len(a.selectedTaskIDs))
	} else {
		a.selectedTaskIDs[task.ID] = true
		a.statusMsg = fmt.Sprintf("Selected (%d tasks)", len(a.selectedTaskIDs))
	}

	return a, nil
}

// handleCopy copies task content to clipboard.
func (a *App) handleCopy() (tea.Model, tea.Cmd) {
	// Only allow in main pane with tasks
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		return a, nil
	}

	// Don't allow in label list view
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	// If there are selected tasks, copy all of them
	if len(a.selectedTaskIDs) > 0 {
		var selectedContents []string
		for _, task := range a.tasks {
			if a.selectedTaskIDs[task.ID] {
				selectedContents = append(selectedContents, task.Content)
			}
		}

		if len(selectedContents) == 0 {
			return a, nil
		}

		// Clear selections after copy
		a.selectedTaskIDs = make(map[string]bool)

		return a, func() tea.Msg {
			// Join all selected task contents with newlines
			content := strings.Join(selectedContents, "\n")
			err := clipboard.WriteAll(content)
			if err != nil {
				return statusMsg{msg: "Failed to copy: " + err.Error()}
			}
			return statusMsg{msg: fmt.Sprintf("Copied %d tasks", len(selectedContents))}
		}
	}

	// Otherwise, copy just the task at cursor
	var task *api.Task
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex := a.taskOrderedIndices[a.taskCursor]
		if taskIndex >= 0 && taskIndex < len(a.tasks) {
			task = &a.tasks[taskIndex]
		}
	} else if a.taskCursor < len(a.tasks) {
		task = &a.tasks[a.taskCursor]
	}

	if task == nil {
		return a, nil
	}

	return a, func() tea.Msg {
		// Copy to clipboard
		err := clipboard.WriteAll(task.Content)
		if err != nil {
			return statusMsg{msg: "Failed to copy: " + err.Error()}
		}
		return statusMsg{msg: "Copied: " + task.Content}
	}
}

// handleDelete deletes the selected task or project.
func (a *App) handleDelete() (tea.Model, tea.Cmd) {
	// Handle project deletion when sidebar is focused
	if a.currentTab == TabProjects && a.focusedPane == PaneSidebar {
		if a.sidebarCursor >= len(a.sidebarItems) {
			return a, nil
		}
		item := a.sidebarItems[a.sidebarCursor]
		if item.Type != "project" {
			return a, nil
		}
		// Find the project
		for i := range a.projects {
			if a.projects[i].ID == item.ID {
				// Don't allow deleting inbox
				if a.projects[i].InboxProject {
					a.statusMsg = "Cannot delete Inbox project"
					return a, nil
				}
				a.editingProject = &a.projects[i]
				a.confirmDeleteProject = true
				return a, nil
			}
		}
		return a, nil
	}

	// Handle task deletion
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		// Handle label deletion when viewing label list
		if a.currentTab == TabLabels && a.currentLabel == nil {
			if a.taskCursor < len(a.labels) {
				a.editingLabel = &a.labels[a.taskCursor]
				a.confirmDeleteLabel = true
				return a, nil
			}
		}
		return a, nil
	}

	// Guard: Don't delete task when viewing label list (no task selected)
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	// If there are selected tasks, delete all of them
	if len(a.selectedTaskIDs) > 0 {
		a.loading = true
		tasksToDelete := make([]api.Task, 0)
		for _, task := range a.tasks {
			if a.selectedTaskIDs[task.ID] {
				tasksToDelete = append(tasksToDelete, task)
			}
		}

		return a, func() tea.Msg {
			// Use channels and goroutines for concurrent API calls
			type result struct {
				success bool
			}

			results := make(chan result, len(tasksToDelete))

			// Launch concurrent API calls
			for _, task := range tasksToDelete {
				go func(t api.Task) {
					err := a.client.DeleteTask(t.ID)
					results <- result{success: err == nil}
				}(task)
			}

			// Collect results
			deleted := 0
			failed := 0
			for i := 0; i < len(tasksToDelete); i++ {
				res := <-results
				if res.success {
					deleted++
				} else {
					failed++
				}
			}

			if failed > 0 {
				return statusMsg{msg: fmt.Sprintf("Deleted %d tasks, %d failed", deleted, failed)}
			}
			return statusMsg{msg: fmt.Sprintf("Deleted %d tasks", deleted)}
		}
	}

	// Use ordered indices if available
	taskIndex := a.taskCursor
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex = a.taskOrderedIndices[a.taskCursor]
	}
	if taskIndex < 0 || taskIndex >= len(a.tasks) {
		return a, nil
	}

	task := &a.tasks[taskIndex]
	a.loading = true

	return a, func() tea.Msg {
		err := a.client.DeleteTask(task.ID)
		if err != nil {
			return errMsg{err}
		}
		return taskDeletedMsg{id: task.ID}
	}
}

// handleAdd opens the add task form.
func (a *App) handleAdd() (tea.Model, tea.Cmd) {
	a.previousView = a.currentView
	a.currentView = ViewTaskForm
	a.taskForm = NewTaskForm(a.projects)
	a.taskForm.SetWidth(a.width)

	// If in project view, default to that project
	if a.currentProject != nil {
		a.taskForm.ProjectID = a.currentProject.ID
		a.taskForm.ProjectName = a.currentProject.Name

		// Try to detect current section based on cursor position
		// First, check if we can map cursor to a viewport line
		cursorViewportLine := -1
		if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
			taskIndex := a.taskOrderedIndices[a.taskCursor]
			// Find which viewport line this task is on
			for i, vTaskIdx := range a.viewportLines {
				if vTaskIdx == taskIndex {
					cursorViewportLine = i
					break
				}
			}
		}

		// Check if cursor is on an empty section header (taskIndex <= -100)
		if cursorViewportLine >= 0 && cursorViewportLine < len(a.viewportLines) {
			taskIdx := a.viewportLines[cursorViewportLine]
			if taskIdx <= -100 {
				// Cursor is on an empty section header, get the section ID
				if cursorViewportLine < len(a.viewportSections) {
					sectionID := a.viewportSections[cursorViewportLine]
					if sectionID != "" {
						a.taskForm.SectionID = sectionID
						// Find section name
						for _, s := range a.sections {
							if s.ID == sectionID {
								a.taskForm.SectionName = s.Name
								break
							}
						}
					}
				}
			} else if taskIdx >= 0 {
				// Cursor is on a task, use its section
				if taskIdx < len(a.tasks) {
					task := a.tasks[taskIdx]
					if task.SectionID != nil && *task.SectionID != "" {
						a.taskForm.SectionID = *task.SectionID
						// Find section name
						for _, s := range a.sections {
							if s.ID == *task.SectionID {
								a.taskForm.SectionName = s.Name
								break
							}
						}
					}
				}
			}
		} else {
			// Fallback: Use the old method
			if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
				taskIndex := a.taskOrderedIndices[a.taskCursor]
				if taskIndex >= 0 && taskIndex < len(a.tasks) {
					task := a.tasks[taskIndex]
					if task.SectionID != nil && *task.SectionID != "" {
						a.taskForm.SectionID = *task.SectionID
						// Find section name
						for _, s := range a.sections {
							if s.ID == *task.SectionID {
								a.taskForm.SectionName = s.Name
								break
							}
						}
					}
				}
			}
		}
	}

	return a, nil
}

// handleEdit opens the edit task form for the selected task, or edit project dialog.
func (a *App) handleEdit() (tea.Model, tea.Cmd) {
	// Handle project editing when sidebar is focused
	if a.currentTab == TabProjects && a.focusedPane == PaneSidebar {
		if a.sidebarCursor >= len(a.sidebarItems) {
			return a, nil
		}
		item := a.sidebarItems[a.sidebarCursor]
		if item.Type != "project" {
			return a, nil
		}
		// Find the project to edit
		for i := range a.projects {
			if a.projects[i].ID == item.ID {
				a.editingProject = &a.projects[i]
				a.projectInput = textinput.New()
				a.projectInput.SetValue(a.projects[i].Name)
				a.projectInput.CharLimit = 100
				a.projectInput.Width = 40
				a.projectInput.Focus()
				a.isEditingProject = true
				return a, nil
			}
		}
		return a, nil
	}

	// Handle task editing
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		// Handle label editing when viewing label list
		if a.currentTab == TabLabels && a.currentLabel == nil {
			if a.taskCursor < len(a.labels) {
				a.editingLabel = &a.labels[a.taskCursor]
				a.labelInput = textinput.New()
				a.labelInput.SetValue(a.labels[a.taskCursor].Name)
				a.labelInput.CharLimit = 100
				a.labelInput.Width = 40
				a.labelInput.Focus()
				a.isEditingLabel = true
				return a, nil
			}
		}
		return a, nil
	}

	// Guard: Don't edit task when viewing label list (no task selected)
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	// Use ordered indices if available
	taskIndex := a.taskCursor
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex = a.taskOrderedIndices[a.taskCursor]
	}
	if taskIndex < 0 || taskIndex >= len(a.tasks) {
		return a, nil
	}

	task := &a.tasks[taskIndex]
	a.previousView = a.currentView
	a.currentView = ViewTaskForm
	a.taskForm = NewEditTaskForm(task, a.projects)
	a.taskForm.SetWidth(a.width)

	return a, nil
}

// handleNewProject opens the project creation input.
func (a *App) handleNewProject() (tea.Model, tea.Cmd) {
	// Only allow creating projects in Projects tab
	if a.currentTab != TabProjects {
		return a, nil
	}

	// Initialize project input
	a.projectInput = textinput.New()
	a.projectInput.Placeholder = "Enter project name..."
	a.projectInput.CharLimit = 100
	a.projectInput.Width = 40
	a.projectInput.Focus()
	a.isCreatingProject = true

	return a, nil
}

// handleProjectInputKeyMsg handles keyboard input during project creation.
func (a *App) handleProjectInputKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel project creation
		a.isCreatingProject = false
		a.projectInput.Reset()
		return a, nil

	case "enter":
		// Submit new project
		name := strings.TrimSpace(a.projectInput.Value())
		if name == "" {
			a.isCreatingProject = false
			a.projectInput.Reset()
			return a, nil
		}

		a.isCreatingProject = false
		a.projectInput.Reset()
		a.loading = true

		return a, func() tea.Msg {
			project, err := a.client.CreateProject(api.CreateProjectRequest{
				Name: name,
			})
			if err != nil {
				return errMsg{err}
			}
			// Refresh projects after creation
			return projectCreatedMsg{project: project}
		}

	default:
		// Update text input
		var cmd tea.Cmd
		a.projectInput, cmd = a.projectInput.Update(msg)
		return a, cmd
	}
}

// handleProjectEditKeyMsg handles keyboard input during project editing.
func (a *App) handleProjectEditKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel project editing
		a.isEditingProject = false
		a.editingProject = nil
		a.projectInput.Reset()
		return a, nil

	case "enter":
		// Submit project update
		name := strings.TrimSpace(a.projectInput.Value())
		if name == "" || a.editingProject == nil {
			a.isEditingProject = false
			a.editingProject = nil
			a.projectInput.Reset()
			return a, nil
		}

		projectID := a.editingProject.ID
		a.isEditingProject = false
		a.editingProject = nil
		a.projectInput.Reset()
		a.loading = true

		return a, func() tea.Msg {
			project, err := a.client.UpdateProject(projectID, api.UpdateProjectRequest{
				Name: &name,
			})
			if err != nil {
				return errMsg{err}
			}
			return projectUpdatedMsg{project: project}
		}

	default:
		// Update text input
		var cmd tea.Cmd
		a.projectInput, cmd = a.projectInput.Update(msg)
		return a, cmd
	}
}

// handleDeleteConfirmKeyMsg handles y/n/esc during project delete confirmation.
func (a *App) handleDeleteConfirmKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirm delete
		if a.editingProject == nil {
			a.confirmDeleteProject = false
			return a, nil
		}

		projectID := a.editingProject.ID
		a.confirmDeleteProject = false
		a.editingProject = nil
		a.loading = true

		return a, func() tea.Msg {
			err := a.client.DeleteProject(projectID)
			if err != nil {
				return errMsg{err}
			}
			return projectDeletedMsg{id: projectID}
		}

	case "n", "N", "esc":
		// Cancel delete
		a.confirmDeleteProject = false
		a.editingProject = nil
		return a, nil

	default:
		return a, nil
	}
}

// handleNewLabel opens the label creation input.
func (a *App) handleNewLabel() (tea.Model, tea.Cmd) {
	// Initialize label input
	a.labelInput = textinput.New()
	a.labelInput.Placeholder = "Enter label name..."
	a.labelInput.CharLimit = 100
	a.labelInput.Width = 40
	a.labelInput.Focus()
	a.isCreatingLabel = true

	return a, nil
}

// handleLabelInputKeyMsg handles keyboard input during label creation.
func (a *App) handleLabelInputKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel label creation
		a.isCreatingLabel = false
		a.labelInput.Reset()
		return a, nil

	case "enter":
		// Submit new label
		name := strings.TrimSpace(a.labelInput.Value())
		if name == "" {
			a.isCreatingLabel = false
			a.labelInput.Reset()
			return a, nil
		}

		a.isCreatingLabel = false
		a.labelInput.Reset()
		a.loading = true

		return a, func() tea.Msg {
			label, err := a.client.CreateLabel(api.CreateLabelRequest{
				Name: name,
			})
			if err != nil {
				return errMsg{err}
			}
			return labelCreatedMsg{label: label}
		}

	default:
		// Update text input
		var cmd tea.Cmd
		a.labelInput, cmd = a.labelInput.Update(msg)
		return a, cmd
	}
}

// handleLabelEditKeyMsg handles keyboard input during label editing.
func (a *App) handleLabelEditKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.isEditingLabel = false
		a.editingLabel = nil
		a.labelInput.Reset()
		return a, nil

	case "enter":
		name := strings.TrimSpace(a.labelInput.Value())
		if name == "" || a.editingLabel == nil {
			a.isEditingLabel = false
			a.editingLabel = nil
			a.labelInput.Reset()
			return a, nil
		}

		labelID := a.editingLabel.ID
		a.isEditingLabel = false
		a.editingLabel = nil
		a.labelInput.Reset()
		a.loading = true

		return a, func() tea.Msg {
			label, err := a.client.UpdateLabel(labelID, api.UpdateLabelRequest{
				Name: &name,
			})
			if err != nil {
				return errMsg{err}
			}
			return labelUpdatedMsg{label: label}
		}

	default:
		var cmd tea.Cmd
		a.labelInput, cmd = a.labelInput.Update(msg)
		return a, cmd
	}
}

// handleLabelDeleteConfirmKeyMsg handles y/n/esc during label delete confirmation.
func (a *App) handleLabelDeleteConfirmKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if a.editingLabel == nil {
			a.confirmDeleteLabel = false
			return a, nil
		}

		labelID := a.editingLabel.ID
		a.confirmDeleteLabel = false
		a.editingLabel = nil
		a.loading = true

		return a, func() tea.Msg {
			if err := a.client.DeleteLabel(labelID); err != nil {
				return errMsg{err}
			}
			return labelDeletedMsg{id: labelID}
		}

	case "n", "N", "esc":
		a.confirmDeleteLabel = false
		a.editingLabel = nil
		return a, nil

	default:
		return a, nil
	}
}

// handleSectionInputKeyMsg handles keyboard input during section creation.
func (a *App) handleSectionInputKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.isCreatingSection = false
		a.sectionInput.Reset()
		return a, nil

	case "enter":
		name := strings.TrimSpace(a.sectionInput.Value())
		if name == "" {
			return a, nil
		}

		// Use current project if available, otherwise fall back to sidebar
		var projectID string
		if a.currentProject != nil {
			projectID = a.currentProject.ID
		} else if a.currentTab == TabProjects && len(a.projects) > 0 && a.sidebarCursor < len(a.sidebarItems) {
			projectID = a.sidebarItems[a.sidebarCursor].ID
		}

		if projectID != "" {
			a.isCreatingSection = false
			a.sectionInput.Reset()
			a.loading = true

			return a, func() tea.Msg {
				section, err := a.client.CreateSection(api.CreateSectionRequest{
					ProjectID: projectID,
					Name:      name,
				})
				if err != nil {
					return errMsg{err}
				}
				return sectionCreatedMsg{section: section}
			}
		}
		a.isCreatingSection = false
		return a, nil

	default:
		var cmd tea.Cmd
		a.sectionInput, cmd = a.sectionInput.Update(msg)
		return a, cmd
	}
}

// handleSectionEditKeyMsg handles keyboard input during section editing.
func (a *App) handleSectionEditKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.isEditingSection = false
		a.editingSection = nil
		a.sectionInput.Reset()
		return a, nil

	case "enter":
		name := strings.TrimSpace(a.sectionInput.Value())
		if name == "" || a.editingSection == nil {
			a.isEditingSection = false
			a.editingSection = nil
			a.sectionInput.Reset()
			return a, nil
		}

		sectionID := a.editingSection.ID
		a.isEditingSection = false
		a.editingSection = nil
		a.sectionInput.Reset()
		a.loading = true

		return a, func() tea.Msg {
			section, err := a.client.UpdateSection(sectionID, api.UpdateSectionRequest{
				Name: name,
			})
			if err != nil {
				return errMsg{err}
			}
			return sectionUpdatedMsg{section: section}
		}

	default:
		var cmd tea.Cmd
		a.sectionInput, cmd = a.sectionInput.Update(msg)
		return a, cmd
	}
}

// handleSectionDeleteConfirmKeyMsg handles y/n/esc during section delete confirmation.
func (a *App) handleSectionDeleteConfirmKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if a.editingSection == nil {
			a.confirmDeleteSection = false
			return a, nil
		}

		sectionID := a.editingSection.ID
		a.confirmDeleteSection = false
		a.editingSection = nil
		a.loading = true

		return a, func() tea.Msg {
			if err := a.client.DeleteSection(sectionID); err != nil {
				return errMsg{err}
			}
			return sectionDeletedMsg{id: sectionID}
		}

	case "n", "N", "esc":
		a.confirmDeleteSection = false
		a.editingSection = nil
		return a, nil

	default:
		return a, nil
	}
}

// handleSectionsKeyMsg handles keyboard input for the sections management view.
func (a *App) handleSectionsKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.currentView = a.previousView
		a.taskCursor = 0
		return a, nil

	case "up", "k":
		if a.taskCursor > 0 {
			a.taskCursor--
		}
		return a, nil

	case "down", "j":
		if a.taskCursor < len(a.sections)-1 {
			a.taskCursor++
		}
		return a, nil

	case "a":
		a.sectionInput = textinput.New()
		a.sectionInput.Placeholder = "New section name..."
		a.sectionInput.CharLimit = 100
		a.sectionInput.Width = 40
		a.sectionInput.Focus()
		a.isCreatingSection = true
		return a, nil

	case "e":
		if len(a.sections) == 0 {
			return a, nil
		}
		if a.taskCursor >= 0 && a.taskCursor < len(a.sections) {
			a.editingSection = &a.sections[a.taskCursor]
			a.sectionInput = textinput.New()
			a.sectionInput.SetValue(a.editingSection.Name)
			a.sectionInput.CharLimit = 100
			a.sectionInput.Width = 40
			a.sectionInput.Focus()
			a.isEditingSection = true
		}
		return a, nil

	case "d", "delete":
		if len(a.sections) == 0 {
			return a, nil
		}
		if a.taskCursor >= 0 && a.taskCursor < len(a.sections) {
			a.editingSection = &a.sections[a.taskCursor]
			a.confirmDeleteSection = true
		}
		return a, nil
	}

	return a, nil
}

// handleMoveTaskKeyMsg handles keyboard input for moving a task to a section.
func (a *App) handleMoveTaskKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.isMovingTask = false
		a.moveSectionCursor = 0
		return a, nil

	case "up", "k":
		if a.moveSectionCursor > 0 {
			a.moveSectionCursor--
		}
		return a, nil

	case "down", "j":
		if a.moveSectionCursor < len(a.sections)-1 {
			a.moveSectionCursor++
		}
		return a, nil

	case "enter":
		if len(a.sections) == 0 {
			a.isMovingTask = false
			return a, nil
		}

		// Get the correct task using ordered indices mapping
		var task *api.Task
		if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
			taskIndex := a.taskOrderedIndices[a.taskCursor]
			if taskIndex >= 0 && taskIndex < len(a.tasks) {
				task = &a.tasks[taskIndex]
			}
		} else if a.taskCursor < len(a.tasks) {
			task = &a.tasks[a.taskCursor]
		}

		if task == nil {
			a.isMovingTask = false
			a.statusMsg = "No task selected"
			return a, nil
		}

		sectionID := a.sections[a.moveSectionCursor].ID

		// Don't move if section ID is empty or if it's the same section
		if sectionID == "" {
			a.isMovingTask = false
			a.statusMsg = "Invalid section"
			return a, nil
		}
		if task.SectionID != nil && *task.SectionID == sectionID {
			a.isMovingTask = false
			a.statusMsg = "Task is already in this section"
			return a, nil
		}

		a.isMovingTask = false
		a.loading = true
		a.statusMsg = "Moving task..."

		return a, func() tea.Msg {
			// Update task with new section_id
			_, err := a.client.UpdateTask(task.ID, api.UpdateTaskRequest{
				SectionID: &sectionID,
			})
			if err != nil {
				return errMsg{err}
			}
			return taskUpdatedMsg{}
		}
	}
	return a, nil
}

// handleCommentInputKeyMsg handles keyboard input for adding a comment.
func (a *App) handleCommentInputKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.isAddingComment = false
		a.commentInput.Reset()
		return a, nil

	case "enter":
		content := strings.TrimSpace(a.commentInput.Value())
		if content == "" {
			return a, nil
		}

		// determine task ID (from selection or cursor)
		taskID := ""
		if a.selectedTask != nil {
			taskID = a.selectedTask.ID
		} else if len(a.tasks) > 0 && a.taskCursor < len(a.tasks) {
			taskID = a.tasks[a.taskCursor].ID
		} else {
			a.isAddingComment = false
			return a, nil
		}

		a.isAddingComment = false
		a.commentInput.Reset()
		a.loading = true
		a.statusMsg = "Adding comment..."

		return a, func() tea.Msg {
			comment, err := a.client.CreateComment(api.CreateCommentRequest{
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
		a.commentInput, cmd = a.commentInput.Update(msg)
		return a, cmd
	}
}

// renderSections renders the sections management view.
func (a *App) renderSections() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Manage Sections"))
	b.WriteString("\n\n")

	if len(a.sections) == 0 {
		b.WriteString(styles.HelpDesc.Render("No sections found. Press 'a' to add one."))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpDesc.Render("Esc: back"))
		return b.String()
	}

	// Render list
	for i, section := range a.sections {
		cursor := "  "
		style := lipgloss.NewStyle()

		if i == a.taskCursor {
			cursor = "> "
			style = lipgloss.NewStyle().Foreground(styles.Highlight)
		}

		b.WriteString(cursor + style.Render(section.Name) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("j/k: nav • a: add • e: edit • d: delete • Esc: back"))

	return b.String()
}

// handleAddSubtask opens the inline subtask creation input.
func (a *App) handleAddSubtask() (tea.Model, tea.Cmd) {
	// Guard: Only in main pane with tasks
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		return a, nil
	}

	// Get the selected task using ordered indices
	taskIndex := a.taskCursor
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex = a.taskOrderedIndices[a.taskCursor]
	}
	if taskIndex < 0 || taskIndex >= len(a.tasks) {
		return a, nil
	}

	// Initialize subtask input
	a.parentTaskID = a.tasks[taskIndex].ID
	a.subtaskInput = textinput.New()
	a.subtaskInput.Placeholder = "Enter subtask..."
	a.subtaskInput.CharLimit = 200
	a.subtaskInput.Width = 50
	a.subtaskInput.Focus()
	a.isCreatingSubtask = true

	return a, nil
}

// handleSubtaskInputKeyMsg handles keyboard input during subtask creation.
func (a *App) handleSubtaskInputKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel subtask creation
		a.isCreatingSubtask = false
		a.parentTaskID = ""
		a.subtaskInput.Reset()
		return a, nil

	case "enter":
		// Submit new subtask
		content := strings.TrimSpace(a.subtaskInput.Value())
		if content == "" {
			a.isCreatingSubtask = false
			a.parentTaskID = ""
			a.subtaskInput.Reset()
			return a, nil
		}

		parentID := a.parentTaskID
		a.isCreatingSubtask = false
		a.parentTaskID = ""
		a.subtaskInput.Reset()
		a.loading = true

		return a, func() tea.Msg {
			_, err := a.client.CreateTask(api.CreateTaskRequest{
				Content:  content,
				ParentID: parentID,
			})
			if err != nil {
				return errMsg{err}
			}
			return subtaskCreatedMsg{}
		}

	default:
		// Update text input
		var cmd tea.Cmd
		a.subtaskInput, cmd = a.subtaskInput.Update(msg)
		return a, cmd
	}
}

// handleFormKeyMsg handles keyboard input when the form is active.
func (a *App) handleFormKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.taskForm == nil {
		return a, nil
	}

	switch msg.String() {
	case "esc":
		// Cancel form and go back
		a.currentView = a.previousView
		a.taskForm = nil
		return a, nil

	case "enter":
		// If on submit button or Ctrl+Enter from any field, submit form
		if a.taskForm.FocusedField == FormFieldSubmit || msg.String() == "ctrl+enter" {
			return a.submitForm()
		}
		// Otherwise, let form handle it (e.g., for project dropdown)
	}

	// Forward to form
	var cmd tea.Cmd
	a.taskForm, cmd = a.taskForm.Update(msg)
	return a, cmd
}

// submitForm submits the task form (create or update).
func (a *App) submitForm() (tea.Model, tea.Cmd) {
	if a.taskForm == nil || !a.taskForm.IsValid() {
		a.statusMsg = "Task name is required"
		return a, nil
	}

	a.loading = true

	if a.taskForm.Mode == "edit" {
		// Update existing task
		taskID := a.taskForm.TaskID
		req := a.taskForm.ToUpdateRequest()
		return a, func() tea.Msg {
			_, err := a.client.UpdateTask(taskID, req)
			if err != nil {
				return errMsg{err}
			}
			return taskCreatedMsg{} // Reuse message type for refresh
		}
	}

	// Create new task
	req := a.taskForm.ToCreateRequest()
	return a, func() tea.Msg {
		_, err := a.client.CreateTask(req)
		if err != nil {
			return errMsg{err}
		}
		return taskCreatedMsg{}
	}
}

// handleSearch opens the search view.
func (a *App) handleSearch() (tea.Model, tea.Cmd) {
	a.previousView = a.currentView
	a.currentView = ViewSearch
	a.searchInput.Reset()
	a.searchInput.Focus()
	a.searchResults = nil
	a.searchQuery = ""
	a.taskCursor = 0
	return a, nil
}

// handleSearchKeyMsg handles keyboard input when search is active.
func (a *App) handleSearchKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel search and go back
		a.currentView = a.previousView
		a.searchInput.Blur()
		a.searchResults = nil
		a.searchQuery = ""
		return a, nil

	case "enter":
		// Select task from search results
		if len(a.searchResults) > 0 && a.taskCursor < len(a.searchResults) {
			a.selectedTask = &a.searchResults[a.taskCursor]
			a.previousView = ViewSearch
			a.currentView = ViewTaskDetail
			return a, nil
		}
		return a, nil

	case "down", "j":
		if a.taskCursor < len(a.searchResults)-1 {
			a.taskCursor++
		}
		return a, nil

	case "up", "k":
		if a.taskCursor > 0 {
			a.taskCursor--
		}
		return a, nil

	case "x":
		// Complete task from search results
		if len(a.searchResults) > 0 && a.taskCursor < len(a.searchResults) {
			task := &a.searchResults[a.taskCursor]
			a.loading = true
			return a, func() tea.Msg {
				var err error
				if task.Checked {
					err = a.client.ReopenTask(task.ID)
				} else {
					err = a.client.CloseTask(task.ID)
				}
				if err != nil {
					return errMsg{err}
				}
				return searchRefreshMsg{}
			}
		}
		return a, nil
	}

	// Update search input and filter results
	var cmd tea.Cmd
	a.searchInput, cmd = a.searchInput.Update(msg)
	a.searchQuery = a.searchInput.Value()
	a.filterSearchResults()
	a.taskCursor = 0 // Reset cursor when query changes
	return a, cmd
}

// filterSearchResults filters tasks based on search query.
func (a *App) filterSearchResults() {
	query := strings.ToLower(strings.TrimSpace(a.searchQuery))
	if query == "" {
		a.searchResults = nil
		return
	}

	var results []api.Task
	for _, task := range a.tasks {
		// Search in content, description, and labels
		if strings.Contains(strings.ToLower(task.Content), query) ||
			strings.Contains(strings.ToLower(task.Description), query) {
			results = append(results, task)
			continue
		}

		// Search in labels
		for _, label := range task.Labels {
			if strings.Contains(strings.ToLower(label), query) {
				results = append(results, task)
				break
			}
		}
	}

	a.searchResults = results
}

// refreshSearchResults refreshes the search results after a task update.
func (a *App) refreshSearchResults() tea.Cmd {
	return func() tea.Msg {
		// Reload all tasks
		tasks, err := a.client.GetTasks(api.TaskFilter{})
		if err != nil {
			return errMsg{err}
		}
		a.tasks = tasks
		a.filterSearchResults()
		return dataLoadedMsg{tasks: tasks}
	}
}

// handlePriority sets task priority.
func (a *App) handlePriority(action string) (tea.Model, tea.Cmd) {
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		return a, nil
	}

	task := &a.tasks[a.taskCursor]
	var priority int
	switch action {
	case "priority1":
		priority = 4 // Todoist uses 4 as highest
	case "priority2":
		priority = 3
	case "priority3":
		priority = 2
	case "priority4":
		priority = 1
	}

	a.loading = true
	return a, func() tea.Msg {
		_, err := a.client.UpdateTask(task.ID, api.UpdateTaskRequest{
			Priority: &priority,
		})
		if err != nil {
			return errMsg{err}
		}
		return taskUpdatedMsg{}
	}
}

// handleDueToday sets the task due date to today.
func (a *App) handleDueToday() (tea.Model, tea.Cmd) {
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		return a, nil
	}

	// Guard: Don't operate when viewing label list
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	task := &a.tasks[a.taskCursor]
	dueString := "today"

	a.loading = true
	a.statusMsg = "Moving to today..."
	return a, func() tea.Msg {
		_, err := a.client.UpdateTask(task.ID, api.UpdateTaskRequest{
			DueString: &dueString,
		})
		if err != nil {
			return errMsg{err}
		}
		return taskUpdatedMsg{}
	}
}

// handleDueTomorrow sets the task due date to tomorrow.
func (a *App) handleDueTomorrow() (tea.Model, tea.Cmd) {
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		return a, nil
	}

	// Guard: Don't operate when viewing label list
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	task := &a.tasks[a.taskCursor]
	dueString := "tomorrow"

	a.loading = true
	a.statusMsg = "Moving to tomorrow..."
	return a, func() tea.Msg {
		_, err := a.client.UpdateTask(task.ID, api.UpdateTaskRequest{
			DueString: &dueString,
		})
		if err != nil {
			return errMsg{err}
		}
		return taskUpdatedMsg{}
	}
}

// loadProjectTasks loads tasks for a specific project.
func (a *App) loadProjectTasks(projectID string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.GetTasks(api.TaskFilter{
			ProjectID: projectID,
		})
		if err != nil {
			return errMsg{err}
		}

		sections, err := a.client.GetSections(projectID)
		if err != nil {
			return errMsg{err}
		}

		return dataLoadedMsg{
			tasks:    tasks,
			sections: sections,
		}
	}
}

// refreshTasks refreshes the current task list.
func (a *App) refreshTasks() tea.Cmd {
	return func() tea.Msg {
		var tasks []api.Task
		var err error

		if a.currentView == ViewProject && a.currentProject != nil {
			tasks, err = a.client.GetTasks(api.TaskFilter{
				ProjectID: a.currentProject.ID,
			})
		} else if a.currentView == ViewLabels && a.currentLabel != nil {
			// Load tasks for the selected label using filter endpoint
			tasks, err = a.client.GetTasksByFilter("@" + a.currentLabel.Name)
		} else {
			// Default to today | overdue
			tasks, err = a.client.GetTasksByFilter("today | overdue")
		}

		if err != nil {
			return errMsg{err}
		}

		return dataLoadedMsg{tasks: tasks}
	}
}

// loadTodayTasks loads today's tasks including overdue.
func (a *App) loadTodayTasks() tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.GetTasksByFilter("today | overdue")
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{tasks: tasks}
	}
}

// loadUpcomingTasks loads all tasks with due dates for the upcoming view.
func (a *App) loadUpcomingTasks() tea.Cmd {
	return func() tea.Msg {
		// Get all tasks and filter to those with due dates
		tasks, err := a.client.GetTasks(api.TaskFilter{})
		if err != nil {
			return errMsg{err}
		}

		// Filter to tasks with due dates and sort by date
		var upcoming []api.Task
		for _, t := range tasks {
			if t.Due != nil {
				upcoming = append(upcoming, t)
			}
		}

		// Sort by due date
		sort.Slice(upcoming, func(i, j int) bool {
			return upcoming[i].Due.Date < upcoming[j].Due.Date
		})

		return dataLoadedMsg{tasks: upcoming, allTasks: tasks}
	}
}

// loadAllTasks loads all tasks (for calendar view).
func (a *App) loadAllTasks() tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.GetTasks(api.TaskFilter{})
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{allTasks: tasks}
	}
}

// loadProjects loads all projects.
func (a *App) loadProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := a.client.GetProjects()
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{projects: projects}
	}
}

// loadLabels loads all labels.
func (a *App) loadLabels() tea.Cmd {
	return func() tea.Msg {
		labels, err := a.client.GetLabels()
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{labels: labels}
	}
}

// loadLabelTasks loads tasks filtered by a specific label.
func (a *App) loadLabelTasks(labelName string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.GetTasksByFilter("@" + labelName)
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{tasks: tasks}
	}
}

// loadCalendarDayTasks loads tasks for the selected calendar day.
func (a *App) loadCalendarDayTasks() tea.Cmd {
	selectedDate := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), a.calendarDay, 0, 0, 0, 0, time.Local)
	dateStr := selectedDate.Format("2006-01-02")

	return func() tea.Msg {
		tasks, err := a.client.GetTasksByFilter(dateStr)
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{tasks: tasks}
	}
}

// loadTaskComments loads comments for the selected task.
func (a *App) loadTaskComments() tea.Cmd {
	if a.selectedTask == nil {
		return nil
	}
	taskID := a.selectedTask.ID
	return func() tea.Msg {
		comments, err := a.client.GetComments(taskID, "")
		if err != nil {
			return errMsg{err}
		}
		return commentsLoadedMsg{comments: comments}
	}
}

// buildSidebarItems constructs the sidebar items list with only projects (for Projects tab).
func (a *App) buildSidebarItems() {
	a.sidebarItems = []SidebarItem{}

	// Add favorite projects first
	for _, p := range a.projects {
		if p.IsFavorite {
			icon := "⭐"
			if p.InboxProject {
				icon = "📥"
			}
			a.sidebarItems = append(a.sidebarItems, SidebarItem{
				Type:       "project",
				ID:         p.ID,
				Name:       p.Name,
				Icon:       icon,
				IsFavorite: true,
				ParentID:   p.ParentID,
			})
		}
	}

	// Add separator if there were favorites
	hasFavorites := false
	for _, p := range a.projects {
		if p.IsFavorite {
			hasFavorites = true
			break
		}
	}
	if hasFavorites {
		a.sidebarItems = append(a.sidebarItems, SidebarItem{Type: "separator", ID: "", Name: ""})
	}

	// Add remaining projects (non-favorites)
	for _, p := range a.projects {
		if !p.IsFavorite {
			icon := "📁"
			if p.InboxProject {
				icon = "📥"
			}
			a.sidebarItems = append(a.sidebarItems, SidebarItem{
				Type:     "project",
				ID:       p.ID,
				Name:     p.Name,
				Icon:     icon,
				ParentID: p.ParentID,
			})
		}
	}
}

// View implements tea.Model.
func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	var content string
	switch a.currentView {
	case ViewHelp:
		content = a.renderHelp()
	case ViewTaskDetail:
		content = a.renderTaskDetail()
	case ViewTaskForm:
		content = a.renderTaskForm()
	case ViewSearch:
		content = a.renderSearch()
	case ViewCalendarDay:
		content = a.renderCalendarDay()
	case ViewSections:
		content = a.renderSections()
	default:
		content = a.renderMainView()
	}

	// Overlay project creation dialog if active
	if a.isCreatingProject {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("📁 New Project") + "\n\n" +
			a.projectInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: create • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay active overlays...
	// (Skipping project/label overlays replication for brevity... waiting for replace_file_content to handle context correctly)
	// I should probably target specific blocks instead of replcaing huge chunks unless necessary.
	// But I will append section overlays at the END of overlays list.

	// Overlay project edit dialog if active
	if a.isEditingProject {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("✏️ Edit Project") + "\n\n" +
			a.projectInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: save • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay project delete confirmation dialog if active
	if a.confirmDeleteProject && a.editingProject != nil {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.ErrorColor).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.StatusBarError.Render("⚠️ Delete Project?") + "\n\n" +
			fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", a.editingProject.Name) +
			styles.HelpDesc.Render("This will delete all tasks in this project.") + "\n\n" +
			styles.HelpDesc.Render("y: confirm • n/Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay label creation dialog if active
	if a.isCreatingLabel {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("🏷️ New Label") + "\n\n" +
			a.labelInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: create • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay label edit dialog if active
	if a.isEditingLabel {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("✏️ Edit Label") + "\n\n" +
			a.labelInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: save • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay label delete confirmation dialog if active
	if a.confirmDeleteLabel && a.editingLabel != nil {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.ErrorColor).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.StatusBarError.Render("⚠️ Delete Label?") + "\n\n" +
			fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", a.editingLabel.Name) +
			styles.HelpDesc.Render("y: confirm • n/Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay subtask creation dialog if active
	if a.isCreatingSubtask {
		dialogWidth := 60
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("➕ Add Subtask") + "\n\n" +
			a.subtaskInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: create • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay section creation dialog if active
	if a.isCreatingSection {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("📂 New Section") + "\n\n" +
			a.sectionInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: create • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay section edit dialog if active
	if a.isEditingSection {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("✏️ Edit Section") + "\n\n" +
			a.sectionInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: save • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay section delete confirmation dialog if active
	if a.confirmDeleteSection && a.editingSection != nil {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.ErrorColor).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.StatusBarError.Render("⚠️ Delete Section?") + "\n\n" +
			fmt.Sprintf("Are you sure you want to delete \"%s\"?\n", a.editingSection.Name) +
			styles.HelpDesc.Render("This will likely delete/move tasks inside.") + "\n\n" +
			styles.HelpDesc.Render("y: confirm • n/Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
		content = strings.Join(contentLines, "\n")
	}

	// Overlay move task dialog if active
	if a.isMovingTask {
		dialogWidth := 50
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		var b strings.Builder
		b.WriteString(styles.Title.Render("➡️ Move Task to Section") + "\n\n")

		if len(a.sections) == 0 {
			b.WriteString(styles.HelpDesc.Render("No sections in this project."))
		} else {
			for i, section := range a.sections {
				cursor := "  "
				style := lipgloss.NewStyle()
				if i == a.moveSectionCursor {
					cursor = "> "
					style = lipgloss.NewStyle().Foreground(styles.Highlight)
				}
				b.WriteString(cursor + style.Render(section.Name) + "\n")
			}
		}

		b.WriteString("\n" + styles.HelpDesc.Render("j/k: select • Enter: move • Esc: cancel"))

		dialog := dialogStyle.Render(b.String())

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	// Overlay add comment dialog if active
	if a.isAddingComment {
		dialogWidth := 60
		dialogStyle := lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(styles.Highlight).
			Padding(1, 2).
			Width(dialogWidth)

		dialogContent := styles.Title.Render("💬 Add Comment") + "\n\n" +
			a.commentInput.View() + "\n\n" +
			styles.HelpDesc.Render("Enter: submit • Esc: cancel")

		dialog := dialogStyle.Render(dialogContent)

		// Center the dialog
		dialogLines := strings.Split(dialog, "\n")
		centeredDialog := ""
		leftPad := (a.width - dialogWidth - 4) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		for _, line := range dialogLines {
			centeredDialog += strings.Repeat(" ", leftPad) + line + "\n"
		}

		// Overlay on content
		contentLines := strings.Split(content, "\n")
		dialogLineCount := len(dialogLines)
		startLine := (len(contentLines) - dialogLineCount) / 2
		if startLine < 0 {
			startLine = 0
		}

		// Replace content lines with dialog
		dialogSplit := strings.Split(centeredDialog, "\n")
		for i := 0; i < len(dialogSplit) && startLine+i < len(contentLines); i++ {
			contentLines[startLine+i] = dialogSplit[i]
		}
		content = strings.Join(contentLines, "\n")
	}

	return content
}

// renderMainView renders the main layout with tab bar and content.
func (a *App) renderMainView() string {
	// Render tab bar
	tabBar := a.renderTabBar()

	// Calculate content height dynamically (total - tab bar - status bar)
	tabBarHeight := lipgloss.Height(tabBar)
	statusBarHeight := 2
	contentHeight := a.height - tabBarHeight - statusBarHeight

	var mainContent string

	// If detail panel is shown, split the view
	if a.showDetailPanel && a.selectedTask != nil {
		// Split layout: task list on left, detail panel on right
		detailWidth := a.width / 2
		listWidth := a.width - detailWidth - 3 // -3 for border/spacing

		var leftPane string
		if a.currentTab == TabProjects {
			leftPane = a.renderProjectsTabContent(listWidth, contentHeight)
		} else {
			leftPane = a.renderTaskList(listWidth, contentHeight)
		}

		// Render detail panel - using component
		a.detailComp.SetSize(detailWidth, contentHeight)
		a.detailComp.SetTask(a.selectedTask)
		a.detailComp.SetComments(a.comments)
		rightPane := a.detailComp.ViewPanel()

		mainContent = lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	} else {
		if a.currentTab == TabProjects {
			// Projects tab shows sidebar + content
			mainContent = a.renderProjectsTabContent(a.width, contentHeight)
		} else {
			// Other tabs show content only (full width)
			mainContent = a.renderTaskList(a.width-2, contentHeight)
		}
	}

	// Add status bar
	statusBar := a.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, mainContent, statusBar)
}

// renderDetailPanel renders task details in the right panel for split view.
func (a *App) renderDetailPanel(width, height int) string {
	if a.selectedTask == nil {
		return ""
	}

	t := a.selectedTask

	// Create border style
	panelStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Highlight).
		Padding(0, 1).
		Width(width - 2).
		Height(height - 2)

	// Build content
	var content strings.Builder

	// Title
	content.WriteString(styles.Title.Render(t.Content) + "\n\n")

	// Due date
	if t.Due != nil {
		content.WriteString(styles.StatusBarKey.Render("Due: "))
		content.WriteString(t.Due.String + "\n")
	}

	// Priority
	priorityStyle := styles.GetPriorityStyle(t.Priority)
	priorityLabel := fmt.Sprintf("P%d", 5-t.Priority)
	content.WriteString(styles.StatusBarKey.Render("Priority: "))
	content.WriteString(priorityStyle.Render(priorityLabel) + "\n")

	// Description
	if t.Description != "" {
		content.WriteString("\n" + styles.StatusBarKey.Render("Description:") + "\n")
		content.WriteString(t.Description + "\n")
	}

	// Comments
	if len(a.comments) > 0 {
		content.WriteString("\n" + styles.StatusBarKey.Render(fmt.Sprintf("Comments (%d):", len(a.comments))) + "\n")
		for _, c := range a.comments {
			content.WriteString("• " + c.Content + "\n")
		}
	}

	// Help
	content.WriteString("\n" + styles.HelpDesc.Render("Esc to close"))

	return panelStyle.Render(content.String())
}

// tabInfo holds tab metadata for rendering and click handling.
type tabInfo struct {
	tab       Tab
	icon      string
	name      string
	shortName string
}

// getTabDefinitions returns the tab definitions.
func getTabDefinitions() []tabInfo {
	return []tabInfo{
		{TabToday, "[T]", "Today", "Tdy"},
		{TabUpcoming, "[U]", "Upcoming", "Up"},
		{TabLabels, "[L]", "Labels", "Lbl"},
		{TabCalendar, "[C]", "Calendar", "Cal"},
		{TabProjects, "[P]", "Projects", "Prj"},
	}
}

// renderTabBar renders the top tab bar.
func (a *App) renderTabBar() string {
	tabs := getTabDefinitions()

	// Determine label style based on available width
	// Full: "T Today" (~9 chars rendered), Short: "T Tdy" (~7 chars), Minimal: "T" (~3 chars)
	// Each tab with padding(2+2) + separator(1) = +5 chars overhead
	// 5 tabs * 14 chars (full with padding) = ~70 chars minimum for full labels
	useShortLabels := a.width < 80
	useMinimalLabels := a.width < 50

	var tabStrs []string
	for _, t := range tabs {
		var label string
		if useMinimalLabels {
			label = t.icon
		} else if useShortLabels {
			label = fmt.Sprintf("%s %s", t.icon, t.shortName)
		} else {
			label = fmt.Sprintf("%s %s", t.icon, t.name)
		}

		if a.currentTab == t.tab {
			tabStrs = append(tabStrs, styles.TabActive.Render(label))
		} else {
			tabStrs = append(tabStrs, styles.Tab.Render(label))
		}
	}

	tabLine := strings.Join(tabStrs, " ")

	// Truncate if still too wide
	maxWidth := a.width - 4 // Account for TabBar padding
	if lipgloss.Width(tabLine) > maxWidth && maxWidth > 0 {
		tabLine = lipgloss.NewStyle().MaxWidth(maxWidth).Render(tabLine)
	}

	return styles.TabBar.Width(a.width).Render(tabLine)
}

// renderProjectsTabContent renders content for the Projects tab (sidebar + tasks).
func (a *App) renderProjectsTabContent(width, height int) string {
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
	a.sidebarComp.SetSize(sidebarWidth, height)
	a.sidebarComp.SetCursor(a.sidebarCursor) // Sync cursor from App state
	if a.focusedPane == PaneSidebar {
		a.sidebarComp.Focus()
	} else {
		a.sidebarComp.Blur()
	}
	if a.currentProject != nil {
		a.sidebarComp.SetActiveProject(a.currentProject.ID)
	}
	sidebar := a.sidebarComp.View()

	// Render main content (tasks for selected project)
	main := a.renderProjectTaskList(mainWidth, height)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, main)
}

// renderProjectSidebar renders the project list sidebar (only in Projects tab).
func (a *App) renderProjectSidebar(width, height int) string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Projects"))
	b.WriteString("\n\n")

	// Max name length (accounting for cursor, indent, icon, and padding)
	maxNameLen := width - 10

	// Render project items
	for i, item := range a.sidebarItems {
		if item.Type == "separator" {
			b.WriteString(styles.SidebarSeparator.Render(strings.Repeat("─", width-4)))
			b.WriteString("\n")
			continue
		}

		cursor := "  "
		style := styles.ProjectItem
		if i == a.sidebarCursor && a.focusedPane == PaneSidebar {
			cursor = "> "
			style = styles.ProjectSelected
		}

		// Highlight active project
		if a.currentProject != nil && a.currentProject.ID == item.ID {
			if a.focusedPane != PaneSidebar {
				style = styles.SidebarActive
			}
		}

		// Indent for child projects
		indent := ""
		if item.ParentID != nil {
			indent = "  "
			maxNameLen = width - 12 // Less space for indented items
		}

		// Truncate long names
		name := item.Name
		if len(name) > maxNameLen && maxNameLen > 3 {
			name = name[:maxNameLen-1] + "…"
		}

		line := style.Render(fmt.Sprintf("%s%s%s %s", cursor, indent, item.Icon, name))
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Add hint for creating new project
	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("n: new project"))

	// Apply container style with fixed height
	// Sidebar has rounded border (2 lines), so inner height = height - 2
	innerHeight := height - 2
	if innerHeight < 3 {
		innerHeight = 3
	}
	containerStyle := styles.Sidebar
	if a.focusedPane == PaneSidebar {
		containerStyle = styles.SidebarFocused
	}

	return containerStyle.Width(width).Height(innerHeight).Render(b.String())
}

// renderProjectTaskList renders the task list for the selected project.
func (a *App) renderProjectTaskList(width, height int) string {
	var content string

	// Reserve space for borders (top + bottom = 2 lines)
	innerHeight := height - 2
	if innerHeight < 5 {
		innerHeight = 5
	}

	if a.currentProject == nil {
		content = styles.HelpDesc.Render("Select a project from the sidebar")
	} else {
		content = a.renderDefaultTaskList(innerHeight)
	}

	containerStyle := styles.MainContent
	if a.focusedPane == PaneMain {
		containerStyle = styles.MainContentFocused
	}

	return containerStyle.Width(width).Height(innerHeight).Render(content)
}

// renderTaskList renders the task list.
func (a *App) renderTaskList(width, height int) string {
	var content string

	// Reserve space for borders (top + bottom = 2 lines)
	innerHeight := height - 2
	if innerHeight < 5 {
		innerHeight = 5
	}

	switch a.currentView {
	case ViewUpcoming:
		content = a.renderUpcoming(innerHeight)
	case ViewLabels:
		content = a.renderLabelsView(innerHeight)
	case ViewCalendar:
		content = a.renderCalendar(innerHeight)
	default:
		content = a.renderDefaultTaskList(innerHeight)
	}

	// Apply container style with fixed height
	// Note: Height() sets INNER content height, so use innerHeight (not full height)
	containerStyle := styles.MainContent
	if a.focusedPane == PaneMain {
		containerStyle = styles.MainContentFocused
	}

	return containerStyle.Width(width).Height(innerHeight).Render(content)
}

// renderDefaultTaskList renders the default task list for Today/Project views.
func (a *App) renderDefaultTaskList(maxHeight int) string {
	var b strings.Builder

	// Title
	var title string
	switch a.currentView {
	case ViewToday:
		title = time.Now().Format("Monday 2 Jan")
	case ViewProject:
		if a.currentProject != nil {
			title = a.currentProject.Name
		}
	default:
		title = "Tasks"
	}
	b.WriteString(styles.Title.Render(title))
	b.WriteString("\n")

	if a.loading {
		b.WriteString(a.spinner.View())
		b.WriteString(" Loading...")
	} else if a.err != nil {
		b.WriteString(styles.StatusBarError.Render(fmt.Sprintf("Error: %v", a.err)))
	} else if len(a.tasks) == 0 {
		msg := "No tasks found"
		if a.currentView == ViewToday {
			msg = "All done for today! \n" + styles.HelpDesc.Render("Enjoy your day off 🏝️")
		} else {
			msg = "No tasks here.\n" + styles.HelpDesc.Render("Press 'a' to add one.")
		}
		b.WriteString(msg)
	} else {
		// Group tasks by due status for Today view
		// Title uses 2 lines (title + newline)
		if a.currentView == ViewToday {
			b.WriteString(a.renderGroupedTasks(maxHeight - 2))
		} else if a.currentView == ViewProject {
			b.WriteString(a.renderProjectTasks(maxHeight - 2))
		} else {
			b.WriteString(a.renderFlatTasks(maxHeight - 2))
		}
	}

	return b.String()
}

// lineInfo represents a display line with optional task reference.
type lineInfo struct {
	content   string
	taskIndex int    // -1 for headers
	sectionID string // section ID if this is a section header
}

// renderProjectTasks renders tasks grouped by section for a project.
func (a *App) renderProjectTasks(maxHeight int) string {
	// Build ordered list of task indices matching display order
	var orderedIndices []int

	// Group tasks by section
	tasksBySection := make(map[string][]int)
	var noSectionTasks []int

	for i, t := range a.tasks {
		if t.SectionID != nil && *t.SectionID != "" {
			tasksBySection[*t.SectionID] = append(tasksBySection[*t.SectionID], i)
		} else {
			noSectionTasks = append(noSectionTasks, i)
		}
	}

	// Build ordered indices: no-section tasks first, then by section
	orderedIndices = append(orderedIndices, noSectionTasks...)

	// Track which sections are empty to add them to orderedIndices later
	emptySectionIndices := make(map[string]int) // sectionID -> index in orderedIndices
	sectionOrderCounter := 0

	for _, section := range a.sections {
		if indices, exists := tasksBySection[section.ID]; exists {
			orderedIndices = append(orderedIndices, indices...)
		} else {
			// Empty section: add a special negative index to make it navigable
			// Use -100 minus counter to distinguish from other special indices
			emptyHeaderIndex := -100 - sectionOrderCounter
			emptySectionIndices[section.ID] = emptyHeaderIndex
			orderedIndices = append(orderedIndices, emptyHeaderIndex)
			sectionOrderCounter++
		}
	}

	// Build lines with scroll support
	var lines []lineInfo

	if len(noSectionTasks) > 0 {
		for _, i := range noSectionTasks {
			lines = append(lines, lineInfo{content: a.renderTaskByDisplayIndex(i, orderedIndices), taskIndex: i})
		}
	}

	for _, section := range a.sections {
		taskIndices := tasksBySection[section.ID]

		// Add blank line before section header for spacing
		// IMPORTANT: Use separate lineInfo entries, NOT embedded \n in content
		// Embedded \n breaks viewport line counting
		lines = append(lines, lineInfo{content: "", taskIndex: -1})

		// Check if section is empty
		if len(taskIndices) == 0 {
			// Empty section: make header hoverable
			emptyHeaderIndex := emptySectionIndices[section.ID]
			lines = append(lines, lineInfo{
				content:   a.renderSectionHeaderByIndex(section.Name, emptyHeaderIndex, orderedIndices),
				taskIndex: emptyHeaderIndex,
				sectionID: section.ID,
			})
		} else {
			// Non-empty section: header is just display (not hoverable)
			lines = append(lines, lineInfo{
				content:   styles.SectionHeader.Render(section.Name),
				taskIndex: -1,
				sectionID: section.ID,
			})

			// Add tasks
			for _, i := range taskIndices {
				lines = append(lines, lineInfo{content: a.renderTaskByDisplayIndex(i, orderedIndices), taskIndex: i})
			}
		}
	}

	return a.renderScrollableLines(lines, orderedIndices, maxHeight)
}

// renderGroupedTasks renders tasks grouped by due status.
func (a *App) renderGroupedTasks(maxHeight int) string {
	var overdue, today, other []int

	// Group tasks
	for i, t := range a.tasks {
		if t.IsOverdue() {
			overdue = append(overdue, i)
		} else if t.IsDueToday() {
			today = append(today, i)
		} else {
			other = append(other, i)
		}
	}

	// Build ordered indices
	var orderedIndices []int
	orderedIndices = append(orderedIndices, overdue...)
	orderedIndices = append(orderedIndices, today...)
	orderedIndices = append(orderedIndices, other...)

	// Build lines
	var lines []lineInfo

	if len(overdue) > 0 {
		lines = append(lines, lineInfo{content: styles.SectionHeader.Render("OVERDUE"), taskIndex: -1})
		for _, i := range overdue {
			lines = append(lines, lineInfo{content: a.renderTaskByDisplayIndex(i, orderedIndices), taskIndex: i})
		}
	}

	if len(today) > 0 {
		// Add blank line before header if there are previous sections
		if len(overdue) > 0 {
			lines = append(lines, lineInfo{content: "", taskIndex: -1})
			// Only show separation if we have overdue tasks
		}

		for _, i := range today {
			lines = append(lines, lineInfo{content: a.renderTaskByDisplayIndex(i, orderedIndices), taskIndex: i})
		}
	}

	if len(other) > 0 {
		// Add blank line before header if there are previous sections
		if len(overdue) > 0 || len(today) > 0 {
			lines = append(lines, lineInfo{content: "", taskIndex: -1})
		}
		lines = append(lines, lineInfo{content: styles.SectionHeader.Render("NO DUE DATE"), taskIndex: -1})
		for _, i := range other {
			lines = append(lines, lineInfo{content: a.renderTaskByDisplayIndex(i, orderedIndices), taskIndex: i})
		}
	}

	return a.renderScrollableLines(lines, orderedIndices, maxHeight)
}

// renderFlatTasks renders tasks in a flat list.
func (a *App) renderFlatTasks(maxHeight int) string {
	var lines []lineInfo
	var orderedIndices []int

	for i := range a.tasks {
		orderedIndices = append(orderedIndices, i)
		lines = append(lines, lineInfo{content: a.renderTaskByDisplayIndex(i, orderedIndices), taskIndex: i})
	}

	return a.renderScrollableLines(lines, orderedIndices, maxHeight)
}

// renderSectionHeaderByIndex renders a section header with cursor highlighting for empty sections.
func (a *App) renderSectionHeaderByIndex(sectionName string, headerIndex int, orderedIndices []int) string {
	// Find display position for cursor
	displayPos := 0
	for i, idx := range orderedIndices {
		if idx == headerIndex {
			displayPos = i
			break
		}
	}

	// Check if cursor is on this empty section header
	isCursorHere := displayPos == a.taskCursor && a.focusedPane == PaneMain

	// Cursor indicator
	cursor := "  "
	if isCursorHere {
		cursor = "> "
	}

	// Build the header line
	line := fmt.Sprintf("%s%s", cursor, sectionName)

	// Apply style - use TaskSelected style when cursor is here, otherwise SectionHeader
	style := styles.SectionHeader
	if isCursorHere {
		style = styles.TaskSelected
	}

	return style.Render(line)
}

// renderTaskByDisplayIndex renders a task with cursor based on display order.
func (a *App) renderTaskByDisplayIndex(taskIndex int, orderedIndices []int) string {
	t := a.tasks[taskIndex]

	// Find display position for cursor
	displayPos := 0
	for i, idx := range orderedIndices {
		if idx == taskIndex {
			displayPos = i
			break
		}
	}

	// Cursor
	cursor := "  "
	if displayPos == a.taskCursor && a.focusedPane == PaneMain {
		cursor = "> "
	}

	// Selection indicator
	selectionMark := " "
	if a.selectedTaskIDs[t.ID] {
		selectionMark = "●"
	}

	// Checkbox
	checkbox := styles.CheckboxUnchecked
	if t.Checked {
		checkbox = styles.CheckboxChecked
	}

	// Indent for subtasks
	indent := ""
	if t.ParentID != nil {
		indent = "  "
	}

	// Content with priority color
	content := t.Content
	priorityStyle := styles.GetPriorityStyle(t.Priority)
	content = priorityStyle.Render(content)

	// Due date
	due := ""
	if t.Due != nil {
		dueStr := t.DueDisplay()
		if t.IsOverdue() {
			due = styles.TaskDueOverdue.Render("| " + dueStr)
		} else if t.IsDueToday() {
			due = styles.TaskDueToday.Render("| " + dueStr)
		} else {
			due = styles.TaskDue.Render("| " + dueStr)
		}
	}

	// Labels
	labels := ""
	if len(t.Labels) > 0 {
		labelStrs := make([]string, len(t.Labels))
		for i, l := range t.Labels {
			labelStrs[i] = "@" + l
		}
		labels = styles.TaskLabel.Render(strings.Join(labelStrs, " "))
	}

	// Build line with selection mark
	line := fmt.Sprintf("%s%s%s%s %s %s %s", cursor, selectionMark, indent, checkbox, content, due, labels)

	// Apply style
	style := styles.TaskItem
	if displayPos == a.taskCursor && a.focusedPane == PaneMain {
		style = styles.TaskSelected
	}
	if t.Checked {
		style = styles.TaskCompleted
	}

	return style.Render(line)
}

// renderScrollableLines renders lines with scrolling support using viewport.
func (a *App) renderScrollableLines(lines []lineInfo, orderedIndices []int, maxHeight int) string {
	// Store ordered indices for use in handleSelect
	a.taskOrderedIndices = orderedIndices

	if len(lines) == 0 {
		a.scrollOffset = 0
		a.viewportLines = nil
		a.viewportSections = nil
		return ""
	}

	// Build content string and track line->task mapping and section mapping
	var content strings.Builder
	a.viewportLines = make([]int, 0, len(lines))
	a.viewportSections = make([]string, 0, len(lines))

	for i, line := range lines {
		content.WriteString(line.content)
		if i < len(lines)-1 {
			content.WriteString("\n")
		}
		// Map this viewport line to its task index (-1 for headers, -2 for section headers)
		a.viewportLines = append(a.viewportLines, line.taskIndex)
		// Map this viewport line to its section ID (empty string for non-section lines)
		a.viewportSections = append(a.viewportSections, line.sectionID)
	}

	// Find which line the cursor is on
	cursorLine := 0
	if a.taskCursor >= 0 && a.taskCursor < len(orderedIndices) {
		targetTaskIndex := orderedIndices[a.taskCursor]
		for i, line := range lines {
			if line.taskIndex == targetTaskIndex {
				cursorLine = i
				break
			}
		}
	}

	// If viewport is ready, use it for scrolling
	if a.viewportReady {
		// Update viewport height if needed (maxHeight is the available height)
		if a.taskViewport.Height != maxHeight && maxHeight > 0 {
			a.taskViewport.Height = maxHeight
		}

		// Set content to viewport
		a.taskViewport.SetContent(content.String())

		// Sync viewport to show cursor
		a.syncViewportToCursor(cursorLine)

		// Store scroll offset for click handling
		a.scrollOffset = a.taskViewport.YOffset

		return a.taskViewport.View()
	}

	// Fallback: viewport not ready, just return raw content (truncated)
	a.scrollOffset = 0
	return content.String()
}

// renderTaskDetail renders the task detail view.
func (a *App) renderTaskDetail() string {
	if a.selectedTask == nil {
		return "No task selected"
	}

	t := a.selectedTask
	var b strings.Builder

	// Title with checkbox status
	checkbox := "[ ]"
	if t.Checked {
		checkbox = "[x]"
	}
	b.WriteString(styles.Title.Render("Task Details"))
	b.WriteString("\n\n")

	// Task content (main title)
	priorityStyle := styles.GetPriorityStyle(t.Priority)
	b.WriteString(fmt.Sprintf("  %s %s\n\n", checkbox, priorityStyle.Render(t.Content)))

	// Horizontal divider
	b.WriteString(styles.DetailSection.Render("  " + strings.Repeat("─", 40)))
	b.WriteString("\n\n")

	// Description (if present)
	if t.Description != "" {
		b.WriteString(styles.DetailIcon.Render("  📝"))
		b.WriteString(styles.DetailLabel.Render("Description"))
		b.WriteString("\n")
		b.WriteString(styles.DetailDescription.Render(t.Description))
		b.WriteString("\n\n")
	}

	// Due date
	if t.Due != nil {
		dueIcon := "📅"
		dueStyle := styles.DetailValue
		if t.IsOverdue() {
			dueIcon = "🔴"
			dueStyle = styles.TaskDueOverdue
		} else if t.IsDueToday() {
			dueIcon = "🟢"
			dueStyle = styles.TaskDueToday
		}
		b.WriteString(styles.DetailIcon.Render("  " + dueIcon))
		b.WriteString(styles.DetailLabel.Render("Due"))
		b.WriteString(dueStyle.Render(t.Due.String))
		if t.Due.IsRecurring {
			b.WriteString(styles.HelpDesc.Render(" (recurring)"))
		}
		b.WriteString("\n")
	}

	// Priority
	priorityIcon := "⚪"
	priorityLabel := "P4 (Low)"
	switch t.Priority {
	case 4:
		priorityIcon = "🔴"
		priorityLabel = "P1 (Urgent)"
	case 3:
		priorityIcon = "🟠"
		priorityLabel = "P2 (High)"
	case 2:
		priorityIcon = "🟡"
		priorityLabel = "P3 (Medium)"
	}
	b.WriteString(styles.DetailIcon.Render("  " + priorityIcon))
	b.WriteString(styles.DetailLabel.Render("Priority"))
	b.WriteString(priorityStyle.Render(priorityLabel))
	b.WriteString("\n")

	// Labels
	if len(t.Labels) > 0 {
		b.WriteString(styles.DetailIcon.Render("  🏷️"))
		b.WriteString(styles.DetailLabel.Render("Labels"))
		for i, l := range t.Labels {
			if i > 0 {
				b.WriteString(" ")
			}
			b.WriteString(styles.TaskLabel.Render("@" + l))
		}
		b.WriteString("\n")
	}

	// Project (find name)
	if t.ProjectID != "" {
		projectName := t.ProjectID
		for _, p := range a.projects {
			if p.ID == t.ProjectID {
				projectName = p.Name
				break
			}
		}
		b.WriteString(styles.DetailIcon.Render("  📁"))
		b.WriteString(styles.DetailLabel.Render("Project"))
		b.WriteString(styles.DetailValue.Render(projectName))
		b.WriteString("\n")
	}

	// Comment count
	if t.NoteCount > 0 {
		b.WriteString(styles.DetailIcon.Render("  💬"))
		b.WriteString(styles.DetailLabel.Render("Comments"))
		b.WriteString(styles.DetailValue.Render(fmt.Sprintf("%d", t.NoteCount)))
		b.WriteString("\n")
	}

	// Comments section
	if len(a.comments) > 0 {
		b.WriteString("\n")
		b.WriteString(styles.DetailSection.Render("  " + strings.Repeat("─", 40)))
		b.WriteString("\n")
		b.WriteString(styles.Subtitle.Render("  Comments"))
		b.WriteString("\n\n")

		for _, c := range a.comments {
			// Parse and format timestamp
			timestamp := c.PostedAt
			if t, err := time.Parse(time.RFC3339, c.PostedAt); err == nil {
				timestamp = t.Format("Jan 2, 2006 3:04 PM")
			}
			b.WriteString(styles.CommentAuthor.Render(fmt.Sprintf("    %s", timestamp)))
			b.WriteString("\n")
			b.WriteString(styles.CommentContent.Render(fmt.Sprintf("    %s", c.Content)))
			b.WriteString("\n\n")
		}
	}

	// Divider before help
	b.WriteString(styles.DetailSection.Render("  " + strings.Repeat("─", 40)))
	b.WriteString("\n\n")

	// Help section
	b.WriteString(styles.HelpDesc.Render("  Shortcuts: "))
	b.WriteString(styles.HelpKey.Render("ESC"))
	b.WriteString(styles.HelpDesc.Render(" back  "))
	b.WriteString(styles.HelpKey.Render("x"))
	b.WriteString(styles.HelpDesc.Render(" complete  "))
	b.WriteString(styles.HelpKey.Render("e"))
	b.WriteString(styles.HelpDesc.Render(" edit  "))
	b.WriteString(styles.HelpKey.Render("s"))
	b.WriteString(styles.HelpDesc.Render(" add subtask"))

	return styles.Dialog.Width(a.width - 4).Render(b.String())
}

// renderTaskForm renders the add/edit task form.
func (a *App) renderTaskForm() string {
	if a.taskForm == nil {
		return styles.Dialog.Width(a.width - 4).Render("Form not initialized")
	}

	return styles.Dialog.Width(a.width - 4).Render(a.taskForm.View())
}

// renderHelp renders the help view.
func (a *App) renderHelp() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	items := a.keymap.HelpItems()
	for _, item := range items {
		if item[0] == "" {
			b.WriteString("\n")
			continue
		}
		if item[1] == "" {
			// Section header
			b.WriteString(styles.Subtitle.Render(item[0]))
			b.WriteString("\n")
			continue
		}
		key := styles.HelpKey.Render(fmt.Sprintf("%-12s", item[0]))
		desc := styles.HelpDesc.Render(item[1])
		b.WriteString(fmt.Sprintf("  %s %s\n", key, desc))
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("Press any key to close"))

	return styles.Dialog.Width(a.width - 4).Render(b.String())
}

// renderSearch renders the search view.
func (a *App) renderSearch() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Search Tasks"))
	b.WriteString("\n\n")

	// Search input
	b.WriteString(styles.InputLabel.Render("Query"))
	b.WriteString("\n")
	b.WriteString(a.searchInput.View())
	b.WriteString("\n\n")

	// Results
	if a.searchQuery == "" {
		b.WriteString(styles.HelpDesc.Render("Type to search..."))
	} else if len(a.searchResults) == 0 {
		b.WriteString(styles.StatusBarError.Render("No results found"))
	} else {
		b.WriteString(styles.Subtitle.Render(fmt.Sprintf("Found %d task(s)", len(a.searchResults))))
		b.WriteString("\n\n")

		// Render search results
		for i, task := range a.searchResults {
			cursor := "  "
			itemStyle := styles.TaskItem
			if i == a.taskCursor {
				cursor = "> "
				itemStyle = styles.TaskSelected
			}

			checkbox := styles.CheckboxUnchecked
			if task.Checked {
				checkbox = styles.CheckboxChecked
			}

			content := task.Content
			priorityStyle := styles.GetPriorityStyle(task.Priority)
			content = priorityStyle.Render(content)

			// Due date
			due := ""
			if task.Due != nil {
				dueStr := task.DueDisplay()
				if task.IsOverdue() {
					due = styles.TaskDueOverdue.Render(" | " + dueStr)
				} else if task.IsDueToday() {
					due = styles.TaskDueToday.Render(" | " + dueStr)
				} else {
					due = styles.TaskDue.Render(" | " + dueStr)
				}
			}

			line := fmt.Sprintf("%s%s %s%s", cursor, checkbox, content, due)
			b.WriteString(itemStyle.Render(line))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("j/k: navigate | Enter: view | x: complete | Esc: back"))

	return styles.Dialog.Width(a.width - 4).Render(b.String())
}

// renderUpcoming renders the upcoming view with tasks grouped by date.
func (a *App) renderUpcoming(maxHeight int) string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Upcoming"))
	b.WriteString("\n")

	if a.loading {
		b.WriteString(a.spinner.View())
		b.WriteString(" Loading...")
		return b.String()
	}

	if len(a.tasks) == 0 {
		b.WriteString("\n")
		b.WriteString(styles.HelpDesc.Render("No upcoming tasks"))
		return b.String()
	}

	// Group tasks by date
	tasksByDate := make(map[string][]int)
	var dates []string

	for i, t := range a.tasks {
		if t.Due == nil {
			continue
		}
		date := t.Due.Date
		if _, exists := tasksByDate[date]; !exists {
			dates = append(dates, date)
		}
		tasksByDate[date] = append(tasksByDate[date], i)
	}

	// Sort dates
	sort.Strings(dates)

	// Build ordered indices for cursor mapping
	var orderedIndices []int
	for _, date := range dates {
		orderedIndices = append(orderedIndices, tasksByDate[date]...)
	}

	// Build lines
	var lines []lineInfo

	for idx, date := range dates {
		// Parse date for display
		displayDate := date
		if parsed, err := time.Parse("2006-01-02", date); err == nil {
			today := time.Now().Format("2006-01-02")
			tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
			switch date {
			case today:
				displayDate = "Today"
			case tomorrow:
				displayDate = "Tomorrow"
			default:
				displayDate = parsed.Format("Mon, Jan 2")
			}
		}

		// Add blank line before header (except first) for spacing
		if idx > 0 {
			lines = append(lines, lineInfo{content: "", taskIndex: -1})
		}

		lines = append(lines, lineInfo{
			content:   styles.DateGroupHeader.Render(displayDate),
			taskIndex: -1,
		})

		for _, i := range tasksByDate[date] {
			lines = append(lines, lineInfo{
				content:   a.renderTaskByDisplayIndex(i, orderedIndices),
				taskIndex: i,
			})
		}
	}

	// Use common scrollable rendering - maxHeight already accounts for borders
	// Subtract 2 for Title line + newline
	result := a.renderScrollableLines(lines, orderedIndices, maxHeight-2)
	b.WriteString(result)

	return b.String()
}

// renderLabelsView renders the labels view.
func (a *App) renderLabelsView(maxHeight int) string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Labels"))
	b.WriteString("\n\n")

	// Account for title + blank line (2 lines used)
	contentHeight := maxHeight - 2

	if a.currentLabel != nil {
		// Show tasks for selected label
		labelTitle := "@" + a.currentLabel.Name
		if a.currentLabel.Color != "" {
			labelTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(a.currentLabel.Color)).Render(labelTitle)
		}
		b.WriteString(styles.Subtitle.Render(labelTitle))
		b.WriteString("\n\n")

		// Account for subtitle + blank line + footer (4 more lines)
		taskHeight := contentHeight - 4

		if len(a.tasks) == 0 {
			b.WriteString(styles.HelpDesc.Render("No tasks with this label"))
		} else {
			// Build lines and ordered indices for scrolling - FIX: populate content field
			var lines []lineInfo
			var orderedIndices []int
			for i := range a.tasks {
				orderedIndices = append(orderedIndices, i)
			}
			for i := range a.tasks {
				lines = append(lines, lineInfo{
					content:   a.renderTaskByDisplayIndex(i, orderedIndices),
					taskIndex: i,
				})
			}
			b.WriteString(a.renderScrollableLines(lines, orderedIndices, taskHeight))
		}

		b.WriteString("\n")
		b.WriteString(styles.HelpDesc.Render("Press ESC to go back to labels list"))
	} else {
		// Extract unique labels from all tasks if personal labels are empty
		labelsToShow := a.labels
		if len(labelsToShow) == 0 {
			labelsToShow = a.extractLabelsFromTasks()
		}

		// Build task count map for labels
		taskCountMap := a.getLabelTaskCounts()

		// Account for footer (2 lines)
		labelHeight := contentHeight - 2

		// Show list of labels
		if len(labelsToShow) == 0 {
			b.WriteString(styles.HelpDesc.Render("No labels found"))
		} else {
			// Calculate scroll window for labels
			startIdx := 0
			if a.taskCursor >= labelHeight {
				startIdx = a.taskCursor - labelHeight + 1
			}
			endIdx := startIdx + labelHeight
			if endIdx > len(labelsToShow) {
				endIdx = len(labelsToShow)
			}

			for i := startIdx; i < endIdx; i++ {
				label := labelsToShow[i]
				cursor := "  "
				style := styles.LabelItem
				if i == a.taskCursor && a.focusedPane == PaneMain {
					cursor = "> "
					style = styles.LabelSelected
				}

				// Label name with optional color
				name := "@" + label.Name
				if label.Color != "" {
					name = lipgloss.NewStyle().Foreground(lipgloss.Color(label.Color)).Render(name)
				}

				// Task count badge
				taskCount := taskCountMap[label.Name]
				countBadge := ""
				if taskCount > 0 {
					countBadge = styles.HelpDesc.Render(fmt.Sprintf(" (%d)", taskCount))
				}

				line := fmt.Sprintf("%s%s%s", cursor, name, countBadge)
				b.WriteString(style.Render(line))
				b.WriteString("\n")
			}
		}

		b.WriteString("\n")
		b.WriteString(styles.HelpDesc.Render("Press Enter to view tasks with label"))
	}

	return b.String()
}

// getLabelTaskCounts returns a map of label name to task count.
func (a *App) getLabelTaskCounts() map[string]int {
	counts := make(map[string]int)

	// Use allTasks if available, otherwise fall back to tasks
	tasksToScan := a.allTasks
	if len(tasksToScan) == 0 {
		tasksToScan = a.tasks
	}

	for _, t := range tasksToScan {
		for _, labelName := range t.Labels {
			counts[labelName]++
		}
	}

	return counts
}

// extractLabelsFromTasks extracts unique labels from all tasks.
func (a *App) extractLabelsFromTasks() []api.Label {
	labelSet := make(map[string]bool)
	var labels []api.Label

	// Check allTasks first, fall back to tasks
	tasksToScan := a.allTasks
	if len(tasksToScan) == 0 {
		tasksToScan = a.tasks
	}

	for _, t := range tasksToScan {
		for _, labelName := range t.Labels {
			if !labelSet[labelName] {
				labelSet[labelName] = true
				labels = append(labels, api.Label{
					Name: labelName,
				})
			}
		}
	}

	// Sort labels alphabetically
	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Name < labels[j].Name
	})

	return labels
}

// renderCalendar renders the calendar view (dispatches based on view mode).
func (a *App) renderCalendar(maxHeight int) string {
	if a.calendarViewMode == CalendarViewExpanded {
		return a.renderCalendarExpanded(maxHeight)
	}
	return a.renderCalendarCompact(maxHeight)
}

// renderCalendarCompact renders the compact calendar view.
func (a *App) renderCalendarCompact(maxHeight int) string {
	var b strings.Builder

	// Header with month/year and navigation hints
	monthYear := a.calendarDate.Format("January 2006")
	b.WriteString(styles.Title.Render(monthYear))
	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("← → prev/next month | h l prev/next day | v toggle view"))
	b.WriteString("\n\n")

	// Weekday headers
	weekdays := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	for _, wd := range weekdays {
		b.WriteString(styles.CalendarWeekday.Render(fmt.Sprintf(" %s ", wd)))
	}
	b.WriteString("\n")

	// Calculate first day and number of days in month
	firstOfMonth := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	startWeekday := int(firstOfMonth.Weekday())
	daysInMonth := lastOfMonth.Day()
	today := time.Now()

	// Build map of tasks by day
	tasksByDay := make(map[int]int) // day -> count
	for _, t := range a.allTasks {
		if t.Due == nil {
			continue
		}
		if parsed, err := time.Parse("2006-01-02", t.Due.Date); err == nil {
			if parsed.Year() == a.calendarDate.Year() && parsed.Month() == a.calendarDate.Month() {
				tasksByDay[parsed.Day()]++
			}
		}
	}

	// Render calendar grid and count weeks rendered
	day := 1
	weeksRendered := 0
	for week := 0; week < 6; week++ {
		if day > daysInMonth {
			break
		}
		weeksRendered++

		for weekday := 0; weekday < 7; weekday++ {
			if week == 0 && weekday < startWeekday {
				b.WriteString("     ")
				continue
			}

			if day > daysInMonth {
				b.WriteString("     ")
				continue
			}

			dayStr := fmt.Sprintf(" %2d ", day)
			style := styles.CalendarDay

			// Check if this is today
			isToday := today.Year() == a.calendarDate.Year() &&
				today.Month() == a.calendarDate.Month() &&
				today.Day() == day

			// Check if this day has tasks
			hasTasks := tasksByDay[day] > 0

			// Check if this is the selected day
			isSelected := day == a.calendarDay && a.focusedPane == PaneMain

			// Check if this is a weekend (Friday=5, Saturday=6 in Jordan)
			isWeekend := weekday == 5 || weekday == 6

			if isSelected {
				style = styles.CalendarDaySelected
			} else if isToday {
				style = styles.CalendarDayToday
			} else if hasTasks {
				style = styles.CalendarDayWithTasks
			} else if isWeekend {
				style = styles.CalendarDayWeekend
			}

			// Add task indicator
			if hasTasks && !isSelected {
				dayStr = fmt.Sprintf(" %2d*", day)
			}

			b.WriteString(style.Render(dayStr))
			b.WriteString(" ")
			day++
		}
		b.WriteString("\n")
	}

	// Show tasks for selected day
	b.WriteString("\n")
	selectedDate := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), a.calendarDay, 0, 0, 0, 0, time.Local)
	b.WriteString(styles.Subtitle.Render(selectedDate.Format("Monday, January 2")))
	b.WriteString("\n\n")

	// Find tasks for selected day
	var dayTasks []api.Task
	selectedDateStr := selectedDate.Format("2006-01-02")
	for _, t := range a.allTasks {
		if t.Due != nil && t.Due.Date == selectedDateStr {
			dayTasks = append(dayTasks, t)
		}
	}

	// Calculate remaining height for task list
	// Used: title(1) + help(1) + blank(1) + weekdays(1) + calendar(weeksRendered) + blank(1) + subtitle(1) + blank(1)
	usedHeight := 7 + weeksRendered
	taskListHeight := maxHeight - usedHeight
	if taskListHeight < 1 {
		taskListHeight = 1
	}

	if len(dayTasks) == 0 {
		b.WriteString(styles.HelpDesc.Render("No tasks for this day"))
	} else {
		// Calculate scroll window
		startIdx := 0
		if a.taskCursor >= taskListHeight {
			startIdx = a.taskCursor - taskListHeight + 1
		}
		endIdx := startIdx + taskListHeight
		if endIdx > len(dayTasks) {
			endIdx = len(dayTasks)
		}

		for i := startIdx; i < endIdx; i++ {
			t := dayTasks[i]
			checkbox := styles.CheckboxUnchecked
			if t.Checked {
				checkbox = styles.CheckboxChecked
			}
			priorityStyle := styles.GetPriorityStyle(t.Priority)
			content := priorityStyle.Render(t.Content)

			cursor := "  "
			if i == a.taskCursor && a.focusedPane == PaneMain {
				cursor = "> "
			}
			b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, checkbox, content))
		}
	}

	return b.String()
}

// renderCalendarExpanded renders the expanded calendar view with task names in cells.
func (a *App) renderCalendarExpanded(maxHeight int) string {
	var b strings.Builder

	// Header with month/year and navigation hints
	monthYear := a.calendarDate.Format("January 2006")
	b.WriteString(styles.Title.Render(monthYear))
	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("← → prev/next month | h l prev/next day | v toggle view"))
	b.WriteString("\n\n")

	// Calculate cell dimensions based on terminal width
	// 7 columns + borders (8 vertical lines)
	availableWidth := a.width - 8 // Subtract for borders
	if availableWidth < 35 {
		availableWidth = 35 // Minimum width
	}
	cellWidth := availableWidth / 7
	if cellWidth < 5 {
		cellWidth = 5
	}
	if cellWidth > 20 {
		cellWidth = 20 // Max cell width
	}

	// Weekday headers
	weekdays := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	headerLine := "│"
	for _, wd := range weekdays {
		header := fmt.Sprintf(" %-*s", cellWidth-1, wd)
		if len(header) > cellWidth {
			header = header[:cellWidth]
		}
		headerLine += styles.CalendarWeekday.Render(header) + "│"
	}
	b.WriteString(headerLine)
	b.WriteString("\n")

	// Top border
	topBorder := "├" + strings.Repeat(strings.Repeat("─", cellWidth)+"┼", 6) + strings.Repeat("─", cellWidth) + "┤\n"
	b.WriteString(topBorder)

	// Calculate first day and number of days in month
	firstOfMonth := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	startWeekday := int(firstOfMonth.Weekday())
	daysInMonth := lastOfMonth.Day()
	today := time.Now()

	// Build map of tasks by day
	tasksByDay := make(map[int][]api.Task) // day -> tasks
	for _, t := range a.allTasks {
		if t.Due == nil {
			continue
		}
		if parsed, err := time.Parse("2006-01-02", t.Due.Date); err == nil {
			if parsed.Year() == a.calendarDate.Year() && parsed.Month() == a.calendarDate.Month() {
				tasksByDay[parsed.Day()] = append(tasksByDay[parsed.Day()], t)
			}
		}
	}

	// Calculate how many weeks we need to display
	weeksNeeded := (daysInMonth + startWeekday + 6) / 7

	// Calculate how many task lines to show per cell based on available height
	// Header(1) + help(1) + blank(1) + weekday(1) + topBorder(1) + statusBar(1) = 6 lines overhead
	// Each week uses: 1 (day number) + maxTasksPerCell (tasks) + 1 (separator) = 2 + maxTasksPerCell
	// Last week doesn't have separator, so: weeksNeeded * (2 + maxTasksPerCell) - 1 + 6 = maxHeight
	availableForWeeks := maxHeight - 6
	if availableForWeeks < weeksNeeded*3 {
		availableForWeeks = weeksNeeded * 3 // Minimum 1 task line per cell
	}
	// Each week row = 1 (day) + tasks + 1 (separator, except last)
	// Solve for maxTasksPerCell: weeksNeeded*(1+tasks+1) - 1 = availableForWeeks
	// weeksNeeded*(2+tasks) = availableForWeeks + 1
	// tasks = (availableForWeeks + 1) / weeksNeeded - 2
	maxTasksPerCell := (availableForWeeks+1)/weeksNeeded - 2
	if maxTasksPerCell < 2 {
		maxTasksPerCell = 2
	}
	if maxTasksPerCell > 6 {
		maxTasksPerCell = 6 // Cap at 6 tasks per cell
	}

	// Render calendar grid
	day := 1
	for week := 0; week < 6; week++ {
		if day > daysInMonth {
			break
		}

		// Row 1: Day numbers
		dayNumLine := "│"
		for weekday := 0; weekday < 7; weekday++ {
			if week == 0 && weekday < startWeekday || day > daysInMonth {
				dayNumLine += strings.Repeat(" ", cellWidth) + "│"
				if week == 0 && weekday < startWeekday {
					continue
				}
				continue
			}

			dayStr := fmt.Sprintf(" %2d", day)
			style := styles.CalendarDay

			isToday := today.Year() == a.calendarDate.Year() &&
				today.Month() == a.calendarDate.Month() &&
				today.Day() == day
			isSelected := day == a.calendarDay && a.focusedPane == PaneMain
			isWeekend := weekday == 5 || weekday == 6
			hasTasks := len(tasksByDay[day]) > 0

			if isSelected {
				style = styles.CalendarDaySelected
			} else if isToday {
				style = styles.CalendarDayToday
			} else if hasTasks {
				style = styles.CalendarDayWithTasks
			} else if isWeekend {
				style = styles.CalendarDayWeekend
			}

			// Pad to cell width
			paddedDay := fmt.Sprintf("%-*s", cellWidth, dayStr)
			dayNumLine += style.Render(paddedDay) + "│"
			day++
		}
		b.WriteString(dayNumLine)
		b.WriteString("\n")

		// Reset day counter for task rows
		day -= 7
		if day < 1 {
			day = 1
		}

		// Rows 2-3: Task previews
		for taskLine := 0; taskLine < maxTasksPerCell; taskLine++ {
			taskRow := "│"
			tempDay := day
			for weekday := 0; weekday < 7; weekday++ {
				if week == 0 && weekday < startWeekday {
					taskRow += strings.Repeat(" ", cellWidth) + "│"
					continue
				}

				if tempDay > daysInMonth {
					taskRow += strings.Repeat(" ", cellWidth) + "│"
					tempDay++
					continue
				}

				tasks := tasksByDay[tempDay]
				var cellContent string

				if taskLine < len(tasks) && taskLine < maxTasksPerCell-1 {
					// Show task name with priority color (truncated to fit cell)
					task := tasks[taskLine]
					taskName := task.Content
					maxLen := cellWidth - 2 // Leave space for " " prefix and margin
					if len(taskName) > maxLen && maxLen > 1 {
						taskName = taskName[:maxLen-1] + "…"
					}
					// Pad the plain text first, then apply priority color
					paddedTask := fmt.Sprintf(" %-*s", cellWidth-1, taskName)
					priorityStyle := styles.GetPriorityStyle(task.Priority)
					cellContent = priorityStyle.Render(paddedTask)
				} else if taskLine == maxTasksPerCell-1 && len(tasks) > maxTasksPerCell-1 {
					// Show "+N more" indicator on the last line if there are more tasks
					hiddenCount := len(tasks) - (maxTasksPerCell - 1)
					moreText := fmt.Sprintf("+%d more", hiddenCount)
					paddedMore := fmt.Sprintf(" %-*s", cellWidth-1, moreText)
					cellContent = styles.CalendarMoreTasks.Render(paddedMore)
				} else if taskLine < len(tasks) {
					// This handles the case where we're on the last allowed line but it's a task
					task := tasks[taskLine]
					taskName := task.Content
					maxLen := cellWidth - 2
					if len(taskName) > maxLen && maxLen > 1 {
						taskName = taskName[:maxLen-1] + "…"
					}
					paddedTask := fmt.Sprintf(" %-*s", cellWidth-1, taskName)
					priorityStyle := styles.GetPriorityStyle(task.Priority)
					cellContent = priorityStyle.Render(paddedTask)
				} else {
					// Empty cell
					cellContent = strings.Repeat(" ", cellWidth)
				}

				taskRow += cellContent + "│"
				tempDay++
			}
			b.WriteString(taskRow)
			b.WriteString("\n")
		}

		// Move day forward after processing the week
		day += 7
		if week == 0 {
			day = 8 - startWeekday
		}

		// Row separator (except for last week)
		if day <= daysInMonth {
			separator := "├" + strings.Repeat(strings.Repeat("─", cellWidth)+"┼", 6) + strings.Repeat("─", cellWidth) + "┤\n"
			b.WriteString(separator)
		}
	}

	// Bottom border
	bottomBorder := "└" + strings.Repeat(strings.Repeat("─", cellWidth)+"┴", 6) + strings.Repeat("─", cellWidth) + "┘\n"
	b.WriteString(bottomBorder)

	return b.String()
}

// renderCalendarDay renders the day detail view showing all tasks for the selected calendar day.
func (a *App) renderCalendarDay() string {
	var b strings.Builder

	// Header with date - styled nicely
	selectedDate := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), a.calendarDay, 0, 0, 0, 0, time.Local)

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Highlight).
		Background(lipgloss.Color("#1a1a2e")).
		Padding(0, 1).
		Width(a.width - 4)

	b.WriteString(titleStyle.Render("📅 " + selectedDate.Format("Monday, January 2, 2006")))
	b.WriteString("\n\n")

	if a.loading {
		b.WriteString(a.spinner.View())
		b.WriteString(" Loading tasks...")
		return b.String()
	}

	// Content area with border
	contentWidth := a.width - 4
	contentHeight := a.height - 6 // title + padding + status bar
	if contentHeight < 5 {
		contentHeight = 5
	}

	var content strings.Builder

	if len(a.tasks) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(styles.Subtle).
			Italic(true).
			Align(lipgloss.Center).
			Width(contentWidth - 4)

		content.WriteString("\n")
		content.WriteString(emptyStyle.Render("No tasks scheduled for this day"))
		content.WriteString("\n\n")
		content.WriteString(emptyStyle.Render("Press 'a' to add a new task"))
	} else {
		// Task count header
		countStyle := lipgloss.NewStyle().
			Foreground(styles.Subtle)
		content.WriteString(countStyle.Render(fmt.Sprintf("%d task(s)", len(a.tasks))))
		content.WriteString("\n\n")

		// Build task lines
		taskHeight := contentHeight - 4 // account for count header and padding
		if taskHeight < 3 {
			taskHeight = 3
		}

		var lines []lineInfo
		var orderedIndices []int
		for i := range a.tasks {
			orderedIndices = append(orderedIndices, i)
			lines = append(lines, lineInfo{
				content:   a.renderTaskByDisplayIndex(i, orderedIndices),
				taskIndex: i,
			})
		}

		content.WriteString(a.renderScrollableLines(lines, orderedIndices, taskHeight))
	}

	// Wrap in a nice container with good padding
	containerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Subtle).
		Padding(1, 3). // More vertical and horizontal padding
		MarginLeft(2).
		MarginRight(2).
		Width(contentWidth - 4) // Account for margins

	b.WriteString(containerStyle.Render(content.String()))

	return b.String()
}

// renderStatusBar renders the bottom status bar.
func (a *App) renderStatusBar() string {
	// Left side: status message or error
	left := ""
	if a.err != nil {
		left = styles.StatusBarError.Render(fmt.Sprintf("Error: %v", a.err))
	} else if a.statusMsg != "" {
		left = styles.StatusBarSuccess.Render(a.statusMsg)
	}

	// Right side: context-specific key hints (or just toggle hint if hidden)
	var right string
	if a.showHints {
		hints := a.getContextualHints()
		hints = append(hints, styles.StatusBarKey.Render("F1")+styles.StatusBarText.Render(":hide"))
		right = strings.Join(hints, " ")
	} else {
		right = styles.StatusBarKey.Render("F1") + styles.StatusBarText.Render(":keys")
	}

	// Calculate spacing
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := styles.StatusBar.GetHorizontalFrameSize()
	spacing := a.width - leftWidth - rightWidth - padding
	if spacing < 0 {
		spacing = 0
	}

	return styles.StatusBar.Width(a.width - padding).Render(left + strings.Repeat(" ", spacing) + right)
}

// getContextualHints returns context-specific key hints for the status bar.
func (a *App) getContextualHints() []string {
	key := func(k string) string { return styles.StatusBarKey.Render(k) }
	desc := func(d string) string { return styles.StatusBarText.Render(d) }

	switch a.currentTab {
	case TabToday, TabUpcoming:
		return []string{
			key("j/k") + desc(":nav"),
			key("x") + desc(":done"),
			key("e") + desc(":edit"),
			key("</>") + desc(":due"),
			key("r") + desc(":refresh"),
			key("?") + desc(":help"),
		}
	case TabLabels:
		if a.currentLabel != nil {
			return []string{
				key("j/k") + desc(":nav"),
				key("x") + desc(":done"),
				key("e") + desc(":edit"),
				key("Esc") + desc(":back"),
				key("?") + desc(":help"),
			}
		}
		return []string{
			key("j/k") + desc(":nav"),
			key("Enter") + desc(":select"),
			key("?") + desc(":help"),
			key("q") + desc(":quit"),
		}
	case TabCalendar:
		return []string{
			key("h/l") + desc(":day"),
			key("←/→") + desc(":month"),
			key("v") + desc(":view"),
			key("Enter") + desc(":select"),
			key("?") + desc(":help"),
		}
	case TabProjects:
		if a.focusedPane == PaneSidebar {
			return []string{
				key("j/k") + desc(":nav"),
				key("Enter") + desc(":select"),
				key("Tab") + desc(":pane"),
				key("?") + desc(":help"),
			}
		}
		return []string{
			key("j/k") + desc(":nav"),
			key("x") + desc(":done"),
			key("e") + desc(":edit"),
			key("Tab") + desc(":pane"),
			key("?") + desc(":help"),
		}
	default:
		return []string{
			key("j/k") + desc(":nav"),
			key("?") + desc(":help"),
			key("q") + desc(":quit"),
		}
	}
}
