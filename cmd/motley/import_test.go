package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/thomashartm/motley/internal/agents/claude"
	"github.com/thomashartm/motley/internal/member"
)

// A harmless process stands in for a running Claude. All discovery/control stays
// in the disposable fixture; no real user sessions are imported or signalled.
func fakeExternalClaude(t *testing.T, f *memberFixture, cwd string) (string, *exec.Cmd) {
	t.Helper()
	process := exec.Command("sleep", "120")
	if err := process.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _ = process.Wait(); close(done) }()
	t.Cleanup(func() { _ = process.Process.Kill(); <-done })
	sid := "11111111-1111-4111-8111-111111111111"
	data, err := json.Marshal([]claude.Session{{SessionID: sid, PID: process.Process.Pid, Cwd: cwd, Kind: "interactive", Name: "Existing Claude", Status: "busy", StartedAt: 12345}})
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(f.home, "sessions.json"), string(data), 0600)
	script := fmt.Sprintf(`#!/bin/sh
if [ "$1" = agents ]; then
 if [ -f "$HOME/discovery-error" ]; then exit 1; fi
 if kill -0 %d 2>/dev/null; then cat "$HOME/sessions.json"; else echo '[]'; fi
else
 printf '%%s\n' "$@" > "$HOME/resumed-args"
fi
`, process.Process.Pid)
	writeFixture(t, filepath.Join(f.home, "fake agents", "claude"), script, 0755)
	return sid, process
}

func TestImportClaudeLifecycle(t *testing.T) {
	bin := buildLifecycleBinary(t)
	for _, kind := range []string{"main", "linked", "non-git"} {
		t.Run(kind, func(t *testing.T) {
			f := newMemberFixture(t, bin, "main")
			cwd := f.repo
			if kind == "linked" {
				cwd = filepath.Join(f.trees, "existing")
				f.git(f.repo, "worktree", "add", "-b", "existing", cwd)
			}
			if kind == "non-git" {
				cwd = filepath.Join(f.home, "notes")
				if err := os.MkdirAll(cwd, 0700); err != nil {
					t.Fatal(err)
				}
			}
			sid, process := fakeExternalClaude(t, f, cwd)
			writeFixture(t, filepath.Join(cwd, "unfinished.txt"), "keep this work", 0600)
			beforeTrees := f.git(f.repo, "worktree", "list", "--porcelain")
			beforeBranches := f.git(f.repo, "branch", "--list")
			if out := f.motley("import", "--list"); !strings.Contains(out, sid) {
				t.Fatal(out)
			}
			if out := f.motley("import", sid); !strings.Contains(out, "keeps running") {
				t.Fatal(out)
			}
			id := "claude-" + sid
			if err := process.Process.Signal(syscall.Signal(0)); err != nil {
				t.Fatal("import interrupted Claude", err)
			}
			if out := f.motley("import", "--list"); strings.Contains(out, sid) {
				t.Fatal("duplicate offered")
			}
			f.refused("import", sid)
			// Motley controls terminals only through tmux; it explains where the session runs.
			for _, command := range []string{"attach", "switch"} {
				if out := f.refused(command, id); !strings.Contains(out, "runs in its original terminal in ") || !strings.Contains(out, "stop it there and use Revive") {
					t.Fatal(command, out)
				}
			}
			assertListState(t, f.motley("ls"), id, "alive")
			rows, err := member.List()
			if err != nil || len(rows) != 1 || !rows[0].External || rows[0].CurrentStatus() != "working" {
				t.Fatal(rows, err)
			}
			if out := f.refused("revive", id); !strings.Contains(out, "still running") {
				t.Fatal(out)
			}
			// Stopping in the original terminal keeps the active entry for Revive.
			if err := process.Process.Signal(syscall.SIGTERM); err != nil {
				t.Fatal(err)
			}
			eventually(t, func() bool { return process.Process.Signal(syscall.Signal(0)) != nil })
			assertListState(t, f.motley("ls"), id, "dead")
			// A stale notification from a different conversation cannot override
			// the explicitly imported primary session on resume.
			writeFixture(t, filepath.Join(f.state, "motley/members", id+".events.jsonl"), `{"agent":"claude","event":"Notification","agent_session_id":"foreign-session"}`+"\n", 0600)
			f.motley("revive", id)
			eventually(t, func() bool {
				data, _ := os.ReadFile(filepath.Join(f.home, "resumed-args"))
				return string(data) == "--resume="+sid+"\n"
			})
			assertListState(t, f.motley("ls"), id, "alive")
			if err := member.Terminate(id); err != nil {
				t.Fatal(err)
			}
			if strings.Contains(f.motley("ls"), id) {
				t.Fatal("terminated import stayed in active list")
			}
			data, err := os.ReadFile(filepath.Join(cwd, "unfinished.txt"))
			if err != nil || string(data) != "keep this work" {
				t.Fatal("imported work deleted", err)
			}
			if beforeTrees != f.git(f.repo, "worktree", "list", "--porcelain") || beforeBranches != f.git(f.repo, "branch", "--list") {
				t.Fatal("imported Git checkout changed")
			}
		})
	}
}

func TestImportRetireRunningAndDiscoveryFailure(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	sid, _ := fakeExternalClaude(t, f, f.repo)
	f.motley("import", sid)
	id := "claude-" + sid
	writeFixture(t, filepath.Join(f.home, "discovery-error"), "fail", 0600)
	if err := member.Terminate(id); err == nil {
		t.Fatal("discovery failure authorized stopping")
	}
	if err := member.Retire(id, true, false); err == nil {
		t.Fatal("discovery failure authorized retirement")
	}
	if _, err := member.List(); err == nil {
		t.Fatal("discovery failure silently marked dead")
	}
	if err := os.Remove(filepath.Join(f.home, "discovery-error")); err != nil {
		t.Fatal(err)
	}
	assertListState(t, f.motley("ls"), id, "alive")
	f.motley("retire", id)
	if _, err := os.Stat(f.repo); err != nil {
		t.Fatal("retired main checkout", err)
	}
}

func TestImportPickerTerminal(t *testing.T) {
	bin := buildLifecycleBinary(t)
	f := newMemberFixture(t, bin, "main")
	sid, _ := fakeExternalClaude(t, f, f.repo)
	terminal := startTerminal(t, exec.Command(bin, "--monitor"))
	eventually(t, func() bool { return strings.Contains(terminal.text(), "[3 Actions]") })
	terminal.send(t, "a")
	eventually(t, func() bool { return strings.Contains(terminal.text(), "Add existing agent") })
	terminal.send(t, "\r")
	eventually(t, func() bool { return strings.Contains(terminal.text(), "Existing Claude") })
	terminal.send(t, "\r")
	eventually(t, func() bool { return strings.Contains(terminal.text(), "Added Existing Claude") })
	rows, err := member.List()
	if err != nil || len(rows) != 1 || rows[0].ClaudeSession != sid || !rows[0].Alive || !rows[0].External {
		t.Fatal("picker did not register running external session", rows, err)
	}
}

func TestSwitchTrackedClaudeAndSubagentReporting(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	sid, process := fakeExternalClaude(t, f, f.repo)
	f.motley("import", sid, "--name", "VAT work")
	id := "claude-" + sid
	// A detached old terminal must never freeze the imported member at idle.
	f.tmux("new-session", "-d", "-s", id, "/bin/sh")
	f.tmux("set-option", "-t", "="+id+":", "@motley_member", id)
	f.tmux("set-option", "-t", "="+id+":", "@motley_status", "idle")
	f.tmux("set-option", "-t", "="+id+":", "@motley_since", "1")
	rows, err := member.List()
	if err != nil || rows[0].CurrentStatus() != "working" || rows[0].Since != 0 {
		t.Fatal(rows, err)
	}
	next := "22222222-2222-4222-8222-222222222222"
	sessions := []claude.Session{
		{SessionID: sid, PID: process.Process.Pid, Cwd: f.repo, Kind: "interactive", Status: "idle"},
		{SessionID: next, PID: process.Process.Pid, Cwd: f.repo, Kind: "interactive", Status: "idle"},
		{SessionID: "elsewhere", PID: process.Process.Pid, Cwd: f.home, Kind: "interactive", Status: "busy"},
	}
	data, _ := json.Marshal(sessions)
	writeFixture(t, filepath.Join(f.home, "sessions.json"), string(data), 0600)
	if out := f.motley("import", "--replace", id, "--list"); !strings.Contains(out, next) || strings.Contains(out, "elsewhere") {
		t.Fatal(out)
	}
	f.refused("import", "elsewhere", "--replace", id)
	oldPane := f.tmux("display-message", "-p", "-t", "="+id+":", "#{pane_id}")
	f.motley("import", next, "--replace", id)
	if err := process.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatal("switch stopped agent", err)
	}
	if got := f.tmux("display-message", "-p", "-t", oldPane, "#{@motley_member}|#{session_name}"); !strings.HasPrefix(got, "|untracked-") {
		t.Fatal("former terminal not preserved and released", got)
	}
	dir := filepath.Join(f.state, "motley/members")
	saved, err := member.Load(dir, id)
	if err != nil || saved.ClaudeSession != next || saved.Name != "VAT work" {
		t.Fatal(saved, err)
	}
	// A late exit from the released pane must not end a subsequently created
	// Motley terminal with the same member ID.
	f.tmux("new-session", "-d", "-s", id, "/bin/sh")
	f.tmux("set-option", "-t", "="+id+":", "@motley_member", id)
	f.tmux("set-option", "-t", "="+id+":", "@motley_status", "working")
	oldEnv := os.Getenv("TMUX_PANE")
	t.Setenv("TMUX_PANE", oldPane)
	f.motley("agent-exited", id)
	t.Setenv("TMUX_PANE", oldEnv)
	if got := f.tmux("show-options", "-v", "-t", "="+id+":", "@motley_status"); got != "working" {
		t.Fatal("released pane ended replacement", got)
	}
	f.tmux("kill-session", "-t", "="+id+":")
	// Original terminal has no MOTLEY_MEMBER and no tmux session; hooks must
	// still reach exactly the selected session, including background children.
	report := func(event, want string) {
		t.Helper()
		payload := fmt.Sprintf(`{"session_id":%q,"cwd":%q,%s}`, next, f.repo, event)
		f.report("", payload)
		rows, err := member.List()
		if err != nil || len(rows) != 1 || !rows[0].External || rows[0].CurrentStatus() != want {
			t.Fatal(rows, err)
		}
	}
	// The discovery registry reports idle for the parent while a child runs.
	// Hook evidence must augment it without requiring a live tmux terminal.
	report(`"hook_event_name":"SubagentStart","agent_id":"reviewer"`, "working")
	report(`"hook_event_name":"Stop","last_assistant_message":"Parent waiting"`, "working")
	report(`"hook_event_name":"Notification","notification_type":"idle_prompt"`, "working")
	// Late hooks from the former session cannot change the chosen entry.
	f.report("", fmt.Sprintf(`{"session_id":%q,"cwd":%q,"hook_event_name":"Stop"}`, sid, f.repo))
	report(`"hook_event_name":"SubagentStop","agent_id":"reviewer"`, "idle")
	report(`"hook_event_name":"Stop","last_assistant_message":"Done"`, "ready")
	f.motley("import", next, "--replace", id)
}
