---
name: clean
description: >
  Git repo health checkup and cleanup. Removes stale worktrees and branches that are merged
  or behind main, prunes remotes, runs gc, and checks for repo hygiene issues like forgotten
  stashes, large untracked files, and corruption. Use "now" to skip confirmation.
user-invocable: true
argument-hint: ["now" to skip confirmation and clean immediately]
allowed-tools: Bash, Read, Glob, Grep, Agent
effort: high
---

# Clean

Run a health checkup on the current git repo and clean up stale artifacts.
Run all independent information-gathering steps in parallel where possible.
**Unless `$ARGUMENTS` contains `now`, always confirm with the user before deleting anything.**


## Step 1: Gather repo state (run in parallel)

Run these via Bash. If any fail, note it and skip gracefully.

**Basics:**
- `git rev-parse --is-inside-work-tree` — confirm this is a git repo
- `git remote update --prune` — prune stale remote tracking refs
- `git status --short --branch` — current branch + uncommitted changes
- Identify the default branch (`git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@'` — fall back to `main` or `master`)

**Worktrees:**
- `git worktree list` — all worktrees
- For each non-main worktree:
  - `git -C <path> status --short` — uncommitted changes?
  - `git -C <path> log --oneline @{upstream}..HEAD 2>/dev/null` — unpushed commits?
  - `git -C <path> log --oneline -1 --format='%cr'` — how old is the latest commit?
  - Check if the worktree's branch has been merged into the default branch (`git branch --merged <default-branch> | grep <branch>`)

**Branches:**
- `git branch -a --sort=-committerdate --format='%(refname:short) %(committerdate:relative) %(objectname:short)'` — all branches
- `git branch --merged <default-branch>` — local branches already merged
- `git branch -r --merged <default-branch>` — remote branches already merged
- Check for branches with no remote tracking ref (`git branch -vv | grep ': gone]'`)

**Misc health:**
- `git fsck --no-full 2>&1 | head -20` — check for corruption (quick check only)
- `git count-objects -vH` — repo size stats
- Check for large untracked files: `find . -not -path './.git/*' -type f -size +10M 2>/dev/null | head -10`
- `git stash list` — forgotten stashes

## Step 2: Classify and report

Present findings in these categories. Skip any section with nothing to report.

### Stale Worktrees (cleanup candidates)
A worktree is **stale** if ALL of these are true:
- Its branch is merged into the default branch, OR it has no commits beyond the default branch
- It has no uncommitted changes
- It has no unpushed commits

List each stale worktree with its path, branch, and last commit date.

### Stale Branches (cleanup candidates)
A branch is a cleanup candidate if ANY of these are true:
- It is merged into the default branch (and is not the default branch itself)
- Its remote tracking ref is gone (remote branch was deleted)
- It has had no commits in 4+ weeks and is behind the default branch

List each with name, last commit date, and reason it's stale.

### Healthy Worktrees (keep)
List worktrees that are actively being worked on (have uncommitted changes, unpushed commits, or recent unmerged work). Briefly note why each is being kept.

### Active Branches (keep)
Branches with recent unmerged work. Briefly list them.

### Repo Health
- Disk usage (from `count-objects`)
- Any fsck warnings
- Large untracked files
- Fornowtten stashes (suggest `git stash drop` if old)
- Dangling objects or other issues

## Step 3: Confirm and clean

Present a summary of proposed actions:

1. **Remove stale worktrees** — `git worktree remove <path>` for each
2. **Delete stale local branches** — `git branch -d <branch>` for each
3. **Delete stale remote branches** — `git push origin --delete <branch>` for each (only branches in the user's repo, not upstream forks)
4. **Prune and GC** — `git gc --auto` to clean up loose objects

If `now` was passed, execute all proposed actions immediately. Otherwise, ask the user to confirm before executing — accept "yes", "all", or let them cherry-pick which items to clean.

After cleanup, run `git worktree list` and `git branch -a` to show the final state.

## Step 4: Summary

Report what was cleaned:
- Number of worktrees removed
- Number of branches deleted (local + remote)
- Disk space reclaimed (if `git gc` ran, compare before/after `count-objects`)
- Any issues that need manual attention

## Style guidelines
- Be direct and scannable — use bullet points, not paragraphs
- Use relative time ("3 days ago") not absolute dates
- Bold anything that needs attention
- If everything is already clean, say so briefly — don't pad the report
