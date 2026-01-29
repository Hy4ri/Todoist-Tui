package state

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
)

func TestTaskForm_LabelSelection(t *testing.T) {
	// Setup
	labels := []api.Label{
		{Name: "Label1", ID: "1"},
		{Name: "Label2", ID: "2"},
		{Name: "Label3", ID: "3"},
	}
	projects := []api.Project{}

	f := NewTaskForm(projects, labels)
	f.FocusIndex = FormFieldLabels // Force focus on labels

	// Helper to send key
	sendKey := func(key string) {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		switch key {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "space":
			msg = tea.KeyMsg{Type: tea.KeySpace}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "up":
			msg = tea.KeyMsg{Type: tea.KeyUp}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		}
		f.Update(msg)
	}

	// 1. Initial State
	if f.ShowLabelList {
		t.Error("Label list should be closed initially")
	}

	// 2. Open Dropdown
	sendKey("enter")
	if !f.ShowLabelList {
		t.Error("Label list should be open after Enter")
	}

	// 3. Navigate Down
	// Current cursor should be 0
	if f.LabelListCursor != 0 {
		t.Errorf("Expected cursor 0, got %d", f.LabelListCursor)
	}
	sendKey("j")
	if f.LabelListCursor != 1 {
		t.Errorf("Expected cursor 1 after 'j', got %d", f.LabelListCursor)
	}
	sendKey("down")
	if f.LabelListCursor != 2 {
		t.Errorf("Expected cursor 2 after 'down', got %d", f.LabelListCursor)
	}
	// Clamp max
	sendKey("j")
	if f.LabelListCursor != 2 {
		t.Errorf("Expected cursor 2 (max) after 'j', got %d", f.LabelListCursor)
	}

	// 4. Select Label 3 (Index 2)
	sendKey("enter")
	if len(f.Labels) != 1 || f.Labels[0] != "Label3" {
		t.Errorf("Expected Label3 selected, got %v", f.Labels)
	}

	// 5. Navigate Up and Select Label 2 (Index 1)
	sendKey("k")
	if f.LabelListCursor != 1 {
		t.Errorf("Expected cursor 1 after 'k', got %d", f.LabelListCursor)
	}
	sendKey("space")
	// Should have Label3 and Label2
	hasL2 := false
	hasL3 := false
	for _, l := range f.Labels {
		if l == "Label2" {
			hasL2 = true
		}
		if l == "Label3" {
			hasL3 = true
		}
	}
	if !hasL2 || !hasL3 || len(f.Labels) != 2 {
		t.Errorf("Expected Label2 and Label3, got %v", f.Labels)
	}

	// 6. Toggle Label 2 Off
	sendKey("enter")
	hasL2 = false
	for _, l := range f.Labels {
		if l == "Label2" {
			hasL2 = true
		}
	}
	if hasL2 {
		t.Errorf("Expected Label2 deselected, got %v", f.Labels)
	}

	// 7. Close Dropdown
	sendKey("esc")
	if f.ShowLabelList {
		t.Error("Label list should be closed after Esc")
	}
}
