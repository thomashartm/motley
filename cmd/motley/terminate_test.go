package main

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/thomashartm/motley/internal/member"
)

func TestTerminateArchivesAndKeepsWork(t *testing.T) {
	bin := buildLifecycleBinary(t)
	for _, mode := range []string{"live", "stopped", "legacy-primary"} {
		t.Run(mode, func(t *testing.T) {
			f := newMemberFixture(t, bin, "main")
			f.keepAgentRunning("claude")
			f.motley("spawn", "--repo", "api", "--branch", "feat/keep", "--detach")
			id := "feat-keep"
			m := f.manifest(id)
			dir := filepath.Join(f.state, "motley/members")
			if mode != "live" {
				f.tmux("kill-session", "-t", "="+id)
			}
			if mode == "legacy-primary" {
				// Reproduce a stopped imported entry that lost its session ID.
				m.Worktree, m.Branch = f.repo, "main"
				data, _ := toml.Marshal(m)
				writeFixture(t, filepath.Join(dir, id+".toml"), string(data), 0600)
			}
			writeFixture(t, filepath.Join(m.Worktree, "unfinished.txt"), "keep dirty work", 0600)
			history := map[string]string{".events.jsonl": "{\"schema\":1,\"event\":\"Stop\"}\n", ".claude.json": "{\"session_id\":\"saved\"}\n", ".prompt.md": "original prompt\n"}
			for suffix, content := range history {
				writeFixture(t, filepath.Join(dir, id+suffix), content, 0600)
			}
			branches := f.git(f.repo, "branch", "--list")
			trees := f.git(f.repo, "worktree", "list", "--porcelain")
			if err := member.Terminate(id); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(f.motley("ls"), id) || strings.Contains(f.tmux("list-sessions", "-F", "#{session_name}"), id) {
				t.Fatal("terminated entry or session still active")
			}
			if branches != f.git(f.repo, "branch", "--list") || trees != f.git(f.repo, "worktree", "list", "--porcelain") {
				t.Fatal("termination changed Git worktrees or branches")
			}
			if data, err := os.ReadFile(filepath.Join(m.Worktree, "unfinished.txt")); err != nil || string(data) != "keep dirty work" {
				t.Fatal("termination lost dirty work", err)
			}
			archived, err := member.Load(filepath.Join(dir, "archive"), id)
			if err != nil || archived.RetiredAt == nil || archived.Worktree != m.Worktree {
				t.Fatal("manifest not archived", err)
			}
			for suffix, content := range history {
				if data, err := os.ReadFile(filepath.Join(dir, "archive", id+suffix)); err != nil || string(data) != content {
					t.Fatal("history not preserved", suffix, err)
				}
			}
		})
	}
}

func TestTerminateArchiveFailureCanBeRetried(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	id, process := fakeExternalClaude(t, f, f.repo)
	f.motley("import", id)
	id = "claude-" + id
	dir := filepath.Join(f.state, "motley/members")
	// A filesystem failure must retain the active manifest after stopping.
	archive := filepath.Join(dir, "archive")
	writeFixture(t, archive, "blocked", 0600)
	if err := member.Terminate(id); err == nil {
		t.Fatal("archive failure reported success")
	}
	if _, err := member.Load(dir, id); err != nil {
		t.Fatal("failed termination lost active manifest", err)
	}
	if err := process.Process.Signal(syscall.Signal(0)); err == nil {
		t.Fatal("external process survived termination")
	}
	if err := os.Remove(archive); err != nil {
		t.Fatal(err)
	}
	if err := member.Terminate(id); err != nil {
		t.Fatal("cannot retry after process stopped", err)
	}
	if _, err := member.Load(archive, id); err != nil {
		t.Fatal("retry did not archive", err)
	}
}
