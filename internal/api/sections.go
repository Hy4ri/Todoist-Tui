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

// GetSections returns all sections, optionally filtered by project.
// Handles v1 API pagination automatically, fetching all pages.
func (c *Client) GetSections(projectID string) ([]Section, error) {
	allSections := []Section{} // Non-nil empty slice
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

// ReorderSections updates the order of sections using the Sync API.
func (c *Client) ReorderSections(sections []Section) error {
	type sectionArg struct {
		ID           string `json:"id"`
		SectionOrder int    `json:"section_order"`
	}

	type commandArgs struct {
		Sections []sectionArg `json:"sections"`
	}

	type syncCommand struct {
		Type string      `json:"type"`
		UUID string      `json:"uuid"`
		Args commandArgs `json:"args"`
	}

	type syncRequest struct {
		Commands []syncCommand `json:"commands"`
	}

	var args []sectionArg
	for _, s := range sections {
		args = append(args, sectionArg{
			ID:           s.ID,
			SectionOrder: s.SectionOrder,
		})
	}

	cmd := syncCommand{
		Type: "section_reorder",
		UUID: fmt.Sprintf("%d", time.Now().UnixNano()), // Simple UUID generation
		Args: commandArgs{
			Sections: args,
		},
	}

	reqBody := syncRequest{
		Commands: []syncCommand{cmd},
	}

	// Sync API URL
	syncURL := "https://api.todoist.com/sync/v9/sync"

	// We need to use valid Authorization header, which c.do handles if we pass relative path?
	// But BaseURL is v1. We need absolute URL support or override.
	// However, c.do prepends BaseURL.
	// So we must manually create request or modify c.do?
	// Let's modify usage.

	// Actually, simpler: define a helper or just do it here using http.NewRequest.
	// Accessing c.httpClient and c.accessToken.

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

	// Sync API returns JSON with status of commands. We assume success if 200 for now.
	return nil
}
