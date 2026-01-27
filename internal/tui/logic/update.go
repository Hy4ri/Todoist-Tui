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

		return nil

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

	case taskUpdatedMsg:
		h.Loading = false
		// Refresh the task list
		return h.refreshTasks()

	case taskDeletedMsg:
		h.Loading = false
		h.StatusMsg = "Task deleted"
		// Keep selections after delete (remaining tasks stay visible)
		return h.refreshTasks()

	case taskCompletedMsg:
		h.Loading = false
		// Keep selections after complete (tasks remain visible)
		return h.refreshTasks()

	case taskCreatedMsg:
		h.Loading = false
		h.StatusMsg = "Task saved"
		h.CurrentView = h.PreviousView
		h.TaskForm = nil
		return h.refreshTasks()

	case projectCreatedMsg:
		h.Loading = false
		h.StatusMsg = fmt.Sprintf("Created project: %s", msg.project.Name)
		// Reload projects
		return h.loadProjects()

	case projectUpdatedMsg:
		h.Loading = false
		h.StatusMsg = fmt.Sprintf("Updated project: %s", msg.project.Name)
		// Reload projects
		return h.loadProjects()

	case projectDeletedMsg:
		h.Loading = false
		h.StatusMsg = "Project deleted"
		h.SidebarCursor = 0
		// Reload projects and switch to first project
		return h.loadProjects()

	case labelCreatedMsg:
		h.Loading = false
		h.StatusMsg = fmt.Sprintf("Created label: %s", msg.label.Name)
		// Reload labels
		return h.loadLabels()

	case labelUpdatedMsg:
		h.Loading = false
		h.StatusMsg = fmt.Sprintf("Updated label: %s", msg.label.Name)
		return h.loadLabels()

	case labelDeletedMsg:
		h.Loading = false
		h.StatusMsg = "Label deleted"
		h.TaskCursor = 0
		return h.loadLabels()

	case sectionCreatedMsg:
		h.Loading = false
		h.StatusMsg = fmt.Sprintf("Created section: %s", msg.section.Name)
		// Reload current project
		if h.SidebarCursor < len(h.SidebarItems) {
			return h.loadProjectTasks(h.SidebarItems[h.SidebarCursor].ID)
		}
		return nil

	case sectionUpdatedMsg:
		h.Loading = false
		h.StatusMsg = fmt.Sprintf("Updated section: %s", msg.section.Name)
		if h.SidebarCursor < len(h.SidebarItems) {
			return h.loadProjectTasks(h.SidebarItems[h.SidebarCursor].ID)
		}
		return nil

	case sectionDeletedMsg:
		h.Loading = false
		h.StatusMsg = "Section deleted"
		if h.SidebarCursor < len(h.SidebarItems) {
			return h.loadProjectTasks(h.SidebarItems[h.SidebarCursor].ID)
		}
		return nil

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

// handleMouseMsg processes mouse input.
