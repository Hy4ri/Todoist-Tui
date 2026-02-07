package ui

import (
	"fmt"
	"strings"
	"time"

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
	// Build map of tasks by day using TasksByDate cache
	tasksByDay := make(map[int]int) // day -> count
	for d := 1; d <= daysInMonth; d++ {
		dateStr := fmt.Sprintf("%04d-%02d-%02d", r.CalendarDate.Year(), r.CalendarDate.Month(), d)
		if tasks, ok := r.TasksByDate[dateStr]; ok {
			tasksByDay[d] = len(tasks)
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

	// Find tasks for selected day using cache
	dateStr := selectedDate.Format("2006-01-02")
	dayTasks := r.TasksByDate[dateStr]

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
	var headerBuilder strings.Builder
	headerBuilder.WriteString("│")
	for _, wd := range weekdays {
		header := fmt.Sprintf(" %-*s", cellWidth-1, wd)
		if len(header) > cellWidth {
			header = header[:cellWidth]
		}
		headerBuilder.WriteString(styles.CalendarWeekday.Render(header) + "│")
	}
	b.WriteString(headerBuilder.String())
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

	// Build map of tasks by day using TasksByDate cache
	tasksByDay := make(map[int][]api.Task) // day -> tasks
	for d := 1; d <= daysInMonth; d++ {
		dateStr := fmt.Sprintf("%04d-%02d-%02d", r.CalendarDate.Year(), r.CalendarDate.Month(), d)
		if tasks, ok := r.TasksByDate[dateStr]; ok {
			tasksByDay[d] = tasks
		}
	}

	// Calculate how many weeks we need to display
	weeksNeeded := (daysInMonth + startWeekday + 6) / 7

	// Calculate how many task lines to show per cell based on available height
	// Header(1) + help(1) + blank(1) + weekday(1) + topBorder(1) + statusBar(1?) + bottomBorder(1) + margin(1) = 8 lines overhead
	// We bump safety to 8 lines to ensure we don't overflow, especially with the bottom border.
	availableForWeeks := maxHeight - 8
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
		var dayNumBuilder strings.Builder
		dayNumBuilder.WriteString("│")
		for weekday := 0; weekday < 7; weekday++ {
			if week == 0 && weekday < startWeekday || day > daysInMonth {
				dayNumBuilder.WriteString(strings.Repeat(" ", cellWidth) + "│")
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
			dayNumBuilder.WriteString(style.Render(paddedDay) + "│")
			day++
		}
		b.WriteString(dayNumBuilder.String())
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
			pos := i
			lines = append(lines, lineInfo{
				content:   r.renderTaskByDisplayIndex(i, pos, contentWidth),
				taskIndex: i,
			})
		}

		content.WriteString(r.renderScrollableLines(lines, orderedIndices, taskHeight, contentWidth))
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
	// Left side: status message or error, followed by goals
	left := ""
	if r.Err != nil {
		errStr := strings.ReplaceAll(r.Err.Error(), "\n", " ")
		left = styles.StatusBarError.Render("Error: " + errStr)
	} else if r.StatusMsg != "" {
		msgStr := strings.ReplaceAll(r.StatusMsg, "\n", " ")
		left = styles.StatusBarSuccess.Render(msgStr)
	}
	var rightParts []string

	// Add goals to right side
	goalsDisplay := r.renderGoalsDisplay()
	if goalsDisplay != "" {
		rightParts = append(rightParts, goalsDisplay)
	} else if r.StatsError != "" {
		// Show the error in red
		errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
		rightParts = append(rightParts, errStyle.Render("Stats Err: "+r.StatsError))
	} else if r.ProductivityStats == nil {
		// Debug: show if stats are missing
		// rightParts = append(rightParts, styles.StatusBarText.Render("[No Stats]"))
	}

	if r.ShowHints {
		hints := r.getContextualHints()
		hints = append(hints, styles.StatusBarKey.Render("F1")+styles.StatusBarText.Render(":hide"))
		rightParts = append(rightParts, strings.Join(hints, " "))
	} else {
		rightParts = append(rightParts, styles.StatusBarKey.Render("F1")+styles.StatusBarText.Render(":keys"))
	}

	right := strings.Join(rightParts, "  ") // Spacer between goals and keys

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

// renderGoalsDisplay renders the daily/weekly goal progress.
func (r *Renderer) renderGoalsDisplay() string {
	if r.ProductivityStats == nil {
		return ""
	}

	goals := r.ProductivityStats.Goals
	if goals.DailyGoal == 0 && goals.WeeklyGoal == 0 {
		return ""
	}

	// Get today's completed count
	todayCompleted := 0
	todayStr := time.Now().Format("2006-01-02")
	for _, day := range r.ProductivityStats.DaysItems {
		if day.Date == todayStr {
			todayCompleted = day.TotalCompleted
			break
		}
	}

	// Get this week's completed count (use first week item which is current week)
	weekCompleted := 0
	if len(r.ProductivityStats.WeekItems) > 0 {
		weekCompleted = r.ProductivityStats.WeekItems[0].TotalCompleted
	}

	var parts []string

	// Daily goal
	if goals.DailyGoal > 0 {
		dailyStyle := styles.StatusBarText
		icon := "📅"
		if todayCompleted >= goals.DailyGoal {
			dailyStyle = styles.StatusBarSuccess
			icon = "✓"
		}
		parts = append(parts, dailyStyle.Render(fmt.Sprintf("%s %d/%d", icon, todayCompleted, goals.DailyGoal)))
	}

	// Weekly goal
	if goals.WeeklyGoal > 0 {
		weeklyStyle := styles.StatusBarText
		icon := "📆"
		if weekCompleted >= goals.WeeklyGoal {
			weeklyStyle = styles.StatusBarSuccess
			icon = "✓"
		}
		parts = append(parts, weeklyStyle.Render(fmt.Sprintf("%s %d/%d", icon, weekCompleted, goals.WeeklyGoal)))
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, " ")
}

// getContextualHints returns context-specific key hints for the status bar.
func (r *Renderer) getContextualHints() []string {
	key := func(k string) string { return styles.StatusBarKey.Render(k) }
	desc := func(d string) string { return styles.StatusBarText.Render(d) }

	// 1. Check Overlays & Modals (Highest priority)
	if r.IsCreatingProject || r.IsEditingProject {
		return []string{
			key("Enter") + desc(":save"),
			key("Esc") + desc(":cancel"),
		}
	}
	if r.IsCreatingLabel || r.IsEditingLabel {
		return []string{
			key("Enter") + desc(":save"),
			key("Esc") + desc(":cancel"),
		}
	}
	if r.IsCreatingSection || r.IsEditingSection {
		return []string{
			key("Enter") + desc(":save"),
			key("Esc") + desc(":cancel"),
		}
	}
	if r.IsCreatingSubtask {
		return []string{
			key("Enter") + desc(":save"),
			key("Esc") + desc(":cancel"),
		}
	}
	if r.IsAddingComment || r.IsEditingComment {
		return []string{
			key("Enter") + desc(":save"),
			key("Esc") + desc(":cancel"),
		}
	}
	if r.IsMovingTask {
		return []string{
			key("j/k") + desc(":move"),
			key("Enter") + desc(":place"),
			key("Esc") + desc(":cancel"),
		}
	}
	if r.ConfirmDeleteProject || r.ConfirmDeleteLabel || r.ConfirmDeleteSection || r.ConfirmDeleteComment {
		return []string{
			key("y") + desc(":confirm"),
			key("n") + desc(":cancel"),
		}
	}
	if r.IsSelectingColor {
		return []string{
			key("j/k") + desc(":nav"),
			key("Enter") + desc(":select"),
			key("Esc") + desc(":back"),
		}
	}

	// 2. Tab-based Context
	switch r.CurrentTab {
	case state.TabToday, state.TabUpcoming, state.TabInbox:
		return []string{
			key("j/k") + desc(":nav"),
			key("a") + desc(":add"),
			key("x") + desc(":done"),
			key("e") + desc(":edit"),
			key("s") + desc(":subtask"),
			key("?") + desc(":help"),
		}
	case state.TabLabels:
		if r.CurrentLabel != nil {
			// Inside a label view
			return []string{
				key("Esc") + desc(":back"),
				key("j/k") + desc(":nav"),
				key("a") + desc(":add"),
				key("x") + desc(":done"),
			}
		}
		// List of labels
		return []string{
			key("j/k") + desc(":nav"),
			key("Enter") + desc(":select"),
			key("a") + desc(":create"),
			key("d") + desc(":delete"),
		}
	case state.TabCalendar:
		return []string{
			key("h/l") + desc(":day"),
			key("j/k") + desc(":week"),
			key("v") + desc(":view"),
			key("Enter") + desc(":open"),
		}
	case state.TabProjects:
		if r.FocusedPane == state.PaneSidebar {
			return []string{
				key("j/k") + desc(":nav"),
				key("Enter") + desc(":open"),
				key("n") + desc(":new"),
				key("f") + desc(":fav"),
				key("Tab") + desc(":tasks"),
			}
		}
		// Focused on tasks in project
		return []string{
			key("Tab") + desc(":projects"),
			key("j/k") + desc(":nav"),
			key("a") + desc(":add"),
			key("x") + desc(":done"),
			key("S") + desc(":sections"),
		}
	case state.TabFilters:
		if r.FocusedPane == state.PaneSidebar {
			return []string{
				key("j/k") + desc(":nav"),
				key("/") + desc(":search"),
				key("Enter") + desc(":run"),
				key("h/l") + desc(":pane"),
			}
		}
		// Focused on tasks from filter
		return []string{
			key("Tab") + desc(":filters"),
			key("j/k") + desc(":nav"),
			key("a") + desc(":add"),
			key("x") + desc(":done"),
		}
	}

	// Default generic fallback
	return []string{
		key("j/k") + desc(":nav"),
		key("?") + desc(":help"),
		key("q") + desc(":quit"),
	}
}

// updateSectionOrderCmd sends an API request to update the section order.
