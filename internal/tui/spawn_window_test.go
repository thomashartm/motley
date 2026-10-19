package tui

import (
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSpawnGhosttyChoice(t *testing.T) {
	m := update(newModel(false, true, "", nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	m.spawn = &spawnForm{step: previewStep}
	// The extra action must not replace launch, edit or cancel. Left wraps
	// from the default launch to the new window choice on macOS.
	m = update(m, tea.KeyMsg{Type: tea.KeyLeft})
	if runtime.GOOS != "darwin" {
		if strings.Contains(m.spawnView(20), "Ghostty") {
			t.Fatal("macOS action offered on another platform")
		}
		return
	}
	if !strings.Contains(m.spawnView(20), "Launch in new Ghostty window") {
		t.Fatal("new window launch is unreachable:", m.spawnView(20))
	}
	m = update(m, tea.KeyMsg{Type: tea.KeyRight})
	if m.spawn.previewAction != 0 {
		t.Fatal("new window action does not wrap back to normal launch")
	}
	m = update(m, tea.KeyMsg{Type: tea.KeyLeft})
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if cmd == nil || !m.busy || m.spawn == nil || m.spawn.step != launchStep {
		t.Fatal("new window choice did not start spawning")
	}
}
