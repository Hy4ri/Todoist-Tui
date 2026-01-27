import (
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/components"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

func (a *App) handleNewProject() (tea.Model, tea.Cmd) {
	// Only allow creating projects in Projects tab
	if a.currentTab != TabProjects {
		return a, nil
	}

	// Initialize project input
	a.projectInput = textinput.New()
	a.projectInput.Placeholder = "Enter project name..."
	a.projectInput.CharLimit = 100
	a.projectInput.Width = 40
	a.projectInput.Focus()
	a.isCreatingProject = true

	return a, nil
}

// handleProjectInputKeyMsg handles keyboard input during project creation.
func (a *App) handleProjectInputKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel project creation
		a.isCreatingProject = false
		a.projectInput.Reset()
		return a, nil

	case "enter":
		// Submit new project
		name := strings.TrimSpace(a.projectInput.Value())
		if name == "" {
			a.isCreatingProject = false
			a.projectInput.Reset()
			return a, nil
		}

		a.isCreatingProject = false
		a.projectInput.Reset()
		a.loading = true

		return a, func() tea.Msg {
			project, err := a.client.CreateProject(api.CreateProjectRequest{
				Name: name,
			})
			if err != nil {
				return errMsg{err}
			}
			// Refresh projects after creation
			return projectCreatedMsg{project: project}
		}

	default:
		// Update text input
		var cmd tea.Cmd
		a.projectInput, cmd = a.projectInput.Update(msg)
		return a, cmd
	}
}

// handleProjectEditKeyMsg handles keyboard input during project editing.
func (a *App) handleProjectEditKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel project editing
		a.isEditingProject = false
		a.editingProject = nil
		a.projectInput.Reset()
		return a, nil

	case "enter":
		// Submit project update
		name := strings.TrimSpace(a.projectInput.Value())
		if name == "" || a.editingProject == nil {
			a.isEditingProject = false
			a.editingProject = nil
			a.projectInput.Reset()
			return a, nil
		}

		projectID := a.editingProject.ID
		a.isEditingProject = false
		a.editingProject = nil
		a.projectInput.Reset()
		a.loading = true

		return a, func() tea.Msg {
			project, err := a.client.UpdateProject(projectID, api.UpdateProjectRequest{
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
		a.projectInput, cmd = a.projectInput.Update(msg)
		return a, cmd
	}
}

// handleDeleteConfirmKeyMsg handles y/n/esc during project delete confirmation.
func (a *App) handleDeleteConfirmKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		// Confirm delete
		if a.editingProject == nil {
			a.confirmDeleteProject = false
			return a, nil
		}

		projectID := a.editingProject.ID
		a.confirmDeleteProject = false
		a.editingProject = nil
		a.loading = true

		return a, func() tea.Msg {
			err := a.client.DeleteProject(projectID)
			if err != nil {
				return errMsg{err}
			}
			return projectDeletedMsg{id: projectID}
		}

	case "n", "N", "esc":
		// Cancel delete
		a.confirmDeleteProject = false
		a.editingProject = nil
		return a, nil

	default:
		return a, nil
	}
}

// handleNewLabel opens the label creation input.
func (a *App) handleNewLabel() (tea.Model, tea.Cmd) {
	// Initialize label input
	a.labelInput = textinput.New()
	a.labelInput.Placeholder = "Enter label name..."
	a.labelInput.CharLimit = 100
	a.labelInput.Width = 40
	a.labelInput.Focus()
	a.isCreatingLabel = true

	return a, nil
}

// handleLabelInputKeyMsg handles keyboard input during label creation.
func (a *App) handleLabelInputKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel label creation
		a.isCreatingLabel = false
		a.labelInput.Reset()
		return a, nil

	case "enter":
		// Submit new label
		name := strings.TrimSpace(a.labelInput.Value())
		if name == "" {
			a.isCreatingLabel = false
			a.labelInput.Reset()
			return a, nil
		}

		a.isCreatingLabel = false
		a.labelInput.Reset()
		a.loading = true

		return a, func() tea.Msg {
			label, err := a.client.CreateLabel(api.CreateLabelRequest{
				Name: name,
			})
			if err != nil {
				return errMsg{err}
			}
			return labelCreatedMsg{label: label}
		}

	default:
		// Update text input
		var cmd tea.Cmd
		a.labelInput, cmd = a.labelInput.Update(msg)
		return a, cmd
	}
}

// handleLabelEditKeyMsg handles keyboard input during label editing.
func (a *App) handleLabelEditKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.isEditingLabel = false
		a.editingLabel = nil
		a.labelInput.Reset()
		return a, nil

	case "enter":
		name := strings.TrimSpace(a.labelInput.Value())
		if name == "" || a.editingLabel == nil {
			a.isEditingLabel = false
			a.editingLabel = nil
			a.labelInput.Reset()
			return a, nil
		}

		labelID := a.editingLabel.ID
		a.isEditingLabel = false
		a.editingLabel = nil
		a.labelInput.Reset()
		a.loading = true

		return a, func() tea.Msg {
			label, err := a.client.UpdateLabel(labelID, api.UpdateLabelRequest{
				Name: &name,
			})
			if err != nil {
				return errMsg{err}
			}
			return labelUpdatedMsg{label: label}
		}

	default:
		var cmd tea.Cmd
		a.labelInput, cmd = a.labelInput.Update(msg)
		return a, cmd
	}
}

// handleLabelDeleteConfirmKeyMsg handles y/n/esc during label delete confirmation.
func (a *App) handleLabelDeleteConfirmKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if a.editingLabel == nil {
			a.confirmDeleteLabel = false
			return a, nil
		}

		labelID := a.editingLabel.ID
		a.confirmDeleteLabel = false
		a.editingLabel = nil
		a.loading = true

		return a, func() tea.Msg {
			if err := a.client.DeleteLabel(labelID); err != nil {
				return errMsg{err}
			}
			return labelDeletedMsg{id: labelID}
		}

	case "n", "N", "esc":
		a.confirmDeleteLabel = false
		a.editingLabel = nil
		return a, nil

	default:
		return a, nil
	}
}

// handleSectionInputKeyMsg handles keyboard input during section creation.
func (a *App) handleSectionInputKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.isCreatingSection = false
		a.sectionInput.Reset()
		return a, nil

	case "enter":
		name := strings.TrimSpace(a.sectionInput.Value())
		if name == "" {
			return a, nil
		}

		// Use current project if available, otherwise fall back to sidebar
		var projectID string
		if a.currentProject != nil {
			projectID = a.currentProject.ID
		} else if a.currentTab == TabProjects && len(a.projects) > 0 && a.sidebarCursor < len(a.sidebarItems) {
			projectID = a.sidebarItems[a.sidebarCursor].ID
		}

		if projectID != "" {
			a.isCreatingSection = false
			a.sectionInput.Reset()
			a.loading = true

			return a, func() tea.Msg {
				section, err := a.client.CreateSection(api.CreateSectionRequest{
					ProjectID: projectID,
					Name:      name,
				})
				if err != nil {
					return errMsg{err}
				}
				return sectionCreatedMsg{section: section}
			}
		}
		a.isCreatingSection = false
		return a, nil

	default:
		var cmd tea.Cmd
		a.sectionInput, cmd = a.sectionInput.Update(msg)
		return a, cmd
	}
}

// handleSectionEditKeyMsg handles keyboard input during section editing.
func (a *App) handleSectionEditKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.isEditingSection = false
		a.editingSection = nil
		a.sectionInput.Reset()
		return a, nil

	case "enter":
		name := strings.TrimSpace(a.sectionInput.Value())
		if name == "" || a.editingSection == nil {
			a.isEditingSection = false
			a.editingSection = nil
			a.sectionInput.Reset()
			return a, nil
		}

		sectionID := a.editingSection.ID
		a.isEditingSection = false
		a.editingSection = nil
		a.sectionInput.Reset()
		a.loading = true

		return a, func() tea.Msg {
			section, err := a.client.UpdateSection(sectionID, api.UpdateSectionRequest{
				Name: name,
			})
			if err != nil {
				return errMsg{err}
			}
			return sectionUpdatedMsg{section: section}
		}

	default:
		var cmd tea.Cmd
		a.sectionInput, cmd = a.sectionInput.Update(msg)
		return a, cmd
	}
}

// handleSectionDeleteConfirmKeyMsg handles y/n/esc during section delete confirmation.
func (a *App) handleSectionDeleteConfirmKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if a.editingSection == nil {
			a.confirmDeleteSection = false
			return a, nil
		}

		sectionID := a.editingSection.ID
		a.confirmDeleteSection = false
		a.editingSection = nil
		a.loading = true

		return a, func() tea.Msg {
			if err := a.client.DeleteSection(sectionID); err != nil {
				return errMsg{err}
			}
			return sectionDeletedMsg{id: sectionID}
		}

	case "n", "N", "esc":
		a.confirmDeleteSection = false
		a.editingSection = nil
		return a, nil

	default:
		return a, nil
	}
}

// handleSectionsKeyMsg handles keyboard input for the sections management view.
func (a *App) handleSectionsKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.currentView = a.previousView
		a.taskCursor = 0
		return a, nil

	case "up", "k":
		if a.taskCursor > 0 {
			a.taskCursor--
		}
		return a, nil

	case "down", "j":
		if a.taskCursor < len(a.sections)-1 {
			a.taskCursor++
		}
		return a, nil

	case "shift+up", "K":
		if a.taskCursor > 0 {
			idx := a.taskCursor
			prev := idx - 1

			// Swap elements locally
			a.sections[idx], a.sections[prev] = a.sections[prev], a.sections[idx]

			// Swap orders locally to maintain consistency
			a.sections[idx].SectionOrder, a.sections[prev].SectionOrder = a.sections[prev].SectionOrder, a.sections[idx].SectionOrder

			// Update cursor to follow the moved item
			a.taskCursor--

			// Send Sync API update with ALL sections
			return a, a.reorderSectionsCmd(a.sections)
		}
		return a, nil

	case "shift+down", "J":
		if a.taskCursor < len(a.sections)-1 {
			idx := a.taskCursor
			next := idx + 1

			// Swap elements locally
			a.sections[idx], a.sections[next] = a.sections[next], a.sections[idx]

			// Swap orders locally
			a.sections[idx].SectionOrder, a.sections[next].SectionOrder = a.sections[next].SectionOrder, a.sections[idx].SectionOrder

			// Update cursor to follow the moved item
			a.taskCursor++

			// Send Sync API update with ALL sections
			return a, a.reorderSectionsCmd(a.sections)
		}
		return a, nil

	case "a":
		a.sectionInput = textinput.New()
		a.sectionInput.Placeholder = "New section name..."
		a.sectionInput.CharLimit = 100
		a.sectionInput.Width = 40
		a.sectionInput.Focus()
		a.isCreatingSection = true
		return a, nil

	case "e":
		if len(a.sections) == 0 {
			return a, nil
		}
		if a.taskCursor >= 0 && a.taskCursor < len(a.sections) {
			a.editingSection = &a.sections[a.taskCursor]
			a.sectionInput = textinput.New()
			a.sectionInput.SetValue(a.editingSection.Name)
			a.sectionInput.CharLimit = 100
			a.sectionInput.Width = 40
			a.sectionInput.Focus()
			a.isEditingSection = true
		}
		return a, nil

	case "d", "delete":
		if len(a.sections) == 0 {
			return a, nil
		}
		if a.taskCursor >= 0 && a.taskCursor < len(a.sections) {
			a.editingSection = &a.sections[a.taskCursor]
			a.confirmDeleteSection = true
		}
		return a, nil
	}

	return a, nil
}

// handleMoveTaskKeyMsg handles keyboard input for moving a task to a section.
func (a *App) handleMoveTaskKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.isMovingTask = false
		a.moveSectionCursor = 0
		return a, nil

	case "up", "k":
		if a.moveSectionCursor > 0 {
			a.moveSectionCursor--
		}
		return a, nil

	case "down", "j":
		if a.moveSectionCursor < len(a.sections)-1 {
			a.moveSectionCursor++
		}
		return a, nil

	case "enter":
		if len(a.sections) == 0 {
			a.isMovingTask = false
			return a, nil
		}

		// Get the correct task using ordered indices mapping
		var task *api.Task
		if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
			taskIndex := a.taskOrderedIndices[a.taskCursor]
			if taskIndex >= 0 && taskIndex < len(a.tasks) {
				task = &a.tasks[taskIndex]
			}
		} else if a.taskCursor < len(a.tasks) {
			task = &a.tasks[a.taskCursor]
		}

		if task == nil {
			a.isMovingTask = false
			a.statusMsg = "No task selected"
			return a, nil
		}

		sectionID := a.sections[a.moveSectionCursor].ID

		// Don't move if section ID is empty or if it's the same section
		if sectionID == "" {
			a.isMovingTask = false
			a.statusMsg = "Invalid section"
			return a, nil
		}
		if task.SectionID != nil && *task.SectionID == sectionID {
			a.isMovingTask = false
			a.statusMsg = "Task is already in this section"
			return a, nil
		}

		a.isMovingTask = false
		a.loading = true
		a.statusMsg = "Moving task..."

		return a, func() tea.Msg {
			// Update task with new section_id using MoveTask (Sync API)
			err := a.client.MoveTask(task.ID, &sectionID, nil, nil)
			if err != nil {
				return errMsg{err}
			}
			return taskUpdatedMsg{task: task} // We don't get the updated task back, but we trigger a refresh usually. Or should we reload?
			// taskUpdatedMsg usually expects a *Task. The old code returned taskUpdatedMsg{}.
			// Let's return taskUpdatedMsg{} or maybe trigger reload if structure changed.
			// Returning taskUpdatedMsg{} usually triggers list update.
		}

	}
	return a, nil
}

// handleCommentInputKeyMsg handles keyboard input for adding a comment.
func (a *App) handleCommentInputKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		a.isAddingComment = false
		a.commentInput.Reset()
		return a, nil

	case "enter":
		content := strings.TrimSpace(a.commentInput.Value())
		if content == "" {
			return a, nil
		}

		// determine task ID (from selection or cursor)
		taskID := ""
		if a.selectedTask != nil {
			taskID = a.selectedTask.ID
		} else if len(a.tasks) > 0 && a.taskCursor < len(a.tasks) {
			taskID = a.tasks[a.taskCursor].ID
		} else {
			a.isAddingComment = false
			return a, nil
		}

		a.isAddingComment = false
		a.commentInput.Reset()
		a.loading = true
		a.statusMsg = "Adding comment..."

		return a, func() tea.Msg {
			comment, err := a.client.CreateComment(api.CreateCommentRequest{
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
		a.commentInput, cmd = a.commentInput.Update(msg)
		return a, cmd
	}
}

// renderSections renders the sections management view.
func (a *App) renderSections() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Manage Sections"))
	b.WriteString("\n\n")

	if len(a.sections) == 0 {
		b.WriteString(styles.HelpDesc.Render("No sections found. Press 'a' to add one."))
		b.WriteString("\n\n")
		b.WriteString(styles.HelpDesc.Render("Esc: back"))
		return b.String()
	}

	// Render list
	for i, section := range a.sections {
		cursor := "  "
		style := lipgloss.NewStyle()

		if i == a.taskCursor {
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
func (a *App) handleAddSubtask() (tea.Model, tea.Cmd) {
	// Guard: Only in main pane with tasks
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		return a, nil
	}

	// Get the selected task using ordered indices
	taskIndex := a.taskCursor
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex = a.taskOrderedIndices[a.taskCursor]
	}
	if taskIndex < 0 || taskIndex >= len(a.tasks) {
		return a, nil
	}

	// Initialize subtask input
	a.parentTaskID = a.tasks[taskIndex].ID
	a.subtaskInput = textinput.New()
	a.subtaskInput.Placeholder = "Enter subtask..."
	a.subtaskInput.CharLimit = 200
	a.subtaskInput.Width = 50
	a.subtaskInput.Focus()
	a.isCreatingSubtask = true

	return a, nil
}

// handleSubtaskInputKeyMsg handles keyboard input during subtask creation.
func (a *App) handleSubtaskInputKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel subtask creation
		a.isCreatingSubtask = false
		a.parentTaskID = ""
		a.subtaskInput.Reset()
		return a, nil

	case "enter":
		// Submit new subtask
		content := strings.TrimSpace(a.subtaskInput.Value())
		if content == "" {
			a.isCreatingSubtask = false
			a.parentTaskID = ""
			a.subtaskInput.Reset()
			return a, nil
		}

		parentID := a.parentTaskID
		a.isCreatingSubtask = false
		a.parentTaskID = ""
		a.subtaskInput.Reset()
		a.loading = true

		return a, func() tea.Msg {
			_, err := a.client.CreateTask(api.CreateTaskRequest{
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
		a.subtaskInput, cmd = a.subtaskInput.Update(msg)
		return a, cmd
	}
}

// handleFormKeyMsg handles keyboard input when the form is active.
func (a *App) handleFormKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if a.taskForm == nil {
		return a, nil
	}

	switch msg.String() {
	case "esc":
		// Cancel form and go back
		a.currentView = a.previousView
		a.taskForm = nil
		return a, nil

	case "enter":
		// Submit on Enter if a text input is focused (and not in project selection dropdown)
		if !a.taskForm.showProjectList && a.taskForm.IsValid() &&
			(a.taskForm.FocusedField == FormFieldContent ||
				a.taskForm.FocusedField == FormFieldDescription ||
				a.taskForm.FocusedField == FormFieldDue ||
				a.taskForm.FocusedField == FormFieldLabels ||
				a.taskForm.FocusedField == FormFieldSubmit) {
			return a.submitForm()
		}

		// If on submit button or Ctrl+Enter from any field, submit form
		if a.taskForm.FocusedField == FormFieldSubmit || msg.String() == "ctrl+enter" {
			return a.submitForm()
		}
		// Otherwise, let form handle it (e.g., for project dropdown)
	}

	// Forward to form
	var cmd tea.Cmd
	a.taskForm, cmd = a.taskForm.Update(msg)
	return a, cmd
}

// submitForm submits the task form (create or update).
func (a *App) submitForm() (tea.Model, tea.Cmd) {
	if a.taskForm == nil || !a.taskForm.IsValid() {
		a.statusMsg = "Task name is required"
		return a, nil
	}

	a.loading = true

	if a.taskForm.Mode == "edit" {
		// Update existing task
		taskID := a.taskForm.TaskID
		req := a.taskForm.ToUpdateRequest()
		return a, func() tea.Msg {
			_, err := a.client.UpdateTask(taskID, req)
			if err != nil {
				return errMsg{err}
			}
			return taskCreatedMsg{} // Reuse message type for refresh
		}
	}

	// Create new task
	req := a.taskForm.ToCreateRequest()
	return a, func() tea.Msg {
		_, err := a.client.CreateTask(req)
		if err != nil {
			return errMsg{err}
		}
		return taskCreatedMsg{}
	}
}

// handleSearch opens the search view.
func (a *App) handleSearch() (tea.Model, tea.Cmd) {
	a.previousView = a.currentView
	a.currentView = ViewSearch
	a.searchInput.Reset()
	a.searchInput.Focus()
	a.searchResults = nil
	a.searchQuery = ""
	a.taskCursor = 0
	return a, nil
}

// handleSearchKeyMsg handles keyboard input when search is active.
func (a *App) handleSearchKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Cancel search and go back
		a.currentView = a.previousView
		a.searchInput.Blur()
		a.searchResults = nil
		a.searchQuery = ""
		return a, nil

	case "enter":
		// Select task from search results
		if len(a.searchResults) > 0 && a.taskCursor < len(a.searchResults) {
			a.selectedTask = &a.searchResults[a.taskCursor]
			a.previousView = ViewSearch
			a.currentView = ViewTaskDetail
			return a, nil
		}
		return a, nil

	case "down", "j":
		if a.taskCursor < len(a.searchResults)-1 {
			a.taskCursor++
		}
		return a, nil

	case "up", "k":
		if a.taskCursor > 0 {
			a.taskCursor--
		}
		return a, nil

	case "x":
		// Complete task from search results
		if len(a.searchResults) > 0 && a.taskCursor < len(a.searchResults) {
			task := &a.searchResults[a.taskCursor]
			a.loading = true
			return a, func() tea.Msg {
				var err error
				if task.Checked {
					err = a.client.ReopenTask(task.ID)
				} else {
					err = a.client.CloseTask(task.ID)
				}
				if err != nil {
					return errMsg{err}
				}
				return searchRefreshMsg{}
			}
		}
		return a, nil
	}

	// Update search input and filter results
	var cmd tea.Cmd
	a.searchInput, cmd = a.searchInput.Update(msg)
	a.searchQuery = a.searchInput.Value()
	a.filterSearchResults()
	a.taskCursor = 0 // Reset cursor when query changes
	return a, cmd
}

// filterSearchResults filters tasks based on search query.
func (a *App) filterSearchResults() {
	query := strings.ToLower(strings.TrimSpace(a.searchQuery))
	if query == "" {
		a.searchResults = nil
		return
	}

	var results []api.Task
	for _, task := range a.tasks {
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

	a.searchResults = results
}

// refreshSearchResults refreshes the search results after a task update.
func (a *App) refreshSearchResults() tea.Cmd {
	return func() tea.Msg {
		// Reload all tasks
		tasks, err := a.client.GetTasks(api.TaskFilter{})
		if err != nil {
			return errMsg{err}
		}
		a.tasks = tasks
		a.filterSearchResults()
		return dataLoadedMsg{tasks: tasks}
	}
}

// handlePriority sets task priority.
func (a *App) handlePriority(action string) (tea.Model, tea.Cmd) {
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		return a, nil
	}

	// Use ordered indices if available
	taskIndex := a.taskCursor
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex = a.taskOrderedIndices[a.taskCursor]
	}
	if taskIndex < 0 || taskIndex >= len(a.tasks) {
		return a, nil
	}

	task := &a.tasks[taskIndex]
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

	a.loading = true
	return a, func() tea.Msg {
		_, err := a.client.UpdateTask(task.ID, api.UpdateTaskRequest{
			Priority: &priority,
		})
		if err != nil {
			return errMsg{err}
		}
		return taskUpdatedMsg{}
	}
}

// handleDueToday sets the task due date to today.
func (a *App) handleDueToday() (tea.Model, tea.Cmd) {
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		return a, nil
	}

	// Guard: Don't operate when viewing label list
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	// Use ordered indices if available
	taskIndex := a.taskCursor
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex = a.taskOrderedIndices[a.taskCursor]
	}
	if taskIndex < 0 || taskIndex >= len(a.tasks) {
		return a, nil
	}

	task := &a.tasks[taskIndex]
	dueString := "today"

	a.loading = true
	a.statusMsg = "Moving to today..."
	return a, func() tea.Msg {
		_, err := a.client.UpdateTask(task.ID, api.UpdateTaskRequest{
			DueString: &dueString,
		})
		if err != nil {
			return errMsg{err}
		}
		return taskUpdatedMsg{}
	}
}

// handleDueTomorrow sets the task due date to tomorrow.
func (a *App) handleDueTomorrow() (tea.Model, tea.Cmd) {
	if a.focusedPane != PaneMain || len(a.tasks) == 0 {
		return a, nil
	}

	// Guard: Don't operate when viewing label list
	if a.currentView == ViewLabels && a.currentLabel == nil {
		return a, nil
	}

	// Use ordered indices if available
	taskIndex := a.taskCursor
	if len(a.taskOrderedIndices) > 0 && a.taskCursor < len(a.taskOrderedIndices) {
		taskIndex = a.taskOrderedIndices[a.taskCursor]
	}
	if taskIndex < 0 || taskIndex >= len(a.tasks) {
		return a, nil
	}

	task := &a.tasks[taskIndex]
	dueString := "tomorrow"

	a.loading = true
	a.statusMsg = "Moving to tomorrow..."
	return a, func() tea.Msg {
		_, err := a.client.UpdateTask(task.ID, api.UpdateTaskRequest{
			DueString: &dueString,
		})
		if err != nil {
			return errMsg{err}
		}
		return taskUpdatedMsg{}
	}
}

// loadProjectTasks loads tasks for a specific project.
func (a *App) loadProjectTasks(projectID string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.GetTasks(api.TaskFilter{
			ProjectID: projectID,
		})
		if err != nil {
			return errMsg{err}
		}

		sections, err := a.client.GetSections(projectID)
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
func (a *App) filterProjectTasks(projectID string) tea.Cmd {
	var tasks []api.Task
	for _, t := range a.allTasks {
		if t.ProjectID == projectID {
			tasks = append(tasks, t)
		}
	}

	var sections []api.Section
	for _, s := range a.allSections {
		if s.ProjectID == projectID {
			sections = append(sections, s)
		}
	}

	// Sort sections
	sort.Slice(sections, func(i, j int) bool {
		return sections[i].SectionOrder < sections[j].SectionOrder
	})

	a.tasks = tasks
	a.sections = sections
	a.sortTasks()
	return nil
}

// refreshTasks refreshes the current task list.
func (a *App) refreshTasks() tea.Cmd {
	return func() tea.Msg {
		// Always fetch all tasks to keep the cache fresh
		allTasks, err := a.client.GetTasks(api.TaskFilter{})
		if err != nil {
			return errMsg{err}
		}

		var filteredTasks []api.Task
		if a.currentView == ViewProject && a.currentProject != nil {
			for _, t := range allTasks {
				if t.ProjectID == a.currentProject.ID {
					filteredTasks = append(filteredTasks, t)
				}
			}
		} else if a.currentView == ViewLabels && a.currentLabel != nil {
			for _, t := range allTasks {
				for _, l := range t.Labels {
					if l == a.currentLabel.Name {
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
func (a *App) loadTodayTasks() tea.Cmd {
	return func() tea.Msg {
		// Use filter endpoint for today | overdue
		tasks, err := a.client.GetTasksByFilter("today | overdue")
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{tasks: tasks}
	}
}

// filterTodayTasks filters cached tasks for today/overdue.
func (a *App) filterTodayTasks() tea.Cmd {
	var tasks []api.Task
	for _, t := range a.allTasks {
		if t.IsOverdue() || t.IsDueToday() {
			tasks = append(tasks, t)
		}
	}
	a.tasks = tasks
	a.sortTasks()
	return nil
}

// loadUpcomingTasks loads all tasks with due dates for the upcoming view.
func (a *App) loadUpcomingTasks() tea.Cmd {
	return func() tea.Msg {
		// Get all tasks
		allTasks, err := a.client.GetTasks(api.TaskFilter{})
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
func (a *App) filterUpcomingTasks() tea.Cmd {
	var upcoming []api.Task
	for _, t := range a.allTasks {
		if t.Due != nil {
			upcoming = append(upcoming, t)
		}
	}
	a.tasks = upcoming
	a.sortTasks()
	return nil
}

// loadAllTasks loads all tasks (for calendar view).
func (a *App) loadAllTasks() tea.Cmd {
	return func() tea.Msg {
		allTasks, err := a.client.GetTasks(api.TaskFilter{})
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{allTasks: allTasks}
	}
}

// loadProjects loads all projects.
func (a *App) loadProjects() tea.Cmd {
	return func() tea.Msg {
		projects, err := a.client.GetProjects()
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{projects: projects}
	}
}

// loadLabels loads all labels.
func (a *App) loadLabels() tea.Cmd {
	return func() tea.Msg {
		labels, err := a.client.GetLabels()
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{labels: labels}
	}
}

// loadLabelTasks loads tasks filtered by a specific label.
func (a *App) loadLabelTasks(labelName string) tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.client.GetTasksByFilter("@" + labelName)
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{tasks: tasks}
	}
}

// filterLabelTasks filters cached tasks for a label.
func (a *App) filterLabelTasks(labelName string) tea.Cmd {
	var tasks []api.Task
	for _, t := range a.allTasks {
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
	a.tasks = tasks
	a.sortTasks()
	return nil
}

// loadCalendarDayTasks loads tasks for the selected calendar day.
func (a *App) loadCalendarDayTasks() tea.Cmd {
	selectedDate := time.Date(a.calendarDate.Year(), a.calendarDate.Month(), a.calendarDay, 0, 0, 0, 0, time.Local)
	dateStr := selectedDate.Format("2006-01-02")

	// Filter from allTasks instead of API call
	var dayTasks []api.Task
	for _, t := range a.allTasks {
		if t.Due != nil && t.Due.Date == dateStr {
			dayTasks = append(dayTasks, t)
		}
	}
	a.tasks = dayTasks
	return nil
}

// filterCalendarTasks filters cached tasks for calendar view (which uses allTasks).
func (a *App) filterCalendarTasks() tea.Cmd {
	// Calendar view mainly relies on a.allTasks which is already loaded.
	// But we might want to refresh specific day tasks if we switch to day view.
	// For main calendar view, we just need to ensured allTasks is available.
	// Since we keep allTasks updated, we just need to trigger a view update if needed.
	return nil
}

// loadTaskComments loads comments for the selected task.
func (a *App) loadTaskComments() tea.Cmd {
	if a.selectedTask == nil {
		return nil
	}
	taskID := a.selectedTask.ID
	return func() tea.Msg {
		comments, err := a.client.GetComments(taskID, "")
		if err != nil {
			return errMsg{err}
		}
		return commentsLoadedMsg{comments: comments}
	}
}

// buildSidebarItems constructs the sidebar items list with only projects (for Projects tab).
func (a *App) buildSidebarItems() {
	a.sidebarItems = []components.SidebarItem{}

	// Map projects for easy count lookup
	counts := make(map[string]int)
	for _, t := range a.allTasks {
		if !t.Checked && t.ProjectID != "" {
			counts[t.ProjectID]++
		}
	}

	// Add favorite projects first
	for _, p := range a.projects {
		if p.IsFavorite {
			icon := "⭐"
			if p.InboxProject {
				icon = "📥"
			}
			a.sidebarItems = append(a.sidebarItems, components.SidebarItem{
				Type:       "project",
				ID:         p.ID,
				Name:       p.Name,
				Icon:       icon,
				Count:      counts[p.ID],
				IsFavorite: true,
				ParentID:   p.ParentID,
			})
		}
	}

	// Add separator if there were favorites
	hasFavorites := false
	for _, p := range a.projects {
		if p.IsFavorite {
			hasFavorites = true
			break
		}
	}
	if hasFavorites {
		a.sidebarItems = append(a.sidebarItems, components.SidebarItem{Type: "separator", ID: "", Name: ""})
	}

	// Add remaining projects (non-favorites)
	for _, p := range a.projects {
		if !p.IsFavorite {
			icon := "📁"
			if p.InboxProject {
				icon = "📥"
			}
			a.sidebarItems = append(a.sidebarItems, components.SidebarItem{
				Type:     "project",
				ID:       p.ID,
				Name:     p.Name,
				Icon:     icon,
				Count:    counts[p.ID],
				ParentID: p.ParentID,
			})
		}
	}
}

// View implements tea.Model.
