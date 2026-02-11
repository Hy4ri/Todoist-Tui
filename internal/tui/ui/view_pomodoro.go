package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/hy4ri/todoist-tui/internal/tui/components"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
	"github.com/hy4ri/todoist-tui/internal/tui/styles"
)

// renderPomodoro renders the Pomodoro view.
func (r *Renderer) renderPomodoro() string {
	width := r.Width - 4
	height := r.Height - 10 // Reserve space for tabs and status bar

	var content strings.Builder

	// 1. Header
	title := "🍅 POMODORO FOCUS"
	header := styles.Title.Width(width).Align(lipgloss.Center).Render(title)
	content.WriteString(header + "\n\n")

	// 2. Timer
	timeStr := components.FormatDuration(r.PomodoroTarget - r.PomodoroElapsed)
	if r.PomodoroMode == state.PomodoroStopwatch {
		timeStr = components.FormatDuration(r.PomodoroElapsed)
	}

	largeTime := components.RenderLargeTime(timeStr)
	timeStyle := styles.PomodoroTimer
	if r.PomodoroPhase != state.PomodoroWork {
		timeStyle = styles.PomodoroBreakTimer
	}

	content.WriteString(timeStyle.Width(width).Align(lipgloss.Center).Render(largeTime) + "\n")

	// 3. Progress Bar
	if r.PomodoroMode == state.PomodoroCountdown && r.PomodoroTarget > 0 {
		progress := float64(r.PomodoroElapsed) / float64(r.PomodoroTarget)
		if progress > 1 {
			progress = 1
		}
		barWidth := width / 2
		filled := int(float64(barWidth) * progress)
		empty := barWidth - filled
		bar := styles.PomodoroProgressBar.Render(strings.Repeat("█", filled) + strings.Repeat("░", empty))
		content.WriteString(lipgloss.NewStyle().Width(width).Align(lipgloss.Center).Render(bar) + "\n")
	}

	// 4. Phase & Session Info
	phaseLabel := "Work Session"
	if r.PomodoroPhase == state.PomodoroShortBreak {
		phaseLabel = "Short Break"
	} else if r.PomodoroPhase == state.PomodoroLongBreak {
		phaseLabel = "Long Break"
	}
	info := fmt.Sprintf("%s #%d", phaseLabel, r.PomodoroSessions+1)
	content.WriteString(styles.Subtitle.Width(width).Align(lipgloss.Center).Render(info) + "\n\n")

	// 5. Associated Task
	content.WriteString(r.renderPomodoroTask(width) + "\n\n")

	// 6. Help / Key Hints (local to view)
	hints := []string{
		"[Space] Start/Pause",
		"[r] Reset",
		"[m] Mode: " + r.modeName(),
		"[n] Next Phase",
		"[+/-] Duration",
		"[x] Complete Task",
		"[c] Clear Task",
	}
	hintsView := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(width/2).Align(lipgloss.Left).Render(strings.Join(hints[:4], "  ")),
		lipgloss.NewStyle().Width(width/2).Align(lipgloss.Right).Render(strings.Join(hints[4:], "  ")),
	)
	content.WriteString(hintsView)

	return lipgloss.NewStyle().MaxHeight(height).Render(content.String())
}

func (r *Renderer) renderPomodoroTask(width int) string {
	if r.PomodoroTask == nil {
		return lipgloss.NewStyle().
			Width(width).
			Height(5).
			Border(lipgloss.NormalBorder()).
			BorderForeground(styles.Subtle).
			Align(lipgloss.Center, lipgloss.Center).
			Render("No task associated.\nPress 'p' on any task in another view to work on it here.")
	}

	task := r.PomodoroTask
	project := r.PomodoroProject
	if project == "" {
		project = "Work"
	}

	priorityStyle := styles.GetPriorityStyle(task.Priority)
	prioLabel := fmt.Sprintf("P%d", 5-task.Priority)

	taskContent := lipgloss.JoinVertical(lipgloss.Left,
		styles.Title.Render(task.Content),
		lipgloss.JoinHorizontal(lipgloss.Left,
			lipgloss.NewStyle().Foreground(styles.Subtle).Render("Project: "+project),
			lipgloss.NewStyle().Foreground(styles.Subtle).Render(" • "),
			priorityStyle.Render("Priority: "+prioLabel),
		),
	)

	return lipgloss.NewStyle().
		Width(width).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Highlight).
		Padding(0, 1).
		Render(taskContent)
}

func (r *Renderer) modeName() string {
	if r.PomodoroMode == state.PomodoroCountdown {
		return "Countdown"
	}
	return "Stopwatch"
}
