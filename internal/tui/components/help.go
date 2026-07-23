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

type helpSection struct {
	title string
	items []HelpItem
}

var sectionIcons = map[string]string{
	"Tab Navigation":        "📑",
	"Navigation":            "🧭",
	"Task Actions":          "📝",
	"Label/Project Actions": "🏷️",
	"Calendar View":         "🗓️",
	"General":               "⚙️",
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

func parseHelpSections(keymap [][]string) []helpSection {
	var sections []helpSection
	var current *helpSection

	for _, item := range keymap {
		if len(item) < 2 {
			continue
		}
		key, desc := item[0], item[1]

		if desc == "" && key != "" {
			if current != nil && len(current.items) > 0 {
				sections = append(sections, *current)
			}
			current = &helpSection{title: key}
			continue
		}

		if key == "" && desc == "" {
			continue
		}

		if current != nil {
			current.items = append(current.items, HelpItem{Key: key, Desc: desc})
		}
	}

	if current != nil && len(current.items) > 0 {
		sections = append(sections, *current)
	}

	return sections
}

func renderSectionBox(sec helpSection, boxWidth int) string {
	icon := sectionIcons[sec.title]
	if icon == "" {
		icon = "💡"
	}

	// Title line
	titleText := icon + " " + sec.title
	headerStr := styles.HelpBoxTitle.Render(titleText)

	// Determine key column width inside box
	maxKeyLen := 0
	for _, item := range sec.items {
		if len(item.Key) > maxKeyLen {
			maxKeyLen = len(item.Key)
		}
	}

	// Calculate inner width (boxWidth minus 2 border chars and 2 padding chars)
	innerWidth := boxWidth - 4
	if innerWidth < 20 {
		innerWidth = 20
	}

	if maxKeyLen > innerWidth/2 {
		maxKeyLen = innerWidth / 2
	}
	if maxKeyLen < 4 {
		maxKeyLen = 4
	}

	var rows []string
	rows = append(rows, headerStr, "")

	for _, item := range sec.items {
		keyStr := styles.HelpKey.Width(maxKeyLen).Align(lipgloss.Right).Render(item.Key)
		sepStr := styles.HelpSeparator.Render("  ")
		descWidth := innerWidth - maxKeyLen - 2
		if descWidth < 8 {
			descWidth = 8
		}
		descStr := styles.HelpDesc.Width(descWidth).Render(item.Desc)
		rows = append(rows, keyStr+sepStr+descStr)
	}

	boxContent := strings.Join(rows, "\n")
	return styles.HelpBox.Width(boxWidth).Render(boxContent)
}

// View implements Component.
func (h *HelpModel) View() string {
	if len(h.keymap) == 0 {
		return styles.Dialog.Render("No keybindings registered")
	}

	sections := parseHelpSections(h.keymap)
	if len(sections) == 0 {
		return styles.Dialog.Render("No keybindings registered")
	}

	targetWidth := h.width
	if targetWidth <= 0 {
		targetWidth = 100
	}

	var numCols int
	if targetWidth >= 115 {
		numCols = 3
	} else if targetWidth >= 75 {
		numCols = 2
	} else {
		numCols = 1
	}

	gap := 2
	boxWidth := (targetWidth - (numCols+1)*gap) / numCols
	if boxWidth > 55 {
		boxWidth = 55
	}
	if boxWidth < 30 && numCols > 1 {
		numCols = 1
		boxWidth = targetWidth - 4
	}
	if boxWidth < 25 {
		boxWidth = 25
	}

	// Height-balanced column distribution
	cols := make([][]string, numCols)
	colHeights := make([]int, numCols)

	for _, sec := range sections {
		boxStr := renderSectionBox(sec, boxWidth)
		boxHeight := lipgloss.Height(boxStr)

		minCol := 0
		for c := 1; c < numCols; c++ {
			if colHeights[c] < colHeights[minCol] {
				minCol = c
			}
		}

		cols[minCol] = append(cols[minCol], boxStr)
		colHeights[minCol] += boxHeight + 1
	}

	var renderedCols []string
	colStyle := lipgloss.NewStyle().MarginRight(gap)

	for c := 0; c < numCols; c++ {
		colContent := strings.Join(cols[c], "\n")
		if c < numCols-1 {
			renderedCols = append(renderedCols, colStyle.Render(colContent))
		} else {
			renderedCols = append(renderedCols, colContent)
		}
	}

	grid := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)

	var b strings.Builder

	// Top Title Banner
	titleBanner := styles.Title.Render("⌨️  Keyboard Shortcuts")
	b.WriteString(lipgloss.NewStyle().Width(targetWidth).Align(lipgloss.Center).Render(titleBanner))
	b.WriteString("\n\n")

	// Grid of Boxed Sections
	b.WriteString(grid)
	b.WriteString("\n\n")

	// Centered Footer
	footer := styles.HelpDesc.Render("Press ESC, q, or ? to close")
	b.WriteString(lipgloss.NewStyle().Width(targetWidth).Align(lipgloss.Center).Render(footer))

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
