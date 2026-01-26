// Package main is the entry point for the Todoist TUI application.
package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/api"
	"github.com/hy4ri/todoist-tui/internal/auth"
	"github.com/hy4ri/todoist-tui/internal/config"
	"github.com/hy4ri/todoist-tui/internal/tui"
)

const version = "0.1.0"

const helpText = `todoist-tui - Terminal-based Todoist client with Vim keybindings

USAGE:
    todoist-tui [OPTIONS]

OPTIONS:
    -h, --help      Show this help message
    -v, --version   Show version information
    --init          Create a template config file

CONFIGURATION:
    Config file: ~/.config/todoist-tui/config.yaml

    To get started:
    1. Run 'todoist-tui --init' to create a config template
    2. Get your API token from: https://app.todoist.com/app/settings/integrations/developer
    3. Add your token to the config file
    4. Run 'todoist-tui'

KEYBINDINGS:
    Navigation:
        j/k         Move down/up
        gg/G        Go to top/bottom
        Ctrl+d/u    Half page down/up
        Tab         Switch between sidebar and tasks
        Enter       Select / Open details
        Esc         Go back

    Task Actions:
        a           Add new task
        e           Edit selected task
        x           Complete/uncomplete task
        dd          Delete task
        1-4         Set priority (1=highest)

    Other:
        /           Search tasks
        r           Refresh
        ?           Show help
        q           Quit

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
