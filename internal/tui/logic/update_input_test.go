package logic

import (
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/tui/components"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

func TestHandleKeyMsg_ColonOpensCommandLine(t *testing.T) {
	// Idle navigation view — : should open command line.
	h := NewHandler(&state.State{
		CurrentView: state.ViewInbox,
		SidebarComp: components.NewSidebar(),
	})

	cmd := h.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})

	if h.CommandLine == nil || !h.CommandLine.Active {
		t.Fatal("expected command line to be active after : in idle view")
	}
	if cmd == nil {
		t.Fatal("expected a tea.Cmd from activateCommandLine")
	}
}

func TestHandleKeyMsg_ColonBlockedInTextEntryStates(t *testing.T) {
	cases := []struct {
		name string
		prep func(*Handler)
	}{
		{
			name: "task form content field",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewTaskForm
				h.TaskForm = state.NewTaskForm(nil, nil)
				h.TaskForm.Focus(state.FormFieldContent)
			},
		},
		{
			name: "task form description field",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewTaskForm
				h.TaskForm = state.NewTaskForm(nil, nil)
				h.TaskForm.Focus(state.FormFieldDescription)
			},
		},
		{
			name: "task form due field",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewTaskForm
				h.TaskForm = state.NewTaskForm(nil, nil)
				h.TaskForm.Focus(state.FormFieldDue)
			},
		},
		{
			name: "task form due time field",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewTaskForm
				h.TaskForm = state.NewTaskForm(nil, nil)
				h.TaskForm.Focus(state.FormFieldDueTime)
			},
		},
		{
			name: "quick add",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewQuickAdd
				h.QuickAddForm = state.NewQuickAddForm()
			},
		},
		{
			name: "search",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewSearch
			},
		},
		{
			name: "comment editing",
			prep: func(h *Handler) {
				h.IsEditingComment = true
			},
		},
		{
			name: "comment adding",
			prep: func(h *Handler) {
				h.IsAddingComment = true
			},
		},
		{
			name: "project creation",
			prep: func(h *Handler) {
				h.IsCreatingProject = true
			},
		},
		{
			name: "label creation",
			prep: func(h *Handler) {
				h.IsCreatingLabel = true
			},
		},
		{
			name: "section creation",
			prep: func(h *Handler) {
				h.IsCreatingSection = true
			},
		},
		{
			name: "subtask creation",
			prep: func(h *Handler) {
				h.IsCreatingSubtask = true
			},
		},
		{
			name: "filter search",
			prep: func(h *Handler) {
				h.IsFilterSearch = true
			},
		},
		{
			name: "filter creation",
			prep: func(h *Handler) {
				h.IsCreatingFilter = true
				h.FilterFormStep = 0
			},
		},
		{
			name: "filter query step",
			prep: func(h *Handler) {
				h.IsCreatingFilter = true
				h.FilterFormStep = 1
			},
		},
		{
			name: "reminder creation",
			prep: func(h *Handler) {
				h.IsAddingReminder = true
			},
		},
		{
			name: "section add",
			prep: func(h *Handler) {
				h.IsAddingToSection = true
			},
		},
		{
			name: "move to project",
			prep: func(h *Handler) {
				h.IsMovingToProject = true
			},
		},
		{
			name: "indent task",
			prep: func(h *Handler) {
				h.IsIndentingTask = true
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(&state.State{
				CurrentView: state.ViewInbox,
				SidebarComp: components.NewSidebar(),
			})
			tc.prep(h)

			_ = h.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})

			if h.CommandLine != nil && h.CommandLine.Active {
				t.Fatalf(": should NOT open command line in %s", tc.name)
			}
		})
	}
}

func TestHandleKeyMsg_ColonInsertedInTextInputs(t *testing.T) {
	cases := []struct {
		name     string
		prep     func(*Handler)
		expectFn func(*Handler) string
	}{
		{
			name: "task form content",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewTaskForm
				h.TaskForm = state.NewTaskForm(nil, nil)
				h.TaskForm.Focus(state.FormFieldContent)
			},
			expectFn: func(h *Handler) string { return h.TaskForm.Content.Value() },
		},
		{
			name: "quick add input",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewQuickAdd
				h.QuickAddForm = state.NewQuickAddForm()
			},
			expectFn: func(h *Handler) string { return h.QuickAddForm.Input.Value() },
		},
		{
			name: "search input",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewSearch
				h.SearchInput = textinput.New()
				h.SearchInput.Focus()
			},
			expectFn: func(h *Handler) string { return h.SearchInput.Value() },
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(&state.State{SidebarComp: components.NewSidebar()})
			tc.prep(h)

			_ = h.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})

			if got := tc.expectFn(h); got != ":" {
				t.Fatalf("expected colon to be inserted in %s, got %q", tc.name, got)
			}
		})
	}
}

func TestHandleKeyMsg_ColonAllowedInNonTextStates(t *testing.T) {
	cases := []struct {
		name string
		prep func(*Handler)
	}{
		{
			name: "task form priority field",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewTaskForm
				h.TaskForm = state.NewTaskForm(nil, nil)
				h.TaskForm.Focus(state.FormFieldPriority)
			},
		},
		{
			name: "task form project field",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewTaskForm
				h.TaskForm = state.NewTaskForm(nil, nil)
				h.TaskForm.Focus(state.FormFieldShowProject)
			},
		},
		{
			name: "task form labels field",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewTaskForm
				h.TaskForm = state.NewTaskForm(nil, nil)
				h.TaskForm.Focus(state.FormFieldLabels)
			},
		},
		{
			name: "task form submit button",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewTaskForm
				h.TaskForm = state.NewTaskForm(nil, nil)
				h.TaskForm.Focus(state.FormFieldSubmit)
			},
		},
		{
			name: "filter color step",
			prep: func(h *Handler) {
				h.IsCreatingFilter = true
				h.FilterFormStep = 2
			},
		},
		{
			name: "delete confirmation",
			prep: func(h *Handler) {
				h.ConfirmDeleteProject = true
			},
		},
		{
			name: "reschedule dialog",
			prep: func(h *Handler) {
				h.IsRescheduling = true
				h.RescheduleOptions = []string{"Today", "Tomorrow"}
			},
		},
		{
			name: "color selection",
			prep: func(h *Handler) {
				h.IsSelectingColor = true
			},
		},
		{
			name: "help view",
			prep: func(h *Handler) {
				h.CurrentView = state.ViewHelp
			},
		},
		{
			name: "project color step",
			prep: func(h *Handler) {
				h.IsCreatingProject = true
				h.IsSelectingColor = true
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHandler(&state.State{
				CurrentView: state.ViewInbox,
				SidebarComp: components.NewSidebar(),
			})
			tc.prep(h)

			_ = h.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})

			if h.CommandLine == nil || !h.CommandLine.Active {
				t.Fatalf(": SHOULD open command line in %s", tc.name)
			}
		})
	}
}

func TestHandleKeyMsg_CommandLineAlreadyActive(t *testing.T) {
	// If the command line is already active, : should be passed to it.
	h := NewHandler(&state.State{
		CurrentView: state.ViewInbox,
		SidebarComp: components.NewSidebar(),
	})
	_ = h.activateCommandLine()

	if h.CommandLine == nil || !h.CommandLine.Active {
		t.Fatal("setup failed: command line should be active")
	}

	prevVal := h.CommandLine.Input.Value()
	_ = h.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})

	if h.CommandLine.Input.Value() != prevVal+":" {
		t.Fatalf("expected ':' to be appended to command line input, got %q", h.CommandLine.Input.Value())
	}
}

func TestHandleKeyMsg_ColonBlockedWhenTaskFormNil(t *testing.T) {
	// ViewTaskForm with nil TaskForm is not a text-entry state.
	h := NewHandler(&state.State{
		CurrentView: state.ViewTaskForm,
		TaskForm:    nil,
		SidebarComp: components.NewSidebar(),
	})

	_ = h.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}})

	if h.CommandLine == nil || !h.CommandLine.Active {
		t.Fatal("expected command line to open when TaskForm is nil")
	}
}
