package tui

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thomashartm/motley/internal/member"
	"github.com/thomashartm/motley/internal/state"
	"github.com/thomashartm/motley/internal/terminal"
)

func TestSpawnMenuStartsWithAgentAndDestination(t *testing.T) {
	m := update(newModel(false, true, "", nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	m = update(m, key("home"))
	if !strings.Contains(m.View(), "Spawn agent (s)") {
		t.Fatal("main menu lacks Spawn agent:", m.View())
	}
	m = update(m, key("s"))
	if m.busy || m.spawn == nil || m.spawn.step != agentStep {
		t.Fatal("spawn must start with agent choice")
	}
	m = arrow(m, tea.KeyDown)
	m = arrow(m, tea.KeyEnter)
	if m.spawn.opts.Agent != "codex" || !strings.Contains(m.spawnView(20), "Launch destination") {
		t.Fatal("destination must follow agent choice:", m.spawnView(20))
	}
	if runtime.GOOS == "darwin" {
		if !strings.Contains(m.spawnView(20), "New Ghostty tab") || !strings.Contains(m.spawnView(20), "tmux session") {
			t.Fatal(m.spawnView(20))
		}
		m = arrow(m, tea.KeyDown)
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if cmd == nil || !m.busy || m.spawn.opts.LaunchTarget != "tmux" || m.spawn.opts.Agent != "codex" {
		t.Fatal("destination did not proceed to worktree setup")
	}
	m = update(m, spawnLoaded{repos: []string{"api"}})
	if m.spawn.step != repoStep {
		t.Fatal("repository must follow destination")
	}
}

func TestSpawnGhosttyChoiceAndCancel(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Ghostty launcher is macOS only")
	}
	m := update(newModel(false, true, "", nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	m = update(m, key("s"))
	m = arrow(m, tea.KeyEnter)
	m = arrow(m, tea.KeyEnter)
	if m.spawn.opts.LaunchTarget != "ghostty" {
		t.Fatal("direct Ghostty destination not retained")
	}
	// Cancelling repository setup must not launch or create a member.
	m = update(m, spawnLoaded{repos: []string{"api"}})
	m = arrow(m, tea.KeyEsc)
	if m.spawn != nil || len(m.allRows) != 0 {
		t.Fatal("cancel created a member")
	}
}

func TestSpawnLaunchFailureKeepsCreatedMember(t *testing.T) {
	m := update(newModel(false, true, "", nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	m.spawn = &spawnForm{step: launchStep}
	m = update(m, spawnFinished{manifest: member.Manifest{ID: "test", LaunchTarget: "ghostty"}, err: errors.New("Ghostty launch failed; run mtly revive test")})
	if m.spawn != nil || len(m.allRows) != 1 || m.allRows[0].Alive || !strings.Contains(m.message, "mtly revive test") {
		t.Fatal("failed launch must retain worktree for recovery:", m.message)
	}
}

func TestSpawnMenuAndDestinationsWithMouse(t *testing.T) {
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	m = update(m, key("home"))
	m = click(m, m.listWidth()+4, actionScreenY(t, m, "Spawn agent (s)"))
	if m.spawn == nil || m.spawn.step != agentStep {
		t.Fatal("menu click did not start spawn")
	}
	m = click(m, m.listWidth()+4, 2+m.panelHeadingGap()+3) // codex (spaced choices)
	if m.spawn.step != destinationStep || m.spawn.opts.Agent != "codex" {
		t.Fatal("agent click did not reach destination")
	}
	m = click(m, m.listWidth()+4, 2+m.panelHeadingGap()+1)
	if !m.busy || (m.spawn.step != repoStep && m.spawn.step != windowStep) {
		t.Fatal("destination click did not open worktree setup")
	}
}

func TestDirectGhosttyOverviewWithoutTmux(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_STATE_HOME", root)
	t.Setenv("PATH", root) // no tmux or other executable
	m := member.Manifest{ID: "direct", Agent: "codex", LaunchTarget: "ghostty", Worktree: root}
	writeManifest(t, m)
	dir, err := state.MembersDir()
	if err != nil {
		t.Fatal(err)
	}
	lock, err := terminal.Acquire(dir, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = lock.Close() }()
	if err := terminal.WriteStatus(dir, m.ID, terminal.Status{Agent: "codex", Status: "working"}); err != nil {
		t.Fatal(err)
	}
	got := pollMembers(&member.Catalog{}, dir, false)
	if got.err != nil || len(got.rows) != 1 || !got.rows[0].Alive || got.rows[0].CurrentStatus() != "working" {
		t.Fatalf("direct-only poll: %+v", got)
	}
	model := update(newModel(false, false, "", nil), got)
	model = update(model, key("d"))
	if model.terminating != nil || !strings.Contains(model.message, "original Ghostty window") {
		t.Fatal("live direct agent offered misleading termination", model.message)
	}
	// The persisted member remains untouched by polling or the refused action.
	if _, err := os.Stat(filepath.Join(dir, "direct.toml")); err != nil {
		t.Fatal(err)
	}
}

func TestSpawnPreviewShowsChosenDestination(t *testing.T) {
	for _, target := range []string{"tmux", "ghostty"} {
		m := update(newModel(false, true, "", nil), tea.WindowSizeMsg{Width: 140, Height: 40})
		m.spawn = &spawnForm{}
		m = update(m, spawnPrepared{plan: member.Prepared{Manifest: member.Manifest{Agent: "codex", LaunchTarget: target}}})
		view := m.spawnView(20)
		if !strings.Contains(view, "Destination: "+spawnDestination(target)) {
			t.Fatal(view)
		}
		m = arrow(m, tea.KeyLeft)
		if m.spawn.previewAction != 2 {
			t.Fatal("preview must have ordinary Launch/Edit/Cancel actions")
		}
	}
}

func TestSmallSpawnPreviewKeepsLaunchVisible(t *testing.T) {
	m := update(newModel(false, true, "", nil), tea.WindowSizeMsg{Width: 60, Height: 10})
	m.spawn = &spawnForm{}
	m = update(m, spawnPrepared{plan: member.Prepared{
		Manifest:  member.Manifest{Agent: "codex", LaunchTarget: "ghostty"},
		SourceRef: "refs/heads/main", Issue: &member.IssueContext{Number: 123, Title: "Fix issue"},
	}})
	if !strings.Contains(m.spawnView(m.contentHeight()), "> Launch") {
		t.Fatal("launch control hidden:", m.spawnView(m.contentHeight()))
	}
}

func TestSpawnSelectsExistingGhosttyWindow(t *testing.T) {
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	m.spawn = &spawnForm{step: destinationStep, opts: member.SpawnOptions{Agent: "claude", LaunchTarget: "ghostty"}, query: inputs("")[0]}
	m = update(m, spawnWindowsLoaded{windows: []terminal.GhosttyWindow{{ID: "front", Name: "Current work", Current: true}, {ID: "other", Name: "Other work"}}})
	if m.spawn.step != windowStep || m.spawn.choice != 0 {
		t.Fatal("current window must be selected first")
	}
	m = arrow(m, tea.KeyDown)
	m = arrow(m, tea.KeyEnter)
	if m.spawn.opts.GhosttyWindow != "other" || m.spawn.step != repoStep {
		t.Fatal("window choice lost", m.spawn.opts)
	}
}

func TestSpawnWindowPreviewRetainsSelection(t *testing.T) {
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	m.spawn = &spawnForm{step: windowStep, choice: 1, windows: []terminal.GhosttyWindow{{ID: "current"}, {ID: "chosen"}}}
	next, cmd := m.Update(key("p"))
	m = next.(Model)
	if cmd == nil || !m.busy || m.spawn.choice != 1 {
		t.Fatal("preview changed selection or did not run")
	}
	m = update(m, spawnWindowPreviewed{})
	if m.busy || m.spawn.step != windowStep || m.spawn.choice != 1 {
		t.Fatal("preview lost spawn state")
	}
	m.spawn.choice = 2
	next, cmd = m.Update(key("p"))
	if cmd != nil || next.(Model).busy {
		t.Fatal("new window must not be created by preview")
	}
}
