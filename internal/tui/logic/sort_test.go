package logic

import (
	"testing"

	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

func TestSortTasks_NewRules(t *testing.T) {
	// Priority constants for clarity
	const (
		P1 = 4
		P2 = 3
		P3 = 2
		P4 = 1
	)

	time1 := "2026-02-12T10:00:00Z"
	time2 := "2026-02-12T11:00:00Z"
	date1 := "2026-02-12"
	date2 := "2026-02-13"

	tasks := []api.Task{
		{ID: "priority_only", Content: "P1 no time", Priority: P1, Due: &api.Due{Date: date1}},
		{ID: "time_late", Content: "11am P4", Priority: P4, Due: &api.Due{Date: date1, Datetime: &time2}},
		{ID: "time_early", Content: "10am P4", Priority: P4, Due: &api.Due{Date: date2, Datetime: &time1}}, // Datetime > Date
		{ID: "priority_low", Content: "P4 no time", Priority: P4, Due: &api.Due{Date: date1}},
		{ID: "no_due", Content: "No due", Priority: P1},
	}

	h := &Handler{
		State: &state.State{
			Tasks: tasks,
		},
	}

	h.sortTasks()

	// Expected order:
	// 1. time_early (10am) - time is top priority
	// 2. time_late (11am) - time is top priority
	// 3. priority_only (P1) - among untimed tasks, higher priority first
	// 4. priority_low (P4) - untimed, lower priority
	// 5. no_due - no due date comes last (due to hasDue check)

	expectedOrder := []string{
		"time_early",
		"time_late",
		"priority_only",
		"priority_low",
		"no_due",
	}

	if len(h.Tasks) != len(expectedOrder) {
		t.Fatalf("Expected %d tasks, got %d", len(expectedOrder), len(h.Tasks))
	}

	for i, id := range expectedOrder {
		if h.Tasks[i].ID != id {
			t.Errorf("At index %d: expected %s, got %s", i, id, h.Tasks[i].ID)
		}
	}
}

func TestSortTasksHierarchically_NewRules(t *testing.T) {
	const (
		P1 = 4
		P4 = 1
	)
	time1 := "2026-02-12T10:00:00Z"

	tasks := []api.Task{
		{ID: "parent", Content: "Parent", ChildOrder: 1},
		{ID: "child_priority", Content: "Child P1", ParentID: stringPtr("parent"), Priority: P1, ChildOrder: 2},
		{ID: "child_time", Content: "Child Time", ParentID: stringPtr("parent"), Priority: P4, ChildOrder: 1, Due: &api.Due{Datetime: &time1}},
	}

	h := &Handler{
		State: &state.State{
			Tasks:       tasks,
			CurrentView: state.ViewProject,
		},
	}

	h.sortTasks()

	// Expected order:
	// 1. parent
	// 2. child_time (Time first)
	// 3. child_priority (Priority second)

	expectedOrder := []string{
		"parent",
		"child_time",
		"child_priority",
	}

	if len(h.Tasks) != len(expectedOrder) {
		t.Fatalf("Expected %d tasks, got %d", len(expectedOrder), len(h.Tasks))
	}

	for i, id := range expectedOrder {
		if h.Tasks[i].ID != id {
			t.Errorf("At index %d: expected %s, got %s", i, id, h.Tasks[i].ID)
		}
	}
}

func stringPtr(s string) *string {
	return &s
}
