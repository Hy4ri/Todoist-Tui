package state

import tea "github.com/charmbracelet/bubbletea"

// Key represents a key binding.
type Key struct {
	Key  string
	Help string
}

// KeymapData contains all key bindings for the application.
// Renamed from Keymap to KeymapData to avoid conflict with interface if any
type KeymapData struct {
	// Navigation
	Up       Key
	Down     Key
	Top      Key
	Bottom   Key
	HalfUp   Key
	HalfDown Key
	Left     Key
	Right    Key

	// Actions
	Select  Key
	Back    Key
	Quit    Key
	Help    Key
	Refresh Key

	// Task actions
	AddTask         Key
	EditTask        Key
	DeleteTask      Key
	CompleteTask    Key
	Priority1       Key
	Priority2       Key
	Priority3       Key
	Priority4       Key
	DueToday        Key
	DueTomorrow     Key
	MoveTaskPrevDay Key
	MoveTaskNextDay Key

	// Navigation between panes
	SwitchPane    Key
	FocusTasks    Key
	FocusProjects Key

	// Search/Filter
	Search Key

	// Calendar
	CalendarView Key

	// Project actions
	NewProject Key

	// Section actions
	NewSection  Key
	MoveSection Key
}

// DefaultKeymap returns the default Vim-style key bindings.
func DefaultKeymap() KeymapData {
	return KeymapData{
		// Navigation
		Up:       Key{Key: "k", Help: "up"},
		Down:     Key{Key: "j", Help: "down"},
		Top:      Key{Key: "g", Help: "top (gg)"},
		Bottom:   Key{Key: "G", Help: "bottom"},
		HalfUp:   Key{Key: "ctrl+u", Help: "half page up"},
		HalfDown: Key{Key: "ctrl+d", Help: "half page down"},
		Left:     Key{Key: "h", Help: "left"},
		Right:    Key{Key: "l", Help: "right"},

		// Actions
		Select:  Key{Key: "enter", Help: "select"},
		Back:    Key{Key: "esc", Help: "back"},
		Quit:    Key{Key: "q", Help: "quit"},
		Help:    Key{Key: "?", Help: "help"},
		Refresh: Key{Key: "r", Help: "refresh"},

		// Task actions
		AddTask:         Key{Key: "a", Help: "add task"},
		EditTask:        Key{Key: "e", Help: "edit task"},
		DeleteTask:      Key{Key: "d", Help: "delete (dd)"},
		CompleteTask:    Key{Key: "x", Help: "complete/uncomplete"},
		Priority1:       Key{Key: "1", Help: "priority 1 (highest)"},
		Priority2:       Key{Key: "2", Help: "priority 2"},
		Priority3:       Key{Key: "3", Help: "priority 3"},
		Priority4:       Key{Key: "4", Help: "priority 4 (lowest)"},
		DueToday:        Key{Key: "<", Help: "due today"},
		DueTomorrow:     Key{Key: ">", Help: "due tomorrow"},
		MoveTaskPrevDay: Key{Key: "H", Help: "move -1 day"},
		MoveTaskNextDay: Key{Key: "L", Help: "move +1 day"},

		// Navigation between panes
		SwitchPane:    Key{Key: "tab", Help: "switch pane"},
		FocusTasks:    Key{Key: "t", Help: "focus tasks"},
		FocusProjects: Key{Key: "p", Help: "focus projects"},

		// Search
		Search: Key{Key: "/", Help: "search"},

		// Calendar
		CalendarView: Key{Key: "v", Help: "switch calendar view"},

		// Project actions
		NewProject: Key{Key: "n", Help: "new project"},

		// Section actions
		NewSection:  Key{Key: "s", Help: "new section"},
		MoveSection: Key{Key: "m", Help: "move task/section"},
	}
}

// KeyState tracks multi-key sequences (like 'gg' or 'dd' or 'yy').
type KeyState struct {
	LastKey  string
	WaitingG bool // Waiting for second 'g' in 'gg'
	WaitingD bool // Waiting for second 'd' in 'dd'
	WaitingY bool // Waiting for second 'y' in 'yy'
}

// HandleKey processes a key press and returns the action to take.
// Returns the action name and whether the key was consumed.
func (ks *KeyState) HandleKey(msg tea.KeyMsg, km interface{}) (string, bool) {
	keymap, ok := km.(KeymapData)
	if !ok {
		// Fallback or log error? safely return false
		if msg.String() == "?" {
			return "help", true
		}
		return "", false
	}

	key := msg.String()

	// Handle 'gg' sequence (go to top)
	if ks.WaitingG {
		ks.WaitingG = false
		if key == "g" {
			return "top", true
		}
		// If not 'g', reset and process normally
	}

	// Handle 'dd' sequence (delete)
	if ks.WaitingD {
		ks.WaitingD = false
		if key == "d" {
			return "delete", true
		}
		// If not 'd', reset and process normally
	}

	// Handle 'yy' sequence (copy)
	if ks.WaitingY {
		ks.WaitingY = false
		if key == "y" {
			return "copy", true
		}
		// If not 'y', reset and process normally
	}

	// Check for multi-key sequence starts
	if key == "g" {
		ks.WaitingG = true
		ks.LastKey = key
		return "", true // Key consumed, waiting for next
	}

	if key == "d" {
		ks.WaitingD = true
		ks.LastKey = key
		return "", true // Key consumed, waiting for next
	}

	if key == "y" {
		ks.WaitingY = true
		ks.LastKey = key
		return "", true // Key consumed, waiting for next
	}

	// Single key mappings
	switch key {
	case keymap.Up.Key, "up":
		return "up", true
	case keymap.Down.Key, "down":
		return "down", true
	case keymap.Bottom.Key:
		return "bottom", true
	case keymap.HalfUp.Key:
		return "half_up", true
	case keymap.HalfDown.Key:
		return "half_down", true
	case keymap.Left.Key, "left":
		return "left", true
	case keymap.Right.Key, "right":
		return "right", true
	case keymap.Select.Key:
		return "select", true
	case keymap.Back.Key:
		return "back", true
	case keymap.Quit.Key:
		return "quit", true
	case keymap.Help.Key:
		return "help", true
	case keymap.Refresh.Key:
		return "refresh", true
	case keymap.AddTask.Key:
		return "add", true
	case keymap.EditTask.Key:
		return "edit", true
	case "s":
		return "add_subtask", true
	case "S":
		return "manage_sections", true
	case keymap.CompleteTask.Key:
		return "complete", true
	case "ctrl+z":
		return "undo", true
	case "m":
		return "move_task", true
	case "M":
		return "move_section", true
	case "A": // Remapped from c
		return "add_comment", true
	case " ": // Space key for selection
		return "toggle_select", true
	case keymap.Priority1.Key, "!":
		return "priority1", true
	case keymap.Priority2.Key, "@":
		return "priority2", true
	case keymap.Priority3.Key, "#":
		return "priority3", true
	case keymap.Priority4.Key, "$":
		return "priority4", true
	case keymap.DueToday.Key:
		return "due_today", true
	case keymap.DueTomorrow.Key:
		return "due_tomorrow", true
	case keymap.SwitchPane.Key:
		return "switch_pane", true
	case keymap.Search.Key:
		return "search", true
	case keymap.CalendarView.Key:
		return "calendar_view", true
	case keymap.NewProject.Key:
		return "new_project", true

	// Tab navigation shortcuts (Case insensitive t, u, p, l, c)
	case "t", "T":
		return "tab_today", true
	case "u", "U":
		return "tab_upcoming", true
	case "p", "P":
		return "tab_projects", true
	case "l":
		return "right", true
	case "L":
		return "move_task_next_day", true
	case "H":
		return "move_task_prev_day", true
	case "c", "C":
		return "tab_calendar", true

	// Hints toggle
	case "f1":
		return "toggle_hints", true
	}

	return "", false
}

// Reset clears any pending multi-key sequences.
func (ks *KeyState) Reset() {
	ks.WaitingG = false
	ks.WaitingD = false
	ks.WaitingY = false
	ks.LastKey = ""
}

// HelpItems returns a slice of key-description pairs for the help view.
func (k KeymapData) HelpItems() [][]string {
	return [][]string{
		{"Navigation", ""},
		{k.Up.Key + "/" + k.Down.Key, "Move up/down"},
		{"gg/G", "Go to top/bottom"},
		{k.HalfUp.Key + "/" + k.HalfDown.Key, "Half page up/down"},
		{k.SwitchPane.Key, "Switch pane (Sidebar/Main)"},
		{"H/L", "Move task -1/+1 day"},
		{"", ""},
		{"View Switching", ""},
		{"i/1", "Inbox"},
		{"t/2", "Today's tasks"},
		{"u/3", "Upcoming tasks"},
		{"u/3", "Upcoming tasks"},
		{"4", "Labels"},
		{"c/5", "Calendar"},
		{"p/6", "Projects"},
		{"", ""},
		{"Task Actions", ""},
		{k.Select.Key, "Open task details"},
		{k.AddTask.Key, "Add new task"},
		{k.EditTask.Key, "Edit task"},
		{k.CompleteTask.Key, "Complete/uncomplete task"},
		{"dd", "Delete task"},
		{"yy", "Copy task(s) to clipboard"},
		{"Space", "Toggle task selection"},
		{"1-4", "Set priority"},
		{k.DueToday.Key + "/" + k.DueTomorrow.Key, "Due today/tomorrow"},
		{"s", "Add subtask"},
		{"m", "Move task to section"},
		{"A", "Add comment"},
		{"ctrl+z", "Undo last action"},
		{"", ""},
		{"Section/Project Actions", ""},
		{"n", "New project (in sidebar)"},
		{"S", "Manage sections (add/edit/delete)"},
		{"M", "Reorder sections"},
		{"", ""},
		{"Calendar", ""},
		{k.CalendarView.Key, "Switch calendar view (Compact/Expanded)"},
		{"h/l", "Previous/next day"},
		{"[/]", "Previous/next month"},
		{"", ""},
		{"General", ""},
		{"Shift+D", "Set default view"},
		{k.Refresh.Key, "Refresh data"},
		{k.Search.Key, "Search"},
		{k.Help.Key, "Toggle help"},
		{k.Back.Key, "Go back / Cancel"},
		{"f1", "Toggle key hints"},
		{k.Quit.Key, "Quit"},
	}
}
