package views

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/config"
	"github.com/hy4ri/todoist-tui/internal/tui/components"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

// PomodoroView handles the Pomodoro timer tab.
type PomodoroView struct {
	*BaseView
	timer *components.TimerModel
}

// NewPomodoroView creates a new PomodoroView.
func NewPomodoroView(s *state.State) *PomodoroView {
	return &PomodoroView{
		BaseView: NewBaseView(s),
		timer:    components.NewTimerModel(99), // 99 as unique ID for Pomodoro timer
	}
}

// Name returns the view identifier.
func (v *PomodoroView) Name() string {
	return "pomodoro"
}

// OnEnter is called when switching to this view.
func (v *PomodoroView) OnEnter() tea.Cmd {
	v.State.FocusedPane = state.PaneMain

	// Initialize timer state if it's the first time, using config preferences.
	if v.State.PomodoroTarget == 0 {
		workMins := v.State.Config.UI.PomodoroWorkDuration
		if workMins <= 0 {
			workMins = 25 // default
		}
		v.State.PomodoroTarget = time.Duration(workMins) * time.Minute
		v.State.PomodoroMode = state.PomodoroCountdown
		v.State.PomodoroPhase = state.PomodoroWork
	}

	return nil
}

// OnExit is called when leaving this view.
func (v *PomodoroView) OnExit() {
	// We keep the timer running in the background
}

// HandleKey processes keyboard input for this view.
func (v *PomodoroView) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	switch msg.String() {
	case " ":
		// Toggle timer
		if v.State.PomodoroRunning {
			v.State.PomodoroRunning = false
			v.timer.Stop()
			return nil, true
		} else {
			v.State.PomodoroRunning = true
			return v.timer.Start(), true
		}

	case "r":
		// Reset timer
		v.State.PomodoroRunning = false
		v.timer.Stop()
		v.State.PomodoroElapsed = 0
		if v.State.PomodoroMode == state.PomodoroCountdown {
			if v.State.PomodoroPhase == state.PomodoroWork {
				// We keep current target
			} else {
				// Break phase
				v.State.PomodoroTarget = 5 * time.Minute
			}
		}
		return nil, true

	case "m":
		// Toggle mode
		if v.State.PomodoroMode == state.PomodoroCountdown {
			v.State.PomodoroMode = state.PomodoroStopwatch
		} else {
			v.State.PomodoroMode = state.PomodoroCountdown
			v.State.PomodoroTarget = 25 * time.Minute
			v.State.PomodoroPhase = state.PomodoroWork
		}
		v.State.PomodoroElapsed = 0
		return nil, true

	case "tab":
		// Cycle presets in countdown mode
		if v.State.PomodoroMode == state.PomodoroCountdown {
			if v.State.PomodoroTarget == 25*time.Minute {
				v.State.PomodoroTarget = 50 * time.Minute
				v.SetStatus("Preset: 50/10 Focus")
			} else {
				v.State.PomodoroTarget = 25 * time.Minute
				v.SetStatus("Preset: 25/5 Focus")
			}
			v.State.PomodoroElapsed = 0
		}
		return nil, true

	case "+":
		// Increase duration and persist only if in work phase
		if v.State.PomodoroMode == state.PomodoroCountdown {
			v.State.PomodoroTarget += 5 * time.Minute
			if v.State.PomodoroPhase == state.PomodoroWork {
				v.persistWorkDuration(v.State.PomodoroTarget)
			}
		}
		return nil, true

	case "-":
		// Decrease duration and persist only if in work phase
		if v.State.PomodoroMode == state.PomodoroCountdown {
			if v.State.PomodoroTarget > 5*time.Minute {
				v.State.PomodoroTarget -= 5 * time.Minute
				if v.State.PomodoroPhase == state.PomodoroWork {
					v.persistWorkDuration(v.State.PomodoroTarget)
				}
			}
		}
		return nil, true

	case "n":
		// Skip to next phase
		if v.State.PomodoroMode == state.PomodoroCountdown {
			return v.nextPhase(), true
		}
		return nil, true

	case "c":
		// Clear task
		v.State.PomodoroTask = nil
		v.State.PomodoroProject = ""
		return nil, true

	case "x":
		// Complete associated task
		if v.State.PomodoroTask != nil {
			// This will be handled by the main logic.Handler.HandleKey dispatch
			// but we return nothing here to let the main handler handle the "x" action
			// Actually, view handlers are called BEFORE the main keymap handling in my plan.
			// So I should return the action or let it fall through.
			return nil, false
		}
	}

	return nil, false
}

// HandleSelect processes Enter/selection for this view.
func (v *PomodoroView) HandleSelect() tea.Cmd {
	return nil
}

// HandleBack processes Escape for this view.
func (v *PomodoroView) HandleBack() (tea.Cmd, bool) {
	return nil, false
}

// Render returns the view's content.
func (v *PomodoroView) Render(width, height int) string {
	// Delegated to ui renderer
	return ""
}

// nextPhase transitions to the next Pomodoro phase.
func (v *PomodoroView) nextPhase() tea.Cmd {
	v.State.PomodoroElapsed = 0
	if v.State.PomodoroPhase == state.PomodoroWork {
		v.State.PomodoroSessions++
		// Determine break length (every 4 sessions long break)
		if v.State.PomodoroSessions%4 == 0 {
			v.State.PomodoroPhase = state.PomodoroLongBreak
			v.State.PomodoroTarget = 15 * time.Minute
		} else {
			v.State.PomodoroPhase = state.PomodoroShortBreak
			// If target was 50, break is 10, else 5
			if v.State.PomodoroTarget >= 50*time.Minute {
				v.State.PomodoroTarget = 10 * time.Minute
			} else {
				v.State.PomodoroTarget = 5 * time.Minute
			}
		}
	} else {
		v.State.PomodoroPhase = state.PomodoroWork
		// Restore the user's preferred work duration from config, falling back to 25m.
		workMins := v.State.Config.UI.PomodoroWorkDuration
		if workMins <= 0 {
			workMins = 25
		}
		v.State.PomodoroTarget = time.Duration(workMins) * time.Minute
	}
	v.SetStatus("Phase: " + v.phaseName())
	return nil
}

// persistWorkDuration saves the current work duration to config so it
// survives restarts. Saves to the in-memory config; the config is written
// to disk the next time config.Save is called (or by SaveWorkDuration).
func (v *PomodoroView) persistWorkDuration(d time.Duration) {
	if v.State.Config == nil {
		return
	}
	minutes := int(d.Minutes())
	v.State.Config.UI.PomodoroWorkDuration = minutes
	// Best-effort disk write; ignore errors to keep the UI responsive.
	_ = config.Save(v.State.Config)
}

func (v *PomodoroView) phaseName() string {
	switch v.State.PomodoroPhase {
	case state.PomodoroWork:
		return "Work"
	case state.PomodoroShortBreak:
		return "Short Break"
	case state.PomodoroLongBreak:
		return "Long Break"
	}
	return ""
}
