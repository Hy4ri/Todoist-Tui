package state

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestApplyOverrides(t *testing.T) {
	km := DefaultKeymap()
	warnings := km.ApplyOverrides(map[string]string{
		"add_task": "A",
	})

	if km.AddTask.Key != "a" {
		t.Fatalf("AddTask.Key = %q, want %q", km.AddTask.Key, "a")
	}
	if km.AddTaskFull.Key != "A" {
		t.Fatalf("AddTaskFull.Key = %q, want %q", km.AddTaskFull.Key, "A")
	}
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want one conflict warning", warnings)
	}
}

func TestApplyOverridesIgnoresEmptyAndUnknownMaps(t *testing.T) {
	km := DefaultKeymap()
	if warnings := km.ApplyOverrides(nil); warnings != nil {
		t.Fatalf("ApplyOverrides(nil) = %v, want nil", warnings)
	}
	if warnings := km.ApplyOverrides(map[string]string{}); warnings != nil {
		t.Fatalf("ApplyOverrides(empty) = %v, want nil", warnings)
	}
}

func TestKeyStateHandleKey(t *testing.T) {
	km := DefaultKeymap()
	ks := &KeyState{}

	tests := []struct {
		name string
		msg  string
		want string
	}{
		{"single up", "k", "up"},
		{"gg sequence start", "g", ""},
		{"gg sequence complete", "g", "top"},
		{"dd sequence", "d", ""},
		{"dd complete", "d", "delete"},
		{"yy sequence", "y", ""},
		{"yy complete", "y", "copy"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, consumed := ks.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(tc.msg)}, km)
			if tc.want == "" {
				if !consumed {
					t.Fatal("expected key to be consumed")
				}
				return
			}
			if !consumed || got != tc.want {
				t.Fatalf("HandleKey() = (%q, %v), want (%q, true)", got, consumed, tc.want)
			}
		})
	}
}

func TestKeyStateFallbackAndReset(t *testing.T) {
	ks := &KeyState{}
	if got, consumed := ks.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}, struct{}{}); got != "help" || !consumed {
		t.Fatalf("fallback help = (%q, %v), want (help, true)", got, consumed)
	}

	ks.WaitingG, ks.WaitingD, ks.WaitingY = true, true, true
	ks.Reset()
	if ks.WaitingG || ks.WaitingD || ks.WaitingY {
		t.Fatal("Reset() did not clear key state")
	}
}

func TestHelpItems(t *testing.T) {
	items := DefaultKeymap().HelpItems()
	if len(items) == 0 {
		t.Fatal("HelpItems() returned no items")
	}
	if items[0][0] != "Tab Navigation" {
		t.Fatalf("first section = %q, want %q", items[0][0], "Tab Navigation")
	}
	found := false
	for _, item := range items {
		if len(item) >= 2 && item[0] == "gg/G" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("HelpItems() missing gg/G entry")
	}
}
