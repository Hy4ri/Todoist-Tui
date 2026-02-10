package logic

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

func TestExecuteMoveToProjectBatch(t *testing.T) {
	// Setup transport that tracks concurrency
	transport := &ConcurrentTransport{
		delay: 10 * time.Millisecond,
	}
	client := api.NewClient("test-token")
	client.SetHTTPClient(&http.Client{
		Transport: transport,
	})

	s := &state.State{
		Client:          client,
		Tasks:           make([]api.Task, 20),
		SelectedTaskIDs: make(map[string]bool),
		AllTasks:        make([]api.Task, 20),
	}

	for i := 0; i < 20; i++ {
		id := fmt.Sprintf("task-%d", i)
		task := api.Task{ID: id, Content: "test"}
		s.Tasks[i] = task
		s.AllTasks[i] = task
		s.SelectedTaskIDs[id] = true
	}

	h := NewHandler(s)

	// Target project
	target := state.MoveTarget{
		ID:        "target-prj",
		Name:      "Target Project",
		ProjectID: "target-prj",
		IsSection: false,
	}

	// Execute move
	cmd := h.executeMoveToProject(target)
	if cmd == nil {
		t.Fatal("executeMoveToProject returned nil")
	}

	// Optimistic check
	// Tasks are NOT removed from AllTasks in optimistic update anymore, they are filtered out after handleRefresh
	// Wait, in executeMoveToProject we call handleRefresh(false) which updates h.Tasks.
	// Since all moved tasks have different ProjectID now, if they don't match current view, they should be removed from h.Tasks.
	// But h.AllTasks still has them.

	// Clear Selection
	if len(h.SelectedTaskIDs) != 0 {
		t.Error("Expected SelectedTaskIDs to be cleared")
	}

	// Run the background command
	msg := cmd()

	// Verify it returned a statusMsg (success)
	if _, ok := msg.(statusMsg); !ok {
		t.Errorf("Expected statusMsg, got %T: %v", msg, msg)
	}

	// Verify batching
	t.Logf("Total move requests: %d", transport.totalRequests)
	if transport.totalRequests != 1 {
		t.Errorf("Expected 1 batch request for 20 tasks, got %d", transport.totalRequests)
	}
}
