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

// FilterTodayTasksWithHierarchy filters a list of tasks for the today view, keeping today's/overdue tasks
// as well as all of their descendants (children, grandchildren, etc.) and all of their ancestors (parents, etc.)
// to preserve the tree structure in the today view.
func FilterTodayTasksWithHierarchy(allTasks []api.Task) []api.Task {
	keep := make(map[string]bool)
	for _, t := range allTasks {
		if t.IsOverdue() || t.IsDueToday() {
			keep[t.ID] = true
		}
	}

	taskMap := make(map[string]*api.Task)
	childMap := make(map[string][]string) // parentID -> childIDs
	for i := range allTasks {
		t := &allTasks[i]
		taskMap[t.ID] = t
		if t.ParentID != nil {
			childMap[*t.ParentID] = append(childMap[*t.ParentID], t.ID)
		}
	}

	// Queue for BFS
	queue := []string{}
	for id := range keep {
		queue = append(queue, id)
	}

	visited := make(map[string]bool)
	for _, id := range queue {
		visited[id] = true
	}

	for len(queue) > 0 {
		currID := queue[0]
		queue = queue[1:]

		currTask := taskMap[currID]
		if currTask == nil {
			continue
		}

		// 1. Add Parent (Ancestor)
		if currTask.ParentID != nil {
			pID := *currTask.ParentID
			if !visited[pID] {
				visited[pID] = true
				keep[pID] = true
				queue = append(queue, pID)
			}
		}

		// 2. Add Children (Descendants)
		for _, childID := range childMap[currID] {
			if !visited[childID] {
				visited[childID] = true
				keep[childID] = true
				queue = append(queue, childID)
			}
		}
	}

	var result []api.Task
	for _, t := range allTasks {
		if keep[t.ID] {
			result = append(result, t)
		}
	}
	return result
}

