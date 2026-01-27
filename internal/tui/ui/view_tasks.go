package ui

import (
	"github.com/hy4ri/todoist-tui/internal/tui/state"
	"fmt"
	"sort"
	"strings"
	"time"

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
	case state.ViewUpcoming:
		content = r.renderUpcoming(innerWidth, innerHeight)
	case state.ViewLabels:
		content = r.renderLabelsView(innerWidth, innerHeight)
	case state.ViewCalendar:
		content = r.renderCalendar(innerHeight) // Calendar handles own sizing
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
		title = time.Now().Format("Monday 2 Jan")
	case state.ViewProject:
		if r.CurrentProject != nil {
			title = r.CurrentProject.Name
		}
	default:
		title = "Tasks"
	}
	b.WriteString(styles.Title.Render(title))
	b.WriteString("\n\n")

	if r.Loading {
		b.WriteString(r.Spinner.View())
		b.WriteString(" Loading...")
	} else if r.Err != nil {
		b.WriteString(styles.StatusBarError.Render(fmt.Sprintf("Error: %v", r.Err)))
	} else if len(r.Tasks) == 0 {
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
		} else if r.CurrentView == state.ViewProject {
			b.WriteString(r.renderProjectTasks(width, maxHeight-2))
		} else {
			b.WriteString(r.renderFlatTasks(width, maxHeight-2))
		}
	}

	return b.String()
}

// lineInfo represents a display line with optional task reference.
type lineInfo struct {
	content   string
	taskIndex int    // -1 for headers
	sectionID string // section ID if this is a section header
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
			orderedIndices = append(orderedIndices, i)
			lines = append(lines, lineInfo{content: r.renderTaskByDisplayIndex(i, orderedIndices, width), taskIndex: i})
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
					orderedIndices = append(orderedIndices, i)
					lines = append(lines, lineInfo{content: r.renderTaskByDisplayIndex(i, orderedIndices, width), taskIndex: i})
				}
			}
			// Add blank line after section for spacing
			lines = append(lines, lineInfo{content: "", taskIndex: -1})
		}
	}

	return r.renderScrollableLines(lines, orderedIndices, maxHeight)
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

	// Build lines
	var lines []lineInfo

	if len(overdue) > 0 {
		lines = append(lines, lineInfo{content: styles.SectionHeader.Render("OVERDUE"), taskIndex: -1})
		for _, i := range overdue {
			lines = append(lines, lineInfo{content: r.renderTaskByDisplayIndex(i, orderedIndices, width), taskIndex: i})
		}
	}

	if len(today) > 0 {
		if len(overdue) > 0 {
			lines = append(lines, lineInfo{content: "", taskIndex: -1})
		}
		for _, i := range today {
			lines = append(lines, lineInfo{content: r.renderTaskByDisplayIndex(i, orderedIndices, width), taskIndex: i})
		}
	}

	if len(other) > 0 {
		if len(overdue) > 0 || len(today) > 0 {
			lines = append(lines, lineInfo{content: "", taskIndex: -1})
		}
		lines = append(lines, lineInfo{content: styles.SectionHeader.Render("NO DUE DATE"), taskIndex: -1})
		for _, i := range other {
			lines = append(lines, lineInfo{content: r.renderTaskByDisplayIndex(i, orderedIndices, width), taskIndex: i})
		}
	}

	return r.renderScrollableLines(lines, orderedIndices, maxHeight)
}

// renderFlatTasks renders tasks in a flat list.
func (r *Renderer) renderFlatTasks(width, maxHeight int) string {
	var lines []lineInfo
	var orderedIndices []int

	for i := range r.Tasks {
		orderedIndices = append(orderedIndices, i)
		lines = append(lines, lineInfo{content: r.renderTaskByDisplayIndex(i, orderedIndices, width), taskIndex: i})
	}

	return r.renderScrollableLines(lines, orderedIndices, maxHeight)
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
func (r *Renderer) renderTaskByDisplayIndex(taskIndex int, orderedIndices []int, width int) string {
	t := r.Tasks[taskIndex]

	// Find display position for cursor
	displayPos := 0
	for i, idx := range orderedIndices {
		if idx == taskIndex {
			displayPos = i
			break
		}
	}

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
		rawDue := t.DueDisplay()
		dueStr = "| " + rawDue
		dueWidth = lipgloss.Width(dueStr) + 1
	}

	labelStr := ""
	labelWidth := 0
	if len(t.Labels) > 0 {
		var lStrs []string
		for _, l := range t.Labels {
			lStrs = append(lStrs, "@"+l)
		}
		labelStr = strings.Join(lStrs, " ")
		labelWidth = lipgloss.Width(labelStr) + 1
	}

	// Calculate fixed overhead (cursor + selection + indent + checkbox + spaces)
	// "> ●  [ ] " = 2 + 1 + indentLen + 4 = 7 + indentLen
	overhead := 7 + len(indent) + dueWidth + labelWidth + 2 // +2 for safety margin

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

	// Build line with selection mark
	line := fmt.Sprintf("%s%s%s%s %s %s %s", cursor, selectionMark, indent, checkbox, styledContent, styledDue, styledLabels)

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

// renderScrollableLines renders lines with scrolling support using viewport.
func (r *Renderer) renderScrollableLines(lines []lineInfo, orderedIndices []int, maxHeight int) string {
	// Store ordered indices for use in handleSelect
	r.TaskOrderedIndices = orderedIndices

	if len(lines) == 0 {
		r.ScrollOffset = 0
		r.State.ViewportLines = nil
		r.State.ViewportSections = nil
		return ""
	}

	// Build content string and track line->task mapping and section mapping
	var content strings.Builder
	r.State.ViewportLines = make([]int, 0, len(lines))
	r.State.ViewportSections = make([]string, 0, len(lines))

	for i, line := range lines {
		content.WriteString(line.content)
		if i < len(lines)-1 {
			content.WriteString("\n")
		}
		// Map this viewport line to its task index (-1 for headers, -2 for section headers)
		r.State.ViewportLines = append(r.State.ViewportLines, line.taskIndex)
		// Map this viewport line to its section ID (empty string for non-section lines)
		r.State.ViewportSections = append(r.State.ViewportSections, line.sectionID)
	}

	// Find which line the cursor is on
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

	// If viewport is ready, use it for scrolling
	if r.State.ViewportReady {
		// Update viewport height if needed (maxHeight is the available height)
		if r.TaskViewport.Height != maxHeight && maxHeight > 0 {
			r.TaskViewport.Height = maxHeight
		}

		// Set content to viewport
		r.TaskViewport.SetContent(content.String())

		// Sync viewport to show cursor
		r.syncViewportToCursor(cursorLine)

		// Store scroll offset for click handling
		r.ScrollOffset = r.TaskViewport.YOffset

		return r.TaskViewport.View()
	}

	// Fallback: viewport not ready, just return raw content (truncated)
	r.ScrollOffset = 0
	return content.String()
}

// renderUpcoming renders the upcoming view with tasks grouped by date.
func (r *Renderer) renderUpcoming(width, maxHeight int) string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Upcoming"))
	b.WriteString("\n")

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
		date := t.Due.Date
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

		for _, i := range tasksByDate[date] {
			lines = append(lines, lineInfo{
				content:   r.renderTaskByDisplayIndex(i, orderedIndices, width),
				taskIndex: i,
			})
		}
	}

	// Use common scrollable rendering - maxHeight already accounts for borders
	// Subtract 2 for Title line + newline
	result := r.renderScrollableLines(lines, orderedIndices, maxHeight-2)
	b.WriteString(result)

	return b.String()
}

// renderLabelsView renders the labels view.
func (r *Renderer) renderLabelsView(width, maxHeight int) string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Labels"))
	b.WriteString("\n\n")

	// Account for title + blank line (2 lines used)
	contentHeight := maxHeight - 2

	if r.CurrentLabel != nil {
		// Show tasks for selected label
		labelTitle := "@" + r.CurrentLabel.Name
		if r.CurrentLabel.Color != "" {
			labelTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(r.CurrentLabel.Color)).Render(labelTitle)
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
				lines = append(lines, lineInfo{
					content:   r.renderTaskByDisplayIndex(i, orderedIndices, width),
					taskIndex: i,
				})
			}
			b.WriteString(r.renderScrollableLines(lines, orderedIndices, taskHeight))
		}

		b.WriteString("\n")
		b.WriteString(styles.HelpDesc.Render("Press ESC to go back to labels list"))
	} else {
		// Extract unique labels from all tasks if personal labels are empty
		labelsToShow := r.Labels
		if len(labelsToShow) == 0 {
			labelsToShow = r.extractLabelsFromTasks()
		}

		// Build task count map for labels
		taskCountMap := r.getLabelTaskCounts()

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
					name = lipgloss.NewStyle().Foreground(lipgloss.Color(label.Color)).Render(name)
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

// syncViewportToCursor ensures the cursor is visible in the viewport.
func (r *Renderer) syncViewportToCursor() {
	height := r.Height - 4 // Approximate content height
	if height < 1 {
		height = 1
	}

	if r.TaskCursor < r.ScrollOffset {
		r.ScrollOffset = r.TaskCursor
	} else if r.TaskCursor >= r.ScrollOffset+height {
		r.ScrollOffset = r.TaskCursor - height + 1
	}
}
