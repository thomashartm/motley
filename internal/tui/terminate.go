package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/member"
)

type terminateDialog struct {
	id       string
	confirm  bool
	external bool
	grouped  bool
	stopped  bool
	name     string
	err      string
}

func (m Model) beginTerminate() (tea.Model, tea.Cmd) {
	id := m.selectedID()
	if id == "" {
		return m, nil
	}
	if m.selectedRow().CodexSession != "" {
		m.message = "Codex runs on a shared server; stop the turn in Codex, or Retire to remove only its Motley entry."
		return m, nil
	}
	r := m.selectedRow()
	m.terminating = &terminateDialog{id: id, name: r.Name, external: r.External, grouped: len(r.ClaudeSessions) > 0, stopped: !r.Alive}
	m.message = ""
	return m, nil
}

func (m Model) updateTerminate(key string) (tea.Model, tea.Cmd) {
	d := *m.terminating
	m.terminating = &d
	switch key {
	case "up", "down", "left", "right", "tab", "shift+tab":
		d.confirm = !d.confirm
		return m, nil
	case "enter":
		key = "esc"
		if d.confirm {
			key = "y"
		}
	}
	switch key {
	case "esc", "q", "n":
		m.terminating = nil
	case "y":
		d.err = ""
		m.busy = true
		m.busyText = "Terminating…"
		return m, func() tea.Msg { return lifecycleDone{id: d.id, action: "Terminated", err: member.Terminate(d.id)} }
	}
	return m, nil
}

func (m Model) terminateButtons() string {
	if m.terminating.confirm {
		return "  [Cancel: esc] > [Terminate: y]"
	}
	return "> [Cancel: esc]   [Terminate: y]"
}

func (m Model) terminateView(height int) string {
	action := "Stops all processes in this agent session."
	if m.terminating.external {
		action = "Stops Claude in its original terminal."
	}
	if m.terminating.grouped {
		action = "Stops all tracked foreground and background conversations."
	}
	if m.terminating.stopped {
		action = "Already stopped. Removes this entry from the active list."
	}
	name := m.terminating.name
	if name == "" {
		name = m.terminating.id
	}
	text := "Terminate " + clean(name) + "?\n" + action + "\nRemoves the entry from the list and archives its history. Keeps all files and branches."
	if m.terminating.err != "" {
		text = "Termination failed: " + clean(name) + "\n" + m.terminating.err + "\nEntry kept. Retry with y or cancel with esc."
	}
	lines := strings.Split(ansi.Hardwrap(text, m.detailWidth(), true), "\n")
	// Keep the confirmation beside its explanation, even in small terminals.
	if height >= 4 {
		lines = lines[:min(len(lines), height-3)]
		lines = append(lines, "")
		lines = append(lines, m.terminateChoices()...)
	}
	return strings.Join(lines[:min(len(lines), height)], "\n")
}

func (m Model) terminateChoices() []string {
	cancel, confirm := "> Cancel (esc)", "  Terminate (y)"
	if m.terminating.confirm {
		cancel, confirm = "  Cancel (esc)", "> Terminate (y)"
	}
	return []string{cancel, confirm}
}
