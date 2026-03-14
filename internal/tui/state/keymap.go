package state

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

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
	AddTaskFull     Key
	EditTask        Key
	DeleteTask      Key
	CompleteTask    Key
	Priority1       Key
	Priority2       Key
	Priority3       Key
	Priority4       Key
	MoveTaskPrevDay Key
	MoveTaskNextDay Key
	AddComment      Key
	RescheduleTask  Key
	IndentTask      Key
	OutdentTask     Key

	// Navigation between panes
	SwitchPane Key

	// Search/Filter
	Search Key

	// Project actions
	NewProject Key

	// Section actions
	NewSection     Key
	MoveSection    Key
	MoveToProject  Key
	Reminder       Key
	SendToPomodoro Key
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
		AddTaskFull:     Key{Key: "A", Help: "add task (full)"},
		EditTask:        Key{Key: "e", Help: "edit task"},
		DeleteTask:      Key{Key: "d", Help: "delete (dd)"},
		CompleteTask:    Key{Key: "x", Help: "complete/uncomplete"},
		Priority1:       Key{Key: "1", Help: "priority 1 (highest)"},
		Priority2:       Key{Key: "2", Help: "priority 2"},
		Priority3:       Key{Key: "3", Help: "priority 3"},
		Priority4:       Key{Key: "4", Help: "priority 4 (lowest)"},
		MoveTaskPrevDay: Key{Key: "<", Help: "move -1 day"},
		MoveTaskNextDay: Key{Key: ">", Help: "move +1 day"},
		AddComment:      Key{Key: "C", Help: "add/view comments"},
		RescheduleTask:  Key{Key: "t", Help: "smart reschedule"},
		IndentTask:      Key{Key: "L", Help: "indent task (subtask)"},
		OutdentTask:     Key{Key: "H", Help: "outdent task"},

		// Navigation between panes
		SwitchPane: Key{Key: "tab", Help: "switch pane"},

		// Search
		Search: Key{Key: "/", Help: "search"},

		// Project actions
		NewProject: Key{Key: "n", Help: "new project"},

		MoveToProject:  Key{Key: "v", Help: "move to project"},
		Reminder:       Key{Key: "R", Help: "manage reminders"},
		SendToPomodoro: Key{Key: "p", Help: "send to pomodoro"},

		// Map 'f' generic action logic will handle context
	}
}

// ApplyOverrides applies user-defined key binding overrides from the config file.
// The overrides map is keyed by action name (snake_case) and maps to the
// replacement key string. Unknown action names are silently ignored.
// If multiple actions would be bound to the same key, the conflict is
// skipped and a warning string is returned for each conflict.
// Example: {"add_task": "o", "complete": "c"}
func (k *KeymapData) ApplyOverrides(overrides map[string]string) []string {
	if len(overrides) == 0 {
		return nil
	}

	// Explicit action → field pointer map avoids reflection and keeps things type-safe.
	actions := map[string]*string{
		"up":               &k.Up.Key,
		"down":             &k.Down.Key,
		"top":              &k.Top.Key,
		"bottom":           &k.Bottom.Key,
		"half_up":          &k.HalfUp.Key,
		"half_down":        &k.HalfDown.Key,
		"left":             &k.Left.Key,
		"right":            &k.Right.Key,
		"select":           &k.Select.Key,
		"back":             &k.Back.Key,
		"quit":             &k.Quit.Key,
		"help":             &k.Help.Key,
		"refresh":          &k.Refresh.Key,
		"add_task":         &k.AddTask.Key,
		"add_task_full":    &k.AddTaskFull.Key,
		"edit_task":        &k.EditTask.Key,
		"delete_task":      &k.DeleteTask.Key,
		"complete":         &k.CompleteTask.Key,
		"priority1":        &k.Priority1.Key,
		"priority2":        &k.Priority2.Key,
		"priority3":        &k.Priority3.Key,
		"priority4":        &k.Priority4.Key,
		"move_prev_day":    &k.MoveTaskPrevDay.Key,
		"move_next_day":    &k.MoveTaskNextDay.Key,
		"add_comment":      &k.AddComment.Key,
		"reschedule":       &k.RescheduleTask.Key,
		"indent":           &k.IndentTask.Key,
		"outdent":          &k.OutdentTask.Key,
		"switch_pane":      &k.SwitchPane.Key,
		"search":           &k.Search.Key,
		"new_project":      &k.NewProject.Key,
		"new_section":      &k.NewSection.Key,
		"move_section":     &k.MoveSection.Key,
		"move_to_project":  &k.MoveToProject.Key,
		"reminder":         &k.Reminder.Key,
		"send_to_pomodoro": &k.SendToPomodoro.Key,
	}

	// Build a reverse map of key → action from the current (default) bindings
	// so we can detect conflicts when applying overrides.
	keyOwner := make(map[string]string, len(actions))
	for action, ptr := range actions {
		keyOwner[*ptr] = action
	}

	var warnings []string
	for action, key := range overrides {
		ptr, ok := actions[action]
		if !ok {
			continue
		}

		// Check for conflict: is this key already claimed by another action?
		if existing, conflict := keyOwner[key]; conflict && existing != action {
			warnings = append(warnings, fmt.Sprintf(
				"keybinding conflict: %q and %q are both bound to key %q; skipping %q",
				existing, action, key, action,
			))
			continue
		}

		// Remove old key mapping, assign new one.
		delete(keyOwner, *ptr)
		*ptr = key
		keyOwner[key] = action
	}

	return warnings
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
	case keymap.AddTaskFull.Key:
		return "add_full", true
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
	case "u":
		return "undo", true
	case "m":
		return "move_task", true
	case "M":
		return "move_section", true
	case keymap.AddComment.Key:
		return "add_comment", true
	case keymap.RescheduleTask.Key:
		return "reschedule", true
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
	case keymap.IndentTask.Key:
		return "indent", true
	case keymap.OutdentTask.Key:
		return "outdent", true
	case keymap.MoveTaskPrevDay.Key:
		return "move_task_prev_day", true
	case keymap.MoveTaskNextDay.Key:
		return "move_task_next_day", true
	case keymap.SwitchPane.Key:
		return "switch_pane", true
	case keymap.Search.Key:
		return "search", true
	case keymap.MoveToProject.Key:
		return "move_to_project", true
	case keymap.SendToPomodoro.Key:
		return "send_to_pomodoro", true
	case keymap.NewProject.Key:
		return "new_project", true
	case "f":
		return "toggle_favorite", true

	// Tab navigation shortcuts (numbers 1-6 are handled in app.go, but we can document or keep placeholders?)
	// Actually, 1-6 keys are NOT in keymap struct explicitly, they are likely handled via "1", "2" etc cases.
	// But let's check HandleKey implementation.

	// But let's check HandleKey implementation.

	// L for next day (REMOVED: now using > and <)
	// case "L":
	// 	return "move_task_next_day", true
	// case "H":
	// 	return "move_task_prev_day", true

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
		{"Tab Navigation", ""},
		{"1", "Inbox"},
		{"2", "Today's tasks"},
		{"3", "Upcoming tasks"},
		{"4", "Labels list"},
		{"5", "Filters"},
		{"6", "Calendar view"},
		{"7", "Projects sidebar"},

		{"", ""},

		{"Navigation", ""},
		{k.Up.Key + "/" + k.Down.Key, "Move up/down"},
		{"gg/G", "Go to top/bottom"},
		{"Ctrl+u/d", "Half page up/down"},
		{k.SwitchPane.Key, "Switch pane (Sidebar/Main)"},
		{"h/l", "Pane navigation (Left/Right)"},
		{"", ""},

		{"Task Actions", ""},
		{k.Select.Key, "Open task details"},
		{k.AddTask.Key, "Add new task"},
		{k.AddTaskFull.Key, "Add new task (full)"},
		{k.EditTask.Key, "Edit task content"},
		{k.CompleteTask.Key, "Complete/uncomplete task"},
		{"dd", "Delete task"},
		{"yy", "Copy task Content (+Desc)"},
		{"Space", "Toggle selection"},
		{"1-4", "Set priority (4 is highest)"},
		{"</>", "Move task date -1/+1 day"},
		{"s", "Add subtask"},
		{k.IndentTask.Key + "/" + k.OutdentTask.Key, "Indent/Outdent task"},
		{"m", "Move task to section"},
		{k.MoveToProject.Key, "Move task to project"},
		{k.AddComment.Key, "Add/View comments"},
		{k.Reminder.Key, "Manage reminders"},
		{k.RescheduleTask.Key, "Smart Reschedule"},
		{"", ""},

		{"Label/Project Actions", ""},
		{"a", "Add new label (in Labels tab)"},
		{"n", "New project (in sidebar)"},
		{"f", "Toggle favorite project"},
		{"e", "Edit selected item"},
		{"d", "Delete selected item"},
		{"S", "Manage sections"},
		{"M", "Reorder sections"},
		{"", ""},

		{"Calendar View", ""},
		{"v", "Switch layout (Compact/Expanded)"},
		{"h/l", "Previous/next day"},
		{"j/k", "Previous/next week"},
		{"[/]", "Previous/next month"},
		{"", ""},

		{"General", ""},
		{"Shift+D", "Set current as default view"},
		{k.Refresh.Key, "Refresh data from Todoist"},
		{k.Search.Key, "Search tasks"},
		{k.Help.Key, "Toggle this help menu"},
		{k.Back.Key, "Go back / Cancel"},
		{"f1", "Toggle key hints bar"},
		{k.Quit.Key, "Quit the application"},
		{"t", "Smart Reschedule"},

		{"", ""},
		{"Pomodoro Timer", ""},
		{"9", "Switch to Pomodoro View"},
		{"Space", "Start/Pause timer"},
		{"r", "Reset timer"},
		{"m", "Toggle Countdown/Stopwatch"},
		{"Tab", "Cycle preset (25/5 ↔ 50/10)"},
		{"+/-", "Adjust work duration"},
		{"n", "Next Pomodoro phase"},
		{"x", "Complete associated task"},
		{"c", "Clear associated task"},
	}
}
