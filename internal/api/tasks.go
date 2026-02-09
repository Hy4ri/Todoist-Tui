package api

import (
	"fmt"
	"net/url"
)

// GetTasks returns all active tasks, optionally filtered by project/section/label.
// Note: The Filter field is NOT supported in v1 API on /tasks endpoint.
// Use GetTasksByFilter for filter-based queries (e.g., "today | overdue").
// Handles v1 API pagination automatically, fetching all pages.
func (c *Client) GetTasks(filter TaskFilter) ([]Task, error) {
	allTasks := make([]Task, 0)
	query := buildFilterQuery(filter)

	for {
		var response PaginatedResponse[Task]
		if err := c.GetWithQuery("/tasks", query, &response); err != nil {
			return nil, fmt.Errorf("failed to get tasks: %w", err)
		}

		allTasks = append(allTasks, response.Results...)

		if response.NextCursor == nil || *response.NextCursor == "" {
			break
		}
		query.Set("cursor", *response.NextCursor)
	}

	return allTasks, nil
}

// GetTasksByFilter returns tasks matching a Todoist filter query.
// This uses the v1 API /tasks/filter endpoint.
// Examples: "today", "today | overdue", "@labelname", "2024-01-22"
// Handles pagination automatically, fetching all pages.
func (c *Client) GetTasksByFilter(filterQuery string) ([]Task, error) {
	if filterQuery == "" {
		return nil, fmt.Errorf("filter query cannot be empty")
	}

	allTasks := make([]Task, 0)
	query := url.Values{}
	query.Set("query", filterQuery)

	for {
		var response PaginatedResponse[Task]
		if err := c.GetWithQuery("/tasks/filter", query, &response); err != nil {
			return nil, fmt.Errorf("failed to get filtered tasks: %w", err)
		}

		allTasks = append(allTasks, response.Results...)

		if response.NextCursor == nil || *response.NextCursor == "" {
			break
		}
		query.Set("cursor", *response.NextCursor)
	}

	return allTasks, nil
}

// GetTask returns a single task by ID.
func (c *Client) GetTask(id string) (*Task, error) {
	var task Task
	if err := c.Get("/tasks/"+id, &task); err != nil {
		return nil, fmt.Errorf("failed to get task %s: %w", id, err)
	}
	return &task, nil
}

// CreateTask creates a new task.
func (c *Client) CreateTask(req CreateTaskRequest) (*Task, error) {
	var task Task
	if err := c.Post("/tasks", req, &task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}
	return &task, nil
}

// UpdateTask updates an existing task.
func (c *Client) UpdateTask(id string, req UpdateTaskRequest) (*Task, error) {
	var task Task
	if err := c.Post("/tasks/"+id, req, &task); err != nil {
		return nil, fmt.Errorf("failed to update task %s: %w", id, err)
	}
	return &task, nil
}

// CloseTask marks a task as completed.
func (c *Client) CloseTask(id string) error {
	if err := c.Post("/tasks/"+id+"/close", nil, nil); err != nil {
		return fmt.Errorf("failed to close task %s: %w", id, err)
	}
	return nil
}

// ReopenTask marks a completed task as not completed.
func (c *Client) ReopenTask(id string) error {
	if err := c.Post("/tasks/"+id+"/reopen", nil, nil); err != nil {
		return fmt.Errorf("failed to reopen task %s: %w", id, err)
	}
	return nil
}

// DeleteTask deletes a task.
func (c *Client) DeleteTask(id string) error {
	if err := c.Delete("/tasks/" + id); err != nil {
		return fmt.Errorf("failed to delete task %s: %w", id, err)
	}
	return nil
}

// QuickAddTask creates a task using natural language parsing.
// Supports: dates ("tomorrow", "every monday"), priorities (p1-p4),
// labels (@label), projects (#project), assignees (+name).
// Example: "Buy milk tomorrow at 3pm @errands #Shopping p1"
func (c *Client) QuickAddTask(text string) (*Task, error) {
	var task Task
	req := map[string]string{"text": text}
	if err := c.Post("/tasks/quick", req, &task); err != nil {
		return nil, fmt.Errorf("quick add failed: %w", err)
	}
	return &task, nil
}

// GetProductivityStats returns the user's productivity statistics including goals.
func (c *Client) GetProductivityStats() (*ProductivityStats, error) {
	var stats ProductivityStats
	if err := c.Get("/tasks/completed/stats", &stats); err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}
	return &stats, nil
}

// MoveTask moves a task to a different section, parent, or project using V1 REST API.
func (c *Client) MoveTask(id string, sectionID *string, projectID *string, parentID *string) error {
	req := map[string]interface{}{}
	if sectionID != nil {
		req["section_id"] = *sectionID
	}
	if projectID != nil {
		req["project_id"] = *projectID
	}
	if parentID != nil {
		req["parent_id"] = *parentID
	}

	if err := c.Post("/tasks/"+id+"/move", req, nil); err != nil {
		return fmt.Errorf("failed to move task %s: %w", id, err)
	}
	return nil
}
