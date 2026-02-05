// Package api provides a client for the Todoist API v1.
package api

import (
	"fmt"
	"strings"
	"time"
)

// Task represents a Todoist task.
type Task struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	ProjectID      string     `json:"project_id"`
	SectionID      *string    `json:"section_id"`
	ParentID       *string    `json:"parent_id"`
	Content        string     `json:"content"`
	Description    string     `json:"description"`
	Checked        bool       `json:"checked"`
	IsDeleted      bool       `json:"is_deleted"`
	IsCollapsed    bool       `json:"is_collapsed"`
	Labels         []string   `json:"labels"`
	ChildOrder     int        `json:"child_order"`
	DayOrder       int        `json:"day_order"`
	Priority       int        `json:"priority"`
	Due            *Due       `json:"due"`
	Deadline       *Deadline  `json:"deadline"`
	URL            string     `json:"url"`
	NoteCount      int        `json:"note_count"`
	AddedAt        string     `json:"added_at"`
	AddedByUID     string     `json:"added_by_uid"`
	AssignedByUID  *string    `json:"assigned_by_uid"`
	ResponsibleUID *string    `json:"responsible_uid"`
	CompletedAt    *string    `json:"completed_at"`
	CompletedByUID *string    `json:"completed_by_uid"`
	UpdatedAt      string     `json:"updated_at"`
	Duration       *Duration  `json:"duration"`
	ParsedDate     *time.Time `json:"-"`
}

// Due represents a task's due date information.
type Due struct {
	String      string  `json:"string"`
	Date        string  `json:"date"`
	IsRecurring bool    `json:"is_recurring"`
	Datetime    *string `json:"datetime"`
	Timezone    *string `json:"timezone"`
	Lang        string  `json:"lang"`
}

// Deadline represents a task's deadline (non-recurring, date-only).
type Deadline struct {
	Date string `json:"date"`
	Lang string `json:"lang"`
}

// Duration represents a task's duration.
type Duration struct {
	Amount int    `json:"amount"`
	Unit   string `json:"unit"` // "minute" or "day"
}

// Project represents a Todoist project.
type Project struct {
	ID                                  string  `json:"id"`
	Name                                string  `json:"name"`
	Description                         string  `json:"description"`
	WorkspaceID                         *string `json:"workspace_id"`
	Color                               string  `json:"color"`
	ParentID                            *string `json:"parent_id"`
	ChildOrder                          int     `json:"child_order"`
	IsCollapsed                         bool    `json:"is_collapsed"`
	Shared                              bool    `json:"shared"`
	CanAssignTasks                      bool    `json:"can_assign_tasks"`
	IsFavorite                          bool    `json:"is_favorite"`
	InboxProject                        bool    `json:"inbox_project"`
	IsInviteOnly                        bool    `json:"is_invite_only"`
	Status                              string  `json:"status"`
	IsLinkSharingEnabled                bool    `json:"is_link_sharing_enabled"`
	CollaboratorRoleDefault             string  `json:"collaborator_role_default"`
	IsDeleted                           bool    `json:"is_deleted"`
	IsArchived                          bool    `json:"is_archived"`
	IsFrozen                            bool    `json:"is_frozen"`
	ViewStyle                           string  `json:"view_style"`
	Role                                string  `json:"role"`
	FolderID                            *string `json:"folder_id"`
	CreatedAt                           string  `json:"created_at"`
	UpdatedAt                           string  `json:"updated_at"`
	IsPendingDefaultCollaboratorInvites bool    `json:"is_pending_default_collaborator_invites"`
}

// Section represents a project section.
type Section struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	ProjectID    string  `json:"project_id"`
	SectionOrder int     `json:"section_order"`
	IsCollapsed  bool    `json:"is_collapsed"`
	UserID       string  `json:"user_id"`
	IsDeleted    bool    `json:"is_deleted"`
	IsArchived   bool    `json:"is_archived"`
	ArchivedAt   *string `json:"archived_at"`
	AddedAt      string  `json:"added_at"`
	UpdatedAt    string  `json:"updated_at"`
}

// Label represents a personal label.
type Label struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Color      string `json:"color"`
	ItemOrder  int    `json:"item_order"`
	IsFavorite bool   `json:"is_favorite"`
	IsDeleted  bool   `json:"is_deleted"`
}

// Comment represents a task or project comment.
type Comment struct {
	ID             string              `json:"id"`
	ItemID         *string             `json:"item_id"`
	ProjectID      *string             `json:"project_id"`
	PostedUID      string              `json:"posted_uid"`
	PostedAt       string              `json:"posted_at"`
	Content        string              `json:"content"`
	FileAttachment *FileAttachment     `json:"file_attachment"`
	UIDsToNotify   []string            `json:"uids_to_notify"`
	IsDeleted      bool                `json:"is_deleted"`
	Reactions      map[string][]string `json:"reactions"`
}

// FileAttachment represents a file attachment in a comment.
type FileAttachment struct {
	FileName    string `json:"file_name"`
	FileType    string `json:"file_type"`
	FileSize    int    `json:"file_size"`
	FileURL     string `json:"file_url"`
	UploadState string `json:"upload_state"`
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
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
	Color       *string `json:"color,omitempty"`
	IsFavorite  *bool   `json:"is_favorite,omitempty"`
	ViewStyle   *string `json:"view_style,omitempty"`
}

// CreateSectionRequest represents the request body for creating a section.
type CreateSectionRequest struct {
	Name      string `json:"name"`
	ProjectID string `json:"project_id"`
	Order     int    `json:"order,omitempty"`
}

// UpdateSectionRequest represents the request body for updating a section.
type UpdateSectionRequest struct {
	Name  string `json:"name,omitempty"`
	Order *int   `json:"section_order,omitempty"`
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

// Location returns the time location for the task's due date.
func (t *Task) Location() *time.Location {
	if t.Due != nil && t.Due.Timezone != nil && *t.Due.Timezone != "" {
		if loc, err := time.LoadLocation(*t.Due.Timezone); err == nil {
			return loc
		}
	}
	return time.Local
}

// IsOverdue returns true if the task is overdue (including past due time today).
func (t *Task) IsOverdue() bool {
	if t.Due == nil || t.Checked {
		return false
	}

	loc := t.Location()
	var dueDate time.Time
	var hasTime bool

	if t.ParsedDate != nil {
		dueDate = t.ParsedDate.In(loc)
		hasTime = t.Due.Datetime != nil && *t.Due.Datetime != ""
		// Extra check: if someone put a timestamp in just the 'date' field
		if !hasTime && len(t.Due.Date) > 10 {
			hasTime = true
		}
	} else {
		// Fallback parsing (same as DueDisplay)
		dateStr := t.Due.Date
		if t.Due.Datetime != nil && *t.Due.Datetime != "" {
			if parsed, err := time.Parse(time.RFC3339, *t.Due.Datetime); err == nil {
				dueDate = parsed.In(loc)
				hasTime = true
			}
		}
		if !hasTime {
			if parsed, err := time.Parse(time.RFC3339, dateStr); err == nil {
				dueDate = parsed.In(loc)
				hasTime = true
			} else if parsed, err := time.ParseInLocation("2006-01-02T15:04:05", dateStr, loc); err == nil {
				dueDate = parsed
				hasTime = true
			} else {
				if len(dateStr) > 10 {
					dateStr = dateStr[:10]
				}
				if parsed, err := time.ParseInLocation("2006-01-02", dateStr, loc); err == nil {
					dueDate = parsed
				} else {
					return false
				}
			}
		}
	}

	now := time.Now().In(loc)

	if hasTime {
		// Specific time: overdue if current time is after due time
		return now.After(dueDate)
	}

	// Date only: overdue if it was due before today's start
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	return dueDate.Before(todayStart)
}

// IsDueToday returns true if the task is due today.
func (t *Task) IsDueToday() bool {
	if t.Due == nil {
		return false
	}

	loc := t.Location()
	var dueDate time.Time
	var err error

	if t.ParsedDate != nil {
		dueDate = t.ParsedDate.In(loc)
	} else {
		dueDate, err = time.ParseInLocation("2006-01-02", t.Due.Date[:10], loc)
		if err != nil {
			return false
		}
	}

	today := time.Now().In(loc)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)
	return dueDate.Year() == today.Year() && dueDate.Month() == today.Month() && dueDate.Day() == today.Day()
}

// DueDisplay returns a human-readable due date string, including time if available.
func (t *Task) DueDisplay() string {
	if t.Due == nil {
		return ""
	}

	loc := t.Location()
	var dueDate time.Time
	var err error

	hasTime := t.Due.Datetime != nil && *t.Due.Datetime != ""

	if t.ParsedDate != nil {
		dueDate = t.ParsedDate.In(loc)
	} else {
		// Use Datetime if available for more precision
		if hasTime {
			dueDate, err = time.Parse(time.RFC3339, *t.Due.Datetime)
			if err == nil {
				dueDate = dueDate.In(loc)
			}
		}

		if err != nil || !hasTime {
			// Fallback to date field, which might also contain a timestamp in some API versions
			dateStr := t.Due.Date

			// Try to parse as full timestamp first
			if parsed, err := time.Parse(time.RFC3339, dateStr); err == nil {
				dueDate = parsed.In(loc)
				hasTime = true
			} else if parsed, err := time.ParseInLocation("2006-01-02T15:04:05", dateStr, loc); err == nil {
				dueDate = parsed
				hasTime = true
			} else {
				// Fallback to date only
				if len(dateStr) > 10 {
					dateStr = dateStr[:10]
				}
				dueDate, err = time.ParseInLocation("2006-01-02", dateStr, loc)
				if err != nil {
					return t.Due.String
				}
			}
		}
	}

	today := time.Now().In(loc)
	today = time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, loc)

	// Calculate difference in days based on midnight of both dates
	dueDay := time.Date(dueDate.Year(), dueDate.Month(), dueDate.Day(), 0, 0, 0, 0, loc)
	diff := int(dueDay.Sub(today).Hours() / 24)

	var display string
	switch {
	case diff < -1:
		display = fmt.Sprintf("%d days ago", -diff)
	case diff == -1:
		display = "yesterday"
	case diff == 0:
		display = "today"
	case diff == 1:
		display = "tomorrow"
	case diff < 7:
		display = dueDate.Weekday().String()
	default:
		display = dueDate.Format("Jan 2")
	}

	// Append time if available
	if hasTime {
		display += " " + dueDate.Format("3:04pm")
	} else if t.Due.String != "" {
		// Fallback: If raw string contains a time but API didn't provide datetime (rare but possible),
		// or if the user wants to see the specific string they typed which includes a time.
		lowerStr := strings.ToLower(t.Due.String)
		if strings.Contains(lowerStr, ":") || strings.Contains(lowerStr, "am") || strings.Contains(lowerStr, "pm") || strings.Contains(lowerStr, "at ") {
			// If it's a "today" task, we can try to just use the raw string if it's informative
			// but cleaning it up a bit if it's like "today at 10pm"
			if diff == 0 {
				display = t.Due.String
			} else {
				// For other days, maybe append the time portion?
				// For now, just use it.
				display = t.Due.String
			}
		}
	}

	return display
}

// ProductivityStats represents the user's productivity statistics.
type ProductivityStats struct {
	Goals      ProductivityGoals `json:"goals"`
	DaysItems  []DayItems        `json:"days_items"`
	WeekItems  []WeekItems       `json:"week_items"`
	Karma      int               `json:"karma"`
	KarmaTrend string            `json:"karma_trend"` // "up", "down", or ""
}

// ProductivityGoals represents the user's configured goals.
type ProductivityGoals struct {
	DailyGoal          int `json:"daily_goal"`
	WeeklyGoal         int `json:"weekly_goal"`
	CurrentDailyStreak struct {
		Count int    `json:"count"`
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"current_daily_streak"`
	CurrentWeeklyStreak struct {
		Count int    `json:"count"`
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"current_weekly_streak"`
	MaxDailyStreak struct {
		Count int    `json:"count"`
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"max_daily_streak"`
	MaxWeeklyStreak struct {
		Count int    `json:"count"`
		Start string `json:"start"`
		End   string `json:"end"`
	} `json:"max_weekly_streak"`
	IgnoreDays    []int `json:"ignore_days"`
	VacationMode  int   `json:"vacation_mode"`
	KarmaDisabled int   `json:"karma_disabled"`
}

// DayItems represents completed items for a specific day.
type DayItems struct {
	Date           string `json:"date"`
	TotalCompleted int    `json:"total_completed"`
}

// WeekItems represents completed items for a specific week.
type WeekItems struct {
	Date           string `json:"date"`
	TotalCompleted int    `json:"total_completed"`
}
