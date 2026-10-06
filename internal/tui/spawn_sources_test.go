package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/worktree"
)

func TestSpawnRequiresSourceSelection(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "api")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"init", "-b", "main"},
		{"-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-m", "fixture"},
		{"branch", "develop"}, {"branch", "feature/stack"},
	} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", err, out)
		}
	}
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 80, Height: 24})
	m.spawnCfg.ReposRoot = filepath.Dir(repo)
	m.spawn = &spawnForm{repos: []string{repo}, query: inputs("")[0]}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if cmd == nil || !m.busy || m.spawn.opts.SourceRef != "" {
		t.Fatal("repository selection skipped loading")
	}
	m = update(m, cmd())
	if m.spawn.step != sourceStep || m.spawn.opts.SourceRef != "" {
		t.Fatal("source chosen without confirmation")
	}
	view := ansi.Strip(m.spawnView(20))
	for _, label := range []string{"Source branch", "[Default branch]", "main (local)", "[Unprefixed branches]", "develop (local)", "[Other branches]", "feature/stack (local)"} {
		if !strings.Contains(view, label) {
			t.Fatalf("missing %q:\n%s", label, view)
		}
	}
	m = update(m, key("feature/stack"))
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.spawn.step != identityStep || m.spawn.opts.SourceRef != "refs/heads/feature/stack" {
		t.Fatalf("wrong selected source: %+v", m.spawn.opts)
	}
	m = update(m, key("42"))
	m = update(m, tea.KeyMsg{Type: tea.KeyTab})
	m = update(m, key("child"))
	m.github = nil
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.spawn.opts.SourceRef != "refs/heads/feature/stack" || m.spawn.opts.Branch != "feature/api-42-child" {
		t.Fatal(m.spawn.opts)
	}
	m.spawn.plan.SourceRef = m.spawn.opts.SourceRef
	if got := m.previewSource(); got != "Source: feature/stack (local)" {
		t.Fatal(got)
	}
}

func TestSourcePickerEmptyFilterAndScrolling(t *testing.T) {
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 60, Height: 10})
	f := &spawnForm{step: sourceStep, query: inputs("")[0]}
	for i := 0; i < 50; i++ {
		name := fmt.Sprintf("feature/branch-%02d", i)
		f.sources = append(f.sources, worktree.SourceBranch{Ref: "refs/heads/" + name, Name: name})
	}
	m.spawn = f
	for range 49 {
		m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	}
	view := ansi.Strip(m.spawnView(5))
	if !strings.Contains(view, "> feature/branch-49") {
		t.Fatal(view)
	}
	for _, line := range strings.Split(view, "\n") {
		if ansi.StringWidth(line) > m.detailWidth() {
			t.Fatal("overflow", line)
		}
	}
	m = update(m, key("missing"))
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.spawn.step != sourceStep || m.spawn.opts.SourceRef != "" || !strings.Contains(m.spawnView(5), "No matching") {
		t.Fatal("empty filter allowed selection")
	}
}

func TestSourceLoadErrorAllowsRetry(t *testing.T) {
	m := newModel(false, false, "", nil)
	m.spawn = &spawnForm{repos: []string{"api"}, query: inputs("")[0]}
	m.busy = true
	m = update(m, spawnSourcesLoaded{err: fmt.Errorf("cannot read branches")})
	if m.busy || m.spawn.step != repoStep || m.spawn.err != "cannot read branches" {
		t.Fatal("load error lost")
	}
}
