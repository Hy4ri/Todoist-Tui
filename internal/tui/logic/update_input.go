package logic

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// handleKeyMsg processes keyboard input.
func (h *Handler) handleKeyMsg(msg tea.KeyMsg) tea.Cmd {
	// Command line handling - Highest priority when active
	if h.CommandLine != nil && h.CommandLine.Active {
		return h.handleCommandLineKeyMsg(msg)
	}

	// Only ctrl+c is truly global
	if msg.String() == "ctrl+c" {
		return tea.Quit
	}

	// Activate command line
	if msg.String() == ":" && h.CurrentView != state.ViewTaskForm && h.CurrentView != state.ViewQuickAdd && h.CurrentView != state.ViewSearch && !h.IsEditingComment && !h.IsCreatingProject && !h.IsCreatingLabel && !h.IsCreatingSection && !h.IsCreatingSubtask {
		return h.activateCommandLine()
	}

	// If we're in help view, any key goes back
	if h.CurrentView == state.ViewHelp {
		h.CurrentView = h.PreviousView
		return nil
	}

	// Comment editing (Modal) - Check BEFORE view switching to capture input
	if h.IsEditingComment {
		return h.handleCommentEditKeyMsg(msg)
	}
	if h.ConfirmDeleteComment {
		return h.handleDeleteCommentConfirmKeyMsg(msg)
	}

	// Route key messages based on current view - BEFORE tab switching
	// This allows forms to capture number keys for text input
	switch h.CurrentView {
	case state.ViewTaskForm:
		return h.handleFormKeyMsg(msg)
	case state.ViewQuickAdd:
		return h.handleQuickAddKeyMsg(msg)
	case state.ViewSearch:
		return h.handleSearchKeyMsg(msg)
	case state.ViewTaskDetail:
		return h.handleTaskDetailKeyMsg(msg)
	}

	// Handle all input/dialog states BEFORE tab switching
	// This prevents number keys from switching views during text entry

	// Project state handling
	if h.IsCreatingProject {
		return h.handleProjectInputKeyMsg(msg)
	}
	if h.IsEditingProject {
		return h.handleProjectEditKeyMsg(msg)
	}
	if h.ConfirmDeleteProject {
		return h.handleDeleteConfirmKeyMsg(msg)
	}

	// Label state handling
	if h.IsCreatingLabel {
		return h.handleLabelInputKeyMsg(msg)
	}
	if h.IsEditingLabel {
		return h.handleLabelEditKeyMsg(msg)
	}
	if h.ConfirmDeleteLabel {
		return h.handleLabelDeleteConfirmKeyMsg(msg)
	}

	// Section state handling
	if h.IsCreatingSection {
		return h.handleSectionInputKeyMsg(msg)
	}
	if h.IsEditingSection {
		return h.handleSectionEditKeyMsg(msg)
	}
	if h.ConfirmDeleteSection {
		return h.handleSectionDeleteConfirmKeyMsg(msg)
	}

	// Filter state handling
	if h.IsCreatingFilter || h.IsEditingFilter {
		return h.handleFilterFormKeyMsg(msg)
	}
	if h.ConfirmDeleteFilter {
		return h.handleDeleteFilterConfirmKeyMsg(msg)
	}

	// Subtask creation handling
	if h.IsCreatingSubtask {
		return h.handleSubtaskInputKeyMsg(msg)
	}

	// Section task addition handling
	if h.IsAddingToSection {
		return h.handleSectionAddInputKeyMsg(msg)
	}

	// Comment input handling
	if h.IsAddingComment {
		return h.handleCommentInputKeyMsg(msg)
	}

	// Move task handling
	if h.IsMovingTask {
		return h.handleMoveTaskKeyMsg(msg)
	}

	// Move to project handling
	if h.IsMovingToProject {
		return h.handleMoveToProjectInput(msg)
	}

	// Indent task picker handling
	if h.IsIndentingTask {
		return h.handleIndentInputKeyMsg(msg)
	}

	// Reschedule handling
	if h.IsRescheduling {
		switch msg.String() {
		case "esc":
			h.IsRescheduling = false
			return nil
		case "j", "down":
			if h.RescheduleCursor < len(h.RescheduleOptions)-1 {
				h.RescheduleCursor++
			}
			return nil
		case "k", "up":
			if h.RescheduleCursor > 0 {
				h.RescheduleCursor--
			}
			return nil
		case "enter":
			if h.RescheduleCursor >= 0 && h.RescheduleCursor < len(h.RescheduleOptions) {
				return h.handleReschedule(h.RescheduleOptions[h.RescheduleCursor])
			}
			return nil
		}
		// Block other input while rescheduling
		return nil
	}

	// Tab switching with number keys (1-9) - only when not in form/input modes
	switch msg.String() {
	case "1":
		return h.switchToTab(state.TabInbox)
	case "2":
		return h.switchToTab(state.TabToday)
	case "3":
		return h.switchToTab(state.TabUpcoming)
	case "4":
		return h.switchToTab(state.TabLabels)
	case "5":
		return h.switchToTab(state.TabFilters)
	case "6":
		return h.switchToTab(state.TabCalendar)
	case "7":
		return h.switchToTab(state.TabProjects)
	case "8":
		return h.switchToTab(state.TabCompleted)
	case "9":
		return h.switchToTab(state.TabPomodoro)
	case "D": // Shift+d
		return h.setDefaultView()
	}

	// Pomodoro view — handle keys BEFORE global keymap steals them
	if h.CurrentView == state.ViewPomodoro {
		cmd, consumed := h.coordinator.HandleKey(msg)
		if consumed {
			return cmd
		}
	}

	// Sections view routing
	if h.CurrentView == state.ViewSections {
		return h.handleSectionsKeyMsg(msg)
	}

	// If we're in calendar view, handle calendar-specific keys
	if h.CurrentView == state.ViewCalendar && h.FocusedPane == state.PaneMain {
		return h.handleCalendarKeyMsg(msg)
	}

	// Filters Tab specific key handling
	if h.CurrentTab == state.TabFilters {
		// If searching, handle input
		if h.IsFilterSearch {
			switch msg.String() {
			case "enter":
				h.IsFilterSearch = false
				h.FilterInput.Blur()
				visible := h.getVisibleFilters()
				if len(visible) > 0 {
					return h.handleFilterSelect()
				}
				return nil
			case "esc":
				h.IsFilterSearch = false
				h.FilterInput.Blur()
				h.FilterInput.SetValue("")
				return nil
			}

			var cmd tea.Cmd
			h.FilterInput, cmd = h.FilterInput.Update(msg)
			return cmd
		}

		switch msg.String() {
		case "/":
			h.IsFilterSearch = true
			h.FilterInput.Focus()
			return textinput.Blink
		case "tab":
			if h.FocusedPane == state.PaneSidebar {
				h.FocusedPane = state.PaneMain
			} else {
				h.FocusedPane = state.PaneSidebar
			}
			return nil
		case "j", "down":
			if h.FocusedPane == state.PaneSidebar {
				h.moveFilterCursor(1)
				return nil
			}
		case "k", "up":
			if h.FocusedPane == state.PaneSidebar {
				h.moveFilterCursor(-1)
				return nil
			}
		case "enter":
			if h.FocusedPane == state.PaneSidebar {
				return h.handleFilterSelect()
			}
		case "n":
			if h.FocusedPane == state.PaneSidebar {
				return h.handleNewFilter()
			}
		case "d":
			if h.FocusedPane == state.PaneSidebar {
				return h.handleDeleteFilter()
			}
		}
	}

	// Process key through keymap
	action, consumed := h.KeyState.HandleKey(msg, h.Keymap)
	if !consumed {
		return nil
	}

	// Handle actions
	switch action {
	case "quit":
		return tea.Quit
	case "help":
		h.PreviousView = h.CurrentView
		h.CurrentView = state.ViewHelp
		return nil
	case "refresh":
		return func() tea.Msg { return refreshMsg{Force: true} }
	case "up":
		h.moveCursor(-1)
	case "down":
		h.moveCursor(1)
	case "top":
		h.moveCursorTo(0)
	case "bottom":
		h.moveCursorToEnd()
	case "half_up":
		h.moveCursor(-10)
	case "half_down":
		h.moveCursor(10)
	case "left":
		// h key - move to sidebar in Projects/Filters tab
		if (h.CurrentTab == state.TabProjects || h.CurrentTab == state.TabFilters) && h.FocusedPane == state.PaneMain {
			h.FocusedPane = state.PaneSidebar
		}
	case "right":
		// l key - move to main pane in Projects/Filters tab
		if (h.CurrentTab == state.TabProjects || h.CurrentTab == state.TabFilters) && h.FocusedPane == state.PaneSidebar {
			h.FocusedPane = state.PaneMain
		}
	case "switch_pane":
		h.switchPane()
	case "select":
		return h.handleSelect()
	case "back":
		return h.handleBack()
	case "complete":
		return h.handleComplete()
	case "delete":
		return h.handleDelete()
	case "add":
		// Context-aware add for Labels tab
		if h.CurrentTab == state.TabLabels && h.CurrentLabel == nil {
			return h.handleNewLabel()
		}
		// Specialized section-aware add for Project/Inbox views
		if h.CurrentView == state.ViewProject || h.CurrentView == state.ViewInbox {
			return h.handleSectionAdd()
		}
		return h.handleAdd()
	case "add_full":
		return h.handleAddTaskFull()
	case "edit":
		return h.handleEdit()
	case "search":
		return h.handleSearch()
	case "priority1", "priority2", "priority3", "priority4":
		return h.handlePriority(action)
	case "due_tomorrow":
		return h.handleDueTomorrow()
	case "move_task_prev_day":
		return h.handleMoveTaskDate(-1, "")
	case "move_task_next_day":
		return h.handleMoveTaskDate(1, "")
	case "indent":
		return h.handleIndent()
	case "outdent":
		return h.handleOutdent()
	case "move_to_project":
		return h.handleMoveToProject()
	case "send_to_pomodoro":
		return h.handleSendTaskToPomodoro()
	case "new_project":
		// 'n' key creates project or label depending on current tab
		if h.CurrentTab == state.TabProjects {
			return h.handleNewProject()
		} else if h.CurrentTab == state.TabLabels {
			return h.handleNewLabel()
		}
	case "toggle_favorite":
		if h.CurrentTab == state.TabProjects && h.FocusedPane == state.PaneSidebar {
			return h.handleToggleFavorite()
		}
	// Tab shortcuts (Shift + letter)
	case "tab_today":
		return h.switchToTab(state.TabToday)
	case "tab_upcoming":
		return h.switchToTab(state.TabUpcoming)
	case "tab_projects":
		return h.switchToTab(state.TabProjects)
	case "tab_labels":
		return h.switchToTab(state.TabLabels)
	case "tab_calendar":
		return h.switchToTab(state.TabCalendar)
	case "toggle_hints":
		h.ShowHints = !h.ShowHints
	case "add_subtask":
		return h.handleAddSubtask()
	case "undo":
		return h.handleUndo()
	case "manage_sections":
		if h.CurrentTab == state.TabProjects && len(h.Projects) > 0 {
			h.PreviousView = h.CurrentView
			h.CurrentView = state.ViewSections
			h.TaskCursor = 0 // use cursor for sections
			return nil
		}
	case "move_task":
		if h.FocusedPane == state.PaneMain && len(h.Tasks) > 0 && len(h.Sections) > 0 {
			h.IsMovingTask = true
			h.MoveSectionCursor = 0
			return nil
		}
	case "new_section":
		// Allow creating sections in project view when a project is selected
		if h.CurrentTab == state.TabProjects && h.CurrentProject != nil {
			h.SectionInput = textinput.New()
			h.SectionInput.Placeholder = "Enter section name..."
			h.SectionInput.CharLimit = 100
			h.SectionInput.Width = 40
			h.SectionInput.Focus()
			h.IsCreatingSection = true
			return nil
		}
	case "move_section":
		// Redirect to section management view
		if h.CurrentTab == state.TabProjects && h.CurrentProject != nil && len(h.Sections) > 1 {
			h.StatusMsg = "Use 'S' to manage sections - select with Space, reorder with Shift+j/k"
			return nil
		}
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
	case "toggle_select":
		return h.handleToggleSelect()
	case "copy":
		return h.handleCopy()
	case "reschedule":
		if h.FocusedPane == state.PaneMain && len(h.Tasks) > 0 {
			h.IsRescheduling = true
			h.RescheduleCursor = 0
			h.RescheduleOptions = []string{
				"Today",
				"Tomorrow",
				"Next Week (Mon)",
				"Weekend (Sat)",
				"Postpone (1 day)",
				"No Date",
			}
			return nil
		}
	}

	return nil
}
