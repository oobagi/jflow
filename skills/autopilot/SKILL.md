---
name: autopilot
description: >
  Automated development loop — works through ROADMAP.md items, open GitHub issues, or both.
  Runs /next → /ship for each item, with /test only when the change warrants it.
  Use "interactive" to confirm before each item. Use "thorough" for phase boundary maintenance.
  Use "dry-run" to preview without executing.
user-invocable: true
argument-hint: >
  ["roadmap" | "issues" | "interactive" | "thorough" | "phase-N" | "dry-run" | combine: "issues interactive"]
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion, Skill, TaskCreate, TaskUpdate, TaskList
effort: high
---

# Autopilot

Automated development loop. Works through ROADMAP.md items, open GitHub issues, or both. Default loop is `/next → /ship` per item, with `/test` only when the change warrants it.

## Agent dispatch policy

**Use specialized agents throughout.** Each sub-skill has agents it should dispatch — follow those instructions. Do not fall back to doing everything yourself.

## 0. Parse arguments

Check `$ARGUMENTS` for:

- **`roadmap`** — only work roadmap items. Skip standalone issues.
- **`issues`** — only work standalone issues not in the roadmap. Skip roadmap items.
- **Neither** — work both (default).
- **`interactive`** — pause and confirm before each item.
- **`thorough`** — enable phase boundary maintenance (harden → docs → simplify → checkup between phases). Off by default.
- **`phase-N`** (e.g., `phase-2`) — start at that phase, skipping earlier ones.
- **`dry-run`** — preview the plan without executing.
- **An issue number** — start from that specific issue.

These combine: `issues interactive` means "only issues, confirm before each."

## 1. Gather work items

### 1a. Roadmap items (skip if `issues` flag)

Read `ROADMAP.md`. Build a list of uncompleted items (`- [ ]` lines) with phase, issue number, and title.

If `phase-N` was given, filter to that phase or later.

### 1b. Standalone issues (skip if `roadmap` flag)

```bash
gh issue list --state open --json number,title,labels --limit 200
```

Filter out issues already in `ROADMAP.md`. The rest are standalone.

### 1c. Order the queue

Simple ordering — no Product Manager agent needed:

1. **Bugs first** — from standalone issues, sorted by label priority or issue number
2. **Roadmap items** — in roadmap order (phase 1 before phase 2, top before bottom)
3. **Standalone enhancements and features** — last, by issue number

When only one source is active (`roadmap` or `issues` flag), use its natural order.

### 1d. Report the plan

```
═══════════════════════════════════════
  Autopilot — Starting
═══════════════════════════════════════

  Source: roadmap + issues (or: roadmap | issues)
  Items: N total
  Mode: [autopilot | interactive | dry-run]
  Thorough: [yes | no]

  Queue:
    1. #N — <title> [roadmap Phase X | issue (bug)]
    2. #N — <title> [...]
    ...

═══════════════════════════════════════
```

### 1e. Dry-run mode

If `dry-run`, print the plan and stop. Do not execute anything.

## 2. Create progress tracker

Use `TaskCreate` to create a parent task for the run, then child tasks for each item.

## 3. The loop

For each work item in order:

### 3a. Gate check (interactive mode only)

If interactive:

> **Next up:** #N — Title [source]
> Continue? [yes / skip / stop]

In default mode, skip the prompt.

### 3b. Run `/next <issue-number>`

Invoke the `next` skill. This picks up the issue, implements it, and reports back.

If `/next` fails: **stop the loop**, report which item failed and why.

### 3c. Decide: test or ship?

Read `/next`'s output. It will recommend either `/ship` directly or `/test → /ship` based on change complexity.

**Follow its recommendation:**
- If `/next` said "Next: /ship" → go to 3d.
- If `/next` said "Next: /test" → run `/test`, then go to 3d if it passes.

If `/test` finds unfixable blockers: **stop the loop**, report the blockers.

### 3d. Run `/ship`

Invoke the `ship` skill. This branches, commits, PRs, merges, and cleans up.

If `/ship` fails: **stop the loop**, report the failure.

### 3e. Mark progress

Update the task for this item to `completed`. Log issue number, title, and PR URL.

### 3f. Phase boundary check (only when `thorough` flag is set)

After completing a roadmap item, check if it's the last item in the current phase.

If yes and `thorough` is enabled:
1. Announce: `Phase N complete. Running harden → docs → simplify → checkup...`
2. Invoke `/harden` in fix mode, scoped to files changed this phase. Ship fixes if any.
3. Invoke `/docs` to sync documentation. Ship changes if any.
4. Invoke `/simplify` (auto-scoped to recent changes). Ship changes if any.
5. Invoke `/checkup now`.

If `thorough` is not set, skip all of this. Just continue to the next item.

## 4. Summary

When the loop finishes:

```
═══════════════════════════════════════
  Autopilot — Complete
═══════════════════════════════════════

  Source: roadmap + issues
  Items shipped: N / M

  PRs merged:
    • #N — <title> (Closes #X)
    • #N — <title> (Closes #X)
    • ...

  Remaining:
    • N items still open

  Resume: /autopilot

═══════════════════════════════════════
```

## 5. Error recovery

Autopilot **stops on first failure** — later items may depend on earlier ones.

The user can:
- Fix the issue manually
- Run `/autopilot` again — it picks up from the first uncompleted item
- Use `/autopilot roadmap` or `/autopilot issues` to narrow scope

## Style guidelines

- Follow the standard output format in `_output-format.md`
- Be concise between items — one-line status updates
- Bold the current item at each step
- Don't go silent in non-interactive mode — still print status per item
