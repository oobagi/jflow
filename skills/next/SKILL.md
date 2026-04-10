---
name: next
description: Pick up the next roadmap item (or a specific issue). Use "parallel" to suggest parallel issues and work in worktrees.
user-invocable: true
argument-hint: [issue number | "parallel" to also suggest parallel issues]
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent
effort: high
---

# Next

Work on the next open roadmap item (or a specific issue if provided). By default, works on a feature branch directly. Use "parallel" to work in worktrees instead.

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

## 3. Implement

**Choose isolation mode based on workflow:**

- **Default (serial):** Create a feature branch from main (`git checkout -b feat/issue-<N>-<slug>`) and work directly on it. No worktree needed.
- **Parallel mode** (when `$ARGUMENTS` contains "parallel"): Use `isolation: "worktree"` when spawning the implementation agent, so multiple issues can run concurrently in separate worktrees.

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

1. If a worktree was used (parallel mode), enter it using `EnterWorktree` so `/test` and `/ship` run in context.
2. **Assess complexity** to determine the recommended next step:

**Recommend `/ship` directly when:**
- The change is small (≤3 files modified, no new files)
- It's a config change, docs update, dependency bump, or simple bug fix
- No security-sensitive code was touched (auth, payments, crypto, API keys)
- No new API endpoints or database changes
- Lint and tests already passed during implementation

**Recommend `/test` then `/ship` when:**
- The change is medium-to-large (4+ files, new modules)
- Security-sensitive code was touched
- New API endpoints, database migrations, or auth logic
- Complex business logic or state management
- The implementation agent reported uncertainty about any part

3. Present the result with a **How to test** section — brief, concrete steps the user can follow to verify each change manually:

```
═══════════════════════════════════════
  Next — Complete
═══════════════════════════════════════

  Issue: #42 — <issue title>
  Branch: feat/issue-42-description

  Done:
    ✓ Read issue spec
    ✓ Implemented
    ✓ Lint — clean
    ✓ Tests — passing

  How to test:
    <command to run the app, e.g. "go run ./cmd/notebook">
    - <step 1 — what to do and what to expect>
    - <step 2 — etc.>

  Next: /ship
  (or)
  Next: /test to validate, then /ship

═══════════════════════════════════════
```

If implementation failed:

```
═══════════════════════════════════════
  Next — Stopped
═══════════════════════════════════════

  Issue: #42 — <issue title>

  Done:
    ✓ Read issue spec
    ✗ Implementation — <reason>

  Fix: <what needs manual attention>
  Resume: resolve the issue, then /ship (or /test then /ship)

═══════════════════════════════════════
```

## Style guidelines
- Follow the standard output format in `_output-format.md`
