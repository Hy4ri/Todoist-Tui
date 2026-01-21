package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// BaseURL is the Todoist REST API v2 base URL.
	BaseURL = "https://api.todoist.com/rest/v2"

	// DefaultTimeout is the default HTTP client timeout.
	DefaultTimeout = 30 * time.Second
)

// Client is the Todoist API client.
type Client struct {
	httpClient  *http.Client
	baseURL     string
	accessToken string
}

// NewClient creates a new Todoist API client with the given access token.
func NewClient(accessToken string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		baseURL:     BaseURL,
		accessToken: accessToken,
	}
}

// SetHTTPClient allows overriding the default HTTP client (useful for testing).
func (c *Client) SetHTTPClient(httpClient *http.Client) {
	c.httpClient = httpClient
}

// do performs an HTTP request and decodes the JSON response.
func (c *Client) do(method, path string, body interface{}, result interface{}) error {
	// Build URL
	reqURL := c.baseURL + path

	// Prepare request body
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Create request
	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.accessToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    string(respBody),
		}
	}

	// Decode response (if expected)
	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// Get performs a GET request.
func (c *Client) Get(path string, result interface{}) error {
	return c.do(http.MethodGet, path, nil, result)
}

// GetWithQuery performs a GET request with query parameters.
func (c *Client) GetWithQuery(path string, query url.Values, result interface{}) error {
	if len(query) > 0 {
		path = path + "?" + query.Encode()
	}
	return c.do(http.MethodGet, path, nil, result)
}

// Post performs a POST request.
func (c *Client) Post(path string, body interface{}, result interface{}) error {
	return c.do(http.MethodPost, path, body, result)
}

// Delete performs a DELETE request.
func (c *Client) Delete(path string) error {
	return c.do(http.MethodDelete, path, nil, nil)
}

// buildFilterQuery builds query parameters for task filtering.
func buildFilterQuery(filter TaskFilter) url.Values {
	query := url.Values{}

	if filter.ProjectID != "" {
		query.Set("project_id", filter.ProjectID)
	}
	if filter.SectionID != "" {
		query.Set("section_id", filter.SectionID)
	}
	if filter.Label != "" {
		query.Set("label", filter.Label)
	}
	if filter.Filter != "" {
		query.Set("filter", filter.Filter)
	}
	if filter.Lang != "" {
		query.Set("lang", filter.Lang)
	}
	if len(filter.IDs) > 0 {
		query.Set("ids", strings.Join(filter.IDs, ","))
	}

	return query
}
