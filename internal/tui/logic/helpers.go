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
