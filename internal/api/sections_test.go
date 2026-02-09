package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetSections(t *testing.T) {
	projectID := "123"
	tests := []struct {
		name       string
		projectID  string
		response   []Section
		statusCode int
		wantErr    bool
	}{
		{
			name:      "successful request",
			projectID: projectID,
			response: []Section{
				{
					ID:        "456",
					Name:      "To Do",
					ProjectID: projectID,
				},
				{
					ID:        "789",
					Name:      "Done",
					ProjectID: projectID,
				},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "api error",
			projectID:  projectID,
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("expected GET request, got %s", r.Method)
				}
				if r.URL.Path != "/sections" {
					t.Errorf("expected /sections path, got %s", r.URL.Path)
				}
				if r.URL.Query().Get("project_id") != tt.projectID {
					t.Errorf("expected project_id %s, got %s", tt.projectID, r.URL.Query().Get("project_id"))
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					// Need a wrapper as it returns PaginatedResponse
					resp := PaginatedResponse[Section]{
						Results: tt.response,
					}
					json.NewEncoder(w).Encode(resp)
				}
			}))
			defer server.Close()

			client := NewClient("test-token")
			client.baseURL = server.URL

			sections, err := client.GetSections(tt.projectID)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSections() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(sections) != len(tt.response) {
				t.Errorf("expected %d sections, got %d", len(tt.response), len(sections))
			}
		})
	}
}

func TestReorderSections(t *testing.T) {
	sections := []Section{
		{ID: "1", SectionOrder: 1},
		{ID: "2", SectionOrder: 2},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		if r.URL.Path != "/sync" {
			t.Errorf("expected /sync path, got %s", r.URL.Path)
		}

		// We could decode body and verify commands structure...

		w.WriteHeader(http.StatusOK)
		// Return basic sync response
		w.Write([]byte(`{"sync_status":{"uuid":"ok"},"temp_id_mapping":{}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	err := client.ReorderSections(sections)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
