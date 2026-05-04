package api

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

type sequenceTransport struct {
	mu        sync.Mutex
	responses []func(*http.Request) (*http.Response, error)
	count     int
}

func (s *sequenceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.count++
	idx := s.count - 1
	if idx >= len(s.responses) {
		return nil, errors.New("unexpected request")
	}
	return s.responses[idx](req)
}

func TestDoMarshalsBodyAndDecodesResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer token")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("Content-Type = %q, want application/json", got)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("io.ReadAll() error = %v", err)
		}
		if !strings.Contains(string(body), `"content":"hello"`) {
			t.Fatalf("request body = %s", string(body))
		}
		_, _ = w.Write([]byte(`{"id":"1","content":"ok"}`))
	}))
	defer server.Close()

	client := NewClient("token")
	client.baseURL = server.URL
	client.SetHTTPClient(server.Client())

	var got Task
	if err := client.do(http.MethodPost, "/tasks", CreateTaskRequest{Content: "hello"}, &got); err != nil {
		t.Fatalf("do() error = %v", err)
	}
	if got.ID != "1" || got.Content != "ok" {
		t.Fatalf("decoded result = %#v", got)
	}
}

func TestDoReturnsMarshalErrorForUnsupportedBody(t *testing.T) {
	client := NewClient("token")
	client.baseURL = "http://example.com"

	type badBody struct{ Fn func() }
	if err := client.do(http.MethodPost, "/tasks", badBody{}, nil); err == nil {
		t.Fatal("do() error = nil, want marshal error")
	}
}

func TestDoWithRetryRetriesTransientFailures(t *testing.T) {
	transport := &sequenceTransport{responses: []func(*http.Request) (*http.Response, error){
		func(*http.Request) (*http.Response, error) { return nil, errors.New("temporary network failure") },
		func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"42","content":"done"}`))}, nil
		},
	}}

	client := NewClient("token")
	client.SetHTTPClient(&http.Client{Transport: transport})
	client.MaxRetries = 1
	client.RetryBaseDelay = 1
	client.baseURL = "http://example.com"

	var got Task
	if err := client.doWithRetry(http.MethodGet, "/tasks/42", nil, &got); err != nil {
		t.Fatalf("doWithRetry() error = %v", err)
	}
	if transport.count != 2 {
		t.Fatalf("attempt count = %d, want 2", transport.count)
	}
	if got.ID != "42" || got.Content != "done" {
		t.Fatalf("decoded result = %#v", got)
	}
}

func TestDoWithRetryDoesNotRetryNonRetryableAPIError(t *testing.T) {
	transport := &sequenceTransport{responses: []func(*http.Request) (*http.Response, error){
		func(*http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewBufferString(`{"error":"missing"}`))}, nil
		},
		func(*http.Request) (*http.Response, error) {
			t.Fatal("unexpected retry")
			return nil, nil
		},
	}}

	client := NewClient("token")
	client.SetHTTPClient(&http.Client{Transport: transport})
	client.MaxRetries = 3
	client.baseURL = "http://example.com"

	if err := client.doWithRetry(http.MethodGet, "/tasks/1", nil, nil); err == nil {
		t.Fatal("doWithRetry() error = nil, want API error")
	}
	if transport.count != 1 {
		t.Fatalf("attempt count = %d, want 1", transport.count)
	}
}

func TestBuildFilterQuery(t *testing.T) {
	got := buildFilterQuery(TaskFilter{
		ProjectID: "p1",
		SectionID: "s1",
		Label:     "home",
		Lang:      "en",
		IDs:       []string{"a", "b"},
	})

	want := url.Values{}
	want.Set("project_id", "p1")
	want.Set("section_id", "s1")
	want.Set("label", "home")
	want.Set("lang", "en")
	want.Set("ids", "a,b")

	if got.Encode() != want.Encode() {
		t.Fatalf("buildFilterQuery() = %q, want %q", got.Encode(), want.Encode())
	}
}

func TestPointerHelpers(t *testing.T) {
	if v := BoolPtr(true); v == nil || !*v {
		t.Fatalf("BoolPtr() = %#v", v)
	}
	if v := IntPtr(7); v == nil || *v != 7 {
		t.Fatalf("IntPtr() = %#v", v)
	}
	if v := StringPtr("x"); v == nil || *v != "x" {
		t.Fatalf("StringPtr() = %#v", v)
	}
}
