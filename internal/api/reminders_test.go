package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Helper for mocking server (since we can't import from tasks_test.go easily if not same package)
// But since this is package api, we can use the one from tasks_test.go if it is in package api (not api_test).
// tasks_test.go says "package api", so we can use mockServer from there if it's exported or same package.
// BUT, tasks_test.go has mockServer NOT exported (lowercase).
// So we must define it here or copy it.
func mockServerReminder(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestGetReminders(t *testing.T) {
	server := mockServerReminder(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		response := struct {
			Reminders []Reminder `json:"reminders"`
		}{
			Reminders: []Reminder{
				{ID: "1", ItemID: "task1", Type: "relative", MinuteOffset: 30},
				{ID: "2", ItemID: "task1", Type: "absolute", Due: &ReminderDue{Date: "2024-10-15T10:00:00"}},
				{ID: "3", ItemID: "task2", Type: "relative", IsDeleted: true},
			},
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	reminders, err := client.GetReminders()
	if err != nil {
		t.Errorf("GetReminders returned error: %v", err)
	}

	if len(reminders) != 2 {
		t.Errorf("Expected 2 active reminders, got %d", len(reminders))
	}

	if reminders[0].ID != "1" {
		t.Errorf("Expected reminder ID 1, got %s", reminders[0].ID)
	}
}

func TestGetRemindersForTask(t *testing.T) {
	server := mockServerReminder(func(w http.ResponseWriter, r *http.Request) {
		response := struct {
			Reminders []Reminder `json:"reminders"`
		}{
			Reminders: []Reminder{
				{ID: "1", ItemID: "task1", Type: "relative", MinuteOffset: 30},
				{ID: "2", ItemID: "task2", Type: "absolute"},
			},
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	reminders, err := client.GetRemindersForTask("task1")
	if err != nil {
		t.Errorf("GetRemindersForTask returned error: %v", err)
	}

	if len(reminders) != 1 {
		t.Errorf("Expected 1 reminder for task1, got %d", len(reminders))
	}

	if reminders[0].ID != "1" {
		t.Errorf("Expected reminder ID 1, got %s", reminders[0].ID)
	}
}

func TestCreateReminder(t *testing.T) {
	server := mockServerReminder(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		r.ParseForm()
		commandsStr := r.Form.Get("commands")
		var commands []map[string]interface{}
		json.Unmarshal([]byte(commandsStr), &commands)

		cmd := commands[0]
		if cmd["type"] != "reminder_add" {
			t.Errorf("Expected command type reminder_add, got %v", cmd["type"])
		}

		args := cmd["args"].(map[string]interface{})
		if args["item_id"] != "task1" {
			t.Errorf("Expected item_id task1, got %v", args["item_id"])
		}

		tempID, ok := cmd["temp_id"].(string)
		if !ok {
			tempID = "temp_id"
		}

		uuidVal, ok := cmd["uuid"].(string)
		if !ok {
			uuidVal = "uuid"
		}

		response := struct {
			TempIDMapping map[string]string      `json:"temp_id_mapping"`
			SyncStatus    map[string]interface{} `json:"sync_status"`
		}{
			TempIDMapping: map[string]string{
				tempID: "new_id_123",
			},
			SyncStatus: map[string]interface{}{
				uuidVal: "ok",
			},
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	req := CreateReminderRequest{
		ItemID:       "task1",
		Type:         "relative",
		MinuteOffset: 30,
	}

	reminder, err := client.CreateReminder(req)
	if err != nil {
		t.Errorf("CreateReminder returned error: %v", err)
	}

	if reminder.ID != "new_id_123" {
		t.Errorf("Expected reminder ID new_id_123, got %s", reminder.ID)
	}
}

func TestDeleteReminder(t *testing.T) {
	server := mockServerReminder(func(w http.ResponseWriter, r *http.Request) {
		response := struct {
			SyncStatus map[string]interface{} `json:"sync_status"`
		}{
			SyncStatus: map[string]interface{}{
				"uuid": "ok",
			},
		}
		json.NewEncoder(w).Encode(response)
	})
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	err := client.DeleteReminder("rem123")
	if err != nil {
		t.Errorf("DeleteReminder returned error: %v", err)
	}
}
