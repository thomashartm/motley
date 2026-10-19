package terminal

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/thomashartm/motley/internal/shellx"
)

// OpenGhostty starts a foreground runner in a new macOS Ghostty window.
func OpenGhostty(id, worktree string, resume bool) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("opening a Ghostty window requires macOS")
	}
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	command := "/usr/bin/env -u TMUX -u TMUX_PANE"
	// Finder-launched Ghostty need not share the launching terminal's environment.
	// Preserve agent configuration and hook/state discovery without importing tmux.
	for _, key := range []string{"PATH", "HOME", "SHELL", "XDG_RUNTIME_DIR", "XDG_STATE_HOME", "XDG_CONFIG_HOME", "XDG_DATA_HOME", "XDG_CACHE_HOME", "CODEX_HOME", "CLAUDE_CONFIG_DIR"} {
		if value, ok := os.LookupEnv(key); ok {
			command += " " + shellx.Quote(key+"="+value)
		}
	}
	command += " " + shellx.Quote("MOTLEY_MEMBER="+id) + " " + shellx.Quote(bin) + " run-direct " + shellx.Quote(id)
	if resume {
		command += " --resume"
	}
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
