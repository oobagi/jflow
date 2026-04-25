# jflow — roadmap

Status of the rewrite from skill-bundle → Go CLI/TUI harness for the `claude` CLI. The detailed design and verification work lives in [`docs/`](docs/) — this file is the running checklist.

Legend: `[x]` done · `[~]` in progress · `[ ]` not started

---

## Phase 0 — verification

- [x] live `claude -p --output-format stream-json` capture
- [x] CLI flag reference + cost finding ($0.20 cold start, 33k cache-creation tokens)
- [x] event-shape catalog
- [x] design docs: overview, architecture, tui, context-mgmt, build-order, open-questions

## Phase 1 — single-pane chat prototype  *(current)*

Working today:
- [x] Go module + cobra root (`jflow` binary, `jflow --version`, `jflow --debug`)
- [x] `internal/claude` driver: spawns `claude -p --resume <uuid>` per turn, decodes JSONL into typed Go events (envelope, system, stream_event, assistant, user, rate_limit, result, hook_*, parse error, exit)
- [x] Bubble Tea v2 single-pane TUI: transcript + composer + status bar
- [x] Streaming render: text / thinking / tool_use blocks with distinct styling
- [x] Status bar: model · permission-mode · tokens used / contextWindow (%) · running cost · rate-limit chip
- [x] Per-turn driver lifecycle (one fresh `claude -p` per user enter, same `--session-id`/`--resume <uuid>`)
- [x] Always-on session log → `~/.jflow/state/logs/<ts>-<sid8>.jsonl` + `last.jsonl` symlink (raw JSONL stream + `_jflow` meta entries)
- [x] `--debug` flag adds verbose key-event meta entries
- [x] Word-wrap with prefix-aware hanging indent (`internal/ui/wrap.go`, 5 unit tests)
- [x] Stdin redirected to `/dev/null` so claude doesn't print the 3s "no stdin data" warning
- [x] Wired keybinds: `⏎` send · `⌃J` newline · `⌃X` interrupt · `⌃K` send `/compact` · `esc`/`⌃C` quit (or interrupt if claude is mid-turn)

Still missing in Phase 1:
- [ ] **Transcript scroll** — arrow keys currently move the textarea cursor; no way to look back. Wrap the transcript output in `charm.land/bubbles/v2/viewport` and route ↑/↓/PgUp/PgDn there.
- [ ] **`?` help overlay** — listed in the design but the binding isn't wired
- [ ] **`⌃E` export current session as markdown**
- [ ] **`⌃L` redraw / clear**
- [ ] **Up-arrow on empty composer = previous-message recall**
- [ ] **Banner / header with model + cwd + session uuid** (currently only the dim system-note shows it)
- [ ] **Better visibility into hook events** — they're written to the log but suppressed in the TUI; non-`--bare` sessions emit four `SessionStart` hooks per turn
- [ ] **Tool result rendering** — when claude executes a tool internally (Bash/Read/Edit), we render the `tool_use` call but don't yet render the matching `tool_result` block

## Phase 2 — workspaces + sessions persistence

- [ ] `internal/workspace/` — cwd-keyed registry, `~/.jflow/state/workspaces.json`
- [ ] `internal/session/` — per-session state (transcript, usage, status), `~/.jflow/state/sessions/<uuid>.json`
- [ ] Three-pane TUI: workspaces / sessions / focus
- [ ] `jflow workspace ls|add|rm`
- [ ] `jflow session ls|export|rm`
- [ ] Resume any prior session by id or name (uses claude's `--resume`)
- [ ] Session list shows cost / tokens / last-active per row

## Phase 3 — first tool program: `autopilot`

- [ ] `internal/tool/` — `Tool` interface (Prepare / NextPrompt / OnEvent / HandoffSummary)
- [ ] `internal/tool/manual/` — explicit no-op tool for manual chat
- [ ] `internal/tool/autopilot/` — port of `skills/autopilot/SKILL.md`
- [ ] `internal/session/compact.go` — `in-place` (`/compact`), `handoff` (close + spawn fresh with brief), `fork` (`--fork-session`)
- [ ] `cmd/run.go` — `jflow run autopilot` headless mode
- [ ] TUI tool picker on `t`
- [ ] Per-tool config in `~/.jflow/config.toml` (compact_at, max_turns, model, allowed_tools)

## Phase 4 — port the rest of the skills

In rough order of value:
- [ ] `next` — pick + work one item
- [ ] `ship` — branch, commit, PR, merge, cleanup
- [ ] `polish` — simplify → harden → test → ship pipeline
- [ ] `qa` — feature testing
- [ ] `release` — preview/production releases
- [ ] `jflow` (the onboarding interview)
- [ ] `setup` — project scaffolding
- [ ] `issue` — GitHub issue authoring
- [ ] `simplify`, `harden`, `test`, `docs`, `checkup`, `sitrep`, `design`, `scrape-design`

## Phase 5 — skill shims

- [ ] Each `skills/<name>/SKILL.md` becomes a thin shell-out to `jflow run <name>` so `/jflow` inside Claude Code keeps working during transition

## Phase 6 — install / release

- [ ] `install.sh` updated to also `go install ./cmd/jflow/`
- [ ] `goreleaser.yml` (mirroring notebook) for binary releases
- [ ] `upgrade-jflow` skill becomes `jflow upgrade`
- [ ] Embed VERSION via `-ldflags`

## Phase 7 — agentic features

- [ ] `--input-format stream-json` mid-flight user injection (open question #8 in `docs/08-open-questions.md`)
- [ ] `--include-hook-events` rendering for non-bare sessions
- [ ] Workspace-wide search across transcripts
- [ ] "Branch from message N" via `--fork-session`
- [ ] Agent-team integration (`--teammate-mode`, `--agents`)
- [ ] Web companion (`--remote` / `--teleport` to claude.ai web)

---

## Open verification questions (from `docs/08-open-questions.md`)

Things I've designed around but haven't yet *observed* end-to-end:

- [ ] Exact JSONL shape for `--input-format=stream-json` user messages
- [ ] `thinking_delta` field name (`thinking` vs `text`)
- [ ] `tool_use` `content_block_start` shape (id/name siblings to type)
- [ ] `tool_result` event shape
- [ ] `/compact` semantics in stream-json output (is there a discrete event?)
- [ ] Full enum of `result.terminal_reason` and `result.subtype`
- [ ] `--max-turns` exit-code + matching `result` event shape
- [ ] `--fork-session` behavior in stream
- [ ] Permission-prompt path when `permission-mode=default` and a tool needs approval

A `scripts/probe-stream.sh` that captures each scenario into `docs/probes/<name>.jsonl` is the natural way to retire these.

---

## How to read this

If something works, it's listed under Phase 1 with `[x]`. Everything else is honest about not being done. The ROADMAP at the root is the source of truth; deeper rationale lives in [`docs/07-build-order.md`](docs/07-build-order.md) and [`docs/08-open-questions.md`](docs/08-open-questions.md).
