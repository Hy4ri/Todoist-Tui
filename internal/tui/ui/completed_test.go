package ui

import (
	"strings"
	"testing"

	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

func TestRenderCompletedTaskListGrouping(t *testing.T) {
	today := "2024-02-09T10:00:00Z"
	yesterday := "2024-02-08T10:00:00Z"

	r := &Renderer{
		State: &state.State{
			Tasks: []api.Task{
				{ID: "1", Content: "Task Today", CompletedAt: &today},
				{ID: "2", Content: "Task Yesterday", CompletedAt: &yesterday},
			},
			Loading: false,
			Width:   80,
			Height:  24,
		},
	}
	r.Width = 80
	r.Height = 24

	// Mock required components
	// (If r.renderCompletedTaskList depends on styles/etc, it should be fine as it's same package)

	output := r.renderCompletedTaskList(80, 20)

	// Verify grouping headers exist
	if !strings.Contains(output, "Today") {
		t.Error("output should contain 'Today' group header")
	}
	if !strings.Contains(output, "Yesterday") {
		t.Error("output should contain 'Yesterday' group header")
	}
	if !strings.Contains(output, "Task Today") {
		t.Error("output should contain 'Task Today'")
	}
}

func TestRenderCompletedHeaderSelection(t *testing.T) {
	r := &Renderer{
		State: &state.State{
			TaskCursor:  0,
			FocusedPane: state.PaneMain,
		},
	}

	// Header at position 0 should have cursor indicator
	orderedIndices := []int{-100, 0}
	output := r.renderCompletedHeader("Today", -100, orderedIndices)

	if !strings.Contains(output, ">") {
		t.Error("header at cursor position should have '>' indicator")
	}

	// Move cursor
	r.TaskCursor = 1
	output = r.renderCompletedHeader("Today", -100, orderedIndices)
	if strings.Contains(output, ">") {
		t.Error("header NOT at cursor position should NOT have '>' indicator")
	}
}
