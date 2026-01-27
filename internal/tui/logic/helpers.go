package logic

import (
	"github.com/hy4ri/todoist-tui/internal/tui/state"
	"github.com/hy4ri/todoist-tui/internal/api"
	tea "github.com/charmbracelet/bubbletea"
)

// extractLabelsFromTasks extracts unique labels from all tasks.
func (h *Handler) extractLabelsFromTasks() []api.Label {
	labelSet := make(map[string]bool)
	var labels []api.Label

	// Check allTasks first, fall back to tasks
	tasksToScan := h.AllTasks
	if len(tasksToScan) == 0 {
		tasksToScan = h.Tasks
	}

	for _, t := range tasksToScan {
		for _, labelName := range t.Labels {
			if !labelSet[labelName] {
				labelSet[labelName] = true
				labels = append(labels, api.Label{
					Name: labelName,
// reorderSectionsCmd updates the section order using the Sync API.
func (h *Handler) reorderSectionsCmd(sections []api.Section) tea.Cmd {
	return func() tea.Msg {
		if err := h.Client.ReorderSections(sections); err != nil {
			return errMsg{err}
		}
		return reorderCompleteMsg{}
	}
}

// truncateString truncates a string to a given width and adds an ellipsis if truncated.
func truncateString(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}

	// Very basic truncation that handles some ANSI/multi-byte
	// In a real app we'd use a more robust version, but this fits the immediate need.
	if width <= 1 {
		return "…"
	}

	// Fallback to simpler character-at-a-time width check if needed,
	// but lipgloss.Width is usually reliable for measurement.
	res := s
	for lipgloss.Width(res+"…") > width && len(res) > 0 {
		// Remove one character/byte at a time until it fits
		_, size := utf8.DecodeLastRuneInString(res)
		res = res[:len(res)-size]
	}
	return res + "…"
