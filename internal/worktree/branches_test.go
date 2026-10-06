package worktree

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func sourceGit(t *testing.T, repo string, args ...string) string {
	t.Helper()
	out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func sourceRepo(t *testing.T) (string, string) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	root := t.TempDir()
	repo, remote := filepath.Join(root, "repo"), filepath.Join(root, "remote.git")
	if err := os.Mkdir(repo, 0700); err != nil {
		t.Fatal(err)
	}
	sourceGit(t, repo, "init", "-b", "main")
	sourceGit(t, repo, "config", "user.name", "Fixture")
	sourceGit(t, repo, "config", "user.email", "fixture@example.invalid")
	sourceGit(t, repo, "commit", "--allow-empty", "-m", "main")
	sourceGit(t, root, "init", "--bare", remote)
	sourceGit(t, repo, "remote", "add", "origin", remote)
	sourceGit(t, repo, "push", "origin", "main")
	sourceGit(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/main")
	return repo, remote
}

func TestSourceBranchesOrderAndExactRefs(t *testing.T) {
	repo, _ := sourceRepo(t)
	for _, branch := range []string{"develop", "feature/stack", "release", "fix/bug"} {
		sourceGit(t, repo, "branch", branch)
	}
	sourceGit(t, repo, "update-ref", "refs/remotes/origin/feature/remote-only", "HEAD")
	sourceGit(t, repo, "update-ref", "refs/remotes/upstream/develop", "HEAD")
	branches, err := SourceBranches(repo)
	if err != nil {
		t.Fatal(err)
	}
	var refs []string
	for _, b := range branches {
		refs = append(refs, b.Ref)
	}
	want := []string{"refs/heads/main", "refs/remotes/origin/main", "refs/heads/develop", "refs/remotes/upstream/develop", "refs/heads/release", "refs/remotes/origin/feature/remote-only", "refs/heads/feature/stack", "refs/heads/fix/bug"}
	if !reflect.DeepEqual(refs, want) {
		t.Fatalf("got %v, want %v", refs, want)
	}
	if !branches[0].Default || !branches[1].Default || branches[2].Default {
		t.Fatal(branches)
	}
	// A nonstandard default is promoted ahead of main and other unprefixed branches.
	sourceGit(t, repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/feature/remote-only")
	branches, err = SourceBranches(repo)
	if err != nil || branches[0].Name != "feature/remote-only" || !branches[0].Default {
		t.Fatal(branches, err)
	}
}

func TestCreateUsesSelectedSourceCommit(t *testing.T) {
	for _, remoteSource := range []bool{false, true} {
		t.Run(map[bool]string{false: "local-unpushed", true: "other-remote"}[remoteSource], func(t *testing.T) {
			repo, remote := sourceRepo(t)
			sourceGit(t, repo, "checkout", "-b", "feature/source")
			sourceGit(t, repo, "commit", "--allow-empty", "-m", "selected source")
			want := sourceGit(t, repo, "rev-parse", "HEAD")
			ref := "refs/heads/feature/source"
			if remoteSource {
				sourceGit(t, repo, "remote", "add", "upstream", remote)
				sourceGit(t, repo, "push", "upstream", "feature/source")
				ref = "refs/remotes/upstream/feature/source"
				// Keep a stale tracking ref to prove Create refreshes the selected remote.
				sourceGit(t, repo, "update-ref", ref, "main")
			}
			sourceGit(t, repo, "checkout", "main")
			if remoteSource {
				sourceGit(t, repo, "branch", "-D", "feature/source")
			}
			path := filepath.Join(t.TempDir(), "new-tree")
			if err := Create(repo, "fix/child", ref, path, io.Discard); err != nil {
				t.Fatal(err)
			}
			if got := sourceGit(t, path, "rev-parse", "HEAD"); got != want {
				t.Fatalf("HEAD=%s want=%s", got, want)
			}
			if got := sourceGit(t, repo, "branch", "--show-current"); got != "main" {
				t.Fatal("changed source checkout", got)
			}
		})
	}
}

func TestCreateRejectsMissingExplicitSource(t *testing.T) {
	repo, _ := sourceRepo(t)
	path := filepath.Join(t.TempDir(), "tree")
	if err := Create(repo, "fix/new", "refs/heads/missing", path, io.Discard); err == nil {
		t.Fatal("missing source accepted")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("created worktree for missing source", err)
	}
}
