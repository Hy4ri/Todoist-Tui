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
	Auth AuthConfig `yaml:"auth"`
	UI   UIConfig   `yaml:"ui"`
}

// AuthConfig holds authentication-related settings.
type AuthConfig struct {
	// APIToken is the personal API token (for simple auth without OAuth)
	APIToken string `yaml:"api_token,omitempty"`

	// OAuth2 credentials
	ClientID     string `yaml:"client_id,omitempty"`
	ClientSecret string `yaml:"client_secret,omitempty"`

	// OAuth2 tokens (obtained after successful auth)
	AccessToken  string `yaml:"access_token,omitempty"`
	RefreshToken string `yaml:"refresh_token,omitempty"`
}

// UIConfig holds UI-related settings.
type UIConfig struct {
	VimMode             bool   `yaml:"vim_mode"`
	CalendarDefaultView string `yaml:"calendar_default_view,omitempty"` // "compact" or "expanded"
}

// DefaultConfig returns a new Config with default values.
func DefaultConfig() *Config {
	return &Config{
		Auth: AuthConfig{},
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

// HasValidAuth returns true if the config has valid authentication credentials.
func (c *Config) HasValidAuth() bool {
	return c.Auth.APIToken != "" || c.Auth.AccessToken != ""
}

// HasOAuthCredentials returns true if OAuth client credentials are configured.
func (c *Config) HasOAuthCredentials() bool {
	return c.Auth.ClientID != "" && c.Auth.ClientSecret != ""
}

// GetToken returns the best available token for API authentication.
// Prefers AccessToken (from OAuth) over APIToken.
func (c *Config) GetToken() string {
	if c.Auth.AccessToken != "" {
		return c.Auth.AccessToken
	}
	return c.Auth.APIToken
}
