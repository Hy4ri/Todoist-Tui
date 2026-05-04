package components

// View represents the available views in the application.
type View int

const (
	ViewInbox View = iota
	ViewToday
	ViewUpcoming
	ViewProjects
	ViewCalendar
	ViewLabels
	ViewCompleted
)

// Tab represents the available tabs in the application.
type Tab int

const (
	TabInbox Tab = iota
	TabToday
	TabUpcoming
	TabProjects
	TabCalendar
	TabLabels
	TabCompleted
)

// Pane represents a UI pane that can be focused.
type Pane int

const (
	SidebarPane Pane = iota
	MainPane
	DetailPane
)

// CalendarViewMode represents the calendar display mode.
type CalendarViewMode int

const (
	CalendarViewCompact CalendarViewMode = iota
	CalendarViewExpanded
)

// SidebarItem represents an item in the sidebar (project, label, etc.).
type SidebarItem struct {
	Type       string // "project", "label", "separator"
	ID         string
	Name       string
	Icon       string
	Count      int
	IsFavorite bool
	ParentID   *string
	Color      string
}

// LastAction tracks the last performed action for undo support.
type LastAction struct {
	Action string
	TaskID string
	Task   string
}

// LineInfo holds information about a rendered line.
type LineInfo struct {
	Offset int
	Length int
	ItemID string
}