package logic

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

type ConcurrentTransport struct {
	active    int32
	maxActive int32
	delay     time.Duration
}

func (t *ConcurrentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	current := atomic.AddInt32(&t.active, 1)

	// Update max observed concurrency
	for {
		max := atomic.LoadInt32(&t.maxActive)
		if current <= max {
			break
		}
		if atomic.CompareAndSwapInt32(&t.maxActive, max, current) {
			break
		}
	}

	time.Sleep(t.delay)

	atomic.AddInt32(&t.active, -1)

	return &http.Response{
		StatusCode: 200,
		Body:       http.NoBody,
	}, nil
}

func TestHandleCompleteConcurrency(t *testing.T) {
	// Setup
	transport := &ConcurrentTransport{
		delay: 10 * time.Millisecond,
	}
	client := api.NewClient("test-token")
	client.SetHTTPClient(&http.Client{
		Transport: transport,
	})

	s := &state.State{
		Client:          client,
		Tasks:           make([]api.Task, 100),
		SelectedTaskIDs: make(map[string]bool),
		// Setup necessary state to pass guard clauses
		CurrentTab:  state.TabInbox, // or any tab that allows completion
		FocusedPane: state.PaneMain,
		CurrentView: state.ViewInbox, // default
	}

	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("task-%d", i)
		s.Tasks[i] = api.Task{ID: id, Content: "test", Checked: false}
		s.SelectedTaskIDs[id] = true
	}

	h := NewHandler(s)

	// Execute
	cmd := h.handleComplete()
	if cmd == nil {
		t.Fatal("handleComplete returned nil")
	}

	// Run the command
	cmd()

	// Verify
	t.Logf("Max concurrent requests: %d", transport.maxActive)

	// We expect the limited concurrency to be around 5 if the implementation is correct
	if transport.maxActive > 10 {
		t.Errorf("Expected max concurrency <= 10 (approx), got %d", transport.maxActive)
	}
}
