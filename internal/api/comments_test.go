package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetComments(t *testing.T) {
	taskID := "123"
	tests := []struct {
		name       string
		taskID     string
		response   []Comment
		statusCode int
		wantErr    bool
	}{
		{
			name:   "successful request",
			taskID: taskID,
			response: []Comment{
				{ID: "1", Content: "Comment 1"},
				{ID: "2", Content: "Comment 2"},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/comments" {
					t.Errorf("expected /comments path, got %s", r.URL.Path)
				}
				if r.URL.Query().Get("task_id") != tt.taskID {
					t.Errorf("expected task_id %s, got %s", tt.taskID, r.URL.Query().Get("task_id"))
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					resp := PaginatedResponse[Comment]{
						Results: tt.response,
					}
					json.NewEncoder(w).Encode(resp)
				}
			}))
			defer server.Close()

			client := NewClient("test-token")
			client.baseURL = server.URL

			comments, err := client.GetComments(tt.taskID, "")
			if (err != nil) != tt.wantErr {
				t.Errorf("GetComments() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(comments) != len(tt.response) {
				t.Errorf("expected %d comments, got %d", len(tt.response), len(comments))
			}
		})
	}
}

func TestCreateComment(t *testing.T) {
	content := "New Comment"
	taskID := "123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Comment{ID: "1", Content: content, ItemID: &taskID})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	comment, err := client.CreateComment(CreateCommentRequest{Content: content, TaskID: taskID})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if comment.Content != content {
		t.Errorf("expected content %s, got %s", content, comment.Content)
	}
}

func TestDeleteComment(t *testing.T) {
	id := "123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	err := client.DeleteComment(id)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
