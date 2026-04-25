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

## Phase 2 — workspaces + sessions persistence

- `internal/workspace/{workspace.go,store.go}` and `internal/session/{session.go,store.go,transcript.go}`
- `~/.jflow/state/` directory layout
- bubbletea root model with three panes (workspaces / sessions / focus)
- `cmd/workspace.go` and `cmd/session.go` for ls/rm/export
- new session = new claude `--session-id <uuid>`; existing session = `--resume <uuid>`
- `Ctrl+E` exports current session to markdown

Acceptance: open `jflow`, see workspaces, pick one, see prior sessions with cost/ctx/timestamps, resume one, see the full prior transcript, send a new message.

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

## Phase 4 — port the rest

In rough order of value:
1. `next` — pick + work one item
2. `ship` — branch, commit, PR, merge, cleanup
3. `polish` — simplify → harden → test → ship pipeline
4. `qa` — feature testing
5. `release` — preview/production releases
6. `jflow` — onboarding interview
7. `setup` — project scaffolding
8. `issue` — github issue authoring

Each is a `<200 line tool program. Most of the value is in the harness; the tools are just playbooks.

## Phase 5 — skill shims

Each existing `skills/<name>/SKILL.md` becomes:

```markdown
---
name: <name>
description: ...
---

This skill now lives in the jflow CLI. Run from your terminal:

    jflow run <name>

Or open the TUI and start a `<name>` session:

    jflow

For backwards compat, this skill will shell out:
    !jflow run <name> "$ARGUMENTS"
```

The shell-out keeps `/jflow` inside Claude Code working without ceremony.

## Phase 6 — install / release

- `install.sh` updates: `go install ./cmd/jflow/` alongside existing symlink dance
- `goreleaser` (mirroring notebook) for binary releases
- `upgrade-jflow` skill becomes `jflow upgrade` (keeps the same UX)
- `VERSION` file kept; binary embeds it via `-ldflags`

## Phase 7 — agents

Long-tail features that aren't blocking:
- `--include-hook-events` rendering for non-bare sessions
- session export/import as `.jflow` packages (zip with transcript + metadata)
- workspace search across sessions (grep-on-transcripts)
- "duplicate session" / "branch from message N" using `--fork-session`
- agent-team integration (`--teammate-mode`, `--agents`)
- web companion (claude.ai-on-the-web link via `--remote` / `--teleport`)
