package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/tmux"
)

func TestSpawnInGhosttyWindow(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Ghostty AppleScript is macOS only")
	}
	bin := buildLifecycleBinary(t)
	for _, failure := range []string{"", "window", "spawn"} {
		t.Run("failure="+failure, func(t *testing.T) {
			f := newMemberFixture(t, bin, "main")
			f.keepAgentRunning("claude")
			// Only the GUI boundary is substituted. The returned command is
			// executed below in a real terminal against the real spawned session.
			receipt := filepath.Join(f.home, "window-command")
			script := "#!/bin/sh\ncat > /dev/null\nprintf '%s' \"$2\" > \"$HOME/window-directory\"\nprintf '%s' \"$3\" > \"$HOME/window-command\"\n"
			if failure == "window" {
				script += "echo 'Automation permission denied' >&2\nexit 1\n"
			}
			writeFixture(t, filepath.Join(f.home, "fake agents/osascript"), script, 0755)
			terminal := startTerminal(t, exec.Command(bin, "--monitor"))
			defer func() {
				if t.Failed() {
					t.Log(ansi.Strip(terminal.text()))
				}
			}()
			send := func(keys, want string) {
				t.Helper()
				offset := len(terminal.text())
				terminal.send(t, keys)
				eventually(t, func() bool { return strings.Contains(ansi.Strip(terminal.text()[offset:]), want) })
			}
			eventually(t, func() bool { return strings.Contains(terminal.text(), "No members yet") })
			send("s", "> api")
			send("api\r", "Source branch")
			send("main\r", "Ticket and name")
			send("\tWindow test\r", "Agent")
			send("\r", "Blueprint")
			send("\r", "Authorization level")
			send("\r", "Prompt preview")
			send("\x1b[D", "Launch in new Ghostty window")
			if failure == "spawn" {
				// A conflicting branch created after the preview must prevent
				// both spawning and opening a window.
				f.git(f.repo, "branch", "feature/api-window-test")
				send("\r", "already exists")
				if _, err := os.Stat(receipt); !os.IsNotExist(err) {
					t.Fatal("window opened after a failed spawn", err)
				}
				return
			}
			send("\r", "Created feature-api-window-test")
			id := "feature-api-window-test"
			m := f.manifest(id)
			if f.git(m.Worktree, "branch", "--show-current") != "feature/api-window-test" {
				t.Fatal("wrong worktree")
			}
			if failure == "window" {
				eventually(t, func() bool {
					return strings.Contains(ansi.Strip(terminal.text()), "mtly attach "+id)
				})
				assertListState(t, f.motley("ls"), id, "alive")
				// The completed dialog closes; opening a new one must not
				// repeat the successful spawn after a window error.
				send("s", "Repository")
				if count := strings.Count(f.motley("ls"), id); count != 1 {
					t.Fatalf("window failure duplicated the member: %d", count)
				}
				return
			}
			command, err := os.ReadFile(receipt)
			if err != nil {
				t.Fatal(err)
			}
			dir, err := os.ReadFile(filepath.Join(f.home, "window-directory"))
			if err != nil || string(dir) != m.Worktree {
				t.Fatal("wrong Ghostty working directory", string(dir), err)
			}
			// A GUI process can inherit an unrelated/stale tmux environment.
			// The command must still attach to this fixture's exact server.
			t.Setenv("TMUX", "/nonexistent/server,1,0")
			t.Setenv("TERM", "xterm-256color")
			work := startTerminal(t, exec.Command("/bin/sh", "-c", string(command)))
			defer func() {
				if t.Failed() {
					t.Log("window terminal:", work.text(), "command:", string(command))
				}
			}()
			eventually(t, func() bool {
				return strings.Contains(f.tmux("list-clients", "-F", "#{session_name}"), id)
			})
			work.send(t, "\x02d")
			// The monitor remains usable after the new terminal attaches.
			send("s", "Repository")
		})
	}
}

// Opt in on a desktop: this creates one real Ghostty window and detaches only
// its fixture session during cleanup. It never invokes a paid agent.
func TestGhosttyLiveWindow(t *testing.T) {
	if runtime.GOOS != "darwin" || os.Getenv("MOTLEY_TEST_GHOSTTY") != "1" {
		t.Skip("set MOTLEY_TEST_GHOSTTY=1 on macOS to open a real Ghostty window")
	}
	countWindows := func() int {
		t.Helper()
		n, err := strconv.Atoi(commandOutput(t, "osascript", "-e", `tell application "Ghostty" to count windows`))
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := countWindows()
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	f.keepAgentRunning("claude")
	f.motley("spawn", "--repo", "api", "--branch", "feature/ghostty-smoke", "--detach")
	id := "feature-ghostty-smoke"
	m := f.manifest(id)
	pid := f.tmux("list-panes", "-t", "="+id+":", "-F", "#{pane_pid}")
	if n, err := strconv.Atoi(pid); err != nil || n <= 0 {
		t.Fatal("missing original agent pane PID", pid, err)
	}
	if err := tmux.OpenGhostty(id, m.Worktree); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = exec.Command(f.tmuxBin, "-L", f.socket, "detach-client", "-s", "="+id).Run() })
	eventually(t, func() bool {
		return strings.Contains(f.tmux("list-clients", "-F", "#{session_name}"), id)
	})
	if countWindows() != before+1 {
		t.Fatal("expected exactly one new Ghostty window")
	}
	if f.tmux("list-panes", "-t", "="+id+":", "-F", "#{pane_pid}") != pid {
		t.Fatal("opening the window restarted the agent pane")
	}
	t.Log("new Ghostty window attached to", id, "without replacing pane", pid)
}
