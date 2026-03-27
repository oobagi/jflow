---
name: autopilot
description: >
  Fully automated development loop — works through ROADMAP.md items, open GitHub issues, or both.
  Runs /next → /test → /ship for each. Runs /harden, /docs, /simplify, and /checkup at phase boundaries.
  Auto-compacts context when usage exceeds threshold (default 35%). Use "interactive" to confirm before each item.
  Use "phase-N" to start at a specific phase. Use "dry-run" to preview without executing.
  Use "roadmap" or "issues" to limit source; default is both.
user-invocable: true
argument-hint: >
  ["roadmap" | "issues" | "interactive" | "phase-N" | "dry-run" | "compact-N%" to set compact threshold (default 35%) | combine: "issues interactive compact-80%"]
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion, Skill, TaskCreate, TaskUpdate, TaskList
effort: high
---

# Autopilot

Fully automated development loop. Works through ROADMAP.md items, open GitHub issues, or both, analyzing them to determine optimal execution order. Runs `/next` → `/test` → `/ship` for each work item, with `/harden`, `/docs`, `/simplify`, and `/checkup` at phase boundaries. Auto-compacts context to stay within window limits.

## Agent dispatch policy

**You MUST use specialized agents throughout the autopilot run.** Each sub-skill has agents it should dispatch — this is not optional. Do not fall back to doing everything yourself when an agent exists for the job.

The table below is the **minimum baseline**, not a ceiling. If a specific situation calls for a different or additional agent — use your judgment and dispatch it. You have access to the full agent roster.

| Skill | Baseline agents | Conditional agents |
|-------|----------------|-------------------|
| `/next` | Frontend Developer OR Backend Architect OR Software Architect (choose by issue type) | — |
| `/test` | Code Reviewer, Reality Checker, Security Engineer | Accessibility Auditor (UI changes), API Tester (API changes), Performance Benchmarker (perf-sensitive changes) |
| `/harden` | Security Engineer (at phase boundaries) | — |
| `/docs` | Technical Writer (review before applying fixes) | — |
| `/simplify` | code-simplifier agents (DRY, dead code, logic — already defined in skill) | — |

When invoking each sub-skill, the skill's own instructions specify which agents to launch and when. Follow them — do not skip agent dispatches to save time. Beyond the baseline, dispatch any additional agent that fits the situation (e.g., UX Researcher for a user-facing feature, Product Manager for scope questions, Accessibility Auditor for a11y-sensitive work).

## 0. Parse arguments

Check `$ARGUMENTS` for:

- **`roadmap`** — only work roadmap items (ROADMAP.md). Skip standalone issues.
- **`issues`** — only work open GitHub issues not in the roadmap. Skip roadmap items.
- **Neither** — work **both** roadmap items and standalone issues (default).
- **`interactive`** — pause and confirm before each item. By default, autopilot runs without stopping.
- **`phase-N`** (e.g., `phase-2`) — start at that phase, skipping earlier phases. Only meaningful when roadmap items are included.
- **`dry-run`** — preview the plan without executing anything (see step 1c).
- **`compact-N%`** (e.g., `compact-80%`) — compact context when usage exceeds N%. Default: `35%`.
- **An issue number** — start from that specific issue and continue forward.

These can be combined: `issues interactive compact-80%` means "only issues, confirm before each item, compact at 80% context usage."

## 1. Gather work items

Depending on the source flag, gather work from one or both sources:

### 1a. Roadmap items (skip if `issues` flag is set)

Read `ROADMAP.md` from the project root. Parse it into a structured list:

```
Phase 1: Foundation
  - [ ] #1 Project scaffolding and CI setup
  - [x] #2 Core architecture (already done — skip)
  - [ ] #3 ...
Phase 2: ...
  - [ ] #4 ...
```

Build a list of **uncompleted roadmap items** (lines matching `- [ ]`). Each item has:
- Phase name and number
- Issue number (extracted from `#N`)
- Title
- Source: `roadmap`

If a `phase-N` argument was given, filter to only items in that phase or later.

### 1b. Standalone issues (skip if `roadmap` flag is set)

Fetch all open GitHub issues:

```bash
gh issue list --state open --json number,title,labels,body --limit 200
```

**Filter out** any issue whose number already appears in `ROADMAP.md` — those are roadmap items, not standalone issues. The remaining issues are standalone work (bugs, enhancements, chores filed outside the roadmap).

For each standalone issue, record:
- Issue number
- Title
- Labels (used for classification)
- Source: `issue`

### 1c. Analyze ordering (when both sources are active)

When working both roadmap items and standalone issues, the execution order matters. Launch a **Product Manager** agent (`subagent_type: "Product Manager"`) with the full list of roadmap items and standalone issues. Tell it to:

1. **Identify superseding features** — find roadmap features that would **fix or make obsolete** one or more standalone issues. For example, an issue about a broken UI element is superseded by a roadmap feature that replaces that entire UI. These features should list which issues they supersede.

2. **Classify standalone issues** — for each remaining standalone issue (not superseded), classify as:
   - **Bug** — something is broken
   - **Enhancement** — improvement to existing functionality
   - **Chore** — refactoring, CI, deps, etc.
   - **Feature** — new functionality not on the roadmap

3. **Produce the ordered work queue** following these priority rules:

   **Priority 1 — Superseding features (roadmap items that fix multiple issues).**
   Build these first. When the PR merges with `Closes #N` for each superseded issue, those issues auto-close — no separate work needed. Order these by how many issues they supersede (most first).

   **Priority 2 — Standalone bugs and chores.**
   Issues generally come before features. Bugs especially — ship on broken foundations is wasted work. Order by severity/impact (P0 > P1 > P2, or by label priority if available).

   **Priority 3 — Remaining roadmap items (in roadmap order).**
   Items that don't supersede any issues proceed in their original roadmap phase order.

   **Priority 4 — Standalone enhancements and features.**
   Nice-to-haves that aren't on the roadmap go last.

The agent returns a structured ordered list. Each entry has:
- Issue number, title, source (`roadmap` or `issue`)
- Phase (for roadmap items) or priority tier (for standalone issues)
- Supersedes: list of issue numbers this item will auto-close (if any)

**When only one source is active** (`roadmap` or `issues` flag), skip the Product Manager analysis:
- `roadmap` only: use roadmap order as-is.
- `issues` only: order bugs before enhancements before features. Within each group, order by labels/priority or by issue number (oldest first).

### 1d. Report the plan

Count total items and report:

> **Autopilot engaged.** N items from [roadmap | issues | roadmap + issues].
> Starting with: #X — Title
> Mode: [autopilot | interactive | dry-run]
> Compact threshold: N%

If superseding features were identified:

> **Superseding features detected:** K roadmap items will auto-close M standalone issues.
> These features are queued first.

### 1e. Dry-run mode

If `dry-run` was specified, print the full plan and stop. Do NOT execute anything.

```
═══════════════════════════════════════
  Autopilot Dry Run
═══════════════════════════════════════

  Source: roadmap + issues (or: roadmap | issues)
  Items: N total (R from roadmap, I from issues)
  Compact threshold: K%

  ── Priority 1: Superseding features ──
    1. #49 — Strip TUI pickers from CLI commands [roadmap, Phase 6]
       └─ supersedes: #55, #58 (auto-close on merge)
    2. #51 — TUI inline text input [roadmap, Phase 6]
       └─ supersedes: #60 (auto-close on merge)

  ── Priority 2: Bugs & chores ──
    3. #56 — fix: crash on empty notebook [issue, bug]
    4. #57 — chore: update Go deps [issue, chore]

  ── Priority 3: Remaining roadmap ──
    5. #50 — TUI help overlay [roadmap, Phase 6]
    6. #52 — TUI delete with type-to-confirm [roadmap, Phase 6]
    ── phase boundary: /docs → /simplify → /checkup ──

  ── Priority 4: Standalone enhancements ──
    7. #59 — enhance: improve search highlighting [issue, enhancement]

  Auto-close summary: 3 issues will be closed by superseding features
  Compact threshold: K% (compacts triggered as needed)

  Run without dry-run to execute:
    /autopilot
═══════════════════════════════════════
```

After printing, exit. Do not proceed to step 2.

## 2. Create progress tracker

Use `TaskCreate` to create a parent task for the full autopilot run, then child tasks for each item. This gives the user visibility into progress.

## 3. The loop

For each work item in the ordered queue (from step 1c/1d), in order:

### 3a. Gate check (interactive mode only)

If in interactive mode, ask the user:

> **Next up:** #N — Title [source: roadmap Phase X | issue (bug/enhancement/chore)]
> Continue? [yes / skip / stop]

- **yes** (or just Enter) — proceed
- **skip** — skip this item, move to the next
- **stop** — end the autopilot run, print summary

In default (autopilot) mode, skip this prompt entirely.

### 3b. Run `/next <issue-number>`

Record the start time for this item: `item_start=$(date +%s)`.

Invoke the `next` skill with the issue number. This:
- Picks up the issue
- Implements it in a worktree
- Enters the worktree

If `/next` fails or the implementation agent reports a problem it can't resolve:
- **Stop the loop.**
- Report which item failed and why.
- Tell the user to fix it manually and re-run `/autopilot` to continue from where it left off.

### 3c. Run `/test`

Invoke the `test` skill to validate the implementation from `/next`. This is the **final gate before shipping**. It:
- Runs lint, tests, code review, and reality check
- Auto-fixes blockers if possible

If `/test` finds unfixable blockers:
- **Stop the loop.**
- Report the blockers.
- Tell the user to fix them manually, then run `/test` followed by `/ship` and `/autopilot` to resume.

### 3d. Run `/ship`

Invoke the `ship` skill. This:
- Creates a branch, commits, opens a PR
- Merges the PR (or queues auto-merge)
- Cleans up the worktree and branch

**Superseded issues:** If the current work item has a `supersedes` list (from step 1c), ensure all superseded issue numbers are included in the commit message and PR body as `Closes #N`. The `/ship` skill already uses `Closes #N` for the primary issue — for superseding features, append additional `Closes #N` entries for every superseded issue. For example, if roadmap item #49 supersedes issues #55 and #58:

- Commit message: `Fix #49: Strip TUI pickers from CLI commands`
- PR title: `feat: strip TUI pickers from CLI commands (Closes #49, Closes #55, Closes #58)`

GitHub auto-closes all referenced issues when the PR merges.

If `/ship` fails (e.g., CI fails, merge blocked):
- **Stop the loop.**
- Report the failure and the PR URL.
- Tell the user to resolve it, then re-run `/autopilot` to continue.

### 3e. Verify auto-closes and skip superseded items

After `/ship` merges a superseding feature:

1. **Verify** the superseded issues actually closed by running `gh issue view <number> --json state` for each. If any didn't auto-close (e.g., the `Closes` keyword was missed), close them manually with `gh issue close <number> -c "Resolved by #<PR>"`.

2. **Remove superseded items from the queue.** Any standalone issue that was in the work queue but is now closed should be skipped when its turn comes. Before starting each item in the loop, check if the issue is still open — if it was already closed by a prior superseding feature, skip it with a log message:

> **Skipping #N** — already closed by #M (superseding feature)

### 3f. Mark progress

Record end time and compute duration: `item_end=$(date +%s); duration=$((item_end - item_start))`.

Update the task for this item to `completed`. Log:
- Issue number and title
- Source (`roadmap` or `issue`)
- PR URL (from `/ship` output)
- Duration (formatted as `Xm Ys`)
- Auto-closed issues (if this was a superseding feature)

Track cumulative durations across items so the summary can report average time per item.

### 3g. Auto-compact context

After each item, check context usage. **Note:** `/context` and `/compact` are built-in Claude Code CLI commands, not jflow skills — do NOT invoke them via the Skill tool. Instead, check the context usage indicator in the conversation and use the `/compact` slash command directly when needed. If usage exceeds the compact threshold (default 35%):

1. Run `/compact` (built-in CLI command) to compress conversation context
2. After compact, briefly re-state the current position:

> **Resuming autopilot.** Phase N, item M of total. Next: #X — Title

This prevents context exhaustion on long runs. The threshold is configurable via `compact-N%` argument — lower values (25-35%) for aggressive compaction, higher values (60%+) to let context accumulate longer.

### 3h. Phase boundary check

After completing a **roadmap** item, check: **is this the last roadmap item in the current phase?**

**Note:** Phase boundaries only apply to roadmap items. Standalone issues don't belong to phases. If the work queue interleaves issues between roadmap items in the same phase, the phase boundary triggers after the last roadmap item in that phase ships — regardless of where standalone issues fall in the queue.

If yes:
1. Announce: `Phase N complete. Running harden → docs → simplify → checkup...`
2. Invoke the `harden` skill in fix mode, scoped to files changed during this phase. This audits for error handling gaps, missing validation, and security issues accumulated across the phase's items, then applies critical and high severity fixes. If harden produces changes, ship them with `/ship` using commit message `harden: phase N security fixes`.
3. Invoke the `docs` skill to sync documentation (README, AGENTS.md, CHANGELOG, etc.) with the code changes from this phase. If docs produces changes, ship them with `/ship` using commit message `docs: sync after phase N`.
4. Invoke the `simplify` skill (no arguments, auto-scopes to changes since last simplify). This cleans up DRY violations, dead code, and complex logic accumulated during the phase. If simplify produces changes, ship them with `/ship` using commit message `simplify: phase N cleanup`.
5. Invoke the `checkup` skill with `now` (auto-clean without confirmation).
6. Report harden, docs, simplify, and checkup results before continuing to the next phase.

Also run `/checkup now` (without `/docs` or `/simplify`) if:
- 5+ items have been shipped since the last checkup (even mid-phase)
- The autopilot run is ending (final item completed)

## 4. Final smoke test (`/qa auto`)

After the loop completes successfully (all items shipped, not stopped early by error or user), run `/qa auto` as a final end-to-end smoke test of everything that was just built.

- Invoke via the Skill tool: `skill: "qa", args: "auto"`
- This exercises the shipped features with real interactions (Playwright for UI, curl for APIs, shell for CLI) — catching integration issues that per-item `/test` reviews can't see, since each `/test` only reviewed code in isolation.
- Include the `/qa auto` results (pass/fail counts, any failures) in the summary below.
- If `/qa auto` finds failures, report them in the summary but do **not** stop or revert — the code is already shipped. The user can follow up.

Skip this step if the loop was stopped early (user chose "stop" in interactive mode, or an error halted the run) — a partial run doesn't warrant a full smoke test.

## 5. Summary

When the loop finishes (all items done, or user stopped it), print a full summary:

```
═══════════════════════════════════════
  Autopilot Complete
═══════════════════════════════════════

  Source: roadmap + issues
  Items shipped:  7 / 12 (4 roadmap, 3 issues)
  Items skipped:  1
  Auto-closed:    2 issues (superseded by features)
  Phases cleared: 2 (Phase 1, Phase 2)
  Total time:     1h 23m (avg 11m 51s/item)
  Compacts:       2 (threshold: 35%)

  PRs merged:
    • #14 — feat: strip TUI pickers (Closes #49, #55, #58) [12m 04s] [roadmap]
       └─ auto-closed: #55, #58
    • #15 — fix: crash on empty notebook (Closes #56) [5m 21s] [issue]
    • #16 — feat: TUI help overlay (Closes #50) [8m 12s] [roadmap]
    • ...

  Docs syncs: 2
    • Phase 1 — updated README quickstart, AGENTS.md structure
    • Phase 2 — added API docs, updated CHANGELOG

  Simplify passes: 2
    • Phase 1 — 3 helpers created, 12 dead items removed, 5 logic simplifications
    • Phase 2 — 1 helper created, 8 dead items removed, 3 logic simplifications

  Checkups run: 2
    • Phase 1 boundary — clean
    • Phase 2 boundary — removed 3 stale branches

  Smoke test (/qa auto):
    Total: 12 | Passed: 11 | Failed: 1 | Skipped: 0
    • FAIL: API-03 — POST /saves returns 500 when slot is full (expected 409)

  Remaining:
    • Roadmap — Phase 3: 4 items
    • Issues — 2 open issues
    • Total: 6 items

  Resume with: /autopilot
═══════════════════════════════════════
```

## 6. Error recovery

The autopilot is designed to **stop on first failure** rather than skip and continue. This is intentional — later roadmap items may depend on earlier ones, so skipping a failure is risky.

The user can always:
- Fix the issue manually
- Run `/autopilot` again — it re-reads ROADMAP.md and re-fetches open issues, then picks up from the first uncompleted item
- Use `/autopilot roadmap` or `/autopilot issues` to resume with a narrower scope

## Style guidelines

- Follow the standard output format in `_output-format.md`
- Be concise between items — the user is watching a stream of work, not reading docs
- Use clear status markers: starting / implementing / testing / shipping / done
- Bold the current item and phase at each step so it's easy to scan
- Even in non-interactive mode, still print one-line status updates per item (don't go silent)
