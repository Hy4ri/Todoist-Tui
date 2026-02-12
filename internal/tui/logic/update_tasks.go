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

const maxConcurrentRequests = 5

func (h *Handler) sortTasks() {

	// Use hierarchical sorting for Project/Inbox views to respect ChildOrder and Tree structure
	if h.CurrentView == state.ViewProject || h.CurrentView == state.ViewInbox {
		h.sortTasksHierarchically()
	} else {
		sort.SliceStable(h.Tasks, func(i, j int) bool {
			ti, tj := h.Tasks[i], h.Tasks[j]

			// 1. Time is top priority — tasks with time come first
			hasTimeI := ti.Due != nil && ti.Due.Datetime != nil && *ti.Due.Datetime != ""
			hasTimeJ := tj.Due != nil && tj.Due.Datetime != nil && *tj.Due.Datetime != ""

			if hasTimeI && !hasTimeJ {
				return true
			}
			if !hasTimeI && hasTimeJ {
				return false
			}

			// Both have time → sort chronologically
			if hasTimeI && hasTimeJ {
				if *ti.Due.Datetime != *tj.Due.Datetime {
					return *ti.Due.Datetime < *tj.Due.Datetime
				}
			}

			// 2. Priority (higher value = more urgent)
			if ti.Priority != tj.Priority {
				return ti.Priority > tj.Priority
			}

			// 3. Due date (earlier first) as tiebreaker
			hasDueI := ti.Due != nil
			hasDueJ := tj.Due != nil

			if hasDueI && !hasDueJ {
				return true
			}
			if !hasDueI && hasDueJ {
				return false
			}

			if hasDueI && hasDueJ {
				if ti.Due.Date != tj.Due.Date {
					return ti.Due.Date < tj.Due.Date
				}
			}

			// 4. Child Order (Manual order within project/list) as tiebreaker
			return ti.ChildOrder < tj.ChildOrder
		})
	}

	h.TasksSorted = true
}

// sortTasksHierarchically sorts tasks by ChildOrder, respecting the parent-child hierarchy.
func (h *Handler) sortTasksHierarchically() {
	childMap := make(map[string][]*api.Task)
	var roots []*api.Task

	// Build map
	for i := range h.Tasks {
		t := &h.Tasks[i]
		if t.ParentID == nil {
			roots = append(roots, t)
		} else {
			childMap[*t.ParentID] = append(childMap[*t.ParentID], t)
		}
	}

	// Helper to sort a slice of tasks by Time, Priority, then ChildOrder
	sortByOrder := func(tasks []*api.Task) {
		sort.Slice(tasks, func(i, j int) bool {
			ti, tj := tasks[i], tasks[j]

			// 1. Time
			hasTimeI := ti.Due != nil && ti.Due.Datetime != nil && *ti.Due.Datetime != ""
			hasTimeJ := tj.Due != nil && tj.Due.Datetime != nil && *tj.Due.Datetime != ""

			if hasTimeI && !hasTimeJ {
				return true
			}
			if !hasTimeI && hasTimeJ {
				return false
			}

			if hasTimeI && hasTimeJ {
				if *ti.Due.Datetime != *tj.Due.Datetime {
					return *ti.Due.Datetime < *tj.Due.Datetime
				}
			}

			// 2. Priority
			if ti.Priority != tj.Priority {
				return ti.Priority > tj.Priority
			}

			// 3. ChildOrder
			return ti.ChildOrder < tj.ChildOrder
		})
	}

	// Sort roots
	sortByOrder(roots)

	// Flatten tree
	var sorted []api.Task
	var traverse func(t *api.Task)
	traverse = func(t *api.Task) {
		sorted = append(sorted, *t)
		children := childMap[t.ID]
		if len(children) > 0 {
			sortByOrder(children)
			for _, child := range children {
				traverse(child)
			}
		}
	}

	for _, root := range roots {
		traverse(root)
	}

	// Handle orphans (tasks with parentID that wasn't found in current list)
	// This can happen if parent is in another section or not loaded.
	// We should just append them at the end or treat them as roots?
	// The traversal above only visits nodes reachable from roots in existing list.
	// If h.Tasks has 10 items, sorted might have 9 if one is an orphan.
	// Check for missing tasks.

	if len(sorted) < len(h.Tasks) {
		visited := make(map[string]bool)
		for _, t := range sorted {
			visited[t.ID] = true
		}

		var orphans []*api.Task
		for i := range h.Tasks {
			if !visited[h.Tasks[i].ID] {
				orphans = append(orphans, &h.Tasks[i])
			}
		}

		// Sort orphans and append
		sortByOrder(orphans)
		for _, t := range orphans {
			// Append orphan and its children if any (recursive?)
			// Simple approach: just append orphans. If they have children in childMap, they weren't visited either.
			// Better: Treat orphans as roots and traverse.
			traverse(t)
		}
	}

	h.Tasks = sorted
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
		sem := make(chan struct{}, maxConcurrentRequests)

		for _, task := range tasksToComplete {
			sem <- struct{}{}
			go func(t api.Task) {
				defer func() { <-sem }()
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
				// Refresh on failure is handled by returning taskUpdatedMsg below
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
				content := task.Content
				if task.Description != "" {
					content += "\n" + task.Description
				}
				selectedContents = append(selectedContents, content)
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
				content := task.Content
				if task.Description != "" {
					content += "\n" + task.Description
				}
				err := clipboard.WriteAll(content)
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
			content := task.Content
			if task.Description != "" {
				content += "\n" + task.Description
			}
			err := clipboard.WriteAll(content)
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
		sem := make(chan struct{}, maxConcurrentRequests)

		for _, task := range tasksToDelete {
			sem <- struct{}{}
			go func(t api.Task) {
				defer func() { <-sem }()
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

// determineContextFromCursor identifies the project and section based on the current view and cursor position.
func (h *Handler) determineContextFromCursor() (projectID, projectName, sectionID, sectionName string) {
	// 1. Determine Project
	if h.CurrentProject != nil {
		projectID = h.CurrentProject.ID
		projectName = h.CurrentProject.Name
	} else if h.CurrentView == state.ViewInbox {
		for _, p := range h.Projects {
			if p.InboxProject {
				projectID = p.ID
				projectName = p.Name
				break
			}
		}
	}

	// 2. Determine Section from cursor
	if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
		taskIndex := h.TaskOrderedIndices[h.TaskCursor]

		// Check if cursor is on an empty section header
		if taskIndex <= -100 {
			for i, vIdx := range h.State.ViewportLines {
				if vIdx == taskIndex && i < len(h.State.ViewportSections) {
					sectionID = h.State.ViewportSections[i]
					for _, s := range h.Sections {
						if s.ID == sectionID {
							sectionName = s.Name
							break
						}
					}
					break
				}
			}
		} else if taskIndex >= 0 && taskIndex < len(h.Tasks) {
			// Cursor is on a task, use its section
			task := h.Tasks[taskIndex]
			if task.SectionID != nil && *task.SectionID != "" {
				sectionID = *task.SectionID
				for _, s := range h.Sections {
					if s.ID == sectionID {
						sectionName = s.Name
						break
					}
				}
			}
		}
	}
	return
}

// handleAdd opens the quick add popup.
func (h *Handler) handleAdd() tea.Cmd {
	h.PreviousView = h.CurrentView
	h.CurrentView = state.ViewQuickAdd
	h.QuickAddForm = state.NewQuickAddForm()
	h.QuickAddForm.SetWidth(h.Width)

	// Set context from current view
	projectID, projectName, sectionID, sectionName := h.determineContextFromCursor()

	h.QuickAddForm.SetContext(projectID, projectName, sectionID, sectionName)

	// If in Today view, pre-populate "today "
	if h.CurrentTab == state.TabToday {
		h.QuickAddForm.Input.SetValue("today ")
		h.QuickAddForm.Input.SetCursor(6)
	}

	return nil
}

// handleAddTaskFull opens the full task creation form.
func (h *Handler) handleAddTaskFull() tea.Cmd {
	h.PreviousView = h.CurrentView
	h.CurrentView = state.ViewTaskForm

	// Create new form
	h.TaskForm = state.NewTaskForm(h.Projects, h.Labels)
	h.TaskForm.SetWidth(h.Width)
	h.TaskForm.Mode = "create"

	// Set context from current view
	projectID, projectName, sectionID, sectionName := h.determineContextFromCursor()

	if projectID != "" {
		h.TaskForm.ProjectID = projectID
		h.TaskForm.ProjectName = projectName
		h.TaskForm.SectionID = sectionID
		h.TaskForm.SectionName = sectionName
	}

	// Pre-fill date if in Today/Upcoming
	if h.CurrentTab == state.TabToday {
		h.TaskForm.SetDue("today")
	}

	return nil
}

// handleMoveTaskDate moves the task due date by the specified number of days.
// preciseDate is optional: if provided, it sets the date exactly to this YYYY-MM-DD string.
func (h *Handler) handleMoveTaskDate(days int, preciseDate string) tea.Cmd {
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

	var newDateStr string

	if preciseDate != "" {
		newDateStr = preciseDate
	} else {
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
		newDateStr = newDate.Format("2006-01-02")
	}

	// Optimistic update
	if task.Due == nil {
		task.Due = &api.Due{}
	}
	task.Due.Date = newDateStr
	// If recurring, we might need to be careful, but updating date is fine for display
	// Note: We don't update task.Due.String here because we don't know the localized string for arbitrary date easily.
	// But resetting it or leaving it might be ok. Todoist usually updates it.
	// Let's rely on date for display mostly.
	task.Due.Datetime = nil

	// Update AllTasks
	for i := range h.AllTasks {
		if h.AllTasks[i].ID == task.ID {
			if h.AllTasks[i].Due == nil {
				h.AllTasks[i].Due = &api.Due{}
			}
			h.AllTasks[i].Due.Date = newDateStr
			h.AllTasks[i].Due.Datetime = nil
			break
		}
	}

	h.StatusMsg = fmt.Sprintf("Moving task to %s...", newDateStr)
	// Remove blocking loading state

	// Prepare UpdateReq
	var updateReq api.UpdateTaskRequest

	if preciseDate == "remove" {
		noDate := "no date"
		updateReq.DueString = &noDate
		h.StatusMsg = "Removing due date..."

		// Update optimistic
		task.Due = nil
		// Update AllTasks
		for i := range h.AllTasks {
			if h.AllTasks[i].ID == task.ID {
				h.AllTasks[i].Due = nil
				break
			}
		}
	} else {
		updateReq.DueDate = &newDateStr

		// CRITICAL: Preserve recurring status
		// If task is recurring, we MUST explicitly send the recurrence string.
		// Otherwise the API might treat this as a one-off date assignment and convert it to non-recurring.
		if task.Due != nil && task.Due.IsRecurring && task.Due.String != "" {
			recurrence := task.Due.String
			updateReq.DueString = &recurrence
		}
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

// handleReschedule handles smart rescheduling options.
func (h *Handler) handleReschedule(option string) tea.Cmd {
	h.IsRescheduling = false // Close dialog

	switch option {
	case "Today":
		return h.handleMoveTaskDate(0, time.Now().Format("2006-01-02"))
	case "Tomorrow":
		return h.handleMoveTaskDate(0, time.Now().AddDate(0, 0, 1).Format("2006-01-02"))
	case "Next Week (Mon)":
		// Find next Monday
		now := time.Now()
		daysUntilMon := (1 + 7 - int(now.Weekday())) % 7
		if daysUntilMon == 0 {
			daysUntilMon = 7
		}
		return h.handleMoveTaskDate(0, now.AddDate(0, 0, daysUntilMon).Format("2006-01-02"))
	case "Weekend (Sat)":
		// Find next Saturday
		now := time.Now()
		daysUntilSat := (6 + 7 - int(now.Weekday())) % 7
		if daysUntilSat == 0 {
			daysUntilSat = 7
		}
		return h.handleMoveTaskDate(0, now.AddDate(0, 0, daysUntilSat).Format("2006-01-02"))
	case "Postpone (1 day)":
		return h.handleMoveTaskDate(1, "")
	case "No Date":
		// Handle removing due date (pass explicit empty date string)
		// handleMoveTaskDate with empty strings works to remove/update date if we explicitly handle it
		// But handleMoveTaskDate implementation expects "days" offset or "preciseDate".
		// If both are 0/empty, it calculates today.
		// We need a way to say "remove".
		// Let's assume handleMoveTaskDate can handle "remove" as preciseDate.

		return h.handleMoveTaskDate(0, "remove")
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

	if h.Loading {
		return nil
	}

	h.Loading = true
	h.StatusMsg = "Saving task..."

	// Capture data before closing form
	isEdit := h.TaskForm.Mode == "edit"
	var taskID string
	var updateReq api.UpdateTaskRequest
	var createReq api.CreateTaskRequest

	if isEdit {
		taskID = h.TaskForm.TaskID
		updateReq = h.TaskForm.ToUpdateRequest()
	} else {
		createReq = h.TaskForm.ToCreateRequest()
	}

	// Optimistically close form and update UI
	h.CurrentView = h.PreviousView

	// Capture form values for optimistic update
	formDate := h.TaskForm.DueString.Value()
	formTime := h.TaskForm.DueTime.Value()
	formParams := updateReq // Use the request object to get other fields if needed

	if isEdit {
		// Find and update local task
		updateLocalTask := func(t *api.Task) {
			if formParams.Content != nil {
				t.Content = *formParams.Content
			}
			if formParams.Description != nil {
				t.Description = *formParams.Description
			}
			if formParams.Priority != nil {
				t.Priority = *formParams.Priority
			}
			// Update Project/Section
			if h.TaskForm.ProjectID != "" {
				t.ProjectID = h.TaskForm.ProjectID
			}
			if h.TaskForm.SectionID != "" {
				sid := h.TaskForm.SectionID
				t.SectionID = &sid
			}
			// Update Labels
			t.Labels = h.TaskForm.Labels

			// Handle Due Date/Time
			if formDate != "" {
				if t.Due == nil {
					t.Due = &api.Due{}
				}
				t.Due.Date = formDate
				t.Due.String = formDate // temporary

				if formTime != "" {
					// Construct local datetime for immediate display
					// formDate is YYYY-MM-DD, formTime is HH:MM
					// Append local timezone offset so time.Parse(RFC3339) works
					offset := time.Now().Format("-07:00")
					localDT := fmt.Sprintf("%sT%s:00%s", formDate, formTime, offset)
					t.Due.Datetime = &localDT
				} else {
					t.Due.Datetime = nil
				}
			}
		}

		// Update in AllTasks
		for i := range h.AllTasks {
			if h.AllTasks[i].ID == taskID {
				updateLocalTask(&h.AllTasks[i])
				break
			}
		}
		// Update in current Tasks view
		for i := range h.Tasks {
			if h.Tasks[i].ID == taskID {
				updateLocalTask(&h.Tasks[i])
				break
			}
		}

		// Re-sort just in case priority/date changed
		h.sortTasks()

		h.TaskForm = nil
		return func() tea.Msg {
			_, err := h.Client.UpdateTask(taskID, updateReq)
			if err != nil {
				return errMsg{err}
			}
			return taskCreatedMsg{} // Reuse message type for refresh
		}
	}

	// Create new task
	return func() tea.Msg {
		_, err := h.Client.CreateTask(createReq)
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
	// Use allTasks if available
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

// handleSectionAdd opens the specialized section-aware add dialog.
func (h *Handler) handleSectionAdd() tea.Cmd {
	// Guard: Only allow in Project/Inbox views
	if h.CurrentView != state.ViewProject && h.CurrentView != state.ViewInbox {
		return h.handleAdd() // Fallback to general quick add
	}

	// Determine target section from cursor position
	projectID, projectName, sectionID, sectionName := h.determineContextFromCursor()

	// If no section found, fallback to general quick add
	if sectionID == "" {
		return h.handleAdd()
	}

	// Setup specialized addition state
	h.TargetProjectID = projectID
	h.TargetProjectName = projectName
	h.TargetSectionID = sectionID
	h.TargetSectionName = sectionName
	h.SectionAddInput = textinput.New()
	h.SectionAddInput.Placeholder = "Task name (e.g. Buy milk tomorrow #Shopping p1)"
	h.SectionAddInput.Focus()
	h.SectionAddInput.Width = 60
	h.IsAddingToSection = true

	return nil
}

// handleSectionAddInputKeyMsg handles keyboard input for the specialized section add dialog.
func (h *Handler) handleSectionAddInputKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsAddingToSection = false
		h.SectionAddInput.Reset()
		return nil

	case "enter":
		content := strings.TrimSpace(h.SectionAddInput.Value())
		if content == "" {
			h.IsAddingToSection = false
			return nil
		}

		h.IsAddingToSection = false
		h.SectionAddInput.Reset()
		h.StatusMsg = "Adding task..."

		return func() tea.Msg {
			// Send clean text to QuickAddTask for NLP (dates, priorities, labels)
			// Do NOT append #ProjectName (fails with spaces in project names)
			task, err := h.Client.QuickAddTask(content)
			if err != nil {
				return errMsg{err}
			}

			// Move task to correct project/section using the dedicated move endpoint
			targetPID := h.TargetProjectID
			targetSID := h.TargetSectionID

			if targetPID != "" && task.ProjectID != targetPID {
				var secPtr *string
				if targetSID != "" {
					secPtr = &targetSID
				}
				h.Client.MoveTask(task.ID, secPtr, &targetPID, nil)
			} else if targetSID != "" && (task.SectionID == nil || *task.SectionID != targetSID) {
				h.Client.MoveTask(task.ID, &targetSID, nil, nil)
			}

			return quickAddTaskCreatedMsg{}
		}

	default:
		var cmd tea.Cmd
		h.SectionAddInput, cmd = h.SectionAddInput.Update(msg)
		return cmd
	}
}

// loadCompletedTasks fetches completed tasks from the API.
func (h *Handler) loadCompletedTasks() tea.Cmd {
	h.Loading = true
	h.StatusMsg = "Loading completed tasks..."

	return func() tea.Msg {
		params := api.CompletedTaskParams{
			Limit:  h.CompletedLimit,
			Offset: h.CompletedPage * h.CompletedLimit,
		}
		// If limit is 0 (not set), set a default
		if params.Limit == 0 {
			params.Limit = 30
			h.CompletedLimit = 30
		}
		params.AnnotateItems = true // Need content/date

		// "until" defaults to now
		if params.Until == "" {
			params.Until = time.Now().Format(time.RFC3339)
		}
		// "since" defaults to 1 month ago
		if params.Since == "" {
			params.Since = time.Now().AddDate(0, -1, 0).Format(time.RFC3339)
		}

		tasks, err := h.Client.GetCompletedTasks(params)
		if err != nil {
			return errMsg{err}
		}

		return completedTasksLoadedMsg(tasks)
	}
}

// handleIndent indents the selected task (makes it a subtask of the one above).
// handleIndent opens a picker to select a parent task.
func (h *Handler) handleIndent() tea.Cmd {
	// Only allow indent in Project or Inbox view where parent logic is clear
	if h.CurrentView != state.ViewProject && h.CurrentView != state.ViewInbox {
		h.StatusMsg = "Indent only available in Project/Inbox views"
		return nil
	}

	taskIndex := h.TaskCursor
	// Check if cursor is on a valid task
	if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
		idx := h.TaskOrderedIndices[h.TaskCursor]
		if idx >= 0 {
			taskIndex = idx
		} else {
			return nil // On header
		}
	} else if taskIndex >= len(h.Tasks) {
		return nil
	}

	currentTask := h.Tasks[taskIndex]

	// Find candidates: all tasks in current view except current task and its descendants
	// To find descendants, we might need to traverse.
	// Simple approach: Filter out current task. API prevents cycles usually.
	// But let's try to be smart.

	candidates := []api.Task{}
	for _, t := range h.Tasks {
		if t.ID == currentTask.ID {
			continue // Skip self
		}
		// Skip if t is child of currentTask (prevent cycle)
		// We'd need accurate ParentID check.
		// For now, simple filter.
		candidates = append(candidates, t)
	}

	if len(candidates) == 0 {
		h.StatusMsg = "No tasks available to indent under"
		return nil
	}

	h.IsIndentingTask = true
	h.IndentCandidates = candidates
	h.IndentFilteredCandidates = candidates // Initialize with all
	h.IndentCursor = 0

	// Try to default cursor to task immediately above current one in list
	// original index of task above
	if h.TaskCursor > 0 {
		// Find visual predecessor
		// If we use TaskOrderedIndices, we can find the task ID above.
		// Then find it in candidates.

		// This logic is complex because indices map to h.Tasks.
		// Let's just default to 0 or try to be smart if time permits.
		// Simple: 0.
	}

	h.IndentInput = textinput.New()
	h.IndentInput.Placeholder = "Filter tasks..."
	h.IndentInput.Focus()

	return textinput.Blink
}

// handleIndentSelect processes the indentation choice.
func (h *Handler) handleIndentSelect() tea.Cmd {
	if h.IndentCursor < 0 || h.IndentCursor >= len(h.IndentFilteredCandidates) {
		return nil
	}

	parentTask := h.IndentFilteredCandidates[h.IndentCursor]

	// We need the ID of the task we are moving.
	// We assume h.TaskCursor is still valid and points to the same task as when we opened dialog.
	// (Modal blocks other interaction, so yes).

	taskIndex := h.TaskCursor
	if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
		idx := h.TaskOrderedIndices[h.TaskCursor]
		if idx >= 0 {
			taskIndex = idx
		}
	}

	if taskIndex >= len(h.Tasks) {
		h.IsIndentingTask = false
		return nil
	}

	currentTask := h.Tasks[taskIndex]

	// Perform Move
	parentID := parentTask.ID

	// Create pointer
	parentIDPtr := &parentID

	// Optimistic update
	currentTask.ParentID = parentIDPtr
	h.Tasks[taskIndex] = currentTask

	// Clear state
	h.IsIndentingTask = false
	h.IndentCandidates = nil
	h.IndentFilteredCandidates = nil
	h.StatusMsg = fmt.Sprintf("Indented under '%s'", parentTask.Content)

	return func() tea.Msg {
		err := h.Client.MoveTask(currentTask.ID, nil, nil, parentIDPtr)
		if err != nil {
			return errMsg{err}
		}
		return refreshMsg{Force: true}
	}
}

// handleOutdent outdents the selected task (moves it up one level).
func (h *Handler) handleOutdent() tea.Cmd {
	// Only allow outdent in Project or Inbox view
	if h.CurrentView != state.ViewProject && h.CurrentView != state.ViewInbox {
		h.StatusMsg = "Outdent only available in Project/Inbox views"
		return nil
	}

	taskIndex := h.TaskCursor
	// Check if cursor is on valid task
	if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
		idx := h.TaskOrderedIndices[h.TaskCursor]
		if idx >= 0 {
			taskIndex = idx
		} else {
			return nil
		}
	} else if taskIndex >= len(h.Tasks) {
		return nil
	}

	currentTask := h.Tasks[taskIndex]

	// If no parent, already at top level
	if currentTask.ParentID == nil {
		h.StatusMsg = "Task is already at top level"
		return nil
	}

	// Find current parent task to get ITS parent
	var parentTask *api.Task
	currentParentID := *currentTask.ParentID

	// Search in all tasks (since parent might be collapsed or outside current view filter but unlikely in Project/Inbox)
	// We should look in h.Tasks first, then maybe need full list?
	// For Project view, h.Tasks usually contains all project tasks.
	for _, t := range h.Tasks {
		if t.ID == currentParentID {
			pt := t
			parentTask = &pt
			break
		}
	}
	// Fallback to AllTasks if not found in current view
	if parentTask == nil {
		for _, t := range h.State.AllTasks {
			if t.ID == currentParentID {
				pt := t
				parentTask = &pt
				break
			}
		}
	}

	var newParentID *string
	if parentTask != nil {
		newParentID = parentTask.ParentID
	} else {
		// Parent not found? Fallback to root level
		newParentID = nil
	}

	// Optimistic update
	currentTask.ParentID = newParentID
	h.Tasks[taskIndex] = currentTask

	return func() tea.Msg {
		var pid string
		var projectID *string
		if newParentID != nil {
			pid = *newParentID
		} else {
			// To un-parent (move to root), we send the project ID and empty parent ID
			pid = ""
			pID := currentTask.ProjectID
			projectID = &pID
		}

		err := h.Client.MoveTask(currentTask.ID, nil, projectID, &pid)

		if err != nil {
			return errMsg{err}
		}
		return refreshMsg{Force: true}
	}
}

// updateIndentFilter filters the indent candidates based on input.
func (h *Handler) updateIndentFilter() {
	filter := strings.ToLower(h.IndentInput.Value())
	if filter == "" {
		h.IndentFilteredCandidates = h.IndentCandidates
	} else {
		filtered := []api.Task{}
		for _, t := range h.IndentCandidates {
			if strings.Contains(strings.ToLower(t.Content), filter) {
				filtered = append(filtered, t)
			}
		}
		h.IndentFilteredCandidates = filtered
	}
	// Reset cursor
	h.IndentCursor = 0
}

// Reminders Implementation

// loadTaskDetails loads all details for the selected task (comments + reminders).
func (h *Handler) loadTaskDetails() tea.Cmd {
	if h.SelectedTask == nil {
		return nil
	}

	h.initReminderInput()
	h.IsAddingReminder = false
	h.IsEditingReminder = false
	h.EditingReminder = nil

	return tea.Batch(
		h.loadTaskComments(),
		h.fetchReminders(h.SelectedTask.ID),
	)
}

// fetchReminders fetches reminders for a task.
func (h *Handler) fetchReminders(taskID string) tea.Cmd {
	return func() tea.Msg {
		reminders, err := h.Client.GetRemindersForTask(taskID)
		if err != nil {
			return errMsg{err}
		}
		return remindersFetchedMsg{taskID: taskID, reminders: reminders}
	}
}

// handleAddReminder initiates the add reminder flow.
func (h *Handler) handleAddReminder() tea.Cmd {
	h.IsAddingReminder = true
	h.IsEditingReminder = false
	h.EditingReminder = nil
	h.ConfirmDeleteReminder = false // Ensure delete confirmation is off

	// Default to relative, 30 min
	h.ReminderTypeCursor = 0 // Relative
	if h.ReminderMinuteInput.Value() == "" {
		h.ReminderMinuteInput.SetValue("30")
	}
	h.ReminderMinuteInput.Focus()
	h.ReminderDateInput.SetValue("")
	h.ReminderTimeInput.SetValue("")

	return textinput.Blink
}

// initReminderInput initializes the reminder inputs.
func (h *Handler) initReminderInput() {
	h.ReminderMinuteInput = textinput.New()
	h.ReminderMinuteInput.Placeholder = "Minutes before (e.g. 30)"
	h.ReminderMinuteInput.CharLimit = 5
	h.ReminderMinuteInput.Width = 10

	h.ReminderDateInput = textinput.New()
	h.ReminderDateInput.Placeholder = "YYYY-MM-DD"
	h.ReminderDateInput.CharLimit = 10
	h.ReminderDateInput.Width = 12

	h.ReminderTimeInput = textinput.New()
	h.ReminderTimeInput.Placeholder = "HH:MM"
	h.ReminderTimeInput.CharLimit = 5
	h.ReminderTimeInput.Width = 8
}

// handleReminderTypeToggle toggles between relative and absolute reminder types.
func (h *Handler) handleReminderTypeToggle() tea.Cmd {
	h.ReminderTypeCursor = 1 - h.ReminderTypeCursor // Toggle 0/1

	if h.ReminderTypeCursor == 0 { // Relative
		h.ReminderMinuteInput.Focus()
		h.ReminderDateInput.Blur()
		h.ReminderTimeInput.Blur()
	} else { // Absolute
		h.ReminderMinuteInput.Blur()
		h.ReminderDateInput.Focus()
		// h.ReminderTimeInput.Blur() // Start with date focused
	}
	return nil
}

// handleDeleteReminder handles reminder deletion.
func (h *Handler) handleDeleteReminder() tea.Cmd {
	if h.EditingReminder == nil {
		return nil
	}

	id := h.EditingReminder.ID
	h.ConfirmDeleteReminder = false
	h.EditingReminder = nil

	h.StatusMsg = "Deleting reminder..."

	return func() tea.Msg {
		err := h.Client.DeleteReminder(id)
		if err != nil {
			return errMsg{err}
		}
		return reminderDeletedMsg{id: id}
	}
}

// submitReminderForm handles the submission of the reminder form (create/update).
func (h *Handler) submitReminderForm() tea.Cmd {
	if h.SelectedTask == nil {
		return nil
	}

	var req api.CreateReminderRequest
	req.ItemID = h.SelectedTask.ID
	req.Type = "relative"

	if h.ReminderTypeCursor == 1 {
		req.Type = "absolute"
		dateStr := h.ReminderDateInput.Value()
		timeStr := h.ReminderTimeInput.Value()

		if dateStr == "" {
			h.StatusMsg = "Date is required"
			return nil
		}

		// Basic validation/formatting would be good here
		finalDate := dateStr
		if timeStr != "" {
			finalDate = dateStr + "T" + timeStr + ":00" // Simple append
		}

		req.Due = &api.ReminderDue{
			Date: finalDate,
		}
	} else {
		// Relative
		minStr := h.ReminderMinuteInput.Value()
		if minStr == "" {
			h.StatusMsg = "Minutes required"
			return nil
		}
		// Parse int
		var mins int
		fmt.Sscanf(minStr, "%d", &mins)
		req.MinuteOffset = mins
	}

	h.StatusMsg = "Saving reminder..."
	h.Loading = true

	return func() tea.Msg {
		rem, err := h.Client.CreateReminder(req)
		if err != nil {
			return errMsg{err}
		}
		return reminderCreatedMsg{reminder: rem}
	}
}
