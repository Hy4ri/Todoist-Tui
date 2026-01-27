package logic

import (
	"github.com/hy4ri/todoist-tui/internal/tui/state"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
)

// Init implements tea.Model.
func (h *Handler) Init() tea.Cmd {
	return tea.Batch(
		h.Spinner.Tick,
		h.LoadInitialData(),
	)
}

// loadInitialData loads all necessary data concurrently.
func (h *Handler) LoadInitialData() tea.Cmd {
	return func() tea.Msg {
		var (
			projects    []api.Project
			labels      []api.Label
			allTasks    []api.Task
			allSections []api.Section
		)

		// Create channels for results
		type projectResult struct {
			data []api.Project
			err  error
		}
		type labelResult struct {
			data []api.Label
			err  error
		}
		type taskResult struct {
			data []api.Task
			err  error
		}
		type sectionResult struct {
			data []api.Section
			err  error
		}

		projChan := make(chan projectResult)
		labelChan := make(chan labelResult)
		taskChan := make(chan taskResult)
		secChan := make(chan sectionResult)

		// Launch concurrent requests
		go func() {
			p, e := h.Client.GetProjects()
			projChan <- projectResult{data: p, err: e}
		}()

		go func() {
			l, e := h.Client.GetLabels()
			labelChan <- labelResult{data: l, err: e}
		}()

		go func() {
			t, e := h.Client.GetTasks(api.TaskFilter{})
			taskChan <- taskResult{data: t, err: e}
		}()

		go func() {
			s, e := h.Client.GetSections("")
			secChan <- sectionResult{data: s, err: e}
		}()

		// Collect results
		pRes := <-projChan
		if pRes.err != nil {
			return errMsg{pRes.err}
		}
		projects = pRes.data

		lRes := <-labelChan
		if lRes.err != nil {
			return errMsg{lRes.err}
		}
		labels = lRes.data

		tRes := <-taskChan
		if tRes.err != nil {
			return errMsg{tRes.err}
		}
		allTasks = tRes.data

		sRes := <-secChan
		if sRes.err != nil {
			return errMsg{sRes.err}
		}
		allSections = sRes.data

		// Filter tasks for the initial view
		var initialTasks []api.Task
		switch h.CurrentTab {
		case state.TabUpcoming:
			for _, t := range allTasks {
				if t.Due != nil {
					initialTasks = append(initialTasks, t)
				}
			}
		case state.TabCalendar:
			// In calendar main view, we don't necessarily show a list initially,
			// or we show allTasks. DataLoadedMsg handler will handle the display.
			initialTasks = nil
		case state.TabProjects, state.TabLabels:
			// Items are selected from sidebar/list
			initialTasks = nil
		default:
			// state.TabToday or fallback
			for _, t := range allTasks {
				if t.IsOverdue() || t.IsDueToday() {
					initialTasks = append(initialTasks, t)
				}
			}
		}

		return dataLoadedMsg{
			projects:    projects,
			tasks:       initialTasks,
			allTasks:    allTasks,
			labels:      labels,
			allSections: allSections,
		}
	}
}

// Message types
type errMsg struct{ err error }
type statusMsg struct{ msg string }
type dataLoadedMsg struct {
	projects    []api.Project
	tasks       []api.Task
	allTasks    []api.Task
	sections    []api.Section
	allSections []api.Section
	labels      []api.Label
}
type taskUpdatedMsg struct{ task *api.Task }
type taskDeletedMsg struct{ id string }
type taskCompletedMsg struct{ id string }
type taskCreatedMsg struct{}
type projectCreatedMsg struct{ project *api.Project }
type projectUpdatedMsg struct{ project *api.Project }
type projectDeletedMsg struct{ id string }
type labelCreatedMsg struct{ label *api.Label }
type labelUpdatedMsg struct{ label *api.Label }
type labelDeletedMsg struct{ id string }
type sectionCreatedMsg struct{ section *api.Section }
type sectionUpdatedMsg struct{ section *api.Section }
type sectionDeletedMsg struct{ id string }
type commentCreatedMsg struct{ comment *api.Comment }
type subtaskCreatedMsg struct{}
type undoCompletedMsg struct{}
type searchRefreshMsg struct{}
type refreshMsg struct{}
type commentsLoadedMsg struct{ comments []api.Comment }

type reorderCompleteMsg struct{}

// Update implements tea.Model.
