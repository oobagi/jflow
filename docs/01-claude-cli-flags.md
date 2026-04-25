# Claude CLI flags we depend on

Verified against `claude --version` = **2.1.119** and the official reference at https://code.claude.com/docs/en/cli-reference. Note from the docs: *"`claude --help` does not list every flag, so a flag's absence from `--help` does not mean it is unavailable."*

## Print mode (the heart of the harness)

| flag | takes | what it does | how jflow uses it |
| --- | --- | --- | --- |
| `-p`, `--print` | — | one-shot, print and exit; required for stream-json | every claude invocation jflow makes |
| `--output-format` | `text`\|`json`\|`stream-json` | output format | always `stream-json` so we get streaming events |
| `--input-format` | `text`\|`stream-json` | input format | `stream-json` when we want to push messages via stdin mid-flight |
| `--include-partial-messages` | — | emit `stream_event/content_block_delta` (text/thinking/input_json deltas) | always on — required to render thinking + streaming text |
| `--include-hook-events` | — | include hook lifecycle events in stream | off by default; on for debug |
| `--verbose` | — | required alongside `--output-format stream-json --print` (per docs) | always on |
| `--replay-user-messages` | — | echo user messages back on stdout for ack | on when using stream-json input so the TUI sees its own send confirmed |

## Sessions (the persistence model)

| flag | takes | what it does | how jflow uses it |
| --- | --- | --- | --- |
| `--session-id` | UUID | use a specific session ID (we generate it) | how each jflow session record gets a stable claude id |
| `--resume`, `-r` | id\|name | resume a session by ID or name | every turn after the first uses this with our own UUID |
| `-c`, `--continue` | — | resume most-recent in cwd | not used — too fuzzy for our state model |
| `--fork-session` | — | when resuming, allocate a *new* session id | used during compaction so the post-compact branch is its own session |
| `--no-session-persistence` | — | don't save session to disk | used for ephemeral sub-runs that shouldn't pollute `~/.claude/projects/...` |
| `--name`, `-n` | string | display name for the session | jflow sets `<workspace>/<tool>` so `/resume` picker is useful |

## Budgets and turn limits

| flag | takes | what it does | how jflow uses it |
| --- | --- | --- | --- |
| `--max-turns` | int | exits with error after N agentic turns (print mode only). Hidden from `--help` but documented. | per-tool default, e.g. `next` gets 8 turns, `polish` gets 20 |
| `--max-budget-usd` | float | hard $ cap per invocation | global default + per-tool override, treated as last-line-of-defense |

## Tool/permission scoping

| flag | takes | what it does | how jflow uses it |
| --- | --- | --- | --- |
| `--tools` | list | restrict which built-in tools are *available* (`""` = none, `"default"` = all, or names) | tightest control; e.g. `qa` gets `Read,Bash,Grep` |
| `--allowed-tools`, `--allowedTools` | list | tools that auto-run without permission prompts (with rule patterns like `Bash(git *)`) | used so headless tool runs don't deadlock |
| `--disallowed-tools`, `--disallowedTools` | list | tools removed from context | rarely used; `--tools` is preferred |
| `--permission-mode` | `default`\|`acceptEdits`\|`auto`\|`plan`\|`dontAsk`\|`bypassPermissions` | initial permission mode | tool sessions usually start in `acceptEdits` or `auto`; manual chat in `default` |
| `--dangerously-skip-permissions` | — | equivalent to `--permission-mode bypassPermissions` | only when the user has explicitly opted in via config |
| `--allow-dangerously-skip-permissions` | — | adds bypass to Shift+Tab cycle without starting in it | not used |
| `--permission-prompt-tool` | mcp tool ref | non-interactive permission handler | future: jflow can hand permission prompts to its own UI |

## System prompt control (this is how we brief the worker)

| flag | takes | what it does |
| --- | --- | --- |
| `--system-prompt` | string | replace the entire default prompt |
| `--system-prompt-file` | path | replace from file |
| `--append-system-prompt` | string | append to default |
| `--append-system-prompt-file` | path | append from file |

`--system-prompt` and `--system-prompt-file` are mutually exclusive. The append flags can combine with either replacement flag.

For tool programs, jflow uses **`--bare` + `--system-prompt-file`** so the worker boots with no Claude Code skills/hooks/CLAUDE.md noise and only the focused brief we hand it. For manual chat sessions, jflow leaves the default system prompt intact.

## Cost-and-startup control

| flag | takes | what it does | how jflow uses it |
| --- | --- | --- | --- |
| `--bare` | — | minimal mode: skip hooks, skills, plugins, MCP servers, auto memory, CLAUDE.md auto-discovery. Sets `CLAUDE_CODE_SIMPLE=1`. Tool surface = Bash, Read, Edit. | default for all tool-program sessions; saves ~33k cached tokens of system prompt per invocation |
| `--exclude-dynamic-system-prompt-sections` | — | move cwd/env/git/memory sections into first user message; improves cache reuse across machines | useful for shared CI runs |
| `--mcp-config` | path/json | load MCP servers | when a tool needs specific MCP (e.g. `release --screenshots` needs `maestro`) |
| `--strict-mcp-config` | — | only use --mcp-config, ignore others | paired with `--mcp-config` for hermetic tool runs |
| `--add-dir` | dirs... | additional directories the tools can access | when a tool needs to touch files outside cwd |
| `--model` | alias\|name | set model | default `sonnet`; `opus` for tools the user marks `effort: high` |
| `--effort` | `low`\|`medium`\|`high`\|`xhigh`\|`max` | reasoning effort level | per-tool default |
| `--fallback-model` | name | fallback when overloaded | `--print` only; jflow sets `sonnet` as fallback when running on `opus` |
| `--settings` | path/json | load extra settings | jflow ships a settings json that disables dock/notification etc. for headless |
| `--setting-sources` | csv | which sources to load (`user`,`project`,`local`) | tool sessions: `user,project` only — skip `local` |

## Subcommands jflow shells out to (rarely)

| command | when |
| --- | --- |
| `claude auth status` | startup, to confirm logged-in state |
| `claude auth login` | only on user's request; jflow never invokes silently |
| `claude mcp list` | reading MCP config for the workspace UI |

## Slash commands jflow injects (via stream-json input)

These are sent as the literal user message text. They run inside the claude session, not via flags.

| slash | when jflow sends it |
| --- | --- |
| `/compact` | when usage % crosses configured threshold and `compactionStrategy = "in-place"` |
| `/clear` | (rare) full reset; usually we prefer ending the session and starting a fresh one |

## Confirmed NOT to be used

- `-c / --continue` — too implicit; jflow always knows the session id it wants
- `--remote-control` — jflow is local
- `--remote` — same
- `--from-pr` — orthogonal feature; might wire into the workspace picker later
- `--worktree` / `-w`, `--tmux` — jflow handles its own workspace concept
- `--ide` — jflow is the IDE-equivalent here
