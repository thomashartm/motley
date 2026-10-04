# motley

<p align="center">
  <img src="motley-logo.png" alt="Motley logo" width="240">
</p>

**Motley brings your all your local coding agents into a single view. 
* See what each session is working on
* see what’s running and know where your attention is
needed
* Keep track of countless terminals or terminal tabs easily or just spawn them inside motley **

Works with Claude Code, Codex and OpenCode on your local machine.

- Start agents in separate Git worktrees.
- Add agents that are already running.
- See which agents are working or need your input.
- Group related agents into crews, link GitHub tickets and keep task context handy.

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
2. Run `mtly` and press **s**.
3. Choose a repository, task and agent. Select **feature** or **fix** for the branch
   prefix; for example, `feature/backend-2991-allow-exempt-tax-code`.
4. Review and launch.
5. Select the member and press **Enter** to open it.

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

Select a member, then press **3** to open its actions.

| Action | Result |
| --- | --- |
| **Terminate (d)** | Stop the agent; keep its worktree, branch and history. |
| **Revive (r)** | Restart a stopped member in its existing worktree. |
| **Retire (x)** | Archive the member; remove its managed worktree and usually its local branch. |

Retirement refuses uncommitted or unpushed work. **Force discards that work.**
Remote branches remain. Retired members cannot be revived.

**Imported sessions:** retirement keeps the checkout. It stops imported Claude;
for imported Codex, it only closes Motley's client and removes the entry.
Imported Codex has no Terminate action. [Lifecycle details →](docs/advanced.md#stop-resume-and-retire)

## Configuration

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
