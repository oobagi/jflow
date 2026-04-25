---
name: ship
description: Branch, commit, PR, merge, and cleanup. Use when work is ready to ship.
user-invocable: true
allowed-tools: Bash, Read, Glob, Grep
effort: medium
---

# Ship

Run these steps in order. Stop and report if any step fails.

## 1. Pre-flight checks

If the project has a lint script, run it. If the project has a build script, run it. Fix any errors before continuing.

## 2. Review changes

Run `git status` and `git diff --stat` to see what changed. Review the changes to understand what's being shipped.

**Dirty-tree guard (critical):** If `git status --porcelain` lists files that are *not* part of what the user is asking you to ship (e.g. unrelated edits to skill files, config, notes, or work-in-progress in other modules), STOP and ask how to handle them:
- **(a) Include** them in this ship (if they belong)
- **(b) Stash** them (`git stash push -m "pre-ship: unrelated WIP"`) and restore after merge
- **(c) Commit separately** on a different branch first
- **(d) Abort**

**Never** use `git reset --hard`, `git checkout --`, `git clean -f`, or `git restore --source` to "clean up" the tree before branching. Those commands destroy uncommitted work — and orphaned edits on `main` are work. If you need a clean tree, stash; never reset.

## 2.5. Identify the issue(s) being shipped

**Do this before writing any commit or PR text.** Missing `Closes #N` is the single most common way shipped work stays marked open on the roadmap.

Run `gh issue list --state open --limit 50 --json number,title,labels` and match the current work against open issues by title, scope, and keywords. Also check:

- Branch name (e.g. `fix/72-add-chord-flow` → #72)
- Existing commit messages on the branch (`git log main..HEAD --oneline`)
- Files touched vs. issue descriptions

If one or more open issues clearly match, note their numbers — they go in the PR title as `(Closes #N)` and in the commit as `Fix #N:`. Multiple issues: `(Closes #18, Closes #19)`.

If the match is ambiguous, ask the user which issue(s) this ships before opening the PR. Do not guess silently. If no issue matches (genuinely new work), proceed without a `Closes` reference.

## 3. Create branch and commit

If already on a feature branch (not `main` or `master`), skip branch creation. Otherwise, create a descriptive branch *from the current HEAD* (do NOT `git checkout main && git reset --hard origin/main` — that wipes any uncommitted edits in the working tree). Name it descriptively (e.g., `fix/play-pause-ui-state`, `feat/wasd-controls`).

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

The PR title **must** include `(Closes #N)` for every issue identified in step 2.5 — e.g., `feat: add WASD controls (Closes #42)` or `feat: mods overlays (Closes #18, Closes #19)`. GitHub only auto-closes issues whose numbers appear with a closing keyword (`Closes`, `Fixes`, `Resolves`) in the merged PR title or body. Putting the number in the body without the keyword does nothing.

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

Show the final result using the standard output format:

```
═══════════════════════════════════════
  Ship — Complete
═══════════════════════════════════════

  PR: https://github.com/owner/repo/pull/N

  Done:
    ✓ Lint — clean
    ✓ Created branch <branch-name>
    ✓ Committed: <commit title>
    ✓ Opened PR #N
    ✓ CI passed
    ✓ Merged PR #N

  Next: done — merged and cleaned up

═══════════════════════════════════════
```

If the ship failed at any step:

```
═══════════════════════════════════════
  Ship — Stopped
═══════════════════════════════════════

  Done:
    ✓ Lint — clean
    ✓ Created branch <branch-name>
    ✓ Opened PR #N
    ✗ CI failed — <reason>

  Fix: <what went wrong>
  Resume: address the CI failure, then re-push

═══════════════════════════════════════
```

## Style guidelines
- Follow the standard output format in `_output-format.md`
