package logic

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// Init implements tea.Model.
func (h *Handler) Init() tea.Cmd {
	return tea.Batch(
		h.Spinner.Tick,
		h.LoadInitialData(),
		checkDueCmd(),
	)
}

// loadInitialData loads all necessary data concurrently.
func (h *Handler) LoadInitialData() tea.Cmd {
	return func() tea.Msg {
		// Create buffered channels to prevent goroutine leaks on early return
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
		type statsResult struct {
			data *api.ProductivityStats
			err  error
		}

		projChan := make(chan projectResult, 1)
		labelChan := make(chan labelResult, 1)
		taskChan := make(chan taskResult, 1)
		secChan := make(chan sectionResult, 1)
		statsChan := make(chan statsResult, 1)

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

		go func() {
			s, e := h.Client.GetProductivityStats()
			statsChan <- statsResult{data: s, err: e}
		}()

		// Collect ALL results before processing errors to ensure all goroutines exit
		pRes := <-projChan
		lRes := <-labelChan
		tRes := <-taskChan
		sRes := <-secChan
		statsRes := <-statsChan

		// Now check for errors
		if pRes.err != nil {
			return errMsg{pRes.err}
		}
		if lRes.err != nil {
			return errMsg{lRes.err}
		}
		if tRes.err != nil {
			return errMsg{tRes.err}
		}
		if sRes.err != nil {
			return errMsg{sRes.err}
		}
		// Stats errors are non-fatal - just capture them
		var prodStats *api.ProductivityStats
		var statsErr error
		if statsRes.err == nil {
			prodStats = statsRes.data
		} else {
			statsErr = statsRes.err
		}

		projects := pRes.data
		labels := lRes.data
		allTasks := tRes.data
		allSections := sRes.data

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
			initialTasks = nil
		case state.TabProjects, state.TabLabels:
			initialTasks = nil
		case state.TabInbox:
			// Find inbox ID
			var inboxID string
			for _, p := range projects {
				if p.InboxProject {
					inboxID = p.ID
					break
				}
			}
			for _, t := range allTasks {
				if t.ProjectID == inboxID && !t.Checked && !t.IsDeleted {
					initialTasks = append(initialTasks, t)
				}
			}
		default:
			// TabToday or fallback
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
			stats:       prodStats,
			statsErr:    statsErr,
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
	stats       *api.ProductivityStats
	statsErr    error
}
type taskUpdatedMsg struct{ task *api.Task }
type taskDeletedMsg struct{ id string }
type taskCompletedMsg struct{ id string }
type taskCreatedMsg struct{}
type quickAddTaskCreatedMsg struct{}
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
type commentUpdatedMsg struct{ comment *api.Comment }
type commentDeletedMsg struct{ id string }
type subtaskCreatedMsg struct{}
type undoCompletedMsg struct{}
type searchRefreshMsg struct{}
type refreshMsg struct{}
type commentsLoadedMsg struct{ comments []api.Comment }

type reorderCompleteMsg struct{}

// Update implements tea.Model.
