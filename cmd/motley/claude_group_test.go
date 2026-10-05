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

func TestTrackForegroundAndBackground(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	fg, process := fakeExternalClaude(t, f, f.repo)
	bg := "22222222-2222-4222-8222-222222222222"
	sessions := []claude.Session{
		{SessionID: fg, PID: process.Process.Pid, Cwd: f.repo, Kind: "interactive", Status: "idle"},
		{SessionID: bg, PID: process.Process.Pid, Cwd: f.repo, Kind: "background", Status: "busy"},
		{SessionID: "unrelated", PID: process.Process.Pid, Cwd: f.repo, Kind: "background", Status: "waiting"},
		{SessionID: "elsewhere", PID: process.Process.Pid, Cwd: f.home, Kind: "background", Status: "busy"},
	}
	writeSessions := func() {
		data, _ := json.Marshal(sessions)
		writeFixture(t, filepath.Join(f.home, "sessions.json"), string(data), 0600)
	}
	writeSessions()
	f.motley("import", fg, "--name", "IDD")
	id := "claude-" + fg
	f.refused("import", "elsewhere", "--with", id)
	f.refused("import", bg, "--with", id, "--replace", id)
	f.motley("import", bg, "--with", id)
	f.refused("import", bg)
	f.refused("import", bg, "--with", id)
	if out := f.motley("import", "--with", id, "--list"); strings.Contains(out, bg) {
		t.Fatal("linked session offered twice", out)
	}
	assertStatus := func(want string) {
		t.Helper()
		rows, err := member.List()
		if err != nil || len(rows) != 1 || rows[0].CurrentStatus() != want || rows[0].SessionLocation() != "F+B" {
			t.Fatalf("want %s F+B, got %+v: %v", want, rows, err)
		}
	}
	assertStatus("working")
	if out := f.motley("ls"); !strings.Contains(out, "F+B") {
		t.Fatal(out)
	}
	f.tmux("new-session", "-d", "-s", id, "/bin/sh")
	f.tmux("set-option", "-t", "="+id+":", "@motley_member", id)
	f.tmux("set-option", "-t", "="+id+":", "@motley_status", "idle")
	// A linked agent can inherit MOTLEY_MEMBER from a managed foreground. Its
	// hooks must still update only its own snapshot, not the foreground pane.
	f.report(id, fmt.Sprintf(`{"session_id":%q,"cwd":%q,"hook_event_name":"SubagentStart","agent_id":"inherited"}`, bg, f.repo))
	if got := f.tmux("show-options", "-v", "-t", "="+id+":", "@motley_status"); got != "idle" {
		t.Fatal("background hook overwrote foreground terminal", got)
	}
	report := func(sid, event string) {
		f.report("", fmt.Sprintf(`{"session_id":%q,"cwd":%q,%s}`, sid, f.repo, event))
	}
	report(bg, `"hook_event_name":"SubagentStart","agent_id":"review"`)
	report(bg, `"hook_event_name":"Stop","last_assistant_message":"Waiting for review"`)
	sessions[1].Status = "idle"
	writeSessions()
	report(fg, `"hook_event_name":"SessionStart","source":"resume"`)
	assertStatus("working")
	report(fg, `"hook_event_name":"PreToolUse","tool_name":"AskUserQuestion","tool_input":{"questions":[{"question":"Choose"}]}`)
	assertStatus("question")
	report(bg, `"hook_event_name":"PostToolUse","agent_id":"review","tool_name":"Read"`)
	assertStatus("question")
	report(fg, `"hook_event_name":"SessionStart","source":"resume"`)
	assertStatus("working")
	report(bg, `"hook_event_name":"SubagentStop","agent_id":"review","background_tasks":[]`)
	assertStatus("ready")
	report(bg, `"hook_event_name":"Notification","notification_type":"idle_prompt"`)
	assertStatus("idle")
	// Selecting another primary does not drop the explicitly linked foreground.
	f.motley("import", bg, "--replace", id)
	m := f.manifest(id)
	if m.ClaudeSession != bg || !m.TracksClaude(fg) || m.Name != "IDD" {
		t.Fatal(m)
	}
	if err := process.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatal("tracking stopped Claude", err)
	}
	report(fg, `"hook_event_name":"SubagentStart","agent_id":"other-review"`)
	assertStatus("working")
	// Both sessions are now gone. Retirement archives every session snapshot.
	sessions = nil
	writeSessions()
	f.motley("retire", id)
	dir := filepath.Join(f.state, "motley/members")
	for _, sid := range []string{fg, bg} {
		if _, err := os.Stat(filepath.Join(dir, "archive", id+"."+sid+".claude.json")); err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(filepath.Join(dir, id+"."+sid+".claude.json")); !os.IsNotExist(err) {
			t.Fatal("active snapshot retained", err)
		}
	}
}

func TestTerminateBothTrackedConversations(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	fg, foreground := fakeExternalClaude(t, f, f.repo)
	background := exec.Command("sleep", "120")
	if err := background.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _ = background.Wait(); close(done) }()
	t.Cleanup(func() { _ = background.Process.Kill(); <-done })
	bg := "22222222-2222-4222-8222-222222222222"
	fgData, _ := json.Marshal(claude.Session{SessionID: fg, PID: foreground.Process.Pid, Cwd: f.repo, Kind: "interactive", Status: "idle"})
	bgData, _ := json.Marshal(claude.Session{SessionID: bg, ID: "background-control", PID: background.Process.Pid, Cwd: f.repo, Kind: "background", Status: "busy"})
	writeFixture(t, filepath.Join(f.home, "fake agents", "claude"), fmt.Sprintf(`#!/bin/sh
if [ "$1" = agents ]; then
 printf '['
 separator=''
 if kill -0 %d 2>/dev/null; then printf '%%s' '%s'; separator=','; fi
 if kill -0 %d 2>/dev/null; then printf '%%s%%s' "$separator" '%s'; fi
 printf ']'
elif [ "$1 $2" = 'stop background-control' ]; then
 kill %d
else
 exit 1
fi
`, foreground.Process.Pid, fgData, background.Process.Pid, bgData, background.Process.Pid), 0755)
	f.motley("import", fg)
	id := "claude-" + fg
	f.motley("import", bg, "--with", id)
	if err := member.Terminate(id); err != nil {
		t.Fatal(err)
	}
	for _, process := range []*exec.Cmd{foreground, background} {
		if err := process.Process.Signal(syscall.Signal(0)); err == nil {
			t.Fatal("tracked process survived termination")
		}
	}
	rows, err := member.List()
	if err != nil || len(rows) != 0 {
		t.Fatal(rows, err)
	}
}
