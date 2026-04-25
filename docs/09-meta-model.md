# Meta-model loop

The harness can spawn a **separate cheap-model `claude -p` invocation** (Sonnet, `--bare`) to ask meta-questions *about* the worker session without polluting the worker's context.

This is the natural extension of "the harness is the foreman": the foreman gets its own brain, runs on cheaper tokens, and makes orchestration decisions the worker shouldn't have to.

## Why

The worker's context window is precious. Every harness-side question that goes through the worker — *"are you stuck?"*, *"is this output good enough?"*, *"should I keep going?"* — costs Opus tokens and adds turns the user has to scroll past. Pulling those questions into a side channel:

- keeps the worker focused on doing the work
- lets the harness make decisions on Sonnet's price/latency
- gives us a layer to evolve orchestration logic without retraining the worker prompts

## Use cases

### 1. "Is the worker stuck?"

The harness watches the worker's emitted text for hedging patterns ("I need clarification", "should I", "let me know if you'd like me to"). When matched, the harness passes the last ~500 tokens of transcript to the meta-model:

> *"The agent below appears to be hedging. Read this transcript snippet and answer: (a) is it actually stuck, (b) what's the smallest concrete next instruction that would unblock it?"*

If the meta-model says "not stuck, keep going" → the harness sends a thin nudge ("continue"). If it says "stuck, do X" → the harness either auto-replies with X or surfaces a confirm dialog to the user.

### 2. "Grade this output"

After `/ship` (or any tool program reaches a checkpoint), the harness asks the meta-model:

> *"Grade this PR diff against the issue body. Output one of: ready-to-ship, needs-more-work, wrong-direction. If not ready, give one-sentence reason."*

`autopilot` uses this to decide whether to advance to the next issue or loop on the current one before letting the user see anything.

### 3. "User interrupted mid-flight — does this need urgent handling?"

When the user types during a streaming turn, the harness asks the meta-model:

> *"The user typed `<message>` while the agent was working on `<current todo>`. Does this require interrupting the agent, or can it queue as the next message?"*

Default behavior remains "queue" — meta-model only escalates to interrupt if the user's message contradicts what the worker is doing.

## Implementation sketch

```go
// internal/meta/meta.go

type Question struct {
    Prompt   string         // the meta-prompt
    Context  string         // optional transcript snippet
    Schema   *jsonschema    // optional structured output
    Model    string         // default: "claude-sonnet-4-6"
    Timeout  time.Duration  // default: 30s
}

type Answer struct {
    Text     string
    Parsed   any            // populated if Schema set
    CostUSD  float64
    Latency  time.Duration
}

func Ask(ctx context.Context, q Question) (*Answer, error) {
    // spawn `claude -p --model <q.Model> --bare --output-format json --max-turns 1`
    // pipe `q.Prompt + "\n\n---\n" + q.Context` to stdin
    // capture single result event, return parsed
}
```

Each tool program declares its meta-calls explicitly:

```go
// internal/tool/autopilot/grade.go

func gradeShipped(ctx context.Context, diff, issueBody string) (verdict string, err error) {
    ans, err := meta.Ask(ctx, meta.Question{
        Prompt: "Grade this PR diff against the issue body...",
        Context: fmt.Sprintf("ISSUE:\n%s\n\nDIFF:\n%s", issueBody, diff),
        Schema: gradeSchema,
    })
    // ...
}
```

## Configuration

Per-tool, in `~/.jflow/config.toml`:

```toml
[tools.autopilot]
meta_enabled = true
meta_model = "claude-sonnet-4-6"
meta_max_calls_per_session = 50  # safety cap

[tools.next]
meta_enabled = false  # too short to need orchestration brain
```

## Cost tracking

Meta-calls accumulate to a separate `meta_cost_usd` field in the session record so users can see the ratio of foreman-tokens to worker-tokens. If the ratio exceeds ~10%, the meta-loop is probably being too chatty and the tool's heuristics need tightening.

## Open questions

- **How aggressive should the "stuck?" pattern matching be?** False positives waste meta-calls; false negatives leave the worker spinning. Probably start strict (only fire on very explicit hedging) and loosen with telemetry.
- **Does the worker see meta-call results?** Default: no — the harness acts on them silently (sends a nudge, advances the queue, etc.). Surface them to the user only when escalating.
- **Should the meta-model also have `todo_*` tools?** Tempting — let it claim the next item from the queue when the worker is done. But it's also a clean separation to keep meta as read-only and let the worker drive the queue.

## Why this is its own doc

Compaction (`05-context-management.md`) is about *trimming* the worker's context. The meta-model loop is about *running a parallel context* — different mechanism, different cost profile. They share the philosophy that the harness owns context-level decisions, but the implementations don't overlap.
