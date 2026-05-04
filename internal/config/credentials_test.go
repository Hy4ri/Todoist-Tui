package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/zalando/go-keyring"
)

// setupFileFallback redirects tokenDirFunc to a temp directory and cleans up on test end.
func setupFileFallback(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	old := tokenDirFunc
	tokenDirFunc = func() (string, error) { return dir, nil }
	t.Cleanup(func() { tokenDirFunc = old })
	return dir
}

func TestSaveAndGetTokenUsesKeyring(t *testing.T) {
	keyring.MockInit()
	t.Setenv("TODOIST_TOKEN", "")

	if err := SaveToken("  secret-token  "); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	got, err := GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if got != "secret-token" {
		t.Fatalf("GetToken() = %q, want %q", got, "secret-token")
	}
	if !HasToken() {
		t.Fatal("HasToken() = false, want true")
	}
}

func TestGetTokenPrefersEnv(t *testing.T) {
	keyring.MockInit()
	if err := keyring.Set(keyringService, keyringUser, "keyring-token"); err != nil {
		t.Fatalf("keyring.Set() error = %v", err)
	}
	t.Setenv("TODOIST_TOKEN", " env-token ")

	got, err := GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if got != "env-token" {
		t.Fatalf("GetToken() = %q, want %q", got, "env-token")
	}
}

func TestSaveTokenRejectsEmpty(t *testing.T) {
	keyring.MockInit()
	t.Setenv("TODOIST_TOKEN", "")

	if err := SaveToken("   "); err == nil {
		t.Fatal("SaveToken() error = nil, want non-nil")
	}
}

func TestGetTokenFallsBackToFileWhenKeyringFails(t *testing.T) {
	keyring.MockInitWithError(errors.New("dbus: name not activatable"))
	t.Setenv("TODOIST_TOKEN", "")
	dir := setupFileFallback(t)

	tokenPath := filepath.Join(dir, tokenFileName)
	if err := os.WriteFile(tokenPath, []byte("file-token"), 0600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	got, err := GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if got != "file-token" {
		t.Fatalf("GetToken() = %q, want %q", got, "file-token")
	}
}

func TestSaveTokenFallsBackToFileWhenKeyringFails(t *testing.T) {
	keyring.MockInitWithError(errors.New("dbus: name not activatable"))
	t.Setenv("TODOIST_TOKEN", "")
	dir := setupFileFallback(t)

	if err := SaveToken("my-token"); err != nil {
		t.Fatalf("SaveToken() error = %v", err)
	}

	tokenPath := filepath.Join(dir, tokenFileName)
	data, err := os.ReadFile(tokenPath)
	if err != nil {
		t.Fatalf("ReadFile error = %v", err)
	}
	if string(data) != "my-token" {
		t.Fatalf("token file content = %q, want %q", string(data), "my-token")
	}
}

func TestGetTokenReturnsEmptyWhenBothFail(t *testing.T) {
	keyring.MockInitWithError(errors.New("dbus: name not activatable"))
	t.Setenv("TODOIST_TOKEN", "")
	setupFileFallback(t)

	got, err := GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if got != "" {
		t.Fatalf("GetToken() = %q, want empty", got)
	}
}

func TestGetTokenPrefersEnvOverFile(t *testing.T) {
	keyring.MockInitWithError(errors.New("dbus error"))
	t.Setenv("TODOIST_TOKEN", " env-token ")
	dir := setupFileFallback(t)

	tokenPath := filepath.Join(dir, tokenFileName)
	if err := os.WriteFile(tokenPath, []byte("file-token"), 0600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	got, err := GetToken()
	if err != nil {
		t.Fatalf("GetToken() error = %v", err)
	}
	if got != "env-token" {
		t.Fatalf("GetToken() = %q, want %q", got, "env-token")
	}
}
