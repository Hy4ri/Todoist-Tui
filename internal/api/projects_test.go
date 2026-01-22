package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetProjects(t *testing.T) {
	tests := []struct {
		name       string
		response   []Project
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful request",
			response: []Project{
				{
					ID:           "123",
					Name:         "Inbox",
					Color:        "grey",
					ChildOrder:   0,
					InboxProject: true,
				},
				{
					ID:         "456",
					Name:       "Work",
					Color:      "blue",
					ChildOrder: 1,
					IsFavorite: true,
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			response:   nil,
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}

				if r.URL.Path != "/projects" {
					t.Errorf("expected /projects path, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				if tt.response != nil {
					// Return paginated response format
					paginatedResp := map[string]interface{}{
						"results":     tt.response,
						"next_cursor": nil,
					}
					json.NewEncoder(w).Encode(paginatedResp)
				}
			}))
			defer server.Close()

			client := NewClient("test-token")
			client.baseURL = server.URL

			projects, err := client.GetProjects()

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

			if len(projects) != len(tt.response) {
				t.Errorf("expected %d projects, got %d", len(tt.response), len(projects))
			}

			if len(projects) > 0 {
				if projects[0].Name != tt.response[0].Name {
					t.Errorf("expected name %q, got %q", tt.response[0].Name, projects[0].Name)
				}
			}
		})
	}
}

func TestGetProject(t *testing.T) {
	projectID := "123"
	project := Project{
		ID:         projectID,
		Name:       "My Project",
		Color:      "red",
		ChildOrder: 5,
		IsFavorite: true,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/projects/" + projectID
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(project)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	result, err := client.GetProject(projectID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != project.ID {
		t.Errorf("expected ID %q, got %q", project.ID, result.ID)
	}

	if result.Name != project.Name {
		t.Errorf("expected name %q, got %q", project.Name, result.Name)
	}
}

func TestCreateProject(t *testing.T) {
	tests := []struct {
		name       string
		request    CreateProjectRequest
		response   Project
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful creation",
			request: CreateProjectRequest{
				Name:  "New Project",
				Color: "green",
			},
			response: Project{
				ID:    "999",
				Name:  "New Project",
				Color: "green",
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "with favorite flag",
			request: CreateProjectRequest{
				Name:       "Favorite Project",
				IsFavorite: true,
			},
			response: Project{
				ID:         "998",
				Name:       "Favorite Project",
				IsFavorite: true,
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "validation error",
			request: CreateProjectRequest{
				Name: "", // Empty name
			},
			statusCode: http.StatusBadRequest,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.response)
				}
			}))
			defer server.Close()

			client := NewClient("test-token")
			client.baseURL = server.URL

			project, err := client.CreateProject(tt.request)

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

			if project.Name != tt.response.Name {
				t.Errorf("expected name %q, got %q", tt.response.Name, project.Name)
			}
		})
	}
}

func TestUpdateProject(t *testing.T) {
	projectID := "123"
	newName := "Updated Name"
	isFavorite := true

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		expectedPath := "/projects/" + projectID
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}

		// Decode request
		var req UpdateProjectRequest
		json.NewDecoder(r.Body).Decode(&req)

		// Build response based on request
		response := Project{
			ID:         projectID,
			Name:       *req.Name,
			IsFavorite: *req.IsFavorite,
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	project, err := client.UpdateProject(projectID, UpdateProjectRequest{
		Name:       &newName,
		IsFavorite: &isFavorite,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if project.Name != newName {
		t.Errorf("expected name %q, got %q", newName, project.Name)
	}

	if !project.IsFavorite {
		t.Error("expected project to be favorite")
	}
}

func TestDeleteProject(t *testing.T) {
	projectID := "789"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE request, got %s", r.Method)
		}

		expectedPath := "/projects/" + projectID
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %q, got %q", expectedPath, r.URL.Path)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	err := client.DeleteProject(projectID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
