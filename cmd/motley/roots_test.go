package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpawnFromMultipleRoots(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	first := filepath.Join(f.home, "other projects")
	if err := os.MkdirAll(first, 0700); err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(f.home, ".motley/config.toml"), fmt.Sprintf("repos_roots = [%q, %q]\nworktrees_root = %q\n", first, filepath.Dir(f.repo), f.trees), 0600)
	writeFixture(t, filepath.Join(f.repo, ".motley/blueprints/plan.md"), "+++\nrepos=['api']\n+++\nPlan {{.Repo}}\n", 0600)
	for i, selection := range []string{"api", f.repo} {
		if out := f.motley("blueprint", "list", "--repo", selection); !strings.Contains(out, "plan") {
			t.Fatal(out)
		}
		branch := fmt.Sprintf("feat/roots-%d", i)
		f.motley("spawn", "--repo", selection, "--branch", branch, "--blueprint", "plan", "--detach")
		m := f.manifest(fmt.Sprintf("feat-roots-%d", i))
		physical, err := filepath.EvalSymlinks(f.repo)
		if err != nil || m.RepoPath != physical || m.Repo != "api" || filepath.Dir(m.Worktree) != filepath.Join(f.trees, "api") {
			t.Fatalf("spawned from wrong repository: %+v (%v)", m, err)
		}
		if f.git(m.Worktree, "rev-parse", "HEAD") != f.git(f.remote, "rev-parse", "refs/heads/"+branch) {
			t.Fatal("new branch was not pushed to fixture origin")
		}
	}
}
