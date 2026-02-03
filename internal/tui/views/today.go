package views

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// TodayView handles the Today tab displaying today's and overdue tasks.
type TodayView struct {
	*BaseView
}

// NewTodayView creates a new TodayView.
func NewTodayView(s *state.State) *TodayView {
	return &TodayView{
		BaseView: NewBaseView(s),
	}
}

// Name returns the view identifier.
func (v *TodayView) Name() string {
	return "today"
}

// OnEnter is called when switching to this view.
func (v *TodayView) OnEnter() tea.Cmd {
	v.State.TaskCursor = 0
	v.State.CurrentProject = nil
	v.State.FocusedPane = state.PaneMain

	// Filter today/overdue tasks from cached AllTasks
	return v.filterTodayTasks()
}

// OnExit is called when leaving this view.
func (v *TodayView) OnExit() {
	// Nothing to clean up for Today view
}

// HandleKey processes keyboard input for this view.
func (v *TodayView) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	// Today view uses default key handling from the base handler
	// View-specific keys could be added here
	return nil, false
}

// HandleSelect processes Enter/selection for this view.
func (v *TodayView) HandleSelect() tea.Cmd {
	task := v.GetSelectedTask()
	if task == nil {
		return nil
	}

	// Show task detail panel
	taskCopy := new(api.Task)
	*taskCopy = *task
	v.State.SelectedTask = taskCopy
	v.State.ShowDetailPanel = true

	return v.loadTaskComments()
}

// HandleBack processes Escape for this view.
func (v *TodayView) HandleBack() (tea.Cmd, bool) {
	// Close detail panel if open
	if v.State.ShowDetailPanel {
		v.State.ShowDetailPanel = false
		v.State.SelectedTask = nil
		v.State.Comments = nil
		return nil, false // Don't exit view, just close panel
	}

	// Nothing else to go back from in Today view
	return nil, false
}

// Render returns the view's content.
func (v *TodayView) Render(width, height int) string {
	// Delegate to existing renderer for now
	// This will be migrated later to be self-contained
	return ""
}

// --- Private helpers ---

// filterTodayTasks filters cached tasks for today/overdue.
func (v *TodayView) filterTodayTasks() tea.Cmd {
	var tasks []api.Task
	for _, t := range v.State.AllTasks {
		if t.IsOverdue() || t.IsDueToday() {
			tasks = append(tasks, t)
		}
	}
	v.State.Tasks = tasks
	return nil
}

// loadTaskComments loads comments for the selected task.
func (v *TodayView) loadTaskComments() tea.Cmd {
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

// Message types (will be moved to messages.go)
type errMsg struct{ err error }
type commentsLoadedMsg struct{ comments []api.Comment }
