# Architecture

## Repo layout (in `/Users/jaden/.jflow`, alongside today's skills)

```
.jflow/
├── cmd/jflow/main.go                  binary entry, calls cmd.Execute()
├── cmd/
│   ├── root.go                        cobra root; default action launches the TUI
│   ├── run.go                         `jflow run <tool>` headless tool runner
│   ├── workspace.go                   `jflow workspace ls|add|rm`
│   ├── session.go                     `jflow session ls|export|rm`
│   └── version.go
├── internal/
│   ├── claude/
│   │   ├── driver.go                  spawns claude, owns stdin/stdout pipes, line-decodes JSONL
│   │   ├── events.go                  typed structs for every event variant in 02-stream-json-events
│   │   ├── usage.go                   running usage accumulator (input/output/cache/cost/contextPct)
│   │   └── slash.go                   helpers to emit /compact, /clear via stream-json input
│   ├── session/
│   │   ├── session.go                 in-memory session state (transcript, usage, status)
│   │   ├── transcript.go              ordered list of bubbles, each bubble = list of blocks
│   │   ├── store.go                   persist to ~/.jflow/state/sessions/<uuid>.json
│   │   └── compact.go                 compaction strategies (in-place, fork, replace)
│   ├── workspace/
│   │   ├── workspace.go               cwd-keyed registry
│   │   └── store.go                   ~/.jflow/state/workspaces.json
│   ├── tool/
│   │   ├── tool.go                    Tool interface (see below)
│   │   ├── registry.go                name → factory
│   │   ├── manual/                    no-op tool: pure manual chat
│   │   ├── autopilot/                 first ported skill
│   │   └── (next, ship, polish, qa, jflow, setup, issue, release, ...)
│   ├── config/
│   │   ├── config.go                  load/save ~/.jflow/config.toml
│   │   └── defaults.go                per-tool defaults (compactAt, maxTurns, model, effort, allowedTools)
│   ├── ui/
│   │   ├── app.go                     root bubbletea model; pane router
│   │   ├── workspace_list.go          left pane (workspaces + sessions nested)
│   │   ├── session_view.go            center pane (transcript + composer + status + banner)
│   │   ├── todopane/                  right pane — flat todo list with active indicator
│   │   ├── transcript.go              renders text/thinking/tool_use blocks
│   │   ├── composer.go                multiline input; pushes to driver stdin
│   │   ├── statusbar.go               model | tokens/ctx | $cost | rate limit | mode
│   │   ├── banner.go                  chat header — tool · model · cwd + ▸ working on: <todo>
│   │   ├── theme.go                   lipgloss styles (dark/light)
│   │   ├── keys.go                    keybindings + help generation
│   │   └── help.go
│   ├── mcp/
│   │   └── todo/                      bundled MCP server: todo_list/add/set_active/complete/...
│   ├── meta/                          cheap-Sonnet meta-loop (see docs/09-meta-model.md)
│   └── storage/
│       └── paths.go                   resolve ~/.jflow/state/, ~/.jflow/config.toml
├── fork/bubbles/                      only if we end up patching, mirrors notebook
├── skills/                            EXISTING — non-jflow-suite skills stay here as Claude Code skills
├── agents/                            EXISTING
├── hooks/                             EXISTING
├── settings/                          EXISTING
├── docs/                              this directory
├── go.mod, go.sum
├── install.sh                         updated to also `go install ./cmd/jflow/`
└── VERSION
```

## Why this layout

- `cmd/<binary>/main.go` + `internal/...` mirrors `~/Developer/notebook` exactly. Same Go conventions, same module layout, same Bubble Tea idioms.
- `internal/claude/` is the *only* place that knows how to spawn `claude`. Everything else talks to it through events on a channel + an outbound message channel. This is the seam that makes the harness testable (we can fake the driver in tests).
- `internal/tool/` has one subpackage per ported skill. Each is small (~200 lines) — the heavy lifting is in `internal/claude` and `internal/session`.
- `internal/ui/` is bubbletea; doesn't import `internal/claude` directly — it talks to `session` which proxies the driver. Keeps view code free of subprocess/IO concerns.
- `skills/` stays put. The jflow suite (`autopilot`, `next`, `ship`, `polish`, `qa`, `release`, `jflow`, `setup`, `issue`) gets ported into `internal/tool/<name>/`; the standalone skills (`simplify`, `harden`, `test`, `docs`, `sitrep`, `checkup`, `design`, `scrape-design`) stay as Claude Code skills — they don't need a harness.

## The Tool interface

```go
// internal/tool/tool.go
package tool

type Action int
const (
    ActionContinue Action = iota
    ActionCompact            // tell harness: compact now (in-place)
    ActionHandoff            // tell harness: end this session, start fresh with HandoffSummary
    ActionDone               // tool's job is done
    ActionInjectMessage      // tool wants to push a synthetic user message
)

type RunOpts struct {
    Bare              bool
    Model             string  // "sonnet" | "opus" | "" (config default)
    Effort            string  // "low" | "medium" | "high" | "xhigh" | "max" | ""
    MaxTurns          int     // 0 = unlimited
    MaxBudgetUSD      float64 // 0 = unlimited
    PermissionMode    string  // "default" | "acceptEdits" | ...
    AllowedTools      []string
    DisallowedTools   []string
    Tools             []string // restricts what's available
    AppendSystem      string   // appended to system prompt
    SystemPrompt      string   // replaces system prompt (mutually exclusive with append-from-default)
    AddDirs           []string
    MCPConfigPath     string
}

type State struct {
    WorkspaceID string
    SessionID   string  // claude session uuid
    Transcript  *session.Transcript
    Usage       claude.Usage
    Iteration   int     // tool-loop iteration counter (separate from claude turns)
    LastResult  *claude.ResultEvent
    Memo        map[string]any  // tool-private scratchpad (carried across handoffs)
}

type Tool interface {
    Name() string
    Description() string

    // Called once before the first claude invocation for this session.
    Prepare(ctx context.Context, ws Workspace) (initialPrompt string, opts RunOpts, err error)

    // Called for every event coming off the claude driver.
    OnEvent(ctx context.Context, ev claude.Event, st *State) (Action, []string /*injectMessages*/, error)

    // After ActionContinue + claude turn completes (terminal `result` event), produce next prompt.
    NextPrompt(ctx context.Context, st *State) (string, error)

    // When the harness decides to handoff (ActionHandoff or context % over threshold), produce
    // the brief that primes the next session. Returned string is appended to the next session's
    // system prompt and an opening user message can be returned too.
    HandoffSummary(ctx context.Context, st *State) (systemAppend string, openingUserMsg string, err error)
}
```

The harness loop (pseudocode):

```go
state := newState(workspace)
prompt, opts := tool.Prepare(ctx, workspace)
for {
    drv := claude.Spawn(ctx, opts)
    drv.SendUserMessage(prompt)
    for ev := range drv.Events() {
        action, inject, err := tool.OnEvent(ctx, ev, state)
        if err != nil { return err }
        for _, m := range inject { drv.SendUserMessage(m) }
        switch action {
        case ActionContinue: // keep streaming
        case ActionCompact:  drv.SendUserMessage("/compact")
        case ActionHandoff:
            drv.Close()
            sysAppend, openingMsg, _ := tool.HandoffSummary(ctx, state)
            opts.AppendSystem += "\n\n" + sysAppend
            prompt = openingMsg
            goto nextLoop
        case ActionDone:     drv.Close(); return nil
        }
        if isResult(ev) && action == ActionContinue {
            prompt, _ = tool.NextPrompt(ctx, state)
            if prompt == "" { drv.Close(); return nil }
            drv.SendUserMessage(prompt)
        }
    }
nextLoop:
}
```

The harness owns the loop; the tool just answers questions about *what to send*, *what events mean*, and *how to summarize before a handoff*. None of that is in markdown anymore.

## Driver contract

```go
// internal/claude/driver.go
package claude

type Driver struct {
    SessionID string  // uuid we generated
    cmd       *exec.Cmd
    stdin     io.WriteCloser
    events    chan Event
    done      chan struct{}
}

func Spawn(ctx context.Context, opts SpawnOpts) (*Driver, error) // builds argv, starts process, kicks off stdout reader
func (d *Driver) Events() <-chan Event                            // closed when subprocess exits
func (d *Driver) SendUserMessage(text string) error               // pushes JSONL user-message line to stdin (stream-json input mode)
func (d *Driver) Close() error                                    // closes stdin, waits, returns final result
func (d *Driver) Cost() float64                                   // running USD
func (d *Driver) ContextPct(model string) float64                 // 0..1 usage-pct based on last seen modelUsage
```

`SpawnOpts` is built from `tool.RunOpts` plus harness-level concerns (sessionID, resume bool, output-format = always stream-json, include-partial-messages = always true, verbose = always true).

## State persistence

- `~/.jflow/state/workspaces.json` — top-level registry: `[{id, name, cwd, lastUsed, sessionIDs}]`
- `~/.jflow/state/sessions/<uuid>.json` — per-session: `{tool, claudeSessionID, transcript, usage, status, createdAt, updatedAt}`
- `~/.jflow/state/sessions/<uuid>.events.jsonl` — append-only raw event log (debugging, replay)
- `~/.jflow/config.toml` — user config

Workspace ID is `sha256(cwd)[:16]` so it's stable across renames of jflow's own files.

## Concurrency model

- One `Driver` per active session. The driver's goroutine reads stdout, decodes JSONL, sends typed events.
- One `Tool` instance per session. Owns no goroutines; it's a state machine the harness drives.
- The TUI is its own goroutine; receives events via tea.Cmd that bridges from the driver's event channel.
- Workspace and session stores are guarded by a single mutex; they're tiny and reads are rare outside of the TUI mounting.
