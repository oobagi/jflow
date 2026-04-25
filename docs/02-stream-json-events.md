# Stream-JSON event reference

Captured by running `claude -p "Say hi in 3 words." --output-format stream-json --verbose --include-partial-messages` against version 2.1.119 on 2026-04-25. Every event has at minimum `type`, `session_id`, and a per-event `uuid`. Most also have `parent_tool_use_id` (null for top-level).

The driver's job is to parse JSONL line-by-line and emit typed Go events on a channel. Below is every event variant we observed, with one real example per variant.

## 1. `system / hook_started` — hook is firing

```json
{"type":"system","subtype":"hook_started","hook_id":"29f8d9f9-...","hook_name":"SessionStart:startup","hook_event":"SessionStart","uuid":"a31eef6d-...","session_id":"ea89890d-..."}
```

These can fire many times at session start (one per registered hook). **Suppressed by `--bare`.** For non-bare sessions, render collapsed in the TUI.

## 2. `system / hook_response` — hook finished

```json
{"type":"system","subtype":"hook_response","hook_id":"70a2951c-...","hook_name":"SessionStart:startup","hook_event":"SessionStart","output":"","stdout":"","stderr":"","exit_code":0,"outcome":"success","uuid":"9bfac47e-...","session_id":"ea89890d-..."}
```

Pair with the matching `hook_started` by `hook_id`. `outcome` ∈ `{success, failure, ...}`. Useful for diagnosing slow startup.

## 3. `system / init` — session boot snapshot (THE big one)

Fires once per invocation. Carries everything jflow needs to render the session header.

```json
{
  "type":"system","subtype":"init",
  "cwd":"/private/tmp",
  "session_id":"ea89890d-...",
  "tools":["Task","Bash","Edit","Read","Write","WebFetch", ... lots ...],
  "mcp_servers":[
    {"name":"plugin:context7:context7","status":"connected"},
    {"name":"playwright","status":"connected"},
    {"name":"claude.ai Gmail","status":"needs-auth"},
    ...
  ],
  "model":"claude-opus-4-7[1m]",
  "permissionMode":"default",
  "slash_commands":["clear","compact","init","review","security-review","jflow","autopilot","ship", ...],
  "apiKeySource":"none",
  "claude_code_version":"2.1.119",
  "output_style":"default",
  "agents":["Explore","Plan","Code Reviewer", ...],
  "skills":["jflow","autopilot","next","ship", ...],
  "plugins":[{"name":"context7","path":"...","source":"context7@claude-plugins-official"}, ...],
  "analytics_disabled":false,
  "uuid":"b261ad6b-...",
  "memory_paths":{"auto":"/Users/jaden/.claude/projects/-private-tmp/memory/"},
  "fast_mode_state":"off"
}
```

Fields the TUI cares about: `model`, `permissionMode`, `cwd`, `mcp_servers[].status` (color-code red if any `needs-auth` or `failed`), `slash_commands`, `claude_code_version`. Treat the rest as opaque metadata stored on the Session struct.

## 4. `system / status` — status change

```json
{"type":"system","subtype":"status","status":"requesting","uuid":"4194a4dc-...","session_id":"ea89890d-..."}
```

Observed: `requesting`. Other values likely (`idle`, `tool_executing`, etc.). Drives the spinner / status-bar word.

## 5. `stream_event / message_start` — assistant turn begins

```json
{
  "type":"stream_event",
  "event":{
    "type":"message_start",
    "message":{
      "model":"claude-opus-4-7","id":"msg_01CjyL...","type":"message","role":"assistant",
      "content":[],"stop_reason":null,"stop_sequence":null,"stop_details":null,
      "usage":{
        "input_tokens":6,
        "cache_creation_input_tokens":33048,
        "cache_read_input_tokens":0,
        "cache_creation":{"ephemeral_5m_input_tokens":0,"ephemeral_1h_input_tokens":33048},
        "output_tokens":4,
        "service_tier":"standard","inference_geo":"not_available"
      }
    }
  },
  "session_id":"ea89890d-...","parent_tool_use_id":null,"uuid":"1fd7b9bd-...","ttft_ms":1843
}
```

`ttft_ms` (time-to-first-token) is on this event. `usage` here is initial — final usage comes on `message_delta`.

## 6. `stream_event / content_block_start` — a new content block begins

```json
{"type":"stream_event","event":{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}},"session_id":"ea89890d-...","parent_tool_use_id":null,"uuid":"f7c4f3e3-..."}
```

`content_block.type` ∈ `{text, thinking, tool_use}`. **This is how the TUI knows what kind of block to start rendering.** The `index` is the block's position in the message; later deltas reference the same index.

For `tool_use` blocks, `content_block` will also carry `id` (the tool_use_id) and `name` (the tool name) — input args stream as later deltas.

## 7. `stream_event / content_block_delta` — incremental content

Three observed delta variants:

**text_delta** — visible assistant text:
```json
{"type":"stream_event","event":{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hi there"}},"session_id":"...","parent_tool_use_id":null,"uuid":"94c3b1cf-..."}
```

**thinking_delta** — extended thinking output (when the model is using thinking):
```json
{"type":"stream_event","event":{"type":"content_block_delta","index":N,"delta":{"type":"thinking_delta","thinking":"...partial thinking text..."}},"session_id":"...","uuid":"..."}
```

**input_json_delta** — partial JSON args of a `tool_use` block:
```json
{"type":"stream_event","event":{"type":"content_block_delta","index":N,"delta":{"type":"input_json_delta","partial_json":"{\"file\":\"./foo.go"}},"session_id":"...","uuid":"..."}
```

Reassemble each block's content by appending deltas keyed on `event.index`.

> The `thinking_delta` and `input_json_delta` variants were *not* exercised by the trivial test prompt; their shapes are inferred from Claude API streaming docs and confirmed by `--include-partial-messages` design. **Open question**: capture a real run that emits both before relying on the exact field names. See `08-open-questions.md`.

## 8. `stream_event / content_block_stop`

```json
{"type":"stream_event","event":{"type":"content_block_stop","index":0},"session_id":"...","uuid":"09850e81-..."}
```

Marks the end of a block. The TUI seals the block — for `tool_use` this is when the input JSON is fully assembled and we can render the args.

## 9. `stream_event / message_delta` — final usage + stop reason

```json
{
  "type":"stream_event",
  "event":{
    "type":"message_delta",
    "delta":{"stop_reason":"end_turn","stop_sequence":null,"stop_details":null},
    "usage":{
      "input_tokens":6,
      "cache_creation_input_tokens":33048,
      "cache_read_input_tokens":0,
      "output_tokens":12,
      "iterations":[{"input_tokens":6,"output_tokens":12,"cache_read_input_tokens":0,"cache_creation_input_tokens":33048,"cache_creation":{...},"type":"message"}]
    },
    "context_management":{"applied_edits":[]}
  },
  "session_id":"...","parent_tool_use_id":null,"uuid":"405a1eba-..."
}
```

`usage.iterations[]` is per-API-call accounting (one element per claude→model call within the agentic turn). `context_management.applied_edits` lists Claude Code's own context edits (auto-pruning); jflow displays this as a side-channel "claude trimmed N items" notice.

## 10. `stream_event / message_stop`

```json
{"type":"stream_event","event":{"type":"message_stop"},"session_id":"...","uuid":"b64d71d1-..."}
```

End of one assistant turn. If the harness is in single-shot `-p` mode without tool use, the next event is `assistant` then `result`.

## 11. `assistant` — full assistant message snapshot

Emitted *after* streaming completes for each assistant turn. Content array is now fully assembled.

```json
{
  "type":"assistant",
  "message":{
    "model":"claude-opus-4-7","id":"msg_01CjyL...","type":"message","role":"assistant",
    "content":[{"type":"text","text":"Hi there, friend!"}],
    "stop_reason":null,"stop_sequence":null,"stop_details":null,
    "usage":{"input_tokens":6,"cache_creation_input_tokens":33048, ...},
    "context_management":null
  },
  "parent_tool_use_id":null,
  "session_id":"...",
  "uuid":"ed7e2bc1-..."
}
```

The TUI can use this to *replace* its accumulated stream with the canonical message — useful for redacting/normalizing what was streamed. We choose to **trust the streamed deltas** for live rendering and use `assistant` only as a sanity check / persistence record.

## 12. `user` — user message echo *and* tool_result delivery

The `user` event has two distinct uses:

### 12a. tool_result (captured 2026-04-25, see `probes/tool-result.jsonl`)

When claude executes a tool internally (Bash, Read, …), the harness feeds
the output back into the conversation as a *user-role* message whose
content carries one or more `tool_result` blocks keyed by `tool_use_id`.

```json
{
  "type":"user",
  "message":{
    "role":"user",
    "content":[{
      "tool_use_id":"toolu_01UEFLUzdZhu46DH4bSGF4Zi",
      "type":"tool_result",
      "content":"hello-from-probe",
      "is_error":false
    }]
  },
  "parent_tool_use_id":null,
  "session_id":"0d5123ea-...",
  "uuid":"6cc0fceb-...",
  "timestamp":"2026-04-25T15:23:11.583Z",
  "tool_use_result":{
    "stdout":"hello-from-probe",
    "stderr":"",
    "interrupted":false,
    "isImage":false,
    "noOutputExpected":false
  }
}
```

Notes:

- `message.content[].tool_use_id` matches the `id` on the originating
  `tool_use` content block — the TUI uses this to attach the result as a
  footer on the corresponding tool block.
- `message.content[].content` is usually a plain string but can also be
  an array of `{type:"text"|"image",...}` parts (for tools that return
  binary data).
- `is_error:true` indicates the tool failed; the text in `content` is
  the error message.
- The sibling `tool_use_result` object is a side-channel with the raw
  stdout/stderr split out — useful when `is_error` truncates the text
  fed to the model.

### 12b. user echo (only with `--replay-user-messages`)

Inferred shape:
```json
{
  "type":"user",
  "message":{"role":"user","content":[{"type":"text","text":"the user's message"}]},
  "parent_tool_use_id":null,
  "session_id":"...","uuid":"..."
}
```

When jflow uses `--input-format stream-json` + `--replay-user-messages`, every user message we push to stdin gets echoed back on stdout as a `user` event. This is how the TUI confirms its send was received.

## 13. `rate_limit_event`

```json
{
  "type":"rate_limit_event",
  "rate_limit_info":{
    "status":"allowed",
    "resetsAt":1777124400,
    "rateLimitType":"five_hour",
    "overageStatus":"allowed",
    "overageResetsAt":1777593600,
    "isUsingOverage":false
  },
  "uuid":"1484eda3-...","session_id":"..."
}
```

Status-bar shows time until `resetsAt`. If `isUsingOverage:true` we display a warning chip. If `status` flips to `exceeded` or `blocked` (likely values), jflow pauses tool sessions.

## 14. `result` — terminal event

Always last. Drives the post-run summary the user sees.

```json
{
  "type":"result","subtype":"success","is_error":false,"api_error_status":null,
  "duration_ms":2728,"duration_api_ms":1937,"num_turns":1,
  "result":"Hi there, friend!",
  "stop_reason":"end_turn",
  "session_id":"...",
  "total_cost_usd":0.20688,
  "usage":{"input_tokens":6,"cache_creation_input_tokens":33048,"output_tokens":12,"server_tool_use":{"web_search_requests":0,"web_fetch_requests":0},"service_tier":"standard","cache_creation":{...},"inference_geo":"","iterations":[...],"speed":"standard"},
  "modelUsage":{
    "claude-opus-4-7[1m]":{
      "inputTokens":6,"outputTokens":12,"cacheReadInputTokens":0,"cacheCreationInputTokens":33048,
      "webSearchRequests":0,"costUSD":0.20688,
      "contextWindow":1000000,
      "maxOutputTokens":64000
    }
  },
  "permission_denials":[],
  "terminal_reason":"completed",
  "fast_mode_state":"off",
  "uuid":"401b970a-..."
}
```

**Fields the harness uses heavily**:
- `modelUsage.<model>.contextWindow` — for computing usage-pct (the trigger for compaction)
- `modelUsage.<model>.maxOutputTokens` — informational
- `total_cost_usd` — running cost on the workspace
- `num_turns` — for `--max-turns` budgeting
- `terminal_reason` — `completed` | `max_turns_reached` | `budget_exceeded` | `error` | etc. (exact enum TBD)
- `subtype` — `success` | `error` | `max_turns` etc. (TBD)
- `permission_denials[]` — items the user denied; surface in the result panel
- `stop_reason` — `end_turn` | `tool_use` | `max_tokens` | `stop_sequence` | `pause_turn` | `refusal`

If `is_error:true`, `api_error_status` carries the upstream error.

## Filtering rules for the TUI

| event | render? |
| --- | --- |
| `system/hook_*` | collapsed/footer; usually `--bare` removes them |
| `system/init` | render once into the session header (model badge, mcp list) |
| `system/status` | drive the spinner word |
| `stream_event/message_start` | start a new assistant bubble |
| `stream_event/content_block_start` | start a new block within the bubble (text/thinking/tool_use) |
| `stream_event/content_block_delta` | append to current block |
| `stream_event/content_block_stop` | seal the block |
| `stream_event/message_delta` | update token totals |
| `stream_event/message_stop` | seal the bubble |
| `assistant` | sanity check; not directly rendered |
| `user` (tool_result) | attach as footer to matching tool_use block |
| `user` (echo) | render echo of our own send |
| `rate_limit_event` | status-bar update |
| `result` | post-run summary; drives autopilot loop decisions |

## Open schema questions

1. Real shape of `thinking_delta` — confirm the field is `thinking` not `text`.
2. Real shape of `tool_use` `content_block_start` — confirm `id` and `name` are siblings to `type`.
3. ~~`tool_result` events — when claude executes a tool internally (e.g. Bash), do we get a separate `tool_result` content block in a *user* message?~~ **Confirmed 2026-04-25** — see §12a.
4. Full enum of `terminal_reason` and `result.subtype`.
5. Whether the stream emits anything when `/compact` is invoked mid-session via stream-json input.

These are tracked in `08-open-questions.md` as things to verify with a richer test (a prompt that triggers thinking + a tool call).
