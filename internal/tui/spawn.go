package tui

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/thomashartm/motley/internal/agents"
	"github.com/thomashartm/motley/internal/blueprint"
	"github.com/thomashartm/motley/internal/config"
	"github.com/thomashartm/motley/internal/crew"
	"github.com/thomashartm/motley/internal/gh"
	"github.com/thomashartm/motley/internal/member"
	"github.com/thomashartm/motley/internal/terminal"
	"github.com/thomashartm/motley/internal/worktree"
)

const (
	repoStep = iota
	sourceStep
	identityStep
	agentStep
	blueprintStep
	modeStep
	varsStep
	previewStep
	launchStep
	crewStep
	destinationStep
	windowStep
)

type spawnForm struct {
	previewAction       int
	step, choice, field int
	repos               []string
	sources             []worktree.SourceBranch
	blueprints          []blueprint.Blueprint
	windows             []terminal.GhosttyWindow
	query               textinput.Model
	fields              []textinput.Model
	opts                member.SpawnOptions
	manualBranch        bool
	vars                []string
	plan                member.Prepared
	preview             viewport.Model
	err, progress       string
	// issue is the TUI's own lookup; warning explains a skipped one.
	issue   *member.IssueContext
	warning string
}
type spawnLoaded struct {
	cfg          config.Config
	repos        []string
	blueprints   []blueprint.Blueprint
	forBlueprint bool
	err          error
}
type spawnPrepared struct {
	plan member.Prepared
	err  error
}
type spawnSourcesLoaded struct {
	sources []worktree.SourceBranch
	err     error
}
type spawnWindowsLoaded struct {
	windows []terminal.GhosttyWindow
	err     error
}
type spawnWindowPreviewed struct{ err error }
type spawnProgress string
type issueLooked struct {
	issue *member.IssueContext
	err   error
}
type spawnFinished struct {
	manifest member.Manifest
	err      error
}

func spawnPreviewActions() []string { return []string{"Launch", "Edit prompt", "Cancel"} }

func spawnDestinations() []string {
	if runtime.GOOS == "darwin" {
		return []string{"New Ghostty tab", "tmux session"}
	}
	return []string{"tmux session"}
}

func spawnDestination(target string) string {
	if target == "ghostty" {
		return "New Ghostty tab (direct agent)"
	}
	return "tmux session"
}

type promptEdited struct {
	prompt string
	err    error
}
type progressWriter struct{ send func(tea.Msg) }

func (w progressWriter) Write(p []byte) (int, error) {
	if w.send != nil {
		w.send(spawnProgress(string(p)))
	}
	return len(p), nil
}

var _ io.Writer = progressWriter{}

func inputs(values ...string) []textinput.Model {
	result := make([]textinput.Model, len(values))
	for i, v := range values {
		result[i] = textinput.New()
		result[i].Prompt = ""
		result[i].CharLimit = 4096
		result[i].SetValue(v)
	}
	if len(result) > 0 {
		result[0].Focus()
	}
	return result
}
func (f *spawnForm) matches() []string {
	var r []string
	for _, v := range f.repos {
		if match(f.query.Value(), v) {
			r = append(r, v)
		}
	}
	return r
}
func branchPreview(kind, repo, ticket, name string) string {
	slug := func(s string) string {
		var out strings.Builder
		dash := false
		for _, r := range strings.ToLower(s) {
			if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
				if dash && out.Len() > 0 {
					out.WriteByte('-')
				}
				out.WriteRune(r)
				dash = false
			} else {
				dash = true
			}
		}
		return out.String()
	}
	part := slug(filepath.Base(strings.TrimSuffix(repo, "/"))) + "-"
	ticket = spawnTicket(ticket)
	if ticket != "" && !strings.Contains(ticket, "://") {
		part += slug(ticket) + "-"
	}
	return kind + "/" + part + slug(name)
}

// spawnTicket keeps a pasted GitHub issue URL out of generated branch names.
func spawnTicket(ticket string) string {
	ticket = strings.TrimSpace(ticket)
	u, err := url.Parse(ticket)
	if err == nil && u.Scheme == "https" && u.Host == "github.com" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) == 4 && parts[2] == "issues" {
			if _, ok := member.IssueNumber(parts[3]); ok {
				return parts[3]
			}
		}
	}
	return ticket
}

func (m Model) beginSpawn() (tea.Model, tea.Cmd) {
	m.spawn = &spawnForm{step: agentStep, query: inputs("")[0]}
	m.message = ""
	return m, nil
}

func (m Model) loadSpawnRepos() (tea.Model, tea.Cmd) {
	m.spawn.step, m.spawn.choice = repoStep, 0
	m.busy = true
	m.busyText = "Finding repositories…"
	m.message = ""
	return m, func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return spawnLoaded{err: err}
		}
		repos, err := member.DiscoverRepos(cfg.RepositoryRoots())
		if err != nil {
			return spawnLoaded{err: err}
		}
		return spawnLoaded{cfg: cfg, repos: repos}
	}
}
func (m Model) spawnMessage(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.spawn == nil {
		return m, nil
	}
	f := m.spawn
	switch msg := msg.(type) {
	case spawnWindowsLoaded:
		m.busy, m.busyText = false, ""
		f.windows, f.step, f.choice = msg.windows, windowStep, 0
		f.err = ""
		if msg.err != nil {
			f.err = msg.err.Error()
		}
	case spawnWindowPreviewed:
		m.busy, m.busyText = false, ""
		if msg.err != nil {
			f.err = msg.err.Error()
		} else {
			f.err = ""
		}
	case spawnSourcesLoaded:
		m.busy, m.busyText = false, ""
		if msg.err != nil {
			f.err = msg.err.Error()
			return m, nil
		}
		f.sources = msg.sources
		f.step, f.choice = sourceStep, 0
		f.query.SetValue("")
		f.err = ""
	case spawnLoaded:
		m.busy = false
		m.busyText = ""
		if msg.err != nil {
			f.err = msg.err.Error()
			return m, nil
		}
		if msg.forBlueprint {
			f.blueprints = msg.blueprints
			f.step = blueprintStep
			f.choice = 0
		} else {
			m.spawnCfg = msg.cfg
			f.step, f.choice = repoStep, 0
			f.repos = msg.repos
			if len(f.repos) == 0 {
				f.err = "No main repositories found under " + strings.Join(msg.cfg.RepositoryRoots(), ", ")
			}
		}
	case spawnPrepared:
		m.busy = false
		m.busyText = ""
		if msg.err != nil {
			f.err = msg.err.Error()
			return m, nil
		}
		f.plan = msg.plan
		f.step = previewStep
		f.preview = viewport.New(m.detailWidth(), m.previewHeight(m.contentHeight()))
		f.preview.SetContent(m.spawnPreviewContent())
		f.err = ""
	case promptEdited:
		if msg.err != nil {
			f.err = msg.err.Error()
		} else {
			f.plan.Prompt = msg.prompt
			f.plan.HasPrompt = true
			f.preview.SetContent(m.spawnPreviewContent())
			f.err = ""
		}
	case issueLooked:
		m.busy = false
		m.busyText = ""
		// The TUI's lookup is final: Prepare must not repeat it.
		f.opts.Issue, f.opts.NoGH, f.issue = msg.issue, true, msg.issue
		if msg.err != nil {
			f.warning = "Issue lookup skipped: " + gh.Hint(msg.err)
		}
		f.choice = 0
		return m.spawnCrewOrBlueprints()
	case spawnProgress:
		f.progress += string(msg)
		if len(f.progress) > 8192 {
			f.progress = f.progress[len(f.progress)-8192:]
		}
	case spawnFinished:
		m.busy = false
		m.busyText = ""
		if msg.err != nil && msg.manifest.ID == "" {
			f.err = msg.err.Error()
			f.step = previewStep
			return m, nil
		}
		directGhostty := msg.manifest.LaunchTarget == "ghostty"
		m.spawn = nil
		m.query.SetValue("")
		m.group = "attention"
		m.overview, m.tableFocus = false, false
		m.panel = listPanel
		m.focusID = msg.manifest.ID
		rows := make([]member.Row, 0, len(m.allRows)+1)
		for _, r := range m.allRows {
			if r.ID != msg.manifest.ID {
				rows = append(rows, r)
			}
		}
		m.allRows = append(rows, member.Row{Manifest: msg.manifest, Alive: msg.err == nil, External: directGhostty, Status: msg.manifest.LaunchStatus(false), Since: msg.manifest.CreatedAt.Unix()})
		m.applyFilter()
		for i, r := range m.rows {
			if r.ID == msg.manifest.ID {
				m.selected = i
			}
		}
		m.updateDetail()
		m.message = "Created " + msg.manifest.ID
		if msg.err != nil {
			m.message += "; " + msg.err.Error()
		} else if directGhostty {
			m.message += " in a new Ghostty tab"
		}
		return m, m.requestDetail(true)
	}
	return m, nil
}
func (f *spawnForm) suggestion() *member.CrewSuggestion {
	if f.issue == nil {
		return nil
	}
	return f.issue.Suggestion
}

// spawnCrews lists existing crews to offer, minus one the suggestion already names.
func (m Model) spawnCrews() []crew.Crew {
	var crews []crew.Crew
	for _, c := range m.crews {
		if s := m.spawn.suggestion(); s == nil || s.CrewID != c.ID {
			crews = append(crews, c)
		}
	}
	return crews
}

// spawnCrewOrBlueprints asks for a crew when one is suggested or any exists.
func (m Model) spawnCrewOrBlueprints() (tea.Model, tea.Cmd) {
	f := m.spawn
	if f.opts.Crew == "" && (f.suggestion() != nil || len(m.crews) > 0) {
		f.step, f.choice = crewStep, 0
		return m, nil
	}
	return m.loadSpawnBlueprints()
}

func (m Model) loadSpawnBlueprints() (tea.Model, tea.Cmd) {
	f := m.spawn
	f.choice = 0
	cfg, repo, agent := m.spawnCfg, f.opts.Repo, f.opts.Agent
	m.busy = true
	m.busyText = "Loading blueprints…"
	return m, func() tea.Msg {
		path, err := member.ResolveRepo(cfg.RepositoryRoots(), repo)
		if err != nil {
			return spawnLoaded{forBlueprint: true, err: err}
		}
		bs, err := blueprint.Discover(path, filepath.Base(repo))
		var filtered []blueprint.Blueprint
		for _, b := range bs {
			if b.Agent == agent || len(b.Args) == 0 {
				filtered = append(filtered, b)
			}
		}
		return spawnLoaded{forBlueprint: true, blueprints: filtered, err: err}
	}
}
func (m Model) prepareSpawn() (tea.Model, tea.Cmd) {
	f := m.spawn
	opts := f.opts
	cfg := m.spawnCfg
	if f.step == varsStep {
		opts.Vars = nil
		for i, key := range f.vars {
			opts.Vars = append(opts.Vars, key+"="+f.fields[i].Value())
		}
	}
	m.busy = true
	m.busyText = "Rendering prompt…"
	f.err = ""
	return m, func() tea.Msg { p, err := member.Prepare(cfg, opts); return spawnPrepared{p, err} }
}
func (m Model) updateSpawn(msg tea.Msg) (tea.Model, tea.Cmd) {
	f := m.spawn
	if key, ok := msg.(tea.KeyMsg); ok {
		k := key.String()
		if k == "esc" {
			m.spawn = nil
			return m, nil
		}
		if f.step == windowStep {
			switch k {
			case "r":
				return m.loadGhosttyWindows()
			case "p":
				if f.choice < len(f.windows) {
					id := f.windows[f.choice].ID
					m.busy, m.busyText = true, "Previewing window; return to Motley to continue…"
					return m, func() tea.Msg { return spawnWindowPreviewed{terminal.PreviewGhosttyWindow(id)} }
				}
				return m, nil
			}
		}
		if f.step == previewStep {
			actionCount := len(spawnPreviewActions())
			switch k {
			case "right", "tab":
				f.previewAction = (f.previewAction + 1) % actionCount
			case "left", "shift+tab":
				f.previewAction = (f.previewAction + actionCount - 1) % actionCount
			case "enter":
				if f.previewAction == 1 {
					return m, m.editPrompt()
				}
				if f.previewAction == 2 {
					m.spawn = nil
					return m, nil
				}
				m.busy = true
				m.busyText = "Spawning…"
				f.step = launchStep
				plan, send := f.plan, m.sendMsg
				return m, func() tea.Msg {
					created, err := member.SpawnPrepared(plan, progressWriter{send})
					return spawnFinished{manifest: created, err: err}
				}
			case "e":
				return m, m.editPrompt()
			case "pgup", "pgdown", "up", "down", "j", "k":
				f.preview, _ = f.preview.Update(msg)
			}
			return m, nil
		}
		if f.step == repoStep {
			switch k {
			case "up":
				f.choice = max(0, f.choice-1)
			case "down":
				f.choice = min(f.choice+1, max(0, len(f.matches())-1))
			case "enter":
				matches := f.matches()
				if len(matches) == 0 {
					return m, nil
				}
				f.opts.Repo = matches[f.choice]
				f.err = ""
				m.busy, m.busyText = true, "Loading source branches…"
				cfg, repo := m.spawnCfg, f.opts.Repo
				return m, func() tea.Msg {
					path, err := member.ResolveRepo(cfg.RepositoryRoots(), repo)
					if err != nil {
						return spawnSourcesLoaded{err: err}
					}
					sources, err := worktree.SourceBranches(path)
					return spawnSourcesLoaded{sources: sources, err: err}
				}
			default:
				f.query, _ = f.query.Update(msg)
				f.choice = 0
			}
			return m, nil
		}
		if f.step == sourceStep {
			switch k {
			case "up":
				f.choice = max(0, f.choice-1)
			case "down":
				f.choice = min(f.choice+1, max(0, len(f.sourceMatches())-1))
			case "enter":
				matches := f.sourceMatches()
				if len(matches) == 0 {
					return m, nil
				}
				f.opts.SourceRef = matches[f.choice].Ref
				f.fields = inputs("", "", "feature", branchPreview("feature", f.opts.Repo, "", ""))
				f.field, f.step = 0, identityStep
				f.err = ""
			default:
				f.query, _ = f.query.Update(msg)
				f.choice = 0
			}
			return m, nil
		}
		if f.step == agentStep || f.step == blueprintStep || f.step == modeStep || f.step == crewStep || f.step == destinationStep || f.step == windowStep {
			size := 3
			switch f.step {
			case windowStep:
				size = len(f.windows) + 1
			case destinationStep:
				size = len(spawnDestinations())
			case crewStep:
				size = len(m.spawnChoiceLabels())
			case blueprintStep:
				size = len(f.blueprints) + 1
			case modeStep:
				size = len(agents.Modes(f.opts.Agent)) + 1
			}
			switch k {
			case "j", "down":
				f.choice = min(f.choice+1, size-1)
			case "k", "up":
				f.choice = max(0, f.choice-1)
			case "enter":
				if f.step == windowStep {
					f.opts.GhosttyWindow = "new"
					if f.choice < len(f.windows) {
						f.opts.GhosttyWindow = f.windows[f.choice].ID
					}
					f.err = ""
					return m.loadSpawnRepos()
				}
				if f.step == crewStep {
					choice := f.choice
					if s := f.suggestion(); s != nil {
						if choice == 0 {
							if s.CrewID != "" {
								f.opts.Crew = s.CrewID
							} else {
								f.opts.CreateCrew = true
							}
							return m.loadSpawnBlueprints()
						}
						choice--
					}
					if choice == 0 {
						f.opts.SkipSuggestion = true
					} else {
						f.opts.Crew = m.spawnCrews()[choice-1].ID
					}
					return m.loadSpawnBlueprints()
				}
				if f.step == agentStep {
					f.opts.Agent = []string{"claude", "codex", "opencode"}[f.choice]
					f.step, f.choice = destinationStep, 0
					return m, nil
				}
				if f.step == destinationStep {
					f.opts.LaunchTarget = "tmux"
					if runtime.GOOS == "darwin" && f.choice == 0 {
						f.opts.LaunchTarget = "ghostty"
						return m.loadGhosttyWindows()
					}
					return m.loadSpawnRepos()
				}

				if f.step == modeStep {
					f.opts.Mode = ""
					if f.choice > 0 {
						mode := agents.Modes(f.opts.Agent)[f.choice-1]
						if arg := agents.ConflictingArg(f.blueprintArgs(), mode.Args); arg != "" {
							f.err = "Blueprint sets " + arg + "; choose its permissions or restart without that blueprint."
							return m, nil
						}
						f.opts.Mode = mode.Name
					}
					f.err = ""
				} else {
					if f.choice > 0 {
						b := f.blueprints[f.choice-1]
						f.opts.Blueprint, f.vars = b.Name, b.Vars
					}
					// Always ask, including when permissions come from a blueprint.
					f.step = modeStep
					f.choice = 0
					return m, nil
				}
				if len(f.vars) > 0 {
					f.fields = inputs(make([]string, len(f.vars))...)
					f.field = 0
					f.step = varsStep
					return m, nil
				}
				return m.prepareSpawn()
			}
			return m, nil
		}
		if k == "tab" || k == "shift+tab" || k == "up" || k == "down" {
			f.fields[f.field].Blur()
			delta := 1
			if k == "shift+tab" || k == "up" {
				delta = -1
			}
			f.field = (f.field + delta + len(f.fields)) % len(f.fields)
			return m, f.fields[f.field].Focus()
		}
		if k == "enter" || k == "ctrl+s" {
			if f.step == varsStep {
				return m.prepareSpawn()
			}
			ticket, name, branch := spawnTicket(f.fields[0].Value()), strings.TrimSpace(f.fields[1].Value()), f.fields[3].Value()
			if name == "" || (!f.manualBranch && strings.HasSuffix(branch, "-")) {
				f.err = "Enter a name containing letters or digits."
				return m, nil
			}
			if ticket != "" {
				if err := member.CheckID(ticket); err != nil {
					f.err = err.Error()
					return m, nil
				}
			}
			f.opts.Ticket, f.opts.Name, f.opts.Branch = ticket, name, branch
			f.choice, f.err = 0, ""
			if _, ok := member.IssueNumber(ticket); !ok || m.github == nil {
				return m.spawnCrewOrBlueprints()
			}
			cfg, client, repo := m.spawnCfg, m.github, f.opts.Repo
			m.busy = true
			m.busyText = "Looking up issue #" + strings.TrimPrefix(strings.TrimSpace(ticket), "#") + "…"
			return m, func() tea.Msg {
				ctx, cancel := context.WithTimeout(context.Background(), member.LookupTimeout)
				defer cancel()
				issue, err := member.LookupIssue(ctx, client, cfg, repo, ticket)
				return issueLooked{issue, err}
			}
		}
	}
	if f.step == identityStep && f.field == 2 {
		if key, ok := msg.(tea.KeyMsg); ok && (key.String() == "left" || key.String() == "right" || key.String() == " ") {
			kind := "fix"
			if f.fields[2].Value() == "fix" {
				kind = "feature"
			}
			f.fields[2].SetValue(kind)
			if f.manualBranch {
				branch := f.fields[3].Value()
				if _, suffix, found := strings.Cut(branch, "/"); found {
					branch = suffix
				}
				f.fields[3].SetValue(kind + "/" + branch)
			} else {
				f.fields[3].SetValue(branchPreview(kind, f.opts.Repo, f.fields[0].Value(), f.fields[1].Value()))
			}
		}
		return m, nil
	}
	if len(f.fields) > 0 {
		old := f.fields[f.field].Value()
		var cmd tea.Cmd
		f.fields[f.field], cmd = f.fields[f.field].Update(msg)
		if f.step == identityStep && old != f.fields[f.field].Value() {
			if f.field == 3 {
				f.manualBranch = true
			}
			if !f.manualBranch {
				f.fields[3].SetValue(branchPreview(f.fields[2].Value(), f.opts.Repo, f.fields[0].Value(), f.fields[1].Value()))
			}
		}
		return m, cmd
	}
	return m, nil
}

func (m Model) loadGhosttyWindows() (tea.Model, tea.Cmd) {
	m.spawn.step = windowStep
	m.busy, m.busyText = true, "Finding Ghostty windows…"
	return m, func() tea.Msg {
		windows, err := terminal.GhosttyWindows()
		return spawnWindowsLoaded{windows, err}
	}
}
func (m Model) editPrompt() tea.Cmd {
	f := m.spawn
	file, err := os.CreateTemp("", "motley-prompt-*.md")
	if err != nil {
		return func() tea.Msg { return promptEdited{err: err} }
	}
	path := file.Name()
	_, err = file.WriteString(blueprint.PromptHeader + f.plan.Prompt)
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
		return func() tea.Msg { return promptEdited{err: err} }
	}
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	// EDITOR is the user's shell command; the prompt and path are never shell code.
	cmd := exec.Command("/bin/sh", "-c", "exec "+editor+` "$1"`, "motley-editor", path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer func() { _ = os.Remove(path) }()
		if err != nil {
			return promptEdited{err: err}
		}
		prompt, err := blueprint.ReadPrompt(path)
		return promptEdited{prompt, err}
	})
}

func (m Model) previewSource() string {
	if ref := m.spawn.plan.SourceRef; ref != "" {
		label := ref
		for _, source := range m.spawn.sources {
			if source.Ref == ref {
				label = source.Label()
				break
			}
		}
		return "Source: " + label
	}
	return ""
}

// previewContext names the issue and crew a prepared spawn uses, if any.
func (m Model) previewContext() string {
	var parts []string
	if i := m.spawn.plan.Issue; i != nil {
		parts = append(parts, fmt.Sprintf("#%d %s", i.Number, i.Title))
	}
	if c := m.spawn.plan.NewCrew; c != nil {
		parts = append(parts, "new crew "+c.Title)
	} else if c, ok := crew.Find(m.crews, m.spawn.plan.Manifest.Crew); ok {
		parts = append(parts, "crew "+c.Title)
	}
	return strings.Join(parts, " · ")
}
