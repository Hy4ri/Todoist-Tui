package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
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
	quickAddURL := "https://api.todoist.com/sync/v9/quick/add"

	formData := url.Values{}
	formData.Set("text", text)

	req, err := http.NewRequest("POST", quickAddURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create quick add request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("quick add request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("quick add API error %d: %s", resp.StatusCode, string(body))
	}

	var task Task
	if err := json.NewDecoder(resp.Body).Decode(&task); err != nil {
		return nil, fmt.Errorf("failed to decode quick add response: %w", err)
	}

	return &task, nil
}

// MoveTask moves a task to a different section, parent, or project using Sync API.
func (c *Client) MoveTask(id string, sectionID *string, projectID *string, parentID *string) error {
	type moveArgs struct {
		ID        string  `json:"id"`
		ProjectID *string `json:"project_id,omitempty"`
		SectionID *string `json:"section_id,omitempty"`
		ParentID  *string `json:"parent_id,omitempty"`
	}

	type syncCommand struct {
		Type string   `json:"type"`
		UUID string   `json:"uuid"`
		Args moveArgs `json:"args"`
	}

	type syncRequest struct {
		Commands []syncCommand `json:"commands"`
	}

	cmd := syncCommand{
		Type: "item_move",
		UUID: fmt.Sprintf("%d", time.Now().UnixNano()),
		Args: moveArgs{
			ID:        id,
			ProjectID: projectID,
			SectionID: sectionID,
			ParentID:  parentID,
		},
	}

	reqBody := syncRequest{
		Commands: []syncCommand{cmd},
	}

	syncURL := "https://api.todoist.com/sync/v9/sync"

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal sync request: %w", err)
	}

	req, err := http.NewRequest("POST", syncURL, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create sync request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sync API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
