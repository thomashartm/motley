package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/rivo/uniseg"
	"github.com/thomashartm/motley/internal/member"
)

type selectionLine struct {
	key, text   string
	x, y, width int
}

type textSelection struct {
	press                       tea.MouseMsg
	lines                       []selectionLine
	anchor, end                 int
	endX                        int
	wholeRows, dragging, active bool
}

func (m Model) memberCopyLine(r member.Row) string {
	name := r.Name
	if name == "" {
		name = r.ID
	}
	ticket, _ := ticketLink(r)
	return strings.Join([]string{r.CurrentStatus(), clean(r.Agent), r.Access(), r.SessionLocation(), clean(name), clean(ticket), clean(m.crewFor(r.Crew).Title)}, "\t")
}

// Visible rows carry stable identities. Refreshes can move rows, but cannot
// silently replace the text being copied with another member's text.
func (m Model) selectableRows() []selectionLine {
	layout := m.listLayout(m.listContentHeight(), m.listWidth())
	var result []selectionLine
	for i := layout.start; i < len(layout.body); i++ {
		y := len(layout.fixed) + i - layout.start
		if y >= m.listContentHeight() {
			break
		}
		index := layout.body[i].index
		if index < 0 {
			continue
		}
		var r member.Row
		if m.group == "crew" {
			entries := m.crewEntries()
			if index >= len(entries) || entries[index].id == "" {
				continue
			}
			for _, candidate := range m.rows {
				if candidate.ID == entries[index].id {
					r = candidate
					break
				}
			}
		} else if index < len(m.rows) {
			r = m.rows[index]
		}
		if r.ID == "" {
			continue
		}
		if y > 0 {
			y += m.listHeadingGap()
		}
		result = append(result, selectionLine{key: r.ID, text: m.memberCopyLine(r), x: 1, y: y + 2, width: m.listWidth()})
	}
	return result
}

func (m Model) selectableTitles() []selectionLine {
	if m.panel != actionsPanel {
		return nil
	}
	lines := strings.Split(m.actionsView(m.contentHeight()), "\n")
	layout := m.layoutActions(m.contentHeight())
	var result []selectionLine
	for y, line := range lines {
		// The fixed title, section headings and action labels are selectable;
		// help text and dividers do not activate a selection.
		if y > layout.menuHeight {
			break
		}
		text := strings.TrimRight(ansi.Strip(line), " ")
		if text == "" {
			continue
		}
		screenY := y + 2
		if y > 0 {
			screenY += m.panelHeadingGap()
		}
		result = append(result, selectionLine{key: m.selectedID() + ":" + text, text: text, x: m.listWidth() + 3, y: screenY, width: m.detailWidth()})
	}
	return result
}

func lineAt(lines []selectionLine, x, y int) int {
	for i, line := range lines {
		if line.y == y && x >= line.x && x < line.x+line.width {
			return i
		}
	}
	return -1
}

func (m Model) mouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.navigationAvailable() && m.width >= 60 && m.height >= 10 {
		if m.selection != nil && m.selection.dragging {
			s := *m.selection
			m.selection = &s
			if msg.Action == tea.MouseActionMotion || msg.Action == tea.MouseActionRelease {
				if msg.X != s.press.X || msg.Y != s.press.Y {
					s.active = true
				}
				s.endX = msg.X
				s.end = s.anchor
				if s.wholeRows {
					for i, line := range s.lines {
						if line.y <= msg.Y {
							s.end = i
						}
					}
					if msg.Y < s.lines[0].y {
						s.end = 0
					}
				}
				if msg.Action == tea.MouseActionRelease {
					s.dragging = false
					if !s.active {
						m.selection = nil
						current := m.selectableTitles()
						if s.wholeRows {
							current = m.selectableRows()
						}
						at := lineAt(current, msg.X, msg.Y)
						if at >= 0 && current[at].key == s.lines[s.anchor].key {
							return m.mouseControls(s.press)
						}
					}
				}
				return m, nil
			}
		}
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			m.selection = nil
			lines, whole := m.selectableRows(), true
			at := lineAt(lines, msg.X, msg.Y)
			if at < 0 {
				lines, whole = m.selectableTitles(), false
				at = lineAt(lines, msg.X, msg.Y)
			}
			if at >= 0 {
				m.selection = &textSelection{press: msg, lines: lines, anchor: at, end: at, endX: msg.X, wholeRows: whole, dragging: true}
				return m, nil
			}
		}
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			m.selection = nil
		}
		// Hover must not change the selected action underneath copied text.
		if m.selection != nil && m.selection.active && msg.Action == tea.MouseActionMotion {
			return m, nil
		}
	}
	return m.mouseControls(msg)
}

// Cell bounds snap to complete grapheme clusters, including wide characters.
func selectionCells(text string, start, end int) (string, int, int) {
	lo, hi := min(start, end), max(start, end)
	lo = max(0, lo)
	hi = min(ansi.StringWidth(text), hi+1)
	var selected strings.Builder
	x, first, last := 0, -1, 0
	g := uniseg.NewGraphemes(text)
	for g.Next() {
		next := x + g.Width()
		if next > lo && x < hi {
			if first < 0 {
				first = x
			}
			last = next
			selected.WriteString(g.Str())
		}
		x = next
	}
	return selected.String(), first, last
}

func (s textSelection) text() string {
	if !s.active {
		return ""
	}
	if !s.wholeRows {
		line := s.lines[s.anchor]
		text, _, _ := selectionCells(line.text, s.press.X-line.x, s.endX-line.x)
		return text
	}
	var lines []string
	for i := min(s.anchor, s.end); i <= max(s.anchor, s.end); i++ {
		lines = append(lines, s.lines[i].text)
	}
	return strings.Join(lines, "\n")
}

func (m Model) copySelection() (tea.Model, tea.Cmd) {
	text := m.selection.text()
	if text == "" || m.copyText == nil {
		return m, nil
	}
	copyText, client := m.copyText, m.client
	if m.monitor {
		client = m.activeMonitorClient()
	}
	return m, func() tea.Msg { return copiedMsg{text: text, err: copyText(client, text)} }
}

func (m *Model) validateSelection() {
	if m.selection == nil {
		return
	}
	s := m.selection
	current := m.selectableTitles()
	if s.wholeRows {
		current = m.selectableRows()
	}
	for i := min(s.anchor, s.end); i <= max(s.anchor, s.end); i++ {
		found := false
		for _, line := range current {
			if line.key == s.lines[i].key && line.text == s.lines[i].text {
				found = true
				break
			}
		}
		if !found {
			m.selection = nil
			return
		}
	}
}

func (m Model) selectionView(view string) string {
	if m.selection == nil || !m.selection.active || !m.navigationAvailable() {
		return view
	}
	s := m.selection
	current := m.selectableTitles()
	if s.wholeRows {
		current = m.selectableRows()
	}
	lines := strings.Split(view, "\n")
	style := lipgloss.NewStyle().Background(lipgloss.Color("6")).Foreground(lipgloss.Color("0"))
	for i := min(s.anchor, s.end); i <= max(s.anchor, s.end); i++ {
		selected := s.lines[i]
		for _, target := range current {
			if target.key != selected.key || target.y >= len(lines) {
				continue
			}
			start, end := target.x, target.x+target.width
			if !s.wholeRows {
				_, a, b := selectionCells(selected.text, s.press.X-selected.x, s.endX-selected.x)
				if a < 0 {
					continue
				}
				start, end = target.x+a, target.x+b
			}
			line := lines[target.y]
			lines[target.y] = ansi.Cut(line, 0, start) + style.Render(ansi.Strip(ansi.Cut(line, start, end))) + ansi.Cut(line, end, ansi.StringWidth(line))
		}
	}
	message := "[CPY] ctrl+c · [CLR] esc"
	if m.copied == s.text() {
		message = "Copied · " + message
	}
	if strings.HasPrefix(m.message, "Copy failed:") {
		message = m.message + " · " + message
	}
	y := m.height - 1 - m.footerRows()
	if y >= 0 && y < len(lines) {
		lines[y] = fit(message, m.width)
	}
	return strings.Join(lines, "\n")
}
