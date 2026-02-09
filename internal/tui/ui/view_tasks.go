package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/hy4ri/todoist-tui/internal/tui/state"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// renderTaskList renders the task list for Today/Upcoming/Labels views.
func (r *Renderer) renderTaskList(width, height int) string {
	// Calculate viewport height (subtract title and padding)
	innerHeight := height - 2
	if innerHeight < 5 {
		innerHeight = 5
	}

	// Calculate inner width for the content
	innerWidth := width - styles.MainContent.GetHorizontalFrameSize()

	var content string
	switch r.CurrentView {
	case state.ViewInbox:
		// Inbox behaves like a project view but we'll ensure title is set correctly
		content = r.renderDefaultTaskList(innerWidth, innerHeight)
	case state.ViewUpcoming:
		content = r.renderUpcoming(innerWidth, innerHeight)
	case state.ViewLabels:
		content = r.renderLabelsView(innerWidth, innerHeight)
	case state.ViewCalendar:
		content = r.renderCalendar(innerHeight) // Calendar handles own sizing
	case state.ViewCompleted:
		content = r.renderCompletedTaskList(innerWidth, innerHeight)
	default:
		content = r.renderDefaultTaskList(innerWidth, innerHeight)
	}

	// Apply container style with fixed height
	containerStyle := styles.MainContent
	if r.FocusedPane == state.PaneMain {
		containerStyle = styles.MainContentFocused
	}

	return containerStyle.Width(width).Height(innerHeight).Render(content)
}

// renderDefaultTaskList renders the default task list for Today/Project views.
func (r *Renderer) renderDefaultTaskList(width, maxHeight int) string {
	var b strings.Builder

	// Title
	var title string
	switch r.CurrentView {
	case state.ViewToday:
		loc := time.Local
		// Try to find a timezone hint from the tasks
		for _, t := range r.Tasks {
			if t.Due != nil && t.Due.Timezone != nil && *t.Due.Timezone != "" {
				if l, err := time.LoadLocation(*t.Due.Timezone); err == nil {
					loc = l
					break
				}
			}
		}
		title = time.Now().In(loc).Format("Monday 2 Jan 3:04pm")
	case state.ViewInbox:
		title = "Inbox"
	case state.ViewProject:
		if r.CurrentProject != nil {
			title = r.CurrentProject.Name
		}
	case state.ViewCompleted:
		title = "Completed Tasks"
	default:
		title = "Tasks"
	}
	// Truncate title to prevent wrapping which breaks layout
	// width is the available inner width. reduce by 1 to be safe against border conditions.
	safeTitleWidth := width - 1
	if safeTitleWidth < 5 {
		safeTitleWidth = 5
	}
	if len(title) > safeTitleWidth {
		title = truncateString(title, safeTitleWidth)
	}

	b.WriteString(styles.Title.Copy().Underline(true).Render(strings.ToUpper(title)) + "\n\n")

	if r.Loading {
		b.WriteString(r.Spinner.View())
		b.WriteString(" Loading...")
	} else if r.Err != nil {
		b.WriteString(styles.StatusBarError.Render(fmt.Sprintf("Error: %v", r.Err)))
	} else if len(r.Tasks) == 0 && (len(r.Sections) == 0 || (r.CurrentView != state.ViewProject && r.CurrentView != state.ViewInbox)) {
		msg := "No tasks found"
		if r.CurrentView == state.ViewToday {
			msg = "All done for today! \n" + styles.HelpDesc.Render("Enjoy your day off 🏝️")
		} else {
			msg = "No tasks here.\n" + styles.HelpDesc.Render("Press 'a' to add one.")
		}
		b.WriteString(msg)
	} else {
		// Group tasks by due status for Today view
		// Title uses 2 lines (title + newline)
		if r.CurrentView == state.ViewToday {
			b.WriteString(r.renderGroupedTasks(width, maxHeight-2))
		} else if r.CurrentView == state.ViewProject || r.CurrentView == state.ViewInbox {
			b.WriteString(r.renderProjectTasks(width, maxHeight-2))
		} else {
			b.WriteString(r.renderFlatTasks(width, maxHeight-2))
		}
	}

	return b.String()
}

// lineInfo represents a display line with optional task reference.
type lineInfo struct {
	content    string
	renderFunc func(width int) string // Lazy renderer
	taskIndex  int                    // -1 for headers
	sectionID  string                 // section ID if this is a section header
}

// renderProjectTasks renders tasks grouped by section for a project.
func (r *Renderer) renderProjectTasks(width, maxHeight int) string {
	// Build ordered list of task indices matching display order
	var orderedIndices []int

	// Group tasks by section
	tasksBySection := make(map[string][]int)
	var noSectionTasks []int

	for i, t := range r.Tasks {
		if t.SectionID != nil && *t.SectionID != "" {
			tasksBySection[*t.SectionID] = append(tasksBySection[*t.SectionID], i)
		} else {
			noSectionTasks = append(noSectionTasks, i)
		}
	}

	var lines []lineInfo

	// 1. First, tasks without sections
	if len(noSectionTasks) > 0 {
		for _, i := range noSectionTasks {
			i, displayPos := i, len(orderedIndices)
			orderedIndices = append(orderedIndices, i)
			lines = append(lines, lineInfo{
				renderFunc: func(w int) string { return r.renderTaskByDisplayIndex(i, displayPos, w) },
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
		// Add spacer if there are sections following
		if len(r.Sections) > 0 {
			lines = append(lines, lineInfo{content: "", taskIndex: -1})
		}
	}

	// 2. Then, tasks by section (in order)
	if len(r.Sections) > 0 {
		for _, section := range r.Sections {
			taskIndices := tasksBySection[section.ID]

			// Create section header index (unique negative value)
			headerIndex := -100 - len(orderedIndices)
			orderedIndices = append(orderedIndices, headerIndex)

			if len(taskIndices) == 0 {
				lines = append(lines, lineInfo{
					content:   r.renderSectionHeaderByIndex(section.Name, headerIndex, orderedIndices),
					taskIndex: headerIndex,
					sectionID: section.ID,
				})
			} else {
				lines = append(lines, lineInfo{
					content:   r.renderSectionHeaderByIndex(section.Name, headerIndex, orderedIndices),
					taskIndex: headerIndex,
					sectionID: section.ID,
				})
				for _, i := range taskIndices {
					i, displayPos := i, len(orderedIndices)
					orderedIndices = append(orderedIndices, i)
					lines = append(lines, lineInfo{
						renderFunc: func(w int) string { return r.renderTaskByDisplayIndex(i, displayPos, w) },
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
			// Add blank line after section for spacing
			lines = append(lines, lineInfo{content: "", taskIndex: -1})
		}
	}

	return r.renderScrollableLines(lines, orderedIndices, maxHeight, width)
}

// renderGroupedTasks renders tasks grouped by due status.
func (r *Renderer) renderGroupedTasks(width, maxHeight int) string {
	var overdue, today, other []int

	// Group tasks
	for i, t := range r.Tasks {
		if t.IsOverdue() {
			overdue = append(overdue, i)
		} else if t.IsDueToday() {
			today = append(today, i)
		} else {
			other = append(other, i)
		}
	}

	// Build ordered indices
	var orderedIndices []int
	orderedIndices = append(orderedIndices, overdue...)
	orderedIndices = append(orderedIndices, today...)
	orderedIndices = append(orderedIndices, other...)

	// Track display position for orderedIndices
	currentDisplayPos := 0

	// Build lines
	var lines []lineInfo

	if len(overdue) > 0 {
		lines = append(lines, lineInfo{content: styles.SectionHeader.Render("OVERDUE"), taskIndex: -1})
		for _, i := range overdue {
			i, pos := i, currentDisplayPos
			currentDisplayPos++
			lines = append(lines, lineInfo{
				renderFunc: func(w int) string { return r.renderTaskByDisplayIndex(i, pos, w) },
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

	if len(today) > 0 {
		if len(overdue) > 0 {
			lines = append(lines, lineInfo{content: "", taskIndex: -1})
		}
		for _, i := range today {
			i, pos := i, currentDisplayPos
			currentDisplayPos++
			lines = append(lines, lineInfo{
				renderFunc: func(w int) string { return r.renderTaskByDisplayIndex(i, pos, w) },
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

	if len(other) > 0 {
		if len(overdue) > 0 || len(today) > 0 {
			lines = append(lines, lineInfo{content: "", taskIndex: -1})
		}
		lines = append(lines, lineInfo{content: styles.SectionHeader.Render("NO DUE DATE"), taskIndex: -1})
		for _, i := range other {
			i, pos := i, currentDisplayPos
			currentDisplayPos++
			lines = append(lines, lineInfo{
				renderFunc: func(w int) string { return r.renderTaskByDisplayIndex(i, pos, w) },
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

	return r.renderScrollableLines(lines, orderedIndices, maxHeight, width)
}

// renderFlatTasks renders tasks in a flat list.
func (r *Renderer) renderFlatTasks(width, maxHeight int) string {
	var lines []lineInfo
	var orderedIndices []int

	for i := range r.Tasks {
		i, displayPos := i, len(orderedIndices)
		orderedIndices = append(orderedIndices, i)
		lines = append(lines, lineInfo{
			renderFunc: func(w int) string { return r.renderTaskByDisplayIndex(i, displayPos, w) },
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

	return r.renderScrollableLines(lines, orderedIndices, maxHeight, width)
}

// renderSectionHeaderByIndex renders a section header with cursor highlighting for empty sections.
func (r *Renderer) renderSectionHeaderByIndex(sectionName string, headerIndex int, orderedIndices []int) string {
	// Find display position for cursor
	displayPos := 0
	for i, idx := range orderedIndices {
		if idx == headerIndex {
			displayPos = i
			break
		}
	}

	// Check if cursor is on this empty section header
	isCursorHere := displayPos == r.TaskCursor && r.FocusedPane == state.PaneMain

	// Cursor indicator
	cursor := "  "
	if isCursorHere {
		cursor = "> "
	}

	// Truncate section name to prevent wrapping
	// Available width = App Width - Sidebar (if open) - Padding?
	// We don't have exact pane width here easily unless we pass it.
	// But usually headers aren't super long. Let's truncate to 40 chars or find a way to access width.
	// Actually, renderProjectTasks DOES pass 'width' to renderTaskByDisplayIndex but NOT to this function.
	// We should update the signature if we want perfect strictness.
	// For now, let's just be conservative.
	maxHeaderLen := 50
	if len(sectionName) > maxHeaderLen {
		sectionName = sectionName[:maxHeaderLen-1] + "…"
	}

	// Build the header line
	line := fmt.Sprintf("%s%s", cursor, sectionName)

	// Apply style - use TaskSelected style when cursor is here, otherwise SectionHeader
	style := styles.SectionHeader
	if isCursorHere {
		style = styles.TaskSelected
	}

	return style.Render(line)
}

// renderTaskByDisplayIndex renders a task with cursor based on display order.
func (r *Renderer) renderTaskByDisplayIndex(taskIndex int, displayPos int, width int) string {
	t := r.Tasks[taskIndex]

	// Cursor
	cursor := "  "
	if displayPos == r.TaskCursor && r.FocusedPane == state.PaneMain {
		cursor = "> "
	}

	// Selection indicator
	selectionMark := " "
	if r.SelectedTaskIDs[t.ID] {
		selectionMark = "●"
	}

	// Checkbox
	checkbox := styles.CheckboxUnchecked
	if t.Checked {
		checkbox = styles.CheckboxChecked
	}

	// Indent for subtasks
	indent := ""
	if t.ParentID != nil {
		indent = "  "
	}

	// Calculate metadata widths
	dueStr := ""
	dueWidth := 0
	if t.Due != nil {
		dueStr = "| " + t.DueDisplay()
		dueWidth = lipgloss.Width(dueStr) + 1
	}

	labelStr := ""
	labelWidth := 0
	if len(t.Labels) > 0 {
		var lStrs []string
		for _, l := range t.Labels {
			lStr := "@" + l
			// Lookup color
			if color := r.getLabelColor(l); color != "" {
				lStr = lipgloss.NewStyle().Foreground(styles.GetColor(color)).Render(lStr)
			}
			lStrs = append(lStrs, lStr)
		}
		labelStr = strings.Join(lStrs, " ")
		// We calculate width based on unstyled string to avoid issues,
		// but Lipgloss styles add ANSI codes which ruin Width() if not careful.
		// Actually lipgloss.Width() strips ANSI codes, so it should be fine.
		labelWidth = lipgloss.Width(labelStr) + 1
	}

	// Calculate fixed overhead (cursor + selection + indent + checkbox + spaces + recurring icon)
	// "> ●  [ ] " = 2 + 1 + indentLen + 4 = 7 + indentLen
	// recurring adds 1 char if present, plus potential spacing artifacts.
	// We bump safety margin from 2 to 6 to be absolutely safe against wrapping.
	overhead := 7 + len(indent) + dueWidth + labelWidth + 6

	// Truncate content if needed
	content := t.Content
	maxContentWidth := width - overhead
	if maxContentWidth < 5 {
		maxContentWidth = 5
	}
	content = truncateString(content, maxContentWidth)

	// Apply priority style - use MaxWidth so Lipgloss handles color continuation if it ever wraps
	priorityStyle := styles.GetPriorityStyle(t.Priority).MaxWidth(maxContentWidth)
	styledContent := priorityStyle.Render(content)

	// Style metadata
	styledDue := ""
	if dueStr != "" {
		if t.IsOverdue() {
			styledDue = styles.TaskDueOverdue.Render(dueStr)
		} else if t.IsDueToday() {
			styledDue = styles.TaskDueToday.Render(dueStr)
		} else {
			styledDue = styles.TaskDue.Render(dueStr)
		}
	}

	styledLabels := ""
	if labelStr != "" {
		styledLabels = styles.TaskLabel.Render(labelStr)
	}

	styledRecurring := ""
	if t.Due != nil && t.Due.IsRecurring {
		styledRecurring = styles.TaskRecurring.Render("↻")
	}

	// Build line with selection mark
	line := fmt.Sprintf("%s%s%s%s %s%s %s %s", cursor, selectionMark, indent, checkbox, styledContent, styledRecurring, styledDue, styledLabels)

	// Apply base style
	style := styles.TaskItem
	if displayPos == r.TaskCursor && r.FocusedPane == state.PaneMain {
		style = styles.TaskSelected
	}
	if t.Checked {
		style = styles.TaskCompleted
	}

	// Force exactly one line and width, no wrapping
	return style.MaxWidth(width - 2).Render(line)
}

// renderTaskDescription renders a task description for the list view.
func (r *Renderer) renderTaskDescription(desc string, width int) string {
	if desc == "" {
		return ""
	}

	// Indent matches styles.TaskListDescription (10 spaces)
	// We leave extra buffer for right border/padding
	padding := 10
	safeBuffer := 4 // Increased buffer
	availableWidth := width - padding - safeBuffer

	if availableWidth < 5 {
		return "" // Too narrow to show description safely
	}

	// Descriptions can have multiple lines, take just the first one for the list view
	firstLine := strings.Split(desc, "\n")[0]

	// Strip markdown links [text](url) -> text
	if start := strings.Index(firstLine, "["); start >= 0 {
		if end := strings.Index(firstLine[start:], "]"); end >= 0 {
			end += start
			if len(firstLine) > end+1 && firstLine[end+1] == '(' {
				if closeParen := strings.Index(firstLine[end+1:], ")"); closeParen >= 0 {
					closeParen += end + 1
					linkText := firstLine[start+1 : end]
					firstLine = firstLine[:start] + linkText + firstLine[closeParen+1:]
				}
			}
		}
	}

	// Strictly truncate using the runewidth-aware helper
	truncated := truncateString(firstLine, availableWidth)

	// Render with explicit MaxWidth to be safe, though usage of truncated string should suffice.
	// We calculate explicit width for style to avoid Lipgloss padding adding to overflow.
	// Style has PaddingLeft(10).
	return styles.TaskListDescription.Copy().Width(availableWidth).MaxWidth(availableWidth).Render(truncated)
}

// renderScrollableLines renders lines with scrolling support using viewport and windowing.
func (r *Renderer) renderScrollableLines(lines []lineInfo, orderedIndices []int, maxHeight int, width int) string {
	// Store ordered indices for use in handleSelect
	r.TaskOrderedIndices = orderedIndices

	if len(lines) == 0 {
		r.ScrollOffset = 0
		r.State.ViewportLines = nil
		r.State.ViewportSections = nil
		return ""
	}

	// Map lines to tasks/sections for click handling (always map all lines)
	r.State.ViewportLines = make([]int, 0, len(lines))
	r.State.ViewportSections = make([]string, 0, len(lines))
	for _, line := range lines {
		r.State.ViewportLines = append(r.State.ViewportLines, line.taskIndex)
		r.State.ViewportSections = append(r.State.ViewportSections, line.sectionID)
	}

	// Update Viewport Width immediately to ensure correct line wrapping/truncation calculations
	// if inner logic relies on it (though we pass width to renderFunc, the viewport needs to know too)
	if width > 0 {
		r.TaskViewport.Width = width
	}

	// Ensure viewport is initialized
	if !r.State.ViewportReady {
		// Just render everything if not ready (fallback)
		var content strings.Builder
		width := r.Width - 4 // Approximate if viewport not ready
		for i, line := range lines {
			if line.renderFunc != nil {
				content.WriteString(line.renderFunc(width))
			} else {
				content.WriteString(line.content)
			}
			if i < len(lines)-1 {
				content.WriteString("\n")
			}
		}
		return content.String()
	}

	// Update viewport height
	if r.TaskViewport.Height != maxHeight && maxHeight > 0 {
		r.TaskViewport.Height = maxHeight
	}

	// Determine visible window
	// 1. Find cursor line
	cursorLine := 0
	if r.TaskCursor >= 0 && r.TaskCursor < len(orderedIndices) {
		targetTaskIndex := orderedIndices[r.TaskCursor]
		for i, line := range lines {
			if line.taskIndex == targetTaskIndex {
				cursorLine = i
				break
			}
		}
	}

	// 2. Predict target YOffset to keep cursor in view
	yOffset := r.TaskViewport.YOffset
	height := r.TaskViewport.Height
	if height <= 0 {
		height = 10 // Fallback
	}

	if cursorLine < yOffset {
		yOffset = cursorLine
	} else if cursorLine >= yOffset+height {
		yOffset = cursorLine - height + 1
	}

	// Clamp YOffset
	maxOffset := len(lines) - height
	if maxOffset < 0 {
		maxOffset = 0
	}
	if yOffset > maxOffset {
		yOffset = maxOffset
	}
	if yOffset < 0 {
		yOffset = 0
	}

	// 3. Define rendering window (add buffer)
	buffer := 5
	renderStart := yOffset - buffer
	renderEnd := yOffset + height + buffer

	// width passed as argument

	// Build content with windowing
	var content strings.Builder
	for i, line := range lines {
		// Always render explicitly provided content (headers/spacers), lazy render tasks
		isVisible := i >= renderStart && i <= renderEnd

		if line.renderFunc != nil {
			if isVisible {
				content.WriteString(line.renderFunc(width))
			} else {
				// Placeholder for invisible task line to maintain scroll height
				// Note: renderTaskByDisplayIndex produces 1 line.
				// If multiline descriptions were lazy, we'd need to know their height.
				// Assume 1 line for now.
				content.WriteString("")
			}
		} else {
			content.WriteString(line.content)
		}

		if i < len(lines)-1 {
			content.WriteString("\n")
		}
	}

	// Update Viewport
	r.TaskViewport.SetContent(content.String())
	r.TaskViewport.SetYOffset(yOffset)
	r.ScrollOffset = yOffset

	return r.TaskViewport.View()
}

// renderUpcoming renders the upcoming view with tasks grouped by date.
func (r *Renderer) renderUpcoming(width, maxHeight int) string {
	var b strings.Builder

	b.WriteString(styles.Title.Copy().Underline(true).Render("UPCOMING TASKS") + "\n\n")

	if r.Loading {
		b.WriteString(r.Spinner.View())
		b.WriteString(" Loading...")
		return b.String()
	}

	if len(r.Tasks) == 0 {
		b.WriteString("\n")
		b.WriteString(styles.HelpDesc.Render("No upcoming tasks"))
		return b.String()
	}

	// Group tasks by date
	tasksByDate := make(map[string][]int)
	var dates []string

	for i, t := range r.Tasks {
		if t.Due == nil {
			continue
		}
		// Use only the date portion (YYYY-MM-DD) for grouping to avoid separate headers for times
		date := t.Due.Date
		if len(date) > 10 {
			date = date[:10]
		}

		if _, exists := tasksByDate[date]; !exists {
			dates = append(dates, date)
		}
		tasksByDate[date] = append(tasksByDate[date], i)
	}

	// Sort dates
	sort.Strings(dates)

	// Build ordered indices for cursor mapping
	var orderedIndices []int
	for _, date := range dates {
		orderedIndices = append(orderedIndices, tasksByDate[date]...)
	}

	// Build lines
	var lines []lineInfo

	for idx, date := range dates {
		// Parse date for display
		displayDate := date
		if parsed, err := time.Parse("2006-01-02", date); err == nil {
			today := time.Now().Format("2006-01-02")
			tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
			switch date {
			case today:
				displayDate = "Today"
			case tomorrow:
				displayDate = "Tomorrow"
			default:
				displayDate = parsed.Format("Mon, Jan 2")
			}
		}

		// Add blank line before header (except first) for spacing
		if idx > 0 {
			lines = append(lines, lineInfo{content: "", taskIndex: -1})
		}

		lines = append(lines, lineInfo{
			content:   styles.DateGroupHeader.Render(displayDate),
			taskIndex: -1,
		})

		// Track display position for orderedIndices
		currentDisplayPos := 0
		for _, date := range dates {
			if idx > 0 && date == dates[idx-1] {
				// Continuation... logic here is iterating dates in outer, so we need to sync currentDisplayPos
				// But we are inside outer loop over 'idx, date'.
				// So we need to advance currentDisplayPos through previous dates?
				// Better: Just loop 'orderedIndices' logic again inside? No.
				// We can just calculate currentDisplayPos by summing lengths of tasksByDate[prevDate].
			}
		}
		// Reset currentDisplayPos logic:
		currentDisplayPos = 0
		for k := 0; k < idx; k++ {
			currentDisplayPos += len(tasksByDate[dates[k]])
		}

		for _, i := range tasksByDate[date] {
			i, pos := i, currentDisplayPos
			currentDisplayPos++
			lines = append(lines, lineInfo{
				renderFunc: func(w int) string { return r.renderTaskByDisplayIndex(i, pos, w) },
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

	// Use common scrollable rendering - maxHeight already accounts for borders
	// Subtract 2 for Title line + newline
	result := r.renderScrollableLines(lines, orderedIndices, maxHeight-2, width)
	b.WriteString(result)

	return b.String()
}

// renderLabelsView renders the labels view.
func (r *Renderer) renderLabelsView(width, maxHeight int) string {
	var b strings.Builder

	b.WriteString(styles.Title.Copy().Underline(true).Render("LABELS") + "\n\n")

	// Account for title + blank line (2 lines used)
	contentHeight := maxHeight - 2

	if r.CurrentLabel != nil {
		// Show tasks for selected label
		labelTitle := "@" + r.CurrentLabel.Name
		if r.CurrentLabel.Color != "" {
			labelTitle = lipgloss.NewStyle().Foreground(styles.GetColor(r.CurrentLabel.Color)).Render(labelTitle)
		}
		b.WriteString(styles.Subtitle.Render(labelTitle))
		b.WriteString("\n\n")

		// Account for subtitle + blank line + footer (4 more lines)
		taskHeight := contentHeight - 4

		if len(r.Tasks) == 0 {
			b.WriteString(styles.HelpDesc.Render("No tasks with this label"))
		} else {
			// Build lines and ordered indices for scrolling
			var lines []lineInfo
			var orderedIndices []int
			for i := range r.Tasks {
				orderedIndices = append(orderedIndices, i)
			}
			for i := range r.Tasks {
				i, pos := i, i // orderedIndices[i] == i
				lines = append(lines, lineInfo{
					renderFunc: func(w int) string { return r.renderTaskByDisplayIndex(i, pos, w) },
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
			b.WriteString(r.renderScrollableLines(lines, orderedIndices, taskHeight, width))
		}

		b.WriteString("\n")
		b.WriteString(styles.HelpDesc.Render("Press ESC to go back to labels list"))
	} else {

		// Invalidate cache if data version changed
		if r.State.DataVersion != r.lastDataVersion || r.cachedTaskCountMap == nil {
			r.cachedTaskCountMap = r.getLabelTaskCounts()
			r.lastDataVersion = r.State.DataVersion
			// Invalidate extracted labels too
			r.cachedExtractedLabels = nil
		}

		// Extract unique labels from all tasks if personal labels are empty
		labelsToShow := r.Labels
		if len(labelsToShow) == 0 {
			if r.cachedExtractedLabels == nil {
				r.cachedExtractedLabels = r.extractLabelsFromTasks()
			}
			labelsToShow = r.cachedExtractedLabels
		}

		// Build task count map for labels
		taskCountMap := r.cachedTaskCountMap

		// Account for footer (2 lines)
		labelHeight := contentHeight - 2

		// Show list of labels
		if len(labelsToShow) == 0 {
			b.WriteString(styles.HelpDesc.Render("No labels found"))
		} else {
			// Calculate scroll window for labels
			startIdx := 0
			if r.TaskCursor >= labelHeight {
				startIdx = r.TaskCursor - labelHeight + 1
			}
			endIdx := startIdx + labelHeight
			if endIdx > len(labelsToShow) {
				endIdx = len(labelsToShow)
			}

			for i := startIdx; i < endIdx; i++ {
				label := labelsToShow[i]
				cursor := "  "
				style := styles.LabelItem
				if i == r.TaskCursor && r.FocusedPane == state.PaneMain {
					cursor = "> "
					style = styles.LabelSelected
				}

				// Label name with optional color
				name := "@" + label.Name
				if label.Color != "" {
					name = lipgloss.NewStyle().Foreground(styles.GetColor(label.Color)).Render(name)
				}

				// Task count badge
				taskCount := taskCountMap[label.Name]
				countBadge := ""
				if taskCount > 0 {
					countBadge = styles.HelpDesc.Render(fmt.Sprintf(" (%d)", taskCount))
				}

				line := fmt.Sprintf("%s%s%s", cursor, name, countBadge)
				b.WriteString(style.Render(line))
				b.WriteString("\n")
			}
		}

		b.WriteString("\n")
		b.WriteString(styles.HelpDesc.Render("Press Enter to view tasks with label"))
	}

	return b.String()
}

// getLabelTaskCounts returns a map of label name to task count.
func (r *Renderer) getLabelTaskCounts() map[string]int {
	counts := make(map[string]int)

	// Use allTasks if available, otherwise fall back to tasks
	tasksToScan := r.AllTasks
	if len(tasksToScan) == 0 {
		tasksToScan = r.Tasks
	}

	for _, t := range tasksToScan {
		for _, labelName := range t.Labels {
			counts[labelName]++
		}
	}

	return counts
}

// extractLabelsFromTasks extracts unique labels from all tasks.
func (r *Renderer) extractLabelsFromTasks() []api.Label {
	labelSet := make(map[string]bool)
	var labels []api.Label

	// Check allTasks first, fall back to tasks
	tasksToScan := r.AllTasks
	if len(tasksToScan) == 0 {
		tasksToScan = r.Tasks
	}

	for _, t := range tasksToScan {
		for _, labelName := range t.Labels {
			if !labelSet[labelName] {
				labelSet[labelName] = true
				labels = append(labels, api.Label{
					Name: labelName,
				})
			}
		}
	}

	// Sort labels alphabetically
	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Name < labels[j].Name
	})

	return labels
}

// syncViewportToCursor ensures the cursor line is visible in the viewport.
func (r *Renderer) syncViewportToCursor(cursorLine int, height int) {
	if !r.ViewportReady {
		return
	}

	visibleStart := r.TaskViewport.YOffset
	visibleEnd := visibleStart + height

	if cursorLine < visibleStart {
		// Cursor above viewport - scroll up
		r.TaskViewport.SetYOffset(cursorLine)
	} else if cursorLine >= visibleEnd {
		// Cursor below viewport - scroll down to show cursor at bottom
		r.TaskViewport.SetYOffset(cursorLine - height + 1)
	}
}

// getLabelColor returns the color name for a given label name.
func (r *Renderer) getLabelColor(name string) string {
	for _, l := range r.Labels {
		if l.Name == name {
			return l.Color
		}
	}
	// Fallback to extractLabelsFromTasks if r.Labels is empty?
	// Or maybe tasks have inconsistent label metadata.
	return ""
}
