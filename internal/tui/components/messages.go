package components

import "github.com/hy4ri/todoist-tui/internal/api"

// ProjectSelectedMsg is emitted when a project is selected in the sidebar.
type ProjectSelectedMsg struct {
	ID   string
	Name string
}

// TaskSelectedMsg is emitted when a task is selected for detail view.
type TaskSelectedMsg struct {
	Task *api.Task
}

// ViewChangeRequestMsg is emitted when a component requests a view change.
type ViewChangeRequestMsg struct {
	View View
}

// FocusPaneMsg is emitted to request focus change between panes.
type FocusPaneMsg struct {
	Pane Pane
}

// CursorMovedMsg is emitted when cursor position changes.
type CursorMovedMsg struct {
	Position int
}

// RefreshRequestMsg is emitted when a component requests data refresh.
type RefreshRequestMsg struct{}

// EditCommentMsg is emitted to request editing a comment.
type EditCommentMsg struct {
	Comment *api.Comment
}

// DeleteCommentMsg is emitted to request deleting a comment.
type DeleteCommentMsg struct {
	CommentID string
}
