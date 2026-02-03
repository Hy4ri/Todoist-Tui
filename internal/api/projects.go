package api

import (
	"fmt"
	"net/url"
)

// GetProjects returns all projects.
// Handles v1 API pagination automatically, fetching all pages.
func (c *Client) GetProjects() ([]Project, error) {
	var allProjects []Project
	query := url.Values{}

	for {
		var response PaginatedResponse[Project]
		if err := c.GetWithQuery("/projects", query, &response); err != nil {
			return nil, fmt.Errorf("failed to get projects: %w", err)
		}

		allProjects = append(allProjects, response.Results...)

		if response.NextCursor == nil || *response.NextCursor == "" {
			break
		}
		query.Set("cursor", *response.NextCursor)
	}

	return allProjects, nil
}

// GetProject returns a single project by ID.
func (c *Client) GetProject(id string) (*Project, error) {
	var project Project
	if err := c.Get("/projects/"+id, &project); err != nil {
		return nil, fmt.Errorf("failed to get project %s: %w", id, err)
	}
	return &project, nil
}

// CreateProject creates a new project.
func (c *Client) CreateProject(req CreateProjectRequest) (*Project, error) {
	var project Project
	if err := c.Post("/projects", req, &project); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}
	return &project, nil
}

// UpdateProject updates an existing project.
func (c *Client) UpdateProject(id string, req UpdateProjectRequest) (*Project, error) {
	var project Project
	if err := c.Post("/projects/"+id, req, &project); err != nil {
		return nil, fmt.Errorf("failed to update project %s: %w", id, err)
	}
	return &project, nil
}

// DeleteProject deletes a project.
func (c *Client) DeleteProject(id string) error {
	if err := c.Delete("/projects/" + id); err != nil {
		return fmt.Errorf("failed to delete project %s: %w", id, err)
	}
	return nil
}

// GetProjectCollaborators returns all collaborators for a shared project.
func (c *Client) GetProjectCollaborators(projectID string) ([]Collaborator, error) {
	var collaborators []Collaborator
	if err := c.Get("/projects/"+projectID+"/collaborators", &collaborators); err != nil {
		return nil, fmt.Errorf("failed to get collaborators for project %s: %w", projectID, err)
	}
	return collaborators, nil
}
