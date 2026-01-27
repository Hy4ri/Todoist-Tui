// Package tui provides the terminal user interface for Todoist.
package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	allSections    []api.Section // All sections for instant project switching
	labels         []api.Label
	comments       []api.Comment // Comments for selected task
	selectedTask   *api.Task
	currentProject *api.Project
	currentLabel   *api.Label

	// Sidebar items (only shown in Projects tab)
	sidebarItems  []components.SidebarItem
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

	app.helpComp.SetKeymap(app.keymap.HelpItems())

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

// loadInitialData loads all necessary data concurrently.
func (a *App) loadInitialData() tea.Cmd {
	return func() tea.Msg {
		var (
			projects    []api.Project
			labels      []api.Label
			allTasks    []api.Task
			allSections []api.Section
		)

		// Create channels for results
		type projectResult struct {
			data []api.Project
			err  error
		}
		type labelResult struct {
			data []api.Label
			err  error
		}
		type taskResult struct {
			data []api.Task
			err  error
		}
		type sectionResult struct {
			data []api.Section
			err  error
		}

		projChan := make(chan projectResult)
		labelChan := make(chan labelResult)
		taskChan := make(chan taskResult)
		secChan := make(chan sectionResult)

		// Launch concurrent requests
		go func() {
			p, e := a.client.GetProjects()
			projChan <- projectResult{data: p, err: e}
		}()

		go func() {
			l, e := a.client.GetLabels()
			labelChan <- labelResult{data: l, err: e}
		}()

		go func() {
			t, e := a.client.GetTasks(api.TaskFilter{})
			taskChan <- taskResult{data: t, err: e}
		}()

		go func() {
			s, e := a.client.GetSections("")
			secChan <- sectionResult{data: s, err: e}
		}()

		// Collect results
		pRes := <-projChan
		if pRes.err != nil {
			return errMsg{pRes.err}
		}
		projects = pRes.data

		lRes := <-labelChan
		if lRes.err != nil {
			return errMsg{lRes.err}
		}
		labels = lRes.data

		tRes := <-taskChan
		if tRes.err != nil {
			return errMsg{tRes.err}
		}
		allTasks = tRes.data

		sRes := <-secChan
		if sRes.err != nil {
			return errMsg{sRes.err}
		}
		allSections = sRes.data

		// Filter tasks for the initial view
		var initialTasks []api.Task
		switch a.currentTab {
		case TabUpcoming:
			for _, t := range allTasks {
				if t.Due != nil {
					initialTasks = append(initialTasks, t)
				}
			}
		case TabCalendar:
			// In calendar main view, we don't necessarily show a list initially,
			// or we show allTasks. DataLoadedMsg handler will handle the display.
			initialTasks = nil
		case TabProjects, TabLabels:
			// Items are selected from sidebar/list
			initialTasks = nil
		default:
			// TabToday or fallback
			for _, t := range allTasks {
				if t.IsOverdue() || t.IsDueToday() {
					initialTasks = append(initialTasks, t)
				}
			}
		}

		return dataLoadedMsg{
			projects:    projects,
			tasks:       initialTasks,
			allTasks:    allTasks,
			labels:      labels,
			allSections: allSections,
		}
	}
}

// Message types
type errMsg struct{ err error }
type statusMsg struct{ msg string }
type dataLoadedMsg struct {
	projects    []api.Project
	tasks       []api.Task
	allTasks    []api.Task
	sections    []api.Section
	allSections []api.Section
	labels      []api.Label
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

type reorderCompleteMsg struct{}
