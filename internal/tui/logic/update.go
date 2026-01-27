package logic

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

type Handler struct {
	*state.State
}

func NewHandler(s *state.State) *Handler {
	return &Handler{State: s}
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

	case errMsg:
		h.Loading = false
		h.Err = msg.err
		return nil

	case statusMsg:
		h.StatusMsg = msg.msg
		return nil

	case dataLoadedMsg:
		return h.handleDataLoaded(msg)

	case taskUpdatedMsg, taskDeletedMsg, taskCompletedMsg, taskCreatedMsg:
		return h.handleTaskMsgs(msg)

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

	return nil
}

func (h *Handler) handleWindowSizeMsg(msg tea.WindowSizeMsg) tea.Cmd {
	h.Width = msg.Width
	h.Height = msg.Height
	// Initialize or update viewport dimensions
	// Reserve space for tab bar (~3 lines), status bar (2 lines), borders (2 lines), title (2 lines)
	vpHeight := msg.Height - 9
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

	if len(msg.allTasks) > 0 {
		h.AllTasks = msg.allTasks
	}

	if len(msg.projects) > 0 {
		h.Projects = msg.projects
		h.buildSidebarItems()

		// Calculate task counts
		counts := make(map[string]int)
		for _, t := range h.AllTasks {
			if !t.Checked && !t.IsDeleted {
				counts[t.ProjectID]++
			}
		}

		// Sync sidebar component with counts
		h.SidebarComp.SetProjects(msg.projects, counts)
	}
	if msg.tasks != nil {
		h.Tasks = msg.tasks
		h.sortTasks()
	}
	if len(msg.labels) > 0 {
		h.Labels = msg.labels
	}
	if len(msg.sections) > 0 {
		h.Sections = msg.sections
		// Sort sections by SectionOrder
		sort.Slice(h.Sections, func(i, j int) bool {
			return h.Sections[i].SectionOrder < h.Sections[j].SectionOrder
		})
	}
	if len(msg.allSections) > 0 {
		h.AllSections = msg.allSections
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
	case state.TabProjects:
		if h.CurrentProject != nil {
			// Reload current project
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
	default:
		// state.TabToday or fallback
		return h.loadTodayTasks()
	}
}

// handleMouseMsg processes mouse input.
