package tui

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
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
	if msg.err != nil || !reflect.DeepEqual(msg.repos, []string{"backend", "motley"}) {
		t.Fatalf("picker repositories: %v (%v)", msg.repos, msg.err)
	}
}
