// Package config handles loading and saving application configuration.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "todoist-tui"
	keyringUser    = "api-token"
	credFileName   = ".credentials"
)

// DataDir returns the path to the data directory for secure storage.
// Uses XDG_DATA_HOME or defaults to ~/.local/share/todoist-tui/
func DataDir() (string, error) {
	// Check XDG_DATA_HOME first
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		dataHome = filepath.Join(homeDir, ".local", "share")
	}

	dataDir := filepath.Join(dataHome, "todoist-tui")
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	return dataDir, nil
}

// GetToken retrieves the API token from available sources.
// Priority: 1. TODOIST_TOKEN env var, 2. System keyring, 3. Credentials file
func GetToken() (string, error) {
	// 1. Check environment variable (highest priority, allows override)
	if token := os.Getenv("TODOIST_TOKEN"); token != "" {
		return strings.TrimSpace(token), nil
	}

	// 2. Try system keyring
	token, err := keyring.Get(keyringService, keyringUser)
	if err == nil && token != "" {
		return strings.TrimSpace(token), nil
	}

	// 3. Fall back to credentials file
	dataDir, err := DataDir()
	if err != nil {
		return "", err
	}

	credPath := filepath.Join(dataDir, credFileName)
	data, err := os.ReadFile(credPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // No token stored
		}
		return "", fmt.Errorf("failed to read credentials file: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// SaveToken stores the API token securely.
// Tries system keyring first, falls back to credentials file.
func SaveToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Try keyring first
	err := keyring.Set(keyringService, keyringUser, token)
	if err == nil {
		return nil
	}

	// Fall back to file storage
	dataDir, err := DataDir()
	if err != nil {
		return err
	}

	credPath := filepath.Join(dataDir, credFileName)
	if err := os.WriteFile(credPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("failed to write credentials file: %w", err)
	}

	return nil
}

// ClearToken removes the stored API token from all locations.
func ClearToken() error {
	// Try to delete from keyring (ignore errors)
	_ = keyring.Delete(keyringService, keyringUser)

	// Delete credentials file if it exists
	dataDir, err := DataDir()
	if err != nil {
		return err
	}

	credPath := filepath.Join(dataDir, credFileName)
	if err := os.Remove(credPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove credentials file: %w", err)
	}

	return nil
}

// HasToken returns true if a token is available from any source.
func HasToken() bool {
	token, _ := GetToken()
	return token != ""
}
