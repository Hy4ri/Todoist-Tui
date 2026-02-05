package state

import "github.com/charmbracelet/bubbles/textinput"

// CommandLine holds the state for the vim-style command line.
type CommandLine struct {
	Input            textinput.Model
	Active           bool
	History          []string
	HistoryCursor    int
	Suggestions      []string
	SuggestionCursor int
}

// NewCommandLine initializes a new CommandLine state.
func NewCommandLine() *CommandLine {
	input := textinput.New()
	input.Prompt = "" // Prompt is rendered externally
	input.Placeholder = ""
	input.CharLimit = 100
	input.Width = 50

	return &CommandLine{
		Input:         input,
		Active:        false,
		History:       []string{},
		HistoryCursor: -1,
		Suggestions:   []string{},
	}
}
