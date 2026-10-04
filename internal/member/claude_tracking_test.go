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
		{"background working", "idle", "busy", "working", "F+B", true, true},
		{"foreground working", "busy", "idle", "working", "F+B", true, true},
		{"both idle", "idle", "idle", "idle", "F+B", true, true},
		{"background question", "busy", "waiting", "question", "F+B", true, true},
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

func TestSessionAccessIsIndependentOfExecutionMode(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		r          Row
	}{
		{"managed", "TMX", Row{Alive: true}},
		{"external", "EXT", Row{Alive: true, External: true}},
		{"stopped", "—", Row{External: true}},
		{"shared Codex with managed client", "TMX", Row{Manifest: Manifest{Agent: "codex", CodexSession: "shared"}, Alive: true}},
		{"mixed", "MIX", Row{ClaudeStatuses: []ClaudeSessionStatus{{Alive: true, Managed: true, Kind: "interactive"}, {Alive: true, Kind: "background"}}}},
		{"external foreground and background", "EXT", Row{ClaudeStatuses: []ClaudeSessionStatus{{Alive: true, Kind: "interactive"}, {Alive: true, Kind: "background"}}}},
		{"managed foreground ended", "EXT", Row{ClaudeStatuses: []ClaudeSessionStatus{{Managed: true, Kind: "interactive"}, {Alive: true, Kind: "background"}}}},
		{"external background ended", "TMX", Row{ClaudeStatuses: []ClaudeSessionStatus{{Alive: true, Managed: true, Kind: "interactive"}, {Kind: "background"}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.r.Access(); got != tc.want {
				t.Fatalf("access = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestClaudeRefreshAccessFromManagedTerminal(t *testing.T) {
	for _, external := range []bool{false, true} {
		r := Row{Manifest: Manifest{ID: "member", ClaudeSession: "fg", ClaudeSessions: []string{"bg"}, Worktree: "/repo"}, Alive: !external, External: external}
		sessions := []claude.Session{
			{SessionID: "fg", PID: 1, Cwd: "/repo", Kind: "interactive", Status: "idle"},
			{SessionID: "bg", PID: 2, Cwd: "/repo", Kind: "background", Status: "busy"},
		}
		if err := r.refreshClaudeSessions(t.TempDir(), sessions); err != nil {
			t.Fatal(err)
		}
		want, primary := "MIX", "TMX"
		if external {
			want, primary = "EXT", "EXT"
		}
		if r.Access() != want || r.ClaudeStatuses[0].Access() != primary || r.ClaudeStatuses[1].Access() != "EXT" || r.CurrentStatus() != "working" || r.SessionLocation() != "F+B" {
			t.Fatalf("wrong access or activity: %+v", r)
		}
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
