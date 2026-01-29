package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// SidebarModel manages the project sidebar navigation.
type SidebarModel struct {
	items           []SidebarItem
	cursor          int
	scrollOffset    int
	width, height   int
	focused         bool
	activeProjectID string // Currently selected project
}

// NewSidebar creates a new SidebarModel.
func NewSidebar() *SidebarModel {
	return &SidebarModel{
		items:   []SidebarItem{},
		cursor:  0,
		focused: false,
	}
}

// Init implements Component.
func (s *SidebarModel) Init() tea.Cmd {
	return nil
}

// Update implements Component.
func (s *SidebarModel) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return s.handleKeyMsg(msg)
	}
	return s, nil
}

// handleKeyMsg processes keyboard input for the sidebar.
func (s *SidebarModel) handleKeyMsg(msg tea.KeyMsg) (Component, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		s.MoveCursor(1)
	case "k", "up":
		s.MoveCursor(-1)
	case "g":
		// Note: 'gg' sequence should be handled by parent
		s.cursor = 0
	case "G":
		s.moveCursorToEnd()
	case "enter":
		if s.cursor < len(s.items) && s.items[s.cursor].Type != "separator" {
			item := s.items[s.cursor]
			return s, func() tea.Msg {
				return ProjectSelectedMsg{ID: item.ID, Name: item.Name}
			}
		}
	}
	return s, nil
}

// truncateString truncates a string to a given width and adds an ellipsis if truncated.
func truncateString(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	if width <= 1 {
		return "…"
	}
	res := s
	// Basic retry loop to shorten string until it fits
	// This serves as a local alternative to app.truncateString since we can't import app
	for lipgloss.Width(res+"…") > width && len(res) > 0 {
		runes := []rune(res)
		res = string(runes[:len(runes)-1])
	}
	return res + "…"
}

// View implements Component.
func (s *SidebarModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Projects"))
	b.WriteString("\n\n")

	// Calculate available height for the list
	// Height - Borders(2) - Title(1) - Blank(1) - Footer/Hint(2) = 6?
	// Let's re-verify:
	// b.WriteString(Title) -> 1
	// b.WriteString("\n\n") -> 2
	// ... list items (listHeight)
	// b.WriteString("\n") -> 1
	// b.WriteString(Hint) -> 1
	// Total overhead inside borders = 1 + 2 + 1 + 1 = 5.
	innerHeight := s.height - 2
	listHeight := innerHeight - 5
	if listHeight < 1 {
		listHeight = 1
	}

	// Update scroll offset
	if s.cursor < s.scrollOffset {
		s.scrollOffset = s.cursor
	}
	if s.cursor >= s.scrollOffset+listHeight {
		s.scrollOffset = s.cursor - listHeight + 1
	}
	// Sanity bounds
	if s.scrollOffset > len(s.items)-listHeight {
		s.scrollOffset = len(s.items) - listHeight
	}
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}

	// Determine window
	startIndex := s.scrollOffset
	endIndex := startIndex + listHeight
	if endIndex > len(s.items) {
		endIndex = len(s.items)
	}

	// Max name length (accounting for cursor, indent, icon, and padding)
	maxNameLen := s.width - 10

	// Render project items (windowed)
	for i := startIndex; i < endIndex; i++ {
		item := s.items[i]

		// Separator handling
		if item.Type == "separator" {
			sepWidth := s.width - 6
			if sepWidth < 1 {
				sepWidth = 1
			}
			lineContent := strings.Repeat("─", sepWidth)
			if i == s.cursor && s.focused {
				b.WriteString(styles.SidebarSeparator.Render("> " + lineContent))
			} else {
				b.WriteString(styles.SidebarSeparator.Render("  " + lineContent))
			}
			b.WriteString("\n")
			continue
		}

		cursor := "  "
		style := styles.ProjectItem
		if i == s.cursor && s.focused {
			cursor = "> "
			style = styles.ProjectSelected
		}

		// Highlight active project
		if s.activeProjectID != "" && s.activeProjectID == item.ID {
			if !s.focused {
				style = styles.SidebarActive
			}
		}

		// Indent
		indent := ""
		nameMaxLen := maxNameLen
		if item.ParentID != nil {
			indent = "  "
			nameMaxLen = s.width - 12
		}

		// Truncate name
		name := item.Name
		countStr := ""
		if item.Count > 0 {
			countStr = fmt.Sprintf(" (%d)", item.Count)
		}

		// Combined truncation logic
		// We want: visibleName + countStr <= nameMaxLen
		// Make sure countStr fits first
		if len(countStr) >= nameMaxLen {
			countStr = "" // Too small even for count
		}

		availForName := nameMaxLen - len(countStr)
		name = truncateString(name, availForName)

		// Apply color if available
		itemStyle := style
		if item.Color != "" {
			color := styles.GetColor(item.Color)
			// Apply color to the icon/name but preserve selection style background if selected
			if i == s.cursor && s.focused {
				// Selected item: Keep background/bold, but tint foreground (if desired) or simple keep generic selection
				// Todoist usually keeps specific color icon but white text on selection.
			} else {
				itemStyle = itemStyle.Foreground(color)
			}
		}

		// We construct line manually to colorize just the icon or name
		// Actually lipgloss applies style to the whole rendered string.
		// Let's color the icon specifically if not selected?
		var renderedItem string
		if item.Color != "" {
			color := styles.GetColor(item.Color)
			iconStyle := lipgloss.NewStyle().Foreground(color)

			// If selected, we might want to ensure high contrast.
			// Usually selected = reversed or distinct background.
			if i == s.cursor && s.focused {
				// Use selection style for the whole line.
				// Todoist style: Colored Icon, White Text (default selection text).
				renderedItem = fmt.Sprintf("%s%s%s %s%s", cursor, indent, iconStyle.Render(item.Icon), name, countStr)
			} else {
				// Not selected - apply color to icon AND name
				// Let's apply to icon only for "folder" icons, or bullet points.
				// User asked for "project colors". Todoist usually colors the hashtag/dot.
				// User explicitly asked to make the name the same color.
				coloredName := iconStyle.Render(name)
				renderedItem = fmt.Sprintf("%s%s%s %s%s", cursor, indent, iconStyle.Render(item.Icon), coloredName, countStr)
			}
		} else {
			renderedItem = fmt.Sprintf("%s%s%s %s%s", cursor, indent, item.Icon, name, countStr)
		}

		line := style.MaxWidth(s.width - 2).Render(renderedItem)
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Add hint for creating new project
	// Only if we have space left in the container?
	// Actually, just append it. If container cuts it off, fine, but list shouldn't be cut off.
	// But we calculated listHeight to be carefully smaller.

	// Fill padding if list is short
	linesRendered := endIndex - startIndex
	if linesRendered < listHeight {
		b.WriteString(strings.Repeat("\n", listHeight-linesRendered))
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("n: new project"))

	// Apply container style
	if innerHeight < 3 {
		innerHeight = 3
	}
	containerStyle := styles.Sidebar
	if s.focused {
		containerStyle = styles.SidebarFocused
	}

	return containerStyle.Width(s.width).Height(innerHeight).Render(b.String())
}

// SetSize implements Component.
func (s *SidebarModel) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// Focus sets the sidebar as focused.
func (s *SidebarModel) Focus() {
	s.focused = true
}

// Blur removes focus from the sidebar.
func (s *SidebarModel) Blur() {
	s.focused = false
}

// Focused returns whether the sidebar is focused.
func (s *SidebarModel) Focused() bool {
	return s.focused
}

// SetItems sets the sidebar items directly.
func (s *SidebarModel) SetItems(items []SidebarItem) {
	s.items = items
}

// SetProjects rebuilds the sidebar items from the given projects.
func (s *SidebarModel) SetProjects(projects []api.Project, counts map[string]int) {
	s.items = []SidebarItem{}

	// Add favorite projects first
	for _, p := range projects {
		if p.IsFavorite {
			icon := "❤︎"
			if p.InboxProject {
				icon = "📥"
				_ = icon
			}
			s.items = append(s.items, SidebarItem{
				Type:       "project",
				ID:         p.ID,
				Name:       p.Name,
				Icon:       icon,
				Count:      counts[p.ID],
				IsFavorite: true,
				ParentID:   p.ParentID,
				Color:      p.Color,
			})
		}
	}

	// Add separator if there were favorites
	hasFavorites := false
	for _, p := range projects {
		if p.IsFavorite {
			hasFavorites = true
			break
		}
	}
	if hasFavorites {
		s.items = append(s.items, SidebarItem{Type: "separator", ID: "", Name: ""})
	}

	// Add remaining projects (non-favorites)
	for _, p := range projects {
		if !p.IsFavorite {
			icon := "#"
			if p.InboxProject {
				icon = "📥"
			}
			s.items = append(s.items, SidebarItem{
				Type:     "project",
				ID:       p.ID,
				Name:     p.Name,
				Icon:     icon,
				Count:    counts[p.ID],
				ParentID: p.ParentID,
				Color:    p.Color,
			})
		}
	}
}

// SetActiveProject highlights the given project as active.
func (s *SidebarModel) SetActiveProject(projectID string) {
	s.activeProjectID = projectID
}

// MoveCursor moves the cursor by delta, skipping separators.
func (s *SidebarModel) MoveCursor(delta int) {
	newPos := s.cursor + delta

	// Skip separators
	for newPos >= 0 && newPos < len(s.items) && s.items[newPos].Type == "separator" {
		if delta > 0 {
			newPos++
		} else {
			newPos--
		}
	}

	// Clamp to bounds
	if newPos < 0 {
		newPos = 0
	}
	if newPos >= len(s.items) {
		newPos = len(s.items) - 1
	}

	// Make sure we don't land on a separator
	for newPos >= 0 && newPos < len(s.items) && s.items[newPos].Type == "separator" {
		newPos--
	}

	if newPos >= 0 {
		s.cursor = newPos
	}
}

// moveCursorToEnd moves cursor to the last item.
func (s *SidebarModel) moveCursorToEnd() {
	if len(s.items) > 0 {
		s.cursor = len(s.items) - 1
		// Skip separator
		for s.cursor > 0 && s.items[s.cursor].Type == "separator" {
			s.cursor--
		}
	}
}

// Cursor returns the current cursor position.
func (s *SidebarModel) Cursor() int {
	return s.cursor
}

// SetCursor sets the cursor to the given position.
func (s *SidebarModel) SetCursor(pos int) {
	if pos >= 0 && pos < len(s.items) {
		s.cursor = pos
	}
}

// Items returns the current sidebar items.
func (s *SidebarModel) Items() []SidebarItem {
	return s.items
}

// CurrentItem returns the item at the current cursor position.
func (s *SidebarModel) CurrentItem() *SidebarItem {
	if s.cursor >= 0 && s.cursor < len(s.items) {
		return &s.items[s.cursor]
	}
	return nil
}
