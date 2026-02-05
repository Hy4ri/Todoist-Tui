package state

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/config"
	"github.com/hy4ri/todoist-tui/internal/tui/components"
)

// View represents the current view/screen.
type View int

const (
	ViewInbox View = iota
	ViewToday
	ViewUpcoming
	ViewLabels
	ViewCalendar
	ViewCalendarDay // Day detail view from calendar
	ViewProject
	ViewTaskDetail
	ViewTaskForm
	ViewQuickAdd
	ViewSearch
	ViewHelp
	ViewSections
)

// Tab represents a top-level tab.
type Tab int

const (
	TabInbox Tab = iota
	TabToday
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

// Keymap defines keybindings.
type Keymap interface {
	HelpItems() [][]string
}

// State holds the application state.
// All fields are exported to allow access from logic and ui packages.
type State struct {
	// Dependencies
	Client *api.Client
	Config *config.Config

	// View state
	CurrentView  View
	PreviousView View
	CurrentTab   Tab
	FocusedPane  Pane

	// Data
	Projects []api.Project
	Tasks    []api.Task

	// DataVersion tracks the version of data to help with caching
	DataVersion int64

	// Performance optimization: track when data was last fetched from API
	LastDataFetch time.Time

	// Performance optimization: cache comments per task to avoid re-fetching
	CommentCache map[string][]api.Comment

	// Performance optimization: track if tasks are already sorted
	TasksSorted bool

	// UI Elements
	SidebarItems []components.SidebarItem

	AllTasks          []api.Task
	Sections          []api.Section
	AllSections       []api.Section
	Labels            []api.Label
	Comments          []api.Comment
	TasksByDate       map[string][]api.Task
	SelectedTask      *api.Task
	CurrentProject    *api.Project
	CurrentLabel      *api.Label
	ProductivityStats *api.ProductivityStats
	StatsError        string

	// Sidebar items

	SidebarCursor int

	// List state
	ProjectCursor int
	TaskCursor    int
	ScrollOffset  int

	// Calendar state
	CalendarDate     time.Time
	CalendarDay      int
	CalendarViewMode CalendarViewMode

	// UI state
	Loading         bool
	Err             error
	StatusMsg       string
	Width           int
	Height          int
	ShowHints       bool
	LastAction      *LastAction
	ShowDetailPanel bool

	// Components
	Spinner spinner.Model
	Keymap  Keymap // Using interface to allow different implementations

	// UI Components
	SidebarComp *components.SidebarModel
	DetailComp  *components.DetailModel
	HelpComp    *components.HelpModel

	// Form state
	TaskForm     *TaskForm
	QuickAddForm *QuickAddForm

	// Search state
	SearchQuery   string
	SearchInput   textinput.Model
	SearchResults []api.Task
	IsSearching   bool

	// Color selection state
	IsSelectingColor bool
	SelectedColor    string
	ColorCursor      int
	AvailableColors  []string

	// New project state
	ProjectInput         textinput.Model
	IsCreatingProject    bool
	IsEditingProject     bool
	EditingProject       *api.Project
	ConfirmDeleteProject bool

	// New label state
	LabelInput         textinput.Model
	IsCreatingLabel    bool
	IsEditingLabel     bool
	EditingLabel       *api.Label
	ConfirmDeleteLabel bool

	// Section state
	SectionInput      textinput.Model
	IsCreatingSection bool
	IsEditingSection  bool
	EditingSection    *api.Section

	// Key handling
	KeyState *KeyState

	ConfirmDeleteSection bool

	// Subtask creation state
	SubtaskInput      textinput.Model
	IsCreatingSubtask bool
	ParentTaskID      string

	// Viewport
	TaskViewport  viewport.Model
	ViewportReady bool

	// Move task state
	IsMovingTask      bool
	MoveSectionCursor int

	// Comment state
	CommentInput         textinput.Model
	IsAddingComment      bool
	IsEditingComment     bool
	EditingComment       *api.Comment
	ConfirmDeleteComment bool

	// Viewport Data
	ViewportContent    string
	ViewportLines      []int
	TaskOrderedIndices []int
	ViewportSections   []string

	// Selection state
	SelectedTaskIDs map[string]bool

	// Notification state
	NotifiedTasks map[string]bool

	// Cursor restoration
	RestoreCursorToTaskID string

	// Performance optimization: cached tab bar rendering
	CachedTabBar      string
	CachedTabBarTab   Tab
	CachedTabBarWidth int
}

// TabInfo holds tab metadata.
type TabInfo struct {
	Tab       Tab
	Icon      string
	Name      string
	ShortName string
}

// GetTabDefinitions returns the tab definitions.
func GetTabDefinitions() []TabInfo {
	return []TabInfo{
		{TabInbox, "📥", "Inbox", "Inb"},
		{TabToday, "📅", "Today", "Tdy"},
		{TabUpcoming, "📆", "Upcoming", "Up"},
		{TabLabels, "🏷️", "Labels", "Lbl"},
		{TabCalendar, "🗓️", "Calendar", "Cal"},
		{TabProjects, "📂", "Projects", "Prj"},
	}
}
