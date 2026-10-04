package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/member"
)

func TestForegroundBackgroundDisplayAndPicker(t *testing.T) {
	r := row("idd", true)
	r.Agent, r.ClaudeSession, r.ClaudeSessions = "claude", "foreground", []string{"background"}
	r.Status = "working"
	r.ClaudeStatuses = []member.ClaudeSessionStatus{
		{ID: "foreground", Name: "Terminal", Kind: "interactive", Status: "idle", Alive: true, Managed: true},
		{ID: "background", Name: "Implementation", Kind: "background", Status: "working", Alive: true},
	}
	m := update(newModel(true, true, "", nil), tea.WindowSizeMsg{Width: 140, Height: 40})
	m = update(m, snapshot{rows: []member.Row{r}})
	for _, width := range []int{30, 60, 100} {
		line := ansi.Strip(m.memberTableRow(r, width, true))
		if !strings.Contains(line, "MIX F+B") || ansi.StringWidth(line) > width {
			t.Fatal(width, line)
		}
	}
	details := ansi.Strip(m.memberDetails())
	for _, want := range []string{"Foreground", "idle", "Background", "working", "foreground", "background", "Open target", "TMX", "EXT", "MIX"} {
		if !strings.Contains(details, want) {
			t.Fatal("missing", want, details)
		}
	}
	next, cmd := m.Update(key("A"))
	m = next.(Model)
	if cmd == nil || m.importing == nil || !m.importing.additional || m.importing.replaceID != r.ID {
		t.Fatal("add linked session did not start")
	}
	m = update(m, importLoaded{agent: "claude", replaceID: r.ID, additional: true, sessions: []member.ImportCandidate{{SessionID: "other", Name: "Background work", Cwd: "/repo", Kind: "background", Status: "working"}}})
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "Track another session") || !strings.Contains(view, "BG") {
		t.Fatal(view)
	}
	next, cmd = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil || !next.(Model).busy {
		t.Fatal("selection did not submit")
	}
	m = update(next.(Model), importDone{member: r.Manifest, additional: true})
	if m.importing != nil || !strings.Contains(m.message, "Tracking both") {
		t.Fatal(m.message)
	}
	m = update(m, key("d"))
	if !strings.Contains(m.terminateView(20), "all tracked foreground and background") {
		t.Fatal(m.terminateView(20))
	}
}
