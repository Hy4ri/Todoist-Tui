package state

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
)

func TestTaskForm_PriorityMapping(t *testing.T) {
	// Setup
	projects := []api.Project{}
	labels := []api.Label{}
	f := NewTaskForm(projects, labels)

	// 1. Verify default priority (P4 = 1)
	if f.Priority != 1 {
		t.Errorf("Expected default priority to be 1 (P4), got %d", f.Priority)
	}

	// Focus priority field
	f.Focus(FormFieldPriority)

	// 2. Test input mapping
	// Input "1" should map to Priority 4 (P1)
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("1")})
	if f.Priority != 4 {
		t.Errorf("Input '1' should set priority to 4 (P1), got %d", f.Priority)
	}

	// Input "2" should map to Priority 3 (P2)
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("2")})
	if f.Priority != 3 {
		t.Errorf("Input '2' should set priority to 3 (P2), got %d", f.Priority)
	}

	// Input "3" should map to Priority 2 (P3)
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("3")})
	if f.Priority != 2 {
		t.Errorf("Input '3' should set priority to 2 (P3), got %d", f.Priority)
	}

	// Input "4" should map to Priority 1 (P4)
	f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("4")})
	if f.Priority != 1 {
		t.Errorf("Input '4' should set priority to 1 (P4), got %d", f.Priority)
	}
}

func TestTaskForm_ToCreateRequest_Priority(t *testing.T) {
	projects := []api.Project{}
	labels := []api.Label{}
	f := NewTaskForm(projects, labels)

	// Set priority to 4 (P1)
	f.Priority = 4

	req := f.ToCreateRequest()

	if req.Priority != 4 {
		t.Errorf("ToCreateRequest should preserve priority 4, got %d", req.Priority)
	}
}

func TestTaskForm_ToUpdateRequest_Priority(t *testing.T) {
	projects := []api.Project{}
	labels := []api.Label{}
	f := NewTaskForm(projects, labels)

	// Set priority to 4 (P1)
	f.Priority = 4

	req := f.ToUpdateRequest()

	if req.Priority == nil {
		t.Fatal("ToUpdateRequest priority should not be nil")
	}

	if *req.Priority != 4 {
		t.Errorf("ToUpdateRequest should preserve priority 4, got %d", *req.Priority)
	}
}
