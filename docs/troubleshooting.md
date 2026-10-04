# Troubleshooting

[Back to the README](../README.md)

## Motley follows the wrong Claude terminal

1. Select the imported member.
2. Press **Shift-S** or choose **Actions → Switch tracked session**.
3. Compare session IDs and statuses, then select the intended session.

Only live, unregistered Claude sessions in the same checkout are offered.
Switching keeps the member's name and crew and leaves both agents running.

To list candidates from the CLI:

```sh
mtly import --replace <member-id> --list
```

## Status looks stale

1. Run `mtly ls` and compare the tracked session with the agent's own terminal.
2. For imported Claude, run `claude agents --json` and compare its session ID.
3. If the wrong conversation is tracked, switch it as described above.
4. Install the current hooks for the affected provider:

```sh
mtly hooks install claude
mtly hooks install codex
mtly hooks install opencode
```

Run only the command for the provider you use. Restart the agent when convenient
to load added hooks. In Codex, also use **`/hooks`** to review and trust them.

Imported Claude hooks report from original terminals as well as Motley terminals.
Subagent activity keeps the member working while the parent waits. Unregistered
external Claude sessions are ignored. Codex and OpenCode reporting requires a
Motley-managed session; a newly opened agent may show `starting` until its first prompt.

Hook diagnostics are in `~/.local/state/motley/report.log`, or beneath your
`XDG_STATE_HOME`. Reopen Motley after updating its binary.

## An existing session is missing from the import picker

| Check | What to do |
| --- | --- |
| Already in Motley? | Find the member in the overview; registered sessions are excluded. |
| Claude discovery fails? | Run `claude agents --json`; update Claude if that command is unavailable. |
| Codex has no shared server? | Use a session on the local shared app-server. Standalone `--no-daemon`, remote, ephemeral and subagent sessions cannot be imported. |
| Session ended? | Import requires a live session. |

## Open agent does not open my imported Claude terminal

Imported Claude stays in its original terminal. Motley shows where it runs;
switch to that terminal and reply there. To move it into Motley, use
**Terminate**, then **Revive** when it is safe to stop the current work.

## Revive says the session already exists

The agent may have exited while its tmux shell remained. Open that shell and
restart the agent there. Revive recreates a missing session; it cannot restore
retired members or deleted worktrees.

## GitHub actions fail or show old results

- Run `gh auth status` and sign in if needed.
- Press **u** or **U** to refresh; GitHub data is not polled automatically.
- If you changed `XDG_CONFIG_HOME`, set `GH_CONFIG_DIR=~/.config/gh` when that is
  where your existing gh login is stored.
- Use `--no-gh` when spawning to skip issue lookup.

## Copying or mouse controls do not work

1. Reattach the member or rerun `mtly monitor` after upgrading.
2. In the overview or monitor, drag across list rows or Actions text, then press
   **Ctrl+C** to copy the highlighted selection. **Esc** clears it.
3. In agent panes on local macOS, drag text and paste with **Cmd-V**.
4. For status/error messages, press **c** or click the message above the footer.

Over SSH, macOS agent-pane drag copying writes to the remote Mac's clipboard.
Message and overview-selection copying use the terminal/tmux clipboard
integration; with tmux `set-clipboard off`, copying may only reach the tmux paste
buffer. Motley reports this instead of claiming a successful clipboard copy.

For an older installation, add the following to
`~/.config/motley/motley.tmux.conf`, then reload it. Use your XDG path if configured.
This sets Motley's default **Ctrl-a** prefix:

```tmux
set -g mouse on
set -g prefix C-a
unbind C-b
bind C-a send-prefix
if -F '#{>=:#{version},3.4}' 'set -as terminal-features ",xterm*:hyperlinks"'
```

```sh
tmux source-file ~/.config/motley/motley.tmux.conf
```

## Installation or update failed

| Operation | Log directory |
| --- | --- |
| Install | `~/.motley/install-history/` |
| Release update | `~/.motley/update-history/` |
| Uninstall | `~/.motley/uninstall-history/` |

For a bug report, include `mtly version`, OS, terminal, agent, reproduction steps
and the relevant error. Remove secrets before sharing logs or screenshots.
