package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/member"
)

func TestRetireDialogRequiresConfirmationAndExplicitForce(t *testing.T) {
	m := update(newModel(false, true, "client", nil), tea.WindowSizeMsg{Width: 100, Height: 25})
	m = update(m, snapshot{rows: []member.Row{row("a", true), row("b", false)}})
	next, cmd := m.Update(key("x"))
	m = next.(Model)
	if cmd == nil || !m.busy || m.retiring == nil {
		t.Fatal("x did not start prechecks")
	}
	m = update(m, retireChecked{id: "a", check: member.RetireCheck{Manifest: m.rows[0].Manifest, Dirty: true, Ahead: 2}})
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if cmd != nil || m.busy || !strings.Contains(m.message, "discarded") {
		t.Fatal("dirty retirement proceeded without force")
	}
	m = update(m, key("f"))
	m = update(m, key("k"))
	if !m.retiring.force || !m.retiring.keep {
		t.Fatal("retirement choices missing")
	}
	m = update(m, tea.KeyMsg{Type: tea.KeyEsc})
	if m.retiring != nil || m.busy {
		t.Fatal("cancel did not return to overview")
	}
	m = update(m, key("x"))
	m = update(m, retireChecked{id: "a", err: errors.New("main worktree")})
	m = update(m, key("f"))
	_, cmd = m.Update(key("y"))
	if cmd != nil {
		t.Fatal("force bypassed structural refusal")
	}
}
func TestReviveOnlyDeadAndLifecycleFailure(t *testing.T) {
	m := update(newModel(false, true, "client", nil), snapshot{rows: []member.Row{row("a", true), row("b", false)}})
	next, cmd := m.Update(key("r"))
	m = next.(Model)
	if cmd != nil || !strings.Contains(m.message, "dead member") {
		t.Fatal("live revive not refused")
	}
	m = update(m, key("j"))
	next, cmd = m.Update(key("r"))
	m = next.(Model)
	if cmd == nil || !m.busy {
		t.Fatal("dead revive not started")
	}
	m = update(m, lifecycleDone{id: "b", action: "Revived", err: errors.New("worktree missing")})
	if m.busy || m.message != "worktree missing" {
		t.Fatal("lifecycle error not presented")
	}
}

func TestTerminateSelectedMemberAndCancel(t *testing.T) {
	m := update(newModel(true, true, "client", nil), tea.WindowSizeMsg{Width: 100, Height: 25})
	m = update(m, snapshot{rows: []member.Row{row("alpha", true), row("beta", true)}})
	// Select beta by clicking its list row, then Actions in the bottom bar.
	m = click(m, 5, listScreenY(t, m, "beta"))
	m = update(m, tea.MouseMsg{X: ansi.StringWidth(m.navigationBar()[:strings.Index(m.navigationBar(), "[3 Actions]")]) + 1, Y: 24, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if m.selectedID() != "beta" || !strings.Contains(m.View(), "Actions: beta") {
		t.Fatal("action target is not selected member")
	}
	index := -1
	for i, a := range m.actions() {
		if a.key == "d" {
			index = i
		}
	}
	if index < 0 {
		t.Fatal("terminate action missing")
	}
	m.actionCursor = index
	next, cmd := mouseClick(m, m.listWidth()+4, actionScreenY(t, m, "Terminate agent (d)"))
	m = next.(Model)
	if cmd != nil || m.terminating == nil || m.terminating.id != "beta" || m.busy {
		t.Fatal("terminate should confirm selected target")
	}
	// Polling can reorder rows without changing the confirmation's target.
	m = update(m, snapshot{rows: []member.Row{row("beta", true), row("alpha", true)}})
	if m.terminating.id != "beta" {
		t.Fatal("confirmation target changed")
	}
	next, cmd = m.Update(tea.MouseMsg{X: strings.Index(m.terminateButtons(), "[Terminate: y]") + 1, Y: 24, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if !next.(Model).busy || cmd == nil {
		t.Fatal("mouse confirmation did not dispatch")
	}
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.terminating != nil || m.busy {
		t.Fatal("default Enter must cancel")
	}
	m = update(m, key("d"))
	m = update(m, tea.KeyMsg{Type: tea.KeyRight})
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !next.(Model).busy || cmd == nil {
		t.Fatal("arrow confirmation did not dispatch")
	}
	m = update(next.(Model), lifecycleDone{id: "beta", action: "Terminated"})
	if m.terminating != nil || m.busy || m.message != "Terminated beta" {
		t.Fatal("termination did not complete")
	}
	m = update(m, snapshot{rows: []member.Row{row("beta", false)}})
	next, cmd = m.Update(key("d"))
	if cmd != nil || next.(Model).terminating == nil || !strings.Contains(next.(Model).terminateView(20), "Already stopped") {
		t.Fatal("stopped entry cannot be removed")
	}
}

func TestTerminateFromListAndVisibleFailure(t *testing.T) {
	for _, size := range [][2]int{{60, 10}, {80, 20}, {120, 30}} {
		m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		m = update(m, snapshot{rows: []member.Row{row("alpha", true), row("beta", false)}})
		m = update(m, key("j"))
		bar := m.navigationBar()
		index := strings.Index(bar, "[d Terminate]")
		if index < 0 || ansi.StringWidth(bar) > m.width-2 {
			t.Fatal("list termination missing or clipped", bar)
		}
		next, cmd := mouseClick(m, ansi.StringWidth(bar[:index])+1, m.height-1)
		m = next.(Model)
		if cmd != nil || m.terminating == nil || m.terminating.id != "beta" || !m.terminating.stopped {
			t.Fatal("direct control did not confirm selected stopped entry")
		}
		m = update(m, tea.WindowSizeMsg{Width: 120, Height: 30})
		m = update(m, lifecycleDone{id: "beta", action: "Terminated", err: errors.New("agent refused to stop")})
		if m.terminating == nil || m.busy || !strings.Contains(m.terminateView(m.contentHeight()), "agent refused to stop") || len(m.rows) != 2 {
			t.Fatal("failure hidden or entry removed")
		}
		next, cmd = m.Update(key("y"))
		if !next.(Model).busy || cmd == nil {
			t.Fatal("failure cannot be retried")
		}
		m = update(next.(Model), lifecycleDone{id: "beta", action: "Terminated"})
		m = update(m, snapshot{err: errors.New("unrelated discovery failed")})
		if len(m.rows) != 1 || m.rows[0].ID != "alpha" || m.terminating != nil {
			t.Fatal("archived entry stayed listed after polling failure")
		}
	}
}

func TestRetireMouseChoices(t *testing.T) {
	m := update(newModel(true, true, "", nil), tea.WindowSizeMsg{Width: 100, Height: 25})
	m.retiring = &retireDialog{id: "a", loaded: true, check: member.RetireCheck{Dirty: true}}
	click := func(index int) (tea.Model, tea.Cmd) {
		first := len(strings.Split(m.retireView(m.contentHeight()), "\n")) - 4
		return m.Update(tea.MouseMsg{X: m.listWidth() + 4, Y: 2 + first + index + m.panelHeadingGap(), Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	}
	next, cmd := click(0)
	m = next.(Model)
	if cmd != nil || m.busy {
		t.Fatal("mouse discarded work without force")
	}
	next, _ = click(1)
	m = next.(Model)
	if !m.retiring.force || m.busy {
		t.Fatal("force click did not toggle")
	}
	next, _ = click(2)
	m = next.(Model)
	if !m.retiring.keep {
		t.Fatal("keep branch click did not toggle")
	}
	next, cmd = click(0)
	if !next.(Model).busy || cmd == nil {
		t.Fatal("confirmed retirement did not dispatch")
	}
	next, cmd = click(3)
	if next.(Model).retiring != nil || cmd != nil {
		t.Fatal("mouse cancel did not cancel")
	}
}

func TestTerminatePanelChoices(t *testing.T) {
	for _, height := range []int{12, 25, 45} {
		m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 100, Height: height})
		m.terminating = &terminateDialog{id: "target"}
		lines := strings.Split(ansi.Strip(m.View()), "\n")
		for _, label := range []string{"Cancel (esc)", "Terminate (y)"} {
			found := false
			for y, line := range lines {
				x := strings.Index(line, label)
				if x < 0 {
					continue
				}
				found = true
				x = ansi.StringWidth(line[:x])
				next, cmd := m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
				got := next.(Model)
				if label == "Terminate (y)" && (!got.busy || cmd == nil) {
					t.Fatal("visible confirmation did not execute")
				}
				if label == "Cancel (esc)" && (got.terminating != nil || cmd != nil) {
					t.Fatal("visible cancel did not cancel")
				}
			}
			if !found {
				t.Fatalf("height %d: missing %s", height, label)
			}
		}
	}
}

func TestRetireLayoutWithLongCheckout(t *testing.T) {
	path := "/Users/thomas/projects/aderis/infrastructure-stacks-feature-125-async-batching-infrastructure"
	for _, size := range [][2]int{{60, 10}, {100, 25}, {200, 35}} {
		for _, imported := range []bool{false, true} {
			m := actionModel(size[0], size[1])
			manifest := member.Manifest{ID: "infrastructure-stacks-feature-125", Worktree: path}
			if imported {
				manifest.ClaudeSession = "tracked"
			}
			m.retiring = &retireDialog{id: manifest.ID, loaded: true, check: member.RetireCheck{Manifest: manifest}}
			view := ansi.Strip(m.retireView(m.contentHeight()))
			if lipgloss.Width(m.View()) > m.width || lipgloss.Height(m.View()) > m.height {
				t.Fatalf("retirement overflows at %v:\n%s", size, m.View())
			}
			lines := strings.Split(view, "\n")
			first := len(lines) - len(m.retireChoices())
			for i, label := range m.retireChoices() {
				if lines[first+i] != fit("  "+control(label, i == 0), m.detailWidth()) {
					t.Fatalf("control missing at %v: %s", size, view)
				}
			}
			if size[0] == 200 && imported {
				t.Logf("retirement confirmation:\n%s", view)
				for _, want := range []string{"  Stops all tracked conversations", "\n  \n  Keeps the checkout", "\n  Dir\n  /Users/"} {
					if !strings.Contains(view, want) {
						t.Fatalf("missing spacing or path label %q:\n%s", want, view)
					}
				}
				if !strings.Contains(strings.Join(strings.Fields(view), ""), path) {
					t.Fatalf("long path lost:\n%s", view)
				}
				// The divider must not confirm retirement; Cancel must still work.
				x := m.listWidth() + 5
				y := 2 + first - 1 + m.panelHeadingGap()
				next, cmd := m.Update(tea.MouseMsg{X: x, Y: y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
				if cmd != nil || next.(Model).busy {
					t.Fatal("divider click retired a member")
				}
				next, cmd = m.Update(tea.MouseMsg{X: x, Y: y + 2, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
				if cmd != nil || next.(Model).retiring != nil {
					t.Fatal("Cancel moved away from its mouse target")
				}
			}
		}
	}
}
