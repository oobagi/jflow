---
name: next
description: Pick up the next roadmap item (or a specific issue), work in a worktree. Use "parallel" to also suggest parallel issues.
user_invocable: true
argument-hint: [issue number | "parallel" to also suggest parallel issues]
---

# Next

Work on the next open roadmap item (or a specific issue if provided). All implementation happens in a worktree.

## 1. Identify work

If `$ARGUMENTS` contains an issue number, use that and **skip straight to step 3**. Otherwise:

- Read `ROADMAP.md` to find the next uncompleted item (first open issue in dependency order).
- Run `gh issue view <number>` to get the full spec.

## 2. Suggest parallel work (only when explicitly requested)

**Skip this entire step unless `$ARGUMENTS` contains "parallel"** (e.g., `/next parallel`).

List a few open issues that are **NOT in the roadmap** — these are standalone bugs, CLI improvements, or features the user can hand off to other agents in separate worktrees. To find them:

- Run `gh issue list --state open` and cross-reference against `ROADMAP.md`
- Any open issue whose number does NOT appear in `ROADMAP.md` is a candidate
- Present them briefly with issue number, one-line description, and rough scope (bug fix / medium / large)

**STOP here.** Present the findings from steps 1-2 to the user and wait for their go-ahead before proceeding to implementation.

## 3. Implement in a worktree

Work strictly in a worktree (use `isolation: "worktree"` when spawning the implementation agent).

**Choose the right agent based on the work:**

- If the issue primarily involves **UI/frontend** files (React, Vue, Svelte, SwiftUI, CSS, HTML): use `subagent_type: "Frontend Developer"`
- If the issue primarily involves **backend/API/infrastructure** (routes, database, services, DevOps): use `subagent_type: "Backend Architect"`
- If the issue involves **architecture or design decisions** (new modules, major refactors, system boundaries): use `subagent_type: "Software Architect"`
- If the issue is a **mix or unclear**: use a general-purpose agent

The agent should:

- Read the issue spec and all likely files listed
- Implement the feature or fix
- Run lint and fix any issues
- Run any relevant tests
- Report back with a summary of changes

## 4. Hand off to user

Once implementation is complete:

1. Enter the worktree using `EnterWorktree` so `/test` and `/ship` run in context
2. Tell the user:
   - What was done (brief summary)
   - Which issue was implemented (e.g., "Working on #42")
   - How to test it (specific commands to run)
   - Prompt them to use `/test` to validate, then `/ship` when clean
