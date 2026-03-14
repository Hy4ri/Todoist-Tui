package logic

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// activateCommandLine initializes and shows the command line.
func (h *Handler) activateCommandLine() tea.Cmd {
	h.CommandLine = state.NewCommandLine()
	h.CommandLine.Active = true
	h.CommandLine.Input.Focus()
	return textinput.Blink
}

// handleCommandLineKeyMsg handles input when the command line is active.
func (h *Handler) handleCommandLineKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		return h.executeCommand(h.CommandLine.Input.Value())
	case "esc":
		h.CommandLine.Active = false
		return nil
	case "tab":
		return h.autocompleteCommand()
	case "up":
		return h.commandHistoryPrev()
	case "down":
		return h.commandHistoryNext()
	}

	// Forward to input
	var cmd tea.Cmd
	h.CommandLine.Input, cmd = h.CommandLine.Input.Update(msg)
	return tea.Batch(cmd, h.updateSuggestions())
}
