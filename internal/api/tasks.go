package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/uuid"
)

// GetTasks returns all active tasks, optionally filtered by project/section/label.
// Note: The Filter field is NOT supported in v1 API on /tasks endpoint.
// Use GetTasksByFilter for filter-based queries (e.g., "today | overdue").
func (c *Client) GetTasks(filter TaskFilter) ([]Task, error) {
	var allTasks []Task
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
func (c *Client) GetTasksByFilter(filterQuery string) ([]Task, error) {
	if filterQuery == "" {
		return nil, fmt.Errorf("filter query cannot be empty")
	}

	query := url.Values{}
	query.Set("query", filterQuery)

	var response PaginatedResponse[Task]
	if err := c.GetWithQuery("/tasks/filter", query, &response); err != nil {
		return nil, fmt.Errorf("failed to get filtered tasks: %w", err)
	}

	return response.Results, nil
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
// labels @label, projects #project, assignees +name.
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

// CompletedTaskParams represents parameters for fetching completed tasks.
type CompletedTaskParams struct {
	ProjectID     string
	SectionID     string
	Label         string
	ParentID      string
	Limit         int
	Offset        int
	Page          int
	Since         string // ISO 8601 date string
	Until         string // ISO 8601 date string
	AnnotateItems bool
	AnnotateNotes bool
}

// GetCompletedTasks returns a list of completed tasks based on the provided parameters.
// This uses the /tasks/completed/by_completion_date Unified v1 endpoint.
func (c *Client) GetCompletedTasks(params CompletedTaskParams) ([]Task, error) {
	type CompletedTasksResponse struct {
		Items []Task `json:"items"`
	}

	query := url.Values{}
	if params.ProjectID != "" {
		query.Set("project_id", params.ProjectID)
	}
	if params.SectionID != "" {
		query.Set("section_id", params.SectionID)
	}
	if params.Label != "" {
		query.Set("label", params.Label)
	}
	if params.ParentID != "" {
		query.Set("parent_id", params.ParentID)
	}
	if params.Limit > 0 {
		query.Set("limit", strconv.Itoa(params.Limit))
	}
	if params.Offset > 0 {
		query.Set("offset", strconv.Itoa(params.Offset))
	}
	if params.Page > 0 {
		query.Set("page", strconv.Itoa(params.Page))
	}
	if params.Since != "" {
		query.Set("since", params.Since)
	}
	if params.Until != "" {
		query.Set("until", params.Until)
	}
	if params.AnnotateItems {
		query.Set("annotate_items", "true")
	}
	if params.AnnotateNotes {
		query.Set("annotate_notes", "true")
	}

	var response CompletedTasksResponse
	if err := c.GetWithQuery("/tasks/completed/by_completion_date", query, &response); err != nil {
		return nil, fmt.Errorf("failed to get completed tasks: %w", err)
	}

	return response.Items, nil
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

// MoveTasksBatch moves multiple tasks to a different project or section using Sync API batching.
func (c *Client) MoveTasksBatch(ids []string, targetProjectID string, targetSectionID string) error {
	if len(ids) == 0 {
		return nil
	}

	commands := make([]interface{}, len(ids))
	for i, id := range ids {
		args := map[string]interface{}{
			"id": id,
		}
		if targetSectionID != "" {
			args["section_id"] = targetSectionID
		}
		if targetProjectID != "" {
			args["project_id"] = targetProjectID
		}

		commands[i] = map[string]interface{}{
			"type": "item_move",
			"uuid": uuid.New().String(),
			"args": args,
		}
	}

	cmdsJSON, _ := json.Marshal(commands)
	formData := url.Values{}
	formData.Set("commands", string(cmdsJSON))

	syncURL := c.baseURL + "/sync"
	req, err := http.NewRequest("POST", syncURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create sync request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

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
