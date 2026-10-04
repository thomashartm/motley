package member

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestRepositoryEntrypoints(t *testing.T) {
	root := t.TempDir()
	projects := filepath.Join(root, "projects")
	aderis := filepath.Join(projects, "aderis")
	git := func(path string, args ...string) {
		t.Helper()
		out, err := exec.Command("git", append([]string{"-C", path}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s %v", args, out, err)
		}
	}
	init := func(path string) {
		t.Helper()
		if err := os.MkdirAll(path, 0700); err != nil {
			t.Fatal(err)
		}
		git(path, "init", "-b", "main")
	}
	init(filepath.Join(projects, "motley"))
	init(filepath.Join(aderis, "backend"))
	init(filepath.Join(aderis, "nested", "hidden"))
	roots := []string{projects, aderis}
	got, err := DiscoverRepos(roots)
	if err != nil || !reflect.DeepEqual(got, []string{"backend", "motley"}) {
		t.Fatal(got, err)
	}
	for _, name := range got {
		path, err := ResolveRepo(roots, name)
		if err != nil || filepath.Base(path) != name {
			t.Fatal(path, err)
		}
	}
	for _, name := range []string{"hidden", "nested/hidden", "../motley", filepath.Join(aderis, "nested", "hidden")} {
		if _, err := ResolveRepo(roots, name); err == nil {
			t.Fatal("accepted non-entrypoint repository", name)
		}
	}
	// Duplicate names require an explicit, directly contained path.
	first := filepath.Join(projects, "api")
	second := filepath.Join(aderis, "api")
	init(first)
	init(second)
	got, err = DiscoverRepos(roots)
	want := []string{first, second, "backend", "motley"}
	// Absolute selectors sort by path rather than configured root order.
	sort.Strings(want)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal(got, err)
	}
	if _, err := ResolveRepo(roots, "api"); err == nil || !strings.Contains(err.Error(), "ambiguous") || !strings.Contains(err.Error(), second) {
		t.Fatal(err)
	}
	for _, path := range []string{first, second} {
		want, err := filepath.EvalSymlinks(path)
		if err != nil {
			t.Fatal(err)
		}
		if got, err := ResolveRepo(roots, path); err != nil || got != want {
			t.Fatal(got, err)
		}
	}
	// Repeated or symlinked roots must not duplicate a physical repository.
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(aderis, alias); err != nil {
		t.Fatal(err)
	}
	got, err = DiscoverRepos(append(roots, aderis, alias))
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatal(got, err)
	}
	if _, err := ResolveRepo([]string{aderis, alias}, "api"); err != nil {
		t.Fatal(err)
	}
	// An alias with a conflicting name must not make another repository's
	// short selector ambiguous after physical-path deduplication.
	if err := os.Symlink(filepath.Join(aderis, "backend"), filepath.Join(aderis, "motley")); err != nil {
		t.Fatal(err)
	}
	got, err = DiscoverRepos(roots)
	if err != nil {
		t.Fatal(err)
	}
	for _, selection := range got {
		if _, err := ResolveRepo(roots, selection); err != nil {
			t.Fatalf("picker offered unusable selection %q: %v", selection, err)
		}
	}
	// Existing linked worktrees cannot be used as main repositories.
	git(second, "-c", "user.name=Fixture", "-c", "user.email=fixture@example.invalid", "commit", "--allow-empty", "-m", "initial")
	git(second, "worktree", "add", "-b", "linked", filepath.Join(aderis, "linked"))
	if _, err := ResolveRepo(roots, "linked"); err == nil || !strings.Contains(err.Error(), "linked worktree") {
		t.Fatal(err)
	}
	if _, err := DiscoverRepos([]string{filepath.Join(root, "missing"), aderis}); err == nil || !strings.Contains(err.Error(), "read repository root") {
		t.Fatal(err)
	}
}

func TestPrepareFromSecondRootAndQualifiedPath(t *testing.T) {
	cfg, root, repo, _ := prepareRepo(t, "ROOT/remote.git")
	first := filepath.Join(root, "other")
	if err := os.MkdirAll(first, 0700); err != nil {
		t.Fatal(err)
	}
	cfg.ReposRoots = []string{first, filepath.Dir(repo)}
	writeBlueprint(t, repo, "plan", "+++\nrepos=['api']\n+++\n{{.Repo}} {{.Worktree}}")
	for _, selection := range []string{"api", repo} {
		plan, err := Prepare(cfg, SpawnOptions{Repo: selection, Branch: "feat/roots", Blueprint: "plan", NoGH: true})
		if err != nil {
			t.Fatal(err)
		}
		physical, _ := filepath.EvalSymlinks(repo)
		if plan.Manifest.RepoPath != physical || plan.Manifest.Repo != "api" || plan.Manifest.Worktree != filepath.Join(cfg.WorktreesRoot, "api", "feat-roots") || !strings.HasPrefix(plan.Prompt, "api ") {
			t.Fatal(plan)
		}
	}
}
