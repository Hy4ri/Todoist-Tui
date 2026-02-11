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

// Reminder represents a Todoist reminder.
type Reminder struct {
	ID           string       `json:"id"`
	NotifyUID    string       `json:"notify_uid"`
	ItemID       string       `json:"item_id"`
	Type         string       `json:"type"`          // "absolute" or "relative"
	Due          *ReminderDue `json:"due"`           // For absolute reminders
	MinuteOffset int          `json:"minute_offset"` // For relative reminders (minutes before task due)
	IsDeleted    bool         `json:"is_deleted"`
}

// ReminderDue holds the due date for absolute reminders.
type ReminderDue struct {
	Date        string  `json:"date"`
	Timezone    *string `json:"timezone"`
	IsRecurring bool    `json:"is_recurring"`
	String      string  `json:"string"`
	Lang        string  `json:"lang"`
}

// CreateReminderRequest represents the request body for creating a reminder.
type CreateReminderRequest struct {
	ItemID       string       `json:"item_id"`
	Type         string       `json:"type"` // "absolute" or "relative"
	Due          *ReminderDue `json:"due,omitempty"`
	MinuteOffset int          `json:"minute_offset,omitempty"`
}

// UpdateReminderRequest represents the request body for updating a reminder.
type UpdateReminderRequest struct {
	ID           string       `json:"id"`
	Due          *ReminderDue `json:"due,omitempty"`
	MinuteOffset int          `json:"minute_offset,omitempty"`
}

// GetReminders fetches all reminders via the Sync API.
func (c *Client) GetReminders() ([]Reminder, error) {
	syncURL := c.baseURL + "/sync"

	formData := url.Values{}
	formData.Set("sync_token", "*")
	formData.Set("resource_types", "[\"reminders\"]")

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
		Reminders []Reminder `json:"reminders"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode sync response: %w", err)
	}

	activeReminders := []Reminder{}
	for _, r := range result.Reminders {
		if !r.IsDeleted {
			activeReminders = append(activeReminders, r)
		}
	}

	return activeReminders, nil
}

// GetRemindersForTask fetches reminders and filters them by task ID.
func (c *Client) GetRemindersForTask(taskID string) ([]Reminder, error) {
	allReminders, err := c.GetReminders()
	if err != nil {
		return nil, err
	}

	taskReminders := []Reminder{}
	for _, r := range allReminders {
		if r.ItemID == taskID {
			taskReminders = append(taskReminders, r)
		}
	}

	return taskReminders, nil
}

// CreateReminder creates a new reminder via Sync API.
func (c *Client) CreateReminder(req CreateReminderRequest) (*Reminder, error) {
	syncURL := c.baseURL + "/sync"

	tempID := uuid.New().String()
	cmdUUID := uuid.New().String()

	args := map[string]interface{}{
		"item_id": req.ItemID,
		"type":    req.Type,
	}

	if req.Type == "relative" {
		args["minute_offset"] = req.MinuteOffset
	} else if req.Type == "absolute" && req.Due != nil {
		args["due"] = req.Due
	}

	command := map[string]interface{}{
		"type":    "reminder_add",
		"temp_id": tempID,
		"uuid":    cmdUUID,
		"args":    args,
	}

	commands, _ := json.Marshal([]interface{}{command})

	formData := url.Values{}
	formData.Set("commands", string(commands))

	httpReq, err := http.NewRequest("POST", syncURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create sync request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.accessToken)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("sync API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response to get the real ID representing success
	var result struct {
		TempIDMapping map[string]string      `json:"temp_id_mapping"`
		SyncStatus    map[string]interface{} `json:"sync_status"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode sync response: %w", err)
	}

	// Check if command succeeded
	if status, ok := result.SyncStatus[cmdUUID]; ok {
		if statusStr, ok := status.(string); ok && statusStr != "ok" {
			return nil, fmt.Errorf("reminder creation failed: %s", statusStr)
		}
		// Sometimes status can be an object with error info
		if _, ok := status.(map[string]interface{}); ok {
			statusBytes, _ := json.Marshal(status)
			return nil, fmt.Errorf("reminder creation failed: %s", string(statusBytes))
		}
	} else {
		// If command UUID is missing from sync_status, it might be a silent failure or partial sync
		// But usually it should be there.
	}

	realID := result.TempIDMapping[tempID]

	// Construct returned reminder (approximation since sync doesn't return full object)
	reminder := &Reminder{
		ID:           realID,
		ItemID:       req.ItemID,
		Type:         req.Type,
		MinuteOffset: req.MinuteOffset,
		Due:          req.Due,
	}

	return reminder, nil
}

// DeleteReminder deletes a reminder via Sync API.
func (c *Client) DeleteReminder(id string) error {
	syncURL := c.baseURL + "/sync"

	cmdUUID := uuid.New().String()

	command := map[string]interface{}{
		"type": "reminder_delete",
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

// UpdateReminder updates a reminder via Sync API.
func (c *Client) UpdateReminder(req UpdateReminderRequest) (*Reminder, error) {
	syncURL := c.baseURL + "/sync"

	cmdUUID := uuid.New().String()

	args := map[string]interface{}{
		"id": req.ID,
	}
	if req.MinuteOffset != 0 {
		args["minute_offset"] = req.MinuteOffset
	}
	if req.Due != nil {
		args["due"] = req.Due
	}

	command := map[string]interface{}{
		"type": "reminder_update",
		"uuid": cmdUUID,
		"args": args,
	}

	commands, _ := json.Marshal([]interface{}{command})

	formData := url.Values{}
	formData.Set("commands", string(commands))

	httpReq, err := http.NewRequest("POST", syncURL, bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create sync request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.accessToken)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sync request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("sync API error %d: %s", resp.StatusCode, string(body))
	}

	return &Reminder{
		ID:           req.ID,
		MinuteOffset: req.MinuteOffset,
		Due:          req.Due,
	}, nil
}
