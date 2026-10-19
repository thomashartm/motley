package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/thomashartm/motley/internal/shellx"
)

// OpenGhostty attaches a new macOS Ghostty window to an already-created
// session. A window failure must never cause the caller to spawn a second agent.
func OpenGhostty(id, worktree string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("opening a Ghostty window requires macOS")
	}
	bin, err := exec.LookPath("tmux")
	if err != nil {
		return err
	}
	// Ghostty is a separate process, potentially with a different TMUX value.
	// Resolve this session's socket before crossing the application boundary.
	socket, err := run("display-message", "-p", "-t", "="+SessionName(id)+":", "#{socket_path}")
	if err != nil {
		return err
	}
	socket = strings.TrimSpace(socket)
	if socket == "" {
		return fmt.Errorf("cannot find the tmux socket for %s", id)
	}
	command := "/usr/bin/env -u TMUX -u TMUX_PANE " + shellx.Quote(bin) + " -S " + shellx.Quote(socket) + " attach-session -t " + shellx.Quote("="+SessionName(id))
	// Values travel as argv, not AppleScript source. Shell quoting keeps paths
	// containing spaces, quotes or metacharacters literal in Ghostty's command.
	const script = `on run argv
	tell application "Ghostty"
		set cfg to new surface configuration
		set initial working directory of cfg to item 1 of argv
		set command of cfg to item 2 of argv
		set win to new window with configuration cfg
		activate window win
	end tell
end run
`
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "osascript", "-", worktree, command)
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("could not open Ghostty window (requires Ghostty 1.3+ and macOS Automation access): %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
