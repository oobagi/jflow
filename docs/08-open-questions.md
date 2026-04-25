# Open questions to verify before / during build

Things that are *probably* true based on docs + observation but haven't been tested end-to-end. Resolve these as we hit them — not before.

## Stream-json schema gaps

1. **`thinking_delta` exact shape.** Docs and Anthropic API streaming convention say it's `{"type":"thinking_delta","thinking":"..."}` inside `content_block_delta.delta`. We have not observed one in our trivial test (the prompt didn't trigger thinking). **Verify**: run `claude -p --include-partial-messages "<prompt that forces thinking>"` once `claude-opus-4-7` thinking is enabled in the session. Capture and confirm field is `thinking` not `text`. Also confirm whether there's a separate `signature_delta` for the thinking signature.

2. **`tool_use` `content_block_start` shape.** Probably:
   ```json
   {"type":"content_block_start","index":N,"content_block":{"type":"tool_use","id":"toolu_...","name":"Read","input":{}}}
   ```
   then `input_json_delta`s with `partial_json` strings. Not yet observed. **Verify**: prompt that forces a Bash/Read call, capture, confirm.

3. **`tool_result` events.** When claude executes a tool itself (not via the user), the tool result becomes a content block in a *user* message in the assistant's view. From the harness side it likely arrives as a `user` event followed by an `assistant` continuation. **Verify**: capture a tool-using session and trace the event order.

4. **`/compact` mid-session via stream-json input.** We hypothesize sending `"/compact"` as a user message text triggers the slash command. **Verify**: open a stream-json input session, push `/compact`, confirm we see compaction-summary text and a smaller subsequent `usage`.

5. **Full enum of `result.terminal_reason`.** Observed: `completed`. Likely other values: `max_turns_reached`, `budget_exceeded`, `user_interrupt`, `error`, `pause`. **Verify**: induce each and capture.

6. **Full enum of `result.subtype`.** Observed: `success`. Likely: `error`, `max_turns`, `budget_exceeded`. Same approach.

7. **`stop_reason` enum on `message_delta`.** Observed: `end_turn`. API docs imply: `end_turn`, `tool_use`, `max_tokens`, `stop_sequence`, `pause_turn`, `refusal`. Worth confirming.

## Stream-json INPUT format

8. **Exact JSONL message shape we push to stdin.** Highest priority. Our hypothesis based on Anthropic SDK conventions:
   ```json
   {"type":"user","message":{"role":"user","content":[{"type":"text","text":"..."}]}}
   ```
   But the CLI may accept a simpler shape, or require a different envelope. **Verify**: use `--input-format stream-json --output-format stream-json --replay-user-messages` and try shapes until echoed `user` event roundtrips correctly. Document the canonical shape in `02-stream-json-events.md`.

9. **Whether stream-json input lets us push during a running assistant turn**, or whether claude only reads stdin between turns. Both cases lead to working composers; the difference is how we render the "queued message" state.

10. **Whether `--replay-user-messages` echoes literal slash commands or processes them silently.** Affects whether the TUI renders `you ▸ /compact` as a chat bubble.

## Compaction behavior

11. **Does `/compact` produce a `result` event?** Or is the session continuous? Hypothesis: continuous. **Verify**.

12. **Does `--fork-session` emit a new `session_id` in the next stream-json events?** Hypothesis: yes, identical to `--session-id`. **Verify**.

## Subprocess lifecycle

13. **Does `claude -p` exit cleanly on stdin close?** Hypothesis: yes (it's the SDK's expected behavior). If stdin close doesn't end the process, we send a sentinel `{"type":"end"}` or similar. **Verify**.

14. **What happens on SIGINT to the subprocess mid-turn?** Hypothesis: claude prints an interrupt result event and exits. Harness needs to surface `permission_denials` and `terminal_reason: user_interrupt` cleanly. **Verify**.

15. **Permission prompts in non-interactive mode.** Without `--permission-prompt-tool`, what does claude do when a tool needs a permission decision? Hypothesis: it errors out with `terminal_reason` indicating denial. We need to either route through `--permission-prompt-tool` or run permission-mode = `acceptEdits` / `bypassPermissions` for tool sessions. **Verify by running a session in `default` mode that triggers an Edit on a never-seen file.**

## TUI plumbing

16. **Bubble Tea v2 + `--input-format stream-json` integration**. We need a goroutine that reads from a `tea.Cmd` channel and writes to claude's stdin. Pattern from notebook is to wrap the channel send in a `tea.Cmd`. Should be straightforward; just want to confirm there's no race between draining events and writing input.

17. **`charm.land/bubbles/v2` textarea word-wrap behavior** — does it handle our composer's vertical growth gracefully? Notebook patches were related to bubbles; check whether our needs intersect.

## Cost / cache

18. **Cache-read effectiveness across `--bare` sessions.** Does `--bare` + identical `--system-prompt-file` produce a stable cache key across invocations? If yes, `cache_read_input_tokens` should dominate after the first call.

19. **Cache-read across `--resume`.** Hypothesis: caching is keyed by exact prompt prefix. Since `--resume` reconstructs the prefix from the persisted session, cache hits should be identical. **Verify by running the same `--resume <id>` twice and inspecting `cache_read_input_tokens`.

## Misc

20. **`--max-turns` exit code.** Docs: "Exits with an error when the limit is reached." We need to know whether non-zero exit + a `result` event with `terminal_reason: max_turns_reached`, or just exit. Affects how the harness recognizes the case.

21. **Behavior with the `[1m]` 1-million-context model variant.** `claude-opus-4-7[1m]` is what our test ran on. Whether `--effort` flags interact with the variant differently, and whether `contextWindow` is reliably `1_000_000` in `modelUsage`. Yes, observed in our test, but worth a sanity check on smaller-window models too.

## How to resolve

When jflow is far enough along that we have a `claude` driver in code, write a small test script (`scripts/probe-stream.sh`) that runs each scenario above and dumps the full event log into `docs/probes/<name>.jsonl`. Update `02-stream-json-events.md` with what's confirmed. The list above shrinks as evidence accumulates.
