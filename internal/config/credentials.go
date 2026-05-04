// Package config handles loading and saving application configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "todoist-tui"
	keyringUser    = "api-token"
	tokenFileName  = ".token"
)

// tokenDirFunc is overridable in tests to control where the token file lives.
var tokenDirFunc = ConfigDir

func tokenFilePath() (string, error) {
	dir, err := tokenDirFunc()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, tokenFileName), nil
}

func readTokenFromFile() (string, error) {
	path, err := tokenFilePath()
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func writeTokenToFile(token string) error {
	path, err := tokenFilePath()
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(token), 0600)
}

// GetToken retrieves the API token from available sources.
// Priority: 1. TODOIST_TOKEN env var, 2. System keyring, 3. File storage.
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

	// 3. Fall back to file storage if keyring is unavailable
	// (covers: no keyring daemon, unsupported platform, not found, etc.)
	token, fileErr := readTokenFromFile()
	if fileErr == nil {
		return token, nil
	}

	// If keyring error was "not found" and file also had no error above (no file), return empty
	if errors.Is(err, keyring.ErrNotFound) || errors.Is(err, keyring.ErrUnsupportedPlatform) {
		return "", nil
	}

	return "", fmt.Errorf("failed to read token: keyring: %w; file: %w", err, fileErr)
}

// SaveToken stores the API token.
// Stores in the system keyring, falling back to a file if unavailable.
func SaveToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	if err := keyring.Set(keyringService, keyringUser, token); err != nil {
		if writeErr := writeTokenToFile(token); writeErr != nil {
			return fmt.Errorf("failed to save token: keyring: %w; file: %w", err, writeErr)
		}
	}

	return nil
}

// HasToken returns true if a token is available from any source.
func HasToken() bool {
	token, _ := GetToken()
	return token != ""
}
