package components

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// HelpModel renders the help view with keyboard shortcuts.
type HelpModel struct {
	width, height int
	keymap        [][]string
}

// NewHelp creates a new HelpModel.
func NewHelp() *HelpModel {
	return &HelpModel{
		keymap: defaultHelpItems(),
	}
}

// defaultHelpItems returns the default help items.
func defaultHelpItems() [][]string {
	return [][]string{
		{"Navigation", ""},
		{"j/k", "Move up/down"},
		{"gg/G", "Go to top/bottom"},
		{"ctrl+u/ctrl+d", "Half page up/down"},
		{"tab", "Switch pane"},
		{"", ""},
		{"Task Actions", ""},
		{"enter", "Open task details"},
		{"a", "Add new task"},
		{"e", "Edit task"},
		{"x", "Complete/uncomplete task"},
		{"dd", "Delete task"},
		{"1-4", "Set priority"},
		{"</> ", "Due today/tomorrow"},
		{"", ""},
		{"Calendar", ""},
		{"v", "Switch calendar view"},
		{"h/l", "Previous/next day"},
		{"[/]", "Previous/next month"},
		{"", ""},
		{"General", ""},
		{"r", "Refresh data"},
		{"/", "Search"},
		{"?", "Toggle help"},
		{"esc", "Go back / Cancel"},
		{"q", "Quit"},
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
	var b strings.Builder

	b.WriteString(styles.Title.Render("⌨️  Keyboard Shortcuts"))
	b.WriteString("\n\n")

	for _, item := range h.keymap {
		if len(item) < 2 {
			continue
		}
		key := item[0]
		desc := item[1]

		if key == "" && desc == "" {
			b.WriteString("\n")
			continue
		}

		if desc == "" {
			// Section header
			b.WriteString(styles.SectionHeader.Render(key))
			b.WriteString("\n")
		} else {
			// Key-description pair
			keyStr := styles.StatusBarKey.Render(key)
			descStr := styles.HelpDesc.Render(desc)
			b.WriteString("  " + keyStr + "  " + descStr + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(styles.HelpDesc.Render("Press ESC or ? to close"))

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
