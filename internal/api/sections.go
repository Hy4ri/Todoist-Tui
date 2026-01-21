package api

import (
	"fmt"
	"net/url"
)

// GetSections returns all sections, optionally filtered by project.
// Handles v1 API pagination automatically, fetching all pages.
func (c *Client) GetSections(projectID string) ([]Section, error) {
	var allSections []Section
	query := url.Values{}
	if projectID != "" {
		query.Set("project_id", projectID)
	}

	for {
		var response SectionsPaginatedResponse
		if err := c.GetWithQuery("/sections", query, &response); err != nil {
			return nil, fmt.Errorf("failed to get sections: %w", err)
		}

		allSections = append(allSections, response.Results...)

		if response.NextCursor == nil || *response.NextCursor == "" {
			break
		}
		query.Set("cursor", *response.NextCursor)
	}

	return allSections, nil
}

// GetSection returns a single section by ID.
func (c *Client) GetSection(id string) (*Section, error) {
	var section Section
	if err := c.Get("/sections/"+id, &section); err != nil {
		return nil, fmt.Errorf("failed to get section %s: %w", id, err)
	}
	return &section, nil
}

// CreateSection creates a new section.
func (c *Client) CreateSection(req CreateSectionRequest) (*Section, error) {
	var section Section
	if err := c.Post("/sections", req, &section); err != nil {
		return nil, fmt.Errorf("failed to create section: %w", err)
	}
	return &section, nil
}

// UpdateSection updates an existing section.
func (c *Client) UpdateSection(id string, req UpdateSectionRequest) (*Section, error) {
	var section Section
	if err := c.Post("/sections/"+id, req, &section); err != nil {
		return nil, fmt.Errorf("failed to update section %s: %w", id, err)
	}
	return &section, nil
}

// DeleteSection deletes a section.
func (c *Client) DeleteSection(id string) error {
	if err := c.Delete("/sections/" + id); err != nil {
		return fmt.Errorf("failed to delete section %s: %w", id, err)
	}
	return nil
}
