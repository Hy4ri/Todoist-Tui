package config

import (
	"testing"

	"github.com/zalando/go-keyring"
)

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
