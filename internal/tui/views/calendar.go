package views

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// CalendarView handles the Calendar tab.
type CalendarView struct {
	*BaseView
}

// NewCalendarView creates a new CalendarView.
func NewCalendarView(s *state.State) *CalendarView {
	return &CalendarView{BaseView: NewBaseView(s)}
}

func (v *CalendarView) Name() string { return "calendar" }

func (v *CalendarView) OnEnter() tea.Cmd {
	v.State.CurrentProject = nil
	v.State.FocusedPane = state.PaneMain
	v.State.CalendarDate = time.Now()
	v.State.CalendarDay = time.Now().Day()
	return nil
}

func (v *CalendarView) OnExit() {}

func (v *CalendarView) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	// Calendar has its own key handling for navigation
	// This will be delegated from the existing handleCalendarKeyMsg
	return nil, false
}

func (v *CalendarView) HandleSelect() tea.Cmd {
	// Open day detail view
	v.State.PreviousView = v.State.CurrentView
	v.State.CurrentView = state.ViewCalendarDay
	v.State.TaskCursor = 0
	return v.loadCalendarDayTasks()
}

func (v *CalendarView) HandleBack() (tea.Cmd, bool) {
	if v.State.CurrentView == state.ViewCalendarDay {
		v.State.CurrentView = state.ViewCalendar
		v.State.TaskCursor = 0
		return v.loadAllTasks(), false
	}
	return nil, false
}

func (v *CalendarView) Render(width, height int) string { return "" }

// --- Private helpers ---

func (v *CalendarView) loadCalendarDayTasks() tea.Cmd {
	selectedDate := time.Date(
		v.State.CalendarDate.Year(),
		v.State.CalendarDate.Month(),
		v.State.CalendarDay, 0, 0, 0, 0, time.Local,
	)
	dateStr := selectedDate.Format("2006-01-02")

	var dayTasks []api.Task
	for _, t := range v.State.AllTasks {
		if t.Due != nil && t.Due.Date == dateStr {
			dayTasks = append(dayTasks, t)
		}
	}
	v.State.Tasks = dayTasks
	return nil
}

func (v *CalendarView) loadAllTasks() tea.Cmd {
	return func() tea.Msg {
		allTasks, err := v.Client.GetTasks(api.TaskFilter{})
		if err != nil {
			return errMsg{err}
		}
		return allTasksLoadedMsg{allTasks: allTasks}
	}
}

type allTasksLoadedMsg struct {
	allTasks []api.Task
}
