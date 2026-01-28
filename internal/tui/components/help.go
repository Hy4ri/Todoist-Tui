package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// HelpItem represents a single help entry.
type HelpItem struct {
	Key  string
	Desc string
}

// HelpModel renders the help view with keyboard shortcuts.
type HelpModel struct {
	width, height int
	keymap        [][]string
}

// NewHelp creates a new HelpModel.
func NewHelp() *HelpModel {
	return &HelpModel{
		keymap: nil, // Will be set by renderer
	}
}

// Init implements Component.
func (h *HelpModel) Init() tea.Cmd {
	return nil
}

// Update implements Component.
func (h *HelpModel) Update(msg tea.Msg) (Component, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "?", "q":
			return h, func() tea.Msg {
				return ViewChangeRequestMsg{View: ViewToday} // Request to go back
			}
		}
	}
	return h, nil
}

// View implements Component.
func (h *HelpModel) View() string {
	if len(h.keymap) == 0 {
		return styles.Dialog.Render("No keybindings registered")
	}

	var b strings.Builder
	b.WriteString(styles.Title.Render("⌨️  Keyboard Shortcuts"))
	b.WriteString("\n\n")

	// Split sections into two columns for better space utilization
	// Column 1: Tab Navigation, Navigation, General
	// Column 2: Task Actions, Label/Project Actions, Calendar View
	var col1Sections = map[string]bool{
		"Tab Navigation": true,
		"Navigation":     true,
		"General":        true,
	}

	var col1Content, col2Content strings.Builder
	var currentColumn *strings.Builder = &col1Content

	for _, item := range h.keymap {
		if len(item) < 2 {
			continue
		}
		key := item[0]
		desc := item[1]

		// Check if this is a section header to potentially switch columns
		if desc == "" && key != "" {
			if col1Sections[key] {
				currentColumn = &col1Content
			} else {
				currentColumn = &col2Content
			}
			currentColumn.WriteString("\n" + styles.SectionHeader.Render(" "+key+" ") + "\n")
			continue
		}

		if key == "" && desc == "" {
			currentColumn.WriteString("\n")
			continue
		}

		// Key-description pair
		// For better alignment, use a fixed width for the key padding
		keyStyle := styles.HelpKey.Copy().Width(12).Align(lipgloss.Right).PaddingRight(2)
		keyStr := keyStyle.Render(key)
		descStr := styles.HelpDesc.Render(desc)
		currentColumn.WriteString(keyStr + descStr + "\n")
	}

	// Join columns horizontally with some padding
	col1 := col1Content.String()
	col2 := col2Content.String()

	// Ensure symmetric padding even if one column is shorter
	colWidth := h.width / 2
	if colWidth > 50 {
		colWidth = 50 // Cap column width for better readability
	}

	columnStyle := lipgloss.NewStyle().Width(colWidth).PaddingLeft(2).PaddingRight(2)
	helpView := lipgloss.JoinHorizontal(lipgloss.Top,
		columnStyle.Render(col1),
		columnStyle.Render(col2),
	)

	b.WriteString(helpView)
	b.WriteString("\n\n")

	// Centered help footer
	footer := styles.HelpDesc.Render("Press ESC or ? to close • j/k: scroll")
	b.WriteString(lipgloss.NewStyle().Width(h.width).Align(lipgloss.Center).Render(footer))

	return b.String()
}

// SetSize implements Component.
func (h *HelpModel) SetSize(width, height int) {
	h.width = width
	h.height = height
}

// SetKeymap sets custom help items.
func (h *HelpModel) SetKeymap(items [][]string) {
	h.keymap = items
}
