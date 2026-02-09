package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/config"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

func TestRenderCalendarCompact_TaskDistribution(t *testing.T) {
	// Setup state
	now := time.Now()
	s := &state.State{
		Config:       &config.Config{},
		CalendarDate: now,
		TasksByDate: map[string][]api.Task{
			now.Format("2006-01-02"): {
				{
					ID:      "task1",
					Content: "Task 1",
					Due: &api.Due{
						Date: now.Format("2006-01-02"),
					},
					ParsedDate: &now,
				},
				{
					ID:      "task2",
					Content: "Task 2",
					Due: &api.Due{
						Date: now.Format("2006-01-02"),
					},
					ParsedDate: &now,
				},
			},
		},
		Width:  100,
		Height: 40,
	}

	r := NewRenderer(s)

	// Render
	output := r.renderCalendarCompact(40)

	// Verify
	// In compact view, days with tasks are marked.
	// We check if the current day is rendered with a task indicator (e.g., "*")
	// The implementation adds "*" to the day number if it has tasks and is not selected.
	// Since CalendarDay defaults to 0 (or not set), let's set it to tomorrow so today isn't selected.
	s.CalendarDay = now.Day() + 1
	output = r.renderCalendarCompact(40)

	dayStr := fmt.Sprintf("%2d*", now.Day())
	if !strings.Contains(output, dayStr) {
		t.Errorf("Expected output to contain task indicator '%s' for day %d, but it didn't.", dayStr, now.Day())
	}
}

func TestRenderCalendarExpanded_TaskDistribution(t *testing.T) {
	// Setup state
	now := time.Now()
	// Ensure we are testing a non-weekend day if possible to avoid unrelated style issues,
	// or just check for content presence.

	s := &state.State{
		Config:       &config.Config{},
		CalendarDate: now,
		TasksByDate: map[string][]api.Task{
			now.Format("2006-01-02"): {
				{
					ID:      "task1",
					Content: "Task1", // Shortened to fit
					Due: &api.Due{
						Date: now.Format("2006-01-02"),
					},
					ParsedDate: &now,
				},
			},
		},
		Width:            120, // Wide enough for expanded view
		Height:           40,
		CalendarViewMode: state.CalendarViewExpanded,
	}

	r := NewRenderer(s)

	// Render
	output := r.renderCalendarExpanded(40)

	// Verify
	if !strings.Contains(output, "Task1") {
		t.Error("Expected output to contain 'Task1' in the correct cell")
	}
}
