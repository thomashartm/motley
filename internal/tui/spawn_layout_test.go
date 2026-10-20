package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/member"
	"strings"
	"testing"
)

func TestSpawnIdentityKeepsFieldsAndErrorsVisible(t *testing.T) {
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 120, Height: 30})
	m.spawn = &spawnForm{step: identityStep, fields: inputs("42", "Payments", "feature", "feature/42-payments")}
	view := ansi.Strip(m.spawnView(20))
	if strings.Contains(view, "…") {
		t.Fatal("short field values appear truncated:\n" + view)
	}
	for _, want := range []string{"Ticket    42", "Name      Payments", "Branch    feature/42-payments"} {
		if !strings.Contains(view, want) {
			t.Fatalf("fields not aligned: missing %q:\n%s", want, view)
		}
	}
	m.spawn.field = 3
	m.spawn.err = "Choose another branch"
	view = ansi.Strip(m.spawnView(4))
	if !strings.Contains(view, "feature/42-payments") || !strings.Contains(view, "Choose another branch") {
		t.Fatal("focused field or error clipped:\n" + view)
	}
}

func TestSpawnReviewShowsReadableSummaryAndActions(t *testing.T) {
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 160, Height: 40})
	m.spawn = &spawnForm{}
	m = update(m, spawnPrepared{plan: member.Prepared{Manifest: member.Manifest{Name: "Payments", Repo: "api", Branch: "feature/42-payments", Agent: "claude", LaunchTarget: "ghostty"}, Prompt: "Please implement payments."}})
	view := ansi.Strip(m.spawnView(28))
	for _, want := range []string{"Name: Payments", "Repository: api", "Branch: feature/42-payments", "Agent: claude", "Launch", "Edit prompt", "Cancel", "Please implement payments."} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q:\n%s", want, view)
		}
	}
}

func TestSpawnReviewScrollsWithoutHidingActions(t *testing.T) {
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 60, Height: 10})
	m.spawn = &spawnForm{}
	m = update(m, spawnPrepared{plan: member.Prepared{Manifest: member.Manifest{Name: "Payments", Repo: "api", Branch: "feature/42-payments", Agent: "claude", Mode: "sandbox"}, Prompt: "END OF REVIEW"}})
	for range 30 {
		m = arrow(m, tea.KeyDown)
	}
	view := ansi.Strip(m.spawnView(m.contentHeight()))
	if !strings.Contains(view, "> Launch") || !strings.Contains(view, "END OF REVIEW") {
		t.Fatal("review or launch inaccessible:\n" + view)
	}
}

func TestSpawnChoicesFitAndKeepSelection(t *testing.T) {
	for _, size := range [][2]int{{60, 10}, {80, 24}, {120, 30}, {160, 40}} {
		m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, step := range []int{agentStep, destinationStep, windowStep, modeStep, blueprintStep} {
			m.spawn = &spawnForm{step: step, opts: member.SpawnOptions{Agent: "claude"}}
			for choice := range m.spawnChoiceLabels() {
				m.spawn.choice = choice
				view := ansi.Strip(m.spawnView(m.contentHeight()))
				if !strings.Contains(view, ">") {
					t.Fatalf("selection hidden at %v step %d:\n%s", size, step, view)
				}
				if len(strings.Split(view, "\n")) > m.contentHeight() {
					t.Fatal("height overflow", view)
				}
				for _, line := range strings.Split(view, "\n") {
					if ansi.StringWidth(line) > m.detailWidth() {
						t.Fatal("width overflow", line)
					}
				}
			}
		}
	}
}

func TestSpawnLongInputsKeepCursorVisible(t *testing.T) {
	m := update(newModel(false, false, "", nil), tea.WindowSizeMsg{Width: 60, Height: 10})
	for _, step := range []int{identityStep, repoStep} {
		m.spawn = &spawnForm{step: step, query: inputs(strings.Repeat("long-", 20) + "END")[0], field: 1, fields: inputs("", strings.Repeat("long-", 20)+"END", "feature", "feature/42-payments")}
		if view := ansi.Strip(m.spawnView(m.contentHeight())); !strings.Contains(view, "END") {
			t.Fatalf("cursor end hidden at step %d:\n%s", step, view)
		}
	}
}
