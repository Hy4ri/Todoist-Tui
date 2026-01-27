package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
)

func (a *App) sortTasks() {
	sort.Slice(a.tasks, func(i, j int) bool {
		ti, tj := a.tasks[i], a.tasks[j]

		// Get due dates (tasks without due dates sort to end)
		hasDueI := ti.Due != nil
		hasDueJ := tj.Due != nil

		if hasDueI && !hasDueJ {
			return true // i has due date, j doesn't -> i first
		}
		if !hasDueI && hasDueJ {
			return false // j has due date, i doesn't -> j first
		}

		// Both have due dates - compare by datetime/date
		if hasDueI && hasDueJ {
			// Use datetime if available, else use date
			dateI := ti.Due.Date
			dateJ := tj.Due.Date
			if ti.Due.Datetime != nil {
				dateI = *ti.Due.Datetime
			}
			if tj.Due.Datetime != nil {
				dateJ = *tj.Due.Datetime
			}

			if dateI != dateJ {
				return dateI < dateJ // Earlier dates first
			}
		}

		// Same due date or both no due date - sort by priority (higher = P1 = 4)
		return ti.Priority > tj.Priority
	})
}

// handleSelect handles the Enter key.
func (a *App) handleComplete() (tea.Model, tea.Cmd) {
	// In Projects tab, only allow in main pane
	if a.currentTab == TabProjects && a.focusedPane != PaneMain {
		return a, nil
	}

	if len(a.tasks) == 0 {
		return a, nil
	}

	// In Labels view without a selected label, we're showing label list
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	// If there are selected tasks, complete/uncomplete all of them
	if len(a.selectedTaskIDs) > 0 {
		a.loading = true
		tasksToComplete := make([]api.Task, 0)
		for _, task := range a.tasks {
			if a.selectedTaskIDs[task.ID] {
				tasksToComplete = append(tasksToComplete, task)
			}
		}

		return a, func() tea.Msg {
			// Use channels and goroutines for concurrent API calls
			type result struct {
				success bool
			}

			results := make(chan result, len(tasksToComplete))

			// Launch concurrent API calls
			for _, task := range tasksToComplete {
				go func(t api.Task) {
					var err error
					if t.Checked {
						err = a.client.ReopenTask(t.ID)
					} else {
						err = a.client.CloseTask(t.ID)
					}
					results <- result{success: err == nil}
				}(task)
			}

			// Collect results
			completed := 0
			failed := 0
			for i := 0; i < len(tasksToComplete); i++ {
				res := <-results
				if res.success {
					completed++
				} else {
					failed++
				}
			}

			if failed > 0 {
				return statusMsg{msg: fmt.Sprintf("Completed %d tasks, %d failed", completed, failed)}
			}
			return statusMsg{msg: fmt.Sprintf("Completed %d tasks", completed)}
		}
	}

	// Get the correct task using ordered indices mapping
	var task *api.Task
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex := a.taskOrderedIndices[a.taskCursor]
		// Skip empty section headers (negative indices <= -100)
		if taskIndex >= 0 && taskIndex < len(a.tasks) {
			task = &a.tasks[taskIndex]
		}
	} else if a.taskCursor < len(a.tasks) {
		// Fallback for views that don't use ordered indices
		task = &a.tasks[a.taskCursor]
	}

	if task == nil {
		return a, nil
	}

	// Store last action for undo
	if task.Checked {
		a.lastAction = &LastAction{Type: "uncomplete", TaskID: task.ID}
	} else {
		a.lastAction = &LastAction{Type: "complete", TaskID: task.ID}
	}

	a.loading = true

	return a, func() tea.Msg {
		var err error
		if task.Checked {
			err = a.client.ReopenTask(task.ID)
		} else {
			err = a.client.CloseTask(task.ID)
		}
		if err != nil {
			return errMsg{err}
		}
		return taskCompletedMsg{id: task.ID}
	}
}

// handleUndo reverses the last undoable action.
func (a *App) handleUndo() (tea.Model, tea.Cmd) {
	if a.lastAction == nil {
		a.statusMsg = "Nothing to undo"
		return a, nil
	}

	action := a.lastAction
	a.lastAction = nil
	a.loading = true

	return a, func() tea.Msg {
		var err error
		switch action.Type {
		case "complete":
			// Was completed, so reopen it
			err = a.client.ReopenTask(action.TaskID)
		case "uncomplete":
			// Was reopened, so close it
			err = a.client.CloseTask(action.TaskID)
		default:
			return statusMsg{msg: "Unknown action"}
		}
		if err != nil {
			return errMsg{err}
		}
		return undoCompletedMsg{}
	}
}

// handleToggleSelect toggles selection of the task under the cursor.
func (a *App) handleToggleSelect() (tea.Model, tea.Cmd) {
	// Only allow in main pane with tasks
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		return a, nil
	}

	// Don't allow selection in label list view
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	// Get the task at cursor
	var task *api.Task
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex := a.taskOrderedIndices[a.taskCursor]
		// Skip placeholders (negative indices < -100)
		if taskIndex >= 0 && taskIndex < len(a.tasks) {
			task = &a.tasks[taskIndex]
		}
	} else if a.taskCursor < len(a.tasks) {
		task = &a.tasks[a.taskCursor]
	}

	if task == nil {
		return a, nil
	}

	// Toggle selection
	if a.selectedTaskIDs[task.ID] {
		delete(a.selectedTaskIDs, task.ID)
		a.statusMsg = fmt.Sprintf("Deselected (%d selected)", len(a.selectedTaskIDs))
	} else {
		a.selectedTaskIDs[task.ID] = true
		a.statusMsg = fmt.Sprintf("Selected (%d tasks)", len(a.selectedTaskIDs))
	}

	return a, nil
}

// handleCopy copies task content to clipboard.
func (a *App) handleCopy() (tea.Model, tea.Cmd) {
	// Only allow in main pane with tasks
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		return a, nil
	}

	// Don't allow in label list view
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	// If there are selected tasks, copy all of them
	if len(a.selectedTaskIDs) > 0 {
		var selectedContents []string
		for _, task := range a.tasks {
			if a.selectedTaskIDs[task.ID] {
				selectedContents = append(selectedContents, task.Content)
			}
		}

		if len(selectedContents) == 0 {
			return a, nil
		}

		// Clear selections after copy
		a.selectedTaskIDs = make(map[string]bool)

		return a, func() tea.Msg {
			// Join all selected task contents with newlines
			content := strings.Join(selectedContents, "\n")
			err := clipboard.WriteAll(content)
			if err != nil {
				return statusMsg{msg: "Failed to copy: " + err.Error()}
			}
			return statusMsg{msg: fmt.Sprintf("Copied %d tasks", len(selectedContents))}
		}
	}

	// Otherwise, copy just the task at cursor
	var task *api.Task
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex := a.taskOrderedIndices[a.taskCursor]
		if taskIndex >= 0 && taskIndex < len(a.tasks) {
			task = &a.tasks[taskIndex]
		}
	} else if a.taskCursor < len(a.tasks) {
		task = &a.tasks[a.taskCursor]
	}

	if task == nil {
		return a, nil
	}

	return a, func() tea.Msg {
		// Copy to clipboard
		err := clipboard.WriteAll(task.Content)
		if err != nil {
			return statusMsg{msg: "Failed to copy: " + err.Error()}
		}
		return statusMsg{msg: "Copied: " + task.Content}
	}
}

// handleDelete deletes the selected task or project.
func (a *App) handleDelete() (tea.Model, tea.Cmd) {
	// Handle project deletion when sidebar is focused
	if a.currentTab == TabProjects && a.focusedPane == PaneSidebar {
		if a.sidebarCursor >= len(a.sidebarItems) {
			return a, nil
		}
		item := a.sidebarItems[a.sidebarCursor]
		if item.Type != "project" {
			return a, nil
		}
		// Find the project
		for i := range a.projects {
			if a.projects[i].ID == item.ID {
				// Don't allow deleting inbox
				if a.projects[i].InboxProject {
					a.statusMsg = "Cannot delete Inbox project"
					return a, nil
				}
				a.editingProject = &a.projects[i]
				a.confirmDeleteProject = true
				return a, nil
			}
		}
		return a, nil
	}

	// Handle task deletion
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		// Handle label deletion when viewing label list
		if a.currentTab == TabLabels && a.currentLabel == nil {
			if a.taskCursor < len(a.labels) {
				a.editingLabel = &a.labels[a.taskCursor]
				a.confirmDeleteLabel = true
				return a, nil
			}
		}
		return a, nil
	}

	// Guard: Don't delete task when viewing label list (no task selected)
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	// If there are selected tasks, delete all of them
	if len(a.selectedTaskIDs) > 0 {
		a.loading = true
		tasksToDelete := make([]api.Task, 0)
		for _, task := range a.tasks {
			if a.selectedTaskIDs[task.ID] {
				tasksToDelete = append(tasksToDelete, task)
			}
		}

		return a, func() tea.Msg {
			// Use channels and goroutines for concurrent API calls
			type result struct {
				success bool
			}

			results := make(chan result, len(tasksToDelete))

			// Launch concurrent API calls
			for _, task := range tasksToDelete {
				go func(t api.Task) {
					err := a.client.DeleteTask(t.ID)
					results <- result{success: err == nil}
				}(task)
			}

			// Collect results
			deleted := 0
			failed := 0
			for i := 0; i < len(tasksToDelete); i++ {
				res := <-results
				if res.success {
					deleted++
				} else {
					failed++
				}
			}

			if failed > 0 {
				return statusMsg{msg: fmt.Sprintf("Deleted %d tasks, %d failed", deleted, failed)}
			}
			return statusMsg{msg: fmt.Sprintf("Deleted %d tasks", deleted)}
		}
	}

	// Use ordered indices if available
	taskIndex := a.taskCursor
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex = a.taskOrderedIndices[a.taskCursor]
	}
	if taskIndex < 0 || taskIndex >= len(a.tasks) {
		return a, nil
	}

	task := &a.tasks[taskIndex]
	a.loading = true

	return a, func() tea.Msg {
		err := a.client.DeleteTask(task.ID)
		if err != nil {
			return errMsg{err}
		}
		return taskDeletedMsg{id: task.ID}
	}
}

// handleAdd opens the add task form.
func (a *App) handleAdd() (tea.Model, tea.Cmd) {
	a.previousView = a.currentView
	a.currentView = ViewTaskForm
	a.taskForm = NewTaskForm(a.projects, a.labels)
	a.taskForm.SetWidth(a.width)

	// If in project view, default to that project
	if a.currentProject != nil {
		a.taskForm.ProjectID = a.currentProject.ID
		a.taskForm.ProjectName = a.currentProject.Name

		// Try to detect current section based on cursor position
		// First, check if we can map cursor to a viewport line
		cursorViewportLine := -1
		if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
			taskIndex := a.taskOrderedIndices[a.taskCursor]
			// Find which viewport line this task is on
			for i, vTaskIdx := range a.viewportLines {
				if vTaskIdx == taskIndex {
					cursorViewportLine = i
					break
				}
			}
		}

		// Check if cursor is on an empty section header (taskIndex <= -100)
		if cursorViewportLine >= 0 && cursorViewportLine < len(a.viewportLines) {
			taskIdx := a.viewportLines[cursorViewportLine]
			if taskIdx <= -100 {
				// Cursor is on an empty section header, get the section ID
				if cursorViewportLine < len(a.viewportSections) {
					sectionID := a.viewportSections[cursorViewportLine]
					if sectionID != "" {
						a.taskForm.SectionID = sectionID
						// Find section name
						for _, s := range a.sections {
							if s.ID == sectionID {
								a.taskForm.SectionName = s.Name
								break
							}
						}
					}
				}
			} else if taskIdx >= 0 {
				// Cursor is on a task, use its section
				if taskIdx < len(a.tasks) {
					task := a.tasks[taskIdx]
					if task.SectionID != nil && *task.SectionID != "" {
						a.taskForm.SectionID = *task.SectionID
						// Find section name
						for _, s := range a.sections {
							if s.ID == *task.SectionID {
								a.taskForm.SectionName = s.Name
								break
							}
						}
					}
				}
			}
		} else {
			// Fallback: Use the old method
			if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
				taskIndex := a.taskOrderedIndices[a.taskCursor]
				if taskIndex >= 0 && taskIndex < len(a.tasks) {
					task := a.tasks[taskIndex]
					if task.SectionID != nil && *task.SectionID != "" {
						a.taskForm.SectionID = *task.SectionID
						// Find section name
						for _, s := range a.sections {
							if s.ID == *task.SectionID {
								a.taskForm.SectionName = s.Name
								break
							}
						}
					}
				}
			}
		}
	}

	// Set default due date based on view context
	switch a.previousView {
	case ViewToday:
		a.taskForm.SetDue(time.Now().Format("2006-01-02"))
		a.taskForm.SetContext("Today")
	case ViewUpcoming:
		a.taskForm.SetDue(time.Now().Format("2006-01-02"))
		a.taskForm.SetContext("Upcoming")
	case ViewCalendar, ViewCalendarDay:
		selectedDate := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), a.calendarDay, 0, 0, 0, 0, time.Local)
		a.taskForm.SetDue(selectedDate.Format("2006-01-02"))
		a.taskForm.SetContext(selectedDate.Format("Jan 2"))
	case ViewProject:
		a.taskForm.SetDue("")
		a.taskForm.SetContext("Project")
	case ViewLabels:
		a.taskForm.SetDue("")
		a.taskForm.SetContext("Labels")
	default:
		a.taskForm.SetDue("")
		a.taskForm.SetContext("")
	}

	return a, nil
}

// handleEdit opens the edit task form for the selected task, or edit project dialog.
func (a *App) handleEdit() (tea.Model, tea.Cmd) {
	// Handle project editing when sidebar is focused
	if a.currentTab == TabProjects && a.focusedPane == PaneSidebar {
		if a.sidebarCursor >= len(a.sidebarItems) {
			return a, nil
		}
		item := a.sidebarItems[a.sidebarCursor]
		if item.Type != "project" {
			return a, nil
		}
		// Find the project to edit
		for i := range a.projects {
			if a.projects[i].ID == item.ID {
				a.editingProject = &a.projects[i]
				a.projectInput = textinput.New()
				a.projectInput.SetValue(a.projects[i].Name)
				a.projectInput.CharLimit = 100
				a.projectInput.Width = 40
				a.projectInput.Focus()
				a.isEditingProject = true
				return a, nil
			}
		}
		return a, nil
	}

	// Handle task editing
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		// Handle label editing when viewing label list
		if a.currentTab == TabLabels && a.currentLabel == nil {
			if a.taskCursor < len(a.labels) {
				a.editingLabel = &a.labels[a.taskCursor]
				a.labelInput = textinput.New()
				a.labelInput.SetValue(a.labels[a.taskCursor].Name)
				a.labelInput.CharLimit = 100
				a.labelInput.Width = 40
				a.labelInput.Focus()
				a.isEditingLabel = true
				return a, nil
			}
		}
		return a, nil
	}

	// Guard: Don't edit task when viewing label list (no task selected)
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	// Use ordered indices if available
	taskIndex := a.taskCursor
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex = a.taskOrderedIndices[a.taskCursor]
	}
	if taskIndex < 0 || taskIndex >= len(a.tasks) {
		return a, nil
	}

	task := &a.tasks[taskIndex]
	a.previousView = a.currentView
	a.currentView = ViewTaskForm
	a.taskForm = NewEditTaskForm(task, a.projects, a.labels)
	a.taskForm.SetWidth(a.width)

	return a, nil
}

// handleNewProject opens the project creation input.
