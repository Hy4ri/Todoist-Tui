package styles

// SidebarWidth returns the responsive sidebar width based on terminal width.
// The same breakpoints are used by both the renderer (view_*.go) and the
// input handler (input_mouse.go) to keep click regions in sync with rendering.
func SidebarWidth(totalWidth int) int {
	switch {
	case totalWidth < 50:
		return 15
	case totalWidth < 80:
		return 20
	default:
		return 30
	}
}
