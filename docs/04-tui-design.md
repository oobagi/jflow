# TUI design

Reference points: opencode (`/Applications` install), codex-app, Claude Code itself, our own `~/Developer/notebook`. Goal: feels like a real harness, not a remote control.

## Three-pane layout

```
┌──────────────────────────┬───────────────────────────────────────────────────────────────┐
│ workspaces (12)          │ sessions in: ~/code/myapp                                      │
│ ─────────────────────────│  ──────────────────────────────────────────────────────────── │
│ ▸ ~/code/myapp     [3]   │  ● autopilot                       2m ago    62% ctx  $0.41   │
│   ~/code/oss-tool        │  ◌ manual chat                     1h ago    18% ctx  $0.06   │
│   ~/.jflow               │  ✓ ship #21                        yest.    completed         │
│   /tmp/scratch    [1]    │                                                                │
│                          │                                                                │
│   + new workspace        │  + new session   ┃ + start tool                                │
└──────────────────────────┴───────────────────────────────────────────────────────────────┘
```

Pressing a workspace or session opens the right pane (the active session view). The three-pane layout is the *home* view; the active session is the *focus* view (left/middle collapse to slim sidebars when the user is in flow).

## Active session view (the focus mode)

```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│ autopilot · ~/code/myapp · sonnet · acceptEdits          tokens 41k/200k (20%) · $0.41  │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                          │
│  you ▸ work through the next 3 issues                                                    │
│                                                                                          │
│  ✱ thinking ▾                                                                            │
│  │ Picking issue #41 first because it's labeled "good first issue" and has no            │
│  │ blockers. Plan: read the failing test, fix the validation logic, run tests.           │
│                                                                                          │
│  claude ▸ Starting on issue #41. Let me read the failing test first.                     │
│                                                                                          │
│  ⚙ Read(file: "src/validate.test.ts") ▾                                                  │
│  │  → 124 lines · 0.2s                                                                   │
│                                                                                          │
│  claude ▸ Found it. The regex in `isEmail` rejects '+' in local-parts. Patching.         │
│                                                                                          │
│  ⚙ Edit(file: "src/validate.ts") ▾                                                       │
│  │  - /^[A-Za-z0-9._-]+@[A-Za-z0-9.-]+$/                                                 │
│  │  + /^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+$/                                               │
│                                                                                          │
│  ⚙ Bash("npm test") ▾                                                                    │
│  │  PASS  src/validate.test.ts                                                           │
│  │  ✓ 124 passing                                                                        │
│                                                                                          │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│ > _                                                                                     │
│ Esc back · ⏎ send · ^J newline · ^C interrupt · ^K compact · t tools                     │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```

### Block rendering

| block type | style | interactive |
| --- | --- | --- |
| `text` (assistant) | regular fg, `claude ▸` prefix on first line | no |
| `text` (user) | accent color, `you ▸` prefix | no |
| `thinking` | dimmed italic, collapsed by default with `✱ thinking ▾` header showing first line | tab to expand/collapse |
| `tool_use` | inline `⚙ ToolName(args)` block, args shown collapsed; `▾` toggle | tab to expand to see full args + result |
| `tool_result` (when claude runs tool internally) | merged into the tool_use block as a "→ N lines · Xs" footer | inline expand shows full output (chroma-highlighted if known mime) |
| `error` | red border, `✗` prefix | no |
| status events (init, status, rate_limit) | not rendered as blocks; status bar updates only | n/a |

### Status bar fields (left to right)

```
<tool> · <workspace-name> · <model> · <permission-mode>      <tokens used>/<contextWindow> (<%>) · $<cost>     [<rate-limit-chip>]
```

`<rate-limit-chip>` only appears when overage active or window approaching exhaustion. Color: green <50%, yellow 50–80%, red >80%.

## Composer

Multiline `textarea` (Bubbles v2). Enter = send. Ctrl+J = newline. Ctrl+C = interrupt current claude turn (sends SIGINT to subprocess, the harness catches the resulting partial result and offers a "resume / discard" prompt). Ctrl+K = compact now (sends `/compact` as a user message). Up arrow on empty = recall previous send.

A user can type during a streaming assistant turn — their message queues and is delivered as the next user message after the current turn completes. When `--input-format stream-json` is in use, the queued message is sent immediately on stdin as a new user-message JSONL line; the harness lets claude finish its current turn and then process it.

## Keybindings (global)

| key | action |
| --- | --- |
| `?` | toggle help |
| `q` | back / quit (context-sensitive: from focus view → home; from home → confirm quit) |
| `n` | new manual chat session in current workspace |
| `t` | start a tool (opens tool picker) |
| `w` | jump to workspaces pane |
| `s` | jump to sessions pane |
| `/` | filter current pane |
| `j/k` `↓/↑` | navigate list |
| `⏎` | open / send |
| `^C` | interrupt current claude turn |
| `^K` | compact now |
| `g/G` | top / bottom of transcript |

## Theme

Two themes shipped: `dark` (default) and `light`. Borrow the lipgloss palette structure from `~/Developer/notebook/internal/ui/theme.go`. Accent color choices:

- assistant text: default fg
- user text: cyan
- thinking: dim gray italic
- tool_use header: yellow
- tool_use content: dim
- error: red
- status bar bg: subtle (1-shade-from-bg)
- status bar accent: green / yellow / red as functions of % and rate-limit

## Bubble Tea v2 specifics (mirrors notebook)

- Module path: `charm.land/bubbletea/v2` (NOT `github.com/charmbracelet/bubbletea`)
- Components: `charm.land/bubbles/v2`, with `replace charm.land/bubbles/v2 => ./fork/bubbles` ready in case we patch (notebook does)
- Lip Gloss: `charm.land/lipgloss/v2`
- One root `tea.Model` (`ui.app.Model`) that owns three sub-models (workspaces, sessions, focus). Routing: `app.Update` dispatches to whichever pane is focused.
- Driver events become `tea.Cmd` via a small adapter that selects on the driver's event channel.

## Single-pane v0 (the prototype)

For v0 we skip workspaces and sessions entirely. The TUI starts directly into the focus view, with one ephemeral manual-chat session. This proves out:
- driver lifecycle (spawn / events / interrupt / close)
- transcript rendering (text + thinking + tool_use)
- composer + stream-json input
- status bar (tokens, cost)

Once that works end-to-end, we add the three-pane chrome around it.

## Right-pane (todo) — Phase 2

**One opinionated thing**: a flat todo list with an active indicator. The model and the user both edit this list. It's the harness's "ticket queue" UX — user queues work for the agent, agent reports progress back via MCP.

```
┌─ todo ──────────────────────────────────────┐
│ [x]  Read failing test                      │
│ ▸    Patch isEmail regex for + in local     │
│ [ ]  Run npm test                           │
│ [ ]  Open PR with changelog entry           │
│ [ ]  Reply on issue #41                     │
│                                             │
│ a add · e edit · ⏎ activate · space done    │
└─────────────────────────────────────────────┘
```

Rendering:
- `[ ]` pending · `▸` active (one at a time) · `[x]` done
- The active item title also appears in the chat header as `▸ working on: <title>` so the user always knows what the worker is on
- Height is pinned (notebook-cli `picker.go` pattern) so adding/removing items never reflows the chat

Keybinds when the right pane is focused:

| key | action |
| --- | --- |
| `j/k` `↓/↑` | move cursor |
| `⏎` | set highlighted item active (clears any prior active) |
| `space` | toggle done/pending |
| `a` | add new todo (inline input) |
| `e` | edit highlighted todo |
| `d` | delete highlighted todo |
| `s` | send highlighted todo to chat as the next user message |

State: per-session JSON in `~/.jflow/state/sessions/<uuid>.json` under a `todos[]` array. The right-pane Bubble Tea sub-model (`internal/ui/todopane/`) and the bundled MCP server (`internal/mcp/todo/`) read and write the same file — single source of truth.

### Model side: bundled MCP server

When `jflow` spawns `claude -p` it auto-registers a small MCP server exposing:

- `todo_list()` — array of `{id, title, status: "pending"|"active"|"done"}`
- `todo_add(title)` — returns id
- `todo_set_active(id)` — clears any prior active
- `todo_complete(id)`
- `todo_delete(id)`
- `todo_update(id, title)`

A short system-prompt addendum tells the worker: *"You are running under jflow. Use `todo_*` to plan and track your work. The user can see and edit this list in real time."*

`todo_*` calls render as a thin `✓ todo_add("…")` line in the transcript — bookkeeping, not work.

### What lives elsewhere

The togglable-panels grab-bag from earlier drafts (files / tool-status / budget / session) is cut. That information surfaces in the status bar (tokens, cost, model, mode) or via dedicated keybinds rather than competing with the todo list.

## Slash commands

Slash command pass-through (the `/commands` palette concept) was cut from MVP. Users can still type `/compact` directly into the composer if they want claude's built-in command, but the harness owns its own controls via dedicated keybinds (`⌃K` compact now, `⌃X` interrupt) rather than a palette overlay.
