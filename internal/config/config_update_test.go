package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestUpdateDefaultViewPreservesUiFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	original := []byte("ui:\n  theme:\n    highlight: \"#123456\"\n  keybindings:\n    add_task: \"o\"\n")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := UpdateDefaultView("today"); err != nil {
		t.Fatalf("UpdateDefaultView() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}

	if cfg.UI.DefaultView != "today" {
		t.Fatalf("DefaultView = %q, want %q", cfg.UI.DefaultView, "today")
	}
	if cfg.UI.Theme.Highlight != "#123456" {
		t.Fatalf("Theme.Highlight = %q, want %q", cfg.UI.Theme.Highlight, "#123456")
	}
	if got := cfg.UI.Keybindings["add_task"]; got != "o" {
		t.Fatalf("Keybindings[add_task] = %q, want %q", got, "o")
	}
}
