package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/member"
	"github.com/thomashartm/motley/internal/state"
)

func TestDetailsShowNameAndInfoBeforeActivity(t *testing.T) {
	for _, width := range []int{60, 80, 120} {
		m := actionModel(width, 20)
		r := row("identity-fixture", true)
		r.Name = "Payment retries and reconciliation"
		r.Info = "Keep retries safe and explain failures to operators"
		m = update(m, snapshot{rows: []member.Row{r}})
		m.event = state.Event{Status: r.CurrentStatus(), Summary: "Investigating retry failures"}
		view := ansi.Strip(m.memberDetails())
		flat := strings.Join(strings.Fields(view), " ")
		if !strings.Contains(flat, "Name: "+r.Name+" Info: "+r.Info+" "+m.event.Summary) {
			t.Fatalf("name and info must stay together before activity at width %d:\n%s", width, view)
		}
		for _, line := range strings.Split(view, "\n") {
			if ansi.StringWidth(line) > m.detailWidth() {
				t.Fatalf("detail line exceeds width: %q", line)
			}
		}
		m = update(m, tea.WindowSizeMsg{Width: width, Height: 10})
		m.panel = detailPanel
		assertFooterFits(t, m)
	}
}

func TestDetailsLegacyIdentityAndSaveRefresh(t *testing.T) {
	m := actionModel(120, 30)
	r := row("legacy-member", true)
	r.Name = ""
	m = update(m, snapshot{rows: []member.Row{r}})
	view := strings.Join(strings.Fields(ansi.Strip(m.memberDetails())), " ")
	if !strings.Contains(view, "Name: legacy-member Info: —") {
		t.Fatal(view)
	}
	m.detail.GotoBottom()
	r.Name, r.Info = "Renamed session", "Remember the original goal"
	m.poll = func() tea.Msg { return snapshot{rows: []member.Row{r}} }
	next, cmd := m.Update(identitySaved{})
	if cmd == nil {
		t.Fatal("save did not request refreshed metadata")
	}
	m = update(next.(Model), cmd())
	view = ansi.Strip(m.detail.View())
	if m.detail.YOffset != 0 || !strings.Contains(view, r.Name) || !strings.Contains(view, r.Info) {
		t.Fatalf("saved metadata not visible at top: %s", view)
	}
}
