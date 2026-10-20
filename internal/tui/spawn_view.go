package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/agents"
	"github.com/thomashartm/motley/internal/crew"
)

var spawnAccent = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
var spawnMuted = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

func (m Model) spawnChoiceLabels() []string {
	f := m.spawn
	switch f.step {
	case agentStep:
		return []string{"Claude Code", "Codex", "OpenCode"}
	case destinationStep:
		return spawnDestinations()
	case windowStep:
		var labels []string
		for _, w := range f.windows {
			label := clean(w.Name)
			if label == "" {
				label = "Untitled window"
			}
			if w.Current {
				label = "Current · " + label
			}
			labels = append(labels, fmt.Sprintf("%s — %d tabs", label, w.Tabs))
		}
		return append(labels, "New window")
	case crewStep:
		s := f.issue.Suggestion
		first := fmt.Sprintf("Create crew %q and assign", s.Title)
		if c, ok := crew.Find(m.crews, s.CrewID); ok {
			first = "Assign crew " + c.Title
		}
		return []string{first, "No crew"}
	case blueprintStep:
		labels := []string{"No blueprint"}
		for _, b := range f.blueprints {
			label := b.Name
			if b.Description != "" {
				label += " — " + b.Description
			}
			labels = append(labels, label)
		}
		return labels
	case modeStep:
		labels := []string{"Default from " + f.opts.Agent + " settings"}
		if args := f.blueprintArgs(); len(args) > 0 {
			labels[0] = "Use blueprint / agent settings"
			for _, mode := range agents.Modes(f.opts.Agent) {
				if agents.ConflictingArg(args, mode.Args) != "" {
					labels[0] = "Use blueprint permissions: " + strings.Join(args, " ")
					break
				}
			}
		}
		for _, mode := range agents.Modes(f.opts.Agent) {
			labels = append(labels, mode.Name+" — "+mode.Summary)
		}
		return labels
	}
	return nil
}

type spawnChoiceRow struct {
	index int
	text  string
}

// The picker and mouse share rows, so wrapped options remain clickable.
func (m Model) spawnChoiceRows(width, height int) []spawnChoiceRow {
	labels := m.spawnChoiceLabels()
	var groups [][]spawnChoiceRow
	for i, label := range labels {
		wrapped := strings.Split(ansi.Wrap(clean(label), max(1, width-2), ""), "\n")
		// Keep long descriptions available for the selected option without hiding
		// the rest of the list on smaller terminals.
		if i != m.spawn.choice || height < 4 {
			wrapped = wrapped[:1]
			wrapped[0] = fit(clean(label), max(1, width-2))
		}
		var rows []spawnChoiceRow
		for j, line := range wrapped[:min(len(wrapped), max(1, height-1))] {
			prefix := "  "
			if i == m.spawn.choice && j == 0 {
				prefix = "> "
			}
			rows = append(rows, spawnChoiceRow{i, prefix + line})
		}
		if height >= len(labels)*3 {
			rows = append(rows, spawnChoiceRow{-1, ""})
		}
		groups = append(groups, rows)
	}
	if len(groups) == 0 {
		return nil
	}
	choice := min(m.spawn.choice, len(groups)-1)
	start, used := choice, len(groups[choice])
	for start > 0 && used+len(groups[start-1]) <= height {
		start--
		used += len(groups[start])
	}
	var rows []spawnChoiceRow
	for _, g := range groups[start:] {
		rows = append(rows, g...)
	}
	return rows[:min(len(rows), height)]
}

func (m Model) spawnMessages(width, height int) []string {
	var rows []string
	for _, text := range []string{m.spawn.err, m.spawn.warning} {
		if text == "" {
			continue
		}
		rows = append(rows, strings.Split(ansi.Wrap(clean(text), width, ""), "\n")...)
	}
	return rows[:min(len(rows), max(0, min(3, height-2)))]
}

func (m Model) spawnPreviewHeader() []string {
	f := m.spawn
	actions := spawnPreviewActions()
	for i := range actions {
		if i == f.previewAction {
			actions[i] = "> " + actions[i]
		}
		actions[i] = "[" + actions[i] + "]"
	}
	actionLine := strings.Join(actions, "  ")
	if ansi.StringWidth(actionLine) > m.detailWidth() {
		actionLine = "> " + spawnPreviewActions()[f.previewAction] + "  ←/→"
	}
	return []string{"Spawn · Prompt preview", actionLine}
}

func (m Model) spawnPreviewContent() string {
	f := m.spawn
	manifest := f.plan.Manifest
	permissions := manifest.Mode
	if permissions == "" {
		permissions = "Blueprint / agent settings"
	}
	summary := []string{"Name: " + manifest.Name, "Repository: " + manifest.Repo, "Branch: " + manifest.Branch, "Agent: " + manifest.Agent, "Permissions: " + permissions, "Destination: " + spawnDestination(manifest.LaunchTarget)}
	if manifest.LaunchTarget == "ghostty" && f.plan.GhosttyWindow != "" {
		label := "New window"
		for _, w := range f.windows {
			if w.ID == f.plan.GhosttyWindow {
				label = clean(w.Name)
				break
			}
		}
		summary = append(summary, "Window: "+label)
	}
	if source := m.previewSource(); source != "" {
		summary = append(summary, source)
	}
	if context := m.previewContext(); context != "" {
		summary = append(summary, context)
	}
	return ansi.Hardwrap(multiline("\n"+strings.Join(summary, "\n")+"\n\nPrompt\n"+f.plan.Prompt), m.detailWidth(), true)
}

func (m Model) previewHeight(height int) int {
	available := height - len(m.spawnMessages(m.detailWidth(), height))
	return max(1, available-len(m.spawnPreviewHeader()))
}

func (m Model) spawnView(height int) string {
	f := m.spawn
	width := m.detailWidth()
	messages := m.spawnMessages(width, height)
	available := max(1, height-len(messages))
	titles := []string{"Repository", "Source branch", "Ticket and name", "Agent", "Blueprint", "Authorization level", "Variables", "Prompt preview", "Launching", "Crew", "Launch destination", "Ghostty window"}
	lines := []string{"Spawn · " + titles[f.step]}
	switch f.step {
	case repoStep, sourceStep:
		input := f.query
		input.Prompt = "Filter  "
		input.Placeholder = "Type to search"
		input.Width = max(1, width-9)
		input.SetCursor(input.Position())
		lines = append(lines, input.View())
		room := max(1, available-len(lines))
		if f.step == repoStep {
			lines = append(lines, f.repositoryView(width, room)...)
		} else {
			lines = append(lines, f.sourceView(width, room)...)
		}
	case identityStep, varsStep:
		labels := []string{"Ticket", "Name", "Type", "Branch"}
		if f.step == varsStep {
			labels = f.vars
		}
		labelWidth := 8
		if f.step == varsStep {
			for _, label := range labels {
				labelWidth = max(labelWidth, ansi.StringWidth(label))
			}
			labelWidth = min(labelWidth, width/3)
		}
		gap := 1
		if available >= 2*len(f.fields)+2 {
			gap = 2
		}
		slots := max(1, (available-1)/gap)
		start := max(0, f.field-slots+1)
		for i := start; i < len(f.fields) && len(lines) < available; i++ {
			v := f.fields[i]
			v.Width = max(1, width-labelWidth-5)
			v.SetCursor(v.Position())
			if i != f.field {
				v.Blur()
			}
			if f.step == identityStep && i == 0 {
				v.Placeholder = "optional"
			}
			value := v.View()
			if f.step == identityStep && i == 2 {
				value = "[feature]  fix"
				if v.Value() == "fix" {
					value = "feature  [fix]"
				}
				if ansi.StringWidth(value) > width-labelWidth-4 {
					value = "[" + v.Value() + "]"
				}
			}
			label := fit(clean(labels[i]), labelWidth)
			label += strings.Repeat(" ", max(0, labelWidth-ansi.StringWidth(label)))
			prefix := "  "
			if i == f.field {
				prefix = "> "
			}
			lines = append(lines, prefix+label+"  "+value)
			if gap == 2 && len(lines) < available {
				lines = append(lines, "")
			}
		}
		if f.step == identityStep && available-len(lines) >= 2 {
			lines = append(lines, "", spawnMuted.Render("Ticket accepts an issue number or GitHub URL."))
		}
	case agentStep, blueprintStep, modeStep, crewStep, destinationStep, windowStep:
		for _, row := range m.spawnChoiceRows(width, max(1, available-1)) {
			line := row.text
			if row.index == f.choice {
				line = spawnAccent.Render(line)
			}
			lines = append(lines, line)
		}
		if f.step == crewStep && len(lines)+1 < available {
			s := f.issue.Suggestion
			lines = append(lines, "", clean(fmt.Sprintf("Suggested by the %s of #%d", s.Source, f.issue.Number)))
		}
	case previewStep:
		lines = m.spawnPreviewHeader()
		v := f.preview
		v.Width, v.Height = width, max(1, available-len(lines))
		lines = append(lines, strings.Split(v.View(), "\n")...)
	case launchStep:
		log := strings.Split(multiline(f.progress), "\n")
		if f.progress == "" {
			log = []string{"Creating worktree and starting agent…"}
		}
		start := max(0, len(log)-max(1, available-1))
		lines = append(lines, log[start:]...)
	}
	lines = lines[:min(len(lines), available)]
	lines = append(lines, messages...)
	for i := range lines {
		lines[i] = fit(lines[i], width)
	}
	if len(lines) > 0 {
		lines[0] = spawnAccent.Render(lines[0])
	}
	return strings.Join(lines, "\n")
}
