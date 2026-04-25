# jflow CLI — design docs

These docs capture the design and verification work for rewriting jflow's skills into a Go CLI + Bubble Tea TUI that drives the `claude` CLI as a subprocess. Written 2026-04-25, before any code was scaffolded.

Read in this order:

1. [`00-overview.md`](00-overview.md) — what we're building and why
2. [`01-claude-cli-flags.md`](01-claude-cli-flags.md) — every claude CLI flag the harness depends on
3. [`02-stream-json-events.md`](02-stream-json-events.md) — every event type with real JSON examples
4. [`03-architecture.md`](03-architecture.md) — repo layout, package boundaries, the `Tool` interface
5. [`04-tui-design.md`](04-tui-design.md) — three-pane layout, keybindings, block rendering rules
6. [`05-context-management.md`](05-context-management.md) — token tracking and compaction strategies
7. [`06-cost-and-bare-mode.md`](06-cost-and-bare-mode.md) — the `$0.20/turn` finding and `--bare` rationale
8. [`07-build-order.md`](07-build-order.md) — phased plan v0 → v6
9. [`08-open-questions.md`](08-open-questions.md) — verifications still owed

## Provenance

All findings are grounded in:
- a live `claude -p --output-format stream-json --verbose --include-partial-messages` capture at version 2.1.119
- the official CLI reference at https://code.claude.com/docs/en/cli-reference
- patterns from `~/Developer/notebook` (Go + Bubble Tea v2 + Cobra)

Where we hypothesize beyond what was directly observed (e.g. exact shape of `thinking_delta`), it's flagged in [`08-open-questions.md`](08-open-questions.md) with a verify-by procedure.
