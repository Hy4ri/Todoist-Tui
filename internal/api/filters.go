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

// Filter represents a saved filter in Todoist.
type Filter struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Query      string `json:"query"`
	Color      string `json:"color"`
	ItemOrder  int    `json:"item_order"`
	IsDeleted  bool   `json:"is_deleted"`
	IsFavorite bool   `json:"is_favorite"`
}

// GetFilters fetches all filters via the Sync API.
func (c *Client) GetFilters() ([]Filter, error) {
	syncURL := c.baseURL + "/sync"

	formData := url.Values{}
	formData.Set("sync_token", "*")
	formData.Set("resource_types", "[\"filters\"]")

	req, err := http.NewRequest("POST", syncURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create sync request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("sync API error %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Filters []Filter `json:"filters"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode sync response: %w", err)
	}

	filtered := []Filter{}
	for _, f := range result.Filters {
		if !f.IsDeleted {
			filtered = append(filtered, f)
		}
	}

	return filtered, nil
}

// CreateFilter creates a new filter via Sync API.
func (c *Client) CreateFilter(name, query, color string) (*Filter, error) {
	syncURL := c.baseURL + "/sync"

	tempID := uuid.New().String()
	cmdUUID := uuid.New().String()

	args := map[string]string{
		"name":  name,
		"query": query,
	}
	if color != "" {
		args["color"] = color
	}

	command := map[string]interface{}{
		"type":    "filter_add",
		"temp_id": tempID,
		"uuid":    cmdUUID,
		"args":    args,
	}

	commands, _ := json.Marshal([]interface{}{command})

	formData := url.Values{}
	formData.Set("commands", string(commands))

	req, err := http.NewRequest("POST", syncURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create sync request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("sync API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to get the real ID
	var result struct {
		TempIDMapping map[string]string `json:"temp_id_mapping"`
		SyncStatus    map[string]string `json:"sync_status"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode sync response: %w", err)
	}

	// Check if command succeeded
	if status, ok := result.SyncStatus[cmdUUID]; ok && status != "ok" {
		return nil, fmt.Errorf("filter creation failed: %s", status)
	}

	realID := result.TempIDMapping[tempID]
	return &Filter{
		ID:    realID,
		Name:  name,
		Query: query,
	}, nil
}

// DeleteFilter deletes a filter via Sync API.
func (c *Client) DeleteFilter(id string) error {
	syncURL := c.baseURL + "/sync"

	cmdUUID := uuid.New().String()

	command := map[string]interface{}{
		"type": "filter_delete",
		"uuid": cmdUUID,
		"args": map[string]string{
			"id": id,
		},
	}

	commands, _ := json.Marshal([]interface{}{command})

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

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("sync API error %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// UpdateFilter updates a filter via Sync API.
func (c *Client) UpdateFilter(id, name, query string) (*Filter, error) {
	syncURL := c.baseURL + "/sync"

	cmdUUID := uuid.New().String()

	args := map[string]string{"id": id}
	if name != "" {
		args["name"] = name
	}
	if query != "" {
		args["query"] = query
	}

	command := map[string]interface{}{
		"type": "filter_update",
		"uuid": cmdUUID,
		"args": args,
	}

	commands, _ := json.Marshal([]interface{}{command})

	formData := url.Values{}
	formData.Set("commands", string(commands))

	req, err := http.NewRequest("POST", syncURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create sync request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("sync API error %d: %s", resp.StatusCode, string(body))
	}

	return &Filter{
		ID:    id,
		Name:  name,
		Query: query,
	}, nil
}
