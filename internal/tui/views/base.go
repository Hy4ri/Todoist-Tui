package views

import (
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// BaseView provides common functionality for all views.
// Views embed this struct to get shared helpers.
type BaseView struct {
	State  *state.State
	Client *api.Client
}

// NewBaseView creates a new BaseView with the given state.
func NewBaseView(s *state.State) *BaseView {
	return &BaseView{
		State:  s,
		Client: s.Client,
	}
}

// --- Common Helpers ---

// GetSelectedTaskIndex returns the actual task index from the cursor position.
// Returns -1 if no valid task is selected.
func (b *BaseView) GetSelectedTaskIndex() int {
	if len(b.State.Tasks) == 0 {
		return -1
	}

	taskIndex := b.State.TaskCursor

	// Use ordered indices if available (for views with sections/groups)
	if len(b.State.TaskOrderedIndices) > 0 && b.State.TaskCursor < len(b.State.TaskOrderedIndices) {
		taskIndex = b.State.TaskOrderedIndices[b.State.TaskCursor]
	}

	// Skip section headers (negative indices <= -100)
	if taskIndex <= -100 {
		return -1
	}

	if taskIndex < 0 || taskIndex >= len(b.State.Tasks) {
		return -1
	}

	return taskIndex
}

// GetSelectedTask returns the currently selected task, or nil if none.
func (b *BaseView) GetSelectedTask() *api.Task {
	idx := b.GetSelectedTaskIndex()
	if idx < 0 {
		return nil
	}
	return &b.State.Tasks[idx]
}

// MoveCursor moves the cursor by delta, respecting bounds.
func (b *BaseView) MoveCursor(delta int) {
	maxItems := len(b.State.Tasks)
	if len(b.State.TaskOrderedIndices) > 0 {
		maxItems = len(b.State.TaskOrderedIndices)
	}

	b.State.TaskCursor += delta
	if b.State.TaskCursor < 0 {
		b.State.TaskCursor = 0
	}
	if maxItems > 0 && b.State.TaskCursor >= maxItems {
		b.State.TaskCursor = maxItems - 1
	}
}

// SetStatus sets a status message.
func (b *BaseView) SetStatus(msg string) {
	b.State.StatusMsg = msg
}

// SetLoading sets the loading state.
func (b *BaseView) SetLoading(loading bool) {
	b.State.Loading = loading
}

// IsMainPaneFocused returns true if the main pane is focused.
func (b *BaseView) IsMainPaneFocused() bool {
	return b.State.FocusedPane == state.PaneMain
}
