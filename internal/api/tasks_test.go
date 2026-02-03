package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockServer creates a test HTTP server for mocking API responses.
func mockServer(handler http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(handler)
}

func TestNewClient(t *testing.T) {
	token := "test-token"
	client := NewClient(token)

	if client.accessToken != token {
		t.Errorf("expected token %q, got %q", token, client.accessToken)
	}

	if client.baseURL != "https://api.todoist.com/api/v1" {
		t.Errorf("unexpected base URL: %s", client.baseURL)
	}
}

func TestGetTasks(t *testing.T) {
	tests := []struct {
		name       string
		filter     TaskFilter
		response   []Task
		statusCode int
		wantErr    bool
	}{
		{
			name:   "successful request",
			filter: TaskFilter{},
			response: []Task{
				{
					ID:        "123",
					Content:   "Test task",
					ProjectID: "456",
					Priority:  1,
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			filter:     TaskFilter{},
			response:   nil,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "filter by project",
			filter: TaskFilter{
				ProjectID: "789",
			},
			response: []Task{
				{
					ID:        "124",
					Content:   "Project task",
					ProjectID: "789",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockServer(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				// Verify authorization header
				authHeader := r.Header.Get("Authorization")
				if authHeader != "Bearer test-token" {
					t.Errorf("expected Bearer token, got %q", authHeader)
				}

				// Verify filter query parameters
				if tt.filter.ProjectID != "" {
					if r.URL.Query().Get("project_id") != tt.filter.ProjectID {
						t.Errorf("expected project_id %q in query", tt.filter.ProjectID)
					}
				}

				// Send response
				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					// Return paginated response format
					paginatedResp := map[string]interface{}{
						"results":     tt.response,
						"next_cursor": nil,
					}
					json.NewEncoder(w).Encode(paginatedResp)
				}
			})
			defer server.Close()

			client := NewClient("test-token")
			client.baseURL = server.URL

			tasks, err := client.GetTasks(tt.filter)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if len(tasks) != len(tt.response) {
				t.Errorf("expected %d tasks, got %d", len(tt.response), len(tasks))
			}

			if len(tasks) > 0 && tasks[0].Content != tt.response[0].Content {
				t.Errorf("expected content %q, got %q", tt.response[0].Content, tasks[0].Content)
			}
		})
	}
}

func TestCreateTask(t *testing.T) {
	tests := []struct {
		name       string
		request    CreateTaskRequest
		response   Task
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful creation",
			request: CreateTaskRequest{
				Content:   "New task",
				ProjectID: "123",
				Priority:  2,
			},
			response: Task{
				ID:        "999",
				Content:   "New task",
				ProjectID: "123",
				Priority:  2,
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "with due date",
			request: CreateTaskRequest{
				Content:   "Task with due",
				DueString: "tomorrow",
			},
			response: Task{
				ID:      "998",
				Content: "Task with due",
				Due: &Due{
					String: "tomorrow",
					Date:   "2026-01-20",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "validation error",
			request: CreateTaskRequest{
				Content: "", // Empty content
			},
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockServer(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				// Verify content type
				if r.Header.Get("Content-Type") != "application/json" {
					t.Error("expected Content-Type: application/json")
				}

				// Decode request body
				var req CreateTaskRequest
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					t.Errorf("failed to decode request: %v", err)
				}

				// Verify request matches
				if req.Content != tt.request.Content {
					t.Errorf("expected content %q, got %q", tt.request.Content, req.Content)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.response)
				}
			})
			defer server.Close()

			client := NewClient("test-token")
			client.baseURL = server.URL

			task, err := client.CreateTask(tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if task.Content != tt.response.Content {
				t.Errorf("expected content %q, got %q", tt.response.Content, task.Content)
			}
		})
	}
}

func TestUpdateTask(t *testing.T) {
	taskID := "123"
	tests := []struct {
		name       string
		request    UpdateTaskRequest
		response   Task
		statusCode int
		wantErr    bool
	}{
		{
			name: "update priority",
			request: UpdateTaskRequest{
				Priority: IntPtr(4),
			},
			response: Task{
				ID:       taskID,
				Content:  "Test task",
				Priority: 4,
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "update content and due",
			request: UpdateTaskRequest{
				Content:   StringPtr("Updated content"),
				DueString: StringPtr("next week"),
			},
			response: Task{
				ID:      taskID,
				Content: "Updated content",
				Due: &Due{
					String: "next week",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "task not found",
			request: UpdateTaskRequest{
				Content: StringPtr("Update"),
			},
			statusCode: http.StatusNotFound,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockServer(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				// Verify task ID in URL
				expectedPath := "/tasks/" + taskID
				if r.URL.Path != expectedPath {
					t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.response)
				}
			})
			defer server.Close()

			client := NewClient("test-token")
			client.baseURL = server.URL

			task, err := client.UpdateTask(taskID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if task.ID != taskID {
				t.Errorf("expected task ID %q, got %q", taskID, task.ID)
			}
		})
	}
}

func TestCloseTask(t *testing.T) {
	taskID := "123"

	server := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		expectedPath := "/tasks/" + taskID + "/close"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	err := client.CloseTask(taskID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestReopenTask(t *testing.T) {
	taskID := "456"

	server := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		expectedPath := "/tasks/" + taskID + "/reopen"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	err := client.ReopenTask(taskID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDeleteTask(t *testing.T) {
	taskID := "789"

	server := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}

		expectedPath := "/tasks/" + taskID
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	err := client.DeleteTask(taskID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
