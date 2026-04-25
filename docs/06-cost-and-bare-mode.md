# Cost findings & --bare mode

## The headline cost finding

A trivial test prompt — `"Say hi in 3 words."` against `claude-opus-4-7[1m]` from `/tmp` (no project context) — cost **$0.20** on a fresh session. Why?

```
"usage": {
  "input_tokens": 6,
  "cache_creation_input_tokens": 33048,
  "cache_read_input_tokens": 0,
  "output_tokens": 12
}
"modelUsage": {
  "claude-opus-4-7[1m]": { "costUSD": 0.20688, "contextWindow": 1000000 }
}
```

**33,048 cache-creation tokens.** That's the full Claude Code system prompt + skills + plugins + agents + memory_paths + CLAUDE.md cascade being baked into the prompt cache on first call. Subsequent `--resume` calls within the 1h cache TTL hit `cache_read_input_tokens` instead and pay maybe 1/10th the cost.

For the harness this means **two problems**:

1. **Tool-program sessions don't need 33k tokens of skill/plugin context.** They have a focused job and a hand-rolled system prompt. Paying $0.20 to start each one is wasteful; if `autopilot` opens 10 fresh sessions (one per issue), that's $2 just on cache creation.

2. **Cache-aware design matters.** A session that gets compacted by `handoff` (close + spawn fresh with new system prompt) blows the cache and pays the $0.20 again. `in-place` `/compact` keeps cache. This is a real input to the strategy choice.

## The fix: `--bare` for tool sessions

From the docs:

> `--bare` — Minimal mode: skip auto-discovery of hooks, skills, plugins, MCP servers, auto memory, and CLAUDE.md so scripted calls start faster. Claude has access to Bash, file read, and file edit tools. Sets `CLAUDE_CODE_SIMPLE`.

Concretely, `--bare`:
- skips `SessionStart` hooks (the four we saw firing in our test go away)
- skips loading skills, agents, plugins
- skips auto-memory
- skips CLAUDE.md auto-discovery
- restricts default tool surface to Bash + Read + Edit (we can still grant more via `--tools`)
- still resolves explicit context: `--system-prompt[-file]`, `--append-system-prompt[-file]`, `--add-dir`, `--mcp-config`, `--settings`, `--agents`, `--plugin-dir`

Expected savings: cache_creation_input_tokens drops from ~33k to **a few hundred** (just the bare-mode system prompt + whatever we append).

## Per-session-type policy

| session kind | `--bare`? | rationale |
| --- | --- | --- |
| manual chat (TUI, user typing) | **no** | user expects skills/agents/MCP/CLAUDE.md to work |
| tool: `autopilot` | **yes** | focused playbook; we own the system prompt |
| tool: `next` | **yes** | same |
| tool: `ship` | **yes** + `--allowed-tools "Bash(git *) Read"` | git-only tool surface |
| tool: `polish` | **yes** | same |
| tool: `qa` | **yes** + `--mcp-config maestro` (when `--screenshots`) | bring in only what's needed |
| tool: `setup` | **no** | needs full Claude Code env to scaffold properly |
| tool: `release` | **yes** | scripted GH workflow trigger |

## Other cost levers

- **Reuse session IDs across tool iterations.** The harness uses `--session-id <uuid>` once and `--resume <uuid>` thereafter. Cache stays warm for an hour.
- **Use `--exclude-dynamic-system-prompt-sections`** when running on a CI runner so per-machine sections don't bust the cache for other runners on the same task.
- **Choose `in-place` compaction over `handoff`** when ctx pct just crossed the line. The cost of `/compact` (one extra small turn) is way less than re-creating cache.
- **Set `--max-budget-usd`** generously per session as a runaway-loop tripwire (e.g. $5 for autopilot per issue). Hitting it surfaces a TUI prompt rather than a silent burn.
- **Track cost per tool over time** in `~/.jflow/state/usage.jsonl`. After a few weeks we'll know which tools are expensive and which aren't.

## What we'll measure post-v0

- Average $ per autopilot issue under the old skill vs the new tool-program. Target: 3–5× reduction.
- Average tokens carried across handoffs under `narrative` vs `structured`. Target: structured handoffs are smaller and more reliable.
- Cache hit rate (cache_read / cache_creation) — this is a real KPI for the harness. Target: >0.7 for tool sessions after warm-up.
