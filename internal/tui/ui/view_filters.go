package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/components"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// renderFiltersTab renders the filters tab content (Sidebar + Task List).
func (r *Renderer) renderFiltersTab(width, height int) string {
	// Sidebar width logic matching Projects tab
	sidebarWidth := 30
	if width < 70 {
		sidebarWidth = 20
	}
	if width < 50 {
		sidebarWidth = 15
	}

	listWidth := width - sidebarWidth - 4 // Gap/borders

	// Render Sidebar
	sidebar := r.renderFilterSidebar(sidebarWidth, height)

	// Render Content (Task List)
	var content string
	if len(r.Tasks) == 0 && r.Loading {
		content = "Loading tasks..."
		content = lipgloss.NewStyle().Foreground(styles.Subtle).Render(content)
	} else if len(r.Tasks) == 0 {
		if r.CurrentFilter != nil {
			content = fmt.Sprintf("No tasks match filter: %s", r.CurrentFilter.Name)
		} else {
			content = "Select a filter to view tasks."
		}
		content = styles.MainContent.Width(listWidth).Render(content)
	} else {
		content = r.renderTaskList(listWidth, height)
	}

	// Layout
	sidebar = lipgloss.Place(sidebarWidth, height, lipgloss.Left, lipgloss.Top, sidebar)
	content = lipgloss.Place(listWidth, height, lipgloss.Left, lipgloss.Top, content)

	return lipgloss.JoinHorizontal(lipgloss.Top, sidebar, " ", content)
}

// renderFilterSidebar renders the filter list with search input using SidebarComp.
func (r *Renderer) renderFilterSidebar(width, height int) string {
	var items []components.SidebarItem
	query := strings.ToLower(r.FilterInput.Value())

	for _, f := range r.Filters {
		// Apply search filter
		if r.IsFilterSearch && query != "" && !strings.Contains(strings.ToLower(f.Name), query) {
			continue
		}

		// Use filter color if available (defaults to charcoal/grey if empty)
		color := f.Color
		if color == "" {
			color = "charcoal"
		}

		items = append(items, components.SidebarItem{
			Type:  "project",
			ID:    f.ID,
			Name:  f.Name,
			Color: color,
		})
	}

	// Configure component
	r.FilterSidebarComp.SetItems(items)
	r.FilterSidebarComp.SetSize(width, height)

	// Sync cursor
	// Note: FilterCursor points to the *visible* list index in our logic (update_filters.go helper)?
	// Or the absolute index?
	// In update_filters.go, moveFilterCursor uses getVisibleFilters(). So cursor is relative to VISIBLE list.
	// SidebarComp cursor expects absolute index relative to ITEMS passed to it.
	// Since we passed 'items' which IS the visible list, the indices match!
	r.FilterSidebarComp.SetCursor(r.FilterCursor)

	if r.IsFilterSearch {
		r.FilterSidebarComp.Title = r.FilterInput.View()
	} else {
		r.FilterSidebarComp.Title = "Filters (/)"
	}

	if r.FocusedPane == state.PaneSidebar {
		r.FilterSidebarComp.Focus()
	} else {
		r.FilterSidebarComp.Blur()
	}

	return r.FilterSidebarComp.View()
}

// renderFilterFormDialog renders the filter creation/editing dialog
func (r *Renderer) renderFilterFormDialog() string {
	title := "New Filter"
	if r.IsEditingFilter {
		title = "Edit Filter"
	}

	highlightStyle := lipgloss.NewStyle().Foreground(styles.Highlight).Bold(true)

	// Build form content
	var content strings.Builder
	content.WriteString(styles.Title.Render(title) + "\n\n")

	// Name field
	nameLabel := "Name:"
	if r.FilterFormStep == 0 {
		nameLabel = highlightStyle.Render("▶ Name:")
	}
	content.WriteString(nameLabel + "\n")
	content.WriteString(r.FilterNameInput.View() + "\n\n")

	// Query field
	queryLabel := "Query:"
	if r.FilterFormStep == 1 {
		queryLabel = highlightStyle.Render("▶ Query:")
	}
	content.WriteString(queryLabel + "\n")
	content.WriteString(r.FilterQueryInput.View() + "\n\n")

	// Color field
	colorLabel := "Color:"
	if r.FilterFormStep == 2 {
		colorLabel = highlightStyle.Render("▶ Color:")
	}
	content.WriteString(colorLabel + "\n")

	// Show color picker
	maxColors := 6 // Show 6 colors at a time
	startIdx := 0
	if r.ColorCursor >= maxColors {
		startIdx = r.ColorCursor - maxColors + 1
	}
	endIdx := startIdx + maxColors
	if endIdx > len(r.AvailableColors) {
		endIdx = len(r.AvailableColors)
	}

	for i := startIdx; i < endIdx; i++ {
		c := r.AvailableColors[i]
		colorStyle := lipgloss.NewStyle().
			Background(styles.GetColor(c)).
			Foreground(lipgloss.Color("#ffffff")).
			Padding(0, 1)

		prefix := "  "
		if i == r.ColorCursor {
			prefix = "▶ "
		}
		content.WriteString(prefix + colorStyle.Render(c) + "\n")
	}
	content.WriteString("\n")

	// Hints
	hint := "Tab: switch field • Enter: next/save • Esc: cancel"
	if r.FilterFormStep == 2 {
		hint = "j/k: select color • Enter: save • Esc: cancel"
	}
	content.WriteString(styles.HelpDesc.Render(hint))

	// Dialog box
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Highlight).
		Padding(1, 2).
		Width(50)

	return dialogStyle.Render(content.String())
}

// renderFilterDeleteDialog renders the filter deletion confirmation dialog
func (r *Renderer) renderFilterDeleteDialog() string {
	if r.EditingFilter == nil {
		return ""
	}

	var content strings.Builder
	content.WriteString(styles.Title.Render("Delete Filter?") + "\n\n")
	content.WriteString(fmt.Sprintf("Are you sure you want to delete '%s'?\n\n", r.EditingFilter.Name))
	content.WriteString(styles.HelpDesc.Render("y: yes • n: no"))

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("9")).
		Padding(1, 2).
		Width(45)

	return dialogStyle.Render(content.String())
}
