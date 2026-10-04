package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

func TestSpawnPickerUsesAllEntrypoints(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "config"))
	for _, name := range []string{"motley", "aderis/backend"} {
		path := filepath.Join(home, "projects", name)
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command("git", "init", path).CombinedOutput(); err != nil {
			t.Fatalf("git init: %s %v", out, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(home, ".motley"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".motley/config.toml"), []byte("repos_roots = ['~/projects', '~/projects/aderis']\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, cmd := (Model{}).beginSpawn()
	msg := cmd().(spawnLoaded)
	if msg.err != nil || !reflect.DeepEqual(msg.repos, []string{filepath.Join(home, "projects/motley"), filepath.Join(home, "projects/aderis/backend")}) {
		t.Fatalf("picker repositories: %v (%v)", msg.repos, msg.err)
	}
}

func TestRepositoryGroupsPathsAndScrolling(t *testing.T) {
	repos := []string{"/projects/zeta", "/projects/aderis/api", "/projects/aderis/backend"}
	f := &spawnForm{repos: repos, query: inputs("")[0]}
	want := "/projects\n  > zeta (/projects/zeta)\n\n/projects/aderis\n    api (/projects/aderis/api)\n    backend (/projects/aderis/backend)"
	if got := strings.Join(f.repositoryView(100, 20), "\n"); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	f.query.SetValue("aderis")
	if got := strings.Join(f.repositoryView(100, 20), "\n"); strings.Contains(got, "zeta") || !strings.Contains(got, "api (/projects/aderis/api)") {
		t.Fatal(got)
	}
	f.query.SetValue("")
	f.choice = 2
	rows := f.repositoryView(24, 5)
	view := strings.Join(rows, "\n")
	if len(rows) > 5 || !strings.Contains(view, "> backend") || !strings.Contains(strings.Join(strings.Fields(view), ""), "(/projects/aderis/backend)") {
		t.Fatal(view)
	}
	for _, row := range rows {
		if ansi.StringWidth(row) > 24 {
			t.Fatal("path overflow", row)
		}
	}
	// Headers are not selectable; moving into the next group chooses its path.
	m := newModel(false, false, "", nil)
	m.spawn = &spawnForm{repos: repos, query: inputs("")[0]}
	m = update(m, tea.KeyMsg{Type: tea.KeyDown})
	m = update(m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.spawn.opts.Repo != repos[1] {
		t.Fatal(m.spawn.opts.Repo)
	}
}
