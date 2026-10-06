package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/thomashartm/motley/internal/agents/claude"
	"github.com/thomashartm/motley/internal/member"
)

func TestReimportClaudeWorkspace(t *testing.T) {
	bin := buildLifecycleBinary(t)
	for _, kind := range []string{"linked", "non-git", "switched-primary"} {
		t.Run(kind, func(t *testing.T) {
			f := newMemberFixture(t, bin, "main")
			sid, process := fakeExternalClaude(t, f, f.repo)
			f.motley("import", sid, "--name", "Keep my name")
			id := "claude-" + sid
			dir := filepath.Join(f.state, "motley/members")
			original, err := member.Load(dir, id)
			if err != nil {
				t.Fatal(err)
			}
			history := filepath.Join(dir, id+".events.jsonl")
			writeFixture(t, history, "preserve history\n", 0600)
			cwd := filepath.Join(f.trees, "new-workspace")
			if kind == "non-git" {
				if err := os.MkdirAll(cwd, 0700); err != nil {
					t.Fatal(err)
				}
			} else {
				f.git(f.repo, "worktree", "add", "-b", "new-workspace", cwd)
			}
			if kind == "switched-primary" {
				sid = "22222222-2222-4222-8222-222222222222"
				data, _ := json.Marshal([]claude.Session{{SessionID: sid, PID: process.Process.Pid, Cwd: f.repo, Kind: "interactive", Status: "busy"}})
				writeFixture(t, filepath.Join(f.home, "sessions.json"), string(data), 0600)
				f.motley("import", sid, "--replace", id)
			}
			writeFixture(t, filepath.Join(cwd, "unfinished.txt"), "keep my work", 0600)
			data, _ := json.Marshal([]claude.Session{{SessionID: sid, PID: process.Process.Pid, Cwd: cwd, Kind: "interactive", Status: "busy", Name: "Moved session", StartedAt: 12345}})
			writeFixture(t, filepath.Join(f.home, "sessions.json"), string(data), 0600)
			rows, err := member.List()
			if err != nil || len(rows) != 1 || !rows[0].Alive || rows[0].CurrentStatus() != "moved" {
				t.Fatal(rows, err)
			}
			for _, args := range [][]string{{"import", "--list"}, {"import", "--replace", id, "--list"}} {
				out := f.motley(args...)
				if !strings.Contains(out, sid) || !strings.Contains(out, cwd) || !strings.Contains(out, "Reimport:") {
					t.Fatal(out)
				}
			}
			if got, err := member.AdditionalSessions(id); err != nil || len(got) != 0 {
				t.Fatal(got, err)
			}
			if err := member.Revive(id); err == nil || !strings.Contains(err.Error(), "Reimport session") {
				t.Fatal("missing recovery guidance", err)
			}
			if err := member.Terminate(id); err == nil {
				t.Fatal("terminated before workspace was revalidated")
			}
			oldPane := ""
			if kind == "linked" {
				f.tmux("new-session", "-d", "-s", id, "/bin/sh")
				f.tmux("set-option", "-t", "="+id+":", "@motley_member", id)
				oldPane = f.tmux("display-message", "-p", "-t", "="+id+":", "#{pane_id}")
			}
			// Exercise the normal picker route and the member-specific repair route.
			if kind == "non-git" {
				f.motley("import", sid, "--replace", id)
			} else {
				f.motley("import", sid)
			}
			if oldPane != "" {
				if got := f.tmux("display-message", "-p", "-t", oldPane, "#{@motley_member}|#{session_name}"); !strings.HasPrefix(got, "|untracked-") {
					t.Fatal("old terminal was not preserved and released", got)
				}
			}
			saved, err := member.Load(dir, id)
			if err != nil || saved.Worktree != realPath(t, cwd) || saved.Name != original.Name || !saved.CreatedAt.Equal(original.CreatedAt) || saved.ClaudeSession != sid {
				t.Fatal(saved, err)
			}
			if kind == "non-git" {
				if saved.RepoPath != "" || saved.Branch != "" || saved.Base != "" || saved.RemoteURL != "" {
					t.Fatal("stale Git metadata", saved)
				}
			} else if saved.Branch != "new-workspace" || saved.RepoPath != realPath(t, f.repo) {
				t.Fatal("wrong checkout metadata", saved)
			}
			if data, err := os.ReadFile(history); err != nil || string(data) != "preserve history\n" {
				t.Fatal(string(data), err)
			}
			if err := process.Process.Signal(syscall.Signal(0)); err != nil {
				t.Fatal("reimport stopped the conversation", err)
			}
			if _, err := os.Stat(filepath.Join(f.home, "resumed-args")); !os.IsNotExist(err) {
				t.Fatal("reimport started a second writer", err)
			}
			rows, err = member.List()
			if err != nil || len(rows) != 1 || rows[0].CurrentStatus() != "working" {
				t.Fatal(rows, err)
			}
			if got := f.motley("import", "--list"); strings.Contains(got, sid) {
				t.Fatal("repaired session offered twice", got)
			}
			routed, err := member.ImportedClaudeMember(sid, cwd)
			if err != nil || routed != id {
				t.Fatal("hooks did not follow the move", routed, err)
			}
			if kind == "non-git" {
				err = member.Retire(id, false, false)
			} else {
				err = member.Terminate(id)
			}
			if err != nil {
				t.Fatal("cleanup still blocked after reimport", err)
			}
			if _, err := member.Load(dir, id); err == nil {
				t.Fatal("old entry remained active")
			}
			if data, err := os.ReadFile(filepath.Join(cwd, "unfinished.txt")); err != nil || string(data) != "keep my work" {
				t.Fatal("cleanup lost work", err)
			}
			if data, err := os.ReadFile(filepath.Join(dir, "archive", id+".events.jsonl")); err != nil || string(data) != "preserve history\n" {
				t.Fatal("cleanup lost history", err)
			}
		})
	}
}

func TestReimportClaudeRejectsInvalidWorkspace(t *testing.T) {
	bin := buildLifecycleBinary(t)
	for _, kind := range []string{"missing", "relative", "protected", "foreign-terminal", "split-conversations"} {
		t.Run(kind, func(t *testing.T) {
			f := newMemberFixture(t, bin, "main")
			sid, process := fakeExternalClaude(t, f, f.repo)
			f.motley("import", sid)
			id := "claude-" + sid
			cwd := filepath.Join(f.trees, "moved")
			switch kind {
			case "relative":
				cwd = "relative"
			case "protected":
				f.git(f.repo, "worktree", "add", "--force", cwd, "main")
			case "foreign-terminal", "split-conversations":
				if err := os.MkdirAll(cwd, 0700); err != nil {
					t.Fatal(err)
				}
			}
			other := "22222222-2222-4222-8222-222222222222"
			if kind == "split-conversations" {
				data, _ := json.Marshal([]claude.Session{{SessionID: sid, PID: process.Process.Pid, Cwd: f.repo, Kind: "interactive"}, {SessionID: other, PID: process.Process.Pid, Cwd: f.repo, Kind: "interactive"}})
				writeFixture(t, filepath.Join(f.home, "sessions.json"), string(data), 0600)
				f.motley("import", other, "--with", id)
			}
			if kind == "foreign-terminal" {
				f.tmux("new-session", "-d", "-s", id, "/bin/sh")
				f.tmux("set-option", "-t", "="+id+":", "@motley_member", "someone-else")
			}
			path := filepath.Join(f.state, "motley/members", id+".toml")
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			sessions := []claude.Session{{SessionID: sid, PID: process.Process.Pid, Cwd: cwd, Kind: "interactive"}}
			if kind == "split-conversations" {
				sessions = append(sessions, claude.Session{SessionID: other, PID: process.Process.Pid, Cwd: f.repo, Kind: "interactive"})
			}
			data, _ := json.Marshal(sessions)
			writeFixture(t, filepath.Join(f.home, "sessions.json"), string(data), 0600)
			f.refused("import", sid)
			after, err := os.ReadFile(path)
			if err != nil || string(after) != string(before) {
				t.Fatal("failed recovery mutated member", err)
			}
			if err := process.Process.Signal(syscall.Signal(0)); err != nil {
				t.Fatal("failed recovery stopped process", err)
			}
		})
	}
}

func realPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestReimportClaudePickerTerminal(t *testing.T) {
	bin := buildLifecycleBinary(t)
	for _, route := range []string{"add", "details", "shortcut"} {
		t.Run(route, func(t *testing.T) {
			f := newMemberFixture(t, bin, "main")
			sid, process := fakeExternalClaude(t, f, f.repo)
			f.motley("import", sid, "--name", "Keep my name")
			data, _ := json.Marshal([]claude.Session{{SessionID: sid, PID: process.Process.Pid, Cwd: f.home, Kind: "interactive", Status: "busy", Name: "Moved Claude"}})
			writeFixture(t, filepath.Join(f.home, "sessions.json"), string(data), 0600)
			terminal := startTerminal(t, exec.Command(bin, "--monitor"))
			eventually(t, func() bool { return strings.Contains(terminal.text(), "[Reimport session (S)]") })
			switch route {
			case "add":
				terminal.send(t, "a")
				eventually(t, func() bool { return strings.Contains(terminal.text(), "Add existing agent") })
				terminal.send(t, "\r")
			case "details":
				terminal.send(t, "2")
				// Actual 120x30 terminal: click the visible Details control.
				terminal.send(t, "\x1b[<0;66;6M\x1b[<0;66;6m")
			case "shortcut":
				before := len(terminal.text())
				terminal.send(t, "2")
				eventually(t, func() bool { return len(terminal.text()) > before })
				terminal.send(t, "S")
			}
			eventually(t, func() bool { return strings.Contains(terminal.text(), "Reimport: Moved Claude") })
			terminal.send(t, "\r")
			eventually(t, func() bool { return strings.Contains(terminal.text(), "Reimported Keep my name") })
			rows, err := member.List()
			if err != nil || len(rows) != 1 || rows[0].Worktree != realPath(t, f.home) || !rows[0].Alive {
				t.Fatal(rows, err)
			}
		})
	}
}
