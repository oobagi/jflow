# Build order

Phases below are sized to ship in single sessions. Each phase ends with something runnable and demoable.

## Phase 0 — verification (DONE)

- [x] `claude --help` introspection
- [x] live `claude -p --output-format stream-json --verbose --include-partial-messages` capture
- [x] official CLI reference fetched
- [x] cost finding (33k cache-creation tokens, $0.20 on cold start) — see `06-cost-and-bare-mode.md`
- [x] event-shape catalog — see `02-stream-json-events.md`
- [x] this docs/ directory written so context survives across sessions

## Phase 1 — single-pane chat prototype (NEXT)

**Goal**: prove the harness model end-to-end. One screen, one ephemeral session, real claude, real streaming.

Deliverables:
- `go.mod` with charm.land v2 modules + cobra + toml
- `cmd/jflow/main.go` + `cmd/root.go` (default action: launch the prototype TUI)
- `internal/claude/{driver.go,events.go,usage.go}` — spawn claude, decode JSONL, emit typed events on a channel
- `internal/ui/{app.go,transcript.go,composer.go,statusbar.go,theme.go,keys.go}` — single-pane bubbletea v2 model
- working: type a message → see thinking → see tool calls → see streamed text → status bar shows tokens/ctx/cost
- working: Ctrl+C interrupts current turn
- working: Ctrl+K sends `/compact`
- working: Esc/Ctrl+Q exits

Not yet:
- workspaces / sessions persistence
- tools (only manual chat)
- compaction policy (manual /compact only)
- multi-session resume

Acceptance: launch `jflow`, have a real conversation, ask claude to read a file, watch the tool call render, send `/compact`, see usage drop, exit cleanly.

## Phase 2 — workspaces + sessions + three-pane shell + todo pane

- `internal/workspace/{workspace.go,store.go}` and `internal/session/{session.go,store.go,transcript.go}`
- `~/.jflow/state/` directory layout (workspaces.json + sessions/<uuid>.json with `todos[]`)
- bubbletea root model with three panes: **workspaces · chat · todo**
- `cmd/workspace.go` (ls/add/rm) and `cmd/session.go` (ls/archive/rm)
- new session = new claude `--session-id <uuid>`; existing session = `--resume <uuid>`
- `internal/ui/todopane/` — flat todo list, active indicator, user keybinds (a add, e edit, ⏎ activate, space done, s send-to-chat)
- `internal/mcp/todo/` — bundled MCP server exposing `todo_list/add/set_active/complete/delete/update`; auto-registered when spawning `claude -p`
- One-line system-prompt addendum tells the worker the `todo_*` tools exist and the user can see/edit the list

Acceptance: open `jflow`, see workspaces, pick one, see prior sessions with cost/ctx/timestamps, resume one, see the full prior transcript, send a new message. Worker creates a todo via `todo_add` → it appears in the right pane within the same turn. User adds a todo via `a` → next worker turn sees it via `todo_list`.

## Phase 3 — first tool program: `autopilot`

- `internal/tool/{tool.go,registry.go}` (the interface from `03-architecture.md`)
- `internal/tool/manual/` (no-op tool, for explicit "manual chat" sessions)
- `internal/tool/autopilot/` (port of `skills/autopilot/SKILL.md`):
  - `Prepare` — reads `ROADMAP.md` and open issues, picks first item
  - `OnEvent` — watches for end of issue (claude says "ready to ship" or `/ship` was successful), returns `ActionHandoff` to start fresh on next issue
  - `NextPrompt` — formats the issue brief
  - `HandoffSummary` — structured JSON: completed_issues, files_touched, decisions, current_state
- `internal/session/compact.go` — implements `in-place`, `handoff`, `fork` strategies
- `cmd/run.go` — `jflow run autopilot` headless mode (no TUI, prints transcript to stderr)
- TUI: tool picker on `t`; tool-driven sessions show iteration counter and tool status

Acceptance: `jflow run autopilot` works through 3 issues without context bloat; comparable session in old `/autopilot` skill bloats by issue 2. Token totals captured for the comparison in `06-cost-and-bare-mode.md`.

## Phase 4 — port the rest of the jflow suite

MVP (needed to actually dogfood the harness end-to-end):
1. `next` — pick + work one item
2. `ship` — branch, commit, PR, merge, cleanup

Post-MVP (tracked, not on the critical path; the existing Claude Code skills cover these until ports land):
3. `polish` — pipeline (composes simplify/harden/test as Claude Code skills)
4. `qa` — feature testing
5. `release` — preview/production releases
6. `jflow` — onboarding interview
7. `setup` — project scaffolding
8. `issue` — github issue authoring

Each is a <200 line tool program. Most of the value is in the harness; the tools are just playbooks.

The standalone Claude Code skills (`simplify`, `harden`, `test`, `docs`, `sitrep`, `checkup`, `design`, `scrape-design`) are **not ported** — they don't need a harness around them and stay invokable as `/skill-name` inside Claude Code.

## Phase 5 — install / release

- `install.sh` updates: `go install ./cmd/jflow/` alongside existing symlink dance
- `goreleaser` (mirroring notebook) for binary releases
- `upgrade-jflow` skill becomes `jflow upgrade` (keeps the same UX)
- `VERSION` file kept; binary embeds it via `-ldflags`

---

## Out of scope (closed during spring-2026 cleanup)

- Markdown export of sessions
- `/commands` palette overlay (claude slash-command pass-through)
- Skill shims (the standalone skills stay as primary entry points; no shell-out wrapper)
- Phase 4 ports of skills outside the jflow suite (simplify, harden, test, docs, sitrep, checkup, design, scrape-design)
- Former Phase 7 items: stream-json mid-flight injection, hook-events rendering, transcript search, fork-session, agent-team integration, web companion
