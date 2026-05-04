package utils

import (
	"regexp"
	"strings"
	"unicode"
)

var (
	ansiCSI  = regexp.MustCompile(`\x1b\[[0-?]*[ -/]*[@-~]`)
	ansiOSC  = regexp.MustCompile(`\x1b\][^\a]*(?:\a|\x1b\\)`)
	ansiDCS  = regexp.MustCompile(`(?s)\x1b[PX^_].*?\x1b\\`)
	ansiMisc = regexp.MustCompile(`\x1b[@-_]`)
)

// SanitizeTerminalText removes terminal escape sequences and control characters
// that can be used to spoof or inject content into the TUI.
func SanitizeTerminalText(s string) string {
	if s == "" {
		return ""
	}

	s = ansiOSC.ReplaceAllString(s, "")
	s = ansiDCS.ReplaceAllString(s, "")
	s = ansiCSI.ReplaceAllString(s, "")
	s = ansiMisc.ReplaceAllString(s, "")

	return strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\t':
			return r
		case '\r':
			return -1
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// SanitizeSingleLineText removes terminal escape sequences and collapses line
// breaks so the result is safe to render in inline UI fields.
func SanitizeSingleLineText(s string) string {
	s = SanitizeTerminalText(s)
	s = strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(s)
	return s
}
