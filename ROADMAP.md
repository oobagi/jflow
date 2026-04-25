# jflow — roadmap

Status of the rewrite from skill-bundle → Go CLI/TUI harness for the `claude` CLI. Detailed design lives in [`docs/`](docs/) — this file is the running checklist.

Legend: `[x]` done · `[~]` in progress · `[ ]` not started

---

## What we're building (MVP)

A three-pane TUI harness that drives `claude -p` as a subprocess:

- **Left:** workspaces (cwd-keyed) with sessions nested under each
- **Center:** chat — streaming transcript, composer, status bar
- **Right:** flat todo list with an active indicator. The model edits it via a bundled MCP server; the user edits it directly. Hitting `⏎` on a todo sets it active and the chat header shows `▸ working on: <todo>`.

The point of the harness: own context budget, compaction, and roadmap looping so the worker session stays focused. The old jflow-suite skills (`autopilot`, `next`, `ship`, `polish`, `qa`, `release`, `jflow`, `setup`, `issue`) are transitional — each gets ported to a Go tool program and then **deleted from the skill bundle**. The harness is the product; the skills aren't a long-term offering. Standalone utilities (`simplify`, `harden`, `test`, `docs`, `sitrep`, `checkup`, `design`, `scrape-design`) stay as Claude Code skills — they don't need a harness.

---

## Phase 0 — verification (DONE)

- [x] live `claude -p --output-format stream-json` capture
- [x] CLI flag reference + cost finding ($0.20 cold start, 33k cache-creation tokens)
- [x] event-shape catalog
- [x] design docs: overview, architecture, tui, context-mgmt, build-order, open-questions

## Phase 1 — single-pane chat prototype  *(current)*

Working today:
- [x] Go module + cobra root (`jflow` binary, `jflow --version`, `jflow --debug`)
- [x] `internal/claude` driver: spawns `claude -p --resume <uuid>` per turn, decodes JSONL into typed Go events; stderr captured so it doesn't bleed into the TUI
- [x] Bubble Tea v2 three-pane TUI shell: workspaces stub (left) · chat (center) · session info (right). The right pane currently shows model / mode / context % / cost / rate-limit; Phase 2 turns it into the todo list
- [x] Streaming render: text / thinking / tool_use blocks with distinct styling, plus a dim "worked for 2.3s" trailer under each completed turn
- [x] Per-turn driver lifecycle (one fresh `claude -p` per user enter, same `--session-id`/`--resume <uuid>`); spawn is async and cancellable via ⌃C/esc during the init window
- [x] Always-on session log → `~/.jflow/state/logs/<ts>-<sid8>.jsonl` + `last.jsonl` symlink
- [x] `--debug` flag adds verbose key-event meta entries
- [x] Word-wrap with prefix-aware hanging indent (`internal/ui/wrap.go`, 5 unit tests; ANSI-aware via `lipgloss.Width`)
- [x] Stdin redirected to `/dev/null` so claude doesn't print the 3s "no stdin data" warning
- [x] **Transcript scroll** (#25) — viewport with mouse wheel scrolling, bottom-anchored content
- [x] **`?` help overlay** (#26) — bottom-sheet help panel listing every wired keybind
- [x] Composer rule labelled with current worktree (cwd) and git branch
- [x] Wired keybinds: `⏎` send · `⇧⏎`/`⌃J` newline · `⌃C` interrupt (no-op when idle, never quits) · `⌃K` `/compact` · `esc` quit (or interrupt mid-turn) · `?` help

Still missing in Phase 1:
- [ ] **Up-arrow on empty composer = previous-message recall** (#29)
- [ ] **Tool result rendering** (#31) — render `tool_result` blocks paired with their `tool_use`

## Phase 2 — workspaces + sessions + three-pane shell + todo pane

- [ ] `internal/workspace/` — cwd-keyed registry, `~/.jflow/state/workspaces.json` (#32)
- [ ] `internal/session/` — per-session state (transcript, usage, status, todos), `~/.jflow/state/sessions/<uuid>.json` (#33)
- [ ] Three-pane TUI shell — workspaces · chat · todo (#34)
- [ ] **Right-pane todo list** — flat list, active indicator, user keybinds for add/edit/done/send-to-chat (#35)
- [ ] **Bundled MCP server** exposing `todo_*` tools to the worker (#70) — model and user share one source of truth
- [ ] `jflow workspace ls|add|rm`
- [ ] `jflow session ls|archive|rm`
- [ ] Resume any prior session by id or name (uses claude's `--resume`)

## Phase 3 — first tool program: `autopilot`

- [ ] `internal/tool/` — `Tool` interface (Prepare / NextPrompt / OnEvent / HandoffSummary) (#37)
- [ ] `internal/session/compact.go` — `in-place` (`/compact`), `handoff`, `fork` strategies (#38)
- [ ] `cmd/run.go` — `jflow run autopilot` headless mode (#39)
- [ ] `internal/tool/autopilot/` — port of `skills/autopilot/SKILL.md` (#40)
- [ ] Per-tool config in `~/.jflow/config.toml` (`compact_at`, `max_turns`, `model`, `allowed_tools`)

## Phase 4 — port the rest of the jflow suite

MVP (needed to dogfood the harness):
- [ ] `next` (#41) — pick + work one item
- [ ] `ship` (#42) — branch, commit, PR, merge, cleanup
- [ ] As each port lands, delete the corresponding `skills/<name>/`

Post-MVP — tracked under #43 (single tracker):
- [ ] `polish`, `qa`, `release`, `jflow`, `setup`, `issue`

## Phase 5 — install / release

- [ ] `install.sh` builds and installs the Go binary (#58)
- [ ] `goreleaser.yml` + GitHub release workflow (#59)
- [ ] `jflow upgrade` subcommand — CLI-only, deletes `skills/upgrade-jflow/` (#60)
- [ ] Embed VERSION via `-ldflags` for `jflow --version` (#61)

---

## Open verification questions

Theoretical until the relevant feature lands. Verify each as it comes up rather than upfront — capture into `docs/probes/<name>.jsonl` and update [`docs/02-stream-json-events.md`](docs/02-stream-json-events.md). Tracked inline in [`docs/08-open-questions.md`](docs/08-open-questions.md).

- [ ] `thinking_delta` field name (`thinking` vs `text`)
- [ ] `tool_use` `content_block_start` shape
- [ ] `tool_result` event shape (covered by #31)
- [ ] `/compact` semantics in stream-json output
- [ ] Full enum of `result.terminal_reason` and `result.subtype`
- [ ] `--max-turns` exit-code + matching `result` event shape
- [ ] Permission-prompt path when `permission-mode=default` and a tool needs approval

---

## Out of scope

Closed during the 2026 cleanups; reopen if scope grows:

- Markdown export of sessions, `/commands` palette, claude slash-command pass-through
- Skill shims (the jflow-suite skills get deleted as their CLI ports land — no shell-out wrappers)
- Ports of `simplify`, `harden`, `test`, `docs`, `sitrep`, `checkup`, `design`, `scrape-design` — these stay as Claude Code skills
- All former Phase 7 items: stream-json mid-flight injection, hook-events rendering, transcript search, fork-session, agent-team integration, web companion
- `⌃L` clear-screen redraw (#28) — terminal-emulator muscle memory; the transcript viewport already scrolls and there's no shell scrollback to clear
- Dedicated session-header banner widget (#30) — model/cwd/session/branch already surface in the right pane and composer rule; active todo lives on the status bar instead
- Standalone verification epic (#68) — verify each open question as the feature lands instead of upfront
- Meta-model loop (#71) — designed in `docs/09-meta-model.md` but unscheduled; reopen once autopilot is dogfoodable and meta-loop value can be measured

---

## How to read this

If something works, it's listed under Phase 1 with `[x]`. Everything else is honest about not being done. Deeper rationale lives in [`docs/07-build-order.md`](docs/07-build-order.md) and [`docs/08-open-questions.md`](docs/08-open-questions.md).
