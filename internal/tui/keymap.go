// Package tui provides the terminal user interface for Todoist.
package tui

import tea "github.com/charmbracelet/bubbletea"

// Key represents a key binding.
type Key struct {
	Key  string
	Help string
}

// Keymap contains all key bindings for the application.
type Keymap struct {
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
	AddTask      Key
	EditTask     Key
	DeleteTask   Key
	CompleteTask Key
	Priority1    Key
	Priority2    Key
	Priority3    Key
	Priority4    Key
	DueToday     Key
	DueTomorrow  Key

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
}

// DefaultKeymap returns the default Vim-style key bindings.
func DefaultKeymap() Keymap {
	return Keymap{
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
		AddTask:      Key{Key: "a", Help: "add task"},
		EditTask:     Key{Key: "e", Help: "edit task"},
		DeleteTask:   Key{Key: "d", Help: "delete (dd)"},
		CompleteTask: Key{Key: "x", Help: "complete/uncomplete"},
		Priority1:    Key{Key: "1", Help: "priority 1 (highest)"},
		Priority2:    Key{Key: "2", Help: "priority 2"},
		Priority3:    Key{Key: "3", Help: "priority 3"},
		Priority4:    Key{Key: "4", Help: "priority 4 (lowest)"},
		DueToday:     Key{Key: "<", Help: "due today"},
		DueTomorrow:  Key{Key: ">", Help: "due tomorrow"},

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
	}
}

// KeyState tracks multi-key sequences (like 'gg' or 'dd').
type KeyState struct {
	LastKey  string
	WaitingG bool // Waiting for second 'g' in 'gg'
	WaitingD bool // Waiting for second 'd' in 'dd'
}

// HandleKey processes a key press and returns the action to take.
// Returns the action name and whether the key was consumed.
func (ks *KeyState) HandleKey(msg tea.KeyMsg, keymap Keymap) (string, bool) {
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
	case keymap.CompleteTask.Key:
		return "complete", true
	case "u":
		return "undo", true
	case "S":
		return "manage_sections", true
	case keymap.Priority1.Key:
		return "priority1", true
	case keymap.Priority2.Key:
		return "priority2", true
	case keymap.Priority3.Key:
		return "priority3", true
	case keymap.Priority4.Key:
		return "priority4", true
	case keymap.DueToday.Key:
		return "due_today", true
	case keymap.DueTomorrow.Key:
		return "due_tomorrow", true
	case keymap.SwitchPane.Key:
		return "switch_pane", true
	case keymap.FocusTasks.Key:
		return "focus_tasks", true
	case keymap.FocusProjects.Key:
		return "focus_projects", true
	case keymap.Search.Key:
		return "search", true
	case keymap.CalendarView.Key:
		return "calendar_view", true
	case keymap.NewProject.Key:
		return "new_project", true
	// Tab navigation shortcuts
	case "T":
		return "tab_today", true
	case "U":
		return "tab_upcoming", true
	case "P":
		return "tab_projects", true
	case "L":
		return "tab_labels", true
	case "C":
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
	ks.LastKey = ""
}

// HelpItems returns a slice of key-description pairs for the help view.
func (k Keymap) HelpItems() [][]string {
	return [][]string{
		{"Navigation", ""},
		{k.Up.Key + "/" + k.Down.Key, "Move up/down"},
		{"gg/G", "Go to top/bottom"},
		{k.HalfUp.Key + "/" + k.HalfDown.Key, "Half page up/down"},
		{k.SwitchPane.Key, "Switch pane"},
		{"", ""},
		{"Task Actions", ""},
		{k.Select.Key, "Open task details"},
		{k.AddTask.Key, "Add new task"},
		{k.EditTask.Key, "Edit task"},
		{k.CompleteTask.Key, "Complete/uncomplete task"},
		{"dd", "Delete task"},
		{"1-4", "Set priority"},
		{k.DueToday.Key + "/" + k.DueTomorrow.Key, "Due today/tomorrow"},
		{"", ""},
		{"Calendar", ""},
		{k.CalendarView.Key, "Switch calendar view"},
		{"h/l", "Previous/next day"},
		{"←/→", "Previous/next month"},
		{"", ""},
		{"General", ""},
		{k.Refresh.Key, "Refresh data"},
		{k.Search.Key, "Search"},
		{k.Help.Key, "Toggle help"},
		{k.Back.Key, "Go back / Cancel"},
		{k.Quit.Key, "Quit"},
	}
}
