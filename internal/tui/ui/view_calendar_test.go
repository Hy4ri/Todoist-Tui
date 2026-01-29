package ui

import (
	"fmt"
	"testing"
	"time"

	"github.com/hy4ri/todoist-tui/internal/api"
)

func BenchmarkDateParsing_Baseline(b *testing.B) {
	// Setup
	numTasks := 1000
	tasks := make([]api.Task, numTasks)
	for i := 0; i < numTasks; i++ {
		dateStr := "2023-10-27"
		tasks[i] = api.Task{
			Due: &api.Due{
				Date: dateStr,
			},
		}
	}
	calendarDate := time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tasksByDay := make(map[int]int)
		for _, t := range tasks {
			if t.Due == nil {
				continue
			}
			if parsed, err := time.Parse("2006-01-02", t.Due.Date); err == nil {
				if parsed.Year() == calendarDate.Year() && parsed.Month() == calendarDate.Month() {
					tasksByDay[parsed.Day()]++
				}
			}
		}
	}
}

func BenchmarkDateParsing_Optimized(b *testing.B) {
	// Setup
	numTasks := 1000
	tasks := make([]api.Task, numTasks)
	taskDates := make(map[string]time.Time)

	for i := 0; i < numTasks; i++ {
		dateStr := "2023-10-27"
		id := fmt.Sprintf("task-%d", i)
		tasks[i] = api.Task{
			ID: id,
			Due: &api.Due{
				Date: dateStr,
			},
		}
		if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
			taskDates[id] = parsed
		}
	}
	calendarDate := time.Date(2023, 10, 1, 0, 0, 0, 0, time.UTC)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tasksByDay := make(map[int]int)
		for _, t := range tasks {
			if t.Due == nil {
				continue
			}
			if parsed, ok := taskDates[t.ID]; ok {
				if parsed.Year() == calendarDate.Year() && parsed.Month() == calendarDate.Month() {
					tasksByDay[parsed.Day()]++
				}
			}
		}
	}
}
