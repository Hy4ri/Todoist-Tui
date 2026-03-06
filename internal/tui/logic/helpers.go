package logic

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/utils"
)

// extractLabelsFromTasks extracts unique labels from all tasks using shared util.
func (h *Handler) extractLabelsFromTasks() []api.Label {
	tasksToScan := h.AllTasks
	if len(tasksToScan) == 0 {
		tasksToScan = h.Tasks
	}
	return utils.ExtractLabelsFromTasks(tasksToScan)
}

// reorderSectionsCmd updates the section order using the Sync API.
func (h *Handler) reorderSectionsCmd(sections []api.Section) tea.Cmd {
	return func() tea.Msg {
		if err := h.Client.ReorderSections(sections); err != nil {
			return errMsg{err}
		}
		return reorderCompleteMsg{}
	}
}

// clearSelection clears both multi-select (SelectedTaskIDs) and detail-panel
// selection (SelectedTask) in one call. All bulk operations and navigation
// transitions should use this instead of zeroing the fields individually.
func (h *Handler) clearSelection() {
	h.SelectedTaskIDs = make(map[string]bool)
	h.SelectedTask = nil
}

// getSelectedTask returns the currently selected task in the main list.
func (h *Handler) getSelectedTask() *api.Task {
	if len(h.Tasks) == 0 {
		return nil
	}

	taskIndex := h.TaskCursor
	if len(h.TaskOrderedIndices) > 0 && h.TaskCursor >= 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
		taskIndex = h.TaskOrderedIndices[h.TaskCursor]
	}

	// Skip headers
	if taskIndex <= -100 {
		return nil
	}

	if taskIndex < 0 || taskIndex >= len(h.Tasks) {
		return nil
	}

	return &h.Tasks[taskIndex]
}

// rebuildSidebarCounts rebuilds the sidebar item list and syncs the sidebar
// component with fresh task counts. Call this after any mutation that changes
// task counts (complete, delete, move).
func (h *Handler) rebuildSidebarCounts() {
	h.buildSidebarItems()

	counts := make(map[string]int)
	for _, t := range h.AllTasks {
		if !t.Checked && !t.IsDeleted {
			counts[t.ProjectID]++
		}
	}
	h.SidebarComp.SetProjects(h.Projects, counts)
}

// taskLess reports whether task ti should sort before task tj.
// Ordering: time present → chronological time → priority → due date → ChildOrder.
func taskLess(ti, tj api.Task) bool {
	// 1. Tasks with a specific time come before date-only or no-due tasks.
	hasTimeI := ti.Due != nil && ti.Due.Datetime != nil && *ti.Due.Datetime != ""
	hasTimeJ := tj.Due != nil && tj.Due.Datetime != nil && *tj.Due.Datetime != ""

	if hasTimeI != hasTimeJ {
		return hasTimeI
	}
	if hasTimeI && hasTimeJ {
		if *ti.Due.Datetime != *tj.Due.Datetime {
			return *ti.Due.Datetime < *tj.Due.Datetime
		}
	}

	// 2. Priority (higher value = more urgent).
	if ti.Priority != tj.Priority {
		return ti.Priority > tj.Priority
	}

	// 3. Due date (earlier first) as tiebreaker.
	hasDueI := ti.Due != nil
	hasDueJ := tj.Due != nil

	if hasDueI != hasDueJ {
		return hasDueI
	}
	if hasDueI && hasDueJ {
		if ti.Due.Date != tj.Due.Date {
			return ti.Due.Date < tj.Due.Date
		}
	}

	// 4. Manual order within project/list.
	return ti.ChildOrder < tj.ChildOrder
}
