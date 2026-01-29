package logic

import (
	"fmt"
	"slices"
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

// handleComplete handles the task completion with optimistic updates.
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

	tasksToComplete := make([]api.Task, 0)

	// Identify tasks to complete
	if len(h.SelectedTaskIDs) > 0 {
		for _, task := range h.Tasks {
			if h.SelectedTaskIDs[task.ID] {
				tasksToComplete = append(tasksToComplete, task)
			}
		}
	} else {
		// Get the correct task using ordered indices mapping
		var task *api.Task
		if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
			taskIndex := h.TaskOrderedIndices[h.TaskCursor]
			if taskIndex >= 0 && taskIndex < len(h.Tasks) {
				task = &h.Tasks[taskIndex]
			}
		} else if h.TaskCursor < len(h.Tasks) {
			task = &h.Tasks[h.TaskCursor]
		}

		if task != nil {
			tasksToComplete = append(tasksToComplete, *task)
		}
	}

	if len(tasksToComplete) == 0 {
		return nil
	}

	// Store last action for undo (just the first one for simplicity/legacy support)
	if len(tasksToComplete) == 1 {
		t := tasksToComplete[0]
		if t.Checked {
			h.State.LastAction = &state.LastAction{Type: "uncomplete", TaskID: t.ID}
		} else {
			h.State.LastAction = &state.LastAction{Type: "complete", TaskID: t.ID}
		}
	}

	// --- Optimistic Update ---

	idsToRemove := make(map[string]bool)
	for _, t := range tasksToComplete {
		idsToRemove[t.ID] = true
	}

	// Update AllTasks (Source of Truth)
	h.AllTasks = slices.DeleteFunc(h.AllTasks, func(t api.Task) bool {
		return idsToRemove[t.ID]
	})

	// Update h.Tasks (Current View)
	h.Tasks = slices.DeleteFunc(h.Tasks, func(t api.Task) bool {
		return idsToRemove[t.ID]
	})

	// Clear Selection
	h.SelectedTaskIDs = make(map[string]bool)

	// Adjust Cursor
	if h.TaskCursor >= len(h.Tasks) {
		h.TaskCursor = max(0, len(h.Tasks)-1)
	}

	// Update Sidebar Counts
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

	// UI Feedback
	h.StatusMsg = fmt.Sprintf("Completed %d tasks", len(tasksToComplete))
	// Do NOT set h.Loading = true to keep UI responsive

	// --- Background API Call ---
	return func() tea.Msg {
		// Concurrent processing
		type result struct {
			success bool
			id      string
			err     error
		}
		results := make(chan result, len(tasksToComplete))

		for _, task := range tasksToComplete {
			go func(t api.Task) {
				var err error
				if t.Checked {
					err = h.Client.ReopenTask(t.ID)
				} else {
					err = h.Client.CloseTask(t.ID)
				}
				results <- result{success: err == nil, id: t.ID, err: err}
			}(task)
		}

		// Wait for results
		failedCount := 0
		for i := 0; i < len(tasksToComplete); i++ {
			res := <-results
			if !res.success {
				failedCount++
				// TODO: handle rollback on failure?
			}
		}

		if failedCount > 0 {
			// Trigger a full refresh if something failed to ensure consistency
			// We return a message that triggers refresh in update.go (like taskUpdatedMsg)
			return taskUpdatedMsg{}
		}

		// Success - UI is already updated.
		return taskCompletedMsg{id: tasksToComplete[0].ID}
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

	tasksToDelete := make([]api.Task, 0)

	// Identify tasks to delete
	if len(h.SelectedTaskIDs) > 0 {
		for _, task := range h.Tasks {
			if h.SelectedTaskIDs[task.ID] {
				tasksToDelete = append(tasksToDelete, task)
			}
		}
	} else {
		// Use ordered indices if available
		taskIndex := h.TaskCursor
		if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
			taskIndex = h.TaskOrderedIndices[h.TaskCursor]
		}
		if taskIndex >= 0 && taskIndex < len(h.Tasks) {
			tasksToDelete = append(tasksToDelete, h.Tasks[taskIndex])
		}
	}

	if len(tasksToDelete) == 0 {
		return nil
	}

	// --- Optimistic Update ---

	idsToRemove := make(map[string]bool)
	for _, t := range tasksToDelete {
		idsToRemove[t.ID] = true
	}

	// Update AllTasks
	h.AllTasks = slices.DeleteFunc(h.AllTasks, func(t api.Task) bool {
		return idsToRemove[t.ID]
	})

	// Update h.Tasks
	h.Tasks = slices.DeleteFunc(h.Tasks, func(t api.Task) bool {
		return idsToRemove[t.ID]
	})

	// Clear Selection
	h.SelectedTaskIDs = make(map[string]bool)

	// Adjust Cursor
	if h.TaskCursor >= len(h.Tasks) {
		h.TaskCursor = max(0, len(h.Tasks)-1)
	}

	// Update Sidebar Counts
	h.buildSidebarItems()
	counts := make(map[string]int)
	for _, t := range h.AllTasks {
		if !t.Checked && !t.IsDeleted {
			counts[t.ProjectID]++
		}
	}
	h.SidebarComp.SetProjects(h.Projects, counts)

	// UI Feedback
	h.StatusMsg = fmt.Sprintf("Deleted %d tasks", len(tasksToDelete))

	// --- Background API Call ---
	return func() tea.Msg {
		type result struct {
			success bool
			id      string
			err     error
		}
		results := make(chan result, len(tasksToDelete))

		for _, task := range tasksToDelete {
			go func(t api.Task) {
				err := h.Client.DeleteTask(t.ID)
				results <- result{success: err == nil, id: t.ID, err: err}
			}(task)
		}

		failedCount := 0
		for i := 0; i < len(tasksToDelete); i++ {
			res := <-results
			if !res.success {
				failedCount++
			}
		}

		if failedCount > 0 {
			// Trigger refresh on failure
			return taskUpdatedMsg{}
		}

		return taskDeletedMsg{id: tasksToDelete[0].ID}
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
		if h.CurrentLabel != nil {
			h.TaskForm.Labels = []string{h.CurrentLabel.Name}
			h.TaskForm.SetContext("@" + h.CurrentLabel.Name)
		} else {
			h.TaskForm.SetContext("Labels")
		}
	default:
		h.TaskForm.SetDue("")
		h.TaskForm.SetContext("")
	}

	return nil
}

// handleMoveTaskDate moves the task due date by the specified number of days.
func (h *Handler) handleMoveTaskDate(days int) tea.Cmd {
	var task *api.Task

	// Determine task
	if h.SelectedTask != nil {
		task = h.SelectedTask
	} else if len(h.Tasks) > 0 {
		if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
			idx := h.TaskOrderedIndices[h.TaskCursor]
			if idx >= 0 && idx < len(h.Tasks) {
				task = &h.Tasks[idx]
			}
		} else if h.TaskCursor < len(h.Tasks) {
			task = &h.Tasks[h.TaskCursor]
		}
	}

	if task == nil {
		return nil
	}

	taskID := task.ID
	var currentDateStr string
	if task.Due != nil {
		currentDateStr = task.Due.Date
	}

	// Calculate new date
	var newDate time.Time
	if currentDateStr != "" {
		parsedDate, err := time.Parse("2006-01-02", currentDateStr)
		if err == nil {
			newDate = parsedDate.AddDate(0, 0, days)
		} else {
			// Try parsing as datetime if needed, or fallback to today
			newDate = time.Now().AddDate(0, 0, days)
		}
	} else {
		// No due date? Set to today + days
		newDate = time.Now().AddDate(0, 0, days)
	}

	newDateStr := newDate.Format("2006-01-02")
	h.StatusMsg = fmt.Sprintf("Moving task to %s...", newDateStr)
	h.Loading = true

	// Prepare UpdateReq
	updateReq := api.UpdateTaskRequest{
		DueDate: &newDateStr,
	}

	// If task is recurring, we MUST explicitly send the recurrence string.
	// Otherwise the API might treat this as a one-off date assignment and convert it to non-recurring.
	if task.Due != nil && task.Due.IsRecurring && task.Due.String != "" {
		recurrence := task.Due.String
		updateReq.DueString = &recurrence
	}

	return func() tea.Msg {
		_, err := h.Client.UpdateTask(taskID, updateReq)
		if err != nil {
			return errMsg{err}
		}
		// Refresh tasks
		return taskCreatedMsg{} // Reuse for refresh
	}
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
