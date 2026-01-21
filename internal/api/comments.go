package api

import (
	"fmt"
	"net/url"
)

// GetComments returns comments for a task or project.
// Either taskID or projectID must be provided.
func (c *Client) GetComments(taskID, projectID string) ([]Comment, error) {
	var comments []Comment
	query := url.Values{}
	if taskID != "" {
		query.Set("task_id", taskID)
	} else if projectID != "" {
		query.Set("project_id", projectID)
	}
	if err := c.GetWithQuery("/comments", query, &comments); err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}
	return comments, nil
}

// GetComment returns a single comment by ID.
func (c *Client) GetComment(id string) (*Comment, error) {
	var comment Comment
	if err := c.Get("/comments/"+id, &comment); err != nil {
		return nil, fmt.Errorf("failed to get comment %s: %w", id, err)
	}
	return &comment, nil
}

// CreateComment creates a new comment on a task or project.
func (c *Client) CreateComment(req CreateCommentRequest) (*Comment, error) {
	var comment Comment
	if err := c.Post("/comments", req, &comment); err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}
	return &comment, nil
}

// UpdateComment updates an existing comment.
func (c *Client) UpdateComment(id string, req UpdateCommentRequest) (*Comment, error) {
	var comment Comment
	if err := c.Post("/comments/"+id, req, &comment); err != nil {
		return nil, fmt.Errorf("failed to update comment %s: %w", id, err)
	}
	return &comment, nil
}

// DeleteComment deletes a comment.
func (c *Client) DeleteComment(id string) error {
	if err := c.Delete("/comments/" + id); err != nil {
		return fmt.Errorf("failed to delete comment %s: %w", id, err)
	}
	return nil
}
