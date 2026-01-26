package components

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// SidebarModel manages the project sidebar navigation.
type SidebarModel struct {
	items           []SidebarItem
	cursor          int
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

// View implements Component.
func (s *SidebarModel) View() string {
	var b strings.Builder

	// Title
	b.WriteString(styles.Title.Render("Projects"))
	b.WriteString("\n\n")

	// Max name length (accounting for cursor, indent, icon, and padding)
	maxNameLen := s.width - 10

	// Render project items
	for i, item := range s.items {
		if item.Type == "separator" {
			b.WriteString(styles.SidebarSeparator.Render(strings.Repeat("─", s.width-4)))
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

		// Indent for child projects
		indent := ""
		nameMaxLen := maxNameLen
		if item.ParentID != nil {
			indent = "  "
			nameMaxLen = s.width - 12 // Less space for indented items
		}

		// Truncate long names
		name := item.Name
		countStr := ""
		if item.Count > 0 {
			countStr = fmt.Sprintf(" (%d)", item.Count)
		}

		totalLen := len(name) + len(countStr)
		if totalLen > nameMaxLen && nameMaxLen > 3 {
			// avail is strict
			avail := nameMaxLen - len(countStr)
			if avail > 1 {
				name = name[:avail-1] + "…"
			} else {
				// Just truncate name entirely if space is super tight, but keep count?
				// Or just truncate string.
				// Simple approach: combine first then truncate? No, count is important.
				name = name[:nameMaxLen-1] + "…"
				countStr = "" // Hide count if no space?
			}
		}

		// Actually better:
		if len(name)+len(countStr) > nameMaxLen {
			avail := nameMaxLen - len(countStr)
			if avail > 1 {
				name = name[:avail-1] + "…"
			} else {
				name = name[:nameMaxLen-1] + "…"
				countStr = ""
			}
		}

		line := style.Render(fmt.Sprintf("%s%s%s %s%s", cursor, indent, item.Icon, name, countStr))
		b.WriteString(line)
		b.WriteString("\n")
	}

	// Add hint for creating new project
	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("n: new project"))

	// Apply container style with fixed height
	innerHeight := s.height - 2
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

// SetProjects rebuilds the sidebar items from the given projects.
func (s *SidebarModel) SetProjects(projects []api.Project, counts map[string]int) {
	s.items = []SidebarItem{}

	// Add favorite projects first
	for _, p := range projects {
		if p.IsFavorite {
			icon := "⭐"
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
			icon := "📁"
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
