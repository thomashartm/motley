package main

import (
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestSelectionCopiesRowsAndActionsThroughTmux(t *testing.T) {
	for _, mode := range []string{"overview", "monitor"} {
		t.Run(mode, func(t *testing.T) { testSelectionClipboard(t, mode == "monitor") })
	}
}

func testSelectionClipboard(t *testing.T, monitor bool) {
	t.Helper()
	bin := buildLifecycleBinary(t)
	f := newMemberFixture(t, bin, "main")
	f.keepAgentRunning("claude")
	title := "Clipboard fixture " + strings.Repeat("full title ", 12) + "END"
	f.motley("spawn", "--repo", "api", "--branch", "feat/copy", "--name", title, "--detach")
	f.tmux("set-option", "-s", "set-clipboard", "external")
	f.tmux("set-option", "-t", "=fixture:", "mouse", "on")
	terminal := f.terminalClient("fixture")
	target := "=fixture:"
	if monitor {
		f.motley("monitor")
		target = "=_motley:"
	} else {
		f.tmux("new-window", "-t", target, bin)
	}
	pane := func() string { return f.tmux("capture-pane", "-p", "-t", target) }
	defer func() {
		if t.Failed() {
			t.Log(pane())
			t.Log(f.tmux("show-options", "-t", target, "mouse"))
			t.Log(f.tmux("display-message", "-p", "-t", target, "#{pane_in_mode}|#{mouse_any_flag}|#{pane_current_command}"))
		}
	}()
	find := func(text string) (int, int) {
		t.Helper()
		var x, y int
		eventually(t, func() bool {
			for i, line := range strings.Split(pane(), "\n") {
				if at := strings.Index(line, text); at >= 0 {
					x, y = ansi.StringWidth(line[:at]), i
					return true
				}
			}
			return false
		})
		return x, y
	}
	drag := func(x, y, endX int) {
		t.Helper()
		terminal.send(t, fmt.Sprintf("\x1b[<0;%d;%dM\x1b[<32;%d;%dM\x1b[<0;%d;%dm", x+1, y+1, endX+1, y+1, endX+1, y+1))
		eventually(t, func() bool { return strings.Contains(pane(), "[CPY]") })
		terminal.send(t, "\x03")
		eventually(t, func() bool { return strings.Contains(pane(), "Copied") })
	}
	x, y := find("CC TMX")
	drag(x, y, x+1)
	copied := f.tmux("show-buffer")
	if !strings.Contains(copied, title) || strings.Contains(copied, "\x1b") {
		t.Fatalf("row clipboard lost full title or contains ANSI: %q", copied)
	}
	payload := base64.StdEncoding.EncodeToString([]byte(copied))
	eventually(t, func() bool { return strings.Contains(terminal.text(), payload+"\a") })
	terminal.send(t, "\x1b")
	eventually(t, func() bool { return !strings.Contains(pane(), "[CPY]") })
	terminal.send(t, "3")
	x, y = find("Retire member")
	drag(x, y, x+5)
	if got := f.tmux("show-buffer"); got != "Retire" {
		t.Fatalf("action text copy = %q", got)
	}
	if strings.Contains(pane(), "Retire this member?") {
		t.Fatal("text selection activated Retire")
	}
}

// The overview runs in a real tmux pane; a pty client stands in for Ghostty.
// tmux must forward the copy to that terminal as OSC 52.
func TestCopyMessageReachesTerminalClipboard(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "motley")
	commandOutput(t, "go", "build", "-o", bin, ".")
	f := newMemberFixture(t, bin, "main")
	f.tmux("set-option", "-s", "set-clipboard", "external")
	terminal := f.terminalClient("fixture")
	f.tmux("new-window", "-t", "=fixture:", bin)
	eventually(t, func() bool { return strings.Contains(terminal.text(), "No members yet") })
	terminal.send(t, "c")
	eventually(t, func() bool { return strings.Contains(terminal.text(), "No message to copy. [c copy]") })
	terminal.send(t, "c")
	sequence := "\x1b]52;"
	payload := base64.StdEncoding.EncodeToString([]byte("No message to copy."))
	eventually(t, func() bool {
		out := terminal.text()
		at := strings.LastIndex(out, sequence)
		return at >= 0 && strings.Contains(out[at:], payload+"\a") && strings.Contains(out, "✓ copied")
	})
	if got := f.tmux("show-buffer"); got != "No message to copy." {
		t.Fatalf("tmux buffer %q", got)
	}
	// Without forwarding, the copy must not claim the clipboard.
	f.tmux("set-option", "-s", "set-clipboard", "off")
	terminal.send(t, "c")
	eventually(t, func() bool {
		return strings.Contains(terminal.text(), "Copy failed: saved as a tmux paste buffer only; tmux set-clipboard is off")
	})
}
