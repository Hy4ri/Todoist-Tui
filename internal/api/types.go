// Package api provides a client for the Todoist REST API v2.
package api

import (
	"fmt"
	"time"
)

// Task represents a Todoist task.
type Task struct {
	ID           string    `json:"id"`
	ProjectID    string    `json:"project_id"`
	SectionID    *string   `json:"section_id"`
	Content      string    `json:"content"`
	Description  string    `json:"description"`
	IsCompleted  bool      `json:"is_completed"`
	Labels       []string  `json:"labels"`
	ParentID     *string   `json:"parent_id"`
	Order        int       `json:"order"`
	Priority     int       `json:"priority"`
	Due          *Due      `json:"due"`
	URL          string    `json:"url"`
	CommentCount int       `json:"comment_count"`
	CreatedAt    string    `json:"created_at"`
	CreatorID    string    `json:"creator_id"`
	AssigneeID   *string   `json:"assignee_id"`
	AssignerID   *string   `json:"assigner_id"`
	Duration     *Duration `json:"duration"`
}

// Due represents a task's due date information.
type Due struct {
	String      string  `json:"string"`
	Date        string  `json:"date"`
	IsRecurring bool    `json:"is_recurring"`
	Datetime    *string `json:"datetime"`
	Timezone    *string `json:"timezone"`
}

// Duration represents a task's duration.
type Duration struct {
	Amount int    `json:"amount"`
	Unit   string `json:"unit"` // "minute" or "day"
}

// Project represents a Todoist project.
type Project struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Color          string  `json:"color"`
	ParentID       *string `json:"parent_id"`
	Order          int     `json:"order"`
	CommentCount   int     `json:"comment_count"`
	IsShared       bool    `json:"is_shared"`
	IsFavorite     bool    `json:"is_favorite"`
	IsInboxProject bool    `json:"is_inbox_project"`
	IsTeamInbox    bool    `json:"is_team_inbox"`
	ViewStyle      string  `json:"view_style"`
	URL            string  `json:"url"`
}

// Section represents a project section.
type Section struct {
	ID        string `json:"id"`
	ProjectID string `json:"project_id"`
	Order     int    `json:"order"`
	Name      string `json:"name"`
}

// Label represents a personal label.
type Label struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	Order      int    `json:"order"`
	IsFavorite bool   `json:"is_favorite"`
}

// Comment represents a task or project comment.
type Comment struct {
	ID         string      `json:"id"`
	TaskID     *string     `json:"task_id"`
	ProjectID  *string     `json:"project_id"`
	PostedAt   string      `json:"posted_at"`
	Content    string      `json:"content"`
	Attachment *Attachment `json:"attachment"`
}

// Attachment represents a file attachment in a comment.
type Attachment struct {
	FileName     string `json:"file_name"`
	FileType     string `json:"file_type"`
	FileURL      string `json:"file_url"`
	ResourceType string `json:"resource_type"`
}

// Collaborator represents a project collaborator.
type Collaborator struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CreateTaskRequest represents the request body for creating a task.
type CreateTaskRequest struct {
	Content      string   `json:"content"`
	Description  string   `json:"description,omitempty"`
	ProjectID    string   `json:"project_id,omitempty"`
	SectionID    string   `json:"section_id,omitempty"`
	ParentID     string   `json:"parent_id,omitempty"`
	Order        int      `json:"order,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	Priority     int      `json:"priority,omitempty"`
	DueString    string   `json:"due_string,omitempty"`
	DueDate      string   `json:"due_date,omitempty"`
	DueDatetime  string   `json:"due_datetime,omitempty"`
	DueLang      string   `json:"due_lang,omitempty"`
	AssigneeID   string   `json:"assignee_id,omitempty"`
	Duration     int      `json:"duration,omitempty"`
	DurationUnit string   `json:"duration_unit,omitempty"`
}

// UpdateTaskRequest represents the request body for updating a task.
type UpdateTaskRequest struct {
	Content      *string  `json:"content,omitempty"`
	Description  *string  `json:"description,omitempty"`
	ProjectID    *string  `json:"project_id,omitempty"`
	SectionID    *string  `json:"section_id,omitempty"`
	Labels       []string `json:"labels,omitempty"`
	Priority     *int     `json:"priority,omitempty"`
	DueString    *string  `json:"due_string,omitempty"`
	DueDate      *string  `json:"due_date,omitempty"`
	DueDatetime  *string  `json:"due_datetime,omitempty"`
	DueLang      *string  `json:"due_lang,omitempty"`
	AssigneeID   *string  `json:"assignee_id,omitempty"`
	Duration     *int     `json:"duration,omitempty"`
	DurationUnit *string  `json:"duration_unit,omitempty"`
}

// CreateProjectRequest represents the request body for creating a project.
type CreateProjectRequest struct {
	Name       string `json:"name"`
	ParentID   string `json:"parent_id,omitempty"`
	Color      string `json:"color,omitempty"`
	IsFavorite bool   `json:"is_favorite,omitempty"`
	ViewStyle  string `json:"view_style,omitempty"`
}

// UpdateProjectRequest represents the request body for updating a project.
type UpdateProjectRequest struct {
	Name       *string `json:"name,omitempty"`
	Color      *string `json:"color,omitempty"`
	IsFavorite *bool   `json:"is_favorite,omitempty"`
	ViewStyle  *string `json:"view_style,omitempty"`
}

// CreateSectionRequest represents the request body for creating a section.
type CreateSectionRequest struct {
	Name      string `json:"name"`
	ProjectID string `json:"project_id"`
	Order     int    `json:"order,omitempty"`
}

// UpdateSectionRequest represents the request body for updating a section.
type UpdateSectionRequest struct {
	Name string `json:"name"`
}

// CreateLabelRequest represents the request body for creating a label.
type CreateLabelRequest struct {
	Name       string `json:"name"`
	Color      string `json:"color,omitempty"`
	Order      int    `json:"order,omitempty"`
	IsFavorite bool   `json:"is_favorite,omitempty"`
}

// UpdateLabelRequest represents the request body for updating a label.
type UpdateLabelRequest struct {
	Name       *string `json:"name,omitempty"`
	Color      *string `json:"color,omitempty"`
	Order      *int    `json:"order,omitempty"`
	IsFavorite *bool   `json:"is_favorite,omitempty"`
}

// CreateCommentRequest represents the request body for creating a comment.
type CreateCommentRequest struct {
	TaskID    string `json:"task_id,omitempty"`
	ProjectID string `json:"project_id,omitempty"`
	Content   string `json:"content"`
}

// UpdateCommentRequest represents the request body for updating a comment.
type UpdateCommentRequest struct {
	Content string `json:"content"`
}

// TaskFilter contains optional filters for listing tasks.
type TaskFilter struct {
	ProjectID string
	SectionID string
	Label     string
	Filter    string // Todoist filter query (e.g., "today", "overdue")
	Lang      string
	IDs       []string
}

// IsOverdue returns true if the task is overdue.
func (t *Task) IsOverdue() bool {
	if t.Due == nil || t.IsCompleted {
		return false
	}

	dueDate, err := time.Parse("2006-01-02", t.Due.Date)
	if err != nil {
		return false
	}

	today := time.Now().Truncate(24 * time.Hour)
	return dueDate.Before(today)
}

// IsDueToday returns true if the task is due today.
func (t *Task) IsDueToday() bool {
	if t.Due == nil {
		return false
	}

	dueDate, err := time.Parse("2006-01-02", t.Due.Date)
	if err != nil {
		return false
	}

	today := time.Now().Truncate(24 * time.Hour)
	return dueDate.Equal(today)
}

// DueDisplay returns a human-readable due date string.
func (t *Task) DueDisplay() string {
	if t.Due == nil {
		return ""
	}

	dueDate, err := time.Parse("2006-01-02", t.Due.Date)
	if err != nil {
		return t.Due.String
	}

	today := time.Now().Truncate(24 * time.Hour)
	diff := int(dueDate.Sub(today).Hours() / 24)

	switch {
	case diff < -1:
		return fmt.Sprintf("%d days ago", -diff)
	case diff == -1:
		return "yesterday"
	case diff == 0:
		return "today"
	case diff == 1:
		return "tomorrow"
	case diff < 7:
		return dueDate.Weekday().String()
	default:
		return dueDate.Format("Jan 2")
	}
}
