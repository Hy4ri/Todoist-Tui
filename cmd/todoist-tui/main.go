// Package main is the entry point for the Todoist TUI application.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/config"
	"github.com/hy4ri/todoist-tui/internal/tui"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

const version = "0.9.9"

const helpText = `todoist-tui - Terminal-based Todoist client with Vim keybindings

USAGE:
    todoist-tui [OPTIONS]

OPTIONS:
    -h, --help      Show this help message
    -v, --version   Show version information
    --init          Create a template config file
    --json          Output today's and overdue tasks in JSON format

CONFIGURATION:
    Config file: ~/.config/todoist-tui/config.yaml

For more information, see: https://github.com/hy4ri/todoist-tui
`

const configTemplate = `# Todoist TUI Configuration
# Location: ~/.config/todoist-tui/config.yaml

ui:
  # Enable Vim-style keybindings (default: true)
  vim_mode: true

  # Default view on startup (projects, upcoming, calendar, labels, inbox)
  default_view: "inbox"

  # Calendar default view (compact, expanded)
  calendar_default_view: "compact"

  # Theme configuration (uncomment to override defaults)
  theme:
     # Core colors
     highlight: "#990000"
     subtle: "#666666"
     error: "#FF0000"
     success: "#00AA00"
     warning: "#FFAA00"
  
     # Priority colors
     priority_1: "#D0473D"
     priority_2: "#EA8811"
     priority_3: "#296FDF"
  
     # Task colors
     task_selected_bg: "#2A2A2A"
     task_recurring: "#00CCCC"
  
     # Calendar colors
     calendar_selected_bg: "#990000" 
     calendar_selected_fg: "#FFFFFF"
  
     # Tab colors
     tab_active_bg: "#990000"        
     tab_active_fg: "#FFFFFF"
  
     # Status bar colors
     status_bar_bg: "#1F1F1F"
     status_bar_fg: "#DDDDDD"
`

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Define flags
	var (
		showHelp     bool
		showVersion  bool
		initConfig   bool
		viewProjects bool
		viewUpcoming bool
		viewCalendar bool
		viewLabels   bool
		viewToday    bool
		viewInbox    bool
		outputJSON   bool
	)

	flag.BoolVar(&showHelp, "help", false, "Show help message")
	flag.BoolVar(&showHelp, "h", false, "Show help message (shorthand)")
	flag.BoolVar(&showVersion, "version", false, "Show version")
	flag.BoolVar(&showVersion, "v", false, "Show version (shorthand)")
	flag.BoolVar(&initConfig, "init", false, "Create template config file")
	flag.BoolVar(&viewProjects, "projects", false, "Start in projects view")
	flag.BoolVar(&viewUpcoming, "upcoming", false, "Start in upcoming view")
	flag.BoolVar(&viewCalendar, "calendar", false, "Start in calendar view")
	flag.BoolVar(&viewLabels, "labels", false, "Start in labels view")
	flag.BoolVar(&viewInbox, "inbox", false, "Start in inbox view")
	flag.BoolVar(&viewToday, "today", false, "Start in today view")
	flag.BoolVar(&outputJSON, "json", false, "Output tasks in JSON format")

	flag.Usage = func() {
		fmt.Print(helpText)
	}

	flag.Parse()

	// Handle flags
	if showHelp {
		fmt.Print(helpText)
		return nil
	}

	if showVersion {
		fmt.Printf("todoist-tui version %s\n", version)
		return nil
	}

	if initConfig {
		return createConfigTemplate()
	}

	if outputJSON {
		return runJSONOutput()
	}

	// determine initial view
	initialView := ""
	if viewProjects {
		initialView = "projects"
	} else if viewUpcoming {
		initialView = "upcoming"
	} else if viewCalendar {
		initialView = "calendar"
	} else if viewLabels {
		initialView = "labels"
	} else if viewInbox {
		initialView = "inbox"
	} else if viewToday {
		initialView = "today"
	}

	// Normal application flow
	return runApp(initialView)
}

// createConfigTemplate creates a template configuration file.
func createConfigTemplate() error {
	path, err := config.ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	// Check if config already exists
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Config file already exists: %s\n", path)
		fmt.Print("Overwrite? [y/N]: ")

		var response string
		fmt.Scanln(&response)

		if response != "y" && response != "Y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	// Ensure directory exists
	if _, err := config.ConfigDir(); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Write template
	if err := os.WriteFile(path, []byte(configTemplate), 0o600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Config file created: %s\n\n", path)

	return nil
}

// runApp starts the main TUI application.
func runApp(initialView string) error {
	// Ensure config exists (auto-create if needed)
	if err := ensureConfig(); err != nil {
		return err
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply user theme
	styles.InitTheme(&cfg.UI.Theme)

	// Get token from secure storage
	token, err := config.GetToken()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}

	// If no token, prompt for it
	if token == "" {
		token, err = promptForToken()
		if err != nil {
			return err
		}
		if token == "" {
			return fmt.Errorf("no token provided")
		}
		// Save the token securely
		if err := config.SaveToken(token); err != nil {
			return fmt.Errorf("failed to save token: %w", err)
		}
	}

	// Create API client
	client := api.NewClient(token)

	// Create and run TUI
	app := tui.NewApp(client, cfg, initialView)
	p := tea.NewProgram(app, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("failed to run TUI: %w", err)
	}

	return nil
}

// ensureConfig creates a default config file if it doesn't exist.
func ensureConfig() error {
	path, err := config.ConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Create config directory
		if _, err := config.ConfigDir(); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		// Write default config
		if err := os.WriteFile(path, []byte(configTemplate), 0o600); err != nil {
			return fmt.Errorf("failed to create config file: %w", err)
		}
		fmt.Printf("Created config file: %s\n", path)
	}

	return nil
}

// promptForToken shows a TUI prompt for the user to enter their API token.
func promptForToken() (string, error) {
	fmt.Println("┌─────────────────────────────────────────────────────────────┐")
	fmt.Println("│  Todoist TUI - First Time Setup                             │")
	fmt.Println("└─────────────────────────────────────────────────────────────┘")
	fmt.Println()
	fmt.Println("Get your API token from:")
	fmt.Println("  https://app.todoist.com/app/settings/integrations/developer")
	fmt.Println()
	fmt.Print("Enter your API token: ")

	var token string
	_, err := fmt.Scanln(&token)
	if err != nil {
		return "", nil // User cancelled
	}

	return strings.TrimSpace(token), nil
}

// runJSONOutput fetches today's and overdue tasks and outputs them as JSON.
func runJSONOutput() error {
	// Get token from secure storage
	token, err := config.GetToken()
	if err != nil {
		return fmt.Errorf("failed to get token: %w", err)
	}
	if token == "" {
		return fmt.Errorf("no token configured. Run 'todoist-tui' first to set up")
	}

	// Create API client
	client := api.NewClient(token)

	// Fetch tasks
	tasks, err := client.GetTasksByFilter("today | overdue")
	if err != nil {
		return fmt.Errorf("failed to fetch tasks: %w", err)
	}

	// Simplified output structure
	type simpleTask struct {
		Name     string `json:"name"`
		Date     string `json:"date"`
		Priority int    `json:"priority"`
	}

	type jsonOutput struct {
		// Waybar specific fields
		Text    string `json:"text"`    // What shows on the bar
		Tooltip string `json:"tooltip"` // What shows on hover
		Class   string `json:"class"`   // For CSS styling

		// Data fields
		Tasks          []simpleTask `json:"tasks"`
		PriorityCounts map[int]int  `json:"priority_counts"`
		Total          int          `json:"total"`
	}

	res := jsonOutput{
		Tasks:          make([]simpleTask, 0, len(tasks)),
		PriorityCounts: make(map[int]int),
		Total:          len(tasks),
	}

	var tooltipLines []string
	maxPriority := 1 // Todoist P1 is 1 (natural), P4 is 4 (urgent)

	for _, t := range tasks {
		date := ""
		if t.Due != nil {
			date = t.Due.Date
		}

		res.Tasks = append(res.Tasks, simpleTask{
			Name:     t.Content,
			Date:     date,
			Priority: t.Priority,
		})

		res.PriorityCounts[t.Priority]++

		// Track highest priority for styling
		if t.Priority > maxPriority {
			maxPriority = t.Priority
		}

		// Build a nice tooltip line for each task with priority
		tooltipLines = append(tooltipLines, fmt.Sprintf("P%d: %s", 5-t.Priority, t.Content))
	}

	// Map the logic to Waybar fields
	res.Text = fmt.Sprintf("%d", res.Total)
	res.Tooltip = strings.Join(tooltipLines, "\n")
	res.Class = fmt.Sprintf("p%d", maxPriority)

	// Output as single-line JSON (better for Waybar exec)
	output, err := json.Marshal(res)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(output))
	return nil
}
