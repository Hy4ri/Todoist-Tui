package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// InboxView handles the Inbox tab.
type InboxView struct {
	*BaseView
}

// NewInboxView creates a new InboxView.
func NewInboxView(s *state.State) *InboxView {
	return &InboxView{BaseView: NewBaseView(s)}
}

func (v *InboxView) Name() string { return "inbox" }

func (v *InboxView) OnEnter() tea.Cmd {
	v.State.TaskCursor = 0
	v.State.CurrentProject = nil
	v.State.Sections = nil
	v.State.FocusedPane = state.PaneMain
	return v.loadInboxTasks()
}

func (v *InboxView) OnExit() {}

func (v *InboxView) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	return nil, false // Use default handling
}

func (v *InboxView) HandleSelect() tea.Cmd {
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

func (v *InboxView) HandleBack() (tea.Cmd, bool) {
	if v.State.ShowDetailPanel {
		v.State.ShowDetailPanel = false
		v.State.SelectedTask = nil
		v.State.Comments = nil
		return nil, false
	}
	return nil, false
}

func (v *InboxView) Render(width, height int) string { return "" }

// --- Private helpers ---

func (v *InboxView) loadInboxTasks() tea.Cmd {
	var inboxID string
	for _, p := range v.State.Projects {
		if p.InboxProject {
			inboxID = p.ID
			break
		}
	}

	if inboxID != "" {
		return func() tea.Msg {
			tasks, err := v.Client.GetTasks(api.TaskFilter{ProjectID: inboxID})
			if err != nil {
				return errMsg{err}
			}
			sections, err := v.Client.GetSections(inboxID)
			if err != nil {
				return errMsg{err}
			}
			return dataLoadedMsg{tasks: tasks, sections: sections}
		}
	}

	return func() tea.Msg {
		tasks, err := v.Client.GetTasksByFilter("inbox")
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{tasks: tasks, sections: []api.Section{}}
	}
}

func (v *InboxView) loadTaskComments() tea.Cmd {
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

// Message types
type dataLoadedMsg struct {
	tasks    []api.Task
	sections []api.Section
}
