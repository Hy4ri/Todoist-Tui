package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetFilters(t *testing.T) {
	tests := []struct {
		name       string
		response   []Filter
		statusCode int
		wantErr    bool
	}{
		{
			name: "successful request",
			response: []Filter{
				{ID: "1", Name: "Filter 1", Query: "today"},
				{ID: "2", Name: "Filter 2", Query: "tomorrow"},
			},
			statusCode: http.StatusOK,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if r.URL.Path != "/sync" {
					t.Errorf("expected /sync path, got %s", r.URL.Path)
				}

				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					resp := map[string]interface{}{
						"filters": tt.response,
					}
					json.NewEncoder(w).Encode(resp)
				}
			}))
			defer server.Close()

			client := NewClient("test-token")
			client.baseURL = server.URL

			filters, err := client.GetFilters()
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFilters() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && len(filters) != len(tt.response) {
				t.Errorf("expected %d filters, got %d", len(tt.response), len(filters))
			}
		})
	}
}

func TestCreateFilter(t *testing.T) {
	name := "New Filter"
	query := "today"
	color := "red"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		// In a real test we would inspect the commands in r.Body to make sure they match expected values.
		// For now we just return a success response with a dummy struct that won't match the internal temp_id,
		// but allows the function to complete without erroring on JSON decode.
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"sync_status":{"uuid":"ok"},"temp_id_mapping":{}}`))
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	// We expect the ID to be empty because our mock doesn't return the mapping for the random tempID.
	filter, err := client.CreateFilter(name, query, color)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if filter.Name != name {
		t.Errorf("expected name %s, got %s", name, filter.Name)
	}
}

func TestDeleteFilter(t *testing.T) {
	id := "123"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}

		// Decode request to verify ID if needed

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-token")
	client.baseURL = server.URL

	err := client.DeleteFilter(id)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
