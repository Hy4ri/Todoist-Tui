package logic

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/components"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
	"github.com/hy4ri/todoist-tui/internal/tui/views"
)

type Handler struct {
	*state.State
	coordinator *views.Coordinator
}

func NewHandler(s *state.State) *Handler {
	return &Handler{
		State:       s,
		coordinator: views.NewCoordinator(s),
	}
}

func (h *Handler) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return h.handleKeyMsg(msg)

	case tea.MouseMsg:
		return h.handleMouseMsg(msg)

	case tea.WindowSizeMsg:
		return h.handleWindowSizeMsg(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		h.Spinner, cmd = h.Spinner.Update(msg)
		return cmd

	case checkDueMsg:
		return h.handleCheckDue(time.Time(msg))

	case errMsg:
		h.Loading = false
		h.Err = msg.err
		return nil

	case statusMsg:
		h.StatusMsg = msg.msg
		return nil

	case dataLoadedMsg:
		return h.handleDataLoaded(msg)

	case taskCompletedMsg:
		h.Loading = false
		h.updateStatsOnCompletion()
		return nil

	case taskDeletedMsg:
		h.Loading = false
		return nil

	case taskUpdatedMsg, taskCreatedMsg:
		return h.handleTaskMsgs(msg)

	case quickAddTaskCreatedMsg:
		// Task added via Quick Add - refresh but keep popup open
		h.StatusMsg = "Task added!"
		return h.refreshTasks()

	case projectCreatedMsg, projectUpdatedMsg, projectDeletedMsg:
		return h.handleProjectMsgs(msg)

	case labelCreatedMsg, labelUpdatedMsg, labelDeletedMsg:
		return h.handleLabelMsgs(msg)

	case sectionCreatedMsg, sectionUpdatedMsg, sectionDeletedMsg:
		return h.handleSectionMsgs(msg)

	case commentCreatedMsg:
		h.Loading = false
		h.StatusMsg = "Comment added"
		return h.loadTaskComments()

	case commentUpdatedMsg:
		h.Loading = false
		h.StatusMsg = "Comment updated"
		return h.loadTaskComments()

	case commentDeletedMsg:
		h.Loading = false
		h.StatusMsg = "Comment deleted"
		return h.loadTaskComments()

	case components.EditCommentMsg:
		h.IsEditingComment = true
		h.EditingComment = msg.Comment
		h.CommentInput = textinput.New()
		h.CommentInput.SetValue(msg.Comment.Content)
		h.CommentInput.Focus()
		h.CommentInput.Width = 50
		return textinput.Blink

	case components.DeleteCommentMsg:
		// Find comment object for context
		for i := range h.Comments {
			if h.Comments[i].ID == msg.CommentID {
				h.EditingComment = &h.Comments[i]
				break
			}
		}
		h.ConfirmDeleteComment = true
		return nil

	case subtaskCreatedMsg:
		h.Loading = false
		h.StatusMsg = "Subtask created"
		// Reload current view to show subtask
		return func() tea.Msg { return refreshMsg{} }

	case undoCompletedMsg:
		h.Loading = false
		h.StatusMsg = "Undo successful"
		// Reload current view
		return func() tea.Msg { return refreshMsg{} }

	case searchRefreshMsg:
		h.Loading = false
		h.StatusMsg = "Task updated"
		return h.refreshSearchResults()

	case refreshMsg:
		return h.handleRefresh()

	case commentsLoadedMsg:
		h.Comments = msg.comments
		return nil

	case reorderCompleteMsg:
		if h.CurrentProject != nil {
			return h.loadProjectTasks(h.CurrentProject.ID)
		}
		return nil
	}

	// Forward non-key messages (like blink) to active inputs

	// Task Form
	if h.CurrentView == state.ViewTaskForm && h.TaskForm != nil {
		return h.TaskForm.Update(msg)
	}

	// Quick Add Form
	if h.CurrentView == state.ViewQuickAdd && h.QuickAddForm != nil {
		return h.QuickAddForm.Update(msg)
	}

	// Search
	if h.CurrentView == state.ViewSearch {
		var cmd tea.Cmd
		h.SearchInput, cmd = h.SearchInput.Update(msg)
		return cmd
	}

	// Project Input
	if h.IsCreatingProject || h.IsEditingProject {
		var cmd tea.Cmd
		h.ProjectInput, cmd = h.ProjectInput.Update(msg)
		return cmd
	}

	// Label Input
	if h.IsCreatingLabel || h.IsEditingLabel {
		var cmd tea.Cmd
		h.LabelInput, cmd = h.LabelInput.Update(msg)
		return cmd
	}

	// Section Input
	if h.IsCreatingSection || h.IsEditingSection {
		var cmd tea.Cmd
		h.SectionInput, cmd = h.SectionInput.Update(msg)
		return cmd
	}

	// Subtask Input
	if h.IsCreatingSubtask {
		var cmd tea.Cmd
		h.SubtaskInput, cmd = h.SubtaskInput.Update(msg)
		return cmd
	}

	// Comment inputs
	if h.IsEditingComment || h.IsAddingComment {
		var cmd tea.Cmd
		h.CommentInput, cmd = h.CommentInput.Update(msg)
		return cmd
	}

	return nil
}

func (h *Handler) handleWindowSizeMsg(msg tea.WindowSizeMsg) tea.Cmd {
	h.Width = msg.Width
	h.Height = msg.Height
	// Initialize or update viewport dimensions
	// Reserve space for tab bar (~3 lines), status bar (1 line), borders (2 lines), title (2 lines)
	vpHeight := msg.Height - 8
	if vpHeight < 5 {
		vpHeight = 5
	}
	vpWidth := msg.Width - 4
	if vpWidth < 20 {
		vpWidth = 20
	}
	if !h.State.ViewportReady {
		h.TaskViewport = viewport.New(vpWidth, vpHeight)
		h.TaskViewport.Style = lipgloss.NewStyle()
		h.TaskViewport.MouseWheelEnabled = true
		h.State.ViewportReady = true
	} else {
		h.TaskViewport.Width = vpWidth
		h.TaskViewport.Height = vpHeight
	}

	h.HelpComp.SetSize(msg.Width, msg.Height)

	if h.TaskForm != nil {
		h.TaskForm.SetWidth(msg.Width)
	}

	return nil
}

func (h *Handler) handleDataLoaded(msg dataLoadedMsg) tea.Cmd {
	h.Loading = false

	dataChanged := false

	if len(msg.allTasks) > 0 {
		h.AllTasks = msg.allTasks
		dataChanged = true
		h.TasksByDate = make(map[string][]api.Task)

		// Optimization: Pre-parse task dates and group by date
		for i := range h.AllTasks {
			t := &h.AllTasks[i]
			if t.Due != nil {
				dateStr := t.Due.Date
				if len(dateStr) > 10 {
					dateStr = dateStr[:10]
				}
				h.TasksByDate[dateStr] = append(h.TasksByDate[dateStr], *t)

				// Parse full datetime if available, otherwise just date
				if t.Due.Datetime != nil {
					if parsed, err := time.Parse(time.RFC3339, *t.Due.Datetime); err == nil {
						localParsed := parsed.Local()
						t.ParsedDate = &localParsed
					}
				}

				// Fallback to date only if Datetime not present or failed to parse
				if t.ParsedDate == nil {
					if parsed, err := time.ParseInLocation("2006-01-02", dateStr, time.Local); err == nil {
						t.ParsedDate = &parsed
					}
				}
			}
		}
	}

	if len(msg.projects) > 0 {
		h.Projects = msg.projects
	}

	// Always rebuild sidebar items and update counts if we have projects
	if len(h.Projects) > 0 {
		h.buildSidebarItems()

		// Calculate task counts
		counts := make(map[string]int)
		for _, t := range h.AllTasks {
			if !t.Checked && !t.IsDeleted {
				counts[t.ProjectID]++
			}
		}

		// Sync sidebar component with counts
		h.SidebarComp.SetProjects(h.Projects, counts)
	}
	if msg.tasks != nil {
		h.Tasks = msg.tasks
		h.sortTasks()
		dataChanged = true
	}
	if len(msg.labels) > 0 {
		h.Labels = msg.labels
	}
	if msg.sections != nil {
		h.Sections = msg.sections
		// Sort sections by SectionOrder
		sort.Slice(h.Sections, func(i, j int) bool {
			return h.Sections[i].SectionOrder < h.Sections[j].SectionOrder
		})
	}
	if len(msg.allSections) > 0 {
		h.AllSections = msg.allSections
	}
	if msg.stats != nil {
		h.ProductivityStats = msg.stats
	}
	if msg.statsErr != nil {
		h.StatsError = msg.statsErr.Error()
	} else {
		h.StatsError = ""
	}

	// Restore cursor position if we have a task ID to restore to
	if h.RestoreCursorToTaskID != "" {
		for i, task := range h.Tasks {
			if task.ID == h.RestoreCursorToTaskID {
				h.TaskCursor = i
				break
			}
		}
		h.RestoreCursorToTaskID = "" // Clear after restoring
	}

	if dataChanged {
		h.DataVersion++
	}

	return nil
}

func (h *Handler) handleTaskMsgs(msg tea.Msg) tea.Cmd {
	h.Loading = false
	switch msg.(type) {
	case taskDeletedMsg:
		h.StatusMsg = "Task deleted"
	case taskCreatedMsg:
		h.StatusMsg = "Task saved"
		h.CurrentView = h.PreviousView
		h.TaskForm = nil
	}
	return h.refreshTasks()
}

func (h *Handler) handleProjectMsgs(msg tea.Msg) tea.Cmd {
	h.Loading = false
	switch m := msg.(type) {
	case projectCreatedMsg:
		h.StatusMsg = fmt.Sprintf("Created project: %s", m.project.Name)
	case projectUpdatedMsg:
		h.StatusMsg = fmt.Sprintf("Updated project: %s", m.project.Name)
	case projectDeletedMsg:
		h.StatusMsg = "Project deleted"
		h.SidebarCursor = 0
	}
	return h.loadProjects()
}

func (h *Handler) handleLabelMsgs(msg tea.Msg) tea.Cmd {
	h.Loading = false
	switch m := msg.(type) {
	case labelCreatedMsg:
		h.StatusMsg = fmt.Sprintf("Created label: %s", m.label.Name)
	case labelUpdatedMsg:
		h.StatusMsg = fmt.Sprintf("Updated label: %s", m.label.Name)
	case labelDeletedMsg:
		h.StatusMsg = "Label deleted"
		h.TaskCursor = 0
	}
	return h.loadLabels()
}

func (h *Handler) handleSectionMsgs(msg tea.Msg) tea.Cmd {
	h.Loading = false
	switch m := msg.(type) {
	case sectionCreatedMsg:
		h.StatusMsg = fmt.Sprintf("Created section: %s", m.section.Name)
	case sectionUpdatedMsg:
		h.StatusMsg = fmt.Sprintf("Updated section: %s", m.section.Name)
	case sectionDeletedMsg:
		h.StatusMsg = "Section deleted"
	}

	if h.SidebarCursor < len(h.SidebarItems) {
		return h.loadProjectTasks(h.SidebarItems[h.SidebarCursor].ID)
	}
	return nil
}

func (h *Handler) handleRefresh() tea.Cmd {
	// Store current task ID to restore cursor position after reload
	if len(h.Tasks) > 0 && h.TaskCursor >= 0 && h.TaskCursor < len(h.Tasks) {
		h.RestoreCursorToTaskID = h.Tasks[h.TaskCursor].ID
	}
	h.Loading = true

	if h.CurrentView == state.ViewSearch {
		return h.refreshSearchResults()
	}

	switch h.CurrentTab {
	case state.TabInbox:
		return h.loadInboxTasks()
	case state.TabProjects:
		if h.CurrentProject != nil {
			// Reload current project
			h.Sections = nil // Clear sections to ensure clean reload
			return h.loadProjectTasks(h.CurrentProject.ID)
		}
		// Just reload project list
		return h.loadProjects()
	case state.TabUpcoming:
		return h.loadUpcomingTasks()
	case state.TabCalendar:
		// Preserve calendar date and reload all tasks
		return h.loadAllTasks()
	case state.TabLabels:
		if h.CurrentLabel != nil {
			return h.loadLabelTasks(h.CurrentLabel.Name)
		}
		return h.loadLabels()
	case state.TabToday:
		return h.loadTodayTasks()
	default:
		return h.loadTodayTasks()
	}
}

// updateStatsOnCompletion updates the productivity stats when a task is completed.
func (h *Handler) updateStatsOnCompletion() {
	if h.ProductivityStats == nil {
		return
	}

	todayStr := time.Now().Format("2006-01-02")
	goals := h.ProductivityStats.Goals
	var msg string

	// Update daily stats
	foundToday := false
	for i := range h.ProductivityStats.DaysItems {
		if h.ProductivityStats.DaysItems[i].Date == todayStr {
			h.ProductivityStats.DaysItems[i].TotalCompleted++
			newCount := h.ProductivityStats.DaysItems[i].TotalCompleted
			if newCount == goals.DailyGoal {
				msg = "Daily goal reached! 🎉"
			} else if newCount > goals.DailyGoal {
				msg = fmt.Sprintf("Daily goal exceeded! (%d/%d) 🔥", newCount, goals.DailyGoal)
			}
			foundToday = true
			break
		}
	}
	if !foundToday {
		h.ProductivityStats.DaysItems = append(h.ProductivityStats.DaysItems, api.DayItems{
			Date:           todayStr,
			TotalCompleted: 1,
		})
		if goals.DailyGoal == 1 {
			msg = "Daily goal reached! 🎉"
		}
	}

	// Update weekly stats
	if len(h.ProductivityStats.WeekItems) > 0 {
		h.ProductivityStats.WeekItems[0].TotalCompleted++
		// Only celebrate weekly if daily wasn't just celebrated (avoid spam)
		if msg == "" {
			newCount := h.ProductivityStats.WeekItems[0].TotalCompleted
			if newCount == goals.WeeklyGoal {
				msg = "Weekly goal reached! 🚀"
			}
		}
	}

	if msg != "" {
		h.StatusMsg = msg
	}
}
