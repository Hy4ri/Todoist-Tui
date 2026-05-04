package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/google/uuid"
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
		var response PaginatedResponse[Section]
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
	var args []map[string]interface{}
	for _, s := range sections {
		args = append(args, map[string]interface{}{
			"id":            s.ID,
			"section_order": s.SectionOrder,
		})
	}

	cmdUUID := uuid.NewString()
	command := map[string]interface{}{
		"type": "section_reorder",
		"uuid": cmdUUID,
		"args": map[string]interface{}{
			"sections": args,
		},
	}

	syncURL := c.baseURL + "/sync"
	commands, err := json.Marshal([]interface{}{command})
	if err != nil {
		return fmt.Errorf("failed to marshal sync request: %w", err)
	}

	formData := url.Values{}
	formData.Set("commands", string(commands))

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read sync response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sync API error %d", resp.StatusCode)
	}

	if len(body) > 0 {
		var result struct {
			SyncStatus map[string]string `json:"sync_status"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return fmt.Errorf("failed to decode sync response: %w", err)
		}
		if status, ok := result.SyncStatus[cmdUUID]; ok && status != "ok" {
			return fmt.Errorf("section reorder failed: %s", status)
		}
	}

	return nil
}
