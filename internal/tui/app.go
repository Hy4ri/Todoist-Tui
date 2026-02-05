package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/config"
	"github.com/hy4ri/todoist-tui/internal/tui/components"
	"github.com/hy4ri/todoist-tui/internal/tui/logic"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
	"github.com/hy4ri/todoist-tui/internal/tui/ui"
	"github.com/hy4ri/todoist-tui/internal/tui/views"
)

// App is the main Bubble Tea model for the application.
type App struct {
	*state.State
	handler     *logic.Handler
	renderer    *ui.Renderer
	coordinator *views.Coordinator
}

// NewApp creates a new App instance.
func NewApp(client *api.Client, cfg *config.Config, initialView string) *App {
	s := &state.State{
		Client: client,
		Config: cfg,

		SearchResults:   []api.Task{},
		SelectedTaskIDs: make(map[string]bool),
		NotifiedTasks:   make(map[string]bool),
	}

	// Initialize UI components
	s.SidebarComp = components.NewSidebar()
	s.FilterSidebarComp = components.NewSidebar()
	s.FilterSidebarComp.Title = "Filters"
	s.FilterSidebarComp.Hint = "/: search  Tab: switch pane"
	s.DetailComp = components.NewDetail()
	s.HelpComp = components.NewHelp()

	km := state.DefaultKeymap()
	s.Keymap = km
	s.HelpComp.SetKeymap(km.HelpItems())
	s.KeyState = &state.KeyState{}

	// Initialize other components
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = styles.Spinner
	s.Spinner = spin

	searchInput := textinput.New()
	searchInput.Placeholder = "Search tasks..."
	searchInput.CharLimit = 100
	searchInput.Width = 40
	s.SearchInput = searchInput

	// Set initial view
	// Set initial view based on config or argument
	s.CurrentView = state.ViewInbox
	s.CurrentTab = state.TabInbox
	s.FocusedPane = state.PaneMain // Ensure main pane is focused by default

	// Helper to set view/tab
	setView := func(v string) {
		switch v {
		case "projects":
			s.CurrentView = state.ViewProject
			s.CurrentTab = state.TabProjects
		case "upcoming":
			s.CurrentView = state.ViewUpcoming
			s.CurrentTab = state.TabUpcoming
		case "labels":
			s.CurrentView = state.ViewLabels
			s.CurrentTab = state.TabLabels
		case "calendar":
			s.CurrentView = state.ViewCalendar
			s.CurrentTab = state.TabCalendar
		case "today":
			s.CurrentView = state.ViewToday
			s.CurrentTab = state.TabToday
		case "inbox":
			s.CurrentView = state.ViewInbox
			s.CurrentTab = state.TabInbox
		case "filters":
			s.CurrentView = state.ViewFilters
			s.CurrentTab = state.TabFilters
			s.FocusedPane = state.PaneSidebar
		}
	}

	// Default from config first
	if cfg.UI.DefaultView != "" {
		setView(cfg.UI.DefaultView)
	}

	// CLI argument overrides config
	if initialView != "" {
		setView(initialView)
	}

	// Initialize calendar view mode from config
	if cfg.UI.CalendarDefaultView == "expanded" {
		s.CalendarViewMode = state.CalendarViewExpanded
	} else {
		s.CalendarViewMode = state.CalendarViewCompact
	}

	app := &App{
		State: s,
	}
	app.handler = logic.NewHandler(s)
	app.renderer = ui.NewRenderer(s)
	app.coordinator = views.NewCoordinator(s)

	return app
}

func (a *App) Init() tea.Cmd {
	return a.handler.Init()
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Delegate to handler
	// The handler returns (tea.Model, tea.Cmd) but the model is the handler itself.
	// We discard the returned model and return 'a'.
	cmd := a.handler.Update(msg)
	return a, cmd
}

func (a *App) View() string {
	return a.renderer.View()
}
