# Delivery checkpoints

## Keep empty Needs You visible — 2026-10-05

Repository rules checked: local Git/gh CLI, feature branch retained in this
linked worktree, and delivery notes recorded here.

Publication requested on `fix/needs-you-empty-section`, based on `origin/main`
at `e25ae93` after PR #67 merged. Hosted CI is pending publication.

The attention list keeps the `NEEDS YOU` heading and shows a muted `None` row
when no visible members need attention, including an empty or filtered list.
`WORKING` stays below the empty section. The placeholder cannot select a member.

Validation: `go test ./internal/tui`, `go vet ./internal/tui` and
`git diff --check` passed. Regression coverage checks attention transitions,
empty and filtered lists, and mouse targets for the placeholder and member row.

Document schema: `1`

motley is built one usable slice at a time. Scope and acceptance criteria live in
[REQUIREMENTS.md](REQUIREMENTS.md#12-delivery-plan-thin-vertical-slices).

After each item, stop, summarize what was built and left out, report validation
and open questions, and wait for user feedback before starting the next item.
Feedback may change the scope or order of later items. Keep milestone status and
implementation-process notes here; the README describes how to use the tool.

## W0 — Shell: complete

Delivered the Go CLI shell, configuration loading for repository and worktree
roots, `motley version`, help, executable smoke tests, CI and GoReleaser snapshots.

- Commit: `e4492c8`; release tag: `v0.0.0`.
- All nine [release-tag CI jobs passed](https://github.com/thomashartm/motley/actions/runs/36705427765):
  tests and vet on macOS/Linux, the Go 1.22 minimum on Linux, lint, four platform
  builds and snapshot packaging.
- Local validation passed with Go 1.25.5. Go 1.22 test execution on the local
  macOS failed with `missing LC_UUID load command`; its minimum-version check
  passed on Linux in CI.
- Session management, tmux integration, hooks and the TUI were deliberately
  left out of this slice.

Local checks, hosted CI and tags are reported separately. A local pass does not
establish a hosted CI result. Release numbering follows work-item numbering:
W0 is `v0.0.0`, W1 is `v0.1.0`, and so on.

## CI keyboard compatibility and retirement layout — implemented

- Rule check: Git/GitHub operations use local `git`/`gh`; delivery evidence stays
  here. Changes cover the reported CI failure and retirement confirmation only.
- The Shift+Enter test assumed tmux 3.5+ fallback behavior. Ubuntu CI uses 3.4,
  which drops the unbound key. Version-aware baseline checks now use trailing
  markers to verify processing even when the key produces no output.
- A real tmux 3.4 run exposed a second issue: `send-keys S-Enter` could insert
  literal text in a shell. The fallback now forwards the original bound key with
  argument-free `send-keys`. Agent assertions still require the exact CSI-u bytes.
- CI keeps all test matrix jobs running after a failure and prints the tmux
  version. Push and pull-request triggers retain their existing coverage.
- Retirement confirmations gain inset, word-wrapped paragraphs, a separate
  muted checkout path under `Dir`, and a divider above highlighted controls.
  Small terminals retain the controls and mark truncated explanations.
- Validation: the real tmux 3.4 keyboard regression passed; `make test` passed
  with local tmux 3.6a, along with `go vet ./...`, CI-pinned golangci-lint (zero
  issues), and all four cross-builds. Layout checks cover 60×10, 100×25 and
  200×35 terminals, long paths, visible controls and mouse hit targets.
  Hosted CI results are tracked separately on the pull request.

## W1 — First member: complete

Delivered new-branch worktree creation from main/master, immediate upstream push,
artifact copying, schema-1 manifests and interactive agent startup in tmux.
Commands: spawn, ls, attach and switch. Agent exit leaves a login shell.

- Fixed defaults; exact repository directory names. No additional config settings.
- Spawn attaches outside tmux and switches inside; `--detach` supports background
  creation and integration tests.
- Worktree paths follow the configured-root plus repository plus branch-slug rule.
  Ticket-based ids do not change the directory naming rule.
- Id allocation accounts for normalized tmux names. Existing identities get a
  repository prefix; a second collision is refused. Simultaneous spawn commands
  are serialized with an OS lock; a competing invocation asks the user to retry.
- Tests exercise real git and isolated tmux servers with fake Claude, Codex and
  OpenCode executables: startup, environment, no prompt arguments, upstream push,
  main/master bases, shell fallback, switching, alive/dead state and preflight
  refusals. Attach and outside-tmux switch argv are tested with a fake tmux.
- Artifact-copy tests compare an explicit golden inventory and native attributes
  against the unchanged wt function. Nested targets are excluded, and destination
  symlink parents are refused to prevent copying outside the worktree.
- Implementation commit: `46b45a6`; release tag: `v0.1.0`.
- Local tests, vet, lint, all four builds and snapshot packaging passed.
  All nine [implementation CI jobs passed](https://github.com/thomashartm/motley/actions/runs/36716731133),
  including the lifecycle tests on macOS/Linux and the Go 1.22 check on Linux.

Deliberately left out: fuzzy repo selection, existing-branch creation, prompt
passing, agent hooks/status, retire/revive, TUI, GitHub lookups and further config.
W1 supports branch slugs containing ASCII letters/digits, dots, underscores and
hyphens; broader name handling can follow actual usage.

## W2 — Overview TUI: complete

Delivered the Bubble Tea overview with alive/dead sections, a scrollable manifest
detail pane, stable selection across refreshes, and terminal-aware jumping.

- Polls sessions and clients once each per second. Manifest parsing is cached
  by modification time and size. Client activity supports automatic work-tab
  selection; action-time checks catch disconnected or repurposed clients.
- `motley init` scaffolds the current three-key config and popup binding, without
  overwriting either file. Generated tmux config carries a schema comment.
- `motley monitor` creates or reconnects to `_motley`. Enter targets another
  client; T pins a work tab or restores automatic selection. q detaches the
  monitor client, keeping the overview alive. A missing work tab gets a hint.
- `motley config` retains the former no-argument config display now that the root
  command opens the TUI. The overview needs at least a 60 × 10 terminal.
- Model tests cover selection, ordering, removals, dead-session refusal, monitor
  targeting and pinning, and narrow layouts. Real pseudoterminal tests exercise
  normal and popup jumps, outside-tmux attach, monitor reuse, pinning, detach,
  and alive-to-dead refresh on an isolated tmux server.
- Implementation: `9f5d5a6`, compatibility fix: `ec61c54`, final test adjustment:
  `ba91a73`; release tag: `v0.2.0`.
- Local tests, vet, lint, darwin/linux × amd64/arm64 builds, and GoReleaser
  snapshot packaging passed. All nine
  [implementation CI jobs passed](https://github.com/thomashartm/motley/actions/runs/36733809284),
  including macOS, Linux and Go 1.22. The terminal integration also passed locally
  on Linux arm64 with Go 1.22 and tmux 3.3a in a disposable container.
- Linux compatibility testing found that older tmux versions sanitize tab
  separators without UTF-8 mode. Session/client reads now explicitly use `-u`;
  the standalone terminal test also runs with the C locale. Popup tests wait
  for tmux to acknowledge the prefix key before sending h.

Deliberately left out: attention states/hooks, spawn forms, crews, permission-mode
selection and other later-item features. Native Claude and Codex permission-mode
requests remain tracked in [#1](https://github.com/thomashartm/motley/issues/1) and
[#2](https://github.com/thomashartm/motley/issues/2).

## W3 — Claude attention states: complete

Delivered Claude hook reporting, attention sections and details, monitor counts,
new-attention markers and optional terminal bell (`monitor_bell = true`).

- `motley hooks install claude` merges seven hooks idempotently and backs up an
  existing settings file. Global user hooks are installed only on explicit use
  of that command; implementation testing used temporary settings.
- Recorded real Claude Code 2.1.285 lifecycle, Read, Bash, question, permission
  and idle payloads. Fixtures include version and reproduction notes. Stop's
  direct assistant message avoids a transcript flush delay observed in testing;
  older payloads use a bounded transcript-tail fallback.
- `report` always exits 0 and emits no stdout/stderr. It ignores sessions without
  MOTLEY_MEMBER, caps payloads and event lines, bounds stdin/tmux waits at 250 ms,
  and uses two tmux invocations. Error logging gets at most another 10 ms and
  appends schema-1 report.log records when the state filesystem is available.
- tmux holds status, transition time, last-seen time and bounded latest event
  context. Generic permission notifications preserve a pending question, or
  retain the preceding tool/command. Hooks never write manifests. Events append
  only for transitions and the specified lifecycle/notification events.
- NEEDS YOU sorts oldest first; WORKING and ENDED / DEAD follow. Selected details
  tail-refresh on event mtime/size changes; ready detail includes commit and diff.
  `ls` includes STATUS alongside the existing alive/dead STATE column. Existing
  unreported sessions remain usable. New sessions get a tmux status display;
  init preserves existing config files, with manual upgrade steps in the README.
- Owned event records have schema 1. Claude settings preserve their native format
  and backups preserve exact bytes (versioned filenames); no unrelated schema
  key is inserted into third-party configuration.
- Tests cover recorded mappings, backup/idempotency, bounded records, timeout and
  error behavior, ordering/alerts, and real hook-to-terminal updates with an
  isolated tmux server and fixture Git repository. The terminal test covers the
  optional bell, fresh selected details and a ready commit/diff. It also verifies
  exactly two tmux calls and unchanged manifest bytes/mtime.
- Local full-process benchmark: **14.39 ms p95** across 300 invocations on macOS
  arm64 (Apple M4 Pro, Go 1.25.5, tmux 3.6a), using the shipped CGO_ENABLED=0 /
  trimpath build and real tmux. This is a local measurement, not a guarantee for
  every host. Reproduce with `go test ./cmd/motley -run '^$' -bench
  '^BenchmarkReportCLI$' -benchtime=300x`.

Local tests, vet, lint, all four static builds and snapshot packaging passed.
Implementation commit: `a6dee0a`; release tag: `v0.3.0`. All nine
[implementation CI jobs passed](https://github.com/thomashartm/motley/actions/runs/36742146219),
including real terminal/report tests on macOS/Linux and Go 1.22 on Linux.

Deliberately left out: event rotation and stale detection (W11), Codex/OpenCode
reporting (W8), retire/revive (W4), crews, spawn forms and native permission modes
([#1](https://github.com/thomashartm/motley/issues/1),
[#2](https://github.com/thomashartm/motley/issues/2)).

## W4 — Finish and resume members: complete

Delivered `retire`, `adopt`, `revive`, and the overview's x/r actions.

- Retirement refuses dirty/untracked or unpushed work, using the base branch
  when no upstream exists. Force explicitly skips these checks; ownership and
  main-worktree checks always apply. Normal retirement checks again after
  stopping the agent in case work changed during shutdown.
- Ports wt-clean's double-force removal, directory fallback, unlock/prune,
  registration verification and local branch deletion. Main/master/develop and
  all remote branches are retained; CLI/TUI can retain other local branches.
  Cleanup failures retain the active manifest and allow retry after a worktree
  has already been removed.
- Retirement archives all existing state files with retired_at. Archive copies
  are written before active files move; earlier retirements of a reused id get
  preserved through a timestamp suffix. Lifecycle changes share the existing
  OS lock with spawn.
- Adopt derives state from a linked worktree, including detached HEAD, and
  renames/registers its current session. It never adopts a main checkout or a
  worktree already managed by another member. Session environment changes apply
  to new processes; the CLI explains how to export the id and restart an existing
  agent for reporting.
- Revive recreates only a missing session in an existing worktree and leaves it
  detached. Claude uses its latest recorded session id; empty history and the
  other agents start fresh, without a prompt. The installed Claude Code 2.1.286
  help confirms --resume accepts a session id. The integration test verifies the
  argument through a fake agent, including history beyond a single tail window.
- x opens a retirement dialog with dirty/ahead counts, explicit force and
  keep-branch toggles, confirm and cancel. r revives dead members. Both operations
  run outside the UI update loop and report errors in the overview.
- Current-session retirement is refused because killing the caller's pane could
  interrupt cleanup. The monitor, another session and outside terminals support
  retirement. This is an explicit MVP boundary, documented in the README.
- Tests cover refusal and force, archive content/timestamps/collisions, local and
  remote branch retention, locked worktrees, fallback cleanup and retry, changed
  worktree ownership, main checkout protection, adoption, resume/fresh starts,
  and actual terminal confirmation/revive actions on isolated tmux servers.

Local tests, vet, lint, all four static builds and snapshot packaging passed.
Implementation commit: `ac568bc`; release tag: `v0.4.0`. All nine
[implementation CI jobs passed](https://github.com/thomashartm/motley/actions/runs/36774661364),
including lifecycle and terminal tests on macOS/Linux and Go 1.22 on Linux.

Deliberately left out: GitHub open-PR warnings (W9), configurable protected
branches and batch cleanup (W10), restore-from-archive, automatic environment
injection into running agents, native Codex/OpenCode resume (W8), crews and
permission-mode selection.

## W5 — Crews and colours: complete

Delivered crew add/list/edit/remove/assign commands, spawn/adopt crew and colour
flags, and the overview's crew browser and editor.

- Crews live in schema-1 TOML, written atomically under the shared lifecycle
  lock. IDs use title slugs with collision suffixes; new crews take the next
  unused palette colour. URL shape determines text/issue/project/link kind.
- Member colour resolves from its override, then crew, then a stable id hash.
  Live sessions receive crew title, colour and emoji options, contrasting tmux
  status bars and emoji titles. Crew edits update all live members immediately;
  revive restores the same styling. User titles are escaped as literal tmux text.
- Agent badges and the eight-colour palette are fixed. Status icon colours stay
  independent of identity bars. No configuration settings were added: tmux
  colour/title switches and badge customization remain deferred.
- g cycles attention/crew/repo grouping. Crew rows are collapsed, with status
  counts and clickable titles; Tab focuses the member table. Selection survives
  status changes and polling. Members support the existing jump/retire/revive
  actions and e editing. Right/Space expands, Left collapses, H shows inactive
  crews. Narrow tables drop SINCE and then BRANCH; the PR column awaits W9.
- G manages crews: add/edit, cycle colour, delete with explicit force-unassign.
  e edits member name/ticket/crew/override. Forms use Tab, Ctrl-s save and Esc.
  Removal counts live and dead references; force retains individual overrides.
  Archived manifests keep their historical crew ids.
- Tests cover URL kinds, palette resolution, schema/defaults, id collisions,
  real tmux recolouring, override retention, force-unassign, adopt/revive styling,
  grouped selection, responsive layouts, terminal editing and monitor table jumps.

Local tests, vet, lint, all four static builds and GoReleaser snapshot packaging
passed. Implementation commit: `18a2faf`; release tag: `v0.5.0`. All nine
[implementation CI jobs passed](https://github.com/thomashartm/motley/actions/runs/36778336786),
including crew and terminal tests on macOS/Linux and Go 1.22 on Linux.

Deliberately left out: GitHub title fetching and crew suggestions (W9), PR table
columns (W9), configurable styling, blueprints (W6) and native permission modes.

## W6 — Blueprints: complete

Delivered global/repository blueprint discovery, TOML front matter, Go template
rendering, spawn blueprint/variable flags, and list/show/validate commands.

- Repository definitions override global names, then repo restrictions filter
  the effective set. Duplicate names within one directory and malformed files
  are errors. Schema defaults to 1; unknown front-matter fields are ignored.
- Available data is Repo/Branch/Base/Ticket/Name/Worktree, Crew fields and Vars.
  Missing variable keys render empty; variables are optional and repeated CLI
  assignments use the last value. Template errors are caught before worktree
  creation. The authoring validator renders with empty data.
- CLI agent selection overrides the blueprint; otherwise use the blueprint's
  agent, defaulting to Claude. Arguments belonging to another blueprint agent
  cause a clear refusal. No permission-mode picker was added.
- Prompt files carry a schema-1 Markdown comment and mode 0600. The comment is
  stripped before delivery; the rendered body is passed byte-for-byte. A 64 KiB
  limit keeps the single argument below platform limits. Manifests record the
  blueprint and arguments; revive keeps arguments without replaying the prompt.
- Verified installed CLI help: Claude 2.1.286 and Codex 0.159.2 accept positional
  prompts, OpenCode 1.18.21 accepts --prompt. W6 brings only native initial prompt
  delivery forward from W8, avoiding a fallback dependent on future idle hooks.
  Version/contract evidence is in internal/agents/testdata/prompt-contracts.md.
- Added a ready-to-copy plan-first example and blueprint name in TUI details.
  Tests cover rendering goldens, missing variables, precedence/filtering, schema
  validation, invalid-input refusal before worktree creation, and real tmux with
  fake agents recording exact argv for all three agents. Multiline prompts,
  leading hyphens and shell metacharacters remain literal; revive/archive tests
  verify no replay and preserved prompt bytes. No live model calls were made.

Local tests, vet, lint, all four static builds and GoReleaser snapshot packaging
passed. The packaged example also passed CLI validation. Implementation commit:
`815e318`; release tag: `v0.6.0`. All nine
[implementation CI jobs passed](https://github.com/thomashartm/motley/actions/runs/36781525991),
including blueprint handoff and lifecycle tests on macOS/Linux and Go 1.22.

Deliberately left out: TUI spawn form and prompt editor (W7), Issue template data
and issue fetching (W9), agent status/resume integration for Codex/OpenCode (W8),
and native permission-mode selection ([#1](https://github.com/thomashartm/motley/issues/1),
[#2](https://github.com/thomashartm/motley/issues/2)).

## W7 — Spawn and steer: implemented (Done)

Current status, 2026-10-01: implemented in `513be9a` and pushed to main.
The full local gate and [hosted CI](https://github.com/thomashartm/motley/actions/runs/36834127856)
passed. The roadmap ticket is Done based on implementation, as requested;
the `v0.7.0` release tag remains a separate delivery step. Earlier implementation
and validation notes below record the original checkpoint history.

Implemented the TUI spawn form, reply input, work-tab picker, filtering, and
`motley tabs` / `motley send`.

- Spawn steps: repo filter → ticket/name/editable branch → agent → compatible
  blueprint or none → variables → prompt preview/editor → launch with progress.
  Preview preparation is read-only; launch uses its exact reviewed prompt and
  arguments, rechecking worktree availability and member identity. Success
  focuses the new member. Failed creation retains the existing cleanup behavior.
- `$EDITOR` opens a private schema-1 temporary prompt file; it is removed after
  the editor returns. Edited text also works without a blueprint. A new optional
  manifest `prompt` boolean records this case; older blueprint manifests still
  load their prompt. Revive does not replay either kind of initial prompt.
- `/` performs case-insensitive subsequence matching over id/name/ticket/repo/
  branch and preserves attention order. The matcher is local: fetching the
  planned sahilm/fuzzy dependency was blocked by this session's network policy.
- `i` sends literal text and Enter to the active pane, rejecting permission,
  ended/dead states and control characters. `t` chooses another work client;
  send rechecks both endpoints and protects the monitor client.
- New coverage includes form validation/navigation, preview editing and focus,
  narrow layouts, filter/refresh behavior, read-only preparation, and reply/send
  checks with a fake tmux. Real tmux integration tests were added for reply/send
  and the full terminal spawn/editor flow.

Local verification passed: all internal package tests, CLI smoke/non-terminal
checks, form/filter/reply/send checks, go vet, golangci-lint, four static builds,
and GoReleaser snapshot packaging. Prepared-launch tests use real local Git with
a fake tmux to verify the reviewed prompt snapshot and manual-prompt metadata.
The local binary is available as `./bin/motley` (`0.7.0-dev`).

Full integration and publication are pending. This session blocks tmux socket creation
(`Operation not permitted`), network access, writes to Git metadata, and writes
outside the workspace/temp roots. W7 must not be marked complete or released
until the real tmux tests and hosted CI pass.

Continuation check: `go test ./...` was rerun on October 1, 2026. All internal
packages passed; CLI integration tests, including the new reply/send and terminal
spawn tests, failed during isolated tmux fixture startup before exercising their
assertions. They remain unverified, not skipped or accepted as passing.

To finish in a session that permits tmux sockets, Git metadata writes and network
access, run the full gate from the repository root:

```sh
make check GOLANGCI_LINT=.tools/golangci-lint-2.14.0-darwin-arm64/golangci-lint
make snapshot GORELEASER=.tools/goreleaser/goreleaser
```

Then finish the W7 commit, hosted CI, release tag and local installation. Do not
start W8 until this checkpoint is complete and the user has given feedback.

Deliberately left out: configurable branch templates, ranked fuzzy matching,
setup commands (W10), GitHub lookup (W9) and native permission-mode selectors.

## Next checkpoint

### Installer and test walkthrough

Added `install.sh`: downloads `main` with Git (or builds the current checkout with
`--local`), builds using Go 1.25.5, atomically installs to `~/.local/bin`, and
configures bash/zsh/fish PATH, the tmux popup and installed Claude's hooks.
Missing prerequisites use an existing supported package manager. Existing
configuration is preserved; shell/tmux edits are backed up and idempotent.
The README now includes download commands and a first-member test walkthrough.
`AGENTS.md` records the user's instruction to use Git/gh, never the GitHub connector.

Installer checks passed with disposable bash/zsh/fish homes, repeat installs,
failed builds and old tmux refusal. ShellCheck passed. A real source build/install
in a temporary home verified config, `ls`, Claude hook installation and binary
discovery by a fresh zsh. Package-manager execution and network downloading were
mocked; live tmux reload remains subject to the W7 integration gate above.
Installer tests run via `make test` and CI. These changes remain local; the README
download URLs become available after publication. Use `bash install.sh --local`
from this checkout in the meantime.

Finish W7 validation and release before starting W8. Feedback focus: spawn-form
flow, prompt preview/editor handoff, filtering and steering across work tabs.

Continue using the original [wt 1.2.0](reference/wt) and
[wt-clean 1.0.0](reference/wt-clean) alongside motley until W10 delivers the full
worktree tooling.

## Motley naming checkpoint — 2026-10-01

Renamed the product and Go module to `motley`; origin is
`https://github.com/thomashartm/motley`, verified with Git and `gh`.
The full command is `motley`, with `mtly` available from local builds, the
installer, and all four release archives.

- A running agent session is a member; members form a crew. A gig describes the
  crew's package of work. Crews have an optional `gig` field, editable through
  `crew add/edit --gig` and the TUI crew editor, shown in text/JSON crew lists,
  crew details and member details. Empty input clears a gig.
- Renamed command/package paths, manifests, environment variables, tmux options,
  hooks, configuration, blueprints, test fixtures, docs and release artifacts.
  Removed the obsolete branded logo and replaced generated local artifacts.
- This is a pre-v1 naming reset: configuration lives under `motley`, member
  manifests under `motley/members`, hooks use `MOTLEY_MEMBER`, and tmux uses
  `@motley_*`. There are no compatibility aliases for previous naming. Existing
  user configuration and running sessions were not modified; the README
  documents setup and adoption into the new namespace.
- Preserved the existing W7 and installer work while applying the rename;
  these changes are included together in the user-requested local commit.
  Push and release publication remain pending.

Validation passed: `make check` (installer tests, all Go tests including real
tmux/terminal integration, vet, lint and four cross-builds), ShellCheck,
`make build`, and the GoReleaser snapshot. Both command names report the same
version; the short command uses the same configuration. Installer tests cover
the short command across bash/zsh/fish and repeat/failed installs. Gig coverage
includes CLI add/edit/clear/invalid input, terminal editing and TUI display.
All four archives contain both commands. Source, filenames, binary contents and
archive contents were checked for stale product and member terminology.

This successful local full gate supersedes the earlier local tmux blocker.
Hosted CI, W7 publication and the next release remain pending.

### README logo — 2026-10-01

Added the supplied Motley logo to the README and release archives, preserving
the original image. Checked that the repository asset matches the supplied file
and that all four snapshot archives contain the exact logo and README reference.
GoReleaser snapshot packaging passed. The logo and naming commits are ready
for the user-requested push to `origin/main`; hosted CI and release publication
remain separate checks.

### README cleanup — 2026-10-01

Shortened the README to installation, everyday commands, essential keyboard
controls, configuration and development checks. Removed repeated walkthroughs
and implementation detail while retaining launch/retirement side effects,
agent reporting limits, adoption and pre-v1 setup instructions. Documentation
only; checked local links, command examples and diff whitespace.

### #15 — Arrow-key navigation — 2026-10-01

- Added a visible List → Details → Actions focus path. Actions exposes the
  existing member editor, crew manager, spawn, reply, tab and lifecycle actions.
  Left/Esc returns focus; existing letter shortcuts remain available.
- Crew Right expands first, then enters the member table; another Right opens
  Actions. Crew management has an arrow-accessible action menu. Up/Down moves
  through editor fields and Save/Cancel; Left/Right remains caret movement.
- Retirement and crew deletion expose selectable confirmation, force and cancel
  controls. Spawn fields support Up/Down; prompt preview has selectable launch,
  edit and cancel actions. Polling preserves focus and member identity.
- Updated the README controls. Added model coverage at 60×10 and a real terminal
  edit/save/cancel test; overview integration exercises arrows in standalone,
  popup and monitor modes, including monitor work-client routing.
- Kept this ticket on `feat/15-arrow-navigation`, separate from the pending W8
  changes. No mouse support, configurable keymaps or new configuration subsystem.

Validation: `make check` passed (installer tests, all Go tests including real
tmux/terminal integration, vet, lint and all four cross-builds). Focused model
tests also passed after the final crew focus/footer adjustment.
[Hosted CI](https://github.com/thomashartm/motley/actions/runs/36838983647) passed
for code commit `d3fe63f`, including Linux/macOS tests and release snapshots.
[PR #17](https://github.com/thomashartm/motley/pull/17) is ready for review.
Stop after this ticket for navigation feedback.

## Roadmap tickets — 2026-10-01

Created and read back one GitHub issue for every W0–W11 item. Done means
implemented: W0–W7 are closed with the completed reason; W8–W11 remain open.
W7 is implemented and CI-green; its release tag is tracked separately.

| Item | Ticket | Status |
| --- | --- | --- |
| W0 — Shell | [#3](https://github.com/thomashartm/motley/issues/3) | Done |
| W1 — First member, end to end | [#4](https://github.com/thomashartm/motley/issues/4) | Done |
| W2 — Overview TUI v0 | [#5](https://github.com/thomashartm/motley/issues/5) | Done |
| W3 — Attention states for Claude Code | [#6](https://github.com/thomashartm/motley/issues/6) | Done |
| W4 — Finish and resume members | [#7](https://github.com/thomashartm/motley/issues/7) | Done |
| W5 — Crews and colors | [#8](https://github.com/thomashartm/motley/issues/8) | Done |
| W6 — Blueprints | [#9](https://github.com/thomashartm/motley/issues/9) | Done |
| W7 — Spawn and steer from the TUI | [#10](https://github.com/thomashartm/motley/issues/10) | Done |
| W8 — Codex and OpenCode | [#11](https://github.com/thomashartm/motley/issues/11) | Done |
| W9 — GitHub on demand and links | [#12](https://github.com/thomashartm/motley/issues/12) | Done |
| W10 — Full worktree tooling; retire wt and wt-clean | [#13](https://github.com/thomashartm/motley/issues/13) | Open |
| W11 — Pop out and hardening | [#14](https://github.com/thomashartm/motley/issues/14) | Open |

Existing permission-mode tickets #1 and #2 remain open and separate.
Each roadmap ticket includes scope, acceptance criteria, and either completion
evidence or the remaining implementation work.

## W8 — Codex and OpenCode: implementation in progress

Implemented locally for roadmap ticket #11:

- Codex native hook parsing and idempotent `hooks install codex`, with backups
  and preservation of existing hooks/config. Installed CLI 0.159.2 reports native
  hooks enabled; Motley uses those instead of legacy notify or screen heuristics.
  The user must review/trust the hooks in Codex's `/hooks` UI. Explicitly disabled
  hooks and approval policies are preserved. Codex runs with `--no-daemon` so each
  member's reporting environment belongs to its own process.
- OpenCode plugin installation and event mapping, verified against CLI 1.18.21,
  its embedded event definitions and official plugin documentation. Plugin
  callbacks return immediately; bounded background reporting is serialized,
  excludes child sessions, preserves pending approvals across busy events, and
  supplies the last assistant text excerpt at idle. Reporter errors are contained.
- Native resume for Codex and OpenCode using the latest recorded session id,
  preserving stored options and never replaying the initial prompt. Fresh start
  remains the fallback when no id was recorded.
- The installer configures all installed agents; CLI help and concise README
  usage cover reporting and resume. No new runtime dependency for Motley itself;
  Node 24 runs the OpenCode plugin test in development and CI.

Validation passed: internal Go package tests, CLI smoke/non-terminal checks,
real process replacement into fake Codex/OpenCode executables (prompt, fresh
start and resume), plugin subprocess/ordering/child-session tests, installer
tests, ShellCheck, vet, lint, four static platform builds and snapshot packaging.
The installed Codex binary's feature detection and both hook installers were
also exercised in a disposable home, without touching actual agent settings.

The full `go test ./...` run remains blocked at isolated tmux fixture startup
in this session, including the new native-reporting integration tests. Payload
fixtures are synthetic contract examples; live agent-turn recordings, real
tmux reporting/resume acceptance and hosted CI remain outstanding. No model
calls, hook-trust changes, commits, pushes or release tags were made in this run.
Git/gh network access is unavailable and Git metadata is read-only here.

See `internal/agents/testdata/w8-contracts.md` for version evidence, sources and
fixture provenance. Preview: `./bin/mtly` (`0.8.0-dev`). W8 is not yet released or
marked Done; finish its remaining validation and publication before W9.
Deferred: legacy Codex notify/capture compatibility and permission-mode selectors
(the existing separate tickets #1 and #2).

### Installer checklist and history — 2026-10-01

Installer terminal output now consists of concise bullets with green success
and red failure marks. Full command stdout/stderr, hook backup paths, restart
instructions and exit status are saved in a private timestamped log under
`~/.motley/install-history/`; the checklist prints the log path. Repeat runs
keep separate logs. Build failures retain their exit code and leave installed
files intact; individual hook failures remain nonfatal and show a red mark.
Codex still requires restart and explicit hook review/trust.

Installer tests passed for bash/zsh/fish, repeat installs, output separation,
history preservation, build failure, prerequisite refusal, hook failure and
download mode. ShellCheck and diff whitespace checks passed. Other in-progress
agent integration changes were preserved; no commit or push was performed.

Created and verified [#16 — Add an uninstall script](https://github.com/thomashartm/motley/issues/16)
as Open. It covers installation/integration removal, preserved user work and
settings, repeat runs, concise status output and detailed history. The uninstall
script itself is not implemented in this checkpoint.


### Uninstall and release updates — 2026-10-01

Implemented #16 as `uninstall.sh`, a standalone Bash/Python 3 script. It removes
installed Motley commands and owned agent hooks/plugins, disconnects tmux popup
includes, and preserves user settings, shared shell PATH, saved state, worktrees,
branches and running sessions. File identity is checked before any removal;
modified plugins, unrelated binaries/aliases and invalid settings are refused.
Changed settings and removed plugins receive exact backups. Checklist output and
private timestamped logs mirror installation, under `~/.motley/uninstall-history`.
Repeat/partial installs are handled without deleting unrelated files. The script
is included in all four release archives; uninstall tests run in Make and CI.

Added `motley update` / `mtly update`, with `--check` for discovery only. The
command uses `gh` against `thomashartm/motley`, selects the current OS/architecture
archive from the latest published release, verifies SHA-256 against checksums.txt,
and stages both commands in their installation directory. It verifies Motley
binary identity/platform, preserves an existing mtly symlink, refuses unrelated
files, skips current/newer stable versions and rolls back earlier replacements
if a later rename fails. Update and uninstall share an installation lock.
Configuration, worktrees and running sessions are not changed. Full update
details are logged under `~/.motley/update-history`.

Validation: full local `make check` passed (installer, uninstall, plugin, all Go
and real-tmux tests, vet, lint, four platform builds). After adding the shared
uninstall lock, all ten uninstall tests and focused update tests passed again.
ShellCheck, build, diff checks and GoReleaser snapshot passed; archive readback
verified both commands and the exact uninstall script on all four platforms.
Tests cover mixed user hooks, shell configuration preservation, invalid/replaced
files, repeat runs, concurrent operations, checksums and failed-update rollback.
A live read-only `mtly update --check` confirmed there are currently no published
GitHub releases; real release installation remains pending the first publication.
No actual user installation was removed or updated. #16 is Done based on
implementation. The user requested committing and pushing this checkpoint,
including the pending Codex/OpenCode integrations, installer checklist/history,
roadmap tracking, uninstall script and release updater. Hosted CI and release
publication are separate from the local validation above.

### Agent navigation guide — 2026-10-01

Added a short README guide for the tmux prefix, Motley popup/details, returning
to an agent, window/pane navigation, scrollback and detach. Documented opening a
Ghostty tab and attaching an existing member, including how to move the view and
the distinction between creating a terminal tab and sending to an attached tab.
Checked commands against the CLI, popup binding and TUI handlers; checked tmux
bindings and Ghostty's installed default keybindings. Documentation only.

### Persistent shortcuts and mouse navigation — 2026-10-01

Member sessions now show three help rows below their existing tmux status row,
using the configured prefix and the current member's attach command. Spawn,
revive, adopt and attachment install the footer. Prefix H toggles the originating
client between the persistent monitor and its previous session; prefix h retains
the popup. Underlined footer controls support mouse clicks for monitor, details,
detach, windows, previous/next window and scrollback. Clicks dispatch on release
to avoid tmux's rapid-click key translation. Existing key actions remain the
fallback outside Motley controls.

The overview supports clickable List/Details/Actions/Enter/q controls, member
and crew selection, action selection and wheel scrolling. Forms retain keyboard
input. Navigation callbacks target the originating client and its active session;
no navigation command is typed into the agent. README documents the controls.

Validation passed: installer, uninstall and plugin checks; full Go suite; vet;
lint; four platform builds; whitespace check. Real isolated tmux tests cover a
custom prefix, existing status row and key fallback, reattachment, keyboard and
mouse monitor round trips, rapid window/scroll clicks and non-destructive detach.
TUI tests cover navigation buttons, member/crew/action clicks, scrolling, modal
guards and minimum-size handling. Existing monitors must be restarted to load
the updated TUI; agent processes need not be restarted for the tmux footer.

### Local navigation rollout and tmux compatibility — 2026-10-01

The installed binary was still `local-a2ac75b`, explaining the popup-only shortcut
and missing arrow navigation. Hosted macOS CI also exposed tmux 3.7c's single-key
`list-keys` output going to the status line. Shortcut setup now reads the whole
key table and extracts the original binding. Focused parser, real-tmux footer and
overview/monitor tests passed before updating the local installation.
Installed `local-76fc7f4` with `install.sh --local` (all checklist steps passed),
closed the old popup and switched the existing client to `_motley`. Readback
confirmed the full monitor, clickable navigation, enabled mouse support and
preserved `feat-cleanup-tasks` session. The updated PR's hosted checks are pending.

### #15 follow-up — Grouped footer — 2026-10-01

- Grouped footer shortcuts by navigation, actions, view and session operations.
  Whole groups wrap into at most two reserved rows; small terminals use compact
  navigation and confirm/back controls. Extra actions stay in the Actions menu.
- Accounted for footer height in panels and prompt scrolling, retaining the
  60×10 minimum. Footer text leaves room before the terminal's right edge.
- Added resize checks for every footer context, from 60×10 to 160×30, including
  whole-view bounds and essential controls. Existing navigation bindings remain.

Validation results are recorded on the follow-up PR. This is a footer correction
only; the next delivery item waits for feedback.

### PR #18 merge with grouped footer — 2026-10-01

Preserved both delivery histories and combined clickable navigation with the
grouped footer. Navigation buttons occupy the last reserved footer row; short
terminals show compact hints or just the buttons. Mouse hit testing uses the
same content height as rendering, so footer clicks cannot select hidden rows.
Validation results are recorded on PR #18.

### Open agents directly from the monitor — 2026-10-01

Added a persistent Open agent button and o shortcut, plus Open agent as the first
Actions item. A monitor with no separate work client opens the selected member
in its own tab; another attached work tab remains preferred and a missing pin
still refuses rather than redirecting. The agent footer labels its clickable
return control Back to monitor. No shell command or tmux prefix is needed for
the normal open/return route.

Implemented in an isolated worktree, leaving the other session's checkout
untouched. Rebased onto its merged footer work, preserving the grouped layout
and bottom-row mouse controls.

Validation passed: full Go suite, vet, lint, four platform builds and whitespace
checks. Real tmux tests cover opening with Enter and clicking Open agent, then
clicking Back to monitor, with no separate work tab. Tests also retain the
separate/pinned work-tab behavior and guard ambiguous monitor clients.

### Selected member control and independent monitor — 2026-10-01

The Actions panel names its selected member. Terminate agent (X) confirms the
target and stops its owned tmux session while keeping the worktree, branch and
history for Revive. Confirmation supports mouse clicks and arrow keys. Retire's
confirmation, Force, Keep branch and Cancel controls also accept mouse clicks.
Lifecycle completion refreshes the member list immediately.

Running bare Motley inside an agent now switches to the independent monitor,
preventing the self-retirement refusal caused by an overview running inside its
target session. Ownership checks and explicit force for discarding work remain.

Validation passed: full Go suite, vet, lint, four cross-builds and whitespace
checks. A real tmux regression starts the overview inside its target agent,
terminates it without losing dirty files, revives it, then force-retires it
while the monitor stays alive. UI tests cover mouse targeting, confirmation,
cancellation, changed row order and resized footers. Other session's checkout
was left untouched.

### Visible panel shortcuts — 2026-10-01

The bottom bar now labels direct keys: [o Open agent] [1 List] [2 Details]
[3 Actions] [q Close]. Number keys focus their panel without a tmux prefix;
Actions says Enter runs the selected command. Text editors retain numeric input.

Full tests, vet, lint, cross-builds and whitespace checks passed. Real tmux tests
click all three panel buttons and exercise their number keys. The user's live
mouse failure is not yet reproduced: mouse reporting was enabled, but the client
detached before the requested click trace. Temporary tracing was removed and
the original MouseDown1Pane binding restored. Do not consider that report fixed
based only on automated mouse tests.

### Import running Claude sessions — 2026-10-01

Added `mtly import --list`, `mtly import <session-id>` and the monitor's Add
existing Claude action (a). Discovery uses Claude's supported `agents --json`
interface and rechecks the selected conversation before registration. Imported
sessions remain running in their original terminal, including sessions in main
checkouts or non-Git directories. Their live status appears in the monitor.

Open agent focused a unique matching Ghostty directory through AppleScript. That
broke the "No AppleScript" principle and was removed before landing on main (see
the #30 entry).
Terminate revalidates the Claude session/process, then waits for it to stop.
Revive resumes the saved conversation under Motley. Imported directories and
branches are always preserved on retirement, including after revival.

Validation: full Go suite and vet passed; focused import tests passed again
after error-message lint fixes; lint and four cross-builds passed. Tests use
isolated tmux servers and harmless fixture processes for main, linked and
non-Git directories, discovery failure, duplicates, resumed conversation IDs,
file/branch preservation and the real terminal picker. Read-only discovery
found six local running sessions. Ghostty scripting compiled and its directory
inventory was verified; no live Claude sessions were imported or interrupted.

### First-run configuration — 2026-10-01

Configuration now lives at ~/.motley/config.toml. The first config load or
`mtly init` creates it with defaults, or copies an existing XDG config verbatim.
Existing files, including invalid TOML, are never reset or overwritten. A
complete temporary file is linked into place atomically so concurrent first
launches cannot expose partial content or overwrite each other. Tmux snippets
and blueprint paths retain their current locations.

Validation passed: full Go suite, vet, lint, four cross-builds and whitespace
checks. Tests cover defaults, legacy settings and comments, concurrent first
loads, unchanged modification times, invalid-file preservation and CLI loading
from the new location.

### Monitor shortcut — 2026-10-01

Changed the monitor toggle to prefix m (Ctrl-b m with the default prefix).
Updated the member footer, monitor return hint and README. Prefix h still opens
Details. The real-tmux shortcut test now exercises m in both directions with
custom/default prefixes and preserves the user's m binding outside Motley.

Validation: binding parser and real-tmux shortcut integration tests passed.

### Claude permission modes (#1) — 2026-10-01

`motley spawn --mode <preset>` and a Permission mode step in the spawn form
(after Blueprint, Claude only) select manual, acceptEdits, plan, auto, dontAsk,
bypassPermissions or sandbox. Omitting the mode keeps today's bare `claude`, so
Claude's configured default still applies; there is no CLI menu or TTY
requirement. Presets are hard-coded argv in `internal/agents/modes.go`, resolved
in Prepare before any worktree, branch or manifest exists. Unknown presets, a
non-Claude agent, or a blueprint that already sets the same option (or
`--dangerously-skip-permissions`) are refused. The manifest records `mode` and the
resolved `agent_args`; revive reapplies them. Old manifests are unchanged.

Verified against Claude Code 2.1.286: `--permission-mode` lists manual (the
issue's "default" is now a hidden alias, so Motley does not offer it). The
script's sandbox block was not ported: `mode: autoAllow` is not a setting, and
`denyWrite ["~/"]` would override the worktree under `~/worktrees`. The sandbox
preset is `--permission-mode acceptEdits --settings
'{"sandbox":{"enabled":true,"failIfUnavailable":true,"autoAllowBashIfSandboxed":true}}'`,
so no file is written to the worktree.

macOS confinement was checked with real Claude (haiku, `-p`) in a linked
worktree under `$HOME`, and confirmed on disk: worktree writes and `git commit`
succeeded; writes to `$HOME`, the main checkout, `.git/hooks` and `.git/config`
were denied. Linux verification with real Claude is deferred to #26.

Validation: preset/conflict unit tests, TUI mode-step tests, real-tmux tests
with a fake Claude recording argv/CWD (spawn, revive, blueprint combination,
refusals leave no worktree/branch/manifest), the terminal spawn-form test,
full suite on macOS, cmd/motley ×2 on an Ubuntu 24.04/tmux 3.4 replica,
golangci-lint v2.14.0 (0 issues), vet, gofmt and four cross-builds.

### W8 live acceptance (#11) — 2026-10-02

Ran real Codex 0.159.3 and OpenCode 1.18.21 under Motley in a disposable home
with its own tmux server; credentials were copied in, unchanged by the run and
deleted afterwards. Both agents reported working, permission, question, ready,
interrupt and quit, and `motley revive` resumed the recorded session with its
context. Recorded payloads are now `internal/agents/{codex,opencode}/testdata/live.json`,
replayed by the parser tests and through real tmux; their expectations matched
every event Motley logged live.

Fixed from the run: OpenCode reports nothing on `/exit`, and no agent reports a
crash, so the member kept its last status (for example `ready`) with a shell in
the pane, where a reply would have run as a command. The pane now runs the
hidden `motley agent-exited` after the agent returns, recording `ended` once.
OpenCode's idle events after an abort showed the partial reply as finished; the
plugin marks them interrupted, matching Codex's `idle` / "Turn interrupted".
Uninstall recognises both shipped plugin revisions.

Known limitations, without a native signal: a bare or revived Codex/OpenCode
member shows `starting` until the first prompt; Codex asks questions only in Plan
mode, whose Stop has no message; OpenCode logs two Stop events per turn. Details:
`internal/agents/testdata/w8-contracts.md`. W8 is Done; the legacy Codex fallback
is #27.

Validation: `make check` stages (installer, uninstall, plugin, all Go and real
tmux tests, vet), golangci-lint v2.14.0 (0 issues), gofmt and four cross-builds.
The new exit test fails with the pane change reverted, and the plugin test fails
with the interrupt marking disabled.

### Imported sessions: no terminal control outside tmux (#30) — 2026-10-02

PRs #22 (import) and #23 (first-run config) had been merged into stacked branches
and never reached main; this change merges them. #22 focused the original Ghostty
tab through AppleScript and failed with "Cannot identify a unique Ghostty tab"
whenever several tabs shared the session's directory. AppleScript contradicts
REQUIREMENTS §0 ("No AppleScript or Ghostty API"), so the focus is removed rather
than repaired. A tty-title variant built in PR #32 was removed before merge for
the same reason.

Open agent, `attach` and `switch` on a running imported member now explain where
it runs: "<name> runs in its original terminal in <dir>; switch to it there, or
Terminate and Revive to run it in Motley." Reply and send messages no longer point
to Open agent. Revive remains the tmux-only way to take a session over.

Validation: `TestNoAppleScript` scans every Go, shell and TypeScript source and
fails on AppleScript use; it caught the removed package when restored. The import
lifecycle test checks the attach/switch explanation for main, linked and non-Git
sessions. The Claude discovery contract test now uses a generous bound through a
test seam; it had hit the 3 s production bound at a load average near 20.

### Wider member list — 2026-10-02

The list used a third of the window, capped at 42 columns, so details took about
80% of wide windows (200 columns: 42 | 153). The panels now split evenly and the
list stops at 100 columns: 80 → 40 | 35, 120 → 60 | 55, 200 → 100 | 95,
240 → 100 | 135. `TestPanelSplit` checks those widths and that no rendered line
exceeds the window; the full suite, including the real-terminal monitor tests,
passed unchanged.


The message line above the footer is cut to the window width and the monitor owns
the mouse, so messages such as errors could not be copied. **c**, a click on the
line, or **Copy message** (last in Actions, shown only while a message is present)
copies the full text: the poll error when there is one, otherwise the message.
Busy and filter text are not messages. The line shows **[c copy]**, then
**✓ copied** until the message changes; failures replace it with "Copy failed: …".

Inside tmux, `tmux load-buffer -w` stores a paste buffer and forwards it to the
client terminal as OSC 52. The monitor targets its most recently active tab, the
same rule as detach (now one helper). With `set-clipboard off` tmux cannot forward,
and Motley says so rather than reporting success. Outside tmux Motley writes OSC 52
itself. No AppleScript, `pbcopy` or terminal-specific API.

Validation: TUI tests cover the full untruncated text, poll-error preference,
busy/filter exclusion, no message, key, click, Actions entry, failure, monitor tab
choice and the exact OSC 52 bytes. A real-binary test runs the overview in an
isolated tmux pane with a pty client: the client receives OSC 52 with the payload,
`show-buffer` matches and `set-clipboard off` reports the paste-buffer-only result.
Four mutations (no `-w`, no off check, truncated copy, wrong monitor tab) each fail.

### Actions panel grouping and consequence help — 2026-10-02

Grouped actions under Member, Crews & members, Session & cleanup, and View,
with headings and blank lines between groups. Keyboard selection and mouse
hover update a highlighted explanation at the bottom of the panel. Terminate
and Retire use caution styling and explain confirmation, file/branch retention
or removal, and the effect of Force; Revive explains immediate restart and
its requirement for an existing worktree.

Enabled mouse motion without holding a button. Hover never executes an action;
click and Enter retain existing dispatch and confirmation behavior. Rendering
and mouse hit testing share the grouped rows, excluding headings, gaps and the
help box. Scrolling retains stable pointer targets. Small terminals use compact,
truncated help while keeping the selected action reachable.

Validation: TUI tests cover grouping, consequences, hover versus click, scrolling,
resize, empty selection, footer clicks and 60x10 bounds. The real-tmux overview
test verifies no-button hover and the resulting explanation. Installer,
uninstaller, plugin and all Go tests passed, as did vet, golangci-lint 2.14.0
(0 issues), formatting and four cross-builds. The linter was absent from PATH;
the pinned release was run from a temporary directory after make check reached
that missing-tool step. No manual visual acceptance is claimed.

Follow-up: removed the help heading. The box now fits only its explanation,
with no vertical padding and one character of horizontal padding. Reserved
space stays above the box so its bottom edge remains flush with the panel
interior without moving menu rows on hover. Updated layout assertions and
all TUI tests passed; rebuilt `bin/motley`.

### Panel dividers, Revive validation and lowercase shortcuts — 2026-10-02

Panel headings now have a dotted divider and a blank row beneath them. The
minimum-height layout remains compact to preserve controls. Mouse coordinates
account for the added rows, and the Actions explanation remains at the bottom.
Its enclosing box and heading are gone: a dotted divider above the highlighted
text matches the panel heading style, with no side or bottom borders.

Revive was using the pre-removal worktree validator, so a member recorded in the
main checkout failed with “refusing to remove the main worktree.” It now checks
registered checkout identity without applying deletion restrictions. Both main
and linked checkouts must match the recorded branch. Retirement still uses the
strict removal check and refuses the main checkout, including with Force.

Replaced case-only shortcut distinctions: d terminates, x retires, r revives,
c manages crews, g changes grouping, p pins a work tab, t sends to a tab, and
h toggles inactive crews. Menus, footer hints, README and terminal tests use the
new bindings.

Targeted validation passed: all three Revive input routes, real-tmux lifecycle,
crew and overview tests, main-checkout restart with saved session and preserved
files/HEAD/worktree registrations, and unchanged main-checkout retirement
refusal. Layout assertions cover dotted headings, spacing, borderless help,
bottom anchoring and compact terminal bounds.

Final validation: full `make check` passed with the pinned golangci-lint 2.14.0
binary: installer/uninstaller/plugin tests, all Go and real-tmux tests, vet,
lint (0 issues), and four cross-builds. Rebuilt the local `bin/motley` executable.

### Edit panel field and action styling — 2026-10-02

Editor labels are bold with trailing colons, above indented values; empty
fields show a muted placeholder. Fields have blank rows between them. Actions
use clearly bracketed, coloured buttons with a filled focus highlight and
spacing between controls. Save/Send/Delete and Cancel remain at the bottom,
while the fields scroll to keep the active label and value visible.

Clicking a label or value focuses that field. Buttons and the Force unassign
toggle are clickable; blank rows and hints do not activate actions, and a
pending save blocks further clicks. Crew deletion uses the same styling.

Validation: all TUI tests, real-terminal crew editing and overview tests, vet,
golangci-lint (0 issues) and build passed. The crew terminal test now saves
through an actual mouse event as well as Ctrl-s. Layout/mouse tests cover
field selection, Save/Cancel/Send/Delete, the force toggle, pending writes,
spacing, resize and 60x10 bounds. Rebuilt `bin/motley`.

### Colour selector in editors — 2026-10-02

Member and crew colour fields now render a swatch/name selector with visible
left/right arrows. Arrow keys and mouse clicks cycle through the shared palette
and wrap through Inherit (member) or Automatic (crew). The default choice stores
an empty override, preserving the existing inheritance behaviour. Existing
colours initialise the selector; text entry, paste and deletion do not mutate
it. Tab/up/down/Enter retain field navigation, and colours apply on Save.

Validation: all TUI tests, real-terminal crew/member saves using arrow-selected
blue/yellow colours, vet, lint (0 issues), formatting and build passed. Selector
tests cover every palette entry, wrapping, inherited/automatic defaults, saved
values, mouse arrows, rejection of text input and the minimum terminal size.
Rebuilt `bin/motley`.

### PR integration checkpoint — 2026-10-02

Rebased the UI and Revive changes onto current main, preserving Claude import,
clipboard copying and the wider panel split. Manage crews now uses m so c keeps
its existing Copy message action. Imported-member help explains that retirement
keeps files, and that Open agent reports the original terminal location. Import
picker mouse targets account for heading dividers. Compact Delete/Cancel spacing
fits the narrower detail panel, and terminal mouse tests use the new split.

The prior validation entries above describe the earlier branch snapshots;
validation of this integrated revision is recorded below.

Integrated revision validation: all Go tests passed, including the updated
terminal Save/Cancel assertions, real mouse/colour editing, clipboard and import
flows, and the main-checkout Revive regression. Installer and uninstaller tests,
vet, golangci-lint (0 issues), and four cross-builds passed. The unchanged
OpenCode plugin reporter test missed its first event on the initial make check
and a focused rerun; the baseline copy and a subsequent branch rerun passed.
No plugin implementation or test was changed for that intermittent failure.

### Crew selector in member editor — 2026-10-02

The Crew field now shares the colour selector's keyboard and mouse pattern.
Left/right arrows cycle through existing crew names with colour markers and
No crew, while saving the corresponding ID automatically. Selection stays tied
to its ID when the crew list refreshes. Empty lists, unavailable assignments,
long names and compact terminals are handled without accepting typed IDs.
Changes apply on Save; Cancel leaves the member unchanged.

Validation: all TUI tests, real-terminal crew/member editing and arrow navigation,
TUI vet, golangci-lint (0 issues), formatting and build passed. Tests cover
keyboard/mouse selection, wrapping, refreshed lists, empty/unavailable crews,
long Unicode names, cancellation and 60x10 bounds. Rebuilt `bin/motley`.

### Overview actions, grouped member table and details — 2026-10-02

Status sections now group members by crew and render aligned title, ticket and
crew columns with the existing status, agent and colour indicators. Overview
stays pinned above the list and is selectable by mouse, Home or Up from the first
entry. Main actions are separate from member actions; opening an agent from
Overview uses an explicit session picker, while Spawn, Add and Manage crews need
no member selection. Polling preserves Overview and picker targets by identity.
Rendering and mouse hit testing share the list layout, including decorative
headings, scrolling and compact terminals.

Ticket hyperlinks accept explicit web URLs and resolve numeric GitHub/GitLab.com
tickets using the member's remote. Details now have bold labels, aligned and
wrapped values, consistent field gaps and Workspace/Session sections with dotted
dividers. Narrow panels stack labels above indented values, matching the editor.

Validation: all Go tests passed, including real-terminal Home/Open picker,
editor navigation, monitor switching and popup operation. Model coverage includes
grouping, selection and polling, action scope, keyboard/mouse opening,
removed/stopped targets, ticket links, Unicode wrapping and 60x10 bounds.
Vet, golangci-lint (0 issues), build and diff checks passed. Rebuilt `bin/motley`.

### Ticket clicks and tmux mouse/prefix — 2026-10-02

Left-clicking a visible ticket or crew hyperlink now opens it through the system
browser, including in tmux panes and popups. Hit testing follows the rendered
cells, including scrolling and Unicode. Wrapped detail hyperlinks close at each
line boundary so they cannot extend into adjacent labels or the other panel.
Hover/release and background links behind editors do not open a browser; opener
failures appear in the message line. Web URLs are passed as individual process
arguments without shell interpolation.

New tmux configuration enables mouse support, uses Ctrl-a as its prefix and
binds Ctrl-a Ctrl-a to send-prefix. Native xterm-family hyperlinks are enabled
when tmux is 3.4 or newer; plain TUI clicks also work without native hyperlink
support. Existing user config remains preserved by init, with an upgrade snippet
in the README.

Validation: all Go tests passed, including real tmux pane/popup ticket clicks
with a stub browser, ordinary mouse navigation, Ctrl-a popup/detach and config
reload checks. Vet, golangci-lint (0 issues), build and diff checks passed.
Applied the interaction settings to the local tmux config after backing it up;
server readback confirms Ctrl-a and mouse on across existing sessions.

### Compact Details rows — 2026-10-02

Removed the empty row between each Details field. Labels and wrapped values keep
their alignment; dotted headings and spacing between sections remain intact.

Validation: TUI tests, targeted real-terminal overview/monitor and mouse ticket
link tests, TUI vet, build and diff checks passed.

### Crew wordmark — 2026-10-02

The TUI header pairs three crew bars in cyan, violet and amber with a bold
uppercase MOTLEY wordmark, followed by amber ASCII devil horns (`\m/_`). Both
overview and monitor keep a single header row.

Validation: TUI tests, real-terminal overview/monitor checks, TUI vet, build and
diff checks passed.

### Footer menu separators — 2026-10-02

Added middle-dot separators between the Open agent, List, Details, Actions and
Close buttons. Narrow terminals omit padding around separators to keep every
button visible on one row. Mouse targets use the displayed layout; separators
do not activate adjacent buttons. Hit testing measures terminal columns so the
Unicode dots do not offset mouse targets.

Left-panel selection uses two colour blocks in a fixed two-column marker area,
replacing reverse-video highlighting. The first block, status and text stay in
place when selection changes, including crew entries and Overview.

Validation: TUI tests, narrow/wide footer bounds and mouse checks, real-terminal
overview/monitor checks, TUI vet, build and diff checks passed. The tmux ticket
link test initially timed out opening its popup and passed on a targeted rerun.

### List heading spacing — 2026-10-02

Removed the blank row between the left-panel divider and the ST/AG/TITLE table
headings. The list uses the recovered row for members; scrolling and mouse
coordinates account for its tighter header spacing.

Validation: TUI tests, real-terminal overview/monitor and tmux ticket-link tests,
TUI vet, build and diff checks passed.

### W9 GitHub on demand — Phases 1 and 2 (#12) — 2026-10-02

Phase 1 (links): **b** opens a browser menu with the selected member's
branch, compare view, issue and crew links; only http/https URLs are offered.
Details link Branch and Compare when origin is on GitHub. One remote parser
(`gitx.Web`) serves these links and the existing ticket links.

Phase 2 (issues and crews): `internal/gh` wraps `gh` behind a runner
interface with typed missing, unauthenticated, scope and timeout failures.
A numeric ticket on a GitHub origin looks up the issue once at spawn (10 s);
its title and URL are recorded in the manifest and `Issue.*` reaches
blueprints (bodies capped at 32 KiB on a rune boundary). The parent issue,
else the milestone, suggests a crew: a crew with the same URL is assigned,
the spawn form offers a Crew step, and the CLI prints a copy-ready hint or
creates it with `--create-crew` after the worktree exists. `--no-gh` skips
the lookup. `crew add --url` and the crew form fetch a missing title for
GitHub issue and project URLs. Shell quoting moved to `internal/shellx`.

Validation: gofmt clean; `go vet ./...`; `make test` passed, including fake-gh
CLI integration tests over a GitHub-shaped origin served by a fake ssh, an
absent-gh PATH test and a real-terminal spawn form test that drives the async
lookup and Crew step; golangci-lint 0 issues; cross-build ok. Live read-only
check against this repository:
`motley crew add --url https://github.com/thomashartm/motley/issues/12` →
`Created crew w9-github-on-demand-and-links (W9 — GitHub on demand and links, red)`.
With a scratch `XDG_CONFIG_HOME` and no `GH_CONFIG_DIR`, gh found no login and
crew add printed one hint line (`GitHub CLI is not authenticated; run gh auth
login; or pass --title`) and exited 1. A live spawn lookup was not run because
spawn pushes a branch.

### W9 GitHub on demand — Phase 3 (#12) — 2026-10-02

Phase 3 (pull requests): **u** refreshes the selected member's PR (newest for
the branch, any state) and, for numeric tickets, its issue title; **U**
refreshes every GitHub member with one `gh pr list` per repository. Results
are saved under `[gh]` in the manifest through a locked reload, so edits made
while gh runs survive. Details show the PR with its checks, review and age;
the crew table gains a PR column on wide terminals. **P** offers Create PR
(`gh pr create --fill`, warning about unpushed commits), Mark ready for review
for drafts, and Open PR. Retire warns about an open PR in the TUI dialog and
on the CLI, best effort with a 5 s limit; the PR stays open. Lower and upper
case `u`/`U` resolve the §14 R/Shift+R conflict; `c` and `p` keep their
meanings.

Deliberately left out: automatic or periodic refresh, a CLI refresh command,
lookup on adopt, a `crew_suggest` setting, a `fetch_issue` blueprint key,
persisting issue bodies, and backfilling v0.7.0/v0.8.0 tags.

Validation: gofmt clean; `go vet ./...`; `make test` passed, including fake-gh
member tests (refresh, refresh-all grouping and stop on missing/unauthenticated
gh, concurrent manifest edits, unpushed-commit warning against real git), TUI
key/menu/column tests and CLI retire warnings with and without gh; golangci-lint
0 issues; cross-build and goreleaser snapshot ok. Live read-only check against
this repository with scratch members (`XDG_STATE_HOME`) driven through the
built TUI in an isolated tmux server: `u` on branch `feat/w9-github` →
`PR #41 open · checks ✔ passing · open` (rechecked after the review fixes),
details `PR #41 open · checks ✔ passing`; a
`fix/crew-selector` member showed `PR #38 merged`; `U` → `Refreshed 2 members
in 1 repo`; `P` → `Open PR #41 in browser`. `member.OpenPR` returned #41 for
the retire warning.

Live PR check, 2026-10-02, on a throwaway branch with one pushed and one
unpushed commit: **P** → Create PR printed `Created
https://github.com/thomashartm/motley/pull/47 · 1 local commit is not pushed;
the PR shows pushed commits only`, and details showed `PR #47 open · checks …
pending`. After `gh pr ready 47 --undo`, **u** showed `PR #47 open · draft ·
checks … pending`; **P** offered Mark ready for review and Open PR, and Mark
ready printed `PR #47 is ready for review` (GitHub: not a draft). `motley
retire` printed `warning: PR #47 is still open: …` before refusing the
unpushed work. #47 and its branch were then closed and deleted.

Final review fixes: editing a ticket drops the issue recorded for the old one,
and `u` applies an issue title only while the ticket still matches; a failed
issue read no longer discards the PR refresh (it is reported after the PR
state); `U` bounds each gh call by the 10 s lookup limit and, when a
repository's 200-PR batch is full, looks up unmatched branches individually
instead of recording "No PR"; P and u are offered only for GitHub members and
U only when one exists; a PR with no available action shows a message instead
of an empty menu; a created or readied PR is reported even if the follow-up
refresh fails; GitHub titles are stripped of control characters before they
reach manifests, crews or CLI output; `u` reports draft, checks and review.

Release: merged in [#41](https://github.com/thomashartm/motley/pull/41) as
`f76e73a`; release tag: `v0.9.0`. All nine
[main CI jobs](https://github.com/thomashartm/motley/actions/runs/37041383867)
and all nine [release-tag CI jobs](https://github.com/thomashartm/motley/actions/runs/37041671700)
passed. v0.7.0 and v0.8.0 remain untagged, as agreed.

### Import existing Codex sessions (#50) — 2026-10-02

`mtly import --agent codex --list` and `mtly import --agent codex <id>` now
register existing Codex conversations, with optional name and crew overrides.
The TUI's Add existing agent action offers Claude and Codex before its session
picker. Claude remains the CLI default and existing manifests remain compatible.

Discovery uses `codex app-server daemon version` to locate the existing local
Unix socket, followed by WebSocket initialize, thread/loaded/list and metadata-only
thread/read calls. It never starts a daemon, reads transcripts, subscribes,
resumes or starts turns during import. Imported identity is rechecked before
opening; duplicate, stale and unavailable selections fail without launching.
Standalone --no-daemon sessions, remote endpoints, ephemeral threads and
subagents are outside this slice.

Codex Open/attach/switch and Revive use `codex resume --remote unix://<socket>
-- <thread-id>` so the original server remains the conversation executor.
Retire closes only Motley's terminal client and archives its entry, preserving
running Codex work, conversation, files and branches. Terminate is refused for
these shared sessions and omitted from their Actions menu. Codex status is read
from its server, including approval/input waiting flags; failed discovery never
silently declares sessions dead. Reply rechecks the server's approval state
before sending, because shared sessions do not emit Motley terminal hooks.

Evidence: read-only discovery on Codex CLI 0.159.3 / daemon 0.160.0 found three
eligible local sessions without importing or interrupting them. An isolated
native 0.159.3 server/client test resumed a persisted fixture conversation and
rendered its existing message; the loaded thread ID was unchanged and no second
thread appeared. The fixture used a local unavailable model endpoint and an
interrupted test turn; no real model inference or user conversation was used.
The initial empty-thread probe could not resume before any turn was persisted.

Validation: full Go suite, focused CLI/TUI import and real-tmux open tests, race
checks for Codex discovery/TUI/member operations, vet, golangci-lint, four-platform
cross-builds, installer/uninstaller and OpenCode plugin tests passed. Coverage includes
main/linked/non-Git checkout preservation, exact remote-client argv, stale IDs,
changed directories, duplicate imports, missing crews, server failure, pagination,
status mapping, keyboard/mouse selection and legacy Claude import.

### Refresh all GitHub as a main action — 2026-10-03

**Refresh all GitHub (U)** moved from the member action menu to Main actions
in Overview, shown once a member is on GitHub; the `U` key still works with a
member selected. The member footer hint is now `GitHub: b browser · P PR · u
refresh`. Policy check of the action menus: member and session actions act on
the selected member; main and view actions appear only in Overview. Copy
message (`c`) stays in both menus because it copies the status line, not
member data.

### Copy drag selections to the macOS clipboard (#29) — 2026-10-03

Member and monitor sessions now pipe drag selections to `/usr/bin/pbcopy` on
macOS, in both Emacs and vi copy modes. Drags enter tmux copy mode even when
an agent or the monitor requests mouse events. Ordinary clicks, wheel events
and the member footer retain their navigation behavior. Existing members gain
the bindings on reattachment; `mtly monitor` reapplies monitor setup.

The bindings use the existing Motley session guards and preserve the user's
original actions elsewhere, including chained commands. Repeated setup does
not nest wrappers. No global mouse, copy-command, set-clipboard, terminal
feature or override settings are changed, and no configuration files are
rewritten. As with navigation, tmux's server-wide key tables contain guarded
wrappers and Motley metadata remembers their fallbacks. There is no AppleScript
runtime dependency.

Validation: the full `make test` suite and `go vet ./...` passed. `make check`
stopped at its lint step because the executable was absent; running the CI-pinned
`golangci-lint` v2.14.0 with `go run` then reported 0 issues. `make cross-build`
passed for Darwin/Linux on amd64/arm64. New isolated real-tmux tests send mouse
press/drag/release sequences for member, monitor and unrelated sessions in both
copy modes, verify the pipe contents and tmux buffer, exercise custom chained
fallbacks, and check idempotence and unchanged clipboard/click/wheel options.
The pipe writes a temporary file so automated tests do not change the user's
clipboard. Existing real-tmux footer, monitor and clipboard-message tests pass.

Ghostty check: agent-operated GUI smoke test with Ghostty 1.3.1 and tmux 3.6a,
using a throwaway server and fixture member. With `set-clipboard off`, dragging
`CLIPBOARD_FIXTURE` in the member pane put that exact text in `pbpaste` and the
tmux buffer. Clicking **Back to monitor** opened the actual monitor; dragging
its `MOTLEY` heading reached the macOS clipboard, and clicking **Actions** still
changed the monitor panel. Ghostty's AppleScript mouse API was used only by the
validation harness, outside the repository. The fixture session/window was
closed and the previous plain-text clipboard restored afterward.

Human manual check still pending: in Ghostty, drag text in a member and the
monitor, then press Cmd-V in another app and confirm the text; check the footer
and monitor clicks and wheel scrolling. The agent-driven check above verifies
the clipboard contents but does not claim a human cross-app Cmd-V check.
iTerm2 and Terminal.app were not exercised interactively; local macOS uses the
same terminal-independent pbcopy path, so OSC 52 support/settings are not needed.
Linux retains its existing copy bindings. For remote macOS sessions, pbcopy
writes to the host Mac's clipboard, not the SSH client's clipboard.

PR preparation: rebased `fix/29-macos-clipboard` onto main at `1373916`,
preserving both delivery entries. Focused real-tmux clipboard/footer/monitor
and TUI GitHub/navigation tests passed again after the rebase.

### README screenshot — 2026-10-03

The README shows a centered screenshot of the TUI (`motley-screen.png`, 900 px
wide, linking to the full-size image) between the introduction and Install.
Release archives include it alongside the logo so the packaged README renders.

### Claude status sync and switching tracked sessions — 2026-10-04

Repository checks: local Git CLI for repository work, gh for GitHub operations,
and delivery evidence here.

The reported VAT member tracked `f02922cf…` in a detached Motley tmux terminal,
while the user's visible original terminal was running `abf42be6…` in the same
checkout. Its tmux status had only the initial SessionStart/idle event. Imported
Claude members now refresh from live discovery even when a tmux session exists;
unknown transition ages no longer reuse the old tmux timestamp.

Claude hooks now reach imported sessions in their original terminals by exact
session ID and checkout. A locked activity snapshot separates parent and child
states. SubagentStart/SubagentStop and Stop's background-task snapshot keep an
idle parent working while children run; real permission requests and questions
remain visible. Duplicate child events are idempotent, completed children are
removed, and a new session resets activity. Activity state is archived with the
member. No inactivity timeout is used to guess that subagents finished.

Imported Claude members offer **Switch tracked session (Shift-S)**, with live
status, session IDs and checkout paths. CLI: `mtly import --replace <member-id>
--list`, then `mtly import <session-id> --replace <member-id>`. Selection is
revalidated against live, unregistered sessions in the same checkout. Switching
preserves member metadata and history and leaves both processes running. A
former owned tmux session is renamed `untracked-…` and released; its late hooks
and exit cannot overwrite the replacement. Detail views exclude the old
conversation's last message. Codex switching is outside this change.

Validation: full `go test ./...` passed; focused real-tmux reporting/rebinding
and TUI tests passed, including stale idle, original-terminal subagent activity,
wrong-checkout rejection, preserving the released pane, and late-exit isolation.
Race checks, vet, CI-pinned golangci-lint (0 issues), four-platform cross-build,
installer/uninstaller tests and OpenCode plugin tests passed. New subagent hook
registrations require an existing Claude session to reload hooks or restart;
existing tool hooks and live discovery already work with the updated binary.

Local rollout: rebuilt `bin/motley` and installed it in `~/.local/bin/motley`.
The previous binary and VAT manifest are backed up under
`~/.motley/local-fix-backups/20261004T100256Z/`; the hook installer also backed up
Claude settings before adding the two subagent hooks. Used the new replacement
command to reconnect the existing VAT member to `abf42be6…`, preserving its name
and `backend-vat-fixers` crew. Readback showed `working`, and a real PostToolUse
hook at 10:03:14 UTC wrote the replacement session's working activity snapshot.
Both original and detached Claude PIDs remained alive. Reopen Motley to load
the new picker in an already-running TUI.

PR preparation: `fix/claude-status-session-switch`, based on current `origin/main`
at `6084440`. The user requested publication of the verified local changes;
remote CI will be reported separately from the completed local checks.

### README quick start and contribution guidance — 2026-10-04

Implemented the approved shorter README structure, including the requested
Contributing section. The README is roughly one-third its original 452 lines, with capabilities,
numbered setup/start/import steps, action tables, lifecycle consequences, a small
configuration example and development commands. Contributions cover bug reports,
feature proposals, focused branches, behavior tests and PR validation notes.
The introduction explains the user's motivation: one view across local terminal
sessions to retain task context, see running work and focus attention where needed.

Moved optional CLI, crew, GitHub, permission-mode, blueprint, terminal and lifecycle
details to `docs/advanced.md`; recovery instructions are in
`docs/troubleshooting.md`. Corrected configuration guidance against current
source: the main config stays in `~/.motley/config.toml`, while XDG settings
control blueprint/tmux and state roots. Hook instructions account for reporting
from imported Claude sessions in their original terminals. Release archives now
include `docs/*.md` so the new README links work in downloaded packages.

Repository rules checked: Git/gh CLI only; progress recorded here. Changes are on
`docs/simplify-readme`, based on `origin/main` at `c05e93e` after PR #55 merged.
Validation: all 16 local documentation links and anchors, code-fence pairing,
documented command/flag names against source and CLI help, archive file patterns,
and `git diff --check` passed. No application code changed; runtime tests were
not rerun. GoReleaser is not installed locally, so its validation command and a
fresh release archive were not run. The docs are not yet committed or published.

### Multiple repository entrypoints — 2026-10-04

Repository rules checked: use local Git/gh; retain the pending README changes;
record delivery here. Added `repos_roots = ["~/projects", "~/projects/aderis"]`.
Each directory is scanned one level deep for main Git repositories. Both the
spawn picker and CLI resolve across all roots; blueprint and issue lookup use
the same selection. The existing `repos_root` remains supported, with the plural
setting taking precedence. Loading never rewrites an existing configuration.

Duplicate repository names are shown as absolute paths and require an explicit
selection; repeated/symlinked physical repositories appear once. Linked worktrees
remain excluded. Empty/blank root lists are rejected and unreadable entrypoints
produce a path-specific error. Qualified selections retain the repository's
basename in manifests, blueprint filters and worktree destinations. Existing
worktree destination collision checks still apply to same-named repositories.

Updated the default/example config, README, advanced usage and relevant source
requirements. Added config compatibility/validation, discovery, picker and CLI
integration coverage, including actual worktree creation and branch pushes to
isolated fixture remotes from the second entrypoint.

Validation: focused feature tests, race checks for config/member/TUI, `go vet`,
CI-pinned golangci-lint (0 issues), installer/uninstaller/OpenCode plugin tests,
four-platform cross-build, documentation paths/fences and `git diff --check`
passed. Two full-suite attempts hit different integration failures: a tmux
popup/link timeout, then a Codex hook status assertion. Both failing tests passed
individually. A sequential-package full-suite run also failed, this time on an
OpenCode hook status assertion and Codex import picker timeout. These failures
are outside the modified paths, but no clean full-suite pass is claimed.
The final focused rerun of both native reporting subtests and the Codex import
picker passed (4.7 seconds).

Local rollout: installed the verified build in `~/.local/bin/motley` (`mtly`
remains its symlink) and changed only the repository setting in the user's
`~/.motley/config.toml` to the two requested entrypoints. Binary and config backups
are under `~/.motley/local-fix-backups/20261004T110329Z-repository-roots/`.
`mtly config` lists both expanded paths; read-only blueprint discovery resolved
`motley` from the first root and `backend` from the second. The installed binary
matches the local build. Reopen an existing TUI to load the update.

PR preparation: the user requested publication of the multi-entrypoint feature
and pending README improvements. Repository rules rechecked; Git/gh CLI only.
Publishing on `feature/motley-multiple-repository-roots`, based on fetched `origin/main`
at `c05e93e`. The PR includes the shorter README, contribution guidance and
advanced/troubleshooting pages. Local validation and intermittent full-suite
failures are disclosed above; remote CI is separate evidence.

Before publication, the user requested grouped repository presentation and
shorter branch suggestions. Discovery now preserves configured root order and
returns full selectors. The picker groups by root, renders `name (full path)`,
wraps paths, and keeps selection tied to repositories while scrolling/filtering.
The spawn form defaults to `feature/<repo>-<ticket>-<name>` and provides a
`feature`/`fix` choice. Ticket is optional; pasted GitHub issue URLs contribute
only their issue number. Manual branch edits survive task-name edits; explicitly
changing branch type changes only the prefix of a manually edited branch.
Added rendering/filter/scroll/selection and branch-form interaction regression
tests, and updated usage docs to match.

Final refinement validation: member/config/TUI tests and race checks, focused CLI
spawn/config/blueprint integration tests, vet, lint (0 issues), cross-builds and
documentation checks passed. Reinstalled the updated picker and branch form; the
previous binary is backed up at `/Users/thomas/.motley/local-fix-backups/20261004T122602Z-spawn-picker/`.
The earlier broad-suite failures remain disclosed; they were not treated as a
clean full-suite pass.

## Global prompt templates — PR delivery

Repository rules checked: use Git/gh CLI, retain progress in DELIVERY.md and
usage instructions in the documentation. Scope: Main actions template browser,
raw view, vi/default-editor editing, argument entry and copyable prompt generation
for external agents. Preserve existing blueprint selection/delivery during spawn.
No repository, worktree or agent session is needed to generate a prompt.

Implemented **Main actions → Prompt templates (f)** with global file discovery,
raw Markdown/header view, vi and default-editor actions, reload, multiline argument
entry and generated-prompt preview/copy. `$VISUAL` takes precedence over `$EDITOR`;
without either, use the macOS text editor or Linux Markdown application. GUI editors
may return before saving, so Reload remains available. Invalid templates stay
listed for repair, while spawn discovery continues to reject malformed templates.

The generator uses declared variables plus optional standard context fields. It
runs without a member/repository and copies the exact rendered text through the
existing terminal/tmux clipboard path. It does not create a worktree, start an
agent, fetch GitHub data or send a message. Spawn retains its compatible-blueprint
picker, variable form and reviewed initial prompt delivery. Raw/rendered views,
argument values and selection survive resizing; keyboard and list/action mouse
navigation remain modal. Updated advanced usage documentation.

Validation: focused unit and race tests, a real terminal/tmux check of both
editor choices and exact clipboard forwarding, blueprint spawn/resume integration,
vet, CI-pinned lint (0 issues), installer/uninstaller/plugin checks and all four
macOS/Linux builds passed. The first broad run exposed a stale terminal spawn
expectation from the earlier branch-format change. Corrected the test to expect
`feature/api-412-fx-cache`; its focused rerun passed. The final `go test ./...`
passed, including the full CLI integration package (110 seconds).
The local executable is built in `bin/motley` (`bin/mtly`); not installed.

PR preparation: publication requested by the user. Created
`feature/global-prompt-templates` from fetched `origin/main` at `9895b4d`, whose
source tree matches the validated base. This PR contains only the template feature,
its tests/documentation and the corrected spawn-test expectation. Hosted CI is
separate from the completed local validation.

Published [PR #57](https://github.com/thomashartm/motley/pull/57) against `main`.
Implementation commit: `1ec36dc`. Hosted CI results are pending publication checks.

## Existing-agent picker presentation — PR delivery

Repository rules checked: Git/gh CLI only; delivery notes stay here. Scope is the
existing-session picker shared by Claude/Codex import and tracked-session switching.
Improve session hierarchy, spacing, selected styling and readable identity details;
keep session discovery and import/lifecycle behavior unchanged. Started on
`fix/existing-agent-picker` from current `origin/main` after PR #57 merged.

Replaced the dense two-line session list with separated blocks: bold titles
(wrapped to two lines where space permits), a muted working-directory line, and
coloured status plus an abbreviated session ID. The selected block has a cyan
accent and a light/dark-aware background; the header shows selection/total count.
Home paths abbreviate to `~`, long paths preserve their meaningful suffix, and
unnamed sessions have an explicit fallback title. Duplicate session names remain
distinguishable by status and ID.

Scrolling keeps whole session blocks visible, including at 60x10. Mouse targeting
uses the rendered rows so title wrapping and gaps cannot select another session;
headers/gaps are inert and the wheel moves selection without importing. The same
presentation applies to replacement-session selection.

Validation: all TUI tests and TUI race checks, real-terminal Claude/Codex import
and tracked-Claude-session integration tests, vet, CI-pinned lint (0 issues), all
four macOS/Linux builds and `git diff --check` passed. Layout coverage includes
long/wide-character paths, duplicate names, scrolling, resizing and mouse targets.
Built locally in `bin/motley` (`bin/mtly`); not installed.

PR publication requested by the user. Publishing `fix/existing-agent-picker`
against current `origin/main` at `cb0ac5a`; the validated source is unchanged.
Hosted CI is separate from the completed local checks above.

## Consistent footer labels — PR delivery

Repository rules checked: Git/gh CLI only; keep progress in DELIVERY.md. Scope:
footer wording only. Use bracketed group labels throughout, including `[Actions]`,
`[View]`, `[Run]`, `[GitHub]`, `[Shortcuts]` and `[Options]`, matching `[List]` and
other context labels. Remove redundant Nav:/Edit: prefixes inside context groups.
Update existing footer/terminal expectations and verify narrow layouts.

Implemented bracketed labels in all full footer groups and removed the redundant
inner prefixes. Combined the Actions-panel hints under one `[Actions]` label and
shortened the list's search hint to `find` so all groups still fit at 80 columns.
Compact hints and navigation buttons retain their existing keys and behavior.

Validation: all TUI/layout tests, the terminal overview/monitor and arrow-editor
checks, targeted vet, CI-pinned lint (0 issues), local build and diff checks passed.
Terminal assertions distinguish panel context from the generic `[Actions]` group
and cover both crew and ordinary list navigation. Built in `bin/motley`
(`bin/mtly`); not installed.

PR publication requested by the user. Publishing `fix/consistent-footer-labels`
against fetched `origin/main` at `ca3d2e3`. The validated source is unchanged;
hosted CI remains separate from local validation.

## Track foreground and background Claude conversations — 2026-10-04

Repository rules checked: Git/gh CLI, delivery notes here. The reported IDD member
tracked an idle foreground conversation while the visible work and its background
subagent belonged to another conversation. Claude's discovery exposes them as
separate sessions; sharing a checkout does not prove a parent/child relationship.

At the user's request, explicitly link both conversations to one member. Added
`mtly import <session-id> --with <member-id>` and **Track another session (A)**.
The list identifies live conversations as FG, BG or FG+BG; Details shows each
conversation's status, name and ID. Questions/permissions take priority, followed
by working, so an idle foreground cannot hide active background work. Each
conversation retains its own hook snapshot and child activity across foreground
restarts. Unrelated sessions in the checkout do not contribute status.

Open targets the primary conversation, shown in Details. Switching the primary
keeps explicitly linked conversations monitored. Terminate/Retire confirmations
state that they stop all tracked conversations; retirement archives their hook
snapshots and keeps imported files. No running conversations are restarted by
linking them.

Validation: grouping/status/restart/permission, hook isolation, duplicate and
wrong-checkout refusal, primary switching, two-process termination and snapshot
archival tests passed. TUI display/picker tests, member/report/TUI race checks,
vet, CI-pinned lint (0 issues), four-platform builds, installer/uninstaller tests
and OpenCode plugin tests passed. Two full `go test ./...` attempts each had one
different timing failure: Codex native hook reporting, then the tmux popup mouse
test. Both passed twice on isolated reruns; no clean full-suite run is claimed.

Installed the build in `~/.local/bin/motley` and explicitly linked IDD background
conversation `fc950500…` alongside foreground `270a5f58…`. Reloaded only the Motley
monitor; live CLI and monitor readback show **working / FG+BG**. Claude PIDs 32182
and 80891 and their start times were unchanged. Binary and original manifest
backup: `~/.motley/local-fix-backups/foreground-background-20261004T153043Z/`.
PR publication requested by the user. Publishing
`fix/track-foreground-background-sessions` against fetched `origin/main` at
`ca92e91`. The base fast-forward changed no source files; the implementation and
validation evidence above are unchanged. Hosted CI is pending publication.

## Session access and text selection (#61) — 2026-10-04

Repository rules checked: Git/gh CLI, progress recorded here. Created issue #61
from the accepted proposal and started `feat/61-access-labels-copy-selection`
on merged `origin/main` (`34e43fa`).

Implemented independent ACC (TMX/EXT/MIX) and MOD (FG/BG/F+B)
columns with labels at most three characters, per-conversation access in Details,
Actions text selection and complete-row copying with full titles. Drag gestures
in these areas do not dispatch actions or open links; ordinary clicks activate on release.
Ctrl+C copies a selection; Esc clears it. The additional request makes all AG
badges grey. Refreshes preserve selection by member identity and clear it when
selected content disappears or changes; resizing and navigation clear it too.

The macOS tmux monitor drag binding now forwards gestures to the TUI. Agent panes
retain native copy-mode, and the monitor still supports explicit tmux scrollback.
Clipboard copying uses the existing terminal/OSC 52 route and reports forwarding
failures. README and clipboard documentation describe the interaction.

Validation: `go test ./...` passed, including real tmux overview and monitor
drag/copy tests that verify untruncated titles, terminal clipboard forwarding,
and selecting Retire text without activating it. Selection tests cover reversed
multirow drags, Unicode graphemes, refresh identity, resize, Esc, copy errors, and
Ctrl+C behavior. Access/grouping and narrow layout checks passed. Member/TUI/tmux
race checks, `go vet ./...`, CI-pinned golangci-lint (0 issues), local build,
four-platform cross-build and diff checks passed. The initial new terminal test
needed fixture mouse reporting enabled and a row-specific target to avoid hitting
the Details access field; the corrected test passes in the full suite.

Built in `bin/motley` (`bin/mtly`); not installed. Publishing a reviewable draft PR
against fetched `origin/main` at `34e43fa`; hosted CI remains separate from local
validation. No live member session was changed or restarted for this work.

## Preserve Shift+Enter inside tmux agents — 2026-10-04

Repository rules checked: Git/gh CLI; delivery evidence here. Kept this follow-up
separate from the access/selection work (draft PR #62, hosted CI green).

Live diagnosis: tmux 3.6a had extended-keys off, xterm output format, and the
Ghostty client lacked the extkeys capability. A raw-byte terminal fixture
reproduced Shift+Enter arriving as a carriage return. Enabling extended keys
alone does not repair a running pane that missed negotiation at startup.

Generated configuration now enables extended-key support and xterm-compatible
terminal capability detection, selects CSI-u on tmux 3.5+, and forwards Shift+Enter
literally in live Motley member sessions or foreground Codex/Claude/OpenCode
panes. The binding preserves plain Enter and falls back to native handling in
other sessions and after an agent has ended. Existing user files remain untouched
by `mtly init`; upgrade instructions describe the small manual config addition.

Validation: the full Go suite passed. A real tmux/PTY test reproduces the old
failure and checks both Shift+Enter encodings, plain Enter, Ctrl+C, Tab, arrows,
live-pane compatibility, capability discovery after reattachment, and native
fallback outside active agents. The same test keeps the older-tmux config path
loadable without requiring the newer CSI-u option. Vet, CI-pinned lint (0 issues),
local build, four-platform cross-build and diff checks passed. The repository
change is prepared as a separate draft PR against `origin/main` at `34e43fa`.

Applied the same block to the user's existing Motley tmux config and live server.
Backup: `~/.motley/local-fix-backups/shift-enter-20261004T161230Z/`. An attempted
client reconnection left the terminal at the standalone Motley overview; no tmux
client was attached on readback. Reopening an agent picks up terminal capabilities.
All pane PIDs/terminals were unchanged, and the existing Codex and Claude processes
were confirmed still running. No prompt or key was sent to a live agent.

## Consolidated delivery — 2026-10-04

At the user's request, combine the access/selection and Shift+Enter changes in
PR #62 and publish it as ready for review. PR #64 will be closed as superseded.
Both implementations are preserved; the only cherry-pick conflict was the two
appended DELIVERY.md sections, which were retained. The combined full Go suite,
vet, CI-pinned lint (0 issues), local build and four-platform builds passed.
Publishing the combined PR as ready for review; hosted CI will run on the new
head. The combined binary is built locally in `bin/motley`, not installed.

### Clear monitor header labels — 2026-10-04

The monitor header said `Open agent: this tab` and `[attention]` without saying
what they meant or which key changes them. It now shows `opens in: <target> (p)`
and `group: <grouping> (g)`. The target comes from `openClient`, the function
Open agent uses, so a lost pin or several monitor tabs no longer show "this tab"
while Open agent refuses. The **p** picker is titled "Open agents in", explains
Automatic and, with no work tab, says to run `mtly attach <member-id>` in another
tab. Two errors named the wrong key (`T`, `t`); they now name **p** and the
attach step. The help action is "Where agents open (p)".

Validation: unit tests for every header state, both errors, the picker hint and
picker scrolling at heights 4, 8 and full; the tmux pin test now checks
`, pinned (p)`. vet and the full `go test ./...` passed. The first full run had
one timing failure in `TestTmuxMouseTicketLinksAndPrefix` ("timed out waiting for
fixture session"); it passed twice in isolation with and without this change, and
the full `cmd/motley` package passed on rerun.

## PR #63 conflict resolution — 2026-10-04

Repository rules checked: Git/gh CLI and delivery notes here. Merged current
`origin/main` (`b8ab54e`) into the monitor-header PR in an isolated worktree,
preserving the original checkout and its untracked work. The rendering conflict
keeps both `group: <name> (g)` and the selection overlay from PR #62. All delivery
entries are retained. All TUI tests and the real-tmux overview/monitor, selection
clipboard and message clipboard tests passed. `go vet ./...`, CI-pinned lint
(0 issues), and diff checks passed. Pushing the merge resolution to the existing
PR branch; hosted CI will run on the updated head.

## Workspace identity, feature branches and termination — 2026-10-05

Repository rules checked: Git/gh CLI and delivery notes here. Moved this clean
checkout from `main` to `fix/session-workspace-cleanup`, then switched the primary
`~/projects/motley` checkout to `main`. Its untracked `tasks/` and the former
`docs/readme-screenshot` branch are preserved. The reflog shows this worktree was
switched to main after earlier feature work; this was not a spawn operation.
AGENTS.md now explicitly reserves main for the primary checkout, including after
merges and cleanup.

Live diagnosis of `infrastructure-stacks-8d`: its manifest names the primary
infrastructure-stacks checkout on main, but no longer contains `claude_session`.
Its owned terminal runs `claude agents`; the visible conversation works in
infinite-deployment-drive. The shell still reports infrastructure-stacks as its
working directory. The retained log contains only Notifications, spanning two
conversation IDs. Thus a terminal's starting directory did not identify the
conversation displayed by Claude's picker. Hooks trusted inherited MOTLEY_MEMBER
without comparing payload cwd, and resume selected the last event's conversation
ID. The live files do not establish which earlier write removed claude_session.

Changes:
- Hook workspace mismatches produce an attention message without retaining the
  foreign conversation ID. Imported resumes use their explicitly tracked primary
  conversation. Notification and subagent IDs cannot become native resume targets.
- Creation, adoption, import and restart reject protected/default branches in
  linked worktrees. origin/HEAD identifies custom defaults; main/master remain
  the fallback. Existing borrowed primary checkouts remain supported. Cleanup
  retains default branches. Agent startup rechecks checkout registration.
- Terminate has clickable Cancel/Terminate choices beside its explanation and
  retains its footer shortcuts. Owned terminal termination no longer depends on
  successful discovery of unrelated imported agents. Ownership checks remain.

The user reports that the confirmation key never appeared. A fresh isolated
instance of the installed binary displayed its footer at 120x30, so the exact
live clipping condition was not reproduced. New tests click the visible in-panel
choice in a real tmux terminal, verify the session stops and preserve dirty files,
branch and manifest. Resize tests cover the same choices in smaller layouts.

Validation: full Go suite passed; focused tests after the final changes passed.
Coverage includes workspace mismatch and symlink aliases, notification pollution,
explicit imported resume identity, default-branch creation/import/restart refusal,
adoption refusal and termination despite unrelated discovery failure. Vet,
CI-pinned lint (0 issues), installer/uninstaller and OpenCode plugin tests passed.
Four-platform builds and final local installation are checked below.

Local installation completed after four-platform builds passed. Previous binary:
`~/.motley/local-fix-backups/session-workspace-cleanup-20261005T070613Z/motley`.
Installed binary checksum matches `bin/motley`. Restarted only `_motley`'s monitor
pane; readback confirmed every other pane and PID unchanged. The live monitor is
filtered to infrastructure-stacks and displays both `Cancel (esc)` and
`Terminate (y)` inside its confirmation panel, as well as the footer controls.
The dialog is left open with Cancel selected. No agent was terminated and no live
member manifest, conversation, worktree or event history was removed.

Publication requested: publish `fix/session-workspace-cleanup` as a pull request
against `main`. Refreshed origin before committing; the branch base is current
at `6d6aabf`. Source is unchanged from the locally validated implementation.
Hosted CI will run after publication; local verification is recorded above.

## Terminate from the list and remove stopped entries — 2026-10-05

Repository rules checked: Git/gh CLI, feature branches for linked worktrees and
delivery notes here. PR #66 is merged. This follow-up uses
`fix/terminate-list-action` from `origin/main` (`040e308`); only the primary
`~/projects/motley` checkout is on main.

Live diagnosis: `infrastructure-stacks-8d` was already stopped, with no owned tmux
session. Terminate returned an already-stopped message and left it listed. The
user confirmed the desired behavior: stop and remove from the active list,
keeping files, branch and history.

Terminate now stops tracked agents and archives their entries, including entries
that have already stopped. Ownership and shared Codex protections remain; errors
retain the active entry and stay in the confirmation dialog for retry. Successful
termination removes the row immediately, even if unrelated discovery prevents
the next refresh. Archived entries cannot be revived; usage guidance reflects
that distinction.

The bottom bar exposes a clickable `d Terminate` action directly beside Open
agent. Rendering and hit testing share the same labels, including compact labels
for smaller terminals. The confirmation explains what will be stopped and kept.

Focused validation passed: real tmux mouse selection and confirmation, keyboard
input, 60/80/120-column controls, stopped entries, imported foreground/background
agents, archive-failure retry, dirty file/branch/worktree preservation and archived
history contents. Full checks and local installation are recorded below.

Local installation: binary matches `bin/motley`. Backup is
`~/.motley/local-fix-backups/terminate-list-action-20261005T075524Z/motley`.
Restarted only `_motley`; every other tmux pane and PID remained unchanged.
Live readback shows `[d Terminate]` directly in the bottom bar. No live agents
were stopped and no live member entries were removed during validation.

Validation: the full command/integration package passed, then the full TUI suite
passed after adapting an older narrow-footer test to `[q]`. A repeated full Go
run passed all packages except an intermittent Codex import client-switch timeout;
that unchanged test passed three consecutive focused retries. Termination tests
passed in both full runs. Vet, CI-pinned lint (0 issues), installer/uninstaller,
OpenCode plugin tests and all four platform builds passed. Publishing this
follow-up separately because PR #66 is already merged.
