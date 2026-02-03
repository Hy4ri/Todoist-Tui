package views

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/components"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// ProjectsView handles the Projects tab with sidebar navigation.
type ProjectsView struct {
	*BaseView
}

// NewProjectsView creates a new ProjectsView.
func NewProjectsView(s *state.State) *ProjectsView {
	return &ProjectsView{BaseView: NewBaseView(s)}
}

func (v *ProjectsView) Name() string { return "projects" }

func (v *ProjectsView) OnEnter() tea.Cmd {
	v.State.FocusedPane = state.PaneSidebar
	v.State.SidebarCursor = 0
	v.buildSidebarItems()
	return nil
}

func (v *ProjectsView) OnExit() {
	v.State.CurrentProject = nil
}

func (v *ProjectsView) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	// Handle pane switching
	switch msg.String() {
	case "h", "left":
		if v.State.FocusedPane == state.PaneMain {
			v.State.FocusedPane = state.PaneSidebar
			return nil, true
		}
	case "l", "right":
		if v.State.FocusedPane == state.PaneSidebar {
			v.State.FocusedPane = state.PaneMain
			return nil, true
		}
	case "tab":
		if v.State.FocusedPane == state.PaneSidebar {
			v.State.FocusedPane = state.PaneMain
		} else {
			v.State.FocusedPane = state.PaneSidebar
		}
		return nil, true
	}
	return nil, false
}

func (v *ProjectsView) HandleSelect() tea.Cmd {
	if v.State.FocusedPane == state.PaneSidebar {
		return v.selectSidebarItem()
	}
	return v.selectTask()
}

func (v *ProjectsView) HandleBack() (tea.Cmd, bool) {
	if v.State.ShowDetailPanel {
		v.State.ShowDetailPanel = false
		v.State.SelectedTask = nil
		v.State.Comments = nil
		return nil, false
	}
	if v.State.CurrentProject != nil {
		v.State.CurrentProject = nil
		v.State.Tasks = nil
		v.State.FocusedPane = state.PaneSidebar
		return nil, false
	}
	return nil, false
}

func (v *ProjectsView) Render(width, height int) string { return "" }

// --- Private helpers ---

func (v *ProjectsView) selectSidebarItem() tea.Cmd {
	if v.State.SidebarCursor >= len(v.State.SidebarItems) {
		return nil
	}

	item := v.State.SidebarItems[v.State.SidebarCursor]
	if item.Type == "separator" {
		return nil
	}

	v.State.FocusedPane = state.PaneMain
	v.State.TaskCursor = 0

	for i := range v.State.Projects {
		if v.State.Projects[i].ID == item.ID {
			v.State.CurrentProject = &v.State.Projects[i]
			v.State.Sections = nil

			if v.State.ShowDetailPanel {
				v.State.ShowDetailPanel = false
				v.State.SelectedTask = nil
				v.State.Comments = nil
			}

			return v.loadProjectTasks(v.State.CurrentProject.ID)
		}
	}
	return nil
}

func (v *ProjectsView) selectTask() tea.Cmd {
	task := v.GetSelectedTask()
	if task == nil {
		return nil
	}
	taskCopy := new(api.Task)
	*taskCopy = *task
	v.State.SelectedTask = taskCopy
	v.State.ShowDetailPanel = true
	return v.loadTaskComments()
}

func (v *ProjectsView) loadProjectTasks(projectID string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := v.Client.GetTasks(api.TaskFilter{ProjectID: projectID})
		if err != nil {
			return errMsg{err}
		}
		sections, err := v.Client.GetSections(projectID)
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{tasks: tasks, sections: sections}
	}
}

func (v *ProjectsView) loadTaskComments() tea.Cmd {
	if v.State.SelectedTask == nil {
		return nil
	}
	taskID := v.State.SelectedTask.ID
	return func() tea.Msg {
		comments, err := v.Client.GetComments(taskID, "")
		if err != nil {
			return errMsg{err}
		}
		return commentsLoadedMsg{comments: comments}
	}
}

func (v *ProjectsView) buildSidebarItems() {
	v.State.SidebarItems = []components.SidebarItem{}

	counts := make(map[string]int)
	for _, t := range v.State.AllTasks {
		if !t.Checked && !t.IsDeleted && t.ProjectID != "" {
			counts[t.ProjectID]++
		}
	}

	// Favorites first
	for _, p := range v.State.Projects {
		if p.InboxProject {
			continue
		}
		if p.IsFavorite {
			v.State.SidebarItems = append(v.State.SidebarItems, components.SidebarItem{
				Type:       "project",
				ID:         p.ID,
				Name:       p.Name,
				Icon:       "❤︎",
				Count:      counts[p.ID],
				IsFavorite: true,
				ParentID:   p.ParentID,
				Color:      p.Color,
			})
		}
	}

	// Separator if favorites exist
	hasFavorites := false
	for _, p := range v.State.Projects {
		if p.IsFavorite {
			hasFavorites = true
			break
		}
	}
	if hasFavorites {
		v.State.SidebarItems = append(v.State.SidebarItems, components.SidebarItem{Type: "separator"})
	}

	// Non-favorites
	for _, p := range v.State.Projects {
		if p.InboxProject || p.IsFavorite {
			continue
		}
		v.State.SidebarItems = append(v.State.SidebarItems, components.SidebarItem{
			Type:     "project",
			ID:       p.ID,
			Name:     p.Name,
			Icon:     "#",
			Count:    counts[p.ID],
			ParentID: p.ParentID,
			Color:    p.Color,
		})
	}

	// Sort by name
	sort.Slice(v.State.SidebarItems, func(i, j int) bool {
		if v.State.SidebarItems[i].Type == "separator" || v.State.SidebarItems[j].Type == "separator" {
			return false
		}
		return v.State.SidebarItems[i].Name < v.State.SidebarItems[j].Name
	})
}
