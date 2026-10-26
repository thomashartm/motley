package agents

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestStartArgv(t *testing.T) {
	prompt := "--literal\n$(touch nope) `pwd` \"quotes\""
	for _, name := range []string{"claude", "codex", "opencode"} {
		args := []string{"--model", "example"}
		before := append([]string(nil), args...)
		want := []string{name, "--model", "example", "--", prompt}
		if name == "codex" {
			want = []string{name, "--model", "example", "--no-daemon", "--", prompt}
		}
		if name == "opencode" {
			want = []string{name, "--model", "example", "--prompt=" + prompt}
		}
		if got := StartArgv(name, args, prompt); !reflect.DeepEqual(got, want) {
			t.Fatal(got, want)
		}
		if !reflect.DeepEqual(args, before) {
			t.Fatal("mutated arguments")
		}
		empty := []string{name}
		if name == "codex" {
			empty = append(empty, "--no-daemon")
		}
		if got := StartArgv(name, nil, ""); !reflect.DeepEqual(got, empty) {
			t.Fatal(got)
		}
	}
}

func TestResumeArgv(t *testing.T) {
	id := "--literal-id"
	args := []string{"--model", "fixture"}
	for name, want := range map[string][]string{
		"claude":   {"claude", "--model", "fixture", "--resume=" + id},
		"codex":    {"codex", "resume", "--model", "fixture", "--no-daemon", "--", id},
		"opencode": {"opencode", "--model", "fixture", "--session=" + id},
	} {
		if got := ResumeArgv(name, args, id); !reflect.DeepEqual(got, want) {
			t.Fatal(got, want)
		}
		fresh := []string{name, "--model", "fixture"}
		if name == "codex" {
			fresh = append(fresh, "--no-daemon")
		}
		if got := ResumeArgv(name, args, ""); !reflect.DeepEqual(got, fresh) {
			t.Fatal(got)
		}
	}
}

func TestCodexClientUsesOriginalServer(t *testing.T) {
	socket := "/path with spaces/server.sock"
	session := "existing-thread"
	want := []string{"codex", "resume", "--remote", "unix://" + socket, "--", session}
	if got := CodexClientArgv(socket, session); !reflect.DeepEqual(got, want) {
		t.Fatal(got, want)
	}
}

func TestHookMismatch(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	t.Setenv("PATH", dir)
	if got := HookMismatch(); !strings.Contains(got, "cannot find motley") {
		t.Fatal(got)
	}
	hook := filepath.Join(dir, "motley")
	// A byte-identical copy, as an install produces, is the same build.
	if err := os.WriteFile(hook, data, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := HookMismatch(); got != "" {
		t.Fatal(got)
	}
	data[len(data)-1] ^= 1
	if err := os.WriteFile(hook, data, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := HookMismatch(); !strings.Contains(got, hook) || !strings.Contains(got, self) {
		t.Fatal(got)
	}
}
