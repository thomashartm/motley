package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/thomashartm/motley/internal/blueprint"
)

type blueprintPage int

const (
	blueprintList blueprintPage = iota
	blueprintActions
	blueprintRaw
	blueprintArguments
	blueprintPrompt
)

type blueprintDialog struct {
	files          []blueprint.File
	dir            string
	page           blueprintPage
	cursor, action int
	argument       int
	labels, values []string
	input          textarea.Model
	preview        viewport.Model
	prompt         string
}

type blueprintsLoaded struct {
	files []blueprint.File
	dir   string
	err   error
}

type blueprintEdited struct{ err error }
type blueprintCopied struct{ err error }

func loadBlueprints() tea.Msg {
	dir, err := blueprint.GlobalDir()
	if err != nil {
		return blueprintsLoaded{err: err}
	}
	files, err := blueprint.GlobalFiles()
	return blueprintsLoaded{files: files, dir: dir, err: err}
}

func (m Model) beginBlueprints() (tea.Model, tea.Cmd) {
	m.blueprints = &blueprintDialog{preview: viewport.New(1, 1)}
	m.busy, m.busyText, m.message = true, "Loading global templates…", ""
	return m, loadBlueprints
}

func (m Model) blueprintMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.blueprints == nil {
		return m, nil
	}
	d := m.blueprints
	switch msg := msg.(type) {
	case blueprintsLoaded:
		m.busy, m.busyText = false, ""
		if msg.err != nil {
			m.message = msg.err.Error()
			return m, nil
		}
		path := ""
		if len(d.files) > 0 {
			path = d.files[d.cursor].Path
		}
		d.files, d.dir = msg.files, msg.dir
		d.cursor = max(0, min(d.cursor, len(d.files)-1))
		for i, file := range d.files {
			if file.Path == path {
				d.cursor = i
			}
		}
		if len(d.files) == 0 {
			d.page = blueprintList
		}
		m.resizeBlueprints()
	case blueprintEdited:
		m.busy, m.busyText = true, "Reloading templates…"
		m.message = "Editor opened. After saving, use Reload to read any further changes."
		if msg.err != nil {
			m.message = "Editor failed: " + msg.err.Error()
		}
		return m, loadBlueprints
	case blueprintCopied:
		m.message = "Prompt copied. Paste it into your agent."
		if d.page == blueprintRaw {
			m.message = "Raw template copied."
		}
		if msg.err != nil {
			m.message = "Copy failed: " + msg.err.Error()
		}
	}
	return m, nil
}

func (d *blueprintDialog) choices() []string {
	if d.page == blueprintActions {
		return []string{"View raw template", "Generate prompt", "Edit in vi", "Edit in default editor", "Reload from disk", "Back to templates"}
	}
	labels := make([]string, len(d.files))
	for i, file := range d.files {
		labels[i] = file.Name + " · " + file.Agent
		if file.Err != nil {
			labels[i] += " · invalid (edit to repair)"
		} else if file.Description != "" {
			labels[i] += " · " + file.Description
		}
	}
	return labels
}

func (d *blueprintDialog) selection() int {
	if d.page == blueprintActions {
		return d.action
	}
	return d.cursor
}

func (m Model) updateBlueprints(msg tea.Msg) (tea.Model, tea.Cmd) {
	d := m.blueprints
	if key, ok := msg.(tea.KeyMsg); ok {
		k := key.String()
		if k == "esc" {
			m.message = ""
			switch d.page {
			case blueprintList:
				m.blueprints = nil
			case blueprintActions:
				d.page = blueprintList
			case blueprintPrompt:
				d.page = blueprintArguments
				return m, d.input.Focus()
			default:
				d.page = blueprintActions
				d.input.Blur()
			}
			return m, nil
		}
		switch d.page {
		case blueprintArguments:
			switch k {
			case "tab", "shift+tab":
				d.values[d.argument] = d.input.Value()
				step := 1
				if k == "shift+tab" {
					step = -1
				}
				d.argument = (d.argument + step + len(d.labels)) % len(d.labels)
				d.input.SetValue(d.values[d.argument])
				d.input.CursorEnd()
				return m, nil
			case "ctrl+s":
				d.values[d.argument] = d.input.Value()
				prompt, err := d.render()
				if err != nil {
					m.message = err.Error()
					return m, nil
				}
				d.prompt, d.page = prompt, blueprintPrompt
				d.input.Blur()
				m.message = "Review the prompt, then copy it into your external agent."
				m.resizeBlueprints()
				d.preview.GotoTop()
				return m, nil
			}
		case blueprintRaw, blueprintPrompt:
			if k == "c" {
				return m.copyBlueprint()
			}
		default:
			if k == "r" {
				m.busy, m.busyText = true, "Reloading templates…"
				return m, loadBlueprints
			}
			choices := d.choices()
			if len(choices) == 0 {
				return m, nil
			}
			selected := d.selection()
			switch k {
			case "up", "k", "shift+tab":
				selected = max(0, selected-1)
			case "down", "j", "tab":
				selected = min(len(choices)-1, selected+1)
			case "enter":
				if d.page == blueprintList {
					d.page, d.action = blueprintActions, 0
					m.message = d.files[d.cursor].Path
					return m, nil
				}
				return m.blueprintAction()
			}
			if d.page == blueprintList {
				d.cursor = selected
			} else {
				d.action = selected
			}
			return m, nil
		}
	}
	var cmd tea.Cmd
	switch d.page {
	case blueprintArguments:
		d.input, cmd = d.input.Update(msg)
	case blueprintRaw, blueprintPrompt:
		d.preview, cmd = d.preview.Update(msg)
	}
	return m, cmd
}

func (m Model) blueprintAction() (tea.Model, tea.Cmd) {
	d := m.blueprints
	file := d.files[d.cursor]
	switch d.action {
	case 0:
		d.page = blueprintRaw
		m.resizeBlueprints()
		d.preview.GotoTop()
	case 1:
		if file.Err != nil {
			m.message = file.Err.Error()
			return m, nil
		}
		d.beginArguments()
		m.message = "Fill variables and any context the template uses; unused fields can stay empty."
		m.resizeBlueprints()
		return m, d.input.Focus()
	case 2, 3:
		cmd, err := blueprintEditor(file.Path, d.action == 2, runtime.GOOS)
		if err != nil {
			m.message = err.Error()
			return m, nil
		}
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return blueprintEdited{err: err} })
	case 4:
		m.busy, m.busyText = true, "Reloading templates…"
		return m, loadBlueprints
	case 5:
		d.page = blueprintList
	}
	return m, nil
}

// Pass the filename as a separate argument even when the configured editor is
// a shell command (e.g. code --wait). A template path is never shell source.
func blueprintEditor(path string, vi bool, platform string) (*exec.Cmd, error) {
	if vi {
		return exec.Command("vi", path), nil
	}
	for _, env := range []string{"VISUAL", "EDITOR"} {
		if editor := strings.TrimSpace(os.Getenv(env)); editor != "" {
			return exec.Command("/bin/sh", "-c", "exec "+editor+` "$1"`, "motley-editor", path), nil
		}
	}
	switch platform {
	case "darwin":
		return exec.Command("open", "-t", path), nil
	case "linux":
		return exec.Command("xdg-open", path), nil
	default:
		return nil, fmt.Errorf("set VISUAL or EDITOR, or choose Edit in vi")
	}
}

func (d *blueprintDialog) beginArguments() {
	d.labels = nil
	for _, name := range d.files[d.cursor].Vars {
		d.labels = append(d.labels, "Vars."+name)
	}
	d.labels = append(d.labels, "Repo", "Branch", "Base", "Ticket", "Name", "Worktree", "Crew.Title", "Crew.URL", "Crew.Kind", "Issue.Title", "Issue.Body", "Issue.URL")
	d.values = make([]string, len(d.labels))
	d.argument, d.page = 0, blueprintArguments
	d.input = textarea.New()
	d.input.Prompt = ""
	d.input.ShowLineNumbers = false
	d.input.CharLimit = blueprint.MaxPromptBytes
	d.input.MaxHeight = 0
	d.input.Placeholder = "Enter value (optional)"
}

func (d *blueprintDialog) render() (string, error) {
	vars := map[string]string{}
	b := d.files[d.cursor].Blueprint
	for i, name := range b.Vars {
		vars[name] = d.values[i]
	}
	v := d.values[len(b.Vars):]
	return b.Render(blueprint.Data{
		Repo: v[0], Branch: v[1], Base: v[2], Ticket: v[3], Name: v[4], Worktree: v[5],
		Crew:  blueprint.Crew{Title: v[6], URL: v[7], Kind: v[8]},
		Issue: blueprint.Issue{Title: v[9], Body: v[10], URL: v[11]}, Vars: vars,
	})
}

func (m Model) resizeBlueprints() {
	d := m.blueprints
	width, height := m.detailWidth(), m.contentHeight()
	d.preview.Width, d.preview.Height = width, max(1, height-1)
	if d.page == blueprintArguments {
		d.input.SetWidth(width)
		d.input.SetHeight(max(1, height-2))
	}
	text := d.prompt
	if d.page == blueprintRaw && len(d.files) > 0 {
		text = d.files[d.cursor].Source
	}
	d.preview.SetContent(ansi.Hardwrap(multiline(text), width, true))
}

func (m Model) copyBlueprint() (tea.Model, tea.Cmd) {
	d := m.blueprints
	text := d.prompt
	if d.page == blueprintRaw {
		text = d.files[d.cursor].Source
	}
	if m.copyText == nil {
		m.message = "Clipboard is unavailable."
		return m, nil
	}
	copyText, client := m.copyText, m.client
	if m.monitor {
		client = m.activeMonitorClient()
	}
	return m, func() tea.Msg { return blueprintCopied{err: copyText(client, text)} }
}

func (m Model) blueprintsView(height int) string {
	d := m.blueprints
	width := m.detailWidth()
	title := "Global prompt templates"
	if d.page != blueprintList && len(d.files) > 0 {
		title = d.files[d.cursor].Name
	}
	switch d.page {
	case blueprintRaw:
		return fit("Raw: "+title, width) + "\n" + d.preview.View()
	case blueprintPrompt:
		return fit("Prompt: "+title, width) + "\n" + d.preview.View()
	case blueprintArguments:
		return fit(fmt.Sprintf("Arguments %d/%d: %s", d.argument+1, len(d.labels), title), width) + "\n" + fit(d.labels[d.argument], width) + "\n" + d.input.View()
	}
	lines := []string{fit(title, width)}
	choices := d.choices()
	if len(choices) == 0 {
		text := "No global templates. Add .md files in " + filepath.Clean(d.dir) + "; r reloads."
		lines = append(lines, strings.Split(ansi.Wrap(text, width, ""), "\n")...)
	} else {
		selected := d.selection()
		start := max(0, selected-max(1, height-1)+1)
		for i := start; i < len(choices) && len(lines) < height; i++ {
			lines = append(lines, fit(control(clean(choices[i]), i == selected), width))
		}
	}
	return strings.Join(lines[:min(len(lines), height)], "\n")
}

func (m Model) blueprintMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if m.busy || msg.X <= m.listWidth()+2 || msg.X >= m.width-1 || msg.Y < 2 || msg.Y >= 2+m.panelHeight() {
		return m, nil
	}
	d := m.blueprints
	if d.page == blueprintRaw || d.page == blueprintPrompt {
		return m.updateBlueprints(msg)
	}
	if d.page == blueprintArguments {
		return m, nil
	}
	if msg.Button == tea.MouseButtonWheelDown {
		return m.updateBlueprints(tea.KeyMsg{Type: tea.KeyDown})
	}
	if msg.Button == tea.MouseButtonWheelUp {
		return m.updateBlueprints(tea.KeyMsg{Type: tea.KeyUp})
	}
	if msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return m, nil
	}
	y := m.panelContentY(msg.Y) - 1
	start := max(0, d.selection()-max(1, m.contentHeight()-1)+1)
	index := start + y
	if y >= 0 && index < len(d.choices()) {
		if d.page == blueprintList {
			d.cursor = index
		} else {
			d.action = index
		}
		return m.updateBlueprints(tea.KeyMsg{Type: tea.KeyEnter})
	}
	return m, nil
}
