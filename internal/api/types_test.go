package api

import (
	"testing"
	"time"
)

func utcMidnightOffset(days int) time.Time {
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return today.AddDate(0, 0, days)
}

func TestTaskDueHelpers(t *testing.T) {
	tz := "UTC"
	loc := time.UTC
	deadline := utcMidnightOffset(-1)

	cases := []struct {
		name      string
		task      Task
		wantOver  bool
		wantToday bool
		wantDisp  string
	}{
		{
			name: "no due date",
			task: Task{},
		},
		{
			name:     "overdue date only",
			task:     Task{Due: &Due{Date: deadline.Format("2006-01-02"), String: "yesterday", Timezone: &tz}},
			wantOver: true,
			wantDisp: "yesterday",
		},
		{
			name:      "due today with parsed date",
			task:      Task{Due: &Due{Date: utcMidnightOffset(0).Format("2006-01-02"), String: "today", Timezone: &tz}, ParsedDate: func() *time.Time { t := utcMidnightOffset(0); return &t }()},
			wantToday: true,
			wantDisp:  "today",
		},
		{
			name:     "future timestamp with parsed date",
			task:     Task{Due: &Due{Date: utcMidnightOffset(2).Format(time.RFC3339), String: "next", Datetime: func() *string { s := utcMidnightOffset(2).Format(time.RFC3339); return &s }(), Timezone: &tz}, ParsedDate: func() *time.Time { t := utcMidnightOffset(2); return &t }()},
			wantDisp: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.task.Due != nil && tc.task.Due.Timezone == nil {
				tc.task.Due.Timezone = &tz
			}
			if got := tc.task.IsOverdue(); got != tc.wantOver {
				t.Fatalf("IsOverdue() = %v, want %v", got, tc.wantOver)
			}
			if got := tc.task.IsDueToday(); got != tc.wantToday {
				t.Fatalf("IsDueToday() = %v, want %v", got, tc.wantToday)
			}
			if tc.task.Due == nil {
				if got := tc.task.DueDisplay(); got != "" {
					t.Fatalf("DueDisplay() = %q, want empty", got)
				}
				return
			}
			if got := tc.task.DueDisplay(); tc.wantDisp != "" && got != tc.wantDisp {
				t.Fatalf("DueDisplay() = %q, want %q", got, tc.wantDisp)
			}
			_ = loc
		})
	}
}

func TestTaskLocationFallsBackToLocalOnUnknownTimezone(t *testing.T) {
	bad := "Mars/Phobos"
	task := Task{Due: &Due{Timezone: &bad}}
	if got := task.Location(); got != time.Local {
		t.Fatalf("Location() = %v, want time.Local", got)
	}
}

func TestTaskDueDisplayFallsBackToRawString(t *testing.T) {
	raw := "tomorrow at 6pm"
	task := Task{Due: &Due{String: raw, Date: "not-a-date"}}
	if got := task.DueDisplay(); got != raw {
		t.Fatalf("DueDisplay() = %q, want %q", got, raw)
	}
}
