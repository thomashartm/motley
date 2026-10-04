package tui

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/thomashartm/motley/internal/blueprint"
	"github.com/thomashartm/motley/internal/tmux"
)

func templateModel(t *testing.T, body string) (Model, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	path := filepath.Join(root, "motley", "blueprints", "test.md")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	src := "+++\nname='example'\nagent='codex'\nvars=['constraints']\n+++\n" + body
	if err := os.WriteFile(path, []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	m.selectOverview()
	m.panel = actionsPanel
	for i, action := range m.actions() {
		if action.key == "f" {
			m.actionCursor = i
		}
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || next.(Model).blueprints == nil {
		t.Fatal("template manager unreachable from Main actions")
	}
	return update(next.(Model), cmd()), path
}

func TestGlobalTemplateRawGenerateCopyAndReturn(t *testing.T) {
	m, path := templateModel(t, "{{.Repo}}: {{.Vars.constraints}}\n{{.Issue.Body}}\n")
	m = arrow(m, tea.KeyEnter)
	m = arrow(m, tea.KeyEnter)
	if m.blueprints.page != blueprintRaw || !strings.Contains(m.blueprints.preview.View(), "+++") {
		t.Fatal("raw source missing front matter")
	}
	var copied, client string
	m.copyText = func(c, text string) error { client, copied = c, text; return nil }
	next, cmd := m.Update(key("c"))
	m = update(next.(Model), cmd())
	src, err := os.ReadFile(path)
	if err != nil || copied != string(src) {
		t.Fatal("raw copy changed bytes", err)
	}
	m = arrow(m, tea.KeyEsc)
	m = arrow(m, tea.KeyDown)
	m = arrow(m, tea.KeyEnter)
	if m.blueprints.page != blueprintArguments {
		t.Fatal("argument editor not opened")
	}
	m = update(m, key("Keep $HOME and `literal`"))
	m = arrow(m, tea.KeyEnter)
	m = update(m, key("Second line"))
	m = arrow(m, tea.KeyTab)
	m = update(m, key("motley"))
	// The optional Issue.Body accepts multiline context without GitHub access.
	for m.blueprints.labels[m.blueprints.argument] != "Issue.Body" {
		m = arrow(m, tea.KeyTab)
	}
	m = update(m, key("Issue text"))
	m = arrow(m, tea.KeyCtrlS)
	want := "motley: Keep $HOME and `literal`\nSecond line\nIssue text\n"
	if m.blueprints.page != blueprintPrompt || m.blueprints.prompt != want {
		t.Fatalf("rendered %q", m.blueprints.prompt)
	}
	// Copy the unwrapped prompt through the monitor's active clipboard target.
	m.monitor = true
	m.clients = []tmux.Client{{Name: "monitor", Session: tmux.MonitorSession}}
	next, cmd = m.Update(key("c"))
	m = update(next.(Model), cmd())
	if copied != want || client != "monitor" || !strings.Contains(m.message, "Prompt copied") {
		t.Fatal(copied, client, m.message)
	}
	if m.spawn != nil || m.busy {
		t.Fatal("generation attempted to launch an agent")
	}
	m = arrow(m, tea.KeyEsc)
	if m.blueprints.page != blueprintArguments || m.blueprints.input.Value() != "Issue text" {
		t.Fatal("back lost entered values")
	}
	m = arrow(m, tea.KeyCtrlS)
	m.copyText = func(string, string) error { return errors.New("clipboard unavailable") }
	next, cmd = m.Update(key("c"))
	m = update(next.(Model), cmd())
	if !strings.Contains(m.message, "Copy failed") || m.blueprints.prompt != want {
		t.Fatal("copy failure discarded prompt")
	}
}

func TestTemplateRepairReloadAndRenderFailure(t *testing.T) {
	m, path := templateModel(t, "{{.Unknown}}")
	m = arrow(m, tea.KeyEnter)
	m = arrow(m, tea.KeyDown)
	m = arrow(m, tea.KeyEnter)
	m = arrow(m, tea.KeyCtrlS)
	if m.blueprints.page != blueprintArguments || !strings.Contains(m.message, "Unknown") {
		t.Fatal("render error lost form", m.message)
	}
	m = arrow(m, tea.KeyEsc)
	if err := os.WriteFile(path, []byte("invalid template"), 0600); err != nil {
		t.Fatal(err)
	}
	next, cmd := m.Update(key("r"))
	m = update(next.(Model), cmd())
	if len(m.blueprints.files) != 1 || m.blueprints.files[0].Err == nil {
		t.Fatal("invalid template disappeared")
	}
	m = arrow(m, tea.KeyEnter) // Generate remains selected.
	if m.blueprints.page != blueprintActions || !strings.Contains(m.message, "front matter") {
		t.Fatal("invalid template generation was allowed")
	}
	if err := os.WriteFile(path, []byte("+++\n+++\nRepaired"), 0600); err != nil {
		t.Fatal(err)
	}
	next, cmd = m.Update(blueprintEdited{})
	m = update(next.(Model), cmd())
	m = arrow(m, tea.KeyEnter)
	m = arrow(m, tea.KeyCtrlS)
	if m.blueprints.prompt != "Repaired" {
		t.Fatal("edited template was not reloaded")
	}
}

func TestTemplateLayoutsAndModalMouse(t *testing.T) {
	m, _ := templateModel(t, strings.Repeat("Long prompt line\n", 80))
	for _, page := range []blueprintPage{blueprintList, blueprintActions, blueprintRaw, blueprintArguments, blueprintPrompt} {
		m.blueprints.beginArguments()
		m.blueprints.page = page
		m.blueprints.prompt = strings.Repeat("preview\n", 100)
		for _, size := range [][2]int{{120, 30}, {60, 10}, {60, 12}, {80, 24}} {
			m = update(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			assertFooterFits(t, m)
			before := m.blueprints.page
			m = click(m, 3, 2)
			if m.blueprints == nil || m.blueprints.page != before || m.navigationAvailable() {
				t.Fatal("mouse escaped template modal")
			}
		}
	}
	m.blueprints.page = blueprintList
	m = update(m, tea.WindowSizeMsg{Width: 120, Height: 30})
	m = click(m, m.listWidth()+4, 3+m.panelHeadingGap())
	if m.blueprints.page != blueprintActions {
		t.Fatal("template list mouse selection failed")
	}
	m = click(m, m.listWidth()+4, 3+m.panelHeadingGap())
	if m.blueprints.page != blueprintRaw {
		t.Fatal("raw view mouse action failed")
	}
}

func TestTemplateEditorCommandsAndLiteralFilename(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	path := filepath.Join(t.TempDir(), "spaces ' $(touch SHOULD_NOT_EXIST) `echo nope`.md")
	for _, tc := range []struct {
		platform string
		vi       bool
		want     []string
	}{
		{"darwin", true, []string{"vi", path}},
		{"darwin", false, []string{"open", "-t", path}},
		{"linux", false, []string{"xdg-open", path}},
	} {
		cmd, err := blueprintEditor(path, tc.vi, tc.platform)
		if err != nil || !reflect.DeepEqual(cmd.Args, tc.want) {
			t.Fatal(cmd, err)
		}
	}
	// VISUAL takes precedence; arguments in it work and the file stays literal.
	t.Setenv("EDITOR", "false")
	t.Setenv("VISUAL", "printf '%s'")
	cmd, err := blueprintEditor(path, false, "darwin")
	if err != nil {
		t.Fatal(err)
	}
	out, err := cmd.Output()
	if err != nil || string(out) != path {
		t.Fatalf("%q %v", out, err)
	}
}

func TestEmptyGlobalTemplates(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	next, cmd := m.beginBlueprints()
	m = update(next.(Model), cmd())
	if len(m.blueprints.files) != 0 || !strings.Contains(m.View(), "No global templates") {
		t.Fatal(m.View())
	}
	m = arrow(m, tea.KeyEnter)
	m = arrow(m, tea.KeyEsc)
	if m.blueprints != nil {
		t.Fatal("cannot exit empty manager")
	}
}

func TestTemplatePromptLimits(t *testing.T) {
	m, _ := templateModel(t, "{{.Vars.constraints}}{{.Vars.constraints}}")
	m.blueprints.beginArguments()
	m.blueprints.input.SetValue(strings.Repeat("x", blueprint.MaxPromptBytes/2+1))
	m = arrow(m, tea.KeyCtrlS)
	if m.blueprints.page != blueprintArguments || !strings.Contains(m.message, "64 KiB") {
		t.Fatal(m.message)
	}
}

func TestGlobalTemplatesRemainAvailableDuringSpawn(t *testing.T) {
	m, path := templateModel(t, "{{.Vars.constraints}}")
	m = arrow(m, tea.KeyEsc)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "api", ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "init", filepath.Join(root, "api")).CombinedOutput(); err != nil {
		t.Fatalf("git init: %s: %v", out, err)
	}
	m.spawnCfg.ReposRoots = []string{root}
	m.spawn = &spawnForm{step: agentStep, choice: 1}
	m.spawn.opts.Repo = "api"
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("no blueprint discovery during spawn")
	}
	m = update(next.(Model), cmd())
	if m.spawn.step != blueprintStep || len(m.spawn.blueprints) != 1 || m.spawn.blueprints[0].Path != path {
		t.Fatalf("spawn discovery: %s", m.spawn.err)
	}
	m = arrow(m, tea.KeyDown)
	m = arrow(m, tea.KeyEnter)
	m = update(m, key("Deliver the feature"))
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if cmd == nil || m.spawn.opts.Blueprint != "example" || m.spawn.fields[0].Value() != "Deliver the feature" || !reflect.DeepEqual(m.spawn.vars, []string{"constraints"}) {
		t.Fatal("spawn did not retain selected template and arguments")
	}
}
