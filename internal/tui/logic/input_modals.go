package logic

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// handleFormKeyMsg handles keyboard input when the task form is active.
func (h *Handler) handleFormKeyMsg(msg tea.KeyMsg) tea.Cmd {
	if h.TaskForm == nil {
		return nil
	}

	switch msg.String() {
	case "esc":
		// Cancel form and go back
		h.CurrentView = h.PreviousView
		h.TaskForm = nil
		return nil

	case "ctrl+enter":
		if h.Loading {
			return nil
		}
		return h.submitForm()

	case "enter":
		if h.Loading {
			return nil
		}

		// If on submit button, submit form
		if h.TaskForm.FocusIndex == state.FormFieldSubmit {
			return h.submitForm()
		}
		// Otherwise, let form handle enter (e.g. for opening project list)
	}

	// Forward to form
	return h.TaskForm.Update(msg)
}

// handleQuickAddKeyMsg handles keyboard input for the Quick Add popup.
func (h *Handler) handleQuickAddKeyMsg(msg tea.KeyMsg) tea.Cmd {
	if h.QuickAddForm == nil {
		return nil
	}

	switch msg.String() {
	case "esc":
		// Close Quick Add and return to previous view
		h.CurrentView = h.PreviousView
		h.QuickAddForm = nil
		return nil

	case "enter":
		// Submit task if there's content
		if !h.QuickAddForm.IsValid() {
			h.StatusMsg = "Enter task content"
			return nil
		}

		// Capture form values
		content := h.QuickAddForm.Value()
		projectID := h.QuickAddForm.ProjectID
		sectionID := h.QuickAddForm.SectionID

		// Clear input and increment count (stays open)
		h.QuickAddForm.Clear()
		h.QuickAddForm.IncrementCount()

		// Re-populate "today " prefix when adding from the Today view
		if h.CurrentTab == state.TabToday {
			h.QuickAddForm.Input.SetValue("today ")
			h.QuickAddForm.Input.SetCursor(6)
		}

		h.StatusMsg = "Adding task..."

		// Create task in background using Quick Add API
		return func() tea.Msg {
			// Send clean text to QuickAddTask — let it handle dates, priorities, labels
			// Do NOT append #ProjectName (fails with spaces in project names)
			task, err := h.Client.QuickAddTask(content)
			if err != nil {
				return errMsg{err}
			}

			// If we have a known project context and the task ended up elsewhere
			// (typically Inbox), move it using the dedicated move endpoint
			if projectID != "" && task.ProjectID != projectID {
				var secPtr *string
				if sectionID != "" {
					secPtr = &sectionID
				}
				if err := h.Client.MoveTask(task.ID, secPtr, &projectID, nil); err != nil {
					return errMsg{err}
				}
			} else if sectionID != "" && (task.SectionID == nil || *task.SectionID != sectionID) {
				// Same project but wrong section
				if err := h.Client.MoveTask(task.ID, &sectionID, nil, nil); err != nil {
					return errMsg{err}
				}
			}

			return quickAddTaskCreatedMsg{}
		}
	}

	// Forward other keys to the form input
	return h.QuickAddForm.Update(msg)
}

// handleCommentEditKeyMsg handles keys for comment editing.
func (h *Handler) handleCommentEditKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsEditingComment = false
		h.EditingComment = nil
		h.CommentInput.Reset()
		return nil
	case "ctrl+enter":
		content := h.CommentInput.Value()
		if content == "" {
			return nil
		}
		h.IsEditingComment = false
		h.Loading = true
		h.StatusMsg = "Updating comment..."
		commentID := h.EditingComment.ID
		h.EditingComment = nil
		h.CommentInput.Reset()

		return func() tea.Msg {
			c, err := h.Client.UpdateComment(commentID, api.UpdateCommentRequest{Content: content})
			if err != nil {
				return errMsg{err}
			}
			return commentUpdatedMsg{comment: c}
		}
	}
	var cmd tea.Cmd
	h.CommentInput, cmd = h.CommentInput.Update(msg)
	return cmd
}

// handleDeleteCommentConfirmKeyMsg handles confirmation for comment deletion.
func (h *Handler) handleDeleteCommentConfirmKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		h.ConfirmDeleteComment = false
		h.Loading = true
		h.StatusMsg = "Deleting comment..."
		commentID := h.EditingComment.ID
		h.EditingComment = nil

		return func() tea.Msg {
			err := h.Client.DeleteComment(commentID)
			if err != nil {
				return errMsg{err}
			}
			return commentDeletedMsg{id: commentID}
		}
	case "n", "N", "esc":
		h.ConfirmDeleteComment = false
		h.EditingComment = nil
		return nil
	}
	return nil
}

// handleCommentInputKeyMsg handles keys for adding a comment.
func (h *Handler) handleCommentInputKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsAddingComment = false
		h.CommentInput.Reset()
		return nil

	case "ctrl+enter":
		content := strings.TrimSpace(h.CommentInput.Value())
		if content == "" {
			return nil
		}

		// determine task ID (from selection or cursor)
		taskID := ""
		if h.SelectedTask != nil {
			taskID = h.SelectedTask.ID
		} else if t := h.getSelectedTask(); t != nil {
			taskID = t.ID
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

// handleIndentInputKeyMsg processes input for the indent picker.
func (h *Handler) handleIndentInputKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsIndentingTask = false
		h.IndentCandidates = nil
		h.IndentInput.Blur()
		return nil
	case "enter":
		return h.handleIndentSelect()
	case "up", "k":
		if h.IndentCursor > 0 {
			h.IndentCursor--
		}
		return nil
	case "down", "j":
		if h.IndentCursor < len(h.IndentCandidates)-1 {
			h.IndentCursor++
		}
		return nil
	}

	// Handle input
	var cmd tea.Cmd
	h.IndentInput, cmd = h.IndentInput.Update(msg)

	// Update filter
	h.updateIndentFilter()

	return cmd
}

// handleReminderInputKeyMsg handles keys for reminder input.
func (h *Handler) handleReminderInputKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsAddingReminder = false
		h.IsEditingReminder = false
		h.StatusMsg = "Cancelled"
		return nil
	case "tab":
		return h.handleReminderTypeToggle()
	case "enter":
		return h.submitReminderForm()
	}

	var cmd tea.Cmd
	if h.ReminderTypeCursor == 0 {
		h.ReminderMinuteInput, cmd = h.ReminderMinuteInput.Update(msg)
	} else {
		var cmd2 tea.Cmd
		h.ReminderDateInput, cmd = h.ReminderDateInput.Update(msg)
		h.ReminderTimeInput, cmd2 = h.ReminderTimeInput.Update(msg)
		cmd = tea.Batch(cmd, cmd2)
	}
	return cmd
}

// handleTaskDetailKeyMsg handles keys for task detail view.
func (h *Handler) handleTaskDetailKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// If any modal state is active, let the specific handler deal with it
	if h.IsEditingComment || h.ConfirmDeleteComment {
		return nil
	}

	// Handle reminder input
	if h.IsAddingReminder || h.IsEditingReminder {
		return h.handleReminderInputKeyMsg(msg)
	}

	// Check for global actions relevant to detail view
	action, consumed := h.KeyState.HandleKey(msg, h.Keymap)

	// Handle explicit ESC if not consumed by keymap (or if keymap maps ESC to back)
	if msg.String() == "esc" || action == "back" {
		return h.handleBack()
	}

	if consumed {
		switch action {
		case "quit":
			return tea.Quit
		case "add":
			return h.handleAdd()
		case "add_full":
			return h.handleAddTaskFull()
		case "add_subtask":
			return h.handleAddSubtask()
		case "add_comment":
			if h.SelectedTask != nil {
				h.IsAddingComment = true
				h.CommentInput = textarea.New()
				h.CommentInput.Placeholder = "Write a comment..."
				h.CommentInput.Focus()
				h.CommentInput.SetWidth(50)
				h.CommentInput.SetHeight(3)
				h.CommentInput.ShowLineNumbers = false
				h.CommentInput.Prompt = ""
				return nil
			}
		case "edit":
			return h.handleEdit()
		case "delete":
			return h.handleDelete()
		case "complete":
			return h.handleComplete()
		case "priority1", "priority2", "priority3", "priority4":
			return h.handlePriority(action)
		case "due_tomorrow":
			return h.handleDueTomorrow()
		case "move_task_prev_day":
			return h.handleMoveTaskDate(-1, "")
		case "move_task_next_day":
			return h.handleMoveTaskDate(1, "")
		}
	}

	// Delegate to component (handles j/k scrolling, etc.)
	_, cmd := h.DetailComp.Update(msg)
	return cmd
}
