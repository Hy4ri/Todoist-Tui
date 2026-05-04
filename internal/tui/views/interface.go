package views

import tea "github.com/charmbracelet/bubbletea"

// ViewHandler defines the contract for a TUI view.
// Each view (Today, Inbox, Calendar, etc.) implements this interface.
type ViewHandler interface {
	// Name returns the view identifier.
	Name() string

	// HandleKey processes keyboard input for this view.
	// Returns the command to execute and whether the key was consumed.
	HandleKey(msg tea.KeyMsg) (cmd tea.Cmd, consumed bool)

	// HandleSelect processes Enter/selection for this view.
	HandleSelect() tea.Cmd

	// HandleBack processes Escape for this view.
	// Returns the command to execute and whether the view should be exited.
	HandleBack() (cmd tea.Cmd, shouldExit bool)

	// OnEnter is called when switching to this view.
	// Use this to load data or initialize state.
	OnEnter() tea.Cmd

	// OnExit is called when leaving this view.
	// Use this to clean up state.
	OnExit()

	// Render returns the view's content.
	Render(width, height int) string
}
