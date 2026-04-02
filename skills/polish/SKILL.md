---
name: polish
description: >
  Post-implementation quality pipeline for ad-hoc work — runs /simplify → /harden → /test → /ship
  in sequence. Use when you've built something on the fly and want to clean it up and ship it.
  Use "dry-run" to preview scope without executing. Use "no-ship" to stop before shipping.
user-invocable: true
argument-hint: >
  ["dry-run" to preview | "no-ship" to stop before /ship | "skip-simplify" | "skip-harden" | combine: "no-ship skip-simplify"]
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion, Skill, TaskCreate, TaskUpdate, TaskList
effort: high
---

# Polish

Post-implementation quality pipeline for ad-hoc work. Chains `/simplify` → `/harden` → `/test` → `/ship` in sequence, stopping on first failure. Designed for the common pattern: you built something on the fly and now want to harden, validate, and ship it without manually invoking each skill.

## Agent dispatch policy

**You MUST use specialized agents when invoking sub-skills.** Each sub-skill (`/simplify`, `/harden`, `/test`, `/ship`) has agents it should dispatch — follow the agent dispatch instructions in each skill. Do not skip agent dispatches to save time.

## 0. Parse arguments

Check `$ARGUMENTS` for:

- **`dry-run`** — show what changed and what the pipeline would do, then stop.
- **`no-ship`** — run simplify → harden → test but stop before `/ship`. Useful when you want to review before shipping.
- **`skip-simplify`** — skip the `/simplify` step (jump straight to `/harden`).
- **`skip-harden`** — skip the `/harden` step.
- **A scope path** (e.g., `scope:src/api`) — pass through to `/simplify` and `/harden` to limit their scope.

These can be combined: `no-ship skip-simplify` means "harden and test only, don't ship."

## 1. Assess the work

Understand what's been built before running the pipeline:

1. Run `git status` and `git diff --stat` to see all uncommitted and staged changes.
2. Run `git diff` to understand the full scope.
3. Check for new untracked files with `git status`.

Report the scope:

```
═══════════════════════════════════════
  Polish — Starting
═══════════════════════════════════════

  Changes detected:
    Modified:  N files
    New:       N files
    Deleted:   N files
    Lines:     +N / -N

  Pipeline: simplify → harden → test → ship
  Mode: [full | no-ship | dry-run]

═══════════════════════════════════════
```

### Dry-run mode

If `dry-run` was specified, print the scope report above and stop. Do NOT execute any pipeline steps.

### No changes guard

If there are no uncommitted changes and no untracked files, report "Nothing to polish" and exit. Don't run an empty pipeline.

## 2. Create progress tracker

Use `TaskCreate` to create a parent task for the polish run, then child tasks for each pipeline step:

- Simplify (if not skipped)
- Harden (if not skipped)
- Test
- Ship (if not `no-ship`)

## 3. Run the pipeline

Execute each step in order. **Stop on first failure** — don't skip broken steps.

### 3a. Simplify (unless `skip-simplify`)

Record start time.

Invoke the `simplify` skill. If a `scope:` argument was given, pass it through.

Default scope (no explicit scope) will auto-scope to recent changes, which is exactly what we want for ad-hoc work.

If `/simplify` fails or introduces test regressions it can't resolve:
- **Stop the pipeline.**
- Report what failed.
- Tell the user: "Fix the issue, then run `/polish skip-simplify` to resume."

Update task to completed. Record duration.

### 3b. Harden (unless `skip-harden`)

Record start time.

Invoke the `harden` skill in fix mode (the default). If a `scope:` argument was given, pass it through.

If `/harden` introduces regressions it can't resolve:
- **Stop the pipeline.**
- Report what failed.
- Tell the user: "Fix the issue, then run `/polish skip-simplify skip-harden` to resume."

If `/harden` finds no issues, it reports clean and moves on immediately.

Update task to completed. Record duration.

### 3c. Test

Record start time.

Invoke the `test` skill to validate everything.

If `/test` finds unfixable blockers:
- **Stop the pipeline.**
- Report the blockers.
- Tell the user: "Fix the blockers, then run `/polish skip-simplify skip-harden` to resume from /test."

Update task to completed. Record duration.

### 3d. Ask to ship (unless `no-ship`)

After `/test` passes, **do not automatically invoke `/ship`.** Instead, ask the user:

> Ready to ship. Run `/ship` to branch, commit, PR, and merge?

Wait for the user's response. Only invoke `/ship` if the user confirms. If the user declines or wants to review first, stop the pipeline gracefully — this is not a failure.

If the user confirms and `/ship` fails:
- **Stop the pipeline.**
- Report the failure and PR URL if one was created.
- Tell the user to resolve it manually.

Update task to completed. Record duration.

## 4. Summary

Print the final summary:

```
═══════════════════════════════════════
  Polish — Complete
═══════════════════════════════════════

  Pipeline results:
    ✓ Simplify    (Xm Ys) — N helpers, N dead items removed, N simplifications
    ✓ Harden      (Xm Ys) — N fixes applied, N deferred
    ✓ Test        (Xm Ys) — no blockers
    ✓ Ship        (Xm Ys) — PR #N merged (after user confirmed)

  Total time: Xm Ys
  PR: <url>

═══════════════════════════════════════
```

If `no-ship` was specified, the summary ends with:

```
  Next: review changes, then /ship
```

If the pipeline stopped early due to failure:

```
═══════════════════════════════════════
  Polish — Stopped
═══════════════════════════════════════

  Pipeline results:
    ✓ Simplify    (Xm Ys) — N helpers, N dead items removed
    ✗ Harden      — test regression in src/api/auth.ts

  Fix the issue, then resume:
    /polish skip-simplify

═══════════════════════════════════════
```

## Error recovery

The pipeline is designed to **stop on first failure**. Each step can introduce code changes, so continuing past a broken step risks compounding errors.

Resume flags (`skip-simplify`, `skip-harden`) let the user skip already-completed steps when resuming after a fix. The user can also run the individual skills directly if they prefer more control.

## Style guidelines

- Follow the standard output format in `_output-format.md`
- Be concise between steps — one-line status updates, not commentary.
- Use clear markers: `▸ Simplify...` → `✓ Simplify (2m 14s)`
- Bold the current step so it's easy to spot in the stream.
- Don't re-explain what each sub-skill does — the user knows. Just run it and report.
