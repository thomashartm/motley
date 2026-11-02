package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/thomashartm/motley/internal/agents/codex"
	"github.com/thomashartm/motley/internal/member"
	"github.com/thomashartm/motley/internal/state"
)

type codexImportFixture struct {
	mu         sync.Mutex
	thread     codex.Session
	others     []codex.Session
	readErrors map[string]bool
	fail       bool
	methods    []string
	socket     string
}

func fakeExternalCodex(t *testing.T, f *memberFixture, cwd string) *codexImportFixture {
	t.Helper()
	dir, err := os.MkdirTemp("", "motley-cx-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	c := &codexImportFixture{socket: filepath.Join(dir, "rpc.sock")}
	c.thread = codex.Session{ID: "11111111-1111-4111-8111-111111111111", Name: "Existing Codex", Cwd: cwd}
	c.thread.Status.Type = "active"
	ln, err := net.Listen("unix", c.socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{ReadHeaderTimeout: time.Second, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		for {
			var req struct {
				ID     int             `json:"id"`
				Method string          `json:"method"`
				Params json.RawMessage `json:"params"`
			}
			if conn.ReadJSON(&req) != nil {
				return
			}
			if req.Method == "initialized" {
				continue
			}
			c.mu.Lock()
			c.methods = append(c.methods, req.Method)
			fail := c.fail
			var result any = map[string]any{}
			switch req.Method {
			case "initialize":
			case "thread/loaded/list":
				ids := []string{}
				if c.thread.Loaded() {
					ids = append(ids, c.thread.ID)
				}
				for _, s := range c.others {
					if s.Loaded() {
						ids = append(ids, s.ID)
					}
				}
				result = map[string]any{"data": ids}
			case "thread/read":
				var p struct {
					ThreadID     string `json:"threadId"`
					IncludeTurns bool   `json:"includeTurns"`
				}
				_ = json.Unmarshal(req.Params, &p)
				thread := c.thread
				for _, s := range c.others {
					if s.ID == p.ThreadID {
						thread = s
					}
				}
				if p.IncludeTurns || p.ThreadID != thread.ID || c.readErrors[p.ThreadID] {
					fail = true
				}
				result = map[string]any{"thread": thread}
			default:
				fail = true
				t.Errorf("unexpected/mutating RPC: %s", req.Method)
			}
			// Marshal while holding the lock, including the status flag slice.
			response := map[string]any{"id": req.ID, "result": result}
			if fail {
				response = map[string]any{"id": req.ID, "error": map[string]any{"code": -32000, "message": "fixture unavailable"}}
			}
			data, _ := json.Marshal(response)
			c.mu.Unlock()
			if conn.WriteMessage(websocket.TextMessage, data) != nil {
				return
			}
		}
	})}
	go func() { _ = server.Serve(ln) }()
	t.Cleanup(func() { _ = server.Close() })
	daemon, _ := json.Marshal(map[string]string{"status": "running", "socketPath": c.socket})
	script := `#!/bin/sh
if [ "$1" = app-server ] && [ "$2" = daemon ] && [ "$3" = version ]; then
 printf '%s\n' ` + quoteShell(string(daemon)) + `
else
 printf '%s\n' "$@" > "$HOME/resumed-codex-args"
 pwd -P > "$HOME/resumed-codex-cwd"
fi
`
	writeFixture(t, filepath.Join(f.home, "fake agents", "codex"), script, 0755)
	return c
}
func (c *codexImportFixture) change(fn func(*codex.Session)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn(&c.thread)
}

func TestImportCodexLifecycle(t *testing.T) {
	bin := buildLifecycleBinary(t)
	for _, kind := range []string{"main", "linked", "non-git"} {
		t.Run(kind, func(t *testing.T) {
			f := newMemberFixture(t, bin, "main")
			cwd := f.repo
			switch kind {
			case "linked":
				cwd = filepath.Join(f.trees, "existing")
				f.git(f.repo, "worktree", "add", "-b", "existing", cwd)
			case "non-git":
				cwd = filepath.Join(f.home, "notes")
				if err := os.MkdirAll(cwd, 0700); err != nil {
					t.Fatal(err)
				}
			}
			physical, err := filepath.EvalSymlinks(cwd)
			if err != nil {
				t.Fatal(err)
			}
			c := fakeExternalCodex(t, f, cwd)
			sid := c.thread.ID
			id := "codex-" + sid
			writeFixture(t, filepath.Join(cwd, "unfinished.txt"), "keep this work", 0600)
			beforeTrees, beforeBranches := f.git(f.repo, "worktree", "list", "--porcelain"), f.git(f.repo, "branch", "--list")
			if out := f.motley("import", "--agent", "codex", "--list"); !strings.Contains(out, sid) {
				t.Fatal(out)
			}
			f.motley("crew", "add", "--title", "Banking")
			f.motley("import", "--agent", "codex", sid, "--name", "Current task", "--crew", "banking")
			if _, err := os.Stat(filepath.Join(f.home, "resumed-codex-args")); !os.IsNotExist(err) {
				t.Fatal("import started another Codex client")
			}
			if out := f.motley("import", "--agent", "codex", "--list"); strings.Contains(out, sid) {
				t.Fatal("duplicate offered")
			}
			f.refused("import", "--agent", "codex", sid)
			rows, err := member.List()
			if err != nil || len(rows) != 1 || !rows[0].Alive || !rows[0].External || rows[0].CurrentStatus() != "working" || rows[0].Name != "Current task" || rows[0].Crew != "banking" {
				t.Fatal(rows, err)
			}
			if err := member.Terminate(id); err == nil {
				t.Fatal("shared daemon termination permitted")
			}
			c.change(func(s *codex.Session) { s.Cwd = filepath.Join(cwd, "changed") })
			if err := member.PrepareOpen(id); err == nil {
				t.Fatal("changed directory accepted")
			}
			c.change(func(s *codex.Session) { s.Cwd = cwd })
			f.motley("revive", id)
			want := []string{"resume", "--remote", "unix://" + c.socket, "--", sid}
			eventually(t, func() bool {
				b, _ := os.ReadFile(filepath.Join(f.home, "resumed-codex-args"))
				return reflect.DeepEqual(strings.Split(strings.TrimSpace(string(b)), "\n"), want)
			})
			if b, _ := os.ReadFile(filepath.Join(f.home, "resumed-codex-cwd")); strings.TrimSpace(string(b)) != physical {
				t.Fatal("Codex client started in wrong checkout", string(b))
			}
			if err := member.PrepareOpen(id); err != nil {
				t.Fatal("opening existing client failed", err)
			}
			// Shared Codex has no Motley hooks: recheck approvals on the server.
			f.tmux("set-option", "-t", "="+id+":", "@motley_status", "idle")
			c.change(func(s *codex.Session) { s.Status.ActiveFlags = []string{"waitingOnApproval"} })
			if err := member.Reply(id, "must not approve"); err == nil || !strings.Contains(err.Error(), "permission needs") {
				t.Fatal("reply ignored server approval state", err)
			}
			c.change(func(s *codex.Session) { s.Status.ActiveFlags = nil; s.Status.Type = "notLoaded" })
			if err := member.Reply(id, "must not send"); err == nil || !strings.Contains(err.Error(), "no longer loaded") {
				t.Fatal("reply accepted an unloaded thread", err)
			}
			c.change(func(s *codex.Session) { s.Status.Type = "active" })
			// Retiring a client preserves the shared server thread, files and Git identity.
			f.motley("retire", id, "--force")
			s, err := codex.ReadSession(c.socket, sid)
			if err != nil || !s.Loaded() {
				t.Fatal("retirement affected shared conversation", err)
			}
			if b, err := os.ReadFile(filepath.Join(cwd, "unfinished.txt")); err != nil || string(b) != "keep this work" {
				t.Fatal("checkout changed", err)
			}
			if beforeTrees != f.git(f.repo, "worktree", "list", "--porcelain") || beforeBranches != f.git(f.repo, "branch", "--list") {
				t.Fatal("Git state changed")
			}
		})
	}
}
func TestImportCodexStaleAndUnavailable(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	c := fakeExternalCodex(t, f, f.repo)
	sid := c.thread.ID
	id := "codex-" + sid
	f.refused("import", "--agent", "unknown", "--list")
	f.refused("import", "--agent", "codex", sid, "--crew", "missing")
	c.change(func(s *codex.Session) { s.Status.Type = "notLoaded" })
	f.refused("import", "--agent", "codex", sid)
	c.change(func(s *codex.Session) { s.Status.Type = "idle" })
	f.motley("import", "--agent", "codex", sid)
	c.mu.Lock()
	c.fail = true
	c.mu.Unlock()
	if _, err := member.List(); err == nil {
		t.Fatal("server error marked session dead")
	}
	if err := member.Revive(id); err == nil {
		t.Fatal("unavailable identity resumed")
	}
	c.mu.Lock()
	c.fail = false
	c.mu.Unlock()
	c.change(func(s *codex.Session) { s.Status.Type = "notLoaded" })
	assertListState(t, f.motley("ls"), id, "dead")
	f.motley("revive", id)
	eventually(t, func() bool {
		b, _ := os.ReadFile(filepath.Join(f.home, "resumed-codex-args"))
		return strings.Contains(string(b), "--remote\n") && strings.Contains(string(b), sid)
	})
	f.motley("retire", id)
}
func TestImportCodexPickerTerminal(t *testing.T) {
	bin := buildLifecycleBinary(t)
	f := newMemberFixture(t, bin, "main")
	c := fakeExternalCodex(t, f, f.repo)
	selected := c.thread
	selected.ID, selected.Name = "22222222-2222-4222-8222-222222222222", "Current Codex work"
	c.mu.Lock()
	c.thread.Status.Type = "idle"
	c.others = []codex.Session{selected}
	c.mu.Unlock()
	terminal := f.terminalClient("fixture")
	f.tmux("new-window", "-t", "=fixture:", bin)
	eventually(t, func() bool { return strings.Contains(terminal.text(), "[3 Actions]") })
	terminal.send(t, "a")
	eventually(t, func() bool { return strings.Contains(terminal.text(), "Add existing agent") })
	terminal.send(t, "\x1b[B\r")
	eventually(t, func() bool { return strings.Contains(terminal.text(), "Existing Codex") })
	terminal.send(t, "\x1b[B\r")
	eventually(t, func() bool { return strings.Contains(terminal.text(), "Added Current Codex work") })
	rows, err := member.List()
	if err != nil || len(rows) != 1 || rows[0].CodexSession != selected.ID || !rows[0].External || rows[0].CurrentStatus() != "working" {
		t.Fatal(fmt.Sprint(rows), err)
	}
	// The list reloads after the message; "o" without a selected row opens the picker.
	eventually(t, func() bool { return strings.Contains(terminal.text(), "External terminal or application") })
	terminal.send(t, "o")
	eventually(t, func() bool { return f.clientSession(f.clientName(terminal)) == "codex-"+selected.ID })
	eventually(t, func() bool {
		b, _ := os.ReadFile(filepath.Join(f.home, "resumed-codex-args"))
		return string(b) == "resume\n--remote\nunix://"+c.socket+"\n--\n"+selected.ID+"\n"
	})
}

func TestCodexRefreshDoesNotKeepTerminalAttention(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	c := fakeExternalCodex(t, f, f.repo)
	m := member.Manifest{ID: "codex-one", Agent: "codex", CodexSession: c.thread.ID, CodexSocket: c.socket, Worktree: f.repo}
	for _, tc := range []struct{ state, want string }{{"active", "working"}, {"idle", "idle"}, {"notLoaded", "dead"}, {"systemError", "unknown"}} {
		t.Run(tc.state, func(t *testing.T) {
			c.change(func(s *codex.Session) { s.Status.Type = tc.state })
			rows, err := member.RefreshExternal([]member.Row{{Manifest: m, Alive: true, Status: "ready", Since: 1}})
			if err != nil || rows[0].CurrentStatus() != tc.want {
				t.Fatalf("status = %s, err = %v; want %s", rows[0].CurrentStatus(), err, tc.want)
			}
		})
	}
}

func TestCodexRefreshIsolatesFailedThreads(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	c := fakeExternalCodex(t, f, f.repo)
	broken := c.thread
	broken.ID = "22222222-2222-4222-8222-222222222222"
	c.mu.Lock()
	c.others = []codex.Session{broken}
	c.readErrors = map[string]bool{broken.ID: true}
	c.mu.Unlock()
	makeRow := func(s codex.Session) member.Row {
		return member.Row{Manifest: member.Manifest{ID: "codex-" + s.ID, Agent: "codex", CodexSession: s.ID, CodexSocket: c.socket, Worktree: f.repo}, Alive: true, Status: "idle"}
	}
	// A broken unrelated thread must not prevent the tracked thread refreshing.
	rows, err := member.RefreshExternal([]member.Row{makeRow(c.thread)})
	if err == nil || rows[0].CurrentStatus() != "working" {
		t.Fatalf("unrelated thread froze refresh: %+v, %v", rows, err)
	}
	// A broken tracked thread must be unknown while the healthy one keeps updating.
	rows, err = member.RefreshExternal([]member.Row{makeRow(broken), makeRow(c.thread)})
	if err == nil || rows[0].CurrentStatus() != "unknown" || rows[1].CurrentStatus() != "working" {
		t.Fatalf("partial refresh: %+v, %v", rows, err)
	}
}

func TestImportCodexSelectedThreadIgnoresUnrelatedFailure(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	c := fakeExternalCodex(t, f, f.repo)
	unrelated := c.thread
	unrelated.ID = "22222222-2222-4222-8222-222222222222"
	c.mu.Lock()
	c.others = []codex.Session{unrelated}
	c.readErrors = map[string]bool{unrelated.ID: true}
	c.mu.Unlock()
	m, err := member.Import("codex", c.thread.ID, "", "")
	if err != nil || m.CodexSession != c.thread.ID {
		t.Fatalf("selected thread was not imported: %+v, %v", m, err)
	}
}

func TestCodexFollowsActiveConversationInWorktree(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	c := fakeExternalCodex(t, f, f.repo)
	m, err := member.Import("codex", c.thread.ID, "My worktree", "")
	if err != nil {
		t.Fatal(err)
	}
	f.motley("revive", m.ID)
	eventually(t, func() bool {
		b, _ := os.ReadFile(filepath.Join(f.home, "resumed-codex-args"))
		return strings.Contains(string(b), m.CodexSession)
	})
	next := c.thread
	next.ID, next.Name = "22222222-2222-4222-8222-222222222222", "Follow-up work"
	c.mu.Lock()
	c.thread.Status.Type = "idle"
	c.others = []codex.Session{next}
	c.mu.Unlock()
	rows, err := member.List()
	if err != nil || len(rows) != 1 || rows[0].CodexSession != next.ID || rows[0].CurrentStatus() != "working" || rows[0].ID != m.ID || rows[0].Name != "My worktree" {
		t.Fatalf("did not follow active conversation: %+v, %v", rows, err)
	}
	dir, _ := state.MembersDir()
	saved, err := member.Load(dir, m.ID)
	if err != nil || saved.CodexSession != next.ID {
		t.Fatalf("binding not persisted: %+v, %v", saved, err)
	}
	if sessions := f.tmux("list-sessions", "-F", "#{session_name}"); !strings.Contains(sessions, "untracked-"+m.ID) {
		t.Fatalf("old client was not preserved and released: %s", sessions)
	}
	f.motley("revive", m.ID)
	eventually(t, func() bool {
		b, _ := os.ReadFile(filepath.Join(f.home, "resumed-codex-args"))
		return strings.Contains(string(b), next.ID)
	})
}

func TestCodexDoesNotGuessBetweenActiveConversations(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	c := fakeExternalCodex(t, f, f.repo)
	m, err := member.Import("codex", c.thread.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	first, second := c.thread, c.thread
	first.ID = "22222222-2222-4222-8222-222222222222"
	second.ID = "33333333-3333-4333-8333-333333333333"
	c.mu.Lock()
	c.thread.Status.Type = "notLoaded"
	c.others = []codex.Session{first, second}
	c.mu.Unlock()
	rows, err := member.List()
	if err == nil || len(rows) != 1 || rows[0].CodexSession != m.CodexSession || rows[0].CurrentStatus() != "unknown" {
		t.Fatalf("ambiguous conversation was guessed: %+v, %v", rows, err)
	}
}

func TestCodexFollowsWorkingConversationWhilePreviousWaits(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	c := fakeExternalCodex(t, f, f.repo)
	m, err := member.Import("codex", c.thread.ID, "", "")
	if err != nil {
		t.Fatal(err)
	}
	next := c.thread
	next.ID = "22222222-2222-4222-8222-222222222222"
	c.mu.Lock()
	c.thread.Status.ActiveFlags = []string{"waitingOnUserInput"}
	c.others = []codex.Session{next}
	c.mu.Unlock()
	rows, err := member.List()
	if err != nil || rows[0].CodexSession != next.ID || rows[0].CurrentStatus() != "working" || rows[0].ID != m.ID {
		t.Fatalf("waiting conversation hid active work: %+v, %v", rows, err)
	}
}

func TestCodexFollowingGuards(t *testing.T) {
	bin := buildLifecycleBinary(t)
	for _, kind := range []string{"already working", "already tracked", "incomplete discovery", "other worktree", "subagent", "invalid ID", "unloaded"} {
		t.Run(kind, func(t *testing.T) {
			f := newMemberFixture(t, bin, "main")
			c := fakeExternalCodex(t, f, f.repo)
			m, err := member.Import("codex", c.thread.ID, "", "")
			if err != nil {
				t.Fatal(err)
			}
			next := c.thread
			next.ID = "22222222-2222-4222-8222-222222222222"
			c.mu.Lock()
			c.thread.Status.Type = "idle"
			switch kind {
			case "already working":
				c.thread.Status.Type = "active"
			case "incomplete discovery":
				c.readErrors = map[string]bool{"broken": true}
				c.others = append(c.others, codex.Session{ID: "broken"})
				c.others[0].Status.Type = "active"
			case "other worktree":
				next.Cwd = filepath.Join(f.repo, "other")
			case "subagent":
				next.ParentThreadID = m.CodexSession
			case "invalid ID":
				next.ID = "invalid/id"
			case "unloaded":
				c.thread.Status.Type = "notLoaded"
			}
			c.others = append(c.others, next)
			c.mu.Unlock()
			if kind == "already tracked" {
				if _, err := member.Import("codex", next.ID, "", ""); err != nil {
					t.Fatal(err)
				}
			}
			rows, err := member.List()
			want := m.CodexSession
			if kind == "unloaded" {
				want = next.ID
			}
			if (kind == "invalid ID" || kind == "incomplete discovery") != (err != nil) {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, r := range rows {
				if kind == "incomplete discovery" && r.ID == m.ID && r.CurrentStatus() != "unknown" {
					t.Fatalf("incomplete discovery retained stale attention: %+v", r)
				}
				if r.ID == m.ID && r.CodexSession != want {
					t.Fatalf("wrong conversation: %+v", r)
				}
			}
			dir, _ := state.MembersDir()
			saved, loadErr := member.Load(dir, m.ID)
			if loadErr != nil || saved.CodexSession != want {
				t.Fatalf("unsafe persisted binding: %+v, %v", saved, loadErr)
			}
		})
	}
}
