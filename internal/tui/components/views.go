package components

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// LabelsModel manages the labels view.
type LabelsModel struct {
	labels        []api.Label
	allTasks      []api.Task
	tasks         []api.Task // Filtered tasks for selected label
	currentLabel  *api.Label
	cursor        int
	width, height int
	focused       bool
}

// NewLabels creates a new LabelsModel.
func NewLabels() *LabelsModel {
	return &LabelsModel{
		labels:   []api.Label{},
		allTasks: []api.Task{},
		tasks:    []api.Task{},
		cursor:   0,
		focused:  false,
	}
}

// Init implements Component.
func (l *LabelsModel) Init() tea.Cmd {
	return nil
}

// Update implements Component.
func (l *LabelsModel) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return l.handleKeyMsg(msg)
	}
	return l, nil
}

// handleKeyMsg processes keyboard input.
func (l *LabelsModel) handleKeyMsg(msg tea.KeyMsg) (Component, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		l.moveCursor(1)
	case "k", "up":
		l.moveCursor(-1)
	case "g":
		l.cursor = 0
	case "G":
		l.moveCursorToEnd()
	case "esc":
		if l.currentLabel != nil {
			l.currentLabel = nil
			l.tasks = nil
			l.cursor = 0
		}
	case "enter":
		if l.currentLabel == nil {
			// Select a label
			labels := l.getLabels()
			if l.cursor < len(labels) {
				l.currentLabel = &labels[l.cursor]
				l.filterTasksByLabel()
				l.cursor = 0
			}
		} else {
			// Select a task for detail
			if l.cursor < len(l.tasks) {
				task := l.tasks[l.cursor]
				return l, func() tea.Msg {
					return TaskSelectedMsg{Task: &task}
				}
			}
		}
	}
	return l, nil
}

// View implements Component.
func (l *LabelsModel) View() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("Labels"))
	b.WriteString("\n\n")

	if l.currentLabel != nil {
		return l.renderLabelTasks(&b)
	}
	return l.renderLabelList(&b)
}

// renderLabelList renders the list of labels.
func (l *LabelsModel) renderLabelList(b *strings.Builder) string {
	labels := l.getLabels()
	taskCounts := l.getLabelTaskCounts()

	if len(labels) == 0 {
		b.WriteString(styles.HelpDesc.Render("No labels found"))
		return b.String()
	}

	contentHeight := l.height - 4
	startIdx := 0
	if l.cursor >= contentHeight {
		startIdx = l.cursor - contentHeight + 1
	}
	endIdx := startIdx + contentHeight
	if endIdx > len(labels) {
		endIdx = len(labels)
	}

	for i := startIdx; i < endIdx; i++ {
		label := labels[i]
		cursor := "  "
		style := styles.LabelItem
		if i == l.cursor && l.focused {
			cursor = "> "
			style = styles.LabelSelected
		}

		name := "@" + label.Name
		if label.Color != "" {
			name = lipgloss.NewStyle().Foreground(lipgloss.Color(label.Color)).Render(name)
		}

		countBadge := ""
		if count := taskCounts[label.Name]; count > 0 {
			countBadge = styles.HelpDesc.Render(fmt.Sprintf(" (%d)", count))
		}

		line := fmt.Sprintf("%s%s%s", cursor, name, countBadge)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("Enter: view tasks | n: new label"))

	return b.String()
}

// renderLabelTasks renders tasks for the selected label.
func (l *LabelsModel) renderLabelTasks(b *strings.Builder) string {
	labelTitle := "@" + l.currentLabel.Name
	if l.currentLabel.Color != "" {
		labelTitle = lipgloss.NewStyle().Foreground(lipgloss.Color(l.currentLabel.Color)).Render(labelTitle)
	}
	b.WriteString(styles.Subtitle.Render(labelTitle))
	b.WriteString("\n\n")

	if len(l.tasks) == 0 {
		b.WriteString(styles.HelpDesc.Render("No tasks with this label"))
	} else {
		contentHeight := l.height - 6
		startIdx := 0
		if l.cursor >= contentHeight {
			startIdx = l.cursor - contentHeight + 1
		}
		endIdx := startIdx + contentHeight
		if endIdx > len(l.tasks) {
			endIdx = len(l.tasks)
		}

		for i := startIdx; i < endIdx; i++ {
			task := l.tasks[i]
			cursor := "  "
			style := styles.TaskItem
			if i == l.cursor && l.focused {
				cursor = "> "
				style = styles.TaskSelected
			}

			checkbox := styles.CheckboxUnchecked
			if task.Checked {
				checkbox = styles.CheckboxChecked
				style = styles.TaskCompleted
			}

			content := styles.GetPriorityStyle(task.Priority).Render(task.Content)
			line := fmt.Sprintf("%s%s %s", cursor, checkbox, content)
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("ESC: back to labels"))

	return b.String()
}

// SetSize implements Component.
func (l *LabelsModel) SetSize(width, height int) {
	l.width = width
	l.height = height
}

// Focus sets focus.
func (l *LabelsModel) Focus() {
	l.focused = true
}

// Blur removes focus.
func (l *LabelsModel) Blur() {
	l.focused = false
}

// Focused returns focus state.
func (l *LabelsModel) Focused() bool {
	return l.focused
}

// SetLabels sets the labels list.
func (l *LabelsModel) SetLabels(labels []api.Label) {
	l.labels = labels
}

// SetAllTasks sets all tasks for counting and filtering.
func (l *LabelsModel) SetAllTasks(tasks []api.Task) {
	l.allTasks = tasks
	if l.currentLabel != nil {
		l.filterTasksByLabel()
	}
}

// CurrentLabel returns the currently selected label.
func (l *LabelsModel) CurrentLabel() *api.Label {
	return l.currentLabel
}

// ClearSelection clears the current label selection.
func (l *LabelsModel) ClearSelection() {
	l.currentLabel = nil
	l.tasks = nil
	l.cursor = 0
}

// Cursor returns the current cursor position.
func (l *LabelsModel) Cursor() int {
	return l.cursor
}

// moveCursor moves the cursor by delta.
func (l *LabelsModel) moveCursor(delta int) {
	maxItems := len(l.getLabels())
	if l.currentLabel != nil {
		maxItems = len(l.tasks)
	}

	l.cursor += delta
	if l.cursor < 0 {
		l.cursor = 0
	}
	if maxItems > 0 && l.cursor >= maxItems {
		l.cursor = maxItems - 1
	}
}

// moveCursorToEnd moves cursor to the last item.
func (l *LabelsModel) moveCursorToEnd() {
	maxItems := len(l.getLabels())
	if l.currentLabel != nil {
		maxItems = len(l.tasks)
	}
	if maxItems > 0 {
		l.cursor = maxItems - 1
	}
}

// getLabels returns labels, extracting from tasks if needed.
func (l *LabelsModel) getLabels() []api.Label {
	if len(l.labels) > 0 {
		return l.labels
	}
	return l.extractLabelsFromTasks()
}

// extractLabelsFromTasks extracts unique labels from all tasks.
func (l *LabelsModel) extractLabelsFromTasks() []api.Label {
	labelMap := make(map[string]bool)
	var labels []api.Label

	for _, t := range l.allTasks {
		for _, labelName := range t.Labels {
			if !labelMap[labelName] {
				labelMap[labelName] = true
				labels = append(labels, api.Label{Name: labelName})
			}
		}
	}

	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Name < labels[j].Name
	})

	return labels
}

// getLabelTaskCounts returns a map of label name to task count.
func (l *LabelsModel) getLabelTaskCounts() map[string]int {
	counts := make(map[string]int)
	for _, t := range l.allTasks {
		for _, labelName := range t.Labels {
			counts[labelName]++
		}
	}
	return counts
}

// filterTasksByLabel filters tasks by the current label.
func (l *LabelsModel) filterTasksByLabel() {
	if l.currentLabel == nil {
		l.tasks = nil
		return
	}

	l.tasks = nil
	for _, t := range l.allTasks {
		for _, labelName := range t.Labels {
			if labelName == l.currentLabel.Name {
				l.tasks = append(l.tasks, t)
				break
			}
		}
	}
}

// SearchModel manages the search view.
type SearchModel struct {
	input         textinput.Model
	query         string
	allTasks      []api.Task
	results       []api.Task
	cursor        int
	width, height int
	active        bool
}

// NewSearch creates a new SearchModel.
func NewSearch() *SearchModel {
	ti := textinput.New()
	ti.Placeholder = "Search tasks..."
	ti.CharLimit = 100
	return &SearchModel{
		input:    ti,
		allTasks: []api.Task{},
		results:  []api.Task{},
		cursor:   0,
		active:   false,
	}
}

// Init implements Component.
func (s *SearchModel) Init() tea.Cmd {
	return nil
}

// Update implements Component.
func (s *SearchModel) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return s.handleKeyMsg(msg)
	}
	return s, nil
}

// handleKeyMsg processes keyboard input.
func (s *SearchModel) handleKeyMsg(msg tea.KeyMsg) (Component, tea.Cmd) {
	switch msg.String() {
	case "esc":
		s.active = false
		s.input.Blur()
	case "enter":
		if s.input.Focused() {
			s.query = s.input.Value()
			s.filterResults()
			s.input.Blur()
			s.cursor = 0
		} else if s.cursor < len(s.results) {
			task := s.results[s.cursor]
			return s, func() tea.Msg {
				return TaskSelectedMsg{Task: &task}
			}
		}
	case "j", "down":
		if !s.input.Focused() {
			s.moveCursor(1)
		}
	case "k", "up":
		if !s.input.Focused() {
			s.moveCursor(-1)
		}
	case "/":
		s.input.Focus()
	default:
		if s.input.Focused() {
			var cmd tea.Cmd
			s.input, cmd = s.input.Update(msg)
			s.query = s.input.Value()
			s.filterResults()
			return s, cmd
		}
	}
	return s, nil
}

// View implements Component.
func (s *SearchModel) View() string {
	var b strings.Builder

	b.WriteString(styles.Title.Render("🔍 Search"))
	b.WriteString("\n\n")
	b.WriteString(s.input.View())
	b.WriteString("\n\n")

	if s.query == "" {
		b.WriteString(styles.HelpDesc.Render("Type to search tasks..."))
		return b.String()
	}

	if len(s.results) == 0 {
		b.WriteString(styles.HelpDesc.Render("No matching tasks"))
		return b.String()
	}

	b.WriteString(styles.Subtitle.Render(fmt.Sprintf("%d results", len(s.results))))
	b.WriteString("\n\n")

	contentHeight := s.height - 8
	startIdx := 0
	if s.cursor >= contentHeight {
		startIdx = s.cursor - contentHeight + 1
	}
	endIdx := startIdx + contentHeight
	if endIdx > len(s.results) {
		endIdx = len(s.results)
	}

	for i := startIdx; i < endIdx; i++ {
		task := s.results[i]
		cursor := "  "
		style := styles.TaskItem
		if i == s.cursor && !s.input.Focused() {
			cursor = "> "
			style = styles.TaskSelected
		}

		checkbox := styles.CheckboxUnchecked
		if task.Checked {
			checkbox = styles.CheckboxChecked
			style = styles.TaskCompleted
		}

		content := styles.GetPriorityStyle(task.Priority).Render(task.Content)
		line := fmt.Sprintf("%s%s %s", cursor, checkbox, content)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	return b.String()
}

// SetSize implements Component.
func (s *SearchModel) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.input.Width = width - 4
}

// Focus activates search.
func (s *SearchModel) Focus() {
	s.active = true
	s.input.Focus()
}

// Blur deactivates search.
func (s *SearchModel) Blur() {
	s.active = false
	s.input.Blur()
}

// Focused returns if search is active.
func (s *SearchModel) Focused() bool {
	return s.active
}

// SetAllTasks sets the tasks to search.
func (s *SearchModel) SetAllTasks(tasks []api.Task) {
	s.allTasks = tasks
	if s.query != "" {
		s.filterResults()
	}
}

// Clear resets the search.
func (s *SearchModel) Clear() {
	s.query = ""
	s.input.SetValue("")
	s.results = nil
	s.cursor = 0
}

// moveCursor moves the cursor by delta.
func (s *SearchModel) moveCursor(delta int) {
	s.cursor += delta
	if s.cursor < 0 {
		s.cursor = 0
	}
	if s.cursor >= len(s.results) {
		s.cursor = len(s.results) - 1
	}
	if s.cursor < 0 {
		s.cursor = 0
	}
}

// filterResults filters tasks by the search query.
func (s *SearchModel) filterResults() {
	if s.query == "" {
		s.results = nil
		return
	}

	query := strings.ToLower(s.query)
	s.results = nil

	for _, t := range s.allTasks {
		if strings.Contains(strings.ToLower(t.Content), query) {
			s.results = append(s.results, t)
			continue
		}
		if strings.Contains(strings.ToLower(t.Description), query) {
			s.results = append(s.results, t)
			continue
		}
		for _, label := range t.Labels {
			if strings.Contains(strings.ToLower(label), query) {
				s.results = append(s.results, t)
				break
			}
		}
	}
}
