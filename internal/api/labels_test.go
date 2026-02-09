package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetLabels(t *testing.T) {
	tests := []struct {
		name       string
		response   []Label
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful request",
			response: []Label{
				{ID: "1", Name: "Label 1", Color: "red"},
				{ID: "2", Name: "Label 2", Color: "blue"},
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
				if r.URL.Path != "/labels" {
					t.Errorf("expected /labels path, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					// Use PaginatedResponse wrapper
					resp := PaginatedResponse[Label]{
						Results: tt.response,
					}
					json.NewEncoder(w).Encode(resp)
				}
			}))
			defer server.Close()

			client := NewClient("test-token")
			client.baseURL = server.URL

			labels, err := client.GetLabels()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetLabels() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(labels) != len(tt.response) {
				t.Errorf("expected %d labels, got %d", len(tt.response), len(labels))
			}
		})
	}
}

func TestCreateLabel(t *testing.T) {
	name := "New Label"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Label{ID: "1", Name: name})
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	label, err := client.CreateLabel(CreateLabelRequest{Name: name})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if label.Name != name {
		t.Errorf("expected name %s, got %s", name, label.Name)
	}
}

func TestDeleteLabel(t *testing.T) {
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

	err := client.DeleteLabel(id)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
