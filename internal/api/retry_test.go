package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// mockTransport is a minimal http.RoundTripper for testing that returns
// a configurable sequence of responses.
type mockTransport struct {
	responses []mockResponse
	index     int
}

type mockResponse struct {
	statusCode int
	body       string
	err        error
}

func (t *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.index >= len(t.responses) {
		return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
	}
	r := t.responses[t.index]
	t.index++
	if r.err != nil {
		return nil, r.err
	}
	return &http.Response{
		StatusCode: r.statusCode,
		Body:       http.NoBody,
	}, nil
}

// newTestClient creates a Client with the given transport for testing.
func newTestClient(transport http.RoundTripper) *Client {
	c := NewClient("test-token")
	c.RetryBaseDelay = 1 * time.Millisecond // Speed up tests
	c.SetHTTPClient(&http.Client{Transport: transport})
	return c
}

func TestRetry_SucceedsAfterTransient429(t *testing.T) {
	transport := &mockTransport{
		responses: []mockResponse{
			{statusCode: 429, body: "rate limited"}, // 1st attempt fails
			{statusCode: 200, body: ""},             // 2nd attempt succeeds
		},
	}
	c := newTestClient(transport)

	// GET should retry and succeed on the 2nd attempt
	err := c.Get("/tasks", nil)
	if err != nil {
		t.Fatalf("expected success after retry, got error: %v", err)
	}
	if transport.index != 2 {
		t.Errorf("expected 2 HTTP attempts, got %d", transport.index)
	}
}

func TestRetry_SucceedsAfterTransient500(t *testing.T) {
	transport := &mockTransport{
		responses: []mockResponse{
			{statusCode: 500, body: "server error"}, // 1st fails
			{statusCode: 500, body: "server error"}, // 2nd fails
			{statusCode: 200, body: ""},             // 3rd succeeds
		},
	}
	c := newTestClient(transport)

	err := c.Get("/tasks", nil)
	if err != nil {
		t.Fatalf("expected success after 2 retries, got error: %v", err)
	}
	if transport.index != 3 {
		t.Errorf("expected 3 HTTP attempts, got %d", transport.index)
	}
}

func TestRetry_NoRetryOn4xx(t *testing.T) {
	for _, code := range []int{400, 401, 403, 404} {
		code := code
		t.Run(fmt.Sprintf("HTTP_%d", code), func(t *testing.T) {
			transport := &mockTransport{
				responses: []mockResponse{
					{statusCode: code, body: "client error"},
				},
			}
			c := newTestClient(transport)

			err := c.Get("/tasks", nil)
			if err == nil {
				t.Fatalf("expected error for %d, got nil", code)
			}
			// Should only have made one attempt — no retry on client errors
			if transport.index != 1 {
				t.Errorf("expected 1 HTTP attempt for %d, got %d", code, transport.index)
			}
		})
	}
}

func TestRetry_ExhaustsMaxRetries(t *testing.T) {
	// All responses are 500 — should exhaust MaxRetries and return last error
	responses := make([]mockResponse, 4) // MaxRetries=3, so 4 total (1 initial + 3 retries)
	for i := range responses {
		responses[i] = mockResponse{statusCode: 500, body: "persistent server error"}
	}
	transport := &mockTransport{responses: responses}
	c := newTestClient(transport)

	err := c.Get("/tasks", nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	apiErr, ok := IsAPIError(err)
	if !ok || !apiErr.IsServerError() {
		t.Errorf("expected APIError with 5xx, got: %v", err)
	}
	// 1 initial + 3 retries = 4 attempts total
	expectedAttempts := c.MaxRetries + 1
	if transport.index != expectedAttempts {
		t.Errorf("expected %d HTTP attempts, got %d", expectedAttempts, transport.index)
	}
}

func TestRetry_NetworkErrorIsRetried(t *testing.T) {
	transport := &mockTransport{
		responses: []mockResponse{
			{err: fmt.Errorf("connection refused")}, // 1st: network error
			{statusCode: 200, body: ""},             // 2nd: succeeds
		},
	}
	c := newTestClient(transport)

	err := c.Get("/tasks", nil)
	if err != nil {
		t.Fatalf("expected success after network error retry, got: %v", err)
	}
	if transport.index != 2 {
		t.Errorf("expected 2 HTTP attempts, got %d", transport.index)
	}
}

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil error", nil, false},
		{"400 Bad Request", &APIError{StatusCode: 400}, false},
		{"401 Unauthorized", &APIError{StatusCode: 401}, false},
		{"403 Forbidden", &APIError{StatusCode: 403}, false},
		{"404 Not Found", &APIError{StatusCode: 404}, false},
		{"429 Rate Limited", &APIError{StatusCode: 429}, true},
		{"500 Server Error", &APIError{StatusCode: 500}, true},
		{"503 Service Unavailable", &APIError{StatusCode: 503}, true},
		{"network error", fmt.Errorf("connect: connection refused"), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isRetryable(tc.err)
			if got != tc.expected {
				t.Errorf("isRetryable(%v) = %v, want %v", tc.err, got, tc.expected)
			}
		})
	}
}
