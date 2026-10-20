# motley — All your local agent crews in one window

<p align="center">
  <img src="motley-logo.png" alt="Motley logo" width="240">
</p>

**Start, monitor and manage your local coding agents from one place.**
* See what each session is working on
* see what’s running and know where your attention is
needed
* Keep track of countless terminals or terminal tabs easily or just spawn them inside motley

**Works with Claude Code, Codex and OpenCode on your local machine.**

- Start agents in separate Git worktrees and motley manages the worktrees for you.
- Add agents that are already running.
- See which agents are working or need your input.
- Group related agents into crews, link GitHub tickets and keep task context handy.
- Associate agents to github issues.
- Add descriptive infos to each motley session
- spwan new agents with already prepared prompt blueprints

An agent session is a **member**. A **crew** groups members. A **gig** describes the crew's work.
Use `motley` or the shorter `mtly`.

<p align="center">
  <a href="motley-screen.png"><img src="motley-screen.png" alt="Motley showing agents grouped by status and crew" width="900"></a>
</p>

## Install

Supports macOS and Linux, on amd64 and arm64, with bash, zsh or fish.
Install and sign in to your agent CLI first. macOS also needs Homebrew.

```sh
curl -fsSL https://raw.githubusercontent.com/thomashartm/motley/main/install.sh -o motley-install.sh
bash motley-install.sh
```

The installer sets up Motley, missing Git/Go/tmux dependencies, PATH and agent hooks.

1. Open a new terminal and run `mtly`.
2. Restart existing agents to load their hooks.
3. In Codex, use **`/hooks`** to review and trust the Motley hooks.

Rerun the installer to update from `main`, then reopen Motley.
[Release updates and uninstall instructions →](docs/advanced.md#update-or-uninstall)

## Start or add an agent

### Start a new agent

1. Put your repository under `~/projects`, or [configure your repository directories](#configuration).
2. Run `mtly`, press **Home**, and select **Spawn agent (s)** from the main menu.
   The **s** shortcut opens the same flow.
3. Choose **Claude**, **Codex** or **OpenCode**, then **New Ghostty tab** or
   **tmux session**. Ghostty runs the agent directly, without tmux (macOS only).
   For Ghostty, choose an existing window (the current one is selected first),
   or **New window**. Press **p** to bring the highlighted window to the front;
   return to Motley to confirm your choice. **r** refreshes the window list.
4. Choose the repository and source branch, then name the task and its new branch.
   Choose a prompt template and authorization level, and review the preview.
5. Select **Launch**. Both destinations create a **new worktree**. Ghostty opens
   the agent there in a new tab; for tmux, select the member and press **Enter**.

Ghostty tab titles follow the member name, prefixed by its colour marker, such
as **🔵 Payments agent**. Editing the member or its crew updates the title; a
member colour overrides its crew colour. **Open agent** focuses the recorded
tab. Sessions launched before tab tracking was added acquire this support
after they stop and you use **Revive**.

**Launching creates a worktree and pushes a new branch.** The repository needs an
`origin` remote and a local `main` or `master` branch.

### Add a running agent

1. Press **Home**, then **a**.
2. Choose **Claude** or **Codex**.
3. Select the session and press **Enter**.

The existing agent keeps running. Imported Claude stays in its original terminal;
reply there. Imported Codex opens through its existing shared server.
[Import requirements and CLI commands →](docs/advanced.md#existing-sessions)

## Daily use

Select a member to act on it. Press **Home** for main actions.
You can also click members, actions and the footer controls.

| I want to… | Action |
| --- | --- |
| Open the selected agent | **Enter** or **o** |
| Return from an agent | Click **Back to monitor** |
| See available actions | **3** |
| Find a member | **/** |
| Change grouping | **g** |
| Edit a member | **e** |
| Manage crews | **Home → m** |
| Switch the tracked Claude session | **Shift-S** |
| Open browser links | **b** |
| Create, open or mark a PR ready | **P** |
| Refresh GitHub status | **u** for one member; **U** for all |
| Copy a status/error message | **c** |

**Needs you** includes questions, permission requests, completed replies and idle
agents. **Working** includes agents with active subagents. **Ended / dead** means
the agent ended or its session is no longer running. Open the agent to approve permissions.

Keep an overview in another terminal with `mtly monitor`.
[All shortcuts and terminal setup →](docs/advanced.md#navigation-and-terminal-tabs)

## Stop, resume or remove

Select a member. Press **d** to terminate, or **3** to open Actions.
You can also click **d Terminate** in the bottom bar.

| Action | Result |
| --- | --- |
| **Terminate (d)** | Stop the agent and archive its entry. Keep files, branches and history. Also removes stopped entries. |
| **Revive (r)** | Restart a stopped member in its existing worktree. |
| **Retire (x)** | Archive the member. Remove its managed worktree and usually its local branch. |

### Before removing a member

- Retirement refuses uncommitted or unpushed work. **Force discards that work.**
- Remote branches remain.
- Only stopped members still in the active list can be revived.
- Terminated and retired members are archived and cannot be revived.

### Imported sessions

Retirement always keeps an imported checkout.

| Agent | Terminate | Retire |
| --- | --- | --- |
| Claude | Stop the conversation and archive the entry. | Stop the conversation and archive the entry. |
| Codex | Unavailable. | Close Motley's client and archive the entry. The agent keeps running. |

[Lifecycle details →](docs/advanced.md#stop-resume-and-retire)

## Manage Claude conversations

### Refresh or reimport a session

Select an imported Claude member. Choose **Actions (3) → Reimport session (S)**,
or press **Shift+S**, to refresh its workspace or choose another conversation.
The picker also works when the workspace is unchanged.

Other entry points:

- **Add existing agent → Claude → Reimport:**
- `mtly import <session-id>`

Reimport keeps the member's name, crew and history. The conversation keeps
running. Terminate and Retire then use the updated workspace.

### Track multiple conversations

Select a member and choose **Track another session (A)**.
Or run `mtly import <session-id> --with <member-id>`; `--list` shows candidates.

| Action or status | Behaviour |
| --- | --- |
| **Open** | Open the primary conversation. |
| **Switch tracked session (S)** | Change the primary conversation. Keep monitoring the others. |
| **Working** | At least one tracked conversation or its subagents is working. |
| **Terminate / Retire** | Stop all explicitly tracked Claude conversations. |

## Read the status columns

| Column | Badge | Meaning |
| --- | --- | --- |
| **ACC** — access | **GHO** | Direct Ghostty |
| | **TMX** | Motley-managed tmux |
| | **EXT** | External session |
| | **MIX** | Managed and external access |
| **MOD** — mode | **FG** | Foreground |
| | **BG** | Background |
| | **F+B** | Foreground and background |
| **AG** — agent | Agent name | Grey for every agent |

Access and mode are independent. An external session may run in another tmux
pane. **Details** shows each conversation's status.

## Select and copy text

Drag over text in **Actions**, or across list rows to select whole rows.
Selections are highlighted, with **CPY / CLR** hints below the panels.

| Input | Result |
| --- | --- |
| **Ctrl+C** with a selection | Copy it. Copied rows include full member titles. |
| **Esc** | Clear the selection. |
| **Ctrl+C** without a selection | Close the TUI. |
| Click and release | Activate an action or link. |

## Configuration

Inside tmux, **Shift+Enter** inserts a line break in agents that support it;
**Enter** submits. If both keys submit, see the
[extended-key configuration](docs/troubleshooting.md#shiftenter-submits-instead-of-inserting-a-line-break).

Motley creates `~/.motley/config.toml` on first launch. Edit the roots to suit your setup:

```toml
schema = 1
repos_roots = ["~/projects", "~/projects/aderis"]
worktrees_root = "~/worktrees"
monitor_bell = false
```

Each entrypoint supplies the repositories directly inside it; scanning is not
recursive. Press **s** to choose a repository, grouped by entrypoint with its full
path shown in parentheses.
The legacy `repos_root` setting still works; `repos_roots` takes precedence.

Run `mtly config` to check the repository and worktree paths.
[Blueprints, permission modes and configuration paths →](docs/advanced.md)

## Development

Install Go 1.22+, Git, tmux, cp, bash, Python 3 and Node 24.
For `make check`, also install golangci-lint 2.14.0.

```sh
make build       # build bin/motley and bin/mtly
./bin/mtly       # run the local build
make test        # Go, plugin and installer tests
make check       # tests, vet, lint and macOS/Linux builds
```

Tests use temporary repositories, fake agents and isolated tmux servers.
[Local installation and release builds →](docs/advanced.md#build-and-install-from-source)

## Contributing

- **Report a bug:** [open an issue](https://github.com/thomashartm/motley/issues) with your OS, terminal, agent, `mtly version`, reproduction steps and expected result. Remove secrets from logs or screenshots.
- **Suggest a feature:** describe the task you want to accomplish and the current limitation. Discuss larger changes in an issue first.
- **Send a fix:** fork the repository, create a branch and keep the change focused. Follow [AGENTS.md](https://github.com/thomashartm/motley/blob/main/AGENTS.md).
- **Before opening a PR:** add tests for changed behavior, update relevant instructions and run `make check`. In the PR, explain the change and list the checks you ran.

## More help

- [Advanced usage](docs/advanced.md): CLI examples, crews, blueprints, GitHub and terminal setup.
- [Troubleshooting](docs/troubleshooting.md): status sync, missing sessions, hooks and clipboard.
- [Delivery notes](https://github.com/thomashartm/motley/blob/main/DELIVERY.md)
- [MIT license](LICENSE)
