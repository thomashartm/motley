package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/thomashartm/motley/internal/member"
	"github.com/thomashartm/motley/internal/state"
	"github.com/thomashartm/motley/internal/worktree"
)

func TestWorkspaceMismatchDoesNotPoisonResume(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	f.keepAgentRunning("claude")
	f.motley("spawn", "--repo", "api", "--branch", "feat/workspace", "--detach")
	id := "feat-workspace"
	m := f.manifest(id)
	path := filepath.Join(f.state, "motley/members", id+".events.jsonl")
	report := func(cwd, session string) {
		t.Helper()
		data, _ := json.Marshal(map[string]string{"hook_event_name": "SessionStart", "session_id": session, "cwd": cwd})
		f.report(id, string(data))
	}
	report(m.Worktree, "own-session")
	report(f.repo, "foreign-session")
	e, err := state.LatestEvent(path)
	if err != nil || e.Event != "WorkspaceMismatch" || !strings.Contains(e.Summary, f.repo) {
		t.Fatalf("mismatch not visible: %+v %v", e, err)
	}
	sid, err := state.LatestSessionID(path, "claude")
	if err != nil || sid != "own-session" {
		t.Fatalf("foreign session selected for resume: %s %v", sid, err)
	}
	link := filepath.Join(f.home, "workspace-link")
	if err := os.Symlink(m.Worktree, link); err != nil {
		t.Fatal(err)
	}
	report(link, "own-session")
	e, err = state.LatestEvent(path)
	if err != nil || e.Event != "SessionStart" {
		t.Fatalf("physical workspace alias rejected: %+v %v", e, err)
	}
}

func TestTerminateIgnoresUnrelatedDiscoveryFailure(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	f.motley("spawn", "--repo", "api", "--branch", "feat/stop", "--detach")
	sid, process := fakeExternalClaude(t, f, f.repo)
	f.motley("import", sid)
	writeFixture(t, filepath.Join(f.home, "discovery-error"), "fail", 0600)
	if err := member.Terminate("feat-stop"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(f.tmux("list-sessions", "-F", "#{session_name}"), "feat-stop") {
		t.Fatal("owned session survived termination")
	}
	if err := process.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatal("unrelated imported process stopped", err)
	}
}

func TestWorktreesRejectDefaultBranches(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "trunk")
	f.git(f.repo, "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/trunk")
	if base, err := worktree.Base(f.repo); err != nil || base != "trunk" {
		t.Fatal(base, err)
	}
	for _, branch := range []string{"main", "master", "develop", "trunk"} {
		if out := f.refused("spawn", "--repo", "api", "--branch", branch, "--detach"); !strings.Contains(out, "feature branch") {
			t.Fatal(branch, out)
		}
	}
	f.motley("spawn", "--repo", "api", "--branch", "feat/trunk", "--detach")
	m := f.manifest("feat-trunk")
	f.tmux("kill-session", "-t", "=feat-trunk")
	f.git(m.Worktree, "switch", "-c", "main")
	m.Branch = "main"
	// Validate the actual checkout even when the manifest matches its default.
	if _, err := worktree.Registered(f.repo, m.Worktree, m.Branch); err == nil || !strings.Contains(err.Error(), "feature branch") {
		t.Fatal("registered default branch accepted", err)
	}
	data, err := toml.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(f.state, "motley/members", m.ID+".toml"), string(data), 0600)
	for _, command := range []string{"revive", "exec-agent"} {
		if out := f.refused(command, m.ID); !strings.Contains(out, "feature branch") {
			t.Fatal(command, out)
		}
	}
	if err := os.Remove(filepath.Join(f.state, "motley/members", m.ID+".toml")); err != nil {
		t.Fatal(err)
	}
	sid, _ := fakeExternalClaude(t, f, m.Worktree)
	if out := f.refused("import", sid); !strings.Contains(out, "feature branch") {
		t.Fatal(out)
	}
}
