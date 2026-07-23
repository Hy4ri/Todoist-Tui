// Package config handles loading and saving application configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration.
type Config struct {
	UI UIConfig `yaml:"ui"`
}

// UIConfig holds UI-related settings.
type UIConfig struct {
	VimMode             bool        `yaml:"vim_mode"`
	DefaultView         string      `yaml:"default_view,omitempty"`          // "inbox", "today", "upcoming", "projects", "calendar"
	CalendarDefaultView string      `yaml:"calendar_default_view,omitempty"` // "compact" or "expanded"
	Theme               ThemeConfig `yaml:"theme,omitempty"`
	// Keybindings allows overriding default key bindings. Map of action name to key string.
	// Example: { "add_task": "o", "complete": "c" }
	// Action names match those in KeymapData (snake_case). An empty or missing map keeps all defaults.
	Keybindings map[string]string `yaml:"keybindings,omitempty"`
}

// ThemeConfig holds color theme settings.
// All colors should be hex strings (e.g., "#FF6B6B").
// Empty values use built-in defaults.
type ThemeConfig struct {
	// Core colors
	Highlight string `yaml:"highlight,omitempty"` // Accent color (default: #874BFD)
	Subtle    string `yaml:"subtle,omitempty"`    // Muted text (default: #666666)
	Error     string `yaml:"error,omitempty"`     // Error red (default: #FF0000)
	Success   string `yaml:"success,omitempty"`   // Success green (default: #00AA00)
	Warning   string `yaml:"warning,omitempty"`   // Warning orange (default: #FFAA00)

	// Priority colors
	Priority1 string `yaml:"priority_1,omitempty"` // P1 red (default: #D0473D)
	Priority2 string `yaml:"priority_2,omitempty"` // P2 orange (default: #EA8811)
	Priority3 string `yaml:"priority_3,omitempty"` // P3 blue (default: #296FDF)

	// Task colors
	TaskSelectedBg string `yaml:"task_selected_bg,omitempty"` // Selected task background
	TaskRecurring  string `yaml:"task_recurring,omitempty"`   // Recurring indicator

	// Calendar colors
	CalendarSelectedBg string `yaml:"calendar_selected_bg,omitempty"` // Selected day background
	CalendarSelectedFg string `yaml:"calendar_selected_fg,omitempty"` // Selected day text

	// Tab colors
	TabActiveBg string `yaml:"tab_active_bg,omitempty"` // Active tab background
	TabActiveFg string `yaml:"tab_active_fg,omitempty"` // Active tab text

	// Status bar colors
	StatusBarBg string `yaml:"status_bar_bg,omitempty"` // Status bar background
	StatusBarFg string `yaml:"status_bar_fg,omitempty"` // Status bar text
}

// DefaultConfig returns a new Config with default values.
func DefaultConfig() *Config {
	return &Config{
		UI: UIConfig{
			VimMode: true,
		},
	}
}

// ConfigDir returns the path to the configuration directory.
// Creates the directory if it doesn't exist.
func ConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "todoist-tui")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return configDir, nil
}

// ConfigPath returns the full path to the configuration file.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

// Load reads the configuration from the config file.
// If the file doesn't exist, returns a default configuration.
func Load() (*Config, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// Save writes the configuration to the config file.
func Save(cfg *Config) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	// Write with restricted permissions (owner read/write only)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// HasValidAuth returns true if authentication is available.
// Checks secure credential storage and OAuth access token.
func (c *Config) HasValidAuth() bool {
	return HasToken()
}

// UpdateDefaultView updates the default_view setting in the config file
// using structural YAML updates to avoid clobbering unrelated ui settings.
func UpdateDefaultView(viewName string) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.UI.DefaultView = viewName
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}
