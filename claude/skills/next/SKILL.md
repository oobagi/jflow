---
name: next
description: Pick up the next roadmap item (or a specific issue), work in a worktree. Suggests parallel issues only when auto-picking.
user_invocable: true
argument-hint: [issue number | "go" to skip parallel suggestions and start immediately]
---

# Next

Work on the next open roadmap item (or a specific issue if provided). All implementation happens in a worktree.

## 1. Identify work

If `$ARGUMENTS` contains an issue number, use that and **skip straight to step 3**. Otherwise:

- Read `ROADMAP.md` to find the next uncompleted item (first open issue in dependency order).
- Run `gh issue view <number>` to get the full spec.

## 2. Suggest parallel work (auto-pick only)

**Skip this entire step if any of these are true:**
- An issue number was provided in `$ARGUMENTS`
- `$ARGUMENTS` contains "go" (e.g., `/next go`) — proceed directly to implementation

Only run when auto-picking from the roadmap with no flags:

List a few open issues that are **NOT in the roadmap** — these are standalone bugs, CLI improvements, or features the user can hand off to other agents in separate worktrees. To find them:

- Run `gh issue list --state open` and cross-reference against `ROADMAP.md`
- Any open issue whose number does NOT appear in `ROADMAP.md` is a candidate
- Present them briefly with issue number, one-line description, and rough scope (bug fix / medium / large)

**STOP here.** Present the findings from steps 1-2 to the user and wait for their go-ahead before proceeding to implementation.

## 3. Implement in a worktree

Work strictly in a worktree (use `isolation: "worktree"` when spawning the implementation agent). The agent should:

- Read the issue spec and all likely files listed
- Implement the feature or fix
- Run `anyzork lint` and fix any issues
- Run any relevant tests
- Report back with a summary of changes

## 5. Hand off to user

Once implementation is complete, tell the user:

1. What was done (brief summary)
2. How to test it (specific commands to run)
3. Prompt them to use `/ship` when satisfied
