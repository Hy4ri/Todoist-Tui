// Package styles provides Lip Gloss styles for the TUI.
package styles

import "github.com/charmbracelet/lipgloss"

// Terminal-adaptive colors that work in both light and dark terminals.
var (
	// Subtle is a muted color for secondary text
	Subtle = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#999999"}

	// Highlight is the accent color for selected items
	Highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#990000"}

	// Special colors
	ErrorColor   = lipgloss.AdaptiveColor{Light: "#FF0000", Dark: "#FF6666"}
	SuccessColor = lipgloss.AdaptiveColor{Light: "#00AA00", Dark: "#66FF66"}
	WarningColor = lipgloss.AdaptiveColor{Light: "#FFAA00", Dark: "#FFCC66"}
)

// Priority colors (Todoist uses P1=red, P2=orange, P3=yellow, P4=default)
var (
	Priority1Color = lipgloss.Color("#D0473D") // P1 - Red (highest)
	Priority2Color = lipgloss.Color("#EA8811") // P2 - Orange
	Priority3Color = lipgloss.Color("#296FDF") // P3 - Blue
	Priority4Color = lipgloss.Color("")        // P4 - Default (no color)
)

// Base styles
var (
	// App is the base style for the entire application
	App = lipgloss.NewStyle().
		Padding(1, 2)

	// Title is the style for section titles
	// NOTE: No margins - they break viewport scroll sync line counting
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Highlight)

	// Subtitle is for secondary headings
	// NOTE: No margins - they break viewport scroll sync line counting
	Subtitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Subtle)
)

// Task styles
var (
	// TaskItem is the base style for a task item
	TaskItem = lipgloss.NewStyle().
			PaddingLeft(2)

	// TaskSelected is the style for a selected task
	TaskSelected = lipgloss.NewStyle().
			PaddingLeft(1).
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeftForeground(Highlight).
			Bold(true).
			Background(lipgloss.AdaptiveColor{Light: "#EEEEEE", Dark: "#2A2A2A"})

	// TaskCompleted is the style for completed tasks
	TaskCompleted = lipgloss.NewStyle().
			PaddingLeft(2).
			Faint(true).
			Strikethrough(true)

	// TaskContent is for the task name/content
	TaskContent = lipgloss.NewStyle()

	// TaskDue is for due date display
	TaskDue = lipgloss.NewStyle().
		Foreground(Subtle).
		PaddingLeft(1)

	// TaskDueOverdue is for overdue tasks
	TaskDueOverdue = lipgloss.NewStyle().
			Foreground(ErrorColor).
			PaddingLeft(1)

	// TaskDueToday is for tasks due today
	TaskDueToday = lipgloss.NewStyle().
			Foreground(SuccessColor).
			PaddingLeft(1)

	// TaskLabel is for label display
	TaskLabel = lipgloss.NewStyle().
			Foreground(Highlight).
			PaddingLeft(1)

	// TaskRecurring is for the recurring icon
	TaskRecurring = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#00AAAA", Dark: "#00CCCC"}).
			PaddingLeft(1)

	// TaskListDescription is for descriptions in task lists
	TaskListDescription = lipgloss.NewStyle().
				Foreground(Subtle).
				Faint(true).
				Italic(true).
				PaddingLeft(10) // Matches cursor + selection + indent + checkbox margin
)

// Priority styles
var (
	TaskPriority1 = lipgloss.NewStyle().Foreground(Priority1Color)
	TaskPriority2 = lipgloss.NewStyle().Foreground(Priority2Color)
	TaskPriority3 = lipgloss.NewStyle().Foreground(Priority3Color)
	TaskPriority4 = lipgloss.NewStyle()
)

// GetPriorityStyle returns the appropriate style for a task priority.
func GetPriorityStyle(priority int) lipgloss.Style {
	switch priority {
	case 4:
		return TaskPriority1 // Todoist uses 4 as highest priority
	case 3:
		return TaskPriority2
	case 2:
		return TaskPriority3
	default:
		return TaskPriority4
	}
}

// Project styles
var (
	// ProjectItem is the base style for a project item
	ProjectItem = lipgloss.NewStyle().
			PaddingLeft(1)

	// ProjectSelected is the style for a selected project
	ProjectSelected = lipgloss.NewStyle().
			PaddingLeft(1).
			Bold(true).
			Background(lipgloss.AdaptiveColor{Light: "#EEEEEE", Dark: "#333333"})

	// ProjectInbox is for the Inbox project
	ProjectInbox = lipgloss.NewStyle().
			PaddingLeft(1).
			Bold(true)
)

// Sidebar styles
var (
	// Sidebar is the style for the sidebar container
	Sidebar = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Subtle).
		Padding(0, 1)

	// SidebarFocused is for when the sidebar is focused
	SidebarFocused = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Highlight).
			Padding(0, 1)

	// SidebarSeparator is for separator lines in the sidebar
	SidebarSeparator = lipgloss.NewStyle().
				Foreground(Subtle).
				Faint(true)

	// SidebarActive is for the currently active item when sidebar is not focused
	SidebarActive = lipgloss.NewStyle().
			PaddingLeft(1).
			Foreground(Highlight).
			Bold(true)
)

// Main content area styles
var (
	// MainContent is the style for the main content area
	MainContent = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Subtle).
			Padding(0, 1)

	// MainContentFocused is for when main content is focused
	MainContentFocused = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(Highlight).
				Padding(0, 1)

	// DetailPanel is the style for the detail panel container
	DetailPanel = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Subtle).
			Padding(0, 1)

	// DetailPanelFocused is for when the detail panel is focused
	DetailPanelFocused = lipgloss.NewStyle().
				BorderStyle(lipgloss.RoundedBorder()).
				BorderForeground(Highlight).
				Padding(0, 1)
)

// StatusBar styles
var (
	// StatusBar is the base style for the status bar
	StatusBar = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#333333", Dark: "#DDDDDD"}).
			Background(lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#1F1F1F"}).
			Padding(0, 1)

	// StatusBarKey is for keyboard shortcut hints
	StatusBarKey = lipgloss.NewStyle().
			Bold(true).
			Foreground(Highlight).
			Background(lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#1F1F1F"})

	// StatusBarText is for status bar descriptions
	StatusBarText = lipgloss.NewStyle().
			Foreground(Subtle).
			Background(lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#1F1F1F"})

	// StatusBarError is for error messages
	StatusBarError = lipgloss.NewStyle().
			Foreground(ErrorColor).
			Background(lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#1F1F1F"}).
			Bold(true)

	// StatusBarSuccess is for success messages
	StatusBarSuccess = lipgloss.NewStyle().
				Foreground(SuccessColor).
				Background(lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#1F1F1F"}).
				Bold(true)
)

// Help styles
var (
	// HelpKey is for key bindings in help
	HelpKey = lipgloss.NewStyle().
		Bold(true).
		Foreground(Highlight)

	// HelpDesc is for key binding descriptions
	HelpDesc = lipgloss.NewStyle().
			Foreground(Subtle)

	// HelpSeparator is the separator between key and description
	HelpSeparator = lipgloss.NewStyle().
			Foreground(Subtle)
)

// Input styles
var (
	// Input is the style for text inputs
	Input = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Subtle).
		Padding(0, 1)

	// InputFocused is for focused inputs
	InputFocused = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(Highlight).
			Padding(0, 1)

	// InputLabel is for input labels
	InputLabel = lipgloss.NewStyle().
			Bold(true).
			MarginBottom(1)
)

// Dialog styles
var (
	// Dialog is the base style for dialog boxes
	Dialog = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Highlight).
		Padding(1, 2)

	// DialogTitle is for dialog titles
	DialogTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Highlight).
			MarginBottom(1)
)

// Spinner style
var (
	Spinner = lipgloss.NewStyle().
		Foreground(Highlight)
)

// Section header style
// NOTE: No margins here - they add extra lines that break viewport scroll sync.
var (
	SectionHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(Subtle).
		Underline(true)
)

// Calendar styles
// NOTE: Width is NOT set here - it's calculated dynamically in renderCalendarExpanded
var (
	// CalendarHeader is for month/year header
	CalendarHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(Highlight).
			Align(lipgloss.Center)

	// CalendarWeekday is for day-of-week headers
	CalendarWeekday = lipgloss.NewStyle().
			Foreground(Subtle)

	// CalendarDay is for regular days
	CalendarDay = lipgloss.NewStyle()

	// CalendarDaySelected is for the selected day
	CalendarDaySelected = lipgloss.NewStyle().
				Bold(true).
				Background(Highlight).
				Foreground(lipgloss.Color("#ffffff"))

	// CalendarDayToday is for today's date
	CalendarDayToday = lipgloss.NewStyle().
				Bold(true).
				Foreground(SuccessColor)

	// CalendarDayWithTasks is for days that have tasks
	CalendarDayWithTasks = lipgloss.NewStyle().
				Foreground(WarningColor)

	// CalendarDayOtherMonth is for days from other months
	CalendarDayOtherMonth = lipgloss.NewStyle().
				Faint(true)
)

// Label styles
var (
	LabelItem = lipgloss.NewStyle().
			PaddingLeft(1)

	LabelSelected = lipgloss.NewStyle().
			PaddingLeft(1).
			Bold(true).
			Background(lipgloss.AdaptiveColor{Light: "#EEEEEE", Dark: "#333333"})

	LabelBadge = lipgloss.NewStyle().
			Foreground(Highlight).
			Bold(true)
)

// Date group header for upcoming view
// NOTE: No margins or borders here - they add extra lines that break viewport scroll sync.
// Using Underline() instead of BorderBottom() keeps headers to single lines.
var (
	DateGroupHeader = lipgloss.NewStyle().
		Bold(true).
		Foreground(Highlight).
		Underline(true)
)

// Checkbox styles
const (
	CheckboxUnchecked = "[ ]"
	CheckboxChecked   = "[x]"
)

// Task Detail styles
var (
	// DetailIcon is for icons in task detail view
	DetailIcon = lipgloss.NewStyle().
			Foreground(Highlight).
			PaddingRight(1)

	// DetailLabel is for field labels in task detail
	DetailLabel = lipgloss.NewStyle().
			Foreground(Subtle).
			Bold(true).
			Width(12)

	// DetailValue is for field values in task detail
	DetailValue = lipgloss.NewStyle().
			PaddingLeft(1)

	// DetailSection is for section dividers
	DetailSection = lipgloss.NewStyle().
			Foreground(Subtle).
			MarginTop(1).
			MarginBottom(1)

	// DetailDescription is for task description
	DetailDescription = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#444444", Dark: "#CCCCCC"}).
				PaddingLeft(2).
				MarginTop(1)

	// SubtaskItem is for subtask display
	SubtaskItem = lipgloss.NewStyle().
			PaddingLeft(4)

	// SubtaskSelected is for selected subtask
	SubtaskSelected = lipgloss.NewStyle().
			PaddingLeft(4).
			Bold(true).
			Background(lipgloss.AdaptiveColor{Light: "#EEEEEE", Dark: "#333333"})
)

// Tab bar styles
var (
	// TabBar is the container for the tab bar
	TabBar = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		BorderForeground(Subtle).
		PaddingLeft(1).
		PaddingRight(1)

	// Tab is for inactive tabs
	Tab = lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(Subtle)

	// TabActive is for the active tab
	TabActive = lipgloss.NewStyle().
			Padding(0, 2).
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(Highlight)

	// TabHover is for hovered tabs (mouse support)
	TabHover = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(Highlight)
)

// Comment styles
var (
	// CommentItem is for individual comments
	CommentItem = lipgloss.NewStyle().
			PaddingLeft(2).
			MarginBottom(1)

	// CommentAuthor is for comment author/timestamp
	CommentAuthor = lipgloss.NewStyle().
			Foreground(Subtle).
			Faint(true)

	// CommentContent is for comment text
	CommentContent = lipgloss.NewStyle().
			PaddingLeft(2)
)

// Scroll indicator styles
var (
	// ScrollIndicatorUp shows there's more content above
	ScrollIndicatorUp = lipgloss.NewStyle().
				Foreground(Subtle).
				Italic(true).
				PaddingLeft(2)

	// ScrollIndicatorDown shows there's more content below
	ScrollIndicatorDown = lipgloss.NewStyle().
				Foreground(Subtle).
				Italic(true).
				PaddingLeft(2)
)

// Calendar expanded view styles
var (
	// CalendarCellBorder is for cell borders in expanded calendar
	CalendarCellBorder = lipgloss.NewStyle().
				Foreground(Subtle)

	// CalendarDayWeekend is for weekend days (Friday/Saturday in Jordan)
	CalendarDayWeekend = lipgloss.NewStyle().
				Foreground(Subtle)

	// CalendarTaskPreview is for task names in expanded calendar cells
	CalendarTaskPreview = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#444444", Dark: "#BBBBBB"})

	// CalendarMoreTasks is for "+N more" indicator in cells
	CalendarMoreTasks = lipgloss.NewStyle().
				Foreground(Subtle).
				Italic(true)
)
