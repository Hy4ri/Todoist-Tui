package ui

import "github.com/mattn/go-runewidth"

// truncateString truncates a string to the given width, appending "…" if truncated.
// It handles wide characters correctly using runewidth.
func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	if runewidth.StringWidth(s) <= maxLen {
		return s
	}

	// Iterate by runes to find cut point
	w := 0
	for i, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > maxLen-1 { // -1 for ellipsis
			return s[:i] + "…"
		}
		w += rw
	}

	return s
}
