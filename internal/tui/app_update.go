package tui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKeyMsg(msg)

	case tea.MouseMsg:
		return a.handleMouseMsg(msg)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
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
		if !a.viewportReady {
			a.taskViewport = viewport.New(vpWidth, vpHeight)
			a.taskViewport.Style = lipgloss.NewStyle()
			a.taskViewport.MouseWheelEnabled = true
			a.viewportReady = true
		} else {
			a.taskViewport.Width = vpWidth
			a.taskViewport.Height = vpHeight
		}

		a.helpComp.SetSize(msg.Width, msg.Height)

		return a, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		return a, cmd

	case errMsg:
		a.loading = false
		a.err = msg.err
		return a, nil

	case statusMsg:
		a.statusMsg = msg.msg
		return a, nil

	case dataLoadedMsg:
		a.loading = false

		if len(msg.allTasks) > 0 {
			a.allTasks = msg.allTasks
		}

		if len(msg.projects) > 0 {
			a.projects = msg.projects
			a.buildSidebarItems()

			// Calculate task counts
			counts := make(map[string]int)
			for _, t := range a.allTasks {
				if !t.Checked && !t.IsDeleted {
					counts[t.ProjectID]++
				}
			}

			// Sync sidebar component with counts
			a.sidebarComp.SetProjects(msg.projects, counts)
		}
		if msg.tasks != nil {
			a.tasks = msg.tasks
			a.sortTasks()
		}
		if len(msg.labels) > 0 {
			a.labels = msg.labels
		}
		if len(msg.sections) > 0 {
			a.sections = msg.sections
			// Sort sections by SectionOrder
			sort.Slice(a.sections, func(i, j int) bool {
				return a.sections[i].SectionOrder < a.sections[j].SectionOrder
			})
		}
		if len(msg.allSections) > 0 {
			a.allSections = msg.allSections
		}

		// Restore cursor position if we have a task ID to restore to
		if a.restoreCursorToTaskID != "" {
			for i, task := range a.tasks {
				if task.ID == a.restoreCursorToTaskID {
					a.taskCursor = i
					break
				}
			}
			a.restoreCursorToTaskID = "" // Clear after restoring
		}

		return a, nil

	case taskUpdatedMsg:
		a.loading = false
		// Refresh the task list
		return a, a.refreshTasks()

	case taskDeletedMsg:
		a.loading = false
		a.statusMsg = "Task deleted"
		// Keep selections after delete (remaining tasks stay visible)
		return a, a.refreshTasks()

	case taskCompletedMsg:
		a.loading = false
		// Keep selections after complete (tasks remain visible)
		return a, a.refreshTasks()

	case taskCreatedMsg:
		a.loading = false
		a.statusMsg = "Task saved"
		a.currentView = a.previousView
		a.taskForm = nil
		return a, a.refreshTasks()

	case projectCreatedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Created project: %s", msg.project.Name)
		// Reload projects
		return a, a.loadProjects()

	case projectUpdatedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Updated project: %s", msg.project.Name)
		// Reload projects
		return a, a.loadProjects()

	case projectDeletedMsg:
		a.loading = false
		a.statusMsg = "Project deleted"
		a.sidebarCursor = 0
		// Reload projects and switch to first project
		return a, a.loadProjects()

	case labelCreatedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Created label: %s", msg.label.Name)
		// Reload labels
		return a, a.loadLabels()

	case labelUpdatedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Updated label: %s", msg.label.Name)
		return a, a.loadLabels()

	case labelDeletedMsg:
		a.loading = false
		a.statusMsg = "Label deleted"
		a.taskCursor = 0
		return a, a.loadLabels()

	case sectionCreatedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Created section: %s", msg.section.Name)
		// Reload current project
		if a.sidebarCursor < len(a.sidebarItems) {
			return a, a.loadProjectTasks(a.sidebarItems[a.sidebarCursor].ID)
		}
		return a, nil

	case sectionUpdatedMsg:
		a.loading = false
		a.statusMsg = fmt.Sprintf("Updated section: %s", msg.section.Name)
		if a.sidebarCursor < len(a.sidebarItems) {
			return a, a.loadProjectTasks(a.sidebarItems[a.sidebarCursor].ID)
		}
		return a, nil

	case sectionDeletedMsg:
		a.loading = false
		a.statusMsg = "Section deleted"
		if a.sidebarCursor < len(a.sidebarItems) {
			return a, a.loadProjectTasks(a.sidebarItems[a.sidebarCursor].ID)
		}
		return a, nil

	case commentCreatedMsg:
		a.loading = false
		a.statusMsg = "Comment added"
		return a, a.loadTaskComments()

	case subtaskCreatedMsg:
		a.loading = false
		a.statusMsg = "Subtask created"
		// Reload current view to show subtask
		return a, func() tea.Msg { return refreshMsg{} }

	case undoCompletedMsg:
		a.loading = false
		a.statusMsg = "Undo successful"
		// Reload current view
		return a, func() tea.Msg { return refreshMsg{} }

	case searchRefreshMsg:
		a.loading = false
		a.statusMsg = "Task updated"
		return a, a.refreshSearchResults()

	case refreshMsg:
		// Store current task ID to restore cursor position after reload
		if len(a.tasks) > 0 && a.taskCursor >= 0 && a.taskCursor < len(a.tasks) {
			a.restoreCursorToTaskID = a.tasks[a.taskCursor].ID
		}
		a.loading = true

		if a.currentView == ViewSearch {
			return a, a.refreshSearchResults()
		}

		switch a.currentTab {
		case TabProjects:
			if a.currentProject != nil {
				// Reload current project
				return a, a.loadProjectTasks(a.currentProject.ID)
			}
			// Just reload project list
			return a, a.loadProjects()
		case TabUpcoming:
			return a, a.loadUpcomingTasks()
		case TabCalendar:
			// Preserve calendar date and reload all tasks
			return a, a.loadAllTasks()
		case TabLabels:
			if a.currentLabel != nil {
				return a, a.loadLabelTasks(a.currentLabel.Name)
			}
			return a, a.loadLabels()
		default:
			// TabToday or fallback
			return a, a.loadTodayTasks()
		}

	case commentsLoadedMsg:
		a.comments = msg.comments
		return a, nil

	case reorderCompleteMsg:
		if a.currentProject != nil {
			return a, a.loadProjectTasks(a.currentProject.ID)
		}
		return a, nil
	}

	return a, nil
}

// handleMouseMsg processes mouse input.
