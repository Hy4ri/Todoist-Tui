// Package utils provides shared utility functions for the TUI.
package utils

import (
	"sort"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
)

// TruncateString truncates a string to a given width and adds an ellipsis if truncated.
func TruncateString(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}

	if width <= 1 {
		return "…"
	}

	res := s
	for lipgloss.Width(res+"…") > width && len(res) > 0 {
		_, size := utf8.DecodeLastRuneInString(res)
		res = res[:len(res)-size]
	}
	return res + "…"
}

// ExtractLabelsFromTasks extracts unique labels from a list of tasks.
// Returns sorted labels by name.
func ExtractLabelsFromTasks(tasks []api.Task) []api.Label {
	labelSet := make(map[string]bool)
	var labels []api.Label

	for _, t := range tasks {
		for _, labelName := range t.Labels {
			if !labelSet[labelName] {
				labelSet[labelName] = true
				labels = append(labels, api.Label{Name: labelName})
			}
		}
	}

	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Name < labels[j].Name
	})

	return labels
}
