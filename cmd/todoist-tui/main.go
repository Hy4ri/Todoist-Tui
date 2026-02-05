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
	"github.com/hy4ri/todoist-tui/internal/auth"
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

    To get started:
    1. Run 'todoist-tui --init' to create a config template
    2. Get your API token from: https://app.todoist.com/app/settings/integrations/developer
    3. Add your token to the config file
    4. Run 'todoist-tui'

For more information, see: https://github.com/hy4ri/todoist-tui
`

const configTemplate = `# Todoist TUI Configuration
# Location: ~/.config/todoist-tui/config.yaml

auth:
  # Option 1: API Token (recommended for personal use)
  # Get your token from: https://app.todoist.com/app/settings/integrations/developer
  api_token: ""

  # Option 2: OAuth2 (for apps/integrations)
  # Create an app at: https://developer.todoist.com/appconsole.html
  # client_id: ""
  # client_secret: ""

ui:
  # Enable Vim-style keybindings (default: true)
  vim_mode: true

  # Theme configuration (uncomment to override defaults)
  # theme:
  #   # Core colors
  #   highlight: "#874BFD"
  #   subtle: "#666666"
  #   error: "#FF0000"
  #   success: "#00AA00"
  #   warning: "#FFAA00"
  #
  #   # Priority colors
  #   priority_1: "#D0473D"
  #   priority_2: "#EA8811"
  #   priority_3: "#296FDF"
  #
  #   # Task colors
  #   task_selected_bg: "#2A2A2A"
  #   task_recurring: "#00CCCC"
  #
  #   # Calendar colors
  #   calendar_selected_bg: "#874BFD" # Defaults to highlight
  #   calendar_selected_fg: "#FFFFFF"
  #
  #   # Tab colors
  #   tab_active_bg: "#874BFD"        # Defaults to highlight
  #   tab_active_fg: "#FFFFFF"
  #
  #   # Status bar colors
  #   status_bar_bg: "#1F1F1F"
  #   status_bar_fg: "#DDDDDD"
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
	if err := os.WriteFile(path, []byte(configTemplate), 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("Config file created: %s\n\n", path)
	fmt.Println("Next steps:")
	fmt.Println("  1. Get your API token from: https://app.todoist.com/app/settings/integrations/developer")
	fmt.Println("  2. Edit the config file and add your api_token")
	fmt.Println("  3. Run 'todoist-tui' to start")

	return nil
}

// runApp starts the main TUI application.
func runApp(initialView string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply user theme
	styles.InitTheme(&cfg.UI.Theme)

	// Check if config exists and has auth
	if !cfg.HasValidAuth() && !cfg.HasOAuthCredentials() {
		path, _ := config.ConfigPath()
		fmt.Println("No authentication configured.")
		fmt.Println()
		fmt.Println("To get started:")
		fmt.Printf("  1. Run 'todoist-tui --init' to create a config file at:\n     %s\n", path)
		fmt.Println("  2. Add your API token to the config file")
		fmt.Println("  3. Run 'todoist-tui' again")
		fmt.Println()
		fmt.Println("Get your API token from:")
		fmt.Println("  https://app.todoist.com/app/settings/integrations/developer")
		return nil
	}

	// Get access token (OAuth or API token)
	token, err := auth.GetAccessToken(cfg)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
	}

	// Update config with new token if OAuth was used
	if token != cfg.Auth.APIToken && token != cfg.Auth.AccessToken {
		cfg.Auth.AccessToken = token
		if err := config.Save(cfg); err != nil {
			// Non-fatal: just warn
			fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
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

// runJSONOutput fetches today's and overdue tasks and outputs them as JSON.
func runJSONOutput() error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if config exists and has auth
	if !cfg.HasValidAuth() && !cfg.HasOAuthCredentials() {
		return fmt.Errorf("no authentication configured. Run 'todoist-tui --init' first")
	}

	// Get access token
	token, err := auth.GetAccessToken(cfg)
	if err != nil {
		return fmt.Errorf("failed to authenticate: %w", err)
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
