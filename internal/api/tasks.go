package api

import "fmt"

// GetTasks returns all active tasks, optionally filtered.
// Handles v1 API pagination automatically, fetching all pages.
func (c *Client) GetTasks(filter TaskFilter) ([]Task, error) {
	var allTasks []Task
	query := buildFilterQuery(filter)

	for {
		var response TasksPaginatedResponse
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
