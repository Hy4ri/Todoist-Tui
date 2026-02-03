package views

import (
	"sort"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// LabelsView handles the Labels tab.
type LabelsView struct {
	*BaseView
}

// NewLabelsView creates a new LabelsView.
func NewLabelsView(s *state.State) *LabelsView {
	return &LabelsView{BaseView: NewBaseView(s)}
}

func (v *LabelsView) Name() string { return "labels" }

func (v *LabelsView) OnEnter() tea.Cmd {
	v.State.TaskCursor = 0
	v.State.CurrentProject = nil
	v.State.CurrentLabel = nil
	v.State.FocusedPane = state.PaneMain
	return nil
}

func (v *LabelsView) OnExit() {
	v.State.CurrentLabel = nil
}

func (v *LabelsView) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	return nil, false
}

func (v *LabelsView) HandleSelect() tea.Cmd {
	if v.State.CurrentLabel == nil {
		// Select a label from the list
		labels := v.getLabels()
		if v.State.TaskCursor < len(labels) {
			v.State.CurrentLabel = &labels[v.State.TaskCursor]
			v.State.TaskCursor = 0
			return v.filterLabelTasks(v.State.CurrentLabel.Name)
		}
	} else {
		// Select a task from the label's task list
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
	return nil
}

func (v *LabelsView) HandleBack() (tea.Cmd, bool) {
	if v.State.ShowDetailPanel {
		v.State.ShowDetailPanel = false
		v.State.SelectedTask = nil
		v.State.Comments = nil
		return nil, false
	}
	if v.State.CurrentLabel != nil {
		v.State.CurrentLabel = nil
		v.State.Tasks = nil
		v.State.TaskCursor = 0
		return nil, false
	}
	return nil, false
}

func (v *LabelsView) Render(width, height int) string { return "" }

// --- Private helpers ---

func (v *LabelsView) getLabels() []api.Label {
	if len(v.State.Labels) > 0 {
		return v.State.Labels
	}
	return v.extractLabelsFromTasks()
}

func (v *LabelsView) extractLabelsFromTasks() []api.Label {
	labelSet := make(map[string]bool)
	var labels []api.Label

	tasksToScan := v.State.AllTasks
	if len(tasksToScan) == 0 {
		tasksToScan = v.State.Tasks
	}

	for _, t := range tasksToScan {
		for _, labelName := range t.Labels {
			if !labelSet[labelName] {
				labelSet[labelName] = true
				labels = append(labels, api.Label{Name: labelName})
			}
		}
	}

	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Name < labels[j].Name
	})

	return labels
}

func (v *LabelsView) filterLabelTasks(labelName string) tea.Cmd {
	var tasks []api.Task
	for _, t := range v.State.AllTasks {
		for _, l := range t.Labels {
			if l == labelName {
				tasks = append(tasks, t)
				break
			}
		}
	}
	v.State.Tasks = tasks
	return nil
}

func (v *LabelsView) loadTaskComments() tea.Cmd {
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
