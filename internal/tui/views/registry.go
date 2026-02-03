package views

import (
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// TabInfo holds metadata for a tab.
type TabInfo struct {
	Tab       state.Tab
	Icon      string
	Name      string
	ShortName string
	ViewName  string // Maps to ViewHandler.Name()
}

// Registry holds all registered views and tabs.
type Registry struct {
	views map[string]ViewHandler
	tabs  []TabInfo
}

// NewRegistry creates a new empty registry.
func NewRegistry() *Registry {
	return &Registry{
		views: make(map[string]ViewHandler),
		tabs:  []TabInfo{},
	}
}

// RegisterView adds a view to the registry.
func (r *Registry) RegisterView(view ViewHandler) {
	r.views[view.Name()] = view
}

// RegisterTab adds a tab with its associated view.
func (r *Registry) RegisterTab(tab state.Tab, icon, name, shortName, viewName string) {
	r.tabs = append(r.tabs, TabInfo{
		Tab:       tab,
		Icon:      icon,
		Name:      name,
		ShortName: shortName,
		ViewName:  viewName,
	})
}

// GetView returns a view by name.
func (r *Registry) GetView(name string) (ViewHandler, bool) {
	view, ok := r.views[name]
	return view, ok
}

// GetViewForTab returns the view associated with a tab.
func (r *Registry) GetViewForTab(tab state.Tab) (ViewHandler, bool) {
	for _, t := range r.tabs {
		if t.Tab == tab {
			return r.GetView(t.ViewName)
		}
	}
	return nil, false
}

// GetTabs returns all registered tabs.
func (r *Registry) GetTabs() []TabInfo {
	return r.tabs
}

// DefaultRegistry creates a registry with all standard views and tabs.
func DefaultRegistry(s *state.State) *Registry {
	r := NewRegistry()

	// Register all views
	r.RegisterView(NewTodayView(s))
	r.RegisterView(NewInboxView(s))
	r.RegisterView(NewUpcomingView(s))
	r.RegisterView(NewLabelsView(s))
	r.RegisterView(NewCalendarView(s))
	r.RegisterView(NewProjectsView(s))

	// Register tabs (maps tab constants to view names)
	r.RegisterTab(state.TabInbox, "📥", "Inbox", "Inb", "inbox")
	r.RegisterTab(state.TabToday, "📅", "Today", "Tdy", "today")
	r.RegisterTab(state.TabUpcoming, "📆", "Upcoming", "Up", "upcoming")
	r.RegisterTab(state.TabLabels, "🏷️", "Labels", "Lbl", "labels")
	r.RegisterTab(state.TabCalendar, "🗓️", "Calendar", "Cal", "calendar")
	r.RegisterTab(state.TabProjects, "📂", "Projects", "Prj", "projects")

	return r
}
