package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/member"
)

func importCardModel() Model {
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	m.importing = &importDialog{agent: "codex", sessions: []member.ImportCandidate{
		{SessionID: "019b1234-aaaa-bbbb-cccc-123456789abc", Name: "Fix reverse-charge VAT rollback", Cwd: "/Users/thomas/projects/aderis/backend-fix-2956", Status: "working"},
		{SessionID: "019b1234-aaaa-bbbb-cccc-123456789def", Name: "Fix reverse-charge VAT rollback", Cwd: "/Users/thomas/projects/aderis/backend-fix-2956", Status: "permission"},
		{SessionID: "019b5678-aaaa-bbbb-cccc-123456780123", Name: "Polish Motley session picker", Cwd: "/Users/thomas/projects/motley-feature-bugfixes", Status: "idle"},
	}}
	return m
}

func TestImportCardsSeparateIdentityAndStatus(t *testing.T) {
	m := importCardModel()
	view := ansi.Strip(m.importView(m.contentHeight()))
	for _, want := range []string{"Add existing Codex", "1/3", "> Fix reverse-charge", "backend-fix-2956", "● working", "⚠ approval", "019b1234…9abc", "019b1234…9def", "\n\n"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q:\n%s", want, view)
		}
	}
	t.Log("\n" + view)
	m.importing.sessions[0].Name = ""
	m.importing.sessions[0].Cwd = "/a/very/long/parent/path/to/日本語/worktree-special"
	m = update(m, tea.WindowSizeMsg{Width: 60, Height: 10})
	view = ansi.Strip(m.importView(m.contentHeight()))
	if !strings.Contains(view, "Untitled session") || !strings.Contains(view, "worktree-special") || !strings.Contains(view, "9abc") {
		t.Fatalf("narrow view lost session identity:\n%s", view)
	}
}

func TestImportCardsKeepWholeSelectionAfterResizeAndScroll(t *testing.T) {
	m := importCardModel()
	for i := 0; i < 12; i++ {
		m.importing.sessions = append(m.importing.sessions, member.ImportCandidate{
			SessionID: fmt.Sprintf("session-%02d", i),
			Name:      "A long session title with details that wrap onto another line 日本語",
			Cwd:       "/workspace/one/two/three/repository", Status: "working",
		})
	}
	for _, size := range [][2]int{{120, 30}, {60, 10}, {60, 12}, {80, 24}, {180, 40}} {
		m = update(m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for i := range m.importing.sessions {
			m.importing.cursor = i
			assertFooterFits(t, m)
			view := ansi.Strip(m.importView(m.contentHeight()))
			if !strings.Contains(view, fmt.Sprintf("%d/%d", i+1, len(m.importing.sessions))) {
				t.Fatal("selected position is hidden")
			}
			// Every visible card has a complete status/identity line, including
			// the selected card; no orphan title from the next session appears.
			visible := map[int]int{}
			for _, row := range m.importRows(m.contentHeight()) {
				if row.index >= 0 {
					visible[row.index]++
				}
			}
			if visible[i] < 3 {
				t.Fatalf("selected card clipped at %dx%d", size[0], size[1])
			}
			for index, count := range visible {
				if count < 3 {
					t.Fatalf("partial card %d", index)
				}
			}
		}
	}
}

func TestImportCardMouseTargetsAndGaps(t *testing.T) {
	m := importCardModel()
	rows := m.importRows(m.contentHeight())
	for y, row := range rows {
		// Convert content rows to screen rows, accounting for the title divider.
		screenY := 2 + y
		if y > 0 {
			screenY += m.panelHeadingGap()
		}
		next, cmd := m.Update(tea.MouseMsg{X: m.listWidth() + 4, Y: screenY, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		got := next.(Model)
		if row.index < 0 {
			if cmd != nil || got.busy {
				t.Fatal("heading or gap imported a session")
			}
		} else if cmd == nil || !got.busy || got.importing.cursor != row.index {
			t.Fatalf("card row selected the wrong session: row=%d want=%d got=%d", y, row.index, got.importing.cursor)
		}
	}
	m = update(m, tea.WindowSizeMsg{Width: 60, Height: 10})
	m = update(m, tea.MouseMsg{X: m.listWidth() + 4, Y: 3, Button: tea.MouseButtonWheelDown, Action: tea.MouseActionPress})
	if m.importing.cursor != 1 || m.busy {
		t.Fatal("wheel should select without importing")
	}
	// With only one card visible, its path and metadata still hit that card.
	next, cmd := m.Update(tea.MouseMsg{X: m.listWidth() + 4, Y: 5, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if cmd == nil || next.(Model).importing.cursor != 1 {
		t.Fatal("scrolled card click selected a hidden session")
	}
}
