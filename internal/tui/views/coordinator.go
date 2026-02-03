package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// Coordinator manages view lifecycle and delegates to the active view.
type Coordinator struct {
	registry    *Registry
	state       *state.State
	currentView ViewHandler
}

// NewCoordinator creates a new view coordinator.
func NewCoordinator(s *state.State) *Coordinator {
	reg := DefaultRegistry(s)
	c := &Coordinator{
		registry: reg,
		state:    s,
	}

	// Set initial view based on current tab
	if view, ok := reg.GetViewForTab(s.CurrentTab); ok {
		c.currentView = view
	}

	return c
}

// GetRegistry returns the registry.
func (c *Coordinator) GetRegistry() *Registry {
	return c.registry
}

// GetCurrentView returns the active view.
func (c *Coordinator) GetCurrentView() ViewHandler {
	return c.currentView
}

// SwitchToTab switches to the view for the given tab.
func (c *Coordinator) SwitchToTab(tab state.Tab) tea.Cmd {
	// Call OnExit for current view
	if c.currentView != nil {
		c.currentView.OnExit()
	}

	// Get new view
	view, ok := c.registry.GetViewForTab(tab)
	if !ok {
		return nil
	}

	c.currentView = view
	c.state.CurrentTab = tab

	// Map tab to view constant
	switch tab {
	case state.TabInbox:
		c.state.CurrentView = state.ViewInbox
	case state.TabToday:
		c.state.CurrentView = state.ViewToday
	case state.TabUpcoming:
		c.state.CurrentView = state.ViewUpcoming
	case state.TabLabels:
		c.state.CurrentView = state.ViewLabels
	case state.TabCalendar:
		c.state.CurrentView = state.ViewCalendar
	case state.TabProjects:
		c.state.CurrentView = state.ViewProject
	}

	return c.currentView.OnEnter()
}

// HandleKey delegates key handling to the current view.
// Returns the command and whether the key was consumed.
func (c *Coordinator) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	if c.currentView == nil {
		return nil, false
	}
	return c.currentView.HandleKey(msg)
}

// HandleSelect delegates selection to the current view.
func (c *Coordinator) HandleSelect() tea.Cmd {
	if c.currentView == nil {
		return nil
	}
	return c.currentView.HandleSelect()
}

// HandleBack delegates back/escape to the current view.
func (c *Coordinator) HandleBack() (tea.Cmd, bool) {
	if c.currentView == nil {
		return nil, false
	}
	return c.currentView.HandleBack()
}

// GetTabs returns all registered tabs.
func (c *Coordinator) GetTabs() []TabInfo {
	return c.registry.GetTabs()
}
