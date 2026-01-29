package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

func TestRenderCalendarExpanded_Correctness(t *testing.T) {
	// Setup state
	targetDate := "2023-10-15"
	s := &state.State{
		CalendarDate:     time.Date(2023, 10, 15, 0, 0, 0, 0, time.Local),
		CalendarViewMode: state.CalendarViewExpanded,
		Width:            100,
		Height:           40,
		TasksByDate:      make(map[string][]api.Task),
	}

	taskContent := "Important Task"
	s.TasksByDate[targetDate] = []api.Task{
		{
			ID:      "1",
			Content: taskContent,
			Due:     &api.Due{Date: targetDate},
		},
	}

	r := NewRenderer(s)
	output := r.renderCalendar(40)

	// Note: content might be truncated depending on cell width
	expectedSubstring := "Important"
	if !strings.Contains(output, expectedSubstring) {
		t.Errorf("Expected output to contain %q, but it didn't. Output: \n%s", expectedSubstring, output)
	}
}

func BenchmarkRenderCalendarExpanded(b *testing.B) {
	// Setup state
	s := &state.State{
		CalendarDate:     time.Date(2023, 10, 15, 0, 0, 0, 0, time.Local),
		CalendarViewMode: state.CalendarViewExpanded,
		Width:            100,
		Height:           40,
	}

	// Generate tasks
	tasks := make([]api.Task, 1000)
	for i := 0; i < 1000; i++ {
		// Alternate between inside and outside the month
		month := 10
		if i%2 == 0 {
			month = 11 // Outside
		}
		day := (i % 28) + 1
		dateStr := fmt.Sprintf("2023-%02d-%02d", month, day)

		tasks[i] = api.Task{
			ID:      fmt.Sprintf("task-%d", i),
			Content: fmt.Sprintf("Task %d", i),
			Due: &api.Due{
				Date: dateStr,
			},
		}
	}
	s.AllTasks = tasks
	s.TasksByDate = make(map[string][]api.Task)
	for _, t := range tasks {
		s.TasksByDate[t.Due.Date] = append(s.TasksByDate[t.Due.Date], t)
	}

	r := NewRenderer(s)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.renderCalendar(40)
	}
}
