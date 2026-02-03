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
