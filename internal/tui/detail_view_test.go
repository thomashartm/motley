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

func TestDetailsDescribeDirectGhosttyAccess(t *testing.T) {
	m := actionModel(120, 30)
	r := row("ghostty-fixture", true)
	r.LaunchTarget = "ghostty"
	m = update(m, snapshot{rows: []member.Row{r}})
	if flat := strings.Join(strings.Fields(ansi.Strip(m.memberDetails())), " "); !strings.Contains(flat, "Access: GHO · Direct Ghostty window") {
		t.Fatal(flat)
	}
}

func TestDetailsShowProcessStates(t *testing.T) {
	for _, tc := range []struct{ status, label, group string }{
		{"waiting", "waiting for process", "WAITING FOR PROCESS"},
		{"background", "background processes", "BACKGROUND PROCESSES"},
		{"permission", "permission", "NEEDS YOU"},
	} {
		for _, width := range []int{60, 120} {
			m := actionModel(width, 30)
			r := statusRow("process-fixture", tc.status, 1)
			r.ClaudeStatuses = []member.ClaudeSessionStatus{{ID: "s", Alive: true, Status: tc.status, Processes: "Last reported: 1 shell(s), 1 monitor(s)"}}
			m = update(m, snapshot{rows: []member.Row{r}})
			view := ansi.Strip(m.memberDetails())
			flat := strings.Join(strings.Fields(view), " ")
			if !strings.Contains(flat, tc.label) || !strings.Contains(flat, "Last reported:") || !strings.Contains(flat, "monitor(s)") {
				t.Fatal(flat)
			}
			if section(r) != tc.group {
				t.Fatal(section(r))
			}
			icon, _ := statusIcon(tc.status)
			if icon == "○" || !strings.Contains(totals([]member.Row{r}), icon+"1") {
				t.Fatal("missing status icon/count")
			}
			for _, line := range strings.Split(view, "\n") {
				if ansi.StringWidth(line) > m.detailWidth() {
					t.Fatalf("overflow: %q", line)
				}
			}
		}
	}
}

func TestManagedDetailsShowBackgroundCounts(t *testing.T) {
	m := actionModel(120, 30)
	r := statusRow("managed-process", "permission", 1)
	m = update(m, snapshot{rows: []member.Row{r}})
	m.event = state.Event{Status: "permission", Detail: map[string]string{"shells": "1", "monitors": "2"}}
	view := strings.Join(strings.Fields(ansi.Strip(m.memberDetails())), " ")
	if !strings.Contains(view, "Last reported: 1 shell(s), 2 monitor(s)") {
		t.Fatal(view)
	}
}
