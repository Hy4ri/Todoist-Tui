package logic

import (
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/gen2brain/beeep"
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

	case completedTasksLoadedMsg:
		h.Loading = false
		h.CompletedTasks = msg
		h.Tasks = msg // Reuse Tasks slice for list rendering
		h.CompletedMore = len(msg) >= h.CompletedLimit && h.CompletedLimit > 0
		h.StatusMsg = "Completed tasks loaded"
		return nil

	case filtersLoadedMsg:
		h.Filters = msg.filters
		return nil

	case filterCreatedMsg:
		h.Loading = false
		h.StatusMsg = "Filter created: " + msg.filter.Name
		return h.loadFilters() // Refresh list

	case filterDeletedMsg:
		h.Loading = false
		h.StatusMsg = "Filter deleted"
		h.EditingFilter = nil
		// Re-adjust cursor if needed
		if h.FilterCursor >= len(h.Filters)-1 && h.FilterCursor > 0 {
			h.FilterCursor--
		}
		return h.loadFilters() // Refresh list

	case taskDeletedMsg, taskUpdatedMsg, taskCreatedMsg, taskCompletedMsg:
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
		// Invalidate cache for this task
		if h.SelectedTask != nil && h.CommentCache != nil {
			delete(h.CommentCache, h.SelectedTask.ID)
		}
		return h.loadTaskComments()

	case commentUpdatedMsg:
		h.Loading = false
		h.StatusMsg = "Comment updated"
		// Invalidate cache for this task
		if h.SelectedTask != nil && h.CommentCache != nil {
			delete(h.CommentCache, h.SelectedTask.ID)
		}
		return h.loadTaskComments()

	case commentDeletedMsg:
		h.Loading = false
		h.StatusMsg = "Comment deleted"
		// Invalidate cache for this task
		if h.SelectedTask != nil && h.CommentCache != nil {
			delete(h.CommentCache, h.SelectedTask.ID)
		}
		return h.loadTaskComments()

	case components.EditCommentMsg:
		h.IsEditingComment = true
		h.EditingComment = msg.Comment
		h.CommentInput = textarea.New()
		h.CommentInput.SetValue(msg.Comment.Content)
		h.CommentInput.Focus()
		h.CommentInput.SetWidth(50)
		h.CommentInput.SetHeight(3)
		h.CommentInput.ShowLineNumbers = false
		h.CommentInput.Prompt = ""
		return textarea.Blink

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
		return h.handleRefresh(msg.Force)

	case commentsLoadedMsg:
		h.Comments = msg.comments
		// Store in cache for instant retrieval next time
		if h.SelectedTask != nil {
			if h.CommentCache == nil {
				h.CommentCache = make(map[string][]api.Comment)
			}
			h.CommentCache[h.SelectedTask.ID] = msg.comments
		}
		return nil

	case reorderCompleteMsg:
		if h.CurrentProject != nil {
			return h.loadProjectTasks(h.CurrentProject.ID)
		}
		return nil

	case remindersFetchedMsg:
		h.Reminders = msg.reminders
		if h.ReminderCache == nil {
			h.ReminderCache = make(map[string][]api.Reminder)
		}
		h.ReminderCache[msg.taskID] = msg.reminders
		return nil

	case reminderCreatedMsg:
		h.Loading = false
		h.StatusMsg = "Reminder added"
		h.IsAddingReminder = false
		// Update cache and current view
		if h.SelectedTask != nil {
			return h.fetchReminders(h.SelectedTask.ID)
		}
		return nil

	case reminderDeletedMsg:
		h.Loading = false
		h.StatusMsg = "Reminder deleted"
		h.ConfirmDeleteReminder = false
		h.EditingReminder = nil
		if h.SelectedTask != nil {
			return h.fetchReminders(h.SelectedTask.ID)
		}
		return nil

	case components.TimerTickMsg:
		if !h.PomodoroRunning {
			return nil
		}
		h.PomodoroElapsed += time.Second

		if h.PomodoroMode == state.PomodoroCountdown {
			if h.PomodoroElapsed >= h.PomodoroTarget {
				h.PomodoroRunning = false
				// Trigger phase complete
				return func() tea.Msg { return components.TimerPhaseCompleteMsg{} }
			}
		}
		// Continue ticking
		return tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return components.TimerTickMsg{ID: msg.ID}
		})

	case components.TimerPhaseCompleteMsg:
		return h.handlePomodoroPhaseComplete()
	}

	// Reminder inputs
	if h.IsAddingReminder || h.IsEditingReminder {
		var cmd tea.Cmd
		if h.ReminderTypeCursor == 0 {
			h.ReminderMinuteInput, cmd = h.ReminderMinuteInput.Update(msg)
		} else {
			var cmd2 tea.Cmd
			h.ReminderDateInput, cmd = h.ReminderDateInput.Update(msg)
			h.ReminderTimeInput, cmd2 = h.ReminderTimeInput.Update(msg)
			cmd = tea.Batch(cmd, cmd2)
		}
		return cmd
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
		h.LastDataFetch = time.Now() // Track when data was fetched

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
		h.TasksSorted = false // Reset sorted flag since tasks changed
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

	if len(msg.reminders) > 0 {
		h.Reminders = msg.reminders
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
	case taskCompletedMsg:
		h.StatusMsg = "Task completed"
		h.updateStatsOnCompletion()
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

func (h *Handler) handleRefresh(force bool) tea.Cmd {
	// Store current task ID to restore cursor position after reload
	if len(h.Tasks) > 0 && h.TaskCursor >= 0 && h.TaskCursor < len(h.Tasks) {
		h.RestoreCursorToTaskID = h.Tasks[h.TaskCursor].ID
	}

	if h.CurrentView == state.ViewSearch {
		h.Loading = true
		return h.refreshSearchResults()
	}

	// Check if data is fresh (fetched within last 30 seconds)
	// If fresh, use local filtering for instant response
	// Manual refresh (force=true) always bypasses cache
	dataIsFresh := !force && len(h.AllTasks) > 0 && time.Since(h.LastDataFetch) < 30*time.Second

	switch h.CurrentTab {
	case state.TabInbox:
		if force || !dataIsFresh {
			h.Loading = true
			return h.refreshTasks()
		}
		return h.loadInboxTasks()
	case state.TabProjects:
		if h.CurrentProject != nil {
			if dataIsFresh {
				// Use cached data for instant filtering
				return h.filterProjectTasks(h.CurrentProject.ID)
			}
			// Reload current project from API
			h.Loading = true
			h.Sections = nil // Clear sections to ensure clean reload
			return h.loadProjectTasks(h.CurrentProject.ID)
		}
		// Just reload project list
		h.Loading = true
		return h.loadProjects()
	case state.TabUpcoming:
		if dataIsFresh {
			return h.filterUpcomingTasks()
		}
		h.Loading = true
		return h.loadUpcomingTasks()
	case state.TabCalendar:
		if dataIsFresh {
			return h.filterCalendarTasks()
		}
		h.Loading = true
		return h.loadAllTasks()
	case state.TabLabels:
		if h.CurrentLabel != nil {
			if dataIsFresh {
				return h.filterLabelTasks(h.CurrentLabel.Name)
			}
			h.Loading = true
			return h.loadLabelTasks(h.CurrentLabel.Name)
		}
		h.Loading = true
		return h.loadLabels()
	case state.TabToday:
		if dataIsFresh {
			return h.filterTodayTasks()
		}
		h.Loading = true
		return h.loadTodayTasks()
	default:
		if dataIsFresh {
			return h.filterTodayTasks()
		}
		h.Loading = true
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

// handlePomodoroPhaseComplete transitions to the next Pomodoro phase.
func (h *Handler) handlePomodoroPhaseComplete() tea.Cmd {
	h.PomodoroElapsed = 0
	if h.PomodoroPhase == state.PomodoroWork {
		h.PomodoroSessions++
		// Determine break length (every 4 sessions long break)
		if h.PomodoroSessions%4 == 0 {
			h.PomodoroPhase = state.PomodoroLongBreak
			h.PomodoroTarget = 15 * time.Minute
		} else {
			h.PomodoroPhase = state.PomodoroShortBreak
			// Scale break based on work duration (50m -> 10m break, 25m -> 5m break)
			if h.PomodoroTarget >= 50*time.Minute {
				h.PomodoroTarget = 10 * time.Minute
			} else {
				h.PomodoroTarget = 5 * time.Minute
			}
		}
	} else {
		h.PomodoroPhase = state.PomodoroWork
		// Restore focus target (default 25 or 50)
		if h.PomodoroTarget == 10*time.Minute || h.PomodoroTarget == 5*time.Minute || h.PomodoroTarget == 15*time.Minute {
			h.PomodoroTarget = 25 * time.Minute
		}
	}

	h.StatusMsg = "🍅 Pomodoro phase complete!"

	// Try to send desktop notification
	_ = h.notifyPhaseComplete()

	return nil
}

func (h *Handler) notifyPhaseComplete() error {
	title := "🍅 Pomodoro"
	message := "Phase complete!"
	if h.PomodoroPhase == state.PomodoroWork {
		message = "Time to focus!"
	} else {
		message = "Take a break!"
	}

	return beeep.Notify(title, message, "")
}

type completedTasksLoadedMsg []api.Task

// Reminder messages
type remindersFetchedMsg struct {
	taskID    string
	reminders []api.Reminder
}

type reminderCreatedMsg struct {
	reminder *api.Reminder
}

type reminderDeletedMsg struct {
	id string
}
