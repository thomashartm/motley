# Advanced usage

[Back to the README](../README.md)

## Start agents from the CLI

```sh
mtly spawn --repo api --branch feat/412-fx-cache --ticket 412 --name "FX cache" --agent codex
mtly ls
mtly attach <member-id>
```

Use the member ID printed by `spawn` or listed by `ls`.

| Option | Purpose |
| --- | --- |
| `--repo api` | Select `api` directly under any configured `repos_roots` entrypoint. |
| `--agent claude\|codex\|opencode` | Choose an agent; Claude is the default without a blueprint. |
| `--detach` | Launch without attaching. |
| `--crew <crew-id>` | Assign the member to a crew. |
| `--blueprint <name>` | Start with a saved prompt template. |
| `--no-gh` | Skip the GitHub issue lookup. |
| `--create-crew` | Create the crew suggested by a GitHub issue. |

The picker groups repositories by entrypoint in config order and shows every
repository as `name (full path)`. Long paths wrap. If names repeat across roots,
choose explicitly in the CLI: `--repo ~/projects/aderis/api`. Repeated paths
to the same physical repository appear only once. All entrypoints must be readable.

In the spawn form, branch suggestions use `feature/<repo>-<ticket>-<name>`.
The ticket is optional; a pasted GitHub issue URL contributes only its issue
number. Tab to **Branch type** and use **←/→** to choose **feature** or **fix**.
The **Branch** field remains editable.

Launching creates and pushes a new branch. Local `.env`, `.env.*` and
`graphify-out` artifacts are copied into the new worktree.
Run `mtly <command> --help` for the available options.

## Existing sessions

```sh
mtly import --list
mtly import <session-id> --name "Existing work"
mtly import --agent codex --list
mtly import --agent codex <session-id> --crew <crew-id>
```

| Provider | Requirement | Opening the imported agent |
| --- | --- | --- |
| Claude | `claude agents --json` must work. | Motley shows its original directory; switch to that terminal yourself. |
| Codex | A local shared app-server with a Unix socket must be running. | Motley opens a client to the existing conversation. |

Codex import excludes standalone `--no-daemon`, remote, ephemeral and subagent
sessions. Import does not restart the agent or create a worktree.

### Switch the tracked Claude session

Use this when a member follows the wrong terminal or conversation:

1. Select the imported Claude member.
2. Press **Shift-S**, or choose **Actions → Switch tracked session**.
3. Select a live, unregistered session in the same checkout.

The member keeps its name, crew and history. Both agents keep running. A former
Motley terminal is renamed `untracked-…` and remains available through tmux.

```sh
mtly import --replace <member-id> --list
mtly import <session-id> --replace <member-id>
```

### Adopt a linked worktree

Inside the existing worktree's tmux session, run:

```sh
mtly adopt --agent claude --name "Existing work"
```

Follow the printed environment and restart instructions. Adoption manages the
whole tmux session, including its other panes.

## Crews and gigs

```sh
mtly crew add --title "Banking" --gig "Ship FX caching" --color blue
mtly crew list
mtly crew assign <member-id> <crew-id>
mtly crew edit <crew-id> --gig "Roll out payments"
mtly crew assign <member-id> none
```

Use the crew ID printed by `crew add` or shown by `crew list`.
Members inherit their crew's colour unless overridden. `--gig ""` clears a gig.

To name a crew from a GitHub issue or project, omit `--title`:

```sh
mtly crew add --url https://github.com/acme/api/issues/400
mtly crew add --url https://github.com/orgs/acme/projects/7
```

## GitHub issues and pull requests

Install and authenticate `gh` for GitHub actions. Without it, you can still use Motley.

| Task | How |
| --- | --- |
| Associate an issue | Pass a numeric `--ticket`, such as `412`, when spawning. |
| Open links | Press **b**, or click a ticket, branch or compare link. |
| Refresh PR checks and reviews | Press **u** for one member or **U** for all. |
| Create, open or mark a PR ready | Press **P** and choose an action. |

GitHub state refreshes only when requested. Creating a PR uses
`gh pr create --fill`. Retiring a member leaves its PR open.

A numeric ticket fetches the issue title and body once. The issue's parent,
otherwise its milestone, suggests a crew. Choose the suggested crew in the spawn
form or pass `--create-crew`. Existing crews with a matching URL are reused.
Explicit ticket URLs can point to other trackers.

## Claude permission modes

Choose a mode in the spawn form or pass `--mode`:

```sh
mtly spawn --repo api --branch feat/example --mode sandbox
```

Available modes: `manual`, `acceptEdits`, `plan`, `auto`, `dontAsk`,
`bypassPermissions`, `sandbox`. Without a mode, Claude uses its configured default.
A blueprint that sets a permission mode cannot also use `--mode`.

`sandbox` accepts edits and confines Bash to the worktree and temporary files.
Linux requires `bubblewrap` and `socat`; launch fails if sandboxing is unavailable.
Claude can still request permission to run a command outside the sandbox.

## Blueprints

1. Create a Markdown template in `~/.config/motley/blueprints/` or the main
   repository's `.motley/blueprints/`.
2. Use [feature-plan-first.md](../examples/blueprints/feature-plan-first.md) as a starting point.
3. Validate it, then choose it when spawning:

```sh
mtly blueprint list --repo api
mtly blueprint validate --repo api
mtly spawn --repo api --branch feat/example --blueprint feature-plan-first --var 'constraints=Keep it small'
```

Repository templates take precedence. In the spawn form, choose the agent, then
its compatible blueprint, enter variables and review the rendered prompt before
launching. The preview's **e** key edits that launch's prompt in `$EDITOR` without
changing the reusable template. Revive does not replay the initial prompt.

### Manage templates and generate prompts for external agents

Open **Main actions → Prompt templates (f)**. The browser lists every global
`.md` template, including templates restricted to particular repositories.
Select one to:

- **View raw template:** read the original Markdown and TOML header; **c** copies it.
- **Generate prompt:** enter the variables declared in the header's `vars` list,
  followed by any repository, branch, ticket, name, worktree, crew or issue context
  the template uses. Unused fields can stay empty. **Tab / Shift-Tab** moves between
  fields, **Enter** adds a newline, and **Ctrl-s** generates the prompt. Review it,
  then press **c** to copy the full text. **Esc** returns to the arguments to revise
  them. Paste the result into the external agent's own conversation.
- **Edit in vi** or **Edit in default editor:** edit the reusable template file.
  Default editor uses `$VISUAL`, then `$EDITOR`. If neither is set, Motley opens
  the macOS default text editor or Linux's default application for Markdown.
  Commands with arguments work, for example `VISUAL='code --wait'`.
- **Reload from disk:** read changes after saving in an editor that opens a separate
  window. Motley also reloads when an editor command returns.

Generation works without a member or repository and does not start an agent or
send it any text. Issue fields are entered manually in this flow. The clipboard
uses the same terminal/tmux mechanism as Copy message; terminal clipboard access
must be enabled. Templates with parse errors remain listed so they can be edited.
The global directory is `$XDG_CONFIG_HOME/motley/blueprints` when set, otherwise
`~/.config/motley/blueprints`. Repository overrides are applied during spawning;
the global manager edits the global files themselves.

GitHub-backed templates can use `{{.Issue.Title}}`, `{{.Issue.Body}}` and
`{{.Issue.URL}}`. Issue bodies are limited to 32 KiB.

## Navigation and terminal tabs

| Task | Control |
| --- | --- |
| Select a row or field | **↑/↓** or **j/k** outside text fields |
| Open List / Details / Actions | **1 / 2 / 3** |
| Move between panels | **←/→** |
| Open main actions | **Home** |
| Scroll details | Mouse wheel or **Page Up / Page Down** |
| Send a reply | **i**; approve permissions in the agent itself |
| Send a member to an attached work tab | **t** |
| Choose where Open agent opens agents | **p** |
| Leave a dialog | **Esc** |
| Close the overview or detach the monitor | **q** |

In crew view, **→** expands a crew, then enters its member table. **←/Esc** returns
to the list; **←** collapses it. **h** shows inactive crews.

In editors, use **↑/↓** to select fields and **←/→** to move the text cursor or
change a colour/crew selector. Choose **Save** or **Cancel**; **Ctrl-s** also saves.

### Keep the monitor in a separate tab

1. Run `mtly monitor` in one terminal tab.
2. Open another tab; run `mtly ls`, then `mtly attach <member-id>`.
3. Use **Open agent** in the monitor to switch the attached work tab.

Attaching shares the existing session. **t** selects an attached tab; it does not
create one. In Ghostty on macOS, **Cmd-T** opens a new tab with the default bindings.

The monitor header shows where **Open agent** goes, and the key that changes it:

| Header | Meaning |
| --- | --- |
| `opens in: this tab (p)` | No work tab is attached; the monitor tab switches to the agent and the monitor keeps running. |
| `opens in: <tab> (p)` | The most recently active work tab. |
| `opens in: <tab>, pinned (p)` | The work tab you pinned with **p**. |
| `opens in: pinned tab gone (p)` | The pinned tab detached; press **p** to choose another or Automatic. |
| `opens in: no tab (p)` | Several monitor tabs and no work tab; detach one or attach a work tab. |

To open agents in a separate tab, attach a work tab as above. **p** then lists it;
Automatic follows the most recently active work tab. `group: attention (g)` shows
the list grouping; **g** cycles attention, crew and repository.

### tmux shortcuts

With Motley's default configuration, press **Ctrl-a**, release, then the next key.
Use your own prefix if you changed it.

| Keys after Ctrl-a | Action |
| --- | --- |
| **h** | Open the Motley popup. |
| **m** | Toggle between the monitor and the previous session. |
| **d** | Detach to the shell; keep the agent running. |
| **w** | Pick a tmux window. |
| **n / p** | Next / previous window. |
| **Arrow key** | Select a pane. |
| **[** | Enter scrollback; **q** leaves it. |
| **Ctrl-a** | Send a literal Ctrl-a to the agent. |

In agent panes on local macOS, drag to select and copy text; paste with **Cmd-V**.
Over SSH, `pbcopy` writes to the Mac running tmux. Linux uses its existing
clipboard setup. In the Motley overview or monitor, drag to select Actions text
or complete member rows, then press **Ctrl+C** to copy through the terminal
clipboard. **Esc** clears the selection. Use tmux scrollback for other pane text.

## Stop, resume and retire

```sh
mtly revive <member-id>
mtly retire <member-id>
mtly retire <member-id> --keep-branch
```

- **Revive:** requires an existing worktree and a missing Motley tmux session.
  If the agent ended but its tmux shell remains, restart it in that shell.
- **Resume:** uses the last recorded agent session ID; without one, starts fresh.
- **Retire managed work:** removes the session, worktree and local branch, except
  protected branches or when `--keep-branch` is used. Remote branches remain.
- **Force retirement:** `--force` discards uncommitted or unpushed work.
- **Retire imported Claude:** stops the agent and removes its entry; keeps files and branches.
- **Retire imported Codex:** closes Motley's client and removes its entry; keeps
  the server conversation and its running work. Stop turns from Codex itself.

Run retirement from the monitor or another session. Retired members cannot be revived.

## Configuration paths

| Purpose | Default location |
| --- | --- |
| Repository/worktree roots and monitor bell | `~/.motley/config.toml` |
| Global blueprints | `~/.config/motley/blueprints/` |
| Generated tmux configuration | `~/.config/motley/motley.tmux.conf` |
| Member state and event history | `~/.local/state/motley/` |

`XDG_CONFIG_HOME` changes the blueprint and tmux configuration root.
`XDG_STATE_HOME` changes the state root. The main config remains under `~/.motley`;
first launch copies an existing XDG config there once, without overwriting it later.
Run `mtly config` to inspect resolved roots and `mtly init` for tmux setup instructions.

## Update or uninstall

Rerun the installer to build from `main`. For published releases, install `gh` and use:

```sh
mtly update --check
mtly update
```

Reopen Motley after updating. To restart the persistent monitor:

```sh
tmux kill-session -t _motley  # closes only the Motley monitor
mtly monitor
```

To uninstall, with Python 3 installed:

```sh
curl -fsSL https://raw.githubusercontent.com/thomashartm/motley/main/uninstall.sh -o motley-uninstall.sh
bash motley-uninstall.sh
```

From a checkout or release archive, use `bash uninstall.sh`.
Uninstall removes the commands and hook/popup integration. It preserves settings,
worktrees, branches and running sessions. Restart agents and tmux after work finishes.

## Build and install from source

```sh
bash install.sh --local  # build and install this checkout
make build              # build only; run ./bin/mtly
make test               # installer, plugin and Go tests
make check              # tests, vet, lint and four platform builds
make snapshot           # archives in dist/; does not publish
```

Requirements: Go 1.22+, Git, tmux, cp, bash, Python 3 and Node 24.
Full checks need golangci-lint 2.14.0; snapshots need GoReleaser 2.18.2.
Set `GOLANGCI_LINT` or `GORELEASER` to use tools outside PATH.
