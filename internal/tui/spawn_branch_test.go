package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thomashartm/motley/internal/worktree"
)

func TestSpawnRepositoryBranchPrefix(t *testing.T) {
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 140, Height: 40})
	m.github = nil
	m.spawn = &spawnForm{repos: []string{"/Users/thomas/projects/aderis/backend"}, query: inputs("")[0]}
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	m = update(m, spawnSourcesLoaded{sources: []worktree.SourceBranch{{Ref: "refs/heads/develop", Name: "develop"}}})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if got := m.spawn.fields[3].Value(); got != "feature/backend-" {
		t.Fatal(got)
	}
	m = update(m, key("https://github.com/aderisERP/backend/issues/2991"))
	m = update(m, tea.KeyMsg{Type: tea.KeyTab})
	m = update(m, key("Allow exempt tax code"))
	if got := m.spawn.fields[3].Value(); got != "feature/backend-2991-allow-exempt-tax-code" {
		t.Fatal(got)
	}
	m = update(m, tea.KeyMsg{Type: tea.KeyTab})
	m = update(m, tea.KeyMsg{Type: tea.KeyRight})
	if got := m.spawn.fields[3].Value(); got != "fix/backend-2991-allow-exempt-tax-code" {
		t.Fatal(got)
	}
	if !strings.Contains(m.spawnView(30), "feature  [fix]") {
		t.Fatal("branch type choice is not visible")
	}
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.spawn.step != agentStep || m.spawn.opts.Ticket != "2991" || m.spawn.opts.Branch != "fix/backend-2991-allow-exempt-tax-code" {
		t.Fatalf("identity not accepted: %+v (%s)", m.spawn.opts, m.spawn.err)
	}
}

func TestSpawnManualBranchPreserved(t *testing.T) {
	m := identityForm(t)
	for range 3 {
		m = update(m, tea.KeyMsg{Type: tea.KeyTab})
	}
	m.spawn.fields[3].SetValue("feature/backend-custom")
	m = update(m, tea.KeyMsg{Type: tea.KeyEnd})
	m = update(m, key("-branch"))
	if !m.spawn.manualBranch {
		t.Fatal("branch edit not recorded")
	}
	m = update(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	m = update(m, tea.KeyMsg{Type: tea.KeyShiftTab})
	m = update(m, key(" another name"))
	if got := m.spawn.fields[3].Value(); got != "feature/backend-custom-branch" {
		t.Fatal(got)
	}
	m = update(m, tea.KeyMsg{Type: tea.KeyTab})
	m = update(m, tea.KeyMsg{Type: tea.KeyLeft})
	if got := m.spawn.fields[3].Value(); got != "fix/backend-custom-branch" {
		t.Fatal(got)
	}
}
