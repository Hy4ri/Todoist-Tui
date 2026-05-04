package views

import (
	"testing"

	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

func TestBaseViewTaskSelectionAndCursor(t *testing.T) {
	s := &state.State{Tasks: []api.Task{{ID: "1"}, {ID: "2"}, {ID: "3"}}, FocusedPane: state.PaneMain}
	b := NewBaseView(s)

	if got := b.GetSelectedTaskIndex(); got != 0 {
		t.Fatalf("GetSelectedTaskIndex() = %d, want 0", got)
	}

	s.TaskCursor = 1
	if got := b.GetSelectedTask(); got == nil || got.ID != "2" {
		t.Fatalf("GetSelectedTask() = %#v, want task 2", got)
	}

	s.TaskOrderedIndices = []int{2, -100, 0}
	s.TaskCursor = 0
	if got := b.GetSelectedTaskIndex(); got != 2 {
		t.Fatalf("ordered selection = %d, want 2", got)
	}
	s.TaskCursor = 1
	if got := b.GetSelectedTaskIndex(); got != -1 {
		t.Fatalf("section header selection = %d, want -1", got)
	}

	b.MoveCursor(-10)
	if s.TaskCursor != 0 {
		t.Fatalf("MoveCursor() lower bound = %d, want 0", s.TaskCursor)
	}
	b.MoveCursor(10)
	if s.TaskCursor != 2 {
		t.Fatalf("MoveCursor() upper bound = %d, want 2", s.TaskCursor)
	}

	b.SetStatus("ok")
	b.SetLoading(true)
	if s.StatusMsg != "ok" || !s.Loading || !b.IsMainPaneFocused() {
		t.Fatalf("state helpers did not update state correctly: %#v", s)
	}
}

func TestBaseViewNoTasks(t *testing.T) {
	b := NewBaseView(&state.State{})
	if got := b.GetSelectedTaskIndex(); got != -1 {
		t.Fatalf("GetSelectedTaskIndex() = %d, want -1", got)
	}
	if got := b.GetSelectedTask(); got != nil {
		t.Fatalf("GetSelectedTask() = %#v, want nil", got)
	}
}
