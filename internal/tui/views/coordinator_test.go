package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/hy4ri/todoist-tui/internal/tui/state"
)

type mockView struct {
	name     string
	exitCnt  int
	enterCnt int
	selectFn func() tea.Cmd
	backFn   func() (tea.Cmd, bool)
}

func (m *mockView) Name() string                         { return m.name }
func (m *mockView) HandleKey(tea.KeyMsg) (tea.Cmd, bool) { return nil, false }
func (m *mockView) HandleSelect() tea.Cmd {
	if m.selectFn != nil {
		return m.selectFn()
	}
	return nil
}
func (m *mockView) HandleBack() (tea.Cmd, bool) {
	if m.backFn != nil {
		return m.backFn()
	}
	return nil, false
}
func (m *mockView) OnEnter() tea.Cmd       { m.enterCnt++; return nil }
func (m *mockView) OnExit()                { m.exitCnt++ }
func (m *mockView) Render(int, int) string { return m.name }

func TestRegistryAndCoordinatorSwitching(t *testing.T) {
	s := &state.State{CurrentTab: state.TabInbox}
	rin := NewRegistry()
	inbox := &mockView{name: "inbox"}
	projects := &mockView{name: "projects"}
	rin.RegisterView(inbox)
	rin.RegisterView(projects)
	rin.RegisterTab(state.TabInbox, "", "Inbox", "Inb", "inbox")
	rin.RegisterTab(state.TabProjects, "", "Projects", "Prj", "projects")

	c := &Coordinator{registry: rin, state: s}
	if view, ok := rin.GetViewForTab(state.TabInbox); !ok || view.Name() != "inbox" {
		t.Fatalf("GetViewForTab() = (%v, %v)", view, ok)
	}

	c.currentView = inbox
	if cmd := c.SwitchToTab(state.TabProjects); cmd != nil {
		t.Fatalf("SwitchToTab() cmd = %v, want nil", cmd)
	}
	if inbox.exitCnt != 1 || projects.enterCnt != 1 {
		t.Fatalf("view lifecycle counts = inbox exit %d, projects enter %d", inbox.exitCnt, projects.enterCnt)
	}
	if s.CurrentTab != state.TabProjects || s.CurrentView != state.ViewProject {
		t.Fatalf("state not updated: tab=%v view=%v", s.CurrentTab, s.CurrentView)
	}
	if got := c.GetCurrentView(); got == nil || got.Name() != "projects" {
		t.Fatalf("current view not switched: %#v", got)
	}
}

func TestCoordinatorNilAndInvalidCases(t *testing.T) {
	c := &Coordinator{registry: NewRegistry(), state: &state.State{}}
	if cmd, ok := c.HandleKey(tea.KeyMsg{}); cmd != nil || ok {
		t.Fatalf("HandleKey() = (%v, %v), want (nil, false)", cmd, ok)
	}
	if cmd := c.HandleSelect(); cmd != nil {
		t.Fatalf("HandleSelect() = %v, want nil", cmd)
	}
	if cmd, ok := c.HandleBack(); cmd != nil || ok {
		t.Fatalf("HandleBack() = (%v, %v), want (nil, false)", cmd, ok)
	}
	if cmd := c.SwitchToTab(state.Tab(-1)); cmd != nil {
		t.Fatalf("SwitchToTab(invalid) = %v, want nil", cmd)
	}
}
