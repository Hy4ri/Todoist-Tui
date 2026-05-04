// Package config handles loading and saving application configuration.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "todoist-tui"
	keyringUser    = "api-token"
)

// GetToken retrieves the API token from available sources.
// Priority: 1. TODOIST_TOKEN env var, 2. System keyring.
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

	if err == keyring.ErrNotFound || err == keyring.ErrUnsupportedPlatform {
		return "", nil
	}

	return "", fmt.Errorf("failed to read token from keyring: %w", err)
}

// SaveToken stores the API token securely.
// Stores the token in the system keyring.
func SaveToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	if err := keyring.Set(keyringService, keyringUser, token); err != nil {
		return fmt.Errorf("failed to save token to keyring: %w", err)
	}

	return nil
}

// HasToken returns true if a token is available from any source.
func HasToken() bool {
	token, _ := GetToken()
	return token != ""
}
