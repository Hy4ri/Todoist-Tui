package state

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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
	// Maybe buffer for chords?
}

// HandleKey processes a key message and returns an action.
func (k *KeyState) HandleKey(msg tea.KeyMsg, km interface{}) (string, bool) {
	// This is a simplified version since we lost the original.
	// Ideally it should use keybubble or similar if that was used.
	// For now, let's assume direct keymap lookup if possible or return false to let default handler work.
	// The original code passed 'h.Keymap' which is interface{}.

	// If we can't restore the complex logic easily, let's make it a pass-through for now
	// or implement basic key translation if we know the keymap structure.

	// Check if HelpItems() is the only method known.
	// If the user presses '?', return "help".
	if msg.String() == "?" {
		return "help", true
	}

	return "", false
}

// Keymap defines keybindings.
type Keymap interface {
	HelpItems() []components.HelpItem
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
	// Data
	Projects []api.Project
	Tasks    []api.Task

	// UI Elements
	SidebarItems []components.SidebarItem

	AllTasks       []api.Task
	Sections       []api.Section
	AllSections    []api.Section
	Labels         []api.Label
	Comments       []api.Comment
	SelectedTask   *api.Task
	CurrentProject *api.Project
	CurrentLabel   *api.Label

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
		{TabToday, "[T]", "Today", "Tdy"},
		{TabUpcoming, "[U]", "Upcoming", "Up"},
		{TabLabels, "[L]", "Labels", "Lbl"},
		{TabCalendar, "[C]", "Calendar", "Cal"},
		{TabProjects, "[P]", "Projects", "Prj"},
	}
}
