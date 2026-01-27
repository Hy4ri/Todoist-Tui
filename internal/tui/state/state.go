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

// KeyState tracks key presses.
type KeyState struct {
	// Add fields if needed from original app definition, assumed empty or trivial
}

// Keymap defines keybindings.
type Keymap interface {
	HelpItems() []components.HelpItem
	// Add other necessary methods or struct definition
}

// TaskForm is the form model.
type TaskForm struct {
	Content     textinput.Model
	Description textinput.Model
	Priority    int
	DueString   textinput.Model
	ProjectID   string
	SectionID   string
	Labels      []string
	Original    *api.Task
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
	Projects       []api.Project
	Tasks          []api.Task
	AllTasks       []api.Task
	Sections       []api.Section
	AllSections    []api.Section
	Labels         []api.Label
	Comments       []api.Comment
	SelectedTask   *api.Task
	CurrentProject *api.Project
	CurrentLabel   *api.Label

	// Sidebar items
	SidebarItems  []components.SidebarItem
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
	Keymap  interface{} // Using interface{} to avoid circular dependency on tui.Keymap if it remains there

	// UI Components
	SidebarComp *components.SidebarModel
	DetailComp  *components.DetailModel
	HelpComp    *components.HelpModel

	// Form state
	TaskForm *TaskForm

	// Search state
	SearchQuery   string
	SearchInput   textinput.Model
	SearchResults []api.Task
	IsSearching   bool

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
	SectionInput         textinput.Model
	IsCreatingSection    bool
	IsEditingSection     bool
	EditingSection       *api.Section
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
	CommentInput    textinput.Model
	IsAddingComment bool

	// Viewport Data
	ViewportContent    string
	ViewportLines      []int
	TaskOrderedIndices []int
	ViewportSections   []string

	// Selection state
	SelectedTaskIDs map[string]bool

	// Cursor restoration
	RestoreCursorToTaskID string
}
