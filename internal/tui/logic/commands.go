package logic

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// CommandHandlerFunc handles a command execution.
type CommandHandlerFunc func(h *Handler, args []string) tea.Cmd

// CommandDef defines a command.
type CommandDef struct {
	Name        string
	Aliases     []string
	Description string
	Handler     CommandHandlerFunc
}

// CommandRegistry holds all available commands.
var CommandRegistry = map[string]CommandDef{}

// initCommands needs to be called to register commands.
// Using init() function or explicit call? Explicit is better context-wise but init is easier.
// Let's use a RegisterCommands method on Handler for access to state?
// No, definitions can be static, handler receives *Handler.

func init() {
	registerCommands()
}

func registerCommands() {
	commands := []CommandDef{
		{
			Name:        "goto",
			Aliases:     []string{"g", "view"},
			Description: "Go to a specific view (inbox, today, upcoming, projects, labels, calendar)",
			Handler:     handleGoto,
		},
		{
			Name:        "add",
			Aliases:     []string{"a", "new"},
			Description: "Add a new task",
			Handler:     handleAddCommand,
		},
		{
			Name:        "delete",
			Aliases:     []string{"d", "del", "rm"},
			Description: "Delete the selected task",
			Handler:     handleDeleteCommand,
		},
		{
			Name:        "complete",
			Aliases:     []string{"c", "done"},
			Description: "Complete the selected task",
			Handler:     handleCompleteCommand,
		},
		{
			Name:        "project",
			Aliases:     []string{"p", "prj"},
			Description: "Switch to a project by name",
			Handler:     handleProjectCommand,
		},
		{
			Name:        "label",
			Aliases:     []string{"l", "lbl"},
			Description: "Filter by label",
			Handler:     handleLabelCommand,
		},
		{
			Name:        "refresh",
			Aliases:     []string{"r", "reload"},
			Description: "Refresh data from API",
			Handler:     handleRefreshCommand,
		},
		{
			Name:        "quit",
			Aliases:     []string{"q", "exit"},
			Description: "Quit application",
			Handler:     handleQuitCommand,
		},
		{
			Name:        "help",
			Aliases:     []string{"h", "?"},
			Description: "Show help",
			Handler:     handleHelpCommand,
		},
		{
			Name:        "sort",
			Aliases:     []string{"s"},
			Description: "Sort tasks (priority, date)",
			Handler:     handleSortCommand,
		},
		{
			Name:        "filter",
			Aliases:     []string{"f"},
			Description: "Run a filter query",
			Handler:     handleFilterCommand,
		},
		{
			Name:        "commands",
			Aliases:     []string{"list", "ls"},
			Description: "List all available commands",
			Handler:     handleCommandsCommand,
		},
	}

	for _, cmd := range commands {
		CommandRegistry[cmd.Name] = cmd
		for _, alias := range cmd.Aliases {
			CommandRegistry[alias] = cmd
		}
	}
}

// Core Handlers

func handleGoto(h *Handler, args []string) tea.Cmd {
	if len(args) == 0 {
		h.StatusMsg = "Usage: :goto <view>"
		return nil
	}

	target := strings.ToLower(args[0])
	switch target {
	case "inbox", "i":
		return h.switchToTab(state.TabInbox)
	case "today", "t":
		return h.switchToTab(state.TabToday)
	case "upcoming", "u":
		return h.switchToTab(state.TabUpcoming)
	case "projects", "p":
		return h.switchToTab(state.TabProjects)
	case "labels", "l":
		return h.switchToTab(state.TabLabels)
	case "calendar", "c", "cal":
		return h.switchToTab(state.TabCalendar)
	default:
		h.StatusMsg = fmt.Sprintf("Unknown view: %s", target)
		return nil
	}
}

func handleAddCommand(h *Handler, args []string) tea.Cmd {
	if len(args) == 0 {
		return h.handleAdd()
	}

	// Quick add with arguments
	content := strings.Join(args, " ")
	h.StatusMsg = "Adding task..."
	return func() tea.Msg {
		_, err := h.Client.QuickAddTask(content)
		if err != nil {
			return errMsg{err}
		}
		return quickAddTaskCreatedMsg{}
	}
}

func handleDeleteCommand(h *Handler, args []string) tea.Cmd {
	return h.handleDelete()
}

func handleCompleteCommand(h *Handler, args []string) tea.Cmd {
	return h.handleComplete()
}

func handleProjectCommand(h *Handler, args []string) tea.Cmd {
	if len(args) == 0 {
		return h.switchToTab(state.TabProjects)
	}

	query := strings.ToLower(strings.Join(args, " "))

	// Switch to projects tab first if not already
	if h.CurrentTab != state.TabProjects {
		h.switchToTab(state.TabProjects)
	}

	// Find best matching project
	var bestMatch *api.Project
	for i := range h.Projects {
		if strings.EqualFold(h.Projects[i].Name, query) {
			bestMatch = &h.Projects[i]
			break
		}
		// Partial match fallback? Maybe later
	}

	if bestMatch != nil {
		h.CurrentProject = bestMatch
		h.FocusedPane = state.PaneMain
		h.Sections = nil
		// We need to trigger load, switching tab sets view to Project but doesn't load specific project
		return h.loadProjectTasks(bestMatch.ID)
	}

	h.StatusMsg = fmt.Sprintf("Project not found: %s", query)
	return nil
}

func handleLabelCommand(h *Handler, args []string) tea.Cmd {
	if len(args) == 0 {
		return h.switchToTab(state.TabLabels)
	}

	query := strings.ToLower(strings.Join(args, " "))

	if h.CurrentTab != state.TabLabels {
		h.switchToTab(state.TabLabels)
	}

	var bestMatch *api.Label
	for i := range h.Labels {
		if strings.EqualFold(h.Labels[i].Name, query) {
			bestMatch = &h.Labels[i]
			break
		}
	}

	if bestMatch != nil {
		h.CurrentLabel = bestMatch
		return h.loadLabelTasks(bestMatch.Name)
	}

	h.StatusMsg = fmt.Sprintf("Label not found: %s", query)
	return nil
}

func handleRefreshCommand(h *Handler, args []string) tea.Cmd {
	h.LastDataFetch = time.Time{} // Reset cache timer
	return h.handleRefresh(true)
}

func handleQuitCommand(h *Handler, args []string) tea.Cmd {
	return tea.Quit
}

func handleHelpCommand(h *Handler, args []string) tea.Cmd {
	h.PreviousView = h.CurrentView
	h.CurrentView = state.ViewHelp
	return nil
}

func handleFilterCommand(h *Handler, args []string) tea.Cmd {
	if len(args) == 0 {
		return h.switchToTab(state.TabFilters)
	}

	query := strings.Join(args, " ")

	if h.CurrentTab != state.TabFilters {
		h.switchToTab(state.TabFilters)
	}

	return h.runAdHocFilter(query)
}

func handleCommandsCommand(h *Handler, args []string) tea.Cmd {
	uniqueCmds := make(map[string]bool)
	var names []string

	// Collect unique command names
	for _, cmd := range CommandRegistry {
		if !uniqueCmds[cmd.Name] {
			uniqueCmds[cmd.Name] = true
			names = append(names, cmd.Name)
		}
	}

	sort.Strings(names)
	h.StatusMsg = "Commands: " + strings.Join(names, ", ")
	return nil
}

func handleSortCommand(h *Handler, args []string) tea.Cmd {
	if len(args) == 0 {
		h.StatusMsg = "Usage: :sort <priority|date>"
		return nil
	}

	// This is a bit tricky since sort logic is currently hardcoded in sortTasks()
	// To implement custom sorting, we'd need to modify sortTasks() to respect a sort mode in state.
	// For now, let's just trigger a re-sort (which sorts by date/priority)
	h.TasksSorted = false
	h.sortTasks()
	h.StatusMsg = "Tasks sorted"
	return nil
}

// Helpers

func (h *Handler) executeCommand(input string) tea.Cmd {
	h.CommandLine.Active = false
	h.CommandLine.Input.Reset()

	// Add to history
	if len(h.CommandLine.History) == 0 || h.CommandLine.History[len(h.CommandLine.History)-1] != input {
		h.CommandLine.History = append(h.CommandLine.History, input)
	}
	h.CommandLine.HistoryCursor = -1

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	cmdName := strings.ToLower(parts[0])
	args := parts[1:]

	if cmdDef, ok := CommandRegistry[cmdName]; ok {
		return cmdDef.Handler(h, args)
	}

	h.StatusMsg = fmt.Sprintf("Unknown command: %s", cmdName)
	return nil
}

func (h *Handler) autocompleteCommand() tea.Cmd {
	input := h.CommandLine.Input.Value()

	// Simple autocomplete: find command starting with input
	if input == "" {
		return nil
	}

	var suggestions []string
	for name := range CommandRegistry {
		if strings.HasPrefix(name, input) {
			suggestions = append(suggestions, name)
		}
	}

	// Logic for cycling through suggestions could go here
	// For now, just pick the first one
	if len(suggestions) > 0 {
		// Use the shortest match or just first?
		// Prefer full match if available
		h.CommandLine.Input.SetValue(suggestions[0] + " ")
		h.CommandLine.Input.SetCursor(len(suggestions[0]) + 1)
	} else {
		// Try autocompleting arguments if we have a command
		parts := strings.Fields(input)
		if len(parts) >= 1 {
			// cmdName := parts[0]
			// autocomplete args...
			// TODO: Add arg completion for projects/labels based on cmdName
		}
	}

	return nil
}

func (h *Handler) commandHistoryPrev() tea.Cmd {
	if len(h.CommandLine.History) == 0 {
		return nil
	}

	if h.CommandLine.HistoryCursor == -1 {
		h.CommandLine.HistoryCursor = len(h.CommandLine.History) - 1
	} else if h.CommandLine.HistoryCursor > 0 {
		h.CommandLine.HistoryCursor--
	}

	if h.CommandLine.HistoryCursor >= 0 && h.CommandLine.HistoryCursor < len(h.CommandLine.History) {
		h.CommandLine.Input.SetValue(h.CommandLine.History[h.CommandLine.HistoryCursor])
		h.CommandLine.Input.SetCursor(len(h.CommandLine.Input.Value()))
	}
	return nil
}

func (h *Handler) commandHistoryNext() tea.Cmd {
	if len(h.CommandLine.History) == 0 || h.CommandLine.HistoryCursor == -1 {
		return nil
	}

	if h.CommandLine.HistoryCursor < len(h.CommandLine.History)-1 {
		h.CommandLine.HistoryCursor++
		h.CommandLine.Input.SetValue(h.CommandLine.History[h.CommandLine.HistoryCursor])
		h.CommandLine.Input.SetCursor(len(h.CommandLine.Input.Value()))
	} else {
		h.CommandLine.HistoryCursor = -1
		h.CommandLine.Input.SetValue("")
	}
	return nil
}

func (h *Handler) updateSuggestions() tea.Cmd {
	input := h.CommandLine.Input.Value()
	if input == "" {
		h.CommandLine.Suggestions = nil
		return nil
	}

	parts := strings.Fields(input)
	if len(parts) == 0 {
		h.CommandLine.Suggestions = nil
		return nil
	}

	// If typing command name
	if len(parts) == 1 && !strings.HasSuffix(input, " ") {
		var matches []string
		for name := range CommandRegistry {
			if strings.HasPrefix(name, input) {
				matches = append(matches, name)
			}
		}
		// Limit suggestions
		if len(matches) > 5 {
			matches = matches[:5]
		}
		h.CommandLine.Suggestions = matches
		return nil
	}

	// TODO: Arg suggestions
	h.CommandLine.Suggestions = nil
	return nil
}
