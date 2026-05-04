package styles

import (
	"testing"

	"github.com/hy4ri/todoist-tui/internal/config"
)

func TestGetColorAndPriorityStyle(t *testing.T) {
	if got := GetColor("blue"); got == "" {
		t.Fatal("GetColor(blue) returned empty color")
	}
	if got := GetColor("unknown"); got != "" {
		t.Fatalf("GetColor(unknown) = %q, want empty", got)
	}
	_ = GetPriorityStyle(4).Render("x")
	_ = GetPriorityStyle(1).Render("x")
}

func TestInitThemeHandlesNilAndOverrides(t *testing.T) {
	InitTheme(nil)
	InitTheme(&config.ThemeConfig{Highlight: "#111111", Subtle: "#222222", Error: "#333333"})
}
