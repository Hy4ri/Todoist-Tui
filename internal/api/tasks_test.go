package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
		response   PaginatedResponse[Task]
		statusCode int
		wantErr    bool
	}{
		{
			name:   "successful request",
			filter: TaskFilter{},
			response: PaginatedResponse[Task]{
				Results: []Task{
					{
						ID:        "123",
						Content:   "Test task",
						ProjectID: "456",
						Priority:  1,
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:   "unauthorized",
			filter: TaskFilter{},
			response: PaginatedResponse[Task]{
				Results: nil,
			},
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name: "filter by project",
			filter: TaskFilter{
				ProjectID: "789",
			},
			response: PaginatedResponse[Task]{
				Results: []Task{
					{
						ID:        "124",
						Content:   "Project task",
						ProjectID: "789",
					},
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
				json.NewEncoder(w).Encode(tt.response)
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

			if len(tasks) != len(tt.response.Results) {
				t.Errorf("expected %d tasks, got %d", len(tt.response.Results), len(tasks))
			}

			if len(tasks) > 0 && tasks[0].Content != tt.response.Results[0].Content {
				t.Errorf("expected content %q, got %q", tt.response.Results[0].Content, tasks[0].Content)
			}
		})
	}
}

func TestGetTasksByFilter(t *testing.T) {
	tests := []struct {
		name        string
		filterQuery string
		response    PaginatedResponse[Task]
		statusCode  int
		wantErr     bool
	}{
		{
			name:        "successful filter request",
			filterQuery: "today",
			response: PaginatedResponse[Task]{
				Results: []Task{
					{
						ID:      "123",
						Content: "Today task",
					},
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:        "empty filter query",
			filterQuery: "",
			wantErr:     true,
		},
		{
			name:        "api error",
			filterQuery: "tomorrow",
			statusCode:  http.StatusInternalServerError,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := mockServer(func(w http.ResponseWriter, r *http.Request) {
				if tt.filterQuery != "" {
					if r.URL.Path != "/tasks/filter" {
						t.Errorf("expected path /tasks/filter, got %s", r.URL.Path)
					}
					if r.URL.Query().Get("query") != tt.filterQuery {
						t.Errorf("expected query %q, got %q", tt.filterQuery, r.URL.Query().Get("query"))
					}
				}

				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.response)
			})
			defer server.Close()

			client := NewClient("test-token")
			client.baseURL = server.URL

			tasks, err := client.GetTasksByFilter(tt.filterQuery)

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

			if len(tasks) != len(tt.response.Results) {
				t.Errorf("expected %d tasks, got %d", len(tt.response.Results), len(tasks))
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

func TestQuickAddTask(t *testing.T) {
	tests := []struct {
		name       string
		text       string
		response   Task
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful quick add",
			text: "Buy milk tomorrow",
			response: Task{
				ID:      "123",
				Content: "Buy milk",
				Due: &Due{
					String: "tomorrow",
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "api error",
			text:       "Invalid",
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
				if r.URL.Path != "/tasks/quick" {
					t.Errorf("expected /tasks/quick, got %s", r.URL.Path)
				}

				var req map[string]string
				json.NewDecoder(r.Body).Decode(&req)
				if req["text"] != tt.text {
					t.Errorf("expected text %q, got %q", tt.text, req["text"])
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.response)
				}
			})
			defer server.Close()

			client := NewClient("test-token")
			client.baseURL = server.URL

			task, err := client.QuickAddTask(tt.text)
			if (err != nil) != tt.wantErr {
				t.Errorf("QuickAddTask() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && task.Content != tt.response.Content {
				t.Errorf("expected content %q, got %q", tt.response.Content, task.Content)
			}
		})
	}
}

func TestGetProductivityStats(t *testing.T) {
	expectedStats := ProductivityStats{
		Karma: 1000,
		Goals: ProductivityGoals{
			DailyGoal:  5,
			WeeklyGoal: 25,
		},
	}

	server := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET request, got %s", r.Method)
		}
		if r.URL.Path != "/tasks/completed/stats" {
			t.Errorf("expected /tasks/completed/stats, got %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(expectedStats)
	})
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	stats, err := client.GetProductivityStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stats.Karma != expectedStats.Karma {
		t.Errorf("expected karma %f, got %f", expectedStats.Karma, stats.Karma)
	}
}

func TestGetProductivityStats_NotFound(t *testing.T) {
	server := mockServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":      "Not found",
			"error_code": 478,
			"http_code":  404,
		})
	})
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = "http://127.0.0.1:0" // Prevent full URL override hack from hitting real API
	// But wait, the implementation uses client.baseURL + path override logic.
	// If I set client.baseURL to something not "https://api.todoist.com/api/v1", it appends "/sync...".
	// The mockServer URL is "http://127.0.0.1:xxxxx".
	// So set client.baseURL = server.URL.
	client.baseURL = server.URL

	_, err := client.GetProductivityStats()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Verify it's an APIError with 404
	apiErr, ok := err.(*APIError)
	if !ok {
		// It might be wrapped
		if internalErr := err.Error(); !strings.Contains(internalErr, "status 404") {
			t.Errorf("expected 404 error, got %v", err)
		}
	} else {
		if apiErr.StatusCode != 404 {
			t.Errorf("expected status 404, got %d", apiErr.StatusCode)
		}
	}
}

func TestMoveTask(t *testing.T) {
	taskID := "123"
	projectID := "456"

	server := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/tasks/"+taskID+"/move" {
			t.Errorf("expected /tasks/%s/move, got %s", taskID, r.URL.Path)
		}

		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req)
		if req["project_id"] != projectID {
			t.Errorf("expected project_id %s, got %v", projectID, req["project_id"])
		}

		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	err := client.MoveTask(taskID, nil, &projectID, nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMoveTasksBatch(t *testing.T) {
	server := mockServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/sync" {
			t.Errorf("expected path /sync, got %s", r.URL.Path)
		}

		// Verify form data
		err := r.ParseForm()
		if err != nil {
			t.Errorf("failed to parse form: %v", err)
		}

		cmdsStr := r.Form.Get("commands")
		if cmdsStr == "" {
			t.Error("expected commands in form data")
		}

		var commands []map[string]interface{}
		if err := json.Unmarshal([]byte(cmdsStr), &commands); err != nil {
			t.Errorf("failed to unmarshal commands: %v", err)
		}

		if len(commands) != 2 {
			t.Errorf("expected 2 commands, got %d", len(commands))
		}

		if commands[0]["type"] != "item_move" {
			t.Errorf("expected command type item_move, got %s", commands[0]["type"])
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"sync_status": {"uuid1": "ok", "uuid2": "ok"}}`))
	})
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	err := client.MoveTasksBatch([]string{"task1", "task2"}, "proj1", "sec1")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
