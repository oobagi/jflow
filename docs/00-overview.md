# jflow CLI — Overview

Captured: 2026-04-25 (during planning conversation, before any code is written).

## What this is

`jflow` is being rewritten from a bundle of Claude Code skills (megaprompts) into a **Go CLI + Bubble Tea TUI that drives the `claude` CLI as a subprocess**. Every old skill (`/jflow`, `/autopilot`, `/setup`, `/issue`, `/next`, `/ship`, `/polish`, `/qa`, `/release`, etc.) becomes a *tool program* implemented in Go that orchestrates one or more `claude -p` invocations and owns the things a single Claude session cannot own about itself: context budget, compaction, hand-off between phases, retry policy, tool whitelisting.

The mental model: **`claude` is the worker, `jflow` is the foreman.**

## Why this is being done

Today's `/jflow` and `/autopilot` skills degrade fast inside one Claude session because the same context window has to:
1. hold the orchestration playbook (the skill markdown — 150–200 lines)
2. plan the work
3. execute every step
4. carry results between steps without compacting

There is no way for a skill to compact itself, allocate fresh sub-sessions, or enforce a budget. By the time `/autopilot` is on its 3rd issue, the context is bloated with tool-call transcripts and the model starts cutting corners.

Putting the orchestration in code means:
- the Go program decides *when* to compact and *what* to carry forward
- each phase runs in its own `claude -p` subprocess with a focused prompt and `--bare` system prompt
- the user sees real-time tool calls, thinking, and text streaming in a TUI like opencode/codex-app
- the user can interject at any point via stream-json input (no need to kill the subprocess)

## What success looks like

A user runs `jflow` (or `jflow run autopilot` for headless), gets a three-pane TUI:
- **Workspaces** (left) — cwd-keyed groupings with sessions nested under each
- **Chat** (center) — streaming transcript with thinking blocks, inline tool calls, and a multiline composer
- **Todo** (right) — flat list with an active indicator. The model edits it via a bundled MCP server; the user edits it directly. The active item shows in the chat header as `▸ working on: <title>`.

The user can:
- type freely into the composer mid-flight (sent as the next user message)
- see token / context-window % live in the status bar
- press `⌃K` to force a compaction now
- press `t` to start a new tool-driven session in the current workspace
- focus the right pane and queue todos for the agent (or send a highlighted todo to the chat with `s`)
- watch `jflow autopilot` chew through 10 GitHub issues without context bloat because the harness opens a fresh session per issue with a structured handoff

## Where this lives

This new harness lives in the **same `~/.jflow` repo** as today's skills. The jflow suite (`autopilot`, `next`, `ship`, `polish`, `qa`, `release`, `jflow`, `setup`, `issue`) gets ported into Go tool programs that the binary runs. The other skills (`simplify`, `harden`, `test`, `docs`, `sitrep`, `checkup`, `design`, `scrape-design`) stay as Claude Code skills — they're standalone utilities and don't need a harness around them.

## Tech stack (locked, mirrors `~/Developer/notebook`)

- Go 1.25+
- `charm.land/bubbletea/v2` — TUI runtime (note: v2 module path, NOT `github.com/charmbracelet/...`)
- `charm.land/bubbles/v2` — components, with a local `replace` fork at `fork/bubbles/` if we need to patch
- `charm.land/lipgloss/v2` — styling
- `github.com/spf13/cobra` — subcommands
- `github.com/BurntSushi/toml` — config (`~/.jflow/config.toml`)
- `github.com/alecthomas/chroma/v2` — syntax highlighting in transcript

## See also

- `01-claude-cli-flags.md` — every CLI flag we depend on
- `02-stream-json-events.md` — every event shape we decode
- `03-architecture.md` — repo layout, package boundaries, the Tool interface
- `04-tui-design.md` — three-pane layout, keybindings, rendering rules
- `05-context-management.md` — token tracking, compaction strategies
- `06-cost-and-bare-mode.md` — why `--bare` matters for tool sessions
- `07-build-order.md` — phased plan, what v0/v1/v2 each contain
- `08-open-questions.md` — verifications still owed
- `09-meta-model.md` — cheap-Sonnet meta-loop for harness-side orchestration decisions
