package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/member"
	"github.com/thomashartm/motley/internal/worktree"
)

type retireDialog struct {
	focus               int
	id                  string
	check               member.RetireCheck
	err                 error
	loaded, force, keep bool
}
type retireChecked struct {
	id    string
	check member.RetireCheck
	err   error
}
type lifecycleDone struct {
	id, action string
	err        error
}

func (m Model) beginRetire() (tea.Model, tea.Cmd) {
	id := m.selectedID()
	if id == "" {
		return m, nil
	}
	m.retiring = &retireDialog{id: id}
	m.busy = true
	m.busyText = "Checking retirement…"
	client := m.github
	return m, func() tea.Msg {
		c, err := member.InspectRetire(id)
		if err == nil && client != nil {
			ctx, cancel := context.WithTimeout(context.Background(), member.RetireLookupTimeout)
			c.OpenPR = member.OpenPR(ctx, client, c.Manifest)
			cancel()
		}
		return retireChecked{id: id, check: c, err: err}
	}
}
func (m Model) beginRevive() (tea.Model, tea.Cmd) {
	id := m.selectedID()
	if id == "" {
		return m, nil
	}
	if m.selectedRow().Alive {
		m.message = "Revive requires a dead member; this session is still alive."
		return m, nil
	}
	m.busy = true
	m.busyText = "Reviving…"
	return m, func() tea.Msg { return lifecycleDone{id: id, action: "Revived", err: member.Revive(id)} }
}
func (m Model) updateRetire(key string) (tea.Model, tea.Cmd) {
	// Copy the dialog so model snapshots remain values, as elsewhere in Bubble Tea.
	dialog := *m.retiring
	m.retiring = &dialog
	switch key {
	case "up", "shift+tab":
		dialog.focus = (dialog.focus + len(m.retireChoices()) - 1) % len(m.retireChoices())
		return m, nil
	case "down", "tab":
		dialog.focus = (dialog.focus + 1) % len(m.retireChoices())
		return m, nil
	case "enter":
		keys := []string{"y", "f", "k", "esc"}
		if dialog.check.Manifest.Imported() {
			keys = []string{"y", "esc"}
		}
		key = keys[dialog.focus]
	}
	switch key {
	case "esc", "q":
		m.retiring = nil
		m.message = ""
	case "f":
		dialog.force = !dialog.force
		m.message = ""
	case "k":
		dialog.keep = !dialog.keep
		m.message = ""
	case "enter", "y":
		if !dialog.loaded || dialog.err != nil {
			return m, nil
		}
		if len(dialog.check.Risks()) > 0 && !dialog.force {
			m.message = "Work would be discarded. Select Force to enable it, or Esc to cancel."
			return m, nil
		}
		m.busy = true
		m.busyText = "Retiring…"
		m.message = ""
		return m, func() tea.Msg {
			return lifecycleDone{id: dialog.id, action: "Retired", err: member.Retire(dialog.id, dialog.force, dialog.keep)}
		}
	}
	return m, nil
}
func (m Model) retireView(height int) string {
	d := m.retiring
	width := m.detailWidth()
	bodyWidth := max(1, width-4)
	var lines []string
	if !d.loaded {
		lines = append(lines, "Checking worktree and commits…")
	} else if d.err != nil {
		lines = append(lines, "Cannot retire:", clean(d.err.Error()))
	} else if d.check.Manifest.CodexSession != "" {
		lines = append(lines, "Closes its Motley terminal and archives its Motley entry.", "", "Codex keeps its conversation and running work on the shared server.", "", "Keeps the checkout, files and all branches.")
	} else if d.check.Manifest.Imported() {
		lines = append(lines, "Stops all tracked conversations and archives its Motley entry.", "", "Keeps the checkout, files and all branches.")
	} else {
		dirty := "no"
		if d.check.Dirty {
			dirty = "YES"
		}
		lines = append(lines, "Dirty/untracked files: "+dirty, fmt.Sprintf("Unpushed commits: %d", d.check.Ahead), fmt.Sprintf("Force: %t · keep branch: %t", d.force, d.keep))
		if pr := d.check.OpenPR; pr != nil {
			lines = append(lines, clean(fmt.Sprintf("Open PR #%d stays open on GitHub: %s", pr.Number, pr.URL)))
		}
		branch := d.check.Manifest.Branch
		keep := d.keep || worktree.Protected(branch)
		action := "delete"
		if keep {
			action = "keep"
		}
		if branch == "" {
			action = "none (detached)"
		}
		lines = append(lines, "Local branch: "+action, "", "Removes the tmux session and worktree. Archives its manifest and history.", "Remote branches are kept.")
	}
	if d.loaded && d.err == nil && d.check.Manifest.Worktree != "" {
		path := ansi.Wrap(clean(d.check.Manifest.Worktree), bodyWidth, "/-")
		lines = append(lines, "", lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("8")).Render("Dir"), lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(path))
	}
	body := strings.Split(ansi.Wrap(strings.Join(lines, "\n"), bodyWidth, ""), "\n")
	labels := m.retireChoices()
	// Keep every control visible. At small heights, remove decorative gaps
	// before shortening the explanation and mark any omitted content.
	space := max(0, height-len(labels)-1)
	separator := space > 1
	if separator {
		space--
	}
	for i := len(body) - 1; len(body) > space && i >= 0; i-- {
		if strings.TrimSpace(body[i]) == "" {
			body = append(body[:i], body[i+1:]...)
		}
	}
	if len(body) > space {
		body = body[:space]
		if space > 0 {
			body[space-1] = ansi.Truncate(body[space-1]+" …", bodyWidth, "…")
		}
	}
	wrapped := []string{lipgloss.NewStyle().Bold(true).Render(fit("Retire "+clean(d.id)+"?", width))}
	for _, line := range body {
		wrapped = append(wrapped, "  "+line)
	}
	if separator {
		wrapped = append(wrapped, panelDivider(width))
	}
	for i, label := range labels {
		line := fit("  "+control(label, d.focus == i), width)
		if d.focus == i {
			line = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6")).Render(line)
		}
		wrapped = append(wrapped, line)
	}
	return strings.Join(wrapped[:min(len(wrapped), height)], "\n")
}

func (m Model) retireChoices() []string {
	d := m.retiring
	if d.check.Manifest.Imported() {
		return []string{"Confirm retirement (keep files)", "Cancel"}
	}
	return []string{"Confirm retirement", fmt.Sprintf("Force: %t", d.force), fmt.Sprintf("Keep branch: %t", d.keep), "Cancel"}
}
