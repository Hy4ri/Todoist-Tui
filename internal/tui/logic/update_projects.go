package logic

import (
	"sort"
	"strings"
	"time"

	"github.com/hy4ri/todoist-tui/internal/tui/state"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/components"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

func (h *Handler) handleNewProject() tea.Cmd {
	// Only allow creating projects in Projects tab
	if h.CurrentTab != state.TabProjects {
		return nil
	}

	// Initialize project input
	h.ProjectInput = textinput.New()
	h.ProjectInput.Placeholder = "Enter project name..."
	h.ProjectInput.CharLimit = 100
	h.ProjectInput.Width = 40
	h.ProjectInput.Focus()
	h.IsCreatingProject = true

	// Pre-populate colors
	if len(h.AvailableColors) == 0 {
		for name := range styles.TodoistColorMap {
			h.AvailableColors = append(h.AvailableColors, name)
		}
		sort.Strings(h.AvailableColors)
	}
	h.ColorCursor = 0

	return nil
}

// handleProjectInputKeyMsg handles keyboard input during project creation.
func (h *Handler) handleProjectInputKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// If choosing color
	if h.IsSelectingColor {
		switch msg.String() {
		case "esc":
			h.IsSelectingColor = false
			h.ProjectInput.Focus()
			return nil
		case "up", "k":
			if h.ColorCursor > 0 {
				h.ColorCursor--
			}
			return nil
		case "down", "j":
			if h.ColorCursor < len(h.AvailableColors)-1 {
				h.ColorCursor++
			}
			return nil
		case "enter":
			// Submit new project with color
			name := strings.TrimSpace(h.ProjectInput.Value())
			color := h.AvailableColors[h.ColorCursor]

			h.IsCreatingProject = false
			h.IsSelectingColor = false
			h.ProjectInput.Reset()
			h.Loading = true

			return func() tea.Msg {
				project, err := h.Client.CreateProject(api.CreateProjectRequest{
					Name:  name,
					Color: color, // Removed &
				})
				if err != nil {
					return errMsg{err}
				}
				// Refresh projects after creation
				return projectCreatedMsg{project: project}
			}
		}
		return nil
	}

	// Inputting name
	switch msg.String() {
	case "esc":
		// Cancel project creation
		h.IsCreatingProject = false
		h.ProjectInput.Reset()
		return nil

	case "enter":
		// Proceed to color selection
		name := strings.TrimSpace(h.ProjectInput.Value())
		if name == "" {
			h.IsCreatingProject = false
			h.ProjectInput.Reset()
			return nil
		}

		h.IsSelectingColor = true
		h.ColorCursor = 0
		h.ProjectInput.Blur()

		// Populate colors if empty (sorted)
		if len(h.AvailableColors) == 0 {
			for name := range styles.TodoistColorMap {
				h.AvailableColors = append(h.AvailableColors, name)
			}
			sort.Strings(h.AvailableColors)
		}
		return nil

	default:
		// Update text input
		var cmd tea.Cmd
		h.ProjectInput, cmd = h.ProjectInput.Update(msg)
		return cmd
	}
}

// handleProjectEditKeyMsg handles keyboard input during project editing.
func (h *Handler) handleProjectEditKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		// Cancel project editing
		h.IsEditingProject = false
		h.EditingProject = nil
		h.ProjectInput.Reset()
		return nil

	case "enter":
		// Submit project update
		name := strings.TrimSpace(h.ProjectInput.Value())
		if name == "" || h.EditingProject == nil {
			h.IsEditingProject = false
			h.EditingProject = nil
			h.ProjectInput.Reset()
			return nil
		}

		projectID := h.EditingProject.ID
		h.IsEditingProject = false
		h.EditingProject = nil
		h.ProjectInput.Reset()
		h.Loading = true

		return func() tea.Msg {
			project, err := h.Client.UpdateProject(projectID, api.UpdateProjectRequest{
				Name: &name,
			})
			if err != nil {
				return errMsg{err}
			}
			return projectUpdatedMsg{project: project}
		}

	default:
		// Update text input
		var cmd tea.Cmd
		h.ProjectInput, cmd = h.ProjectInput.Update(msg)
		return cmd
	}
}

// handleDeleteConfirmKeyMsg handles y/n/esc during project delete confirmation.
func (h *Handler) handleDeleteConfirmKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		// Confirm delete
		if h.EditingProject == nil {
			h.ConfirmDeleteProject = false
			return nil
		}

		projectID := h.EditingProject.ID
		h.ConfirmDeleteProject = false
		h.EditingProject = nil
		h.Loading = true

		return func() tea.Msg {
			err := h.Client.DeleteProject(projectID)
			if err != nil {
				return errMsg{err}
			}
			return projectDeletedMsg{id: projectID}
		}

	case "n", "N", "esc":
		// Cancel delete
		h.ConfirmDeleteProject = false
		h.EditingProject = nil
		return nil

	default:
		return nil
	}
}

// handleNewLabel opens the label creation input.
func (h *Handler) handleNewLabel() tea.Cmd {
	// Initialize label input
	h.LabelInput = textinput.New()
	h.LabelInput.Placeholder = "Enter label name..."
	h.LabelInput.CharLimit = 100
	h.LabelInput.Width = 40
	h.LabelInput.Focus()
	h.IsCreatingLabel = true

	// Pre-populate colors
	if len(h.AvailableColors) == 0 {
		for name := range styles.TodoistColorMap {
			h.AvailableColors = append(h.AvailableColors, name)
		}
		sort.Strings(h.AvailableColors)
	}
	h.ColorCursor = 0

	return nil
}

// handleLabelInputKeyMsg handles keyboard input during label creation.
func (h *Handler) handleLabelInputKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// If choosing color
	if h.IsSelectingColor {
		switch msg.String() {
		case "esc":
			h.IsSelectingColor = false
			h.LabelInput.Focus()
			return nil
		case "up", "k":
			if h.ColorCursor > 0 {
				h.ColorCursor--
			}
			return nil
		case "down", "j":
			if h.ColorCursor < len(h.AvailableColors)-1 {
				h.ColorCursor++
			}
			return nil
		case "enter":
			// Submit new label with color
			name := strings.TrimSpace(h.LabelInput.Value())
			color := h.AvailableColors[h.ColorCursor]

			h.IsCreatingLabel = false
			h.IsSelectingColor = false
			h.LabelInput.Reset()
			h.Loading = true

			return func() tea.Msg {
				label, err := h.Client.CreateLabel(api.CreateLabelRequest{
					Name:  name,
					Color: color, // Removed &
				})
				if err != nil {
					return errMsg{err}
				}
				return labelCreatedMsg{label: label}
			}
		}
		return nil
	}

	switch msg.String() {
	case "esc":
		// Cancel label creation
		h.IsCreatingLabel = false
		h.LabelInput.Reset()
		return nil

	case "enter":
		// Proceed to color selection
		name := strings.TrimSpace(h.LabelInput.Value())
		if name == "" {
			h.IsCreatingLabel = false
			h.LabelInput.Reset()
			return nil
		}

		h.IsSelectingColor = true
		h.ColorCursor = 0
		h.LabelInput.Blur()

		// Populate colors if empty (sorted)
		if len(h.AvailableColors) == 0 {
			for name := range styles.TodoistColorMap {
				h.AvailableColors = append(h.AvailableColors, name)
			}
			sort.Strings(h.AvailableColors)
		}
		return nil

	default:
		// Update text input
		var cmd tea.Cmd
		h.LabelInput, cmd = h.LabelInput.Update(msg)
		return cmd
	}
}

// handleLabelEditKeyMsg handles keyboard input during label editing.
func (h *Handler) handleLabelEditKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsEditingLabel = false
		h.EditingLabel = nil
		h.LabelInput.Reset()
		return nil

	case "enter":
		name := strings.TrimSpace(h.LabelInput.Value())
		if name == "" || h.EditingLabel == nil {
			h.IsEditingLabel = false
			h.EditingLabel = nil
			h.LabelInput.Reset()
			return nil
		}

		labelID := h.EditingLabel.ID
		h.IsEditingLabel = false
		h.EditingLabel = nil
		h.LabelInput.Reset()
		h.Loading = true

		return func() tea.Msg {
			label, err := h.Client.UpdateLabel(labelID, api.UpdateLabelRequest{
				Name: &name,
			})
			if err != nil {
				return errMsg{err}
			}
			return labelUpdatedMsg{label: label}
		}

	default:
		var cmd tea.Cmd
		h.LabelInput, cmd = h.LabelInput.Update(msg)
		return cmd
	}
}

// handleLabelDeleteConfirmKeyMsg handles y/n/esc during label delete confirmation.
func (h *Handler) handleLabelDeleteConfirmKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		if h.EditingLabel == nil {
			h.ConfirmDeleteLabel = false
			return nil
		}

		labelID := h.EditingLabel.ID
		h.ConfirmDeleteLabel = false
		h.EditingLabel = nil
		h.Loading = true

		return func() tea.Msg {
			if err := h.Client.DeleteLabel(labelID); err != nil {
				return errMsg{err}
			}
			return labelDeletedMsg{id: labelID}
		}

	case "n", "N", "esc":
		h.ConfirmDeleteLabel = false
		h.EditingLabel = nil
		return nil

	default:
		return nil
	}
}

// handleSectionInputKeyMsg handles keyboard input during section creation.
func (h *Handler) handleSectionInputKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsCreatingSection = false
		h.SectionInput.Reset()
		return nil

	case "enter":
		name := strings.TrimSpace(h.SectionInput.Value())
		if name == "" {
			return nil
		}

		// Use current project if available, otherwise fall back to sidebar
		var projectID string
		if h.CurrentProject != nil {
			projectID = h.CurrentProject.ID
		} else if h.CurrentTab == state.TabProjects && len(h.Projects) > 0 && h.SidebarCursor < len(h.SidebarItems) {
			projectID = h.SidebarItems[h.SidebarCursor].ID
		}

		if projectID != "" {
			h.IsCreatingSection = false
			h.SectionInput.Reset()
			h.Loading = true

			return func() tea.Msg {
				section, err := h.Client.CreateSection(api.CreateSectionRequest{
					ProjectID: projectID,
					Name:      name,
				})
				if err != nil {
					return errMsg{err}
				}
				return sectionCreatedMsg{section: section}
			}
		}
		h.IsCreatingSection = false
		return nil

	default:
		var cmd tea.Cmd
		h.SectionInput, cmd = h.SectionInput.Update(msg)
		return cmd
	}
}

// handleSectionEditKeyMsg handles keyboard input during section editing.
func (h *Handler) handleSectionEditKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsEditingSection = false
		h.EditingSection = nil
		h.SectionInput.Reset()
		return nil

	case "enter":
		name := strings.TrimSpace(h.SectionInput.Value())
		if name == "" || h.EditingSection == nil {
			h.IsEditingSection = false
			h.EditingSection = nil
			h.SectionInput.Reset()
			return nil
		}

		sectionID := h.EditingSection.ID
		h.IsEditingSection = false
		h.EditingSection = nil
		h.SectionInput.Reset()
		h.Loading = true

		return func() tea.Msg {
			section, err := h.Client.UpdateSection(sectionID, api.UpdateSectionRequest{
				Name: name,
			})
			if err != nil {
				return errMsg{err}
			}
			return sectionUpdatedMsg{section: section}
		}

	default:
		var cmd tea.Cmd
		h.SectionInput, cmd = h.SectionInput.Update(msg)
		return cmd
	}
}

// handleSectionDeleteConfirmKeyMsg handles y/n/esc during section delete confirmation.
func (h *Handler) handleSectionDeleteConfirmKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		if h.EditingSection == nil {
			h.ConfirmDeleteSection = false
			return nil
		}

		sectionID := h.EditingSection.ID
		h.ConfirmDeleteSection = false
		h.EditingSection = nil
		h.Loading = true

		return func() tea.Msg {
			if err := h.Client.DeleteSection(sectionID); err != nil {
				return errMsg{err}
			}
			return sectionDeletedMsg{id: sectionID}
		}

	case "n", "N", "esc":
		h.ConfirmDeleteSection = false
		h.EditingSection = nil
		return nil

	default:
		return nil
	}
}

// handleSectionsKeyMsg handles keyboard input for the sections management view.
func (h *Handler) handleSectionsKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.CurrentView = h.PreviousView
		h.TaskCursor = 0
		return nil

	case "up", "k":
		if h.TaskCursor > 0 {
			h.TaskCursor--
		}
		return nil

	case "down", "j":
		if h.TaskCursor < len(h.Sections)-1 {
			h.TaskCursor++
		}
		return nil

	case "shift+up", "K":
		if h.TaskCursor > 0 {
			idx := h.TaskCursor
			prev := idx - 1

			// Swap elements locally
			h.Sections[idx], h.Sections[prev] = h.Sections[prev], h.Sections[idx]

			// Swap orders locally to maintain consistency
			h.Sections[idx].SectionOrder, h.Sections[prev].SectionOrder = h.Sections[prev].SectionOrder, h.Sections[idx].SectionOrder

			// Update cursor to follow the moved item
			h.TaskCursor--

			// Send Sync API update with ALL sections
			return h.reorderSectionsCmd(h.Sections)
		}
		return nil

	case "shift+down", "J":
		if h.TaskCursor < len(h.Sections)-1 {
			idx := h.TaskCursor
			next := idx + 1

			// Swap elements locally
			h.Sections[idx], h.Sections[next] = h.Sections[next], h.Sections[idx]

			// Swap orders locally
			h.Sections[idx].SectionOrder, h.Sections[next].SectionOrder = h.Sections[next].SectionOrder, h.Sections[idx].SectionOrder

			// Update cursor to follow the moved item
			h.TaskCursor++

			// Send Sync API update with ALL sections
			return h.reorderSectionsCmd(h.Sections)
		}
		return nil

	case "a":
		h.SectionInput = textinput.New()
		h.SectionInput.Placeholder = "New section name..."
		h.SectionInput.CharLimit = 100
		h.SectionInput.Width = 40
		h.SectionInput.Focus()
		h.IsCreatingSection = true
		return nil

	case "e":
		if len(h.Sections) == 0 {
			return nil
		}
		if h.TaskCursor >= 0 && h.TaskCursor < len(h.Sections) {
			h.EditingSection = &h.Sections[h.TaskCursor]
			h.SectionInput = textinput.New()
			h.SectionInput.SetValue(h.EditingSection.Name)
			h.SectionInput.CharLimit = 100
			h.SectionInput.Width = 40
			h.SectionInput.Focus()
			h.IsEditingSection = true
		}
		return nil

	case "d", "delete":
		if len(h.Sections) == 0 {
			return nil
		}
		if h.TaskCursor >= 0 && h.TaskCursor < len(h.Sections) {
			h.EditingSection = &h.Sections[h.TaskCursor]
			h.ConfirmDeleteSection = true
		}
		return nil
	}

	return nil
}

// handleMoveTaskKeyMsg handles keyboard input for moving a task to a section.
func (h *Handler) handleMoveTaskKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsMovingTask = false
		h.MoveSectionCursor = 0
		return nil

	case "up", "k":
		if h.MoveSectionCursor > 0 {
			h.MoveSectionCursor--
		}
		return nil

	case "down", "j":
		if h.MoveSectionCursor < len(h.Sections)-1 {
			h.MoveSectionCursor++
		}
		return nil

	case "enter":
		if len(h.Sections) == 0 {
			h.IsMovingTask = false
			return nil
		}

		sectionID := h.Sections[h.MoveSectionCursor].ID
		if sectionID == "" {
			h.IsMovingTask = false
			h.StatusMsg = "Invalid section"
			return nil
		}

		h.IsMovingTask = false
		h.StatusMsg = "Moving tasks..."

		var tasksToMove []*api.Task

		// Check if we have multiple selected tasks
		if len(h.SelectedTaskIDs) > 0 {
			for i := range h.Tasks {
				if h.SelectedTaskIDs[h.Tasks[i].ID] {
					tasksToMove = append(tasksToMove, &h.Tasks[i])
				}
			}
		} else {
			// Single task selection
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
				tasksToMove = append(tasksToMove, task)
			}
		}

		if len(tasksToMove) == 0 {
			h.StatusMsg = "No tasks selected"
			return nil
		}

		var cmds []tea.Cmd
		movedCount := 0

		for _, task := range tasksToMove {
			// Skip if already in section
			if task.SectionID != nil && *task.SectionID == sectionID {
				continue
			}

			movedCount++

			// Optimistic update
			safeSectionID := sectionID
			task.SectionID = &safeSectionID

			// Update AllTasks
			for i := range h.AllTasks {
				if h.AllTasks[i].ID == task.ID {
					h.AllTasks[i].SectionID = &safeSectionID
					break
				}
			}

			// Capture loop variables
			tID := task.ID
			sID := sectionID
			taskCopy := *task

			cmds = append(cmds, func() tea.Msg {
				err := h.Client.MoveTask(tID, &sID, nil, nil)
				if err != nil {
					return errMsg{err}
				}
				return taskUpdatedMsg{task: &taskCopy}
			})
		}

		if movedCount == 0 {
			h.StatusMsg = "Tasks already in this section"
			return nil
		}

		// Clear selection after move
		h.SelectedTaskIDs = make(map[string]bool)

		return tea.Batch(cmds...)

	}
	return nil
}

// handleCommentInputKeyMsg handles keyboard input for adding a comment.
func (h *Handler) handleCommentInputKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsAddingComment = false
		h.CommentInput.Reset()
		return nil

	case "enter":
		content := strings.TrimSpace(h.CommentInput.Value())
		if content == "" {
			return nil
		}

		// determine task ID (from selection or cursor)
		taskID := ""
		if h.SelectedTask != nil {
			taskID = h.SelectedTask.ID
		} else if len(h.Tasks) > 0 && h.TaskCursor < len(h.Tasks) {
			taskID = h.Tasks[h.TaskCursor].ID
		} else {
			h.IsAddingComment = false
			return nil
		}

		h.IsAddingComment = false
		h.CommentInput.Reset()
		h.Loading = true
		h.StatusMsg = "Adding comment..."

		return func() tea.Msg {
			comment, err := h.Client.CreateComment(api.CreateCommentRequest{
				TaskID:  taskID,
				Content: content,
			})
			if err != nil {
				return errMsg{err}
			}
			return commentCreatedMsg{comment: comment}
		}

	default:
		var cmd tea.Cmd
		h.CommentInput, cmd = h.CommentInput.Update(msg)
		return cmd
	}
}

// renderSections renders the sections management view.
func (h *Handler) renderSections() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Manage Sections"))
	b.WriteString("\n\n")

	if len(h.Sections) == 0 {
		b.WriteString(styles.HelpDesc.Render("No sections found. Press 'a' to add one."))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpDesc.Render("Esc: back"))
		return b.String()
	}

	// Render list
	for i, section := range h.Sections {
		cursor := "  "
		style := lipgloss.NewStyle()

		if i == h.TaskCursor {
			cursor = "> "
			style = lipgloss.NewStyle().Foreground(styles.Highlight)
		}

		b.WriteString(cursor + style.Render(section.Name) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("j/k: nav • a: add • e: edit • d: delete • Esc: back"))

	return b.String()
}

// handleAddSubtask opens the inline subtask creation input.
func (h *Handler) handleAddSubtask() tea.Cmd {
	// Guard: Only in main pane with tasks
	if h.FocusedPane != state.PaneMain || len(h.Tasks) == 0 {
		return nil
	}

	// Get the selected task using ordered indices
	taskIndex := h.TaskCursor
	if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
		taskIndex = h.TaskOrderedIndices[h.TaskCursor]
	}
	if taskIndex < 0 || taskIndex >= len(h.Tasks) {
		return nil
	}

	// Initialize subtask input
	h.ParentTaskID = h.Tasks[taskIndex].ID
	h.SubtaskInput = textinput.New()
	h.SubtaskInput.Placeholder = "Enter subtask..."
	h.SubtaskInput.CharLimit = 200
	h.SubtaskInput.Width = 50
	h.SubtaskInput.Focus()
	h.IsCreatingSubtask = true

	return nil
}

// handleSubtaskInputKeyMsg handles keyboard input during subtask creation.
func (h *Handler) handleSubtaskInputKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		// Cancel subtask creation
		h.IsCreatingSubtask = false
		h.ParentTaskID = ""
		h.SubtaskInput.Reset()
		return nil

	case "enter":
		// Submit new subtask
		content := strings.TrimSpace(h.SubtaskInput.Value())
		if content == "" {
			h.IsCreatingSubtask = false
			h.ParentTaskID = ""
			h.SubtaskInput.Reset()
			return nil
		}

		parentID := h.ParentTaskID
		h.IsCreatingSubtask = false
		h.ParentTaskID = ""
		h.SubtaskInput.Reset()
		h.Loading = true

		return func() tea.Msg {
			_, err := h.Client.CreateTask(api.CreateTaskRequest{
				Content:  content,
				ParentID: parentID,
			})
			if err != nil {
				return errMsg{err}
			}
			return subtaskCreatedMsg{}
		}

	default:
		// Update text input
		var cmd tea.Cmd
		h.SubtaskInput, cmd = h.SubtaskInput.Update(msg)
		return cmd
	}
}

// handleSearch opens the search view.
func (h *Handler) handleSearch() tea.Cmd {
	h.PreviousView = h.CurrentView
	h.CurrentView = state.ViewSearch
	h.SearchInput.Reset()
	h.SearchInput.Focus()
	h.SearchResults = nil
	h.SearchQuery = ""
	h.TaskCursor = 0
	return nil
}

// handleSearchKeyMsg handles keyboard input when search is active.
func (h *Handler) handleSearchKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		// Cancel search and go back
		h.CurrentView = h.PreviousView
		h.SearchInput.Blur()
		h.SearchResults = nil
		h.SearchQuery = ""
		return nil

	case "enter":
		// Select task from search results
		if len(h.SearchResults) > 0 && h.TaskCursor < len(h.SearchResults) {
			h.SelectedTask = &h.SearchResults[h.TaskCursor]
			h.PreviousView = state.ViewSearch
			h.CurrentView = state.ViewTaskDetail
			return nil
		}
		return nil

	case "down", "j":
		if h.TaskCursor < len(h.SearchResults)-1 {
			h.TaskCursor++
		}
		return nil

	case "up", "k":
		if h.TaskCursor > 0 {
			h.TaskCursor--
		}
		return nil

	case "x":
		// Complete task from search results
		if len(h.SearchResults) > 0 && h.TaskCursor < len(h.SearchResults) {
			task := &h.SearchResults[h.TaskCursor]
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
				return searchRefreshMsg{}
			}
		}
		return nil
	}

	// Update search input and filter results
	var cmd tea.Cmd
	h.SearchInput, cmd = h.SearchInput.Update(msg)
	h.SearchQuery = h.SearchInput.Value()
	h.filterSearchResults()
	h.TaskCursor = 0 // Reset cursor when query changes
	return cmd
}

// filterSearchResults filters tasks based on search query.
func (h *Handler) filterSearchResults() {
	query := strings.ToLower(strings.TrimSpace(h.SearchQuery))
	if query == "" {
		h.SearchResults = nil
		return
	}

	var results []api.Task
	for _, task := range h.Tasks {
		// Search in content, description, and labels
		if strings.Contains(strings.ToLower(task.Content), query) ||
			strings.Contains(strings.ToLower(task.Description), query) {
			results = append(results, task)
			continue
		}

		// Search in labels
		for _, label := range task.Labels {
			if strings.Contains(strings.ToLower(label), query) {
				results = append(results, task)
				break
			}
		}
	}

	h.SearchResults = results
}

// refreshSearchResults refreshes the search results after a task update.
func (h *Handler) refreshSearchResults() tea.Cmd {
	return func() tea.Msg {
		// Reload all tasks
		tasks, err := h.Client.GetTasks(api.TaskFilter{})
		if err != nil {
			return errMsg{err}
		}
		h.Tasks = tasks
		h.filterSearchResults()
		return dataLoadedMsg{tasks: tasks}
	}
}

// handlePriority sets task priority.
func (h *Handler) handlePriority(action string) tea.Cmd {
	if h.FocusedPane != state.PaneMain || len(h.Tasks) == 0 {
		return nil
	}

	// Use ordered indices if available
	taskIndex := h.TaskCursor
	if len(h.TaskOrderedIndices) > 0 && h.TaskCursor < len(h.TaskOrderedIndices) {
		taskIndex = h.TaskOrderedIndices[h.TaskCursor]
	}
	// Check specifically for section headers (negative indices)
	if taskIndex < 0 {
		return nil
	}
	if taskIndex >= len(h.Tasks) {
		return nil
	}

	// Optimistic update
	task := &h.Tasks[taskIndex]
	var priority int
	switch action {
	case "priority1":
		priority = 4 // Todoist uses 4 as highest
	case "priority2":
		priority = 3
	case "priority3":
		priority = 2
	case "priority4":
		priority = 1
	}

	// Apply change immediately to local state
	task.Priority = priority

	// Also update in AllTasks to maintain consistency across views/filters
	for i := range h.AllTasks {
		if h.AllTasks[i].ID == task.ID {
			h.AllTasks[i].Priority = priority
			break
		}
	}

	// Perform API update in background without blocking UI
	taskID := task.ID
	return func() tea.Msg {
		_, err := h.Client.UpdateTask(taskID, api.UpdateTaskRequest{
			Priority: &priority,
		})
		if err != nil {
			return errMsg{err}
		}
		// Trigger silent refresh to sync with server eventually
		return taskUpdatedMsg{}
	}
}

// handleDueToday sets the task due date to today.
func (h *Handler) handleDueToday() tea.Cmd {
	if h.FocusedPane != state.PaneMain || len(h.Tasks) == 0 {
		return nil
	}

	// Guard: Don't operate when viewing label list
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
	dueString := "today"

	// Optimistic update
	now := time.Now()
	dateStr := now.Format("2006-01-02")

	// Create or update Due object
	if task.Due == nil {
		task.Due = &api.Due{}
	}
	task.Due.String = "today"
	task.Due.Date = dateStr
	task.Due.Datetime = nil // Clear specific time if moving to "today" broadly

	// Update AllTasks
	for i := range h.AllTasks {
		if h.AllTasks[i].ID == task.ID {
			if h.AllTasks[i].Due == nil {
				h.AllTasks[i].Due = &api.Due{}
			}
			h.AllTasks[i].Due.String = "today"
			h.AllTasks[i].Due.Date = dateStr
			h.AllTasks[i].Due.Datetime = nil
			break
		}
	}

	h.StatusMsg = "Moving to today..."
	// Remove blocking loading state

	return func() tea.Msg {
		_, err := h.Client.UpdateTask(task.ID, api.UpdateTaskRequest{
			DueString: &dueString,
		})
		if err != nil {
			return errMsg{err}
		}
		return taskUpdatedMsg{}
	}
}

// handleDueTomorrow sets the task due date to tomorrow.
func (h *Handler) handleDueTomorrow() tea.Cmd {
	if h.FocusedPane != state.PaneMain || len(h.Tasks) == 0 {
		return nil
	}

	// Guard: Don't operate when viewing label list
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
	dueString := "tomorrow"

	// Optimistic update
	tomorrow := time.Now().AddDate(0, 0, 1)
	dateStr := tomorrow.Format("2006-01-02")

	if task.Due == nil {
		task.Due = &api.Due{}
	}
	task.Due.String = "tomorrow"
	task.Due.Date = dateStr
	task.Due.Datetime = nil

	// Update AllTasks
	for i := range h.AllTasks {
		if h.AllTasks[i].ID == task.ID {
			if h.AllTasks[i].Due == nil {
				h.AllTasks[i].Due = &api.Due{}
			}
			h.AllTasks[i].Due.String = "tomorrow"
			h.AllTasks[i].Due.Date = dateStr
			h.AllTasks[i].Due.Datetime = nil
			break
		}
	}

	h.StatusMsg = "Moving to tomorrow..."

	return func() tea.Msg {
		_, err := h.Client.UpdateTask(task.ID, api.UpdateTaskRequest{
			DueString: &dueString,
		})
		if err != nil {
			return errMsg{err}
		}
		// Return taskUpdatedMsg to trigger eventual consistency refresh
		return taskUpdatedMsg{}
	}
}

// loadProjectTasks loads tasks for a specific project.
func (h *Handler) loadProjectTasks(projectID string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := h.Client.GetTasks(api.TaskFilter{
			ProjectID: projectID,
		})
		if err != nil {
			return errMsg{err}
		}

		sections, err := h.Client.GetSections(projectID)
		if err != nil {
			return errMsg{err}
		}

		return dataLoadedMsg{
			tasks:    tasks,
			sections: sections,
		}
	}
}

// filterProjectTasks filters cached tasks and sections for a project.
func (h *Handler) filterProjectTasks(projectID string) tea.Cmd {
	var tasks []api.Task
	for _, t := range h.AllTasks {
		if t.ProjectID == projectID {
			tasks = append(tasks, t)
		}
	}

	var sections []api.Section
	for _, s := range h.AllSections {
		if s.ProjectID == projectID {
			sections = append(sections, s)
		}
	}

	// Sort sections
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].SectionOrder < sections[j].SectionOrder
	})

	h.Tasks = tasks
	h.Sections = sections
	h.sortTasks()
	return nil
}

// refreshTasks refreshes the current task list.
func (h *Handler) refreshTasks() tea.Cmd {
	return func() tea.Msg {
		// Always fetch all tasks to keep the cache fresh
		allTasks, err := h.Client.GetTasks(api.TaskFilter{})
		if err != nil {
			return errMsg{err}
		}

		var filteredTasks []api.Task
		if h.CurrentView == state.ViewProject && h.CurrentProject != nil {
			for _, t := range allTasks {
				if t.ProjectID == h.CurrentProject.ID {
					filteredTasks = append(filteredTasks, t)
				}
			}
		} else if h.CurrentView == state.ViewLabels && h.CurrentLabel != nil {
			for _, t := range allTasks {
				for _, l := range t.Labels {
					if l == h.CurrentLabel.Name {
						filteredTasks = append(filteredTasks, t)
						break
					}
				}
			}
		} else {
			// Default to today | overdue
			for _, t := range allTasks {
				if t.IsDueToday() || t.IsOverdue() {
					filteredTasks = append(filteredTasks, t)
				}
			}
		}

		return dataLoadedMsg{
			tasks:    filteredTasks,
			allTasks: allTasks,
		}
	}
}

// loadTodayTasks loads today's tasks including overdue.
func (h *Handler) loadTodayTasks() tea.Cmd {
	return func() tea.Msg {
		// Use filter endpoint for today | overdue
		tasks, err := h.Client.GetTasksByFilter("today | overdue")
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{tasks: tasks}
	}
}

// loadInboxTasks loads inbox tasks using the project logic to include sections.
func (h *Handler) loadInboxTasks() tea.Cmd {
	// Find (or guess) Inbox project ID
	var inboxID string
	for _, p := range h.Projects {
		if p.InboxProject {
			inboxID = p.ID
			break
		}
	}

	if inboxID != "" {
		return h.loadProjectTasks(inboxID)
	}

	return func() tea.Msg {
		tasks, err := h.Client.GetTasksByFilter("inbox")
		if err != nil {
			return errMsg{err}
		}
		// Return empty sections explicitly to clear any old ones if filter is used
		return dataLoadedMsg{tasks: tasks, sections: []api.Section{}}
	}
}

// filterTodayTasks filters cached tasks for today/overdue.
func (h *Handler) filterTodayTasks() tea.Cmd {
	var tasks []api.Task
	for _, t := range h.AllTasks {
		if t.IsOverdue() || t.IsDueToday() {
			tasks = append(tasks, t)
		}
	}
	h.Tasks = tasks
	h.sortTasks()
	return nil
}

// loadUpcomingTasks loads all tasks with due dates for the upcoming view.
func (h *Handler) loadUpcomingTasks() tea.Cmd {
	return func() tea.Msg {
		// Get all tasks
		allTasks, err := h.Client.GetTasks(api.TaskFilter{})
		if err != nil {
			return errMsg{err}
		}

		// Filter to tasks with due dates
		var upcoming []api.Task
		for _, t := range allTasks {
			if t.Due != nil {
				upcoming = append(upcoming, t)
			}
		}
		return dataLoadedMsg{
			tasks:    upcoming,
			allTasks: allTasks,
		}
	}
}

// filterUpcomingTasks filters cached tasks for upcoming view.
func (h *Handler) filterUpcomingTasks() tea.Cmd {
	var upcoming []api.Task
	for _, t := range h.AllTasks {
		if t.Due != nil {
			upcoming = append(upcoming, t)
		}
	}
	h.Tasks = upcoming
	h.sortTasks()
	return nil
}

// loadAllTasks loads all tasks (for calendar view).
func (h *Handler) loadAllTasks() tea.Cmd {
	return func() tea.Msg {
		allTasks, err := h.Client.GetTasks(api.TaskFilter{})
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{allTasks: allTasks}
	}
}

// loadProjects loads all projects.
func (h *Handler) loadProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := h.Client.GetProjects()
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{projects: projects}
	}
}

// loadLabels loads all labels.
func (h *Handler) loadLabels() tea.Cmd {
	return func() tea.Msg {
		labels, err := h.Client.GetLabels()
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{labels: labels}
	}
}

// loadLabelTasks loads tasks filtered by a specific label.
func (h *Handler) loadLabelTasks(labelName string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := h.Client.GetTasksByFilter("@" + labelName)
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{tasks: tasks}
	}
}

// filterLabelTasks filters cached tasks for a label.
func (h *Handler) filterLabelTasks(labelName string) tea.Cmd {
	var tasks []api.Task
	for _, t := range h.AllTasks {
		hasLabel := false
		for _, l := range t.Labels {
			if l == labelName {
				hasLabel = true
				break
			}
		}
		if hasLabel {
			tasks = append(tasks, t)
		}
	}
	h.Tasks = tasks
	h.sortTasks()
	return nil
}

// loadCalendarDayTasks loads tasks for the selected calendar day.
func (h *Handler) loadCalendarDayTasks() tea.Cmd {
	selectedDate := time.Date(h.CalendarDate.Year(), h.CalendarDate.Month(), h.CalendarDay, 0, 0, 0, 0, time.Local)
	dateStr := selectedDate.Format("2006-01-02")

	// Filter from allTasks instead of API call
	var dayTasks []api.Task
	for _, t := range h.AllTasks {
		if t.Due != nil && t.Due.Date == dateStr {
			dayTasks = append(dayTasks, t)
		}
	}
	h.Tasks = dayTasks
	return nil
}

// filterCalendarTasks filters cached tasks for calendar view (which uses allTasks).
func (h *Handler) filterCalendarTasks() tea.Cmd {
	// Calendar view mainly relies on h.AllTasks which is already loaded.
	// But we might want to refresh specific day tasks if we switch to day view.
	// For main calendar view, we just need to ensured allTasks is available.
	// Since we keep allTasks updated, we just need to trigger a view update if needed.
	return nil
}

// loadTaskComments loads comments for the selected task.
func (h *Handler) loadTaskComments() tea.Cmd {
	if h.SelectedTask == nil {
		return nil
	}
	taskID := h.SelectedTask.ID
	return func() tea.Msg {
		comments, err := h.Client.GetComments(taskID, "")
		if err != nil {
			return errMsg{err}
		}
		return commentsLoadedMsg{comments: comments}
	}
}

// buildSidebarItems constructs the sidebar items list with only projects (for Projects tab).
func (h *Handler) buildSidebarItems() {
	h.SidebarItems = []components.SidebarItem{}

	// Map projects for easy count lookup
	counts := make(map[string]int)
	for _, t := range h.AllTasks {
		if !t.Checked && !t.IsDeleted && t.ProjectID != "" {
			counts[t.ProjectID]++
		}
	}

	// Add favorite projects first
	for _, p := range h.Projects {
		// Skip inbox project (it has its own tab)
		if p.InboxProject {
			continue
		}
		if p.IsFavorite {
			icon := "❤︎"

			h.SidebarItems = append(h.SidebarItems, components.SidebarItem{
				Type:       "project",
				ID:         p.ID,
				Name:       p.Name,
				Icon:       icon,
				Count:      counts[p.ID],
				IsFavorite: true,
				ParentID:   p.ParentID,
				Color:      p.Color,
			})
		}
	}

	// Add separator if there were favorites
	hasFavorites := false
	for _, p := range h.Projects {
		if p.IsFavorite {
			hasFavorites = true
			break
		}
	}
	if hasFavorites {
		h.SidebarItems = append(h.SidebarItems, components.SidebarItem{Type: "separator", ID: "", Name: ""})
	}

	// Add remaining projects (non-favorites)
	for _, p := range h.Projects {
		// Skip inbox project
		if p.InboxProject {
			continue
		}
		if !p.IsFavorite {
			icon := "#"

			h.SidebarItems = append(h.SidebarItems, components.SidebarItem{
				Type:     "project",
				ID:       p.ID,
				Name:     p.Name,
				Icon:     icon,
				Count:    counts[p.ID],
				ParentID: p.ParentID,
				Color:    p.Color,
			})
		}
	}
}

// handleToggleFavorite toggles the favorite status of the selected project.
func (h *Handler) handleToggleFavorite() tea.Cmd {
	if len(h.SidebarItems) == 0 || h.SidebarCursor >= len(h.SidebarItems) {
		return nil
	}

	item := h.SidebarItems[h.SidebarCursor]
	if item.Type != "project" || item.ID == "" {
		return nil
	}

	// Find the project object to get current status
	var currentFavorite bool
	found := false
	for _, p := range h.Projects {
		if p.ID == item.ID {
			currentFavorite = p.IsFavorite
			found = true
			break
		}
	}

	if !found {
		// Fallback to item's own favorite status if project not found in list
		currentFavorite = item.IsFavorite
	}

	newStatus := !currentFavorite
	projectID := item.ID // Capture by value
	pName := item.Name   // Capture name to ensure request is never "empty"

	h.Loading = true
	if newStatus {
		h.StatusMsg = "Favoriting project..."
	} else {
		h.StatusMsg = "Unfavoriting project..."
	}

	return func() tea.Msg {
		updatedProject, err := h.Client.UpdateProject(projectID, api.UpdateProjectRequest{
			Name:       &pName,
			IsFavorite: api.BoolPtr(newStatus),
		})
		if err != nil {
			return errMsg{err}
		}
		return projectUpdatedMsg{project: updatedProject}
	}
}
