package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/hy4ri/todoist-tui/internal/tui/state"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// renderCalendar renders the calendar view (dispatches based on view mode).
func (r *Renderer) renderCalendar(maxHeight int) string {
	if r.State.CalendarViewMode == state.CalendarViewExpanded {
		return r.renderCalendarExpanded(maxHeight)
	}
	return r.renderCalendarCompact(maxHeight)
}

// renderCalendarCompact renders the compact calendar view.
func (r *Renderer) renderCalendarCompact(maxHeight int) string {
	var b strings.Builder

	// Header with month/year and navigation hints
	monthYear := r.CalendarDate.Format("January 2006")
	b.WriteString(styles.Title.Copy().Underline(true).Render(strings.ToUpper(monthYear)) + "\n")
	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("← → prev/next month | h l prev/next day | v toggle view"))
	b.WriteString("\n\n")

	// Weekday headers
	weekdays := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	for _, wd := range weekdays {
		b.WriteString(styles.CalendarWeekday.Render(fmt.Sprintf(" %s ", wd)))
	}
	b.WriteString("\n")

	// Calculate first day and number of days in month
	firstOfMonth := time.Date(r.CalendarDate.Year(), r.CalendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	startWeekday := int(firstOfMonth.Weekday())
	daysInMonth := lastOfMonth.Day()
	today := time.Now()

	// Build map of tasks by day
	tasksByDay := make(map[int]int) // day -> count
	for _, t := range r.AllTasks {
		if t.Due == nil {
			continue
		}
		if parsed, err := time.Parse("2006-01-02", t.Due.Date); err == nil {
			if parsed.Year() == r.CalendarDate.Year() && parsed.Month() == r.CalendarDate.Month() {
				tasksByDay[parsed.Day()]++
			}
		}
	}

	// Render calendar grid and count weeks rendered
	day := 1
	weeksRendered := 0
	for week := 0; week < 6; week++ {
		if day > daysInMonth {
			break
		}
		weeksRendered++

		for weekday := 0; weekday < 7; weekday++ {
			if week == 0 && weekday < startWeekday {
				b.WriteString("     ")
				continue
			}

			if day > daysInMonth {
				b.WriteString("     ")
				continue
			}

			dayStr := fmt.Sprintf(" %2d ", day)
			style := styles.CalendarDay

			// Check if this is today
			isToday := today.Year() == r.CalendarDate.Year() &&
				today.Month() == r.CalendarDate.Month() &&
				today.Day() == day

			// Check if this day has tasks
			hasTasks := tasksByDay[day] > 0

			// Check if this is the selected day
			isSelected := day == r.CalendarDay && r.FocusedPane == state.PaneMain

			// Check if this is a weekend (Friday=5, Saturday=6 in Jordan)
			isWeekend := weekday == 5 || weekday == 6

			if isSelected {
				style = styles.CalendarDaySelected
			} else if isToday {
				style = styles.CalendarDayToday
			} else if hasTasks {
				style = styles.CalendarDayWithTasks
			} else if isWeekend {
				style = styles.CalendarDayWeekend
			}

			// Add task indicator
			if hasTasks && !isSelected {
				dayStr = fmt.Sprintf(" %2d*", day)
			}

			b.WriteString(style.Render(dayStr))
			b.WriteString(" ")
			day++
		}
		b.WriteString("\n")
	}

	// Show tasks for selected day
	b.WriteString("\n")
	selectedDate := time.Date(r.CalendarDate.Year(), r.CalendarDate.Month(), r.CalendarDay, 0, 0, 0, 0, time.Local)
	b.WriteString(styles.Subtitle.Render(selectedDate.Format("Monday, January 2")))
	b.WriteString("\n\n")

	// Find tasks for selected day
	var dayTasks []api.Task
	selectedDateStr := selectedDate.Format("2006-01-02")
	for _, t := range r.AllTasks {
		if t.Due != nil && t.Due.Date == selectedDateStr {
			dayTasks = append(dayTasks, t)
		}
	}

	// Calculate remaining height for task list
	// Used: title(1) + help(1) + blank(1) + weekdays(1) + calendar(weeksRendered) + blank(1) + subtitle(1) + blank(1)
	usedHeight := 7 + weeksRendered
	taskListHeight := maxHeight - usedHeight
	if taskListHeight < 1 {
		taskListHeight = 1
	}

	if len(dayTasks) == 0 {
		b.WriteString(styles.HelpDesc.Render("No tasks for this day"))
	} else {
		// Calculate scroll window
		startIdx := 0
		if r.TaskCursor >= taskListHeight {
			startIdx = r.TaskCursor - taskListHeight + 1
		}
		endIdx := startIdx + taskListHeight
		if endIdx > len(dayTasks) {
			endIdx = len(dayTasks)
		}

		for i := startIdx; i < endIdx; i++ {
			t := dayTasks[i]
			checkbox := styles.CheckboxUnchecked
			if t.Checked {
				checkbox = styles.CheckboxChecked
			}
			priorityStyle := styles.GetPriorityStyle(t.Priority)
			content := priorityStyle.Render(t.Content)

			cursor := "  "
			if i == r.TaskCursor && r.FocusedPane == state.PaneMain {
				cursor = "> "
			}
			b.WriteString(fmt.Sprintf("%s%s %s\n", cursor, checkbox, content))
		}
	}

	return b.String()
}

// renderCalendarExpanded renders the expanded calendar view with task names in cells.
func (r *Renderer) renderCalendarExpanded(maxHeight int) string {
	var b strings.Builder

	// Header with month/year and navigation hints
	monthYear := r.CalendarDate.Format("January 2006")
	b.WriteString(styles.Title.Copy().Underline(true).Render(strings.ToUpper(monthYear)) + "\n")
	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("← → prev/next month | h l prev/next day | v toggle view"))
	b.WriteString("\n\n")

	// Calculate cell dimensions based on terminal width
	// 7 columns + borders (8 vertical lines)
	availableWidth := r.Width - 8 // Subtract for borders
	if availableWidth < 35 {
		availableWidth = 35 // Minimum width
	}
	cellWidth := availableWidth / 7
	if cellWidth < 5 {
		cellWidth = 5
	}
	if cellWidth > 20 {
		cellWidth = 20 // Max cell width
	}

	// Weekday headers
	weekdays := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	headerLine := "│"
	for _, wd := range weekdays {
		header := fmt.Sprintf(" %-*s", cellWidth-1, wd)
		if len(header) > cellWidth {
			header = header[:cellWidth]
		}
		headerLine += styles.CalendarWeekday.Render(header) + "│"
	}
	b.WriteString(headerLine)
	b.WriteString("\n")

	// Top border
	topBorder := "├" + strings.Repeat(strings.Repeat("─", cellWidth)+"┼", 6) + strings.Repeat("─", cellWidth) + "┤\n"
	b.WriteString(topBorder)

	// Calculate first day and number of days in month
	firstOfMonth := time.Date(r.CalendarDate.Year(), r.CalendarDate.Month(), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	startWeekday := int(firstOfMonth.Weekday())
	daysInMonth := lastOfMonth.Day()
	today := time.Now()

	// Build map of tasks by day
	tasksByDay := make(map[int][]api.Task) // day -> tasks
	for _, t := range r.AllTasks {
		if t.Due == nil {
			continue
		}
		if parsed, err := time.Parse("2006-01-02", t.Due.Date); err == nil {
			if parsed.Year() == r.CalendarDate.Year() && parsed.Month() == r.CalendarDate.Month() {
				tasksByDay[parsed.Day()] = append(tasksByDay[parsed.Day()], t)
			}
		}
	}

	// Calculate how many weeks we need to display
	weeksNeeded := (daysInMonth + startWeekday + 6) / 7

	// Calculate how many task lines to show per cell based on available height
	// Header(1) + help(1) + blank(1) + weekday(1) + topBorder(1) + statusBar(1) = 6 lines overhead
	// Each week uses: 1 (day number) + maxTasksPerCell (tasks) + 1 (separator) = 2 + maxTasksPerCell
	// Last week doesn't have separator, so: weeksNeeded * (2 + maxTasksPerCell) - 1 + 6 = maxHeight
	availableForWeeks := maxHeight - 6
	if availableForWeeks < weeksNeeded*3 {
		availableForWeeks = weeksNeeded * 3 // Minimum 1 task line per cell
	}
	// Each week row = 1 (day) + tasks + 1 (separator, except last)
	// Solve for maxTasksPerCell: weeksNeeded*(1+tasks+1) - 1 = availableForWeeks
	// weeksNeeded*(2+tasks) = availableForWeeks + 1
	// tasks = (availableForWeeks + 1) / weeksNeeded - 2
	maxTasksPerCell := (availableForWeeks+1)/weeksNeeded - 2
	if maxTasksPerCell < 2 {
		maxTasksPerCell = 2
	}
	if maxTasksPerCell > 6 {
		maxTasksPerCell = 6 // Cap at 6 tasks per cell
	}

	// Render calendar grid
	day := 1
	for week := 0; week < 6; week++ {
		if day > daysInMonth {
			break
		}

		// Row 1: Day numbers
		dayNumLine := "│"
		for weekday := 0; weekday < 7; weekday++ {
			if week == 0 && weekday < startWeekday || day > daysInMonth {
				dayNumLine += strings.Repeat(" ", cellWidth) + "│"
				if week == 0 && weekday < startWeekday {
					continue
				}
				continue
			}

			dayStr := fmt.Sprintf(" %2d", day)
			style := styles.CalendarDay

			isToday := today.Year() == r.CalendarDate.Year() &&
				today.Month() == r.CalendarDate.Month() &&
				today.Day() == day
			isSelected := day == r.CalendarDay && r.FocusedPane == state.PaneMain
			isWeekend := weekday == 5 || weekday == 6
			hasTasks := len(tasksByDay[day]) > 0

			if isSelected {
				style = styles.CalendarDaySelected
			} else if isToday {
				style = styles.CalendarDayToday
			} else if hasTasks {
				style = styles.CalendarDayWithTasks
			} else if isWeekend {
				style = styles.CalendarDayWeekend
			}

			// Pad to cell width
			paddedDay := fmt.Sprintf("%-*s", cellWidth, dayStr)
			dayNumLine += style.Render(paddedDay) + "│"
			day++
		}
		b.WriteString(dayNumLine)
		b.WriteString("\n")

		// Reset day counter for task rows
		day -= 7
		if day < 1 {
			day = 1
		}

		// Rows 2-3: Task previews
		for taskLine := 0; taskLine < maxTasksPerCell; taskLine++ {
			var taskRowBuilder strings.Builder
			taskRowBuilder.WriteString("│")
			tempDay := day
			for weekday := 0; weekday < 7; weekday++ {
				if week == 0 && weekday < startWeekday {
					taskRowBuilder.WriteString(strings.Repeat(" ", cellWidth) + "│")
					continue
				}

				if tempDay > daysInMonth {
					taskRowBuilder.WriteString(strings.Repeat(" ", cellWidth) + "│")
					tempDay++
					continue
				}

				tasks := tasksByDay[tempDay]
				var cellContent string

				if taskLine < len(tasks) && taskLine < maxTasksPerCell-1 {
					// Show task name with priority color (truncated to fit cell)
					task := tasks[taskLine]
					taskName := task.Content
					maxLen := cellWidth - 2 // Leave space for " " prefix and margin
					if len(taskName) > maxLen && maxLen > 1 {
						taskName = taskName[:maxLen-1] + "…"
					}
					// Pad the plain text first, then apply priority color
					paddedTask := fmt.Sprintf(" %-*s", cellWidth-1, taskName)
					priorityStyle := styles.GetPriorityStyle(task.Priority)
					cellContent = priorityStyle.Render(paddedTask)
				} else if taskLine == maxTasksPerCell-1 && len(tasks) > maxTasksPerCell-1 {
					// Show "+N more" indicator on the last line if there are more tasks
					hiddenCount := len(tasks) - (maxTasksPerCell - 1)
					moreText := fmt.Sprintf("+%d more", hiddenCount)
					paddedMore := fmt.Sprintf(" %-*s", cellWidth-1, moreText)
					cellContent = styles.CalendarMoreTasks.Render(paddedMore)
				} else if taskLine < len(tasks) {
					// This handles the case where we're on the last allowed line but it's a task
					task := tasks[taskLine]
					taskName := task.Content
					maxLen := cellWidth - 2
					if len(taskName) > maxLen && maxLen > 1 {
						taskName = taskName[:maxLen-1] + "…"
					}
					paddedTask := fmt.Sprintf(" %-*s", cellWidth-1, taskName)
					priorityStyle := styles.GetPriorityStyle(task.Priority)
					cellContent = priorityStyle.Render(paddedTask)
				} else {
					// Empty cell
					cellContent = strings.Repeat(" ", cellWidth)
				}

				taskRowBuilder.WriteString(cellContent + "│")
				tempDay++
			}
			b.WriteString(taskRowBuilder.String())
			b.WriteString("\n")
		}

		// Move day forward after processing the week
		day += 7
		if week == 0 {
			day = 8 - startWeekday
		}

		// Row separator (except for last week)
		if day <= daysInMonth {
			separator := "├" + strings.Repeat(strings.Repeat("─", cellWidth)+"┼", 6) + strings.Repeat("─", cellWidth) + "┤\n"
			b.WriteString(separator)
		}
	}

	// Bottom border
	bottomBorder := "└" + strings.Repeat(strings.Repeat("─", cellWidth)+"┴", 6) + strings.Repeat("─", cellWidth) + "┘\n"
	b.WriteString(bottomBorder)

	return b.String()
}

// renderCalendarDay renders the day detail view showing all tasks for the selected calendar day.
func (r *Renderer) renderCalendarDay() string {
	var b strings.Builder

	// Header with date - styled nicely
	selectedDate := time.Date(r.CalendarDate.Year(), r.CalendarDate.Month(), r.CalendarDay, 0, 0, 0, 0, time.Local)

	// Title bar
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Highlight).
		Background(lipgloss.Color("#1a1a2e")).
		Padding(0, 1).
		Width(r.Width - 4)

	b.WriteString(titleStyle.Render("📅 " + selectedDate.Format("Monday, January 2, 2006")))
	b.WriteString("\n\n")

	if r.Loading {
		b.WriteString(r.Spinner.View())
		b.WriteString(" Loading tasks...")
		return b.String()
	}

	// Content area with border
	contentWidth := r.Width - 4
	contentHeight := r.Height - 6 // title + padding + status bar
	if contentHeight < 5 {
		contentHeight = 5
	}

	var content strings.Builder

	if len(r.Tasks) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(styles.Subtle).
			Italic(true).
			Align(lipgloss.Center).
			Width(contentWidth - 4)

		content.WriteString("\n")
		content.WriteString(emptyStyle.Render("No tasks scheduled for this day"))
		content.WriteString("\n\n")
		content.WriteString(emptyStyle.Render("Press 'a' to add a new task"))
	} else {
		// Task count header
		countStyle := lipgloss.NewStyle().
			Foreground(styles.Subtle)
		content.WriteString(countStyle.Render(fmt.Sprintf("%d task(s)", len(r.Tasks))))
		content.WriteString("\n\n")

		// Build task lines
		taskHeight := contentHeight - 4 // account for count header and padding
		if taskHeight < 3 {
			taskHeight = 3
		}

		var lines []lineInfo
		var orderedIndices []int
		for i := range r.Tasks {
			orderedIndices = append(orderedIndices, i)
			lines = append(lines, lineInfo{
				content:   r.renderTaskByDisplayIndex(i, orderedIndices, contentWidth),
				taskIndex: i,
			})
		}

		content.WriteString(r.renderScrollableLines(lines, orderedIndices, taskHeight))
	}

	// Wrap in a nice container with good padding
	containerStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(styles.Subtle).
		Padding(1, 3). // More vertical and horizontal padding
		MarginLeft(2).
		MarginRight(2).
		Width(contentWidth - 4) // Account for margins

	b.WriteString(containerStyle.Render(content.String()))

	return b.String()
}

// renderStatusBar renders the bottom status bar.
func (r *Renderer) renderStatusBar() string {
	// Left side: status message or error
	left := ""
	if r.Err != nil {
		errStr := strings.ReplaceAll(r.Err.Error(), "\n", " ")
		left = styles.StatusBarError.Render("Error: " + errStr)
	} else if r.StatusMsg != "" {
		msgStr := strings.ReplaceAll(r.StatusMsg, "\n", " ")
		left = styles.StatusBarSuccess.Render(msgStr)
	}

	// Right side: context-specific key hints (or just toggle hint if hidden)
	var right string
	if r.ShowHints {
		hints := r.getContextualHints()
		hints = append(hints, styles.StatusBarKey.Render("F1")+styles.StatusBarText.Render(":hide"))
		right = strings.Join(hints, " ")
	} else {
		right = styles.StatusBarKey.Render("F1") + styles.StatusBarText.Render(":keys")
	}

	// Calculate spacing
	leftWidth := lipgloss.Width(left)
	rightWidth := lipgloss.Width(right)
	padding := styles.StatusBar.GetHorizontalFrameSize()

	// Ensure left doesn't overwhelm right
	maxLeftWidth := r.Width - rightWidth - padding - 4
	if leftWidth > maxLeftWidth && maxLeftWidth > 10 {
		left = truncateString(left, maxLeftWidth)
		leftWidth = lipgloss.Width(left)
	}

	spacing := r.Width - leftWidth - rightWidth - padding
	if spacing < 0 {
		spacing = 0
	}

	return styles.StatusBar.Width(r.Width - padding).Render(left + strings.Repeat(" ", spacing) + right)
}

// getContextualHints returns context-specific key hints for the status bar.
func (r *Renderer) getContextualHints() []string {
	key := func(k string) string { return styles.StatusBarKey.Render(k) }
	desc := func(d string) string { return styles.StatusBarText.Render(d) }

	switch r.CurrentTab {
	case state.TabToday, state.TabUpcoming:
		return []string{
			key("j/k") + desc(":nav"),
			key("x") + desc(":done"),
			key("e") + desc(":edit"),
			key("</>") + desc(":due"),
			key("r") + desc(":refresh"),
			key("?") + desc(":help"),
		}
	case state.TabLabels:
		if r.CurrentLabel != nil {
			return []string{
				key("j/k") + desc(":nav"),
				key("x") + desc(":done"),
				key("e") + desc(":edit"),
				key("Esc") + desc(":back"),
				key("?") + desc(":help"),
			}
		}
		return []string{
			key("j/k") + desc(":nav"),
			key("Enter") + desc(":select"),
			key("?") + desc(":help"),
			key("q") + desc(":quit"),
		}
	case state.TabCalendar:
		return []string{
			key("h/l") + desc(":day"),
			key("←/→") + desc(":month"),
			key("v") + desc(":view"),
			key("Enter") + desc(":select"),
			key("?") + desc(":help"),
		}
	case state.TabProjects:
		if r.FocusedPane == state.PaneSidebar {
			return []string{
				key("j/k") + desc(":nav"),
				key("Enter") + desc(":select"),
				key("state.Tab") + desc(":pane"),
				key("?") + desc(":help"),
			}
		}
		return []string{
			key("j/k") + desc(":nav"),
			key("x") + desc(":done"),
			key("e") + desc(":edit"),
			key("state.Tab") + desc(":pane"),
			key("?") + desc(":help"),
		}
	default:
		return []string{
			key("j/k") + desc(":nav"),
			key("?") + desc(":help"),
			key("q") + desc(":quit"),
		}
	}
}

// updateSectionOrderCmd sends an API request to update the section order.

// truncateString truncates a string to a given width and adds an ellipsis if truncated.
func truncateString(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}

	// Very basic truncation that handles some ANSI/multi-byte
	// In a real app we'd use a more robust version, but this fits the immediate need.
	if width <= 1 {
		return "…"
	}

	// Fallback to simpler character-at-a-time width check if needed,
	// but lipgloss.Width is usually reliable for measurement.
	res := s
	for lipgloss.Width(res+"…") > width && len(res) > 0 {
		// Remove one character/byte at a time until it fits
		_, size := utf8.DecodeLastRuneInString(res)
		res = res[:len(res)-size]
	}
	return res + "…"
}
