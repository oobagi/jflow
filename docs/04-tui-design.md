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
| `^L` | clear screen redraw |
| `^E` | export current session as markdown |
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

## Right-pane (context) — Phase 2

The right pane is for *side-information* that supports the active session without cluttering the transcript. Codex-app shows file tree + token gauge in this slot; ours is more agent-focused. A stack of togglable panels (cycle with `Tab` while the right pane is focused):

- **Todo** (default) — checklist for the active session. User-editable. Optionally synced from a tool program's state (e.g., `autopilot` writes its plan here so the user can see what's queued).
- **Files** — files claude has read / edited / created in this session, with action + last-touched timestamp. Derived from observed `tool_use` events.
- **Tool status** — when a tool program is running: name · iteration counter · current step · max-turns budget remaining · last decision.
- **Budget** — input / output / cache-creation / cache-read tokens for the session, $ cost, rate-limit window with reset countdown.
- **Session** — model · permission-mode · cwd · session uuid · log path · started_at.

Each panel is a small Bubble Tea sub-model under `internal/ui/contextpane/`. Reference notebook-cli at `~/Developer/notebook/internal/ui/picker.go` for the layout-pinning pattern (height stays constant as content changes — important so panel toggles don't reflow the screen).

## `/commands` palette — Phase 2

When the composer's first character is `/`, a Picker overlay opens anchored above the composer. Filter by typing more characters; ⏎ runs the selected command; esc closes; backspace at empty closes too.

Two command categories rendered with a leading icon:

1. `▸ jflow commands` — invoke harness features:
   - `/tool <name>` — start a tool-driven session in the current workspace
   - `/workspace open <path>` / `/workspace new` / `/workspace ls`
   - `/session export` — write current transcript to markdown
   - `/session new` — manual chat session in current workspace
   - `/session resume <name>` — pick a prior session
   - `/compact-now` — force compaction (in-place by default)
   - `/handoff` — force a handoff (close current claude session, spawn fresh with brief)
   - `/panel <name>` — switch right-pane panel (todo · files · tool · budget · session)
   - `/help` — open the help overlay
   - `/quit`
2. `◇ claude commands` — pass-through. Anything starting with `/` matching a known claude slash command (from the `system/init.slash_commands[]` list jflow already receives) is sent verbatim to claude as a user message: `/compact`, `/clear`, `/init`, `/review`, `/security-review`, etc.

Implementation: copy notebook-cli's `~/Developer/notebook/internal/ui/picker.go` Picker pattern verbatim (it already handles fuzzy filter, scroll indicators, height pinning, word-delete, and esc-to-close edge cases). Wrap it in a thin adapter that knows about both command sources and dispatches to `App.Update` as a typed `commandRunMsg`.
