package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/member"
	"github.com/thomashartm/motley/internal/state"
	"github.com/thomashartm/motley/internal/terminal"
)

func fakeGhostty(t *testing.T, f *memberFixture, fail bool) {
	t.Helper()
	script := "#!/bin/sh\ncat > /dev/null\nprintf '%s' \"$2\" > \"$HOME/window-directory\"\nprintf '%s' \"$3\" > \"$HOME/window-command\"\n"
	if fail {
		script += "echo 'Automation permission denied' >&2\nexit 1\n"
	}
	writeFixture(t, filepath.Join(f.home, "fake agents/osascript"), script, 0755)
	// Fail loudly if direct creation, hooks, status or lifecycle touches tmux.
	writeFixture(t, filepath.Join(f.home, "fake agents/tmux"), "#!/bin/sh\necho called >> \"$HOME/tmux-called\"\nexit 93\n", 0755)
}

func windowCommand(t *testing.T, f *memberFixture) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.home, "window-command"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestDirectGhosttyLifecycle(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Ghostty window creation requires macOS")
	}
	bin := buildLifecycleBinary(t)
	for _, agent := range []string{"claude", "codex", "opencode"} {
		t.Run(agent, func(t *testing.T) {
			f := newMemberFixture(t, bin, "main")
			fakeGhostty(t, f, false)
			t.Setenv("COLORFGBG", "15;0")
			writeFixture(t, filepath.Join(f.home, "fake agents", agent), "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$MOTLEY_MEMBER\" \"${TMUX-unset}\" \"${TMUX_PANE-unset}\" \"$@\" > \"$HOME/agent-args\"\nprintf 'DIRECT_READY\\n'\nwhile IFS= read -r line; do [ \"$line\" = quit ] && exit 0; done\n", 0755)
			writeFixture(t, filepath.Join(f.repo, ".motley/blueprints/direct.md"), "+++\nagent='"+agent+"'\nargs=['--model','literal model']\n+++\nA prompt with $(literal) and 'quotes'\n", 0600)
			id := "feature-direct-" + agent
			f.motley("spawn", "--repo", "api", "--branch", "feature/direct-"+agent, "--agent", agent, "--terminal", "ghostty", "--no-gh", "--blueprint", "direct", "--mode", "auto")
			m := f.manifest(id)
			if m.LaunchTarget != "ghostty" || f.git(m.Worktree, "branch", "--show-current") != m.Branch {
				t.Fatalf("wrong manifest: %+v", m)
			}
			command := windowCommand(t, f)
			if strings.Contains(command, "attach-session") || !strings.Contains(command, "run-direct") {
				t.Fatal(command)
			}
			directory, _ := os.ReadFile(filepath.Join(f.home, "window-directory"))
			if string(directory) != m.Worktree {
				t.Fatal("wrong Ghostty cwd", string(directory))
			}
			work := startTerminal(t, exec.Command("/bin/sh", "-c", command))
			defer func() {
				if t.Failed() {
					t.Log("direct terminal:", work.text(), "command:", command)
				}
			}()
			waitDirectReady(t, work)
			receipt, _ := os.ReadFile(filepath.Join(f.home, "agent-args"))
			if !strings.Contains(string(receipt), "\n"+id+"\nunset\nunset\n") {
				t.Fatal("wrong direct environment", string(receipt))
			}
			if !strings.Contains(string(receipt), "literal model") || !strings.Contains(string(receipt), "A prompt with $(literal)") || m.Mode != "auto" {
				t.Fatalf("lost blueprint or authorization: %s %+v", receipt, m)
			}
			assertListState(t, f.motley("ls"), id, "alive")
			if out, err := exec.Command(bin, "run-direct", id).CombinedOutput(); err == nil || !strings.Contains(string(out), "already runs") {
				t.Fatalf("duplicate runner: %s %v", out, err)
			}
			payload := `{"hook_event_name":"UserPromptSubmit","session_id":"direct-session","prompt":"Test direct status"}`
			if agent == "opencode" {
				payload = `{"type":"session.status","properties":{"sessionID":"direct-session","status":{"type":"busy"}}}`
			}
			f.report(id, payload, "--agent", agent)
			dir := filepath.Join(f.state, "motley/members")
			status, err := terminal.ReadStatus(dir, id)
			if err != nil || status.Status != "working" {
				t.Fatalf("hook status: %+v %v", status, err)
			}
			events, err := state.Tail(filepath.Join(dir, id+".events.jsonl"), 4096)
			if err != nil || !strings.Contains(string(events), "direct-session") {
				t.Fatalf("events: %s %v", events, err)
			}
			for _, args := range [][]string{{"retire", id, "--force"}, {"revive", id}, {"attach", id}, {"switch", id}} {
				out, err := exec.Command(bin, args...).CombinedOutput()
				if err == nil || !strings.Contains(string(out), "Ghostty") {
					t.Fatalf("unsafe action %v: %s %v", args, out, err)
				}
			}
			if err := member.Terminate(id); err == nil || !strings.Contains(err.Error(), "Ghostty") {
				t.Fatal("terminate live", err)
			}
			work.send(t, "quit\n")
			eventually(t, func() bool { alive, _ := terminal.Alive(dir, id); return !alive })
			assertListState(t, f.motley("ls"), id, "dead")
			f.motley("revive", id)
			command = windowCommand(t, f)
			if !strings.Contains(command, "--resume") {
				t.Fatal("revive lost target", command)
			}
			resumed := startTerminal(t, exec.Command("/bin/sh", "-c", command))
			waitDirectReady(t, resumed)
			receipt, _ = os.ReadFile(filepath.Join(f.home, "agent-args"))
			if !strings.Contains(string(receipt), "direct-session") || strings.Contains(string(receipt), "A prompt with") {
				t.Fatal("resume did not use recorded conversation", string(receipt))
			}
			resumed.send(t, "quit\n")
			eventually(t, func() bool { alive, _ := terminal.Alive(dir, id); return !alive })
			f.motley("retire", id, "--force")
			if _, err := os.Stat(m.Worktree); !os.IsNotExist(err) {
				t.Fatal("retired worktree retained", err)
			}
			if _, err := os.Stat(filepath.Join(f.home, "tmux-called")); !os.IsNotExist(err) {
				t.Fatal("direct flow invoked tmux", err)
			}
		})
	}
}

func TestGhosttyFailureRetainsSingleMember(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Ghostty requires macOS")
	}
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	fakeGhostty(t, f, true)
	writeFixture(t, filepath.Join(f.repo, ".motley/blueprints/reviewed.md"), "+++\n+++\nReviewed first prompt\n", 0600)
	writeFixture(t, filepath.Join(f.home, "fake agents/claude"), "#!/bin/sh\nprintf '%s\n' \"$@\" > \"$HOME/retried-prompt\"\nprintf 'DIRECT_READY\n'\n", 0755)
	out, err := exec.Command(f.bin, "spawn", "--repo", "api", "--branch", "feature/window-failure", "--terminal", "ghostty", "--no-gh", "--blueprint", "reviewed").CombinedOutput()
	if err == nil || !strings.Contains(string(out), "worktree and manifest retained") {
		t.Fatalf("%s %v", out, err)
	}
	m := f.manifest("feature-window-failure")
	if _, err := os.Stat(m.Worktree); err != nil {
		t.Fatal(err)
	}
	rows, err := member.List()
	if err != nil || len(rows) != 1 || rows[0].Alive {
		t.Fatalf("retained rows: %+v %v", rows, err)
	}
	fakeGhostty(t, f, false)
	f.motley("revive", m.ID)
	command := windowCommand(t, f)
	if strings.Contains(command, "--resume") {
		t.Fatal("first launch retry lost the reviewed prompt", command)
	}
	retry := startTerminal(t, exec.Command("/bin/sh", "-c", command))
	waitDirectReady(t, retry)
	receipt, err := os.ReadFile(filepath.Join(f.home, "retried-prompt"))
	if err != nil || !strings.Contains(string(receipt), "Reviewed first prompt") {
		t.Fatalf("retry prompt: %s %v", receipt, err)
	}
}

// Opt in on a desktop. A fake local agent writes a receipt and exits, so the
// test never invokes a paid agent or leaves a running fixture session behind.
func TestGhosttyLiveWindow(t *testing.T) {
	if runtime.GOOS != "darwin" || os.Getenv("MOTLEY_TEST_GHOSTTY") != "1" {
		t.Skip("set MOTLEY_TEST_GHOSTTY=1 on macOS to open a real Ghostty window")
	}
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	writeFixture(t, filepath.Join(f.home, "fake agents/tmux"), "#!/bin/sh\necho called >> \"$HOME/tmux-called\"\nexit 93\n", 0755)
	f.motley("spawn", "--repo", "api", "--branch", "feature/ghostty-smoke", "--terminal", "ghostty", "--no-gh")
	id := "feature-ghostty-smoke"
	marker := filepath.Join(f.home, "agent-"+id+".txt")
	eventually(t, func() bool { _, err := os.Stat(marker); return err == nil })
	receipt, _ := os.ReadFile(marker)
	if !strings.Contains(string(receipt), id) {
		t.Fatal(string(receipt))
	}
	if _, err := os.Stat(filepath.Join(f.home, "tmux-called")); !os.IsNotExist(err) {
		t.Fatal("live launch called tmux", err)
	}
	data, _ := json.Marshal(f.manifest(id))
	t.Log("Real Ghostty ran the direct fixture agent:", string(data), "PID", strconv.Itoa(os.Getpid()))
}

// The bare test PTY has no emulator to answer the UI dependency's OSC probe.
func waitDirectReady(t *testing.T, p *terminalProcess) {
	answered := 0
	eventually(t, func() bool {
		queries := strings.Count(p.text(), "\x1b[6n")
		for answered < queries {
			p.send(t, "\x1b]11;rgb:0000/0000/0000\a\x1b[1;1R")
			answered++
		}
		return strings.Contains(p.text(), "DIRECT_READY")
	})
}

func TestMainMenuStartsDirectGhostty(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Ghostty requires macOS")
	}
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	fakeGhostty(t, f, false)
	t.Setenv("TMUX", "")
	t.Setenv("TMUX_PANE", "")
	overview := startTerminal(t, exec.Command(f.bin))
	defer func() {
		if t.Failed() {
			t.Log(ansi.Strip(overview.text()))
		}
	}()
	answered := 0
	waitText := func(offset int, want string) {
		t.Helper()
		eventually(t, func() bool {
			queries := strings.Count(overview.text(), "\x1b[6n")
			for answered < queries {
				overview.send(t, "\x1b]11;rgb:0000/0000/0000\a\x1b[1;1R")
				answered++
			}
			return strings.Contains(ansi.Strip(overview.text()[offset:]), want)
		})
	}
	send := func(keys, want string) {
		t.Helper()
		offset := len(overview.text())
		overview.send(t, keys)
		waitText(offset, want)
	}
	waitText(0, "No members yet")
	send("s", "Agent")
	send("\r", "Launch destination")
	send("\r", "> api")
	send("api\r", "Source branch")
	send("main\r", "Ticket and name")
	send("\tDirect menu\r", "Blueprint")
	send("\r", "Authorization level")
	send("\r", "Prompt preview")
	send("\r", "Created feature-api-direct-menu")
	m := f.manifest("feature-api-direct-menu")
	if m.Agent != "claude" || m.LaunchTarget != "ghostty" || f.git(m.Worktree, "branch", "--show-current") != m.Branch {
		t.Fatalf("wrong menu launch %+v", m)
	}
	if !strings.Contains(windowCommand(t, f), "run-direct") {
		t.Fatal("menu did not open direct runner")
	}
	if _, err := os.Stat(filepath.Join(f.home, "tmux-called")); !os.IsNotExist(err) {
		t.Fatal("direct menu called tmux", err)
	}
}
