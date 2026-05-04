package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func withTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestConfigDirAndPath(t *testing.T) {
	home := withTempHome(t)

	dir, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir() error = %v", err)
	}
	wantDir := filepath.Join(home, ".config", "todoist-tui")
	if dir != wantDir {
		t.Fatalf("ConfigDir() = %q, want %q", dir, wantDir)
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}
	if path != filepath.Join(wantDir, "config.yaml") {
		t.Fatalf("ConfigPath() = %q, want %q", path, filepath.Join(wantDir, "config.yaml"))
	}
}

func TestLoadReturnsDefaultConfigWhenMissing(t *testing.T) {
	withTempHome(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg == nil || !cfg.UI.VimMode {
		t.Fatalf("Load() returned %#v, want default config with vim mode enabled", cfg)
	}
}

func TestSaveWritesConfigWithRestrictedPermissions(t *testing.T) {
	withTempHome(t)

	want := &Config{
		UI: UIConfig{
			VimMode:             false,
			DefaultView:         "projects",
			CalendarDefaultView: "expanded",
		},
	}

	if err := Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath() error = %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat() error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("file permissions = %v, want %v", got, 0o600)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	var got Config
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if got.UI.DefaultView != want.UI.DefaultView || got.UI.CalendarDefaultView != want.UI.CalendarDefaultView || got.UI.VimMode != want.UI.VimMode {
		t.Fatalf("saved config = %#v, want %#v", got, want)
	}
}

func TestLoadReturnsParseErrorForInvalidYAML(t *testing.T) {
	home := withTempHome(t)
	path := filepath.Join(home, ".config", "todoist-tui", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(path, []byte("ui: [not-valid"), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want parse error")
	}
}

func TestUpdateDefaultViewPreservesOtherSettings(t *testing.T) {
	home := withTempHome(t)
	path := filepath.Join(home, ".config", "todoist-tui", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("os.MkdirAll() error = %v", err)
	}
	seed := []byte("ui:\n  vim_mode: true\n  default_view: inbox\n  calendar_default_view: expanded\n  theme:\n    highlight: \"#123456\"\n")
	if err := os.WriteFile(path, seed, 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}

	if err := UpdateDefaultView("projects"); err != nil {
		t.Fatalf("UpdateDefaultView() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	var got Config
	if err := yaml.Unmarshal(data, &got); err != nil {
		t.Fatalf("yaml.Unmarshal() error = %v", err)
	}
	if got.UI.DefaultView != "projects" {
		t.Fatalf("DefaultView = %q, want %q", got.UI.DefaultView, "projects")
	}
	if !got.UI.VimMode || got.UI.CalendarDefaultView != "expanded" || got.UI.Theme.Highlight != "#123456" {
		t.Fatalf("UpdateDefaultView() clobbered existing settings: %#v", got.UI)
	}
}
