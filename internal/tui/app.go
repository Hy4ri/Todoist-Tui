package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/config"
	"github.com/hy4ri/todoist-tui/internal/tui/logic"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
	"github.com/hy4ri/todoist-tui/internal/tui/ui"
)

// App is the main Bubble Tea model for the application.
type App struct {
	*state.State
	handler  *logic.Handler
	renderer *ui.Renderer
}

// NewApp creates a new App instance.
func NewApp(client *api.Client, cfg *config.Config, initialView string) *App {
	s := &state.State{
		Client: client,
		Config: cfg,

		SearchResults:   []api.Task{},
		SelectedTaskIDs: make(map[string]bool),
	}

	// Initialize UI components
	// TODO: These were in app.go, need to ensure type compatibility
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
	s.CurrentView = state.ViewToday
	s.CurrentTab = state.TabToday
	if initialView == "projects" {
		s.CurrentView = state.ViewProject
		s.CurrentTab = state.TabProjects
	} else if initialView == "upcoming" {
		s.CurrentView = state.ViewUpcoming
		s.CurrentTab = state.TabUpcoming
	}

	app := &App{
		State: s,
	}
	app.handler = logic.NewHandler(s)
	app.renderer = ui.NewRenderer(s)

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
