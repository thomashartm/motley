package main

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestGlobalTemplateManagerTerminal(t *testing.T) {
	bin := buildLifecycleBinary(t)
	f := newMemberFixture(t, bin, "main")
	path := filepath.Join(f.home, "config/motley/blueprints/external.md")
	writeFixture(t, path, "+++\nname='external'\nvars=['task']\n+++\nOriginal {{.Vars.task}}\n", 0600)
	editor := filepath.Join(f.home, "template editor")
	script := "#!/bin/sh\nprintf '+++\\nname=\"external\"\\nvars=[\"task\"]\\n+++\\nEdited {{.Vars.task}}\\n' > \"$1\"\nprintf edited > \"$HOME/editor-called\"\n"
	writeFixture(t, editor, script, 0755)
	writeFixture(t, filepath.Join(f.home, "fake agents/vi"), script, 0755)
	t.Setenv("VISUAL", quoteShell(editor))
	f.tmux("set-environment", "-g", "PATH", os.Getenv("PATH"))
	f.tmux("set-environment", "-g", "VISUAL", os.Getenv("VISUAL"))
	f.tmux("set-option", "-s", "set-clipboard", "external")
	terminal := f.terminalClient("fixture")
	f.tmux("new-window", "-t", "=fixture:", bin)
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
	send("\x1b[H", "Prompt templates (f)")
	send("f", "Global prompt templates")
	eventually(t, func() bool { return strings.Contains(terminal.text(), "external · claude") })
	send("\r", "View raw template")
	send("\r", "Raw: external")
	send("\x1b", "Generate prompt")
	// Default editor returns to Motley and reloads the changed file.
	send("j", "> Generate prompt")
	send("j", "> Edit in vi")
	send("j", "> Edit in default editor")
	send("\r", "Editor opened")
	if data, err := os.ReadFile(path); err != nil || !strings.Contains(string(data), "Edited") {
		t.Fatal(string(data), err)
	}
	if err := os.Remove(filepath.Join(f.home, "editor-called")); err != nil {
		t.Fatal(err)
	}
	// The explicit vi choice invokes vi, independently of VISUAL.
	send("k", "> Edit in vi")
	send("\r", "Editor opened")
	if _, err := os.Stat(filepath.Join(f.home, "editor-called")); err != nil {
		t.Fatal("vi was not launched", err)
	}
	send("k\r", "Vars.task")
	send("One\rTwo\x13", "Prompt: external")
	prompt := "Edited One\nTwo\n"
	send("c", "Prompt copied")
	payload := base64.StdEncoding.EncodeToString([]byte(prompt))
	eventually(t, func() bool {
		out := terminal.text()
		at := strings.LastIndex(out, "\x1b]52;")
		return at >= 0 && strings.Contains(out[at:], payload+"\a")
	})
	if got := f.tmux("show-buffer"); got != strings.TrimSuffix(prompt, "\n") {
		t.Fatalf("clipboard prompt: %q", got)
	}
	manifests, err := filepath.Glob(filepath.Join(f.state, "motley/members/*.toml"))
	if err != nil || len(manifests) != 0 {
		t.Fatal("prompt generation created a member", manifests, err)
	}
}
