package member

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestManifestIssueRoundTripAndLegacy(t *testing.T) {
	dir := t.TempDir()
	m := Manifest{Schema: 1, ID: "412-fx", Name: "FX", Repo: "api", Branch: "feat/412-fx", Agent: "claude", CreatedAt: time.Unix(0, 0).UTC(),
		Issue: &IssueRef{Title: "Cache FX", URL: "https://github.com/o/r/issues/412"}}
	if err := saveManifest(dir, m); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, "412-fx.toml"))
	if !strings.Contains(string(data), "[issue]") || strings.Index(string(data), "[issue]") < strings.Index(string(data), "agent =") {
		t.Fatalf("issue table must follow scalar keys:\n%s", data)
	}
	got, err := Load(dir, "412-fx")
	if err != nil || got.Issue == nil || got.Issue.Title != "Cache FX" || got.Issue.URL != "https://github.com/o/r/issues/412" {
		t.Fatalf("%+v %v", got.Issue, err)
	}
	legacy := "schema = 1\nid = 'old'\nname = 'Old'\nagent = 'claude'\nfuture = 1\n"
	if err := os.WriteFile(filepath.Join(dir, "old.toml"), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	if old, err := Load(dir, "old"); err != nil || old.Issue != nil {
		t.Fatalf("legacy manifest: %+v %v", old, err)
	}
}

func TestImportedCodexManifestValidation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*Manifest)
		valid  bool
	}{
		{"valid", func(*Manifest) {}, true},
		{"wrong agent", func(m *Manifest) { m.Agent = "claude" }, false},
		{"missing thread", func(m *Manifest) { m.CodexSession = "" }, false},
		{"relative socket", func(m *Manifest) { m.CodexSocket = "relative" }, false},
		{"control socket", func(m *Manifest) { m.CodexSocket = "/tmp/\nsock" }, false},
		{"relative checkout", func(m *Manifest) { m.Worktree = "relative" }, false},
		{"mixed import", func(m *Manifest) { m.ClaudeSession = "claude-one" }, false},
		{"override server", func(m *Manifest) { m.AgentArgs = []string{"--no-daemon"} }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			m := Manifest{Schema: 1, ID: "codex-one", Agent: "codex", CodexSession: "one", CodexSocket: "/server.sock", Worktree: "/repo"}
			tc.change(&m)
			if err := saveManifest(dir, m); err != nil {
				t.Fatal(err)
			}
			got, err := Load(dir, m.ID)
			if (err == nil) != tc.valid {
				t.Fatal(got, err)
			}
			if tc.valid && !got.Imported() {
				t.Fatal("imported ownership lost")
			}
		})
	}
}

func TestLaunchStatus(t *testing.T) {
	for _, tt := range []struct {
		name   string
		m      Manifest
		resume bool
		want   string
	}{
		{"bare spawn waits for input", Manifest{}, false, "idle"},
		{"prompted spawn starts", Manifest{Prompt: true}, false, "starting"},
		{"blueprint spawn starts", Manifest{Blueprint: "plan"}, false, "starting"},
		{"revive skips the prompt", Manifest{Prompt: true}, true, "idle"},
	} {
		if got := tt.m.LaunchStatus(tt.resume); got != tt.want {
			t.Errorf("%s: got %q want %q", tt.name, got, tt.want)
		}
	}
}

func TestCheckCheckoutAllowsSwitchedBranchOnlyForImports(t *testing.T) {
	repo := t.TempDir()
	for _, args := range [][]string{{"init", "-q", "-b", "main"}, {"-c", "user.name=t", "-c", "user.email=t@t", "commit", "-q", "--allow-empty", "-m", "init"}, {"switch", "-q", "-c", "develop"}} {
		if out, err := exec.Command("git", append([]string{"-C", repo}, args...)...).CombinedOutput(); err != nil {
			t.Fatal(string(out), err)
		}
	}
	spawned := Manifest{ID: "spawned", RepoPath: repo, Worktree: repo, Branch: "main"}
	if err := spawned.CheckCheckout(); err == nil || !strings.Contains(err.Error(), "identity changed") {
		t.Fatal("spawned member accepted a switched main checkout:", err)
	}
	imported := spawned
	imported.CodexSession = "01a1658d-daa5-7d61-a44b-5e8a5274a453"
	if err := imported.CheckCheckout(); err != nil {
		t.Fatal("imported member refused its own checkout:", err)
	}
}
