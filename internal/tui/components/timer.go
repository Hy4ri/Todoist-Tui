package components

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// TimerTickMsg is sent every second when the timer is running.
type TimerTickMsg struct {
	ID int
}

// TimerPhaseCompleteMsg is sent when a countdown reaches zero.
type TimerPhaseCompleteMsg struct{}

// TimerModel handles the logic for a countdown or stopwatch timer.
type TimerModel struct {
	id      int
	running bool
}

// NewTimerModel creates a new timer component.
func NewTimerModel(id int) *TimerModel {
	return &TimerModel{
		id: id,
	}
}

// Start starts the timer.
func (m *TimerModel) Start() tea.Cmd {
	m.running = true
	return m.Tick()
}

// Stop stops the timer.
func (m *TimerModel) Stop() {
	m.running = false
}

// Tick returns a command that sends a TimerTickMsg after one second.
func (m *TimerModel) Tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TimerTickMsg{ID: m.id}
	})
}

// FormatDuration formats a duration as MM:SS or HH:MM:SS if needed.
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

// Large clock digits (minimal version for now, can expand later)
var Digits = map[rune][]string{
	'0': {" ███ ", "█   █", "█   █", "█   █", " ███ "},
	'1': {"  █  ", " ██  ", "  █  ", "  █  ", " ███ "},
	'2': {" ███ ", "    █", " ███ ", "█    ", " ███ "},
	'3': {" ███ ", "    █", " ███ ", "    █", " ███ "},
	'4': {"█   █", "█   █", " ███ ", "    █", "    █"},
	'5': {" ███ ", "█    ", " ███ ", "    █", " ███ "},
	'6': {" ███ ", "█    ", " ███ ", "█   █", " ███ "},
	'7': {" ███ ", "    █", "   █ ", "  █  ", "  █  "},
	'8': {" ███ ", "█   █", " ███ ", "█   █", " ███ "},
	'9': {" ███ ", "█   █", " ███ ", "    █", " ███ "},
	':': {"   ", " █ ", "   ", " █ ", "   "},
}

// RenderLargeTime renders time in large block characters.
func RenderLargeTime(tStr string) string {
	var rows [5]string
	for _, r := range tStr {
		lines, ok := Digits[r]
		if !ok {
			continue
		}
		for i := 0; i < 5; i++ {
			rows[i] += lines[i] + "  "
		}
	}

	var res string
	for i := 0; i < 5; i++ {
		res += rows[i] + "\n"
	}
	return res
}
