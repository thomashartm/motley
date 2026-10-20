package terminal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/thomashartm/motley/internal/shellx"
	"github.com/thomashartm/motley/internal/state"
)

// OpenGhostty starts a foreground runner in a tab, creating a window if needed.
func OpenGhostty(dir, id, worktree, title, window string, resume bool) error {
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
		set targetID to item 4 of argv
		if targetID is "new" or (targetID is "" and (count of windows) is 0) then
			set win to new window with configuration cfg
			set newTab to selected tab of win
		else
			if targetID is "" then
				set win to front window
			else
				set matches to every window whose id is targetID
				if (count of matches) is not 1 then error "The selected Ghostty window is no longer open; choose another window"
				set win to item 1 of matches
			end if
			set newTab to new tab in win with configuration cfg
		end if
		set term to focused terminal of newTab
		if not (perform action ("set_tab_title:" & item 3 of argv) on term) then error "Could not set Ghostty tab title"
		focus term
		return id of term
	end tell
end run
`
	out, err := runGhostty(script, "-", worktree, command, title, window)
	if err != nil {
		return err
	}
	terminalID := strings.TrimSpace(out)
	if terminalID == "" {
		return fmt.Errorf("opened Ghostty but received no terminal ID")
	}
	if err := state.WriteAtomic(filepath.Join(dir, id+".ghostty-terminal"), []byte(terminalID)); err != nil {
		return fmt.Errorf("opened Ghostty; could not save its terminal ID: %w", err)
	}
	return nil
}

// UpdateGhostty addresses only the terminal recorded at launch, never a title
// or working-directory match that could belong to another agent.
func UpdateGhostty(dir, id, title string, focus bool) error {
	data, err := os.ReadFile(filepath.Join(dir, id+".ghostty-terminal"))
	if os.IsNotExist(err) && !focus {
		return nil // Older direct sessions acquire an ID on their next launch.
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read Ghostty terminal for %s: %w", id, err)
	}
	if os.IsNotExist(err) || strings.TrimSpace(string(data)) == "" {
		return fmt.Errorf("no recorded Ghostty tab for %s; use its original window, or Revive after it stops", id)
	}
	const script = `on run argv
	if application "Ghostty" is not running then error "Ghostty is not running"
	tell application "Ghostty"
		set matches to every terminal whose id is (item 1 of argv)
		if (count of matches) is not 1 then error "The recorded Ghostty tab is no longer open"
		set term to item 1 of matches
		if not (perform action ("set_tab_title:" & item 2 of argv) on term) then error "Could not set Ghostty tab title"
		if item 3 of argv is "true" then focus term
	end tell
end run
`
	_, err = runGhostty(script, "-", strings.TrimSpace(string(data)), title, fmt.Sprint(focus))
	return err
}

type GhosttyWindow struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Tabs    int    `json:"tabs"`
	Current bool   `json:"current"`
}

func GhosttyWindows() ([]GhosttyWindow, error) {
	// JSON preserves window names containing quotes, tabs or newlines.
	const script = `const app = Application("Ghostty");
if (!app.running()) {
  JSON.stringify([]);
} else {
  const windows = app.windows();
  const front = windows.length ? app.frontWindow().id() : "";
  JSON.stringify(windows.map(w => ({id: w.id(), name: w.name(), tabs: w.tabs().length, current: w.id() === front})).sort((a, b) => Number(b.current) - Number(a.current)));
}`
	out, err := runGhostty(script, "-l", "JavaScript", "-")
	if err != nil {
		return nil, err
	}
	var windows []GhosttyWindow
	err = json.Unmarshal([]byte(out), &windows)
	return windows, err
}

func PreviewGhosttyWindow(id string) error {
	const script = `on run argv
	if application "Ghostty" is not running then error "Ghostty is not running"
	tell application "Ghostty"
		set matches to every window whose id is (item 1 of argv)
		if (count of matches) is not 1 then error "The selected Ghostty window is no longer open"
		activate window (item 1 of matches)
	end tell
end run`
	_, err := runGhostty(script, "-", id)
	return err
}

func runGhostty(script string, args ...string) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", fmt.Errorf("using Ghostty integration requires macOS")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "osascript", args...)
	cmd.Stdin = strings.NewReader(script)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("could not complete Ghostty operation (requires Ghostty 1.3+ and macOS Automation access): %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}
