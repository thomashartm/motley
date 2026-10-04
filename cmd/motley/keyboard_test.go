package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Exercise real terminal input, including a pane started before configuration
// was loaded. No live agent receives these keys.
func TestShiftEnterSurvivesTmux(t *testing.T) {
	f := newMemberFixture(t, buildLifecycleBinary(t), "main")
	terminal := f.terminalClient("fixture")
	received := filepath.Join(f.home, "received keys")
	f.tmux("new-window", "-t", "=fixture:", "/bin/sh", "-c", "stty raw -echo; printf KEYBOARD_READY; exec cat > "+quoteShell(received))
	eventually(t, func() bool { return strings.Contains(terminal.text(), "KEYBOARD_READY") })
	defer func() {
		if t.Failed() {
			data, _ := os.ReadFile(received)
			t.Logf("received = %q", data)
			t.Log(f.tmux("show-options", "-s", "extended-keys"))
		}
	}()
	want := ""
	modern := f.tmux("display-message", "-p", "#{>=:#{version},3.5}") == "1"
	plainShiftEnter := ""
	if modern {
		plainShiftEnter = "\r"
	}
	send := func(input, output string) {
		t.Helper()
		want += output
		terminal.send(t, input)
		eventually(t, func() bool {
			data, err := os.ReadFile(received)
			return err == nil && string(data) == want
		})
	}
	// Older tmux drops unbound Shift+Enter; 3.5+ falls back to plain Enter.
	// A following marker proves the input was processed even if no key arrives.
	f.tmux("set-option", "-s", "extended-keys", "off")
	send("\x1b[13;2ubaseline", plainShiftEnter+"baseline")
	f.motley("init")
	config := filepath.Join(f.home, "config/motley/motley.tmux.conf")
	f.tmux("source-file", config)
	f.tmux("source-file", config)
	if f.tmux("show-options", "-sv", "extended-keys") != "on" {
		t.Fatal("extended-key configuration was not applied")
	}
	if modern && f.tmux("show-options", "-sv", "extended-keys-format") != "csi-u" {
		t.Fatal("modern tmux did not select CSI-u")
	}
	// The byte receiver stands in for a managed agent that missed negotiation.
	f.tmux("set-option", "-t", "=fixture:", "@motley_member", "keyboard-fixture")
	f.tmux("set-option", "-t", "=fixture:", "@motley_status", "working")
	// Ghostty can send either encoding depending on negotiated keyboard mode.
	// The application must receive Shift+Enter, distinct from plain Enter.
	send("\x1b[13;2u", "\x1b[13;2u")
	send("\x1b[27;2;13~", "\x1b[13;2u")
	send("\r", "\r")
	send("\x03\t\x1b[Atext", "\x03\t\x1b[Atext")
	// Terminal features are discovered on attachment. Reconnect the client,
	// keeping the same running pane, to enable Ghostty's key negotiation too.
	pid := f.tmux("display-message", "-p", "-t", "=fixture:", "#{pane_pid}")
	f.tmux("detach-client", "-t", f.clientName(terminal))
	terminal = f.terminalClient("fixture")
	if !strings.Contains(f.tmux("list-clients", "-F", "#{client_termfeatures}"), "extkeys") || f.tmux("display-message", "-p", "-t", "=fixture:", "#{pane_pid}") != pid {
		t.Fatal("reattachment lost terminal features or restarted the pane")
	}
	send("\x1b[27;2;13~\r", "\x1b[13;2u\r")
	// A shell left behind after the agent ends must keep normal key handling.
	f.tmux("set-option", "-t", "=fixture:", "@motley_status", "ended")
	send("\x1b[13;2uended", plainShiftEnter+"ended")
	f.tmux("set-option", "-ut", "=fixture:", "@motley_member")
	send("\x1b[13;2ushell", plainShiftEnter+"shell")
}
