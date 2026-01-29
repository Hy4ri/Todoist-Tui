package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

func BenchmarkRenderCalendarCompact(b *testing.B) {
	// Setup state
	s := &state.State{
		CalendarDate: time.Date(2023, 10, 1, 0, 0, 0, 0, time.Local),
		CalendarDay:  1,
		AllTasks:     make([]api.Task, 1000),
		CalendarViewMode: state.CalendarViewCompact,
		TaskDates:    make(map[string]time.Time),
	}

	// Populate tasks
	for i := 0; i < 1000; i++ {
		date := "2023-10-15"
		if i%2 == 0 {
			date = "2023-10-16"
		}
		s.AllTasks[i] = api.Task{
			ID: fmt.Sprintf("task-%d", i),
			Due: &api.Due{
				Date: date,
			},
			Content: "Task",
		}
		parsed, _ := time.Parse("2006-01-02", date)
		s.TaskDates[s.AllTasks[i].ID] = parsed
	}

	r := NewRenderer(s)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.renderCalendarCompact(20)
	}
}

func BenchmarkRenderCalendarExpanded(b *testing.B) {
	// Setup state
	s := &state.State{
		CalendarDate: time.Date(2023, 10, 1, 0, 0, 0, 0, time.Local),
		CalendarDay:  1,
		AllTasks:     make([]api.Task, 1000),
		CalendarViewMode: state.CalendarViewExpanded,
		Width: 100, // Important for expanded view
		TaskDates:    make(map[string]time.Time),
	}

	// Populate tasks
	for i := 0; i < 1000; i++ {
		date := "2023-10-15"
		if i%2 == 0 {
			date = "2023-10-16"
		}
		s.AllTasks[i] = api.Task{
			ID: fmt.Sprintf("task-%d", i),
			Due: &api.Due{
				Date: date,
			},
			Content: "Task",
		}
		parsed, _ := time.Parse("2006-01-02", date)
		s.TaskDates[s.AllTasks[i].ID] = parsed
	}

	r := NewRenderer(s)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.renderCalendarExpanded(40)
	}
}
