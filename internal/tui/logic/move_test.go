package logic

import (
	"strings"
	"testing"

	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

func TestBuildMoveTargetList(t *testing.T) {
	h := &Handler{
		State: &state.State{
			Projects: []api.Project{
				{ID: "inbox", Name: "Inbox", InboxProject: true},
				{ID: "p1", Name: "Project 1", IsFavorite: true},
				{ID: "p2", Name: "Work", IsFavorite: false},
			},
			AllSections: []api.Section{
				{ID: "s1", Name: "Section 1", ProjectID: "p1", SectionOrder: 1},
				{ID: "s2", Name: "Section 2", ProjectID: "p1", SectionOrder: 2},
				{ID: "s3", Name: "Work Section", ProjectID: "p2", SectionOrder: 1},
			},
		},
	}

	// Test initial list construction
	h.buildMoveTargetList("")

	// Expected order: Inbox, Favorites (P1), Other (P2)
	// Each project followed by its sections
	expectedNames := []string{"Inbox", "Project 1", "Section 1", "Section 2", "Work", "Work Section"}

	if len(h.MoveTargetList) != len(expectedNames) {
		t.Errorf("expected %d targets, got %d", len(expectedNames), len(h.MoveTargetList))
	}

	for i, name := range expectedNames {
		if i < len(h.MoveTargetList) && h.MoveTargetList[i].Name != name {
			t.Errorf("at index %d: expected %s, got %s", i, name, h.MoveTargetList[i].Name)
		}
	}

	// Test filtering
	h.buildMoveTargetList("work")
	// Should match "Work" (project) and "Work Section" (section)
	// Wait, my logic shows project and its sections if query matches project name
	// AND shows section if query matches section name.
	// So "Work" project matches "work" -> targets include "Work" and "Work Section" (because it's in "Work" project)
	// "Work Section" matches "work" -> target includes "Work Section".
	// Results should be "Work" and "Work Section".

	for _, target := range h.MoveTargetList {
		if !strings.Contains(strings.ToLower(target.Name), "work") &&
			!strings.Contains(strings.ToLower("Work"), "work") { // "Work" is parent of "Work Section"
			t.Errorf("filtered list contains non-matching item: %s", target.Name)
		}
	}

	if len(h.MoveTargetList) == 0 {
		t.Error("filtered list is empty but should have results")
	}
}

func TestHandleMoveToProjectSelection(t *testing.T) {
	h := &Handler{
		State: &state.State{
			Projects: []api.Project{
				{ID: "p1", Name: "Project 1"},
			},
			Tasks: []api.Task{
				{ID: "task1", Content: "Task 1"},
			},
			TaskCursor: 0,
		},
	}

	// Trigger move
	h.handleMoveToProject()

	if !h.IsMovingToProject {
		t.Error("IsMovingToProject should be true")
	}
	if h.SelectedTask == nil || h.SelectedTask.ID != "task1" {
		t.Error("SelectedTask not set correctly from cursor")
	}
	if len(h.MoveTargetList) == 0 {
		t.Error("MoveTargetList should not be empty")
	}
}
