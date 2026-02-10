package logic

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// handleMoveToProject initializes the move to project view.
func (h *Handler) handleMoveToProject() tea.Cmd {
	// Only valid if tasks are selected or single task is selected
	if len(h.SelectedTaskIDs) == 0 && h.SelectedTask == nil {
		// If no explicit selection, try to use task under cursor
		if len(h.Tasks) > 0 {
			var task *api.Task
			if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
				idx := h.TaskOrderedIndices[h.TaskCursor]
				if idx >= 0 && idx < len(h.Tasks) {
					task = &h.Tasks[idx]
				}
			} else if h.TaskCursor < len(h.Tasks) {
				task = &h.Tasks[h.TaskCursor]
			}
			if task != nil {
				h.SelectedTask = task
			}
		}
	}

	if len(h.SelectedTaskIDs) == 0 && h.SelectedTask == nil {
		h.StatusMsg = "No task selected to move"
		return nil
	}

	h.IsMovingToProject = true
	h.MoveProjectInput = textinput.New()
	h.MoveProjectInput.Placeholder = "Search project..."
	h.MoveProjectInput.Focus()
	h.MoveTargetCursor = 0

	// Build initial list
	h.buildMoveTargetList("")

	return nil
}

// buildMoveTargetList constructs the list of move targets (projects + sections).
func (h *Handler) buildMoveTargetList(query string) {
	var targets []state.MoveTarget
	query = strings.ToLower(strings.TrimSpace(query))

	// Group sections by project
	sectionsByProject := make(map[string][]api.Section)
	for _, s := range h.AllSections {
		sectionsByProject[s.ProjectID] = append(sectionsByProject[s.ProjectID], s)
	}
	// Sort sections by order
	for pID := range sectionsByProject {
		sort.Slice(sectionsByProject[pID], func(i, j int) bool {
			return sectionsByProject[pID][i].SectionOrder < sectionsByProject[pID][j].SectionOrder
		})
	}

	// Helper to add project and its sections
	addProject := func(p api.Project) {
		// Add project itself
		if query == "" || strings.Contains(strings.ToLower(p.Name), query) {
			targets = append(targets, state.MoveTarget{
				ID:        p.ID,
				Name:      p.Name,
				ProjectID: p.ID,
				IsSection: false,
				Indent:    0,
			})
		}

		// Add sections
		for _, s := range sectionsByProject[p.ID] {
			// If query is empty, show all sections
			// If query matches section name OR project name, show section
			match := query == "" || strings.Contains(strings.ToLower(s.Name), query) || strings.Contains(strings.ToLower(p.Name), query)
			if match {
				targets = append(targets, state.MoveTarget{
					ID:        s.ID,
					Name:      s.Name,
					ProjectID: p.ID,
					IsSection: true,
					Indent:    1,
				})
			}
		}
	}

	// 1. Inbox Project (Find it)
	var inboxID string
	for _, p := range h.Projects {
		if p.InboxProject {
			inboxID = p.ID
			addProject(p)
			break
		}
	}

	// 2. Favorite Projects
	for _, p := range h.Projects {
		if p.IsFavorite && p.ID != inboxID {
			addProject(p)
		}
	}

	// 3. Other Projects
	for _, p := range h.Projects {
		if !p.IsFavorite && !p.InboxProject {
			addProject(p)
		}
	}

	h.MoveTargetList = targets
}

// handleMoveToProjectInput handles keyboard input for the move to project view.
func (h *Handler) handleMoveToProjectInput(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsMovingToProject = false
		h.MoveProjectInput.Reset()
		return nil

	case "up", "ctrl+k":
		if h.MoveTargetCursor > 0 {
			h.MoveTargetCursor--
		}
		return nil

	case "down", "ctrl+j":
		if h.MoveTargetCursor < len(h.MoveTargetList)-1 {
			h.MoveTargetCursor++
		}
		return nil

	case "enter":
		if len(h.MoveTargetList) == 0 {
			return nil
		}
		target := h.MoveTargetList[h.MoveTargetCursor]
		return h.executeMoveToProject(target)
	}

	var cmd tea.Cmd
	h.MoveProjectInput, cmd = h.MoveProjectInput.Update(msg)

	// Update list based on filter
	h.buildMoveTargetList(h.MoveProjectInput.Value())
	// Reset cursor if out of bounds
	if h.MoveTargetCursor >= len(h.MoveTargetList) {
		h.MoveTargetCursor = 0
	}

	return cmd
}

// executeMoveToProject performs the move API call.
func (h *Handler) executeMoveToProject(target state.MoveTarget) tea.Cmd {
	h.IsMovingToProject = false
	h.MoveProjectInput.Reset()
	h.StatusMsg = fmt.Sprintf("Moving to %s...", target.Name)

	// Collect tasks to move
	var tasksToMove []api.Task
	if len(h.SelectedTaskIDs) > 0 {
		for _, t := range h.Tasks {
			if h.SelectedTaskIDs[t.ID] {
				tasksToMove = append(tasksToMove, t)
			}
		}
	} else if h.SelectedTask != nil {
		tasksToMove = append(tasksToMove, *h.SelectedTask)
	}

	if len(tasksToMove) == 0 {
		return nil
	}

	// --- Optimistic Update ---
	idsToRemove := make(map[string]bool)
	for _, t := range tasksToMove {
		idsToRemove[t.ID] = true
	}

	// Update AllTasks (Source of Truth)
	for i := range h.AllTasks {
		if idsToRemove[h.AllTasks[i].ID] {
			if target.IsSection {
				targetID := target.ID
				h.AllTasks[i].SectionID = &targetID
				h.AllTasks[i].ProjectID = target.ProjectID
			} else {
				h.AllTasks[i].ProjectID = target.ID
				h.AllTasks[i].SectionID = nil // Move to project (root)
			}
		}
	}

	// Clear Selection
	h.SelectedTaskIDs = make(map[string]bool)
	h.SelectedTask = nil

	// Refresh the current view based on updated AllTasks
	h.handleRefresh(false)

	// Update Sidebar Counts
	h.buildSidebarItems()

	// --- Background API Call ---
	return func() tea.Msg {
		ids := make([]string, len(tasksToMove))
		for i, t := range tasksToMove {
			ids[i] = t.ID
		}

		var targetProjectID, targetSectionID string
		if target.IsSection {
			targetSectionID = target.ID
			targetProjectID = target.ProjectID
		} else {
			targetProjectID = target.ID
		}

		err := h.Client.MoveTasksBatch(ids, targetProjectID, targetSectionID)
		if err != nil {
			// Trigger refresh on failure to ensure state consistency
			return refreshMsg{Force: true}
		}

		// Success - everything already updated optimistically
		return statusMsg{msg: fmt.Sprintf("Moved %d tasks to %s", len(tasksToMove), target.Name)}
	}
}
