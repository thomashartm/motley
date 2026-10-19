package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// REQUIREMENTS §0: session control stays in tmux. The explicit new-window
// spawn action is the sole application-scripting exception.
func TestApplicationScriptingIsConfinedToGhosttyLaunch(t *testing.T) {
	forbidden := regexp.MustCompile(`(?i)osascript|applescript|tell application|NSAppleScript`)
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	self, err := filepath.Abs("policy_test.go")
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); path != root && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "dist" || name == "tasks") {
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(path) {
		case ".go", ".sh", ".ts", ".mjs", ".js", ".scpt", ".applescript":
		default:
			return nil
		}
		if path == self {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// Keep the exception in one launcher and its GUI-boundary tests;
		// imported-session focus and all other application scripting stay banned.
		switch filepath.ToSlash(rel) {
		case "internal/terminal/ghostty.go", "cmd/motley/ghostty_test.go":
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		checked++
		if match := forbidden.Find(data); match != nil {
			t.Errorf("%s uses %q; application scripting is restricted to the Ghostty new-window launcher", rel, match)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 50 {
		t.Fatalf("checked only %d files; the walk is not covering the repository", checked)
	}
}
