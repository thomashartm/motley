package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/thomashartm/motley/internal/member"
)

type importDialog struct {
	agent     string
	replaceID string
	sessions  []member.ImportCandidate
	cursor    int
}
type importLoaded struct {
	agent     string
	replaceID string
	sessions  []member.ImportCandidate
	err       error
}
type importDone struct {
	member   member.Manifest
	err      error
	replaced bool
}

func (m Model) beginSwitchSession() (tea.Model, tea.Cmd) {
	r := m.selectedRow()
	if r.ClaudeSession == "" {
		return m, nil
	}
	m.importing = &importDialog{agent: "claude", replaceID: r.ID}
	m.busy, m.busyText = true, "Finding replacement sessions…"
	return m, func() tea.Msg {
		s, err := member.ReplacementSessions(r.ID)
		return importLoaded{agent: "claude", replaceID: r.ID, sessions: s, err: err}
	}
}

func (m Model) beginImport() (tea.Model, tea.Cmd) {
	m.menu = &menuDialog{title: "Add existing agent", items: []menuItem{
		{label: "Claude", run: func(m Model) (tea.Model, tea.Cmd) { return m.beginImportAgent("claude") }},
		{label: "Codex", run: func(m Model) (tea.Model, tea.Cmd) { return m.beginImportAgent("codex") }},
	}}
	m.message = ""
	return m, nil
}
func (m Model) beginImportAgent(agent string) (tea.Model, tea.Cmd) {
	m.importing = &importDialog{agent: agent}
	m.busy, m.busyText = true, "Finding "+importAgentLabel(agent)+" sessions…"
	return m, func() tea.Msg {
		s, err := member.DiscoverImports(agent)
		return importLoaded{agent: agent, sessions: s, err: err}
	}
}
func (m Model) importMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.busy, m.busyText = false, ""
	switch msg := msg.(type) {
	case importLoaded:
		if msg.err != nil {
			m.importing = nil
			m.message = msg.err.Error()
			return m, nil
		}
		m.importing = &importDialog{agent: msg.agent, replaceID: msg.replaceID, sessions: msg.sessions}
	case importDone:
		if msg.err != nil {
			m.message = msg.err.Error()
			return m, nil
		}
		m.importing = nil
		m.message = "Added " + msg.member.Name + "; Claude is still running in its original terminal."
		if msg.member.CodexSession != "" {
			m.message = "Added " + msg.member.Name + "; Open agent connects to its existing Codex conversation."
		}
		if msg.replaced {
			m.message = "Now tracking " + msg.member.ClaudeSession + "; both sessions keep running."
		}
		m.focusID = msg.member.ID
		m.group = "attention"
		m.panel = listPanel
		m.tableFocus = false
		m.query.SetValue("")
		return m, m.poll
	}
	return m, nil
}
func (m Model) updateImport(key string) (tea.Model, tea.Cmd) {
	d := *m.importing
	m.importing = &d
	switch key {
	case "esc", "q":
		m.importing = nil
		m.message = ""
	case "up", "k":
		d.cursor = max(0, d.cursor-1)
	case "down", "j":
		d.cursor = min(max(0, len(d.sessions)-1), d.cursor+1)
	case "enter":
		if len(d.sessions) == 0 {
			return m, nil
		}
		id := d.sessions[d.cursor].SessionID
		if d.replaceID != "" {
			m.busy, m.busyText = true, "Switching tracked session…"
			return m, func() tea.Msg {
				member, err := member.SwitchClaudeSession(d.replaceID, id)
				return importDone{member: member, err: err, replaced: true}
			}
		}
		m.busy, m.busyText = true, "Adding "+d.agent+" session…"
		return m, func() tea.Msg {
			member, err := member.Import(d.agent, id, "", "")
			return importDone{member: member, err: err}
		}
	}
	return m, nil
}
func importAgentLabel(agent string) string {
	if agent == "codex" {
		return "Codex"
	}
	return "Claude"
}
