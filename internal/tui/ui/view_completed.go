package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hy4ri/todoist-tui/internal/tui/state"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// renderCompletedTaskList renders the completed tasks view grouped by completion date.
func (r *Renderer) renderCompletedTaskList(width, maxHeight int) string {
	var b strings.Builder

	b.WriteString(styles.Title.Copy().Underline(true).Render("COMPLETED TASKS") + "\n\n")

	if r.Loading {
		b.WriteString(r.Spinner.View())
		b.WriteString(" Loading...")
		return b.String()
	}

	if len(r.Tasks) == 0 {
		b.WriteString("\n")
		b.WriteString(styles.HelpDesc.Render("No completed tasks found in history."))
		return b.String()
	}

	// Group tasks by completion date (YYYY-MM-DD)
	tasksByDate := make(map[string][]int)
	var dates []string

	for i, t := range r.Tasks {
		var dateStr string
		if t.CompletedAt != nil {
			// RFC3339 format: "2023-10-25T14:30:00Z"
			parsed, err := time.Parse(time.RFC3339, *t.CompletedAt)
			if err == nil {
				dateStr = parsed.Local().Format("2006-01-02")
			} else {
				dateStr = "Unknown Date"
			}
		} else {
			dateStr = "Unknown Date"
		}

		if _, exists := tasksByDate[dateStr]; !exists {
			dates = append(dates, dateStr)
		}
		tasksByDate[dateStr] = append(tasksByDate[dateStr], i)
	}

	// Sort dates descending (newest first)
	sort.Slice(dates, func(i, j int) bool {
		return dates[i] > dates[j]
	})

	// Build lines and ordered indices for correct scrolling/cursor logic
	// We interleave headers and tasks so headers are selectable
	var orderedIndices []int
	var lines []lineInfo

	for idx, date := range dates {
		// Calculate display date string
		displayDate := date
		if parsed, err := time.Parse("2006-01-02", date); err == nil {
			today := time.Now().Format("2006-01-02")
			yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

			switch date {
			case today:
				displayDate = "Today"
			case yesterday:
				displayDate = "Yesterday"
			default:
				displayDate = parsed.Format("Mon, Jan 2")
			}
		}

		// Add blank line before header (except first)
		if idx > 0 {
			lines = append(lines, lineInfo{content: "", taskIndex: -1})
		}

		// Create section header index (unique negative value)
		headerIndex := -100 - len(orderedIndices)
		orderedIndices = append(orderedIndices, headerIndex)

		// Header Line
		lines = append(lines, lineInfo{
			content:   r.renderCompletedHeader(displayDate, headerIndex, orderedIndices),
			taskIndex: headerIndex,
		})

		for _, i := range tasksByDate[date] {
			i, displayPos := i, len(orderedIndices)
			orderedIndices = append(orderedIndices, i)

			lines = append(lines, lineInfo{
				renderFunc: func(w int) string { return r.renderCompletedTaskItem(i, displayPos, w) },
				taskIndex:  i,
			})
			if r.Tasks[i].Description != "" {
				desc := r.Tasks[i].Description
				lines = append(lines, lineInfo{
					renderFunc: func(w int) string { return r.renderTaskDescription(desc, w) },
					taskIndex:  i,
				})
			}
		}
	}

	// Add footer hint for "Load More"
	if r.State.CompletedMore {
		lines = append(lines, lineInfo{content: "", taskIndex: -1})
		lines = append(lines, lineInfo{
			content:   styles.HelpDesc.Render("Scroll down to load more... (Not implemented yet)"),
			taskIndex: -1,
		})
	}

	// Render scrollable lines
	// height - 2 (Title + newline)
	return r.renderScrollableLines(lines, orderedIndices, maxHeight-2, width)
}

// renderCompletedHeader renders a date header with cursor highlighting.
func (r *Renderer) renderCompletedHeader(dateStr string, headerIndex int, orderedIndices []int) string {
	// Find display position for cursor
	displayPos := 0
	for i, idx := range orderedIndices {
		if idx == headerIndex {
			displayPos = i
			break
		}
	}

	// Check if cursor is on this header
	isCursorHere := displayPos == r.TaskCursor && r.FocusedPane == state.PaneMain

	// Cursor indicator
	cursor := "  "
	if isCursorHere {
		cursor = "> "
	}

	// Build the header line
	line := fmt.Sprintf("%s%s", cursor, dateStr)

	// Apply style - use TaskSelected style when cursor is here, otherwise DateGroupHeader
	style := styles.DateGroupHeader
	if isCursorHere {
		style = styles.TaskSelected
	}

	return style.Render(line)
}

// renderCompletedTaskItem renders a single completed task line with time info.
func (r *Renderer) renderCompletedTaskItem(taskIndex int, displayPos int, width int) string {
	t := r.Tasks[taskIndex]

	// Cursor
	cursor := "  "
	if displayPos == r.TaskCursor && r.FocusedPane == state.PaneMain {
		cursor = "> "
	}

	// Checkbox (always checked)
	checkbox := styles.CheckboxChecked

	// Time
	timeStr := ""
	if t.CompletedAt != nil {
		if parsed, err := time.Parse(time.RFC3339, *t.CompletedAt); err == nil {
			timeStr = parsed.Local().Format("15:04") // HH:MM
		}
	}
	styledTime := styles.HelpDesc.Render(timeStr) // Dim style

	// Content - Strikethrough style
	content := t.Content

	// Calculate overhead
	// cursor(2) + box(4) + time(5+1) + label overhead
	// "> [x] 14:30 Task Content"
	overhead := 2 + 4 + 6

	maxContentWidth := width - overhead
	if maxContentWidth < 10 {
		maxContentWidth = 10
	}
	content = truncateString(content, maxContentWidth)

	styledContent := styles.TaskCompleted.Render(content)

	line := fmt.Sprintf("%s%s %s %s", cursor, checkbox, styledTime, styledContent)

	// Apply selection style if active
	if displayPos == r.TaskCursor && r.FocusedPane == state.PaneMain {
		return styles.TaskSelected.MaxWidth(width).Render(line)
	}

	return styles.TaskItem.MaxWidth(width).Render(line)
}
