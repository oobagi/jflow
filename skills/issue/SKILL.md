---
name: issue
description: >
  Turn a rough idea into GitHub issues. Auto-scales: small requests get one lean issue,
  big requests get a multi-issue breakdown. Replaces both /issue and /feature.
user-invocable: true
argument-hint: >
  [describe the problem or feature, e.g. "fix the login bug" | "add dark mode" | "build a notification system"]
allowed-tools: Bash, Read, Glob, Grep, Agent, AskUserQuestion, Write, Edit, Skill, TaskCreate, TaskUpdate
effort: high
---

# Issue

Turn a rough description into GitHub issues. This skill auto-scales based on what the user asks for — a one-line bug gets a lean issue, a multi-part feature gets a phased breakdown.

## 0. Parse and classify

`$ARGUMENTS` contains the user's description. Read it and classify the **scale**:

- **Small** — isolated bug fix, config change, single endpoint, typo, chore. One file or function. → **One lean issue** (step 2a)
- **Medium** — a self-contained feature or enhancement that touches a few files but is still one unit of work. → **One standard issue** (step 2b)
- **Large** — spans multiple modules, needs phased work, or the user explicitly lists multiple features. → **Multi-issue breakdown** (step 2c)

**How to decide:** If you can describe the work in one sentence and it touches ≤3 files, it's small. If it's one feature but needs design decisions, it's medium. If the user says "and" between distinct features or the work has clear phases/dependencies, it's large.

When in doubt, go smaller. A lean issue is always better than an over-scoped one.

## 1. Investigate

Before writing anything, understand the relevant code:

1. **Find the area** — use `Grep` and `Glob` to locate relevant files.
2. **Read the code** — understand current behavior, patterns, conventions.
3. **Check for duplicates** — `gh issue list --state open --search "<keywords>"`. If a duplicate exists, tell the user.

For **small** issues, this can be 2-3 quick searches. For **large** issues, spawn an **Explore agent** for a thorough scan.

## 2a. Small issue (lean format)

Create a single issue with minimal ceremony:

### Title
Short, prefixed: `fix: ...`, `feat: ...`, `enhance: ...`, `chore: ...`

### Body

```markdown
## Problem
[1-2 sentences. What's wrong or what's needed.]

## Context
[Which file(s) and why. Link to the specific code.]

## Tasks
- [ ] [Concrete step]
- [ ] [Another step if needed]
```

That's it. No approaches section, no test plan, no scope metadata. Keep it tight.

## 2b. Medium issue (standard format)

One issue with enough context for implementation:

### Title
Same prefix convention.

### Body

```markdown
## Problem
[What's wrong or missing. Include technical context from investigation.]

## Context
[Relevant code paths, files, patterns. Be specific with paths and line references.]

## Approach
[One recommended approach. Brief — what to do and which files to touch.
Only include alternatives if there's a genuine trade-off worth discussing.]

## Tasks
- [ ] [Concrete implementation step]
- [ ] [Another step]
- [ ] [Tests if the change warrants them]

## Acceptance criteria
- [ ] [Specific verifiable outcome]
- [ ] [Another if needed]
```

## 2c. Large issue (multi-issue breakdown)

For work that needs multiple issues, break it down:

1. **Analyze the repo** — spawn an **Explore agent** (`thoroughness: "very thorough"`) to understand architecture, conventions, and patterns. Focus on: directory layout, how similar features are structured, testing patterns, and recent commit style.

2. **Break down the work** — split into concrete, implementable issues. Each issue should be independently shippable. Order by dependency.

3. **Present the plan** before creating anything:

```
═══════════════════════════════════════
  Issue — Plan
═══════════════════════════════════════

  Breakdown: N issues

  1. <title> [small/medium] — one-line description
  2. <title> [small/medium] — one-line description
  3. ...

  Dependencies: #1 → #2 (if any)

═══════════════════════════════════════
  Create these issues? (or adjust)
═══════════════════════════════════════
```

Wait for confirmation. Then create each issue using the **small** or **medium** format above (match to scope of each individual issue). Cross-reference dependencies in issue bodies.

4. **Update ROADMAP.md** if it exists — append the new issues following the existing format. If it doesn't exist and there are 3+ issues, create one.

5. **Commit and push** the roadmap change.

## 3. Create the issue(s)

For **small** and **medium**: show a brief draft preview, then create with `gh issue create`. Apply labels from `gh label list` — don't invent labels.

For **large**: create after the user confirms the plan (step 2c).

## 4. Confirm

```
═══════════════════════════════════════
  Issue — Created
═══════════════════════════════════════

  #42 — fix: login button unresponsive on mobile Safari
  URL: https://github.com/owner/repo/issues/42

  Pick it up with: /next 42

═══════════════════════════════════════
```

For multi-issue:

```
═══════════════════════════════════════
  Issues — Created
═══════════════════════════════════════

  Created N issues:
    • #42 — feat: notification model + storage
    • #43 — feat: notification triggers
    • #44 — feat: notification UI

  Roadmap updated.
  Start with: /next 42

═══════════════════════════════════════
```

## Style guidelines

- Follow the standard output format in `_output-format.md`
- **Scale to the request.** A one-liner bug should produce a one-paragraph issue. Resist the urge to over-investigate or over-document simple things.
- Be specific about file paths. Vague issues waste time.
- Task lists should be concrete. "Implement the fix" is not a task.
- When breaking down large work, each issue should stand alone — someone should be able to pick it up without reading all the other issues.
