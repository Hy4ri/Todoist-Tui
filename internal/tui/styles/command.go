package styles

import "github.com/charmbracelet/lipgloss"

var (
	// CommandPrompt is the style for the ":" prompt.
	CommandPrompt = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFCC00")).
			Bold(true)

	// CommandInput is the style for the active command input text.
	CommandInput = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	// CommandSuggestion is the style for autocomplete suggestions.
	CommandSuggestion = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#888888"))

	// CommandSuggestionSelected is the style for the selected autocomplete suggestion.
	CommandSuggestionSelected = lipgloss.NewStyle().
					Foreground(lipgloss.Color("#FFFFFF")).
					Background(lipgloss.Color("#444444"))

	// CommandLineContainer is the container for the command line area.
	CommandLineContainer = lipgloss.NewStyle().
				Padding(0, 1)
)
