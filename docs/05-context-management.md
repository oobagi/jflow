# Context management

The single biggest reason we're rebuilding jflow as a harness: **a Claude session cannot decide for itself when to compact, or what to carry forward.** The harness can.

## What we track

From the stream:
- `usage.input_tokens` (running)
- `usage.cache_creation_input_tokens` (one-shot at session start, cached for 5m or 1h)
- `usage.cache_read_input_tokens` (running, cheap)
- `usage.output_tokens` (running)
- on terminal `result`: `modelUsage.<model>.contextWindow`, `modelUsage.<model>.maxOutputTokens`, `total_cost_usd`, `num_turns`

From the harness side:
- iteration counter (tool-loop iterations, separate from claude turns)
- handoff count (how many times we've reset the session for this tool run)
- per-handoff cost so we can detect runaway loops

**Context-pct formula** (used to decide when to compact):

```
ctxPct = (input_tokens + cache_read_input_tokens + cache_creation_input_tokens + output_tokens) / contextWindow
```

We use the most recent `modelUsage[model].contextWindow` as denominator. Until we receive a `result`, we estimate `contextWindow` from a static map (`opus-4-7[1m]` → 1_000_000, `sonnet-4-6` → 200_000, etc.) and override once a real value arrives.

> Note: claude's own automatic context management (`message_delta.context_management.applied_edits`) may already be trimming the visible context. Our pct is therefore *upper bound* on what claude is "feeling." We treat that as the right thing to be conservative on.

## Compaction strategies

Configurable per tool in `config.toml`:

```toml
[tools.autopilot]
compact_at = 0.70       # trigger when ctxPct >= 0.70
strategy = "handoff"    # "in-place" | "handoff" | "fork"
handoff_style = "structured"   # "narrative" | "structured" | "replay"

[tools.next]
compact_at = 0.85
strategy = "in-place"
```

### `in-place` — send `/compact`

Cheapest. The harness pushes the literal `/compact` user-message via stream-json input. Claude does its built-in summarization, the session continues with the same `--session-id`. We record a "compaction event" on the transcript so the user can see where it happened.

**When to use**: short-running tools where the focus is one task and we just want to free space (`next`, `ship`, `polish` on a single PR).

### `handoff` — end session, start fresh with summary

The harness:
1. Sends a final user message: a structured prompt asking the worker to produce a handoff brief in a specific shape.
2. Waits for `result`.
3. Captures the assistant's brief.
4. Closes the driver.
5. Allocates a new `--session-id` (UUID).
6. Spawns claude fresh with `--system-prompt-file` containing the brief, plus the tool's appended system prompt.
7. Updates the session record's `claudeSessionID` to the new uuid.
8. Resumes the tool loop.

**When to use**: long-running tools that complete discrete units of work (`autopilot` between issues, `qa` between feature areas).

### `fork` — `--resume <id> --fork-session`

Branches the current session into a new id. We pass `--fork-session` so the new session has a fresh id but inherits the resume context. Useful for "experiment, abandon if it goes wrong" patterns. v1 may not need this; documented for completeness.

## Handoff brief styles

### `narrative`

Free-form prose. The harness asks: *"Summarize this session into a handoff brief covering: open work, decisions made, files touched, next steps. ≤ 800 words."* The brief becomes part of the next session's system prompt.

**Pros**: easy, claude is good at this. **Cons**: variance in quality; sometimes drops critical context.

### `structured` (default for autopilot)

The harness asks claude to fill a JSON schema:
```json
{
  "completed_steps": ["..."],
  "current_step": "...",
  "pending_steps": ["..."],
  "files_touched": [{"path":"...","action":"create|edit|delete","summary":"..."}],
  "decisions": [{"q":"...","a":"...","why":"..."}],
  "open_questions": ["..."],
  "next_action": "..."
}
```

We parse with `--json-schema` (already supported in print mode). The harness then re-renders the JSON into a markdown brief for the next session, so claude sees prose but we get structured persistence in the workspace state.

### `replay` (rare)

The harness re-sends the original prompt with an "everything we've done so far" appendix. Used when the work is more like a rolling buffer than a transcript (e.g. a tool that's iterating on a single artifact).

## When compaction triggers

Three triggers, ordered by precedence:

1. **User-initiated**: `Ctrl+K` in the TUI, or `jflow run autopilot --compact-now` flag.
2. **Tool-initiated**: `Tool.OnEvent` returns `ActionCompact` or `ActionHandoff`.
3. **Threshold-initiated**: harness sees `ctxPct >= compact_at` after a `message_delta` and the tool returned `ActionContinue`.

Once a trigger fires:
- the harness drains the current turn (waits for `result`)
- runs the chosen strategy
- emits a TUI banner "compacted: 142k → 18k tokens" (the new sessions's first `system/init` gives us the new baseline)

## Budget guards (last line of defense)

In addition to compaction, every tool session has:
- `--max-turns` — claude exits when it hits this, harness sees `terminal_reason: max_turns_reached`
- `--max-budget-usd` — claude exits when this $ is spent
- jflow-side wall-clock timeout (default 30 min per session)

Hitting any of these surfaces a confirm dialog: "this session hit <budget> at <state>. resume / handoff / done?"

## What we never do

- **No mid-stream context surgery.** We don't try to delete messages from claude's session log. The fork/handoff strategies achieve the same goal cleanly.
- **No re-rendering of past tool calls in the new session.** The handoff brief mentions them; the new worker doesn't re-enact them.
- **No autonomous decision to switch models mid-session.** Models are session-scoped. If we want a different model for a phase, that's a new session.
