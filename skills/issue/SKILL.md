---
name: issue
description: >
  Turn a rough idea or complaint into a single well-scoped GitHub issue with context, solution avenues,
  task list, and test plan. For multi-issue feature planning, use /feature instead.
user-invocable: true
argument-hint: >
  [describe the problem or feature in plain language, e.g. "the login button doesn't work on mobile" | "add a /health endpoint"]
allowed-tools: Bash, Read, Glob, Grep, Agent, AskUserQuestion, TaskCreate, TaskUpdate
effort: high
---

# Issue

Turn a rough, informal description into a single, well-structured GitHub issue. The user describes a bug, enhancement, chore, or small feature in whatever words they want — this skill investigates the codebase, fills in technical context, and creates one detailed issue ready for implementation.

**Scope guard:** This skill produces exactly one issue. If the user's description implies multiple features, a phased rollout, or architectural planning, tell them:

> This sounds like it needs multi-issue planning. Use `/feature` instead — it'll break this down into phased issues with a roadmap.

Then stop. Do not attempt to create multiple issues or break work into sub-issues — that's `/feature`'s job.

## 0. Parse the description

`$ARGUMENTS` contains the user's raw description. It might be:

- A bug report: "the button doesn't work", "login is broken on mobile"
- A feature request: "add dark mode", "we need rate limiting"
- A vague complaint: "the dashboard is slow"
- A detailed spec: "add a /health endpoint that returns 200 with uptime"

Accept whatever they give you. Your job is to turn it into something actionable.

## 1. Investigate the codebase

Before writing the issue, understand the relevant code. This makes the issue actually useful — not just a restated complaint.

1. **Identify the area** — based on the description, figure out which part of the codebase is involved. Use `Grep` and `Glob` to find relevant files. For bugs, trace the likely code path. For features, find where the new code would live.

2. **Read the relevant code** — read the files you found. Understand the current behavior, data flow, and any existing patterns that are relevant.

3. **Check for related issues** — run `gh issue list --state open --search "<keywords>"` to see if a similar issue already exists. If one does, tell the user and ask if they want to update the existing issue or create a new one.

4. **Check recent commits** — run `git log --oneline -20` to see if someone has been working in this area recently. Note any relevant context.

Spawn an **Explore agent** if the investigation requires searching across many files or the codebase is large. For simple, localized issues, direct Grep/Glob/Read is fine.

## 2. Scope with Product Manager

Launch a **Product Manager** agent (`subagent_type: "Product Manager"`) with the user's raw description and your investigation findings from step 1. Tell it to:

- Assess the issue's priority and impact from a product perspective
- Identify any user-facing implications or dependencies
- Suggest acceptance criteria that focus on user outcomes, not just technical correctness
- Flag if this is too large for a single issue (suggest `/feature` instead)

Merge its input into the issue draft in step 3 — particularly the acceptance criteria and scope assessment.

## 3. Determine issue type and scope

Based on the investigation, classify the issue:

- **Bug** — something is broken or behaving incorrectly
- **Feature** — new functionality that doesn't exist yet
- **Enhancement** — improvement to existing functionality
- **Chore** — refactoring, dependency updates, CI changes, etc.

Assess the scope:

- **Small** — isolated change, single file or function, < 1 hour of work
- **Medium** — touches a few files, may need tests, < half a day
- **Large** — spans multiple modules, needs design decisions

If the scope is **large**, suggest the user run `/feature` instead for proper multi-issue planning. If they insist on a single issue, proceed — but keep it to one issue.

## 4. Draft the issue

Structure the issue with these sections:

### Title

Short, specific, and prefixed with the type:

- `fix: login button unresponsive on mobile Safari`
- `feat: add dark mode toggle to settings`
- `enhance: improve dashboard query performance`
- `chore: migrate from Jest to Vitest`

### Body

Use this template:

```markdown
## Problem

[What's wrong or what's missing. Be specific — include the user's original complaint
but add technical context from your investigation. For bugs, describe the current
behavior vs expected behavior.]

## Context

[Relevant code paths, files, and architectural context discovered during investigation.
Link to specific files/lines where the issue lives or where changes would go.
Mention any related patterns or conventions in the codebase.]

## Possible approaches

[2-3 solution avenues with brief trade-offs. Don't prescribe one — give the implementer
options to consider.]

### Approach A: [name]
- How: [brief description]
- Pros: [why this is good]
- Cons: [downsides or risks]
- Files likely touched: [list]

### Approach B: [name]
- How: [brief description]
- Pros: [why this is good]
- Cons: [downsides or risks]
- Files likely touched: [list]

## Tasks

- [ ] [Concrete implementation step]
- [ ] [Another step]
- [ ] [Update/add tests for ...]
- [ ] [Update docs if needed]

## Test plan

- [ ] [Specific test case or scenario to verify]
- [ ] [Edge case to cover]
- [ ] [Regression to check — existing behavior that must not break]

## Scope

**Type:** bug | feature | enhancement | chore
**Size:** small | medium | large
```

### Labels

Determine appropriate labels based on what the repo already uses. Run `gh label list` to see available labels. Pick the most relevant ones (type, area, priority). Don't invent labels that don't exist.

## 5. Create the issue

Present the draft to the user before creating:

```
═══════════════════════════════════════
  Issue Draft
═══════════════════════════════════════

  Title: fix: login button unresponsive on mobile Safari
  Labels: bug, frontend, P1

  [full body preview]

═══════════════════════════════════════
  Create this issue? (or suggest changes)
═══════════════════════════════════════
```

Wait for the user to confirm or request changes. Once confirmed:

```bash
gh issue create --title "Title here" --body "$(cat <<'EOF'
Body here
EOF
)" --label "bug,frontend"
```

Only one issue should be created per invocation.

## 6. Confirm

Report the created issue:

```
═══════════════════════════════════════
  Issue Created
═══════════════════════════════════════

  #42 — fix: login button unresponsive on mobile Safari
  URL: https://github.com/owner/repo/issues/42
  Labels: bug, frontend, P1

  Pick it up with: /next 42
═══════════════════════════════════════
```

## Style guidelines

- Follow the standard output format in `_output-format.md`
- Write issues for the implementer, not the reporter. The person fixing this should be able to start working immediately from the issue alone.
- Be specific about file paths and code references — vague issues waste time.
- Keep the tone neutral and technical. Don't editorialize ("this is a terrible bug").
- The "Possible approaches" section is what makes a good issue great. Doing the investigation upfront saves the implementer from re-discovering the same context.
- Task lists should be concrete and ordered. "Implement the fix" is not a task. "Add null check in `handleClick` before dispatching" is.
- Test plans should be specific scenarios, not "test that it works."
