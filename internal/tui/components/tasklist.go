package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// RenderMode determines how tasks are rendered.
type RenderMode int

const (
	RenderModeFlat    RenderMode = iota // Simple flat list
	RenderModeGrouped                   // Grouped by overdue/today/other
	RenderModeProject                   // Grouped by section
)

// TaskListModel manages a scrollable list of tasks.
type TaskListModel struct {
	tasks          []api.Task
	sections       []api.Section
	cursor         int
	orderedIndices []int // Maps display position to tasks[] index
	viewportLines  []int // Maps viewport line to task index (-1 for headers)
	scrollOffset   int
	width, height  int
	focused        bool
	viewportReady  bool
	viewport       viewport.Model
	renderMode     RenderMode
	title          string
	emptyMessage   string
	loading        bool
	err            error
}

// NewTaskList creates a new TaskListModel.
func NewTaskList() *TaskListModel {
	return &TaskListModel{
		tasks:         []api.Task{},
		sections:      []api.Section{},
		cursor:        0,
		focused:       false,
		viewportReady: false,
		renderMode:    RenderModeFlat,
		title:         "Tasks",
		emptyMessage:  "No tasks found",
	}
}

// Init implements Component.
func (t *TaskListModel) Init() tea.Cmd {
	return nil
}

// Update implements Component.
func (t *TaskListModel) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return t.handleKeyMsg(msg)
	}
	return t, nil
}

// handleKeyMsg processes keyboard input.
func (t *TaskListModel) handleKeyMsg(msg tea.KeyMsg) (Component, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		t.MoveCursor(1)
	case "k", "up":
		t.MoveCursor(-1)
	case "g":
		t.cursor = 0
	case "G":
		t.moveCursorToEnd()
	case "ctrl+d":
		t.MoveCursor(10)
	case "ctrl+u":
		t.MoveCursor(-10)
	case "enter":
		if t.cursor >= 0 && t.cursor < len(t.orderedIndices) {
			taskIndex := t.orderedIndices[t.cursor]
			if taskIndex >= 0 && taskIndex < len(t.tasks) {
				task := t.tasks[taskIndex]
				return t, func() tea.Msg {
					return TaskSelectedMsg{Task: &task}
				}
			}
		}
	}
	return t, nil
}

// View implements Component.
func (t *TaskListModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render(t.title))
	b.WriteString("\n")

	if t.loading {
		b.WriteString("Loading...")
		return b.String()
	}

	if t.err != nil {
		b.WriteString(styles.StatusBarError.Render(fmt.Sprintf("Error: %v", t.err)))
		return b.String()
	}

	if len(t.tasks) == 0 {
		b.WriteString(t.emptyMessage)
		return b.String()
	}

	// Render based on mode
	contentHeight := t.height - 2 // Title takes 2 lines
	var content string
	switch t.renderMode {
	case RenderModeGrouped:
		content = t.renderGroupedTasks(contentHeight)
	case RenderModeProject:
		content = t.renderProjectTasks(contentHeight)
	default:
		content = t.renderFlatTasks(contentHeight)
	}
	b.WriteString(content)

	return b.String()
}

// SetSize implements Component.
func (t *TaskListModel) SetSize(width, height int) {
	t.width = width
	t.height = height

	if !t.viewportReady {
		t.viewport = viewport.New(width, height-2)
		t.viewport.Style = lipgloss.NewStyle()
		t.viewport.MouseWheelEnabled = true
		t.viewportReady = true
	} else {
		t.viewport.Width = width
		t.viewport.Height = height - 2
	}
}

// Focus sets focus on the task list.
func (t *TaskListModel) Focus() {
	t.focused = true
}

// Blur removes focus.
func (t *TaskListModel) Blur() {
	t.focused = false
}

// Focused returns focus state.
func (t *TaskListModel) Focused() bool {
	return t.focused
}

// SetTasks updates the task list.
func (t *TaskListModel) SetTasks(tasks []api.Task) {
	t.tasks = tasks
	t.loading = false
	t.err = nil
	// Reset cursor if out of bounds
	if t.cursor >= len(tasks) {
		t.cursor = 0
	}
}

// SetSections updates sections for project view.
func (t *TaskListModel) SetSections(sections []api.Section) {
	t.sections = sections
}

// SetRenderMode sets how tasks are rendered.
func (t *TaskListModel) SetRenderMode(mode RenderMode) {
	t.renderMode = mode
}

// SetTitle sets the title header.
func (t *TaskListModel) SetTitle(title string) {
	t.title = title
}

// SetEmptyMessage sets the message shown when no tasks exist.
func (t *TaskListModel) SetEmptyMessage(msg string) {
	t.emptyMessage = msg
}

// SetLoading sets loading state.
func (t *TaskListModel) SetLoading(loading bool) {
	t.loading = loading
}

// SetError sets error state.
func (t *TaskListModel) SetError(err error) {
	t.err = err
}

// Cursor returns the current cursor position.
func (t *TaskListModel) Cursor() int {
	return t.cursor
}

// SetCursor sets the cursor position.
func (t *TaskListModel) SetCursor(pos int) {
	if pos >= 0 && pos < len(t.tasks) {
		t.cursor = pos
	}
}

// OrderedIndices returns the task indices in display order.
func (t *TaskListModel) OrderedIndices() []int {
	return t.orderedIndices
}

// ViewportLines returns the viewport line to task index mapping.
func (t *TaskListModel) ViewportLines() []int {
	return t.viewportLines
}

// ScrollOffset returns the current scroll offset.
func (t *TaskListModel) ScrollOffset() int {
	return t.scrollOffset
}

// SelectedTask returns the task at the current cursor.
func (t *TaskListModel) SelectedTask() *api.Task {
	if t.cursor >= 0 && t.cursor < len(t.orderedIndices) {
		idx := t.orderedIndices[t.cursor]
		if idx >= 0 && idx < len(t.tasks) {
			return &t.tasks[idx]
		}
	}
	// Fallback to direct index
	if t.cursor >= 0 && t.cursor < len(t.tasks) {
		return &t.tasks[t.cursor]
	}
	return nil
}

// MoveCursor moves the cursor by delta.
func (t *TaskListModel) MoveCursor(delta int) {
	maxItems := len(t.tasks)
	if len(t.orderedIndices) > 0 {
		maxItems = len(t.orderedIndices)
	}

	t.cursor += delta
	if t.cursor < 0 {
		t.cursor = 0
	}
	if maxItems > 0 && t.cursor >= maxItems {
		t.cursor = maxItems - 1
	}
}

// moveCursorToEnd moves cursor to the last item.
func (t *TaskListModel) moveCursorToEnd() {
	maxItems := len(t.tasks)
	if len(t.orderedIndices) > 0 {
		maxItems = len(t.orderedIndices)
	}
	if maxItems > 0 {
		t.cursor = maxItems - 1
	}
}

// renderFlatTasks renders a simple flat list.
func (t *TaskListModel) renderFlatTasks(maxHeight int) string {
	var lines []LineInfo
	t.orderedIndices = nil

	for i := range t.tasks {
		t.orderedIndices = append(t.orderedIndices, i)
		lines = append(lines, LineInfo{Content: t.renderTask(i), TaskIndex: i})
	}

	return t.renderScrollableLines(lines, maxHeight)
}

// renderGroupedTasks renders tasks grouped by overdue/today/other.
func (t *TaskListModel) renderGroupedTasks(maxHeight int) string {
	var overdue, today, other []int

	for i, task := range t.tasks {
		if task.IsOverdue() {
			overdue = append(overdue, i)
		} else if task.IsDueToday() {
			today = append(today, i)
		} else {
			other = append(other, i)
		}
	}

	// Build ordered indices
	t.orderedIndices = nil
	t.orderedIndices = append(t.orderedIndices, overdue...)
	t.orderedIndices = append(t.orderedIndices, today...)
	t.orderedIndices = append(t.orderedIndices, other...)

	// Build lines
	var lines []LineInfo

	if len(overdue) > 0 {
		lines = append(lines, LineInfo{Content: styles.SectionHeader.Render("OVERDUE"), TaskIndex: -1})
		for _, i := range overdue {
			lines = append(lines, LineInfo{Content: t.renderTask(i), TaskIndex: i})
		}
	}

	if len(today) > 0 {
		if len(overdue) > 0 {
			lines = append(lines, LineInfo{Content: "", TaskIndex: -1})
		}
		for _, i := range today {
			lines = append(lines, LineInfo{Content: t.renderTask(i), TaskIndex: i})
		}
	}

	if len(other) > 0 {
		if len(overdue) > 0 || len(today) > 0 {
			lines = append(lines, LineInfo{Content: "", TaskIndex: -1})
		}
		lines = append(lines, LineInfo{Content: styles.SectionHeader.Render("NO DUE DATE"), TaskIndex: -1})
		for _, i := range other {
			lines = append(lines, LineInfo{Content: t.renderTask(i), TaskIndex: i})
		}
	}

	return t.renderScrollableLines(lines, maxHeight)
}

// renderProjectTasks renders tasks grouped by section.
func (t *TaskListModel) renderProjectTasks(maxHeight int) string {
	tasksBySection := make(map[string][]int)
	var noSectionTasks []int

	for i, task := range t.tasks {
		if task.SectionID != nil && *task.SectionID != "" {
			tasksBySection[*task.SectionID] = append(tasksBySection[*task.SectionID], i)
		} else {
			noSectionTasks = append(noSectionTasks, i)
		}
	}

	// Build ordered indices
	t.orderedIndices = nil
	t.orderedIndices = append(t.orderedIndices, noSectionTasks...)
	for _, section := range t.sections {
		if indices, exists := tasksBySection[section.ID]; exists {
			t.orderedIndices = append(t.orderedIndices, indices...)
		}
	}

	// Build lines
	var lines []LineInfo

	for _, i := range noSectionTasks {
		lines = append(lines, LineInfo{Content: t.renderTask(i), TaskIndex: i})
	}

	for _, section := range t.sections {
		taskIndices := tasksBySection[section.ID]
		lines = append(lines, LineInfo{Content: "", TaskIndex: -1})
		lines = append(lines, LineInfo{Content: styles.SectionHeader.Render(section.Name), TaskIndex: -1})
		for _, i := range taskIndices {
			lines = append(lines, LineInfo{Content: t.renderTask(i), TaskIndex: i})
		}
	}

	return t.renderScrollableLines(lines, maxHeight)
}

// renderTask renders a single task line.
func (t *TaskListModel) renderTask(taskIndex int) string {
	task := t.tasks[taskIndex]

	// Find display position for cursor
	displayPos := 0
	for i, idx := range t.orderedIndices {
		if idx == taskIndex {
			displayPos = i
			break
		}
	}

	// Cursor
	cursor := "  "
	if displayPos == t.cursor && t.focused {
		cursor = "> "
	}

	// Checkbox
	checkbox := styles.CheckboxUnchecked
	if task.Checked {
		checkbox = styles.CheckboxChecked
	}

	// Indent for subtasks
	indent := ""
	if task.ParentID != nil {
		indent = "  "
	}

	// Content with priority color
	content := task.Content
	priorityStyle := styles.GetPriorityStyle(task.Priority)
	content = priorityStyle.Render(content)

	// Due date
	due := ""
	if task.Due != nil {
		dueStr := task.DueDisplay()
		if task.IsOverdue() {
			due = styles.TaskDueOverdue.Render("| " + dueStr)
		} else if task.IsDueToday() {
			due = styles.TaskDueToday.Render("| " + dueStr)
		} else {
			due = styles.TaskDue.Render("| " + dueStr)
		}
	}

	// Labels
	labels := ""
	if len(task.Labels) > 0 {
		labelStrs := make([]string, len(task.Labels))
		for i, l := range task.Labels {
			labelStrs[i] = "@" + l
		}
		labels = styles.TaskLabel.Render(strings.Join(labelStrs, " "))
	}

	// Build line
	line := fmt.Sprintf("%s%s%s %s %s %s", cursor, indent, checkbox, content, due, labels)

	// Apply style
	style := styles.TaskItem
	if displayPos == t.cursor && t.focused {
		style = styles.TaskSelected
	}
	if task.Checked {
		style = styles.TaskCompleted
	}

	return style.Render(line)
}

// renderScrollableLines renders lines with viewport scrolling.
func (t *TaskListModel) renderScrollableLines(lines []LineInfo, maxHeight int) string {
	if len(lines) == 0 {
		t.scrollOffset = 0
		t.viewportLines = nil
		return ""
	}

	// Build content and mapping
	var content strings.Builder
	t.viewportLines = make([]int, 0, len(lines))

	for i, line := range lines {
		content.WriteString(line.Content)
		if i < len(lines)-1 {
			content.WriteString("\n")
		}
		t.viewportLines = append(t.viewportLines, line.TaskIndex)
	}

	// Find cursor line
	cursorLine := 0
	if t.cursor >= 0 && t.cursor < len(t.orderedIndices) {
		targetTaskIndex := t.orderedIndices[t.cursor]
		for i, line := range lines {
			if line.TaskIndex == targetTaskIndex {
				cursorLine = i
				break
			}
		}
	}

	// Use viewport if ready
	if t.viewportReady {
		if t.viewport.Height != maxHeight && maxHeight > 0 {
			t.viewport.Height = maxHeight
		}
		t.viewport.SetContent(content.String())
		t.syncViewportToCursor(cursorLine)
		t.scrollOffset = t.viewport.YOffset
		return t.viewport.View()
	}

	t.scrollOffset = 0
	return content.String()
}

// syncViewportToCursor ensures the viewport shows the cursor line.
func (t *TaskListModel) syncViewportToCursor(cursorLine int) {
	vpHeight := t.viewport.Height
	if vpHeight <= 0 {
		return
	}

	currentTop := t.viewport.YOffset
	currentBottom := currentTop + vpHeight - 1

	if cursorLine < currentTop {
		t.viewport.SetYOffset(cursorLine)
	} else if cursorLine > currentBottom {
		t.viewport.SetYOffset(cursorLine - vpHeight + 1)
	}
}
