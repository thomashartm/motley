package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/member"
)

func drag(t *testing.T, m Model, x1, y1, x2, y2 int) Model {
	t.Helper()
	for _, msg := range []tea.MouseMsg{
		{X: x1, Y: y1, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress},
		{X: x2, Y: y2, Button: tea.MouseButtonLeft, Action: tea.MouseActionMotion},
		{X: x2, Y: y2, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease},
	} {
		next, cmd := m.Update(msg)
		m = next.(Model)
		if cmd != nil || m.editor != nil || m.retiring != nil || m.terminating != nil || m.attachID != "" {
			t.Fatal("drag dispatched an action")
		}
	}
	if m.selection == nil || !m.selection.active || m.selection.dragging {
		t.Fatal("drag did not select text")
	}
	return m
}

func TestDragRowsCopiesFullTitlesInEitherDirection(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		m, calls := copyModel(false, nil)
		a, b := row("a", true), row("b", true)
		a.Name = "Alpha " + strings.Repeat("long title ", 20) + "END"
		b.Name = "Beta 界e\u0301"
		m = update(m, snapshot{rows: []member.Row{a, b}})
		lines := m.selectableRows()
		from, to := lines[0], lines[1]
		if reverse {
			from, to = to, from
		}
		m = drag(t, m, from.x+4, from.y, to.x+6, to.y)
		if got := ansi.Strip(m.View()); !strings.Contains(got, "[CPY]") || !strings.Contains(got, "[CLR]") {
			t.Fatal("missing selection hints", got)
		}
		next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		m = run(t, next.(Model), cmd)
		want := m.memberCopyLine(a) + "\n" + m.memberCopyLine(b)
		if len(*calls) != 1 || (*calls)[0].text != want || strings.Contains((*calls)[0].text, "\x1b") {
			t.Fatalf("copy lost rows or full titles: %v", *calls)
		}
		if !strings.Contains(ansi.Strip(m.View()), "Copied") {
			t.Fatal("missing copy acknowledgement")
		}
		m = update(m, tea.KeyMsg{Type: tea.KeyEsc})
		if m.selection != nil {
			t.Fatal("escape did not clear selection")
		}
		_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
		if cmd == nil {
			t.Fatal("Ctrl+C without selection no longer quits")
		}
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Fatal("Ctrl+C without selection must quit")
		}
	}
}

func TestDragActionsAndTicketDoesNotActivate(t *testing.T) {
	m := actionModel(140, 40)
	x, y := m.listWidth()+5, actionScreenY(t, m, "Retire member + worktree (x)")
	m = drag(t, m, x, y, x+5, y)
	if !strings.Contains(m.selection.text(), "Retire") {
		t.Fatal(m.selection.text())
	}
	m = update(m, tea.KeyMsg{Type: tea.KeyEsc})
	next, cmd := mouseClick(m, x, y)
	if cmd == nil || next.(Model).retiring == nil {
		t.Fatal("ordinary click no longer opens Retire")
	}
	m, _ = copyModel(false, nil)
	r := row("ticket", true)
	r.Ticket, r.RemoteURL = "42", "https://github.com/owner/repo.git"
	m = update(m, snapshot{rows: []member.Row{r}})
	m.openURL = func(string) error { t.Fatal("drag opened browser"); return nil }
	for y, line := range strings.Split(m.View(), "\n") {
		for x := 0; x < m.listWidth(); x++ {
			if hyperlinkAt(line, x, 0) != "" {
				m = drag(t, m, x, y, x+1, y)
				if !strings.Contains(m.selection.text(), "ticket") {
					t.Fatal("ticket drag did not select full row")
				}
				return
			}
		}
	}
	t.Fatal("ticket link not visible")
}

func TestActionsTitleSelectionUnicode(t *testing.T) {
	m, calls := copyModel(false, nil)
	r := row("a", true)
	r.Name = "界e\u0301 title"
	m = update(m, snapshot{rows: []member.Row{r}})
	m = update(m, key("3"))
	title := m.selectableTitles()[0]
	// Start in the second cell of a wide character; copy complete graphemes.
	m = drag(t, m, title.x+11, title.y, title.x+10, title.y)
	if m.selection.text() != "界e\u0301" {
		t.Fatalf("split grapheme: %q", m.selection.text())
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	run(t, next.(Model), cmd)
	if len(*calls) != 1 || (*calls)[0].text != "界e\u0301" {
		t.Fatal(*calls)
	}
}

func TestCopySelectionFailureAndRefresh(t *testing.T) {
	m, _ := copyModel(false, errors.New("clipboard unavailable"))
	line := m.selectableRows()[0]
	m = drag(t, m, line.x, line.y, line.x+1, line.y)
	m = update(m, snapshot{rows: []member.Row{row("0", true), row("a", true)}})
	if m.selection == nil || !strings.Contains(m.selection.text(), "\ta\t") {
		t.Fatal("refresh changed selected identity")
	}
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = run(t, next.(Model), cmd)
	if !strings.Contains(ansi.Strip(m.View()), "Copy failed:") {
		t.Fatal("copy failure hidden")
	}
	m = update(m, snapshot{rows: []member.Row{row("0", true)}})
	if m.selection != nil {
		t.Fatal("removed row remained selected")
	}
	line = m.selectableRows()[0]
	m = drag(t, m, line.x, line.y, line.x+1, line.y)
	m = update(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.selection != nil {
		t.Fatal("resize retained stale selection coordinates")
	}
}

func TestClickReleaseAfterRefreshDoesNotActivateDifferentRow(t *testing.T) {
	m, _ := copyModel(false, nil)
	line := m.selectableRows()[0]
	m = update(m, tea.MouseMsg{X: line.x, Y: line.y, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	m = update(m, snapshot{rows: []member.Row{row("0", true), row("a", true)}})
	selected := m.selectedID()
	next, cmd := m.Update(tea.MouseMsg{X: line.x, Y: line.y, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	if cmd != nil || next.(Model).selectedID() != selected {
		t.Fatal("refresh turned click into another row's action")
	}
}
