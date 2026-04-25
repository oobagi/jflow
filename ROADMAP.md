# jflow — roadmap

Status of the rewrite from skill-bundle → Go CLI/TUI harness for the `claude` CLI. Detailed design lives in [`docs/`](docs/) — this file is the running checklist.

Legend: `[x]` done · `[~]` in progress · `[ ]` not started

---

## What we're building (MVP)

A three-pane TUI harness that drives `claude -p` as a subprocess:

- **Left:** workspaces (cwd-keyed) with sessions nested under each
- **Center:** chat — streaming transcript, composer, status bar
- **Right:** flat todo list with an active indicator. The model edits it via a bundled MCP server; the user edits it directly. Hitting `⏎` on a todo sets it active and the chat header shows `▸ working on: <todo>`.

The point of the harness: own context budget, compaction, and roadmap looping so the worker session stays focused. Old `jflow`-suite skills (`autopilot`, `next`, `ship`, `polish`, `qa`, `release`, `jflow`, `setup`, `issue`) get ported to deterministic Go tool programs that orchestrate `claude -p` invocations. Other skills (`simplify`, `harden`, `test`, `docs`, `sitrep`, `checkup`, `design`, `scrape-design`) stay as Claude Code skills — they don't need a harness.

---

## Phase 0 — verification (DONE)

- [x] live `claude -p --output-format stream-json` capture
- [x] CLI flag reference + cost finding ($0.20 cold start, 33k cache-creation tokens)
- [x] event-shape catalog
- [x] design docs: overview, architecture, tui, context-mgmt, build-order, open-questions

## Phase 1 — single-pane chat prototype  *(current)*

Working today:
- [x] Go module + cobra root (`jflow` binary, `jflow --version`, `jflow --debug`)
- [x] `internal/claude` driver: spawns `claude -p --resume <uuid>` per turn, decodes JSONL into typed Go events
- [x] Bubble Tea v2 single-pane TUI: transcript + composer + status bar
- [x] Streaming render: text / thinking / tool_use blocks with distinct styling
- [x] Status bar: model · permission-mode · tokens used / contextWindow (%) · running cost · rate-limit chip
- [x] Per-turn driver lifecycle (one fresh `claude -p` per user enter, same `--session-id`/`--resume <uuid>`)
- [x] Always-on session log → `~/.jflow/state/logs/<ts>-<sid8>.jsonl` + `last.jsonl` symlink
- [x] `--debug` flag adds verbose key-event meta entries
- [x] Word-wrap with prefix-aware hanging indent (`internal/ui/wrap.go`, 5 unit tests)
- [x] Stdin redirected to `/dev/null` so claude doesn't print the 3s "no stdin data" warning
- [x] Wired keybinds: `⏎` send · `⌃J` newline · `⌃X` interrupt · `⌃K` send `/compact` · `esc`/`⌃C` quit (or interrupt if claude is mid-turn)

Still missing in Phase 1:
- [ ] **Transcript scroll** (#25) — viewport for ↑/↓/PgUp/PgDn
- [ ] **`?` help overlay** (#26)
- [ ] **`⌃L` redraw / clear** (#28)
- [ ] **Up-arrow on empty composer = previous-message recall** (#29)
- [ ] **Banner / header with model + cwd + session uuid** (#30)
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
- [ ] **Meta-model loop** — cheap Sonnet calls for "is the worker stuck?" / "grade this output" decisions (see [`docs/09-meta-model.md`](docs/09-meta-model.md))

## Phase 4 — port the rest of the jflow suite

In rough order of value:
- [ ] `next` (#41) — pick + work one item
- [ ] `ship` (#42) — branch, commit, PR, merge, cleanup
- [ ] `polish` (#43) — pipeline (composes existing Claude Code skills for simplify/harden/test phases)
- [ ] `qa` (#44) — feature testing
- [ ] `release` (#45) — preview/production releases
- [ ] `jflow` (#46) — onboarding interview
- [ ] `setup` (#47) — project scaffolding
- [ ] `issue` (#48) — GitHub issue authoring

`simplify`, `harden`, `test`, `docs`, `sitrep`, `checkup`, `design`, `scrape-design` stay as Claude Code skills — they're standalone utilities and don't need a harness around them.

## Phase 5 — install / release

- [ ] `install.sh` builds and installs the Go binary (#58)
- [ ] `goreleaser.yml` + GitHub release workflow (#59)
- [ ] `jflow upgrade` subcommand (#60)
- [ ] Embed VERSION via `-ldflags` (#61)

---

## Open verification questions (#68)

Things designed around but not yet *observed* end-to-end. A `scripts/probe-stream.sh` capturing each scenario into `docs/probes/<name>.jsonl` is the way to retire these.

- [ ] `thinking_delta` field name (`thinking` vs `text`)
- [ ] `tool_use` `content_block_start` shape
- [ ] `tool_result` event shape
- [ ] `/compact` semantics in stream-json output
- [ ] Full enum of `result.terminal_reason` and `result.subtype`
- [ ] `--max-turns` exit-code + matching `result` event shape
- [ ] Permission-prompt path when `permission-mode=default` and a tool needs approval

---

## Out of scope

Closed during the spring-2026 cleanup; reopen if scope grows:

- Markdown export of sessions, `/commands` palette, claude slash-command pass-through
- Skill shims (the Claude Code skills stay as primary entry points for the non-jflow-suite commands)
- Phase 4 ports of `simplify`, `harden`, `test`, `docs`, `sitrep`, `checkup`, `design`, `scrape-design`
- All former Phase 7 items: stream-json mid-flight injection, hook-events rendering, transcript search, fork-session, agent-team integration, web companion

---

## How to read this

If something works, it's listed under Phase 1 with `[x]`. Everything else is honest about not being done. Deeper rationale lives in [`docs/07-build-order.md`](docs/07-build-order.md) and [`docs/08-open-questions.md`](docs/08-open-questions.md).
