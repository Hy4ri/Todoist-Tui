package logic

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
)

// loadFilters loads filters from API.
func (h *Handler) loadFilters() tea.Cmd {
	return func() tea.Msg {
		filters, err := h.Client.GetFilters()
		if err != nil {
			return errMsg{err}
		}
		return filtersLoadedMsg{filters: filters}
	}
}

type filtersLoadedMsg struct {
	filters []api.Filter
}

// runFilter executes a filter query.
func (h *Handler) runFilter(filter *api.Filter) tea.Cmd {
	h.CurrentFilter = filter
	h.Loading = true
	h.StatusMsg = fmt.Sprintf("Running filter: %s", filter.Name)
	h.Tasks = nil // Clear current tasks

	return func() tea.Msg {
		tasks, err := h.Client.GetTasksByFilter(filter.Query)
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{tasks: tasks}
	}
}

// runAdHocFilter runs a custom query string.
func (h *Handler) runAdHocFilter(query string) tea.Cmd {
	h.CurrentFilter = &api.Filter{
		ID:    "adhoc",
		Name:  "Custom: " + query,
		Query: query,
	}
	h.Loading = true
	h.StatusMsg = fmt.Sprintf("Running query: %s", query)
	h.Tasks = nil

	return func() tea.Msg {
		tasks, err := h.Client.GetTasksByFilter(query)
		if err != nil {
			return errMsg{err}
		}
		return dataLoadedMsg{tasks: tasks}
	}
}

// getVisibleFilters returns filters matching the search query.
func (h *Handler) getVisibleFilters() []api.Filter {
	if h.FilterSearchQuery == "" {
		return h.Filters
	}

	var visible []api.Filter
	query := strings.ToLower(h.FilterSearchQuery)
	for _, f := range h.Filters {
		if strings.Contains(strings.ToLower(f.Name), query) {
			visible = append(visible, f)
		}
	}
	return visible
}

func (h *Handler) moveFilterCursor(delta int) {
	visible := h.getVisibleFilters()
	if len(visible) == 0 {
		return
	}

	h.FilterCursor += delta
	if h.FilterCursor < 0 {
		h.FilterCursor = 0
	} else if h.FilterCursor >= len(visible) {
		h.FilterCursor = len(visible) - 1
	}
}

func (h *Handler) handleFilterSelect() tea.Cmd {
	visible := h.getVisibleFilters()
	if len(visible) == 0 || h.FilterCursor >= len(visible) {
		return nil
	}

	// Selections are view-local: clear them when switching to a different filter.
	h.clearSelection()

	selected := visible[h.FilterCursor]
	return h.runFilter(&selected)
}

// handleNewFilter opens the filter creation dialog
func (h *Handler) handleNewFilter() tea.Cmd {
	h.IsCreatingFilter = true
	h.FilterFormStep = 0
	h.FilterNameInput = textinput.New()
	h.FilterNameInput.Placeholder = "Filter name..."
	h.FilterNameInput.Focus()
	h.FilterNameInput.CharLimit = 50
	h.FilterQueryInput = textinput.New()
	h.FilterQueryInput.Placeholder = "e.g., today | overdue | p1"
	h.FilterQueryInput.CharLimit = 200
	h.SelectedColor = "charcoal" // Default color
	h.ColorCursor = 0
	return textinput.Blink
}

// handleDeleteFilter initiates filter deletion
func (h *Handler) handleDeleteFilter() tea.Cmd {
	visible := h.getVisibleFilters()
	if len(visible) == 0 || h.FilterCursor >= len(visible) {
		return nil
	}

	selected := visible[h.FilterCursor]

	h.EditingFilter = &selected
	h.ConfirmDeleteFilter = true
	return nil
}

// handleFilterFormKeyMsg handles input in filter creation/editing form
// Steps: 0=name, 1=query, 2=color
func (h *Handler) handleFilterFormKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		h.IsCreatingFilter = false
		h.IsEditingFilter = false
		h.FilterFormStep = 0
		return nil
	case "enter":
		if h.FilterFormStep == 0 {
			// Move to query step
			if h.FilterNameInput.Value() == "" {
				h.StatusMsg = "Filter name is required"
				return nil
			}
			h.FilterFormStep = 1
			h.FilterNameInput.Blur()
			h.FilterQueryInput.Focus()
			return textinput.Blink
		} else if h.FilterFormStep == 1 {
			// Move to color step
			if h.FilterQueryInput.Value() == "" {
				h.StatusMsg = "Filter query is required"
				return nil
			}
			h.FilterFormStep = 2
			h.FilterQueryInput.Blur()
			return nil
		} else {
			// Submit on step 2
			return h.submitFilterForm()
		}
	case "tab":
		// Cycle forward: name -> query -> color -> name
		h.FilterFormStep = (h.FilterFormStep + 1) % 3
		h.FilterNameInput.Blur()
		h.FilterQueryInput.Blur()
		if h.FilterFormStep == 0 {
			h.FilterNameInput.Focus()
			return textinput.Blink
		} else if h.FilterFormStep == 1 {
			h.FilterQueryInput.Focus()
			return textinput.Blink
		}
		return nil
	case "shift+tab":
		// Cycle backward
		h.FilterFormStep = (h.FilterFormStep + 2) % 3 // +2 is same as -1 mod 3
		h.FilterNameInput.Blur()
		h.FilterQueryInput.Blur()
		if h.FilterFormStep == 0 {
			h.FilterNameInput.Focus()
			return textinput.Blink
		} else if h.FilterFormStep == 1 {
			h.FilterQueryInput.Focus()
			return textinput.Blink
		}
		return nil
	case "j", "down":
		// Color selection navigation
		if h.FilterFormStep == 2 {
			if h.ColorCursor < len(h.AvailableColors)-1 {
				h.ColorCursor++
			}
			h.SelectedColor = h.AvailableColors[h.ColorCursor]
			return nil
		}
	case "k", "up":
		if h.FilterFormStep == 2 {
			if h.ColorCursor > 0 {
				h.ColorCursor--
			}
			h.SelectedColor = h.AvailableColors[h.ColorCursor]
			return nil
		}
	}

	// Update active input (only for text steps)
	var cmd tea.Cmd
	if h.FilterFormStep == 0 {
		h.FilterNameInput, cmd = h.FilterNameInput.Update(msg)
	} else if h.FilterFormStep == 1 {
		h.FilterQueryInput, cmd = h.FilterQueryInput.Update(msg)
	}
	return cmd
}

func (h *Handler) submitFilterForm() tea.Cmd {
	name := h.FilterNameInput.Value()
	query := h.FilterQueryInput.Value()
	color := h.SelectedColor

	h.IsCreatingFilter = false
	h.IsEditingFilter = false
	h.Loading = true
	h.StatusMsg = "Creating filter..."

	return func() tea.Msg {
		filter, err := h.Client.CreateFilter(name, query, color)
		if err != nil {
			return errMsg{err}
		}
		return filterCreatedMsg{filter: filter}
	}
}

type filterCreatedMsg struct {
	filter *api.Filter
}

type filterDeletedMsg struct {
	filterID string
}

// handleDeleteFilterConfirm handles y/n confirmation
func (h *Handler) handleDeleteFilterConfirmKeyMsg(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		h.ConfirmDeleteFilter = false
		if h.EditingFilter == nil {
			return nil
		}
		filterID := h.EditingFilter.ID

		// Delete via API
		h.Loading = true
		h.StatusMsg = "Deleting filter..."
		return func() tea.Msg {
			err := h.Client.DeleteFilter(filterID)
			if err != nil {
				return errMsg{err}
			}
			return filterDeletedMsg{filterID: filterID}
		}
	case "n", "N", "esc":
		h.ConfirmDeleteFilter = false
		h.EditingFilter = nil
		return nil
	}
	return nil
}
