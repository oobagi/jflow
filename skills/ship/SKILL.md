---
name: ship
description: Branch, commit, PR, merge, and cleanup. Use when work is ready to ship.
user_invocable: true
---

# Ship

Run these steps in order. Stop and report if any step fails.

## 1. Pre-flight checks

If the project has a lint script, run it. If the project has a build script, run it. Fix any errors before continuing.

## 2. Review changes

Run `git status` and `git diff --stat` to see what changed. Review the changes to understand what's being shipped.

## 3. Create branch and commit

If already on a feature branch (not `main` or `master`), skip branch creation. Otherwise, create a descriptive branch from the main branch (e.g., `fix/play-pause-ui-state`, `feat/wasd-controls`).

Stage all relevant files (do NOT use `git add -A` — be selective, avoid committing .env, .db files, or other artifacts).

Write a commit message that:

- Has a short title line describing the change
- If fixing a GitHub issue, prefix with `Fix #N:` to auto-close the issue on merge
- Includes a body explaining the why, not just the what
- Ends with `Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>`

Use a HEREDOC to pass the message:

```
git commit -m "$(cat <<'EOF'
Title here

Body explaining the change.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

## 4. Push and create PR

Push the branch with `git push -u origin <branch>`.

Create a PR with `gh pr create`.

The PR title should include `(Closes #N)` when implementing or fixing an issue — e.g., `feat: add WASD controls (Closes #42)`. This auto-closes the issue on merge.

The PR body should include:

- `## Summary` with bullet points of what changed
- `## Test plan` with a checklist of how it was verified
- Footer: `🤖 Generated with [Claude Code](https://claude.com/claude-code)`

## 5. Wait for CI, then merge

**Always wait for CI checks to pass before merging** — even if the repo has no branch protection.

1. Poll with `gh pr view <number> --json statusCheckRollup` every 15 seconds.
2. If no checks are registered after 30 seconds, skip the wait — the repo may not have CI configured.
3. If all checks pass, proceed to merge.
4. If any check fails, report the failure and stop — do not retry or bypass.
5. If 5 minutes elapse without all checks completing, report the current status and stop.

Once checks pass (or no CI exists), merge the PR with `gh pr merge <number> --merge`.

If branch protection blocks the merge (e.g. checks passed but another rule blocks), queue auto-merge with `gh pr merge <number> --auto --merge` and continue polling `gh pr view <number> --json state` until the PR merges or 5 minutes total have elapsed.

If in a worktree, use `ExitWorktree` to return to the main working directory.

Then clean up:

```
git checkout main && git pull
git branch -d <branch>
```

Only delete the remote branch manually (`git push origin --delete <branch>`) if the repo does NOT have auto-delete enabled. Check with `gh repo view --json deleteBranchOnMerge -q .deleteBranchOnMerge` — if `true`, skip the remote delete.

## 6. Confirm

Show the user the merged PR URL and a one-line confirmation.
