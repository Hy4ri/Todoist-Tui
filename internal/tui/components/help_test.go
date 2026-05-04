package components

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestHelpViewEmptyAndPopulated(t *testing.T) {
	h := NewHelp()
	if got := h.View(); !strings.Contains(got, "No keybindings registered") {
		t.Fatalf("empty help view = %q", got)
	}

	h.SetSize(120, 40)
	h.SetKeymap([][]string{{"Tab Navigation", ""}, {"1", "Inbox"}, {"gg/G", "Go to top/bottom"}, {"", ""}, {"General", ""}, {"q", "Quit the application"}})
	got := h.View()
	checks := []string{"Keyboard Shortcuts", "Tab Navigation", "1", "Press ESC or ? to close"}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Fatalf("help view missing %q", want)
		}
	}
}

func TestHelpUpdateReturnsViewChangeRequest(t *testing.T) {
	h := NewHelp()
	_, cmd := h.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("esc")})
	if cmd == nil {
		t.Fatal("Update() returned nil cmd for esc")
	}
	msg := cmd()
	change, ok := msg.(ViewChangeRequestMsg)
	if !ok || change.View != ViewToday {
		t.Fatalf("Update() msg = %#v, want ViewChangeRequestMsg{View: ViewToday}", msg)
	}
}
