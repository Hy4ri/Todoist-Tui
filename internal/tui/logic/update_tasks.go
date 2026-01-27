package logic

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hy4ri/todoist-tui/internal/tui/state"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
)

func (h *Handler) sortTasks() {
	sort.Slice(h.Tasks, func(i, j int) bool {
		ti, tj := h.Tasks[i], h.Tasks[j]

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
func (h *Handler) handleComplete() tea.Cmd {
	// In Projects tab, only allow in main pane
	if h.CurrentTab == state.TabProjects && h.FocusedPane != state.PaneMain {
		return nil
	}

	if len(h.Tasks) == 0 {
		return nil
	}

	// In Labels view without a selected label, we're showing label list
	if h.CurrentView == state.ViewLabels && h.CurrentLabel == nil {
		return nil
	}

	// If there are selected tasks, complete/uncomplete all of them
	if len(h.SelectedTaskIDs) > 0 {
		h.Loading = true
		tasksToComplete := make([]api.Task, 0)
		for _, task := range h.Tasks {
			if h.SelectedTaskIDs[task.ID] {
				tasksToComplete = append(tasksToComplete, task)
			}
		}

		return func() tea.Msg {
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
						err = h.Client.ReopenTask(t.ID)
					} else {
						err = h.Client.CloseTask(t.ID)
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
	if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
		taskIndex := h.TaskOrderedIndices[h.TaskCursor]
		// Skip empty section headers (negative indices <= -100)
		if taskIndex >= 0 && taskIndex < len(h.Tasks) {
			task = &h.Tasks[taskIndex]
		}
	} else if h.TaskCursor < len(h.Tasks) {
		// Fallback for views that don't use ordered indices
		task = &h.Tasks[h.TaskCursor]
	}

	if task == nil {
		return nil
	}

	// Store last action for undo
	if task.Checked {
		h.State.LastAction = &state.LastAction{Type: "uncomplete", TaskID: task.ID}
	} else {
		h.State.LastAction = &state.LastAction{Type: "complete", TaskID: task.ID}
	}

	h.Loading = true

	return func() tea.Msg {
		var err error
		if task.Checked {
			err = h.Client.ReopenTask(task.ID)
		} else {
			err = h.Client.CloseTask(task.ID)
		}
		if err != nil {
			return errMsg{err}
		}
		return taskCompletedMsg{id: task.ID}
	}
}

// handleUndo reverses the last undoable action.
func (h *Handler) handleUndo() tea.Cmd {
	if h.State.LastAction == nil {
		h.StatusMsg = "Nothing to undo"
		return nil
	}

	action := h.State.LastAction
	h.State.LastAction = nil
	h.Loading = true

	return func() tea.Msg {
		var err error
		switch action.Type {
		case "complete":
			// Was completed, so reopen it
			err = h.Client.ReopenTask(action.TaskID)
		case "uncomplete":
			// Was reopened, so close it
			err = h.Client.CloseTask(action.TaskID)
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
func (h *Handler) handleToggleSelect() tea.Cmd {
	// Only allow in main pane with tasks
	if h.FocusedPane != state.PaneMain || len(h.Tasks) == 0 {
		return nil
	}

	// Don't allow selection in label list view
	if h.CurrentView == state.ViewLabels && h.CurrentLabel == nil {
		return nil
	}

	// Get the task at cursor
	var task *api.Task
	if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
		taskIndex := h.TaskOrderedIndices[h.TaskCursor]
		// Skip placeholders (negative indices < -100)
		if taskIndex >= 0 && taskIndex < len(h.Tasks) {
			task = &h.Tasks[taskIndex]
		}
	} else if h.TaskCursor < len(h.Tasks) {
		task = &h.Tasks[h.TaskCursor]
	}

	if task == nil {
		return nil
	}

	// Toggle selection
	if h.SelectedTaskIDs[task.ID] {
		delete(h.SelectedTaskIDs, task.ID)
		h.StatusMsg = fmt.Sprintf("Deselected (%d selected)", len(h.SelectedTaskIDs))
	} else {
		h.SelectedTaskIDs[task.ID] = true
		h.StatusMsg = fmt.Sprintf("Selected (%d tasks)", len(h.SelectedTaskIDs))
	}

	return nil
}

// handleCopy copies task content to clipboard.
func (h *Handler) handleCopy() tea.Cmd {
	// Only allow in main pane with tasks
	if h.FocusedPane != state.PaneMain || len(h.Tasks) == 0 {
		return nil
	}

	// Don't allow in label list view
	if h.CurrentView == state.ViewLabels && h.CurrentLabel == nil {
		return nil
	}

	// If there are selected tasks, copy all of them
	if len(h.SelectedTaskIDs) > 0 {
		var selectedContents []string
		for _, task := range h.Tasks {
			if h.SelectedTaskIDs[task.ID] {
				selectedContents = append(selectedContents, task.Content)
			}
		}

		if len(selectedContents) > 0 {
			// Clear selections after copy
			h.SelectedTaskIDs = make(map[string]bool)

			return func() tea.Msg {
				// Join all selected task contents with newlines
				content := strings.Join(selectedContents, "\n")
				err := clipboard.WriteAll(content)
				if err != nil {
					return statusMsg{msg: "Failed to copy: " + err.Error()}
				}
				return statusMsg{msg: fmt.Sprintf("Copied %d tasks", len(selectedContents))}
			}
		}
	}

	// Determine what's under the cursor
	if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
		taskIndex := h.TaskOrderedIndices[h.TaskCursor]

		// 1. Check if it's a section header
		if taskIndex <= -100 {
			// Find viewport line to get section ID
			sectionID := ""
			for i, vIdx := range h.State.ViewportLines {
				if vIdx == taskIndex {
					if i < len(h.State.ViewportSections) {
						sectionID = h.State.ViewportSections[i]
					}
					break
				}
			}

			if sectionID != "" {
				// Get section name
				sectionName := "Section"
				for _, s := range h.Sections {
					if s.ID == sectionID {
						sectionName = s.Name
						break
					}
				}

				// Get tasks in this section
				var tasksInSection []string
				for _, t := range h.Tasks {
					if t.SectionID != nil && *t.SectionID == sectionID {
						tasksInSection = append(tasksInSection, "- "+t.Content)
					}
				}

				return func() tea.Msg {
					content := sectionName
					if len(tasksInSection) > 0 {
						content += "\n" + strings.Join(tasksInSection, "\n")
					}
					err := clipboard.WriteAll(content)
					if err != nil {
						return statusMsg{msg: "Failed to copy section: " + err.Error()}
					}
					return statusMsg{msg: "Copied section: " + sectionName}
				}
			}
		}

		// 2. Check if it's a normal task
		if taskIndex >= 0 && taskIndex < len(h.Tasks) {
			task := &h.Tasks[taskIndex]
			return func() tea.Msg {
				err := clipboard.WriteAll(task.Content)
				if err != nil {
					return statusMsg{msg: "Failed to copy: " + err.Error()}
				}
				return statusMsg{msg: "Copied: " + task.Content}
			}
		}
	} else if h.TaskCursor < len(h.Tasks) {
		// Fallback for views that don't use ordered indices
		task := &h.Tasks[h.TaskCursor]
		return func() tea.Msg {
			err := clipboard.WriteAll(task.Content)
			if err != nil {
				return statusMsg{msg: "Failed to copy: " + err.Error()}
			}
			return statusMsg{msg: "Copied: " + task.Content}
		}
	}

	return nil
}

// handleDelete deletes the selected task or project.
func (h *Handler) handleDelete() tea.Cmd {
	// Handle project deletion when sidebar is focused
	if h.CurrentTab == state.TabProjects && h.FocusedPane == state.PaneSidebar {
		if h.SidebarCursor >= len(h.SidebarItems) {
			return nil
		}
		item := h.SidebarItems[h.SidebarCursor]
		if item.Type != "project" {
			return nil
		}
		// Find the project
		for i := range h.Projects {
			if h.Projects[i].ID == item.ID {
				// Don't allow deleting inbox
				if h.Projects[i].InboxProject {
					h.StatusMsg = "Cannot delete Inbox project"
					return nil
				}
				h.EditingProject = &h.Projects[i]
				h.ConfirmDeleteProject = true
				return nil
			}
		}
		return nil
	}

	// Handle task deletion
	if h.FocusedPane != state.PaneMain || len(h.Tasks) == 0 {
		// Handle label deletion when viewing label list
		if h.CurrentTab == state.TabLabels && h.CurrentLabel == nil {
			if h.TaskCursor < len(h.Labels) {
				h.EditingLabel = &h.Labels[h.TaskCursor]
				h.ConfirmDeleteLabel = true
				return nil
			}
		}
		return nil
	}

	// Guard: Don't delete task when viewing label list (no task selected)
	if h.CurrentView == state.ViewLabels && h.CurrentLabel == nil {
		return nil
	}

	// If there are selected tasks, delete all of them
	if len(h.SelectedTaskIDs) > 0 {
		h.Loading = true
		tasksToDelete := make([]api.Task, 0)
		for _, task := range h.Tasks {
			if h.SelectedTaskIDs[task.ID] {
				tasksToDelete = append(tasksToDelete, task)
			}
		}

		return func() tea.Msg {
			// Use channels and goroutines for concurrent API calls
			type result struct {
				success bool
			}

			results := make(chan result, len(tasksToDelete))

			// Launch concurrent API calls
			for _, task := range tasksToDelete {
				go func(t api.Task) {
					err := h.Client.DeleteTask(t.ID)
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
	taskIndex := h.TaskCursor
	if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
		taskIndex = h.TaskOrderedIndices[h.TaskCursor]
	}
	if taskIndex < 0 || taskIndex >= len(h.Tasks) {
		return nil
	}

	task := &h.Tasks[taskIndex]
	h.Loading = true

	return func() tea.Msg {
		err := h.Client.DeleteTask(task.ID)
		if err != nil {
			return errMsg{err}
		}
		return taskDeletedMsg{id: task.ID}
	}
}

// handleAdd opens the add task form.
func (h *Handler) handleAdd() tea.Cmd {
	h.PreviousView = h.CurrentView
	h.CurrentView = state.ViewTaskForm
	h.TaskForm = state.NewTaskForm(h.Projects, h.Labels)
	h.TaskForm.SetWidth(h.Width)

	// If in project view, default to that project
	if h.CurrentProject != nil {
		h.TaskForm.ProjectID = h.CurrentProject.ID
		h.TaskForm.ProjectName = h.CurrentProject.Name

		// Try to detect current section based on cursor position
		// First, check if we can map cursor to a viewport line
		cursorViewportLine := -1
		if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
			taskIndex := h.TaskOrderedIndices[h.TaskCursor]
			// Find which viewport line this task is on
			for i, vTaskIdx := range h.State.ViewportLines {
				if vTaskIdx == taskIndex {
					cursorViewportLine = i
					break
				}
			}
		}

		// Check if cursor is on an empty section header (taskIndex <= -100)
		if cursorViewportLine >= 0 && cursorViewportLine < len(h.State.ViewportLines) {
			taskIdx := h.State.ViewportLines[cursorViewportLine]
			if taskIdx <= -100 {
				// Cursor is on an empty section header, get the section ID
				if cursorViewportLine < len(h.State.ViewportSections) {
					sectionID := h.State.ViewportSections[cursorViewportLine]
					if sectionID != "" {
						h.TaskForm.SectionID = sectionID
						// Find section name
						for _, s := range h.Sections {
							if s.ID == sectionID {
								h.TaskForm.SectionName = s.Name
								break
							}
						}
					}
				}
			} else if taskIdx >= 0 {
				// Cursor is on a task, use its section
				if taskIdx < len(h.Tasks) {
					task := h.Tasks[taskIdx]
					if task.SectionID != nil && *task.SectionID != "" {
						h.TaskForm.SectionID = *task.SectionID
						// Find section name
						for _, s := range h.Sections {
							if s.ID == *task.SectionID {
								h.TaskForm.SectionName = s.Name
								break
							}
						}
					}
				}
			}
		} else {
			// Fallback: Use the old method
			if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
				taskIndex := h.TaskOrderedIndices[h.TaskCursor]
				if taskIndex >= 0 && taskIndex < len(h.Tasks) {
					task := h.Tasks[taskIndex]
					if task.SectionID != nil && *task.SectionID != "" {
						h.TaskForm.SectionID = *task.SectionID
						// Find section name
						for _, s := range h.Sections {
							if s.ID == *task.SectionID {
								h.TaskForm.SectionName = s.Name
								break
							}
						}
					}
				}
			}
		}
	}

	// Set default due date based on view context
	switch h.PreviousView {
	case state.ViewToday:
		h.TaskForm.SetDue(time.Now().Format("2006-01-02"))
		h.TaskForm.SetContext("Today")
	case state.ViewUpcoming:
		h.TaskForm.SetDue(time.Now().Format("2006-01-02"))
		h.TaskForm.SetContext("Upcoming")
	case state.ViewCalendar, state.ViewCalendarDay:
		selectedDate := time.Date(h.CalendarDate.Year(), h.CalendarDate.Month(), h.CalendarDay, 0, 0, 0, 0, time.Local)
		h.TaskForm.SetDue(selectedDate.Format("2006-01-02"))
		h.TaskForm.SetContext(selectedDate.Format("Jan 2"))
	case state.ViewProject:
		h.TaskForm.SetDue("")
		h.TaskForm.SetContext("Project")
	case state.ViewLabels:
		h.TaskForm.SetDue("")
		h.TaskForm.SetContext("Labels")
	default:
		h.TaskForm.SetDue("")
		h.TaskForm.SetContext("")
	}

	return nil
}

// handleEdit opens the edit task form for the selected task, or edit project dialog.
func (h *Handler) handleEdit() tea.Cmd {
	// Handle project editing when sidebar is focused
	if h.CurrentTab == state.TabProjects && h.FocusedPane == state.PaneSidebar {
		if h.SidebarCursor >= len(h.SidebarItems) {
			return nil
		}
		item := h.SidebarItems[h.SidebarCursor]
		if item.Type != "project" {
			return nil
		}
		// Find the project to edit
		for i := range h.Projects {
			if h.Projects[i].ID == item.ID {
				h.EditingProject = &h.Projects[i]
				h.ProjectInput = textinput.New()
				h.ProjectInput.SetValue(h.Projects[i].Name)
				h.ProjectInput.CharLimit = 100
				h.ProjectInput.Width = 40
				h.ProjectInput.Focus()
				h.IsEditingProject = true
				return nil
			}
		}
		return nil
	}

	// Handle task editing
	if h.FocusedPane != state.PaneMain || len(h.Tasks) == 0 {
		// Handle label editing when viewing label list
		if h.CurrentTab == state.TabLabels && h.CurrentLabel == nil {
			if h.TaskCursor < len(h.Labels) {
				h.EditingLabel = &h.Labels[h.TaskCursor]
				h.LabelInput = textinput.New()
				h.LabelInput.SetValue(h.Labels[h.TaskCursor].Name)
				h.LabelInput.CharLimit = 100
				h.LabelInput.Width = 40
				h.LabelInput.Focus()
				h.IsEditingLabel = true
				return nil
			}
		}
		return nil
	}

	// Guard: Don't edit task when viewing label list (no task selected)
	if h.CurrentView == state.ViewLabels && h.CurrentLabel == nil {
		return nil
	}

	// Use ordered indices if available
	taskIndex := h.TaskCursor
	if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
		taskIndex = h.TaskOrderedIndices[h.TaskCursor]
	}
	if taskIndex < 0 || taskIndex >= len(h.Tasks) {
		return nil
	}

	task := &h.Tasks[taskIndex]
	h.PreviousView = h.CurrentView
	h.CurrentView = state.ViewTaskForm
	h.TaskForm = state.NewEditTaskForm(task, h.Projects, h.Labels)
	h.TaskForm.SetWidth(h.Width)

	return nil
}

// handleNewProject opens the project creation input.

// submitForm submits the task form (create or update).
func (h *Handler) submitForm() tea.Cmd {
	if h.TaskForm == nil || !h.TaskForm.IsValid() {
		h.StatusMsg = "Task name is required"
		return nil
	}

	h.Loading = true

	if h.TaskForm.Mode == "edit" {
		// Update existing task
		taskID := h.TaskForm.TaskID
		req := h.TaskForm.ToUpdateRequest()
		return func() tea.Msg {
			_, err := h.Client.UpdateTask(taskID, req)
			if err != nil {
				return errMsg{err}
			}
			return taskCreatedMsg{} // Reuse message type for refresh
		}
	}

	// Create new task
	req := h.TaskForm.ToCreateRequest()
	return func() tea.Msg {
		_, err := h.Client.CreateTask(req)
		if err != nil {
			return errMsg{err}
		}
		return taskCreatedMsg{}
	}
}

// filterInboxTasks filters tasks for the inbox project.
func (h *Handler) filterInboxTasks() tea.Cmd {
	var inboxID string
	for _, p := range h.Projects {
		if p.InboxProject {
			inboxID = p.ID
			break
		}
	}

	if inboxID == "" {
		h.StatusMsg = "Inbox not found"
		return nil
	}

	// Filter tasks by project ID
	var tasks []api.Task
	// Us allTasks if available
	tasksToFilter := h.AllTasks
	if len(tasksToFilter) == 0 {
		tasksToFilter = h.Tasks
	}

	for _, t := range tasksToFilter {
		if t.ProjectID == inboxID && !t.Checked && !t.IsDeleted {
			tasks = append(tasks, t)
		}
	}

	h.Tasks = tasks
	h.sortTasks()
	return nil
}
