package components

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

// Pane represents which pane is currently focused.
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

// SidebarItem represents an item in the sidebar (special views or projects).
type SidebarItem struct {
	Type       string // "special", "separator", "project"
	ID         string // View name for special, project ID for projects
	Name       string
	Icon       string
	Count      int
	IsFavorite bool
	ParentID   *string
}

// LastAction represents an undoable action.
type LastAction struct {
	Type   string // "complete", "uncomplete"
	TaskID string
}

// lineInfo represents a display line with optional task reference.
type LineInfo struct {
	Content   string
	TaskIndex int // -1 for non-task lines (headers, separators)
}
