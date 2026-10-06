package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thomashartm/motley/internal/agents"
	"github.com/thomashartm/motley/internal/member"
)

func TestSpawnAuthorizationForEveryAgent(t *testing.T) {
	for _, agent := range []string{"claude", "codex", "opencode"} {
		for index, mode := range agents.Modes(agent) {
			t.Run(agent+"/"+mode.Name, func(t *testing.T) {
				m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 100, Height: 24})
				m.spawn = &spawnForm{step: blueprintStep, opts: member.SpawnOptions{Agent: agent}}
				m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
				if m.spawn.step != modeStep || !strings.Contains(m.spawnView(20), "Authorization level") {
					t.Fatal("authorization skipped")
				}
				m.spawn.choice = index + 1
				next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m = next.(Model)
				if cmd == nil || !m.busy || m.spawn.opts.Mode != mode.Name {
					t.Fatal(m.spawn.opts)
				}
			})
		}
	}
}
