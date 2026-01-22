// Package components provides reusable UI components for the Todoist TUI.
package components

import tea "github.com/charmbracelet/bubbletea"

// Component is a sub-model that handles a specific part of the UI.
// Each component manages its own state, handles relevant messages,
// and renders its own view.
type Component interface {
	// Init initializes the component and returns any initial command.
	Init() tea.Cmd

	// Update handles messages and returns an updated component and command.
	Update(msg tea.Msg) (Component, tea.Cmd)

	// View renders the component to a string.
	View() string

	// SetSize updates the component's dimensions.
	SetSize(width, height int)
}

// Focusable is an optional interface for components that can receive focus.
type Focusable interface {
	Component
	// Focus sets the component as focused.
	Focus()
	// Blur removes focus from the component.
	Blur()
	// Focused returns whether the component is focused.
	Focused() bool
}

// DataReceiver is an optional interface for components that receive external data.
type DataReceiver[T any] interface {
	// SetData updates the component's data source.
	SetData(data T)
}
