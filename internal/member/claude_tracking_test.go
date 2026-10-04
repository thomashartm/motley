package member

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/thomashartm/motley/internal/agents/claude"
)

func TestClaudeGroupStatus(t *testing.T) {
	for _, tc := range []struct {
		name, foreground, background, want, location string
		fgAlive, bgAlive                             bool
	}{
		{"background working", "idle", "busy", "working", "FG+BG", true, true},
		{"foreground working", "busy", "idle", "working", "FG+BG", true, true},
		{"both idle", "idle", "idle", "idle", "FG+BG", true, true},
		{"background question", "busy", "waiting", "question", "FG+BG", true, true},
		{"foreground gone", "idle", "busy", "working", "BG", false, true},
		{"background gone", "idle", "busy", "idle", "FG", true, false},
		{"both gone", "busy", "busy", "dead", "", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := Row{Manifest: Manifest{ID: "member", ClaudeSession: "fg", ClaudeSessions: []string{"bg"}, Worktree: "/repo"}}
			sessions := []claude.Session{{SessionID: "fg", Cwd: "/repo", Kind: "interactive", Status: tc.foreground}, {SessionID: "bg", Cwd: "/repo", Kind: "background", Status: tc.background}, {SessionID: "unrelated", PID: 123, Cwd: "/repo", Kind: "background", Status: "waiting"}}
			if tc.fgAlive {
				sessions[0].PID = 1
			}
			if tc.bgAlive {
				sessions[1].PID = 2
			}
			if err := r.refreshClaudeSessions(t.TempDir(), sessions); err != nil {
				t.Fatal(err)
			}
			if r.CurrentStatus() != tc.want || r.SessionLocation() != tc.location || len(r.ClaudeStatuses) != 2 {
				t.Fatalf("got %+v; want %s %s", r, tc.want, tc.location)
			}
		})
	}
}

func TestClaudeGroupPreservesBackgroundChildrenWhenForegroundRestarts(t *testing.T) {
	dir := t.TempDir()
	r := Row{Manifest: Manifest{ID: "member", ClaudeSession: "fg", ClaudeSessions: []string{"bg"}, Worktree: "/repo"}}
	var activity claude.Activity
	for _, payload := range []string{
		`{"hook_event_name":"SubagentStart","session_id":"bg","agent_id":"child"}`,
		`{"hook_event_name":"Stop","session_id":"bg"}`,
	} {
		e, err := claude.Parse([]byte(payload), "")
		if err != nil {
			t.Fatal(err)
		}
		e.TS = time.Now()
		activity.Apply(e)
	}
	data, _ := json.Marshal(activity)
	if err := os.WriteFile(r.ClaudeActivityPath(dir, "bg"), data, 0600); err != nil {
		t.Fatal(err)
	}
	sessions := []claude.Session{
		{SessionID: "fg", PID: 1, Cwd: "/repo", Kind: "interactive", Status: "idle", StartedAt: time.Now().UnixMilli()},
		{SessionID: "bg", PID: 2, Cwd: "/repo", Kind: "background", Status: "idle"},
	}
	if err := r.refreshClaudeSessions(dir, sessions); err != nil {
		t.Fatal(err)
	}
	if r.CurrentStatus() != "working" || r.ClaudeStatuses[0].Status != "idle" || r.ClaudeStatuses[1].Status != "working" {
		t.Fatal(r)
	}
	// Reusing the same session ID in another checkout cannot contribute status.
	sessions[1].Cwd = "/other"
	if err := r.refreshClaudeSessions(dir, sessions); err != nil {
		t.Fatal(err)
	}
	if r.CurrentStatus() != "idle" || r.SessionLocation() != "FG" {
		t.Fatal(r)
	}
}

func TestAdditionalClaudeManifestValidation(t *testing.T) {
	for _, ids := range [][]string{{"fg"}, {"bg", "bg"}, {"../escape"}, {""}} {
		dir := t.TempDir()
		m := Manifest{Schema: 1, ID: "member", Agent: "claude", ClaudeSession: "fg", ClaudeSessions: ids}
		if err := saveManifest(dir, m); err != nil {
			t.Fatal(err)
		}
		if _, err := Load(dir, m.ID); err == nil {
			t.Fatalf("accepted %q", ids)
		}
	}
}
