package utils

import "testing"

func TestSanitizeTerminalText(t *testing.T) {
	input := "ok\x1b[31mred\x1b[0m\nline\r\x07"
	got := SanitizeTerminalText(input)
	want := "okred\nline"
	if got != want {
		t.Fatalf("SanitizeTerminalText() = %q, want %q", got, want)
	}
}

func TestSanitizeSingleLineText(t *testing.T) {
	input := "one\n\x1b[31mtwo\x1b[0m\tthree"
	got := SanitizeSingleLineText(input)
	want := "one two three"
	if got != want {
		t.Fatalf("SanitizeSingleLineText() = %q, want %q", got, want)
	}
}
