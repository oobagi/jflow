---
name: autopilot
description: >
  Fully automated development loop — iterates through ROADMAP.md items one at a time,
  running /next → /test → /ship for each. Runs /checkup at phase boundaries.
  Use "yolo" to skip confirmations. Use "phase-N" to start at a specific phase.
user_invocable: true
argument-hint: >
  ["interactive" to confirm before each item | "phase-N" to start at phase N | both: "interactive phase-2"]
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion, Skill, TaskCreate, TaskUpdate, TaskList
effort: high
---

# Autopilot

Fully automated development loop. Reads ROADMAP.md and works through every uncompleted item in order: `/next` → `/test` → `/ship`, with `/checkup` at phase boundaries.

## 0. Parse arguments

Check `$ARGUMENTS` for:

- **`interactive`** — pause and confirm before each item. By default, autopilot runs without stopping.
- **`phase-N`** (e.g., `phase-2`) — start at that phase, skipping earlier phases.
- **An issue number** — start from that specific issue and continue forward.

These can be combined: `interactive phase-2` means "start at phase 2, confirm before each item."

## 1. Read the roadmap

Read `ROADMAP.md` from the project root. Parse it into a structured list:

```
Phase 1: Foundation
  - [ ] #1 Project scaffolding and CI setup
  - [x] #2 Core architecture (already done — skip)
  - [ ] #3 ...
Phase 2: ...
  - [ ] #4 ...
```

Build an ordered list of **uncompleted items** (lines matching `- [ ]`). Each item has:
- Phase name and number
- Issue number (extracted from `#N`)
- Title

If a `phase-N` argument was given, filter to only items in that phase or later.

Count total items to work through and report:

> **Autopilot engaged.** N items remaining across M phases.
> Starting with: #X — Title
> Mode: [yolo | interactive]

## 2. Create progress tracker

Use `TaskCreate` to create a parent task for the full autopilot run, then child tasks for each item. This gives the user visibility into progress.

## 3. The loop

For each uncompleted roadmap item, in order:

### 3a. Gate check (unless yolo)

If NOT in yolo mode, ask the user:

> **Next up:** #N — Title (Phase X)
> Continue? [yes / skip / stop]

- **yes** (or just Enter) — proceed
- **skip** — skip this item, move to the next
- **stop** — end the autopilot run, print summary

If in yolo mode, skip this prompt entirely.

### 3b. Run `/next <issue-number>`

Invoke the `next` skill with the issue number. This:
- Picks up the issue
- Implements it in a worktree
- Enters the worktree

If `/next` fails or the implementation agent reports a problem it can't resolve:
- **Stop the loop.**
- Report which item failed and why.
- Tell the user to fix it manually and re-run `/autopilot` to continue from where it left off.

### 3c. Run `/test`

Invoke the `test` skill to validate the implementation. This:
- Runs lint, tests, code review, and reality check
- Auto-fixes blockers if possible

If `/test` finds unfixable blockers:
- **Stop the loop.**
- Report the blockers.
- Tell the user to fix them manually, then run `/ship` followed by `/autopilot` to resume.

### 3d. Run `/ship`

Invoke the `ship` skill. This:
- Creates a branch, commits, opens a PR
- Merges the PR (or queues auto-merge)
- Cleans up the worktree and branch

If `/ship` fails (e.g., CI fails, merge blocked):
- **Stop the loop.**
- Report the failure and the PR URL.
- Tell the user to resolve it, then re-run `/autopilot` to continue.

### 3e. Mark progress

Update the task for this item to `completed`. Log:
- Issue number and title
- PR URL (from `/ship` output)
- Time taken (if trackable)

### 3f. Phase boundary check

After completing an item, check: **is this the last item in the current phase?**

If yes:
- Announce: `Phase N complete. Running health checkup...`
- Invoke the `checkup` skill with `now` (auto-clean without confirmation).
- Report checkup results before continuing to the next phase.

Also run `/checkup now` if:
- 5+ items have been shipped since the last checkup (even mid-phase)
- The autopilot run is ending (final item completed)

## 4. Summary

When the loop finishes (all items done, or user stopped it), print a full summary:

```
═══════════════════════════════════════
  Autopilot Complete
═══════════════════════════════════════

  Items shipped:  7 / 12
  Items skipped:  1
  Phases cleared: 2 (Phase 1, Phase 2)

  PRs merged:
    • #14 — feat: add WASD controls (Closes #3)
    • #15 — feat: implement save system (Closes #4)
    • ...

  Checkups run: 2
    • Phase 1 boundary — clean
    • Phase 2 boundary — removed 3 stale branches

  Remaining:
    • Phase 3: 4 items
    • Phase 4: 1 item

  Resume with: /autopilot phase-3
═══════════════════════════════════════
```

## Error recovery

The autopilot is designed to **stop on first failure** rather than skip and continue. This is intentional — later roadmap items may depend on earlier ones, so skipping a failure is risky.

The user can always:
- Fix the issue manually
- Run `/autopilot` again — it re-reads ROADMAP.md and picks up from the first uncompleted item

## Style guidelines

- Be concise between items — the user is watching a stream of work, not reading docs
- Use clear status markers: starting / implementing / testing / shipping / done
- Bold the current item and phase at each step so it's easy to scan
- If in yolo mode, still print one-line status updates per item (don't go silent)
