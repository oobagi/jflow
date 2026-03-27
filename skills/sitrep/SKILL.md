---
name: sitrep
description: >
  Situation report for the current project. Use when returning after being away from a project
  or Claude Code session, when you can't remember what was done, or to orient in a fresh repo.
  Shows recent activity, branch health, stale worktrees, uncommitted work, and project context.
user-invocable: true
allowed-tools: Bash, Read, Glob, Grep, Agent
effort: high
---

# Situation Report

Generate a concise but thorough situation report for the current working directory.
Run all independent information-gathering steps in parallel where possible.

## Step 1: Gather raw data (run in parallel)

Run these commands via Bash to collect project state. If any fail (e.g. not a git repo), note it and skip gracefully.

**Git basics:**
- `git rev-parse --is-inside-work-tree` — confirm this is a git repo
- `git branch -a --sort=-committerdate --format='%(refname:short) %(committerdate:relative) %(objectname:short)'` — all branches sorted by recency
- `git status --short --branch` — current branch + uncommitted changes
- `git stash list` — any stashed work
- `git log --all --oneline --graph --decorate -20` — recent history across all branches

**Worktrees:**
- `git worktree list` — all worktrees
- For each worktree (other than the main one), check if it has uncommitted changes (`git -C <path> status --short`) and how its HEAD compares to its upstream (`git -C <path> log --oneline -3`). Flag worktrees that have NO new changes beyond their base as **stale**.

**Project context:**
- Check for `CLAUDE.md` or `.claude/CLAUDE.md` in the project root — read it for project context
- Check for `README.md` or `README` — read the first ~50 lines for project summary
- Check for `.claude/plans/*.md` — read any recent plans to understand what was being worked on
- Check the project memory directory if it exists (`.claude/projects/` path matching this working dir under `~/.claude/projects/`)

## Step 2: Synthesize the report

Present findings in this structure. Skip any section that has nothing to report.

### Current State
- What branch you're on, any uncommitted/staged changes, stashes

### Recent Activity
- Summarize the last ~10 commits (across branches). Who did what, when. Group by branch if multiple branches have recent work.

### Branch Health
- List active branches (commits in last 2 weeks)
- Flag stale branches (no commits in 2+ weeks) — suggest cleanup if appropriate
- Note any branches that appear to be unmerged feature work

### Worktrees
- List all worktrees with their branch and status
- Highlight any that are **stale** (no uncommitted changes, no unpushed commits) — these are cleanup candidates

### In-Progress Plans
- If `.claude/plans/` has files, briefly summarize what each plan was about (read the title/first few lines)
- Note which plans look active vs completed

### Project Overview
- Only include this if the repo seems unfamiliar (no prior plans, no project memory, or user is likely new to it)
- Brief summary from README/CLAUDE.md: what is this project, key tech, how to run it

## Style guidelines
- Be direct and scannable — use bullet points, not paragraphs
- Use relative time ("3 days ago") not absolute dates
- Bold anything that needs attention (stale worktrees, uncommitted work, failing CI)
- If everything looks clean, say so briefly — don't pad the report
- Follow the standard output format in `_output-format.md`
- Wrap the entire report in the standard box: header `Sitrep — Complete`, body with all sections, and a `Next:` line with your suggestion of what to do
