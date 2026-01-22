package components

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// CalendarViewMode represents the calendar display mode.
type CalendarViewModeType int

const (
	CalendarViewModeCompact  CalendarViewModeType = iota // Small grid view
	CalendarViewModeExpanded                             // Grid with task names in cells
)

// DaySelectedMsg is emitted when a day is selected for detail view.
type DaySelectedMsg struct {
	Date time.Time
}

// MonthChangedMsg is emitted when the month is changed.
type MonthChangedMsg struct {
	Date time.Time
}

// CalendarModel manages the calendar view.
type CalendarModel struct {
	date          time.Time
	day           int
	viewMode      CalendarViewModeType
	tasks         []api.Task
	width, height int
	focused       bool
}

// NewCalendar creates a new CalendarModel.
func NewCalendar() *CalendarModel {
	now := time.Now()
	return &CalendarModel{
		date:     now,
		day:      now.Day(),
		viewMode: CalendarViewModeCompact,
		tasks:    []api.Task{},
		focused:  false,
	}
}

// Init implements Component.
func (c *CalendarModel) Init() tea.Cmd {
	return nil
}

// Update implements Component.
func (c *CalendarModel) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return c.handleKeyMsg(msg)
	}
	return c, nil
}

// handleKeyMsg processes keyboard input for calendar navigation.
func (c *CalendarModel) handleKeyMsg(msg tea.KeyMsg) (Component, tea.Cmd) {
	firstOfMonth := time.Date(c.date.Year(), c.date.Month(), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	daysInMonth := lastOfMonth.Day()

	switch msg.String() {
	case "h", "left":
		// Previous day
		if c.day > 1 {
			c.day--
		} else {
			c.date = c.date.AddDate(0, -1, 0)
			prevMonth := time.Date(c.date.Year(), c.date.Month(), 1, 0, 0, 0, 0, time.Local)
			c.day = prevMonth.AddDate(0, 1, -1).Day()
		}
	case "l", "right":
		// Next day
		if c.day < daysInMonth {
			c.day++
		} else {
			c.date = c.date.AddDate(0, 1, 0)
			c.day = 1
		}
	case "k", "up":
		// Previous week
		if c.day > 7 {
			c.day -= 7
		} else {
			c.date = c.date.AddDate(0, -1, 0)
			prevMonth := time.Date(c.date.Year(), c.date.Month(), 1, 0, 0, 0, 0, time.Local)
			prevDays := prevMonth.AddDate(0, 1, -1).Day()
			newDay := c.day - 7 + prevDays
			if newDay > prevDays {
				newDay = prevDays
			}
			c.day = newDay
		}
	case "j", "down":
		// Next week
		if c.day+7 <= daysInMonth {
			c.day += 7
		} else {
			leftover := c.day + 7 - daysInMonth
			c.date = c.date.AddDate(0, 1, 0)
			nextMonth := time.Date(c.date.Year(), c.date.Month(), 1, 0, 0, 0, 0, time.Local)
			nextDays := nextMonth.AddDate(0, 1, -1).Day()
			if leftover > nextDays {
				leftover = nextDays
			}
			c.day = leftover
		}
	case "[":
		// Previous month
		c.date = c.date.AddDate(0, -1, 0)
		prevMonth := time.Date(c.date.Year(), c.date.Month(), 1, 0, 0, 0, 0, time.Local)
		prevDays := prevMonth.AddDate(0, 1, -1).Day()
		if c.day > prevDays {
			c.day = prevDays
		}
	case "]":
		// Next month
		c.date = c.date.AddDate(0, 1, 0)
		nextMonth := time.Date(c.date.Year(), c.date.Month(), 1, 0, 0, 0, 0, time.Local)
		nextDays := nextMonth.AddDate(0, 1, -1).Day()
		if c.day > nextDays {
			c.day = nextDays
		}
	case "t":
		// Go to today
		c.date = time.Now()
		c.day = time.Now().Day()
	case "v":
		// Toggle view mode
		if c.viewMode == CalendarViewModeCompact {
			c.viewMode = CalendarViewModeExpanded
		} else {
			c.viewMode = CalendarViewModeCompact
		}
	case "enter":
		// Open day detail view
		selectedDate := time.Date(c.date.Year(), c.date.Month(), c.day, 0, 0, 0, 0, time.Local)
		return c, func() tea.Msg {
			return DaySelectedMsg{Date: selectedDate}
		}
	}
	return c, nil
}

// View implements Component.
func (c *CalendarModel) View() string {
	if c.viewMode == CalendarViewModeExpanded {
		return c.renderExpanded()
	}
	return c.renderCompact()
}

// SetSize implements Component.
func (c *CalendarModel) SetSize(width, height int) {
	c.width = width
	c.height = height
}

// Focus sets focus on the calendar.
func (c *CalendarModel) Focus() {
	c.focused = true
}

// Blur removes focus.
func (c *CalendarModel) Blur() {
	c.focused = false
}

// Focused returns focus state.
func (c *CalendarModel) Focused() bool {
	return c.focused
}

// SetTasks updates the tasks for the calendar.
func (c *CalendarModel) SetTasks(tasks []api.Task) {
	c.tasks = tasks
}

// SelectedDate returns the currently selected date.
func (c *CalendarModel) SelectedDate() time.Time {
	return time.Date(c.date.Year(), c.date.Month(), c.day, 0, 0, 0, 0, time.Local)
}

// Date returns the current month's date reference.
func (c *CalendarModel) Date() time.Time {
	return c.date
}

// Day returns the selected day.
func (c *CalendarModel) Day() int {
	return c.day
}

// ViewMode returns the current view mode.
func (c *CalendarModel) ViewMode() CalendarViewModeType {
	return c.viewMode
}

// SetViewMode sets the view mode.
func (c *CalendarModel) SetViewMode(mode CalendarViewModeType) {
	c.viewMode = mode
}

// renderCompact renders the compact calendar view.
func (c *CalendarModel) renderCompact() string {
	var b strings.Builder

	// Header with month/year
	monthYear := c.date.Format("January 2006")
	b.WriteString(styles.Title.Render(monthYear))
	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("← → prev/next month | h l prev/next day | v toggle view"))
	b.WriteString("\n\n")

	// Weekday headers
	weekdays := []string{"Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"}
	for _, wd := range weekdays {
		b.WriteString(styles.CalendarWeekday.Render(fmt.Sprintf(" %s ", wd)))
	}
	b.WriteString("\n")

	// Calculate first day and number of days
	firstOfMonth := time.Date(c.date.Year(), c.date.Month(), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	startWeekday := int(firstOfMonth.Weekday())
	daysInMonth := lastOfMonth.Day()
	today := time.Now()

	// Build task count by day
	tasksByDay := make(map[int]int)
	for _, t := range c.tasks {
		if t.Due == nil {
			continue
		}
		if parsed, err := time.Parse("2006-01-02", t.Due.Date); err == nil {
			if parsed.Year() == c.date.Year() && parsed.Month() == c.date.Month() {
				tasksByDay[parsed.Day()]++
			}
		}
	}

	// Render calendar grid
	day := 1
	for week := 0; week < 6; week++ {
		if day > daysInMonth {
			break
		}

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

			isToday := today.Year() == c.date.Year() &&
				today.Month() == c.date.Month() &&
				today.Day() == day
			hasTasks := tasksByDay[day] > 0
			isSelected := day == c.day && c.focused
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

			if hasTasks && !isSelected {
				dayStr = fmt.Sprintf(" %2d*", day)
			}

			b.WriteString(style.Render(dayStr))
			b.WriteString(" ")
			day++
		}
		b.WriteString("\n")
	}

	// Show selected day info
	b.WriteString("\n")
	selectedDate := time.Date(c.date.Year(), c.date.Month(), c.day, 0, 0, 0, 0, time.Local)
	b.WriteString(styles.Subtitle.Render(selectedDate.Format("Monday, January 2")))
	b.WriteString("\n\n")

	// Find tasks for selected day
	selectedDateStr := selectedDate.Format("2006-01-02")
	taskCount := 0
	for _, t := range c.tasks {
		if t.Due != nil && t.Due.Date == selectedDateStr {
			taskCount++
		}
	}

	if taskCount == 0 {
		b.WriteString(styles.HelpDesc.Render("No tasks for this day"))
	} else {
		b.WriteString(styles.HelpDesc.Render(fmt.Sprintf("%d task(s) - press Enter for details", taskCount)))
	}

	return b.String()
}

// renderExpanded renders the expanded calendar view with task names.
func (c *CalendarModel) renderExpanded() string {
	var b strings.Builder

	// Header
	monthYear := c.date.Format("January 2006")
	b.WriteString(styles.Title.Render(monthYear))
	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("← → prev/next month | h l prev/next day | v toggle view"))
	b.WriteString("\n\n")

	// Calculate cell dimensions
	availableWidth := c.width - 8
	if availableWidth < 35 {
		availableWidth = 35
	}
	cellWidth := availableWidth / 7
	if cellWidth < 5 {
		cellWidth = 5
	}
	if cellWidth > 20 {
		cellWidth = 20
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

	// Calculate dates
	firstOfMonth := time.Date(c.date.Year(), c.date.Month(), 1, 0, 0, 0, 0, time.Local)
	lastOfMonth := firstOfMonth.AddDate(0, 1, -1)
	startWeekday := int(firstOfMonth.Weekday())
	daysInMonth := lastOfMonth.Day()
	today := time.Now()

	// Build task map
	tasksByDay := make(map[int][]api.Task)
	for _, t := range c.tasks {
		if t.Due == nil {
			continue
		}
		if parsed, err := time.Parse("2006-01-02", t.Due.Date); err == nil {
			if parsed.Year() == c.date.Year() && parsed.Month() == c.date.Month() {
				tasksByDay[parsed.Day()] = append(tasksByDay[parsed.Day()], t)
			}
		}
	}

	maxTasksPerCell := 2

	// Render grid
	day := 1
	for week := 0; week < 6; week++ {
		if day > daysInMonth {
			break
		}

		// Day numbers row
		dayNumLine := "│"
		weekStart := day
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

			isToday := today.Year() == c.date.Year() &&
				today.Month() == c.date.Month() &&
				today.Day() == day
			isSelected := day == c.day && c.focused
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

			paddedDay := fmt.Sprintf("%-*s", cellWidth, dayStr)
			dayNumLine += style.Render(paddedDay) + "│"
			day++
		}
		b.WriteString(dayNumLine)
		b.WriteString("\n")

		// Task rows
		for taskLine := 0; taskLine < maxTasksPerCell; taskLine++ {
			taskRow := "│"
			tempDay := weekStart
			for weekday := 0; weekday < 7; weekday++ {
				if week == 0 && weekday < startWeekday {
					taskRow += strings.Repeat(" ", cellWidth) + "│"
					continue
				}

				if tempDay > daysInMonth {
					taskRow += strings.Repeat(" ", cellWidth) + "│"
					tempDay++
					continue
				}

				tasks := tasksByDay[tempDay]
				var cellContent string

				if taskLine < len(tasks) {
					task := tasks[taskLine]
					taskName := task.Content
					maxLen := cellWidth - 2
					if len(taskName) > maxLen && maxLen > 1 {
						taskName = taskName[:maxLen-1] + "…"
					}
					paddedTask := fmt.Sprintf(" %-*s", cellWidth-1, taskName)
					priorityStyle := styles.GetPriorityStyle(task.Priority)
					cellContent = priorityStyle.Render(paddedTask)
				} else if taskLine == maxTasksPerCell-1 && len(tasks) > maxTasksPerCell {
					hiddenCount := len(tasks) - maxTasksPerCell
					moreText := fmt.Sprintf("+%d more", hiddenCount)
					paddedMore := fmt.Sprintf(" %-*s", cellWidth-1, moreText)
					cellContent = styles.CalendarMoreTasks.Render(paddedMore)
				} else {
					cellContent = strings.Repeat(" ", cellWidth)
				}

				taskRow += cellContent + "│"
				tempDay++
			}
			b.WriteString(taskRow)
			b.WriteString("\n")
		}

		// Separator
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
