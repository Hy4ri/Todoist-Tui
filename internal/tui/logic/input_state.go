package logic

import "github.com/hy4ri/todoist-tui/internal/tui/state"

// isTextEntryActive returns true when the user is currently typing into a
// free-form text input or textarea. In these states the : key should be passed
// through to the focused widget instead of opening the command line.
//
// Non-text modal states (colour pickers, delete confirmations, reschedule
// dialogs, move/indent pickers without an active search field, etc.) are
// deliberately excluded so : still works globally there.
func (h *Handler) isTextEntryActive() bool {
	// Command line itself is text entry but is handled separately and
	// earlier in the routing chain.

	// Task form — only when a text field is focused.
	if h.CurrentView == state.ViewTaskForm && h.TaskForm != nil && h.TaskForm.IsTextEntryFocused() {
		return true
	}

	// Quick add.
	if h.CurrentView == state.ViewQuickAdd && h.QuickAddForm != nil {
		return true
	}

	// Search view.
	if h.CurrentView == state.ViewSearch {
		return true
	}

	// Filter sidebar search.
	if h.IsFilterSearch {
		return true
	}

	// Filter creation/editing only blocks while the text steps are active.
	if (h.IsCreatingFilter || h.IsEditingFilter) && h.FilterFormStep < 2 {
		return true
	}

	// Project creation/editing only blocks while the name field is active.
	if (h.IsCreatingProject || h.IsEditingProject) && !h.IsSelectingColor {
		return true
	}

	// Label creation/editing only blocks while the name field is active.
	if (h.IsCreatingLabel || h.IsEditingLabel) && !h.IsSelectingColor {
		return true
	}

	// Section creation/editing.
	if h.IsCreatingSection || h.IsEditingSection {
		return true
	}

	// Subtask creation.
	if h.IsCreatingSubtask {
		return true
	}

	// Comment editing or adding.
	if h.IsEditingComment || h.IsAddingComment {
		return true
	}

	// Reminder creation/editing.
	if h.IsAddingReminder || h.IsEditingReminder {
		return true
	}

	// Section-aware task addition.
	if h.IsAddingToSection {
		return true
	}

	// Move-to-project search input.
	if h.IsMovingToProject {
		return true
	}

	// Indent task picker search input.
	if h.IsIndentingTask {
		return true
	}

	return false
}
