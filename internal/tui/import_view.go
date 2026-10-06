package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/member"
)

// The view and pointer handling share the same rows. Headers and the space
// between session cards never select or import a session.
type importRow struct {
	text  string
	index int
}

func (m Model) importRows(height int) []importRow {
	d := m.importing
	width := m.detailWidth()
	title := "Add existing " + importAgentLabel(d.agent)
	if d.replaceID != "" {
		title = "Switch tracked session"
	}
	if d.additional {
		title = "Track another session"
	}
	reimporting := len(d.sessions) > 0 && d.sessions[d.cursor].ReimportID != ""
	if reimporting && d.replaceID != "" {
		title = "Reimport session"
	}
	count := fmt.Sprintf("%d/%d", min(d.cursor+1, len(d.sessions)), len(d.sessions))
	header := cell(title, max(1, width-len(count)-1)) + " " + count
	rows := []importRow{{fit(header, width), -1}}
	if len(d.sessions) == 0 {
		message := "No unregistered " + importAgentLabel(d.agent) + " sessions."
		if d.replaceID != "" {
			message = "No running sessions available. Start Claude, then retry reimport; use Revive to resume a stopped session."
		}
		for _, line := range strings.Split(ansi.Wrap(message, width, ""), "\n") {
			rows = append(rows, importRow{line, -1})
		}
		return rows[:min(len(rows), height)]
	}
	if d.replaceID != "" && height >= 8 {
		message := "Both sessions keep running."
		if reimporting {
			message = "Updates workspace; session keeps running."
		}
		rows = append(rows, importRow{fit(message, width), -1})
	}
	available := height - len(rows)
	titleLines := 1
	if available >= 4 {
		titleLines = 2
	}
	cards := make([][]string, len(d.sessions))
	for i, session := range d.sessions {
		cards[i] = importCard(session, width, titleLines, i == d.cursor)
	}
	// Start with the complete selected card, then include as many complete
	// preceding cards as fit. Never show half of the next session.
	start, used := d.cursor, len(cards[d.cursor])
	for start > 0 && used+1+len(cards[start-1]) <= available {
		start--
		used += 1 + len(cards[start])
	}
	for i := start; i < len(cards); i++ {
		gap := 0
		if i > start {
			gap = 1
		}
		if len(rows)+gap+len(cards[i]) > height {
			break
		}
		if gap > 0 {
			rows = append(rows, importRow{"", -1})
		}
		for _, line := range cards[i] {
			rows = append(rows, importRow{line, i})
		}
	}
	return rows
}

func importCard(s member.ImportCandidate, width, titleLines int, selected bool) []string {
	bodyWidth := max(1, width-2)
	name := strings.TrimSpace(clean(s.Name))
	if name == "" {
		name = "Untitled session"
	}
	if s.ReimportID != "" {
		name = "Reimport: " + name
	}
	title := strings.Split(ansi.Wrap(name, bodyWidth, ""), "\n")
	if len(title) > titleLines {
		title = title[:titleLines]
		title[titleLines-1] = fit(title[titleLines-1]+" …", bodyWidth)
	}
	path := clean(s.Cwd)
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(path, home+string(os.PathSeparator)) {
		path = "~" + strings.TrimPrefix(path, home)
	}
	if path == "" {
		path = "No working directory"
	}
	if size := ansi.StringWidth(path); size > bodyWidth {
		// A cut through a wide grapheme can retain one extra display cell.
		// Advance the cut rather than losing the meaningful path suffix.
		for cut := size - bodyWidth + 1; cut <= size; cut++ {
			short := ansi.TruncateLeft(path, cut, "…")
			if ansi.StringWidth(short) <= bodyWidth {
				path = short
				break
			}
		}
	}
	status := clean(s.Status)
	if status == "" {
		status = "unknown"
	}
	icon, colour := statusIcon(status)
	label := status
	if label == "permission" {
		label = "approval"
	}
	statusText := icon + " " + label
	if s.Kind != "" {
		location := (member.ClaudeSessionStatus{Kind: s.Kind}).Location()
		statusText += " · " + location
	}
	idWidth := max(5, bodyWidth-ansi.StringWidth(statusText)-3)
	id := clean(s.SessionID)
	if ansi.StringWidth(id) > min(13, idWidth) {
		keep := min(13, idWidth) - 5
		id = ansi.Cut(id, 0, keep) + "…" + ansi.Cut(id, ansi.StringWidth(id)-4, ansi.StringWidth(id))
	}
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	var lines []string
	for i, line := range title {
		prefix := "  "
		if selected && i == 0 {
			prefix = "> "
		}
		lines = append(lines, lipgloss.NewStyle().Bold(true).Render(prefix+line))
	}
	lines = append(lines, "  "+muted.Render(path))
	lines = append(lines, "  "+lipgloss.NewStyle().Foreground(colour).Render(statusText)+muted.Render(" · "+id))
	for i, line := range lines {
		style := lipgloss.NewStyle()
		if selected {
			style = style.Background(lipgloss.AdaptiveColor{Light: "#E1F3F5", Dark: "#16343B"}).Foreground(lipgloss.Color("6"))
		}
		lines[i] = style.Render(cell(line, width))
	}
	return lines
}

func (m Model) importView(height int) string {
	rows := m.importRows(height)
	lines := make([]string, len(rows))
	for i, row := range rows {
		lines[i] = row.text
	}
	return strings.Join(lines, "\n")
}

func (m Model) importMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.busy || m.width < 60 || m.height < 10 || msg.X <= m.listWidth()+2 || msg.X >= m.width-1 || msg.Y < 2 || msg.Y >= 2+m.panelHeight() {
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelUp {
		return m.updateImport("up")
	}
	if msg.Button == tea.MouseButtonWheelDown {
		return m.updateImport("down")
	}
	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return m, nil
	}
	y := m.panelContentY(msg.Y)
	rows := m.importRows(m.contentHeight())
	if y < 0 || y >= len(rows) || rows[y].index < 0 {
		return m, nil
	}
	d := *m.importing
	d.cursor = rows[y].index
	m.importing = &d
	return m.updateImport("enter")
}
