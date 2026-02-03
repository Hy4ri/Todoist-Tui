package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// UpcomingView handles the Upcoming tab.
type UpcomingView struct {
	*BaseView
}

// NewUpcomingView creates a new UpcomingView.
func NewUpcomingView(s *state.State) *UpcomingView {
	return &UpcomingView{BaseView: NewBaseView(s)}
}

func (v *UpcomingView) Name() string { return "upcoming" }

func (v *UpcomingView) OnEnter() tea.Cmd {
	v.State.TaskCursor = 0
	v.State.CurrentProject = nil
	v.State.FocusedPane = state.PaneMain
	return v.filterUpcomingTasks()
}

func (v *UpcomingView) OnExit() {}

func (v *UpcomingView) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	return nil, false
}

func (v *UpcomingView) HandleSelect() tea.Cmd {
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

func (v *UpcomingView) HandleBack() (tea.Cmd, bool) {
	if v.State.ShowDetailPanel {
		v.State.ShowDetailPanel = false
		v.State.SelectedTask = nil
		v.State.Comments = nil
		return nil, false
	}
	return nil, false
}

func (v *UpcomingView) Render(width, height int) string { return "" }

// --- Private helpers ---

func (v *UpcomingView) filterUpcomingTasks() tea.Cmd {
	var upcoming []api.Task
	for _, t := range v.State.AllTasks {
		if t.Due != nil {
			upcoming = append(upcoming, t)
		}
	}
	v.State.Tasks = upcoming
	return nil
}

func (v *UpcomingView) loadTaskComments() tea.Cmd {
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
