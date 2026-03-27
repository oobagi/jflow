---
name: feature
description: >
  Add new features to an existing project while matching its established style.
  Analyzes the repo's architecture, conventions, and patterns, then plans, scopes,
  and creates issues for the new features. Optionally kicks off implementation.
  Works whether the repo was scaffolded with /setup or not.
user-invocable: true
argument-hint: >
  [describe the features you want, e.g. "add dark mode and a settings page" | "webhook support and retry logic"]
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion, Skill, TaskCreate, TaskUpdate
effort: high
---

# Feature

Add new features to an existing project while respecting its established conventions. Analyzes the repo to understand how code is written, structured, tested, and shipped — then plans and scopes features that fit naturally into the codebase.

## 0. Parse input

`$ARGUMENTS` contains the user's feature descriptions. They might say:

- Specific features: "add dark mode and a notification system"
- Vague goals: "needs to support webhooks"
- A mix: "add Stripe billing, also the dashboard needs charts"
- Nothing (empty) — ask them

If `$ARGUMENTS` is empty, ask the user with `AskUserQuestion`:

> **What features do you want to add?** Describe them however you like — a bullet list, a paragraph, a vague wish. I'll figure out the details by reading the codebase.

## 1. Analyze the repo's style

This is the critical step. Before planning any features, deeply understand how this codebase works. Run steps 1a and 1b in parallel.

### 1a. Codebase scan (Explore agent)

Spawn an **Explore agent** (`subagent_type: "Explore"`, thoroughness: "very thorough") to build a style profile. The agent should:

**Architecture & structure:**
- Map the directory layout and module boundaries
- Identify the tech stack, framework, and key dependencies
- Note the entry points and how code flows through the system
- Check for `AGENTS.md`, `CLAUDE.md`, `README.md` — read them for documented conventions

**Code conventions:**
- Naming style (camelCase, snake_case, PascalCase for what)
- File naming and organization patterns (one component per file? grouped by feature or type?)
- Import/export patterns
- How errors are handled (Result types, try/catch, error boundaries, custom error classes)
- How types/interfaces are defined and where they live
- Comment style and density

**Testing patterns:**
- Where tests live (`__tests__/`, `*.test.ts`, `tests/`, inline)
- Testing framework and style (unit vs integration, mocking approach, fixtures)
- Test naming conventions
- Coverage expectations (are there coverage configs?)

**API & data patterns:**
- How routes/endpoints are defined
- Database access patterns (ORM, raw queries, repository pattern)
- How state is managed (Redux, Zustand, Context, signals, etc.)
- How configs and environment variables are handled

**Existing patterns to replicate:**
- Find 2-3 recently added features (check `git log --oneline -30`) and note the file patterns they follow — this is the template for new features

Return a structured **Style Profile** — a concise document covering each of the above areas with specific examples from the codebase. Include file paths and line numbers for key examples.

### 1b. Recent activity scan

Run these Bash commands in parallel:

```bash
# Recent commits — what's been worked on
git log --oneline -20

# Recent feature branches (merged)
git log --merges --oneline -10

# Check for ROADMAP.md
cat ROADMAP.md 2>/dev/null || echo "NO_ROADMAP"

# Check for existing issues
gh issue list --state open --limit 10 2>/dev/null || echo "NO_ISSUES"

# Check for existing labels
gh label list 2>/dev/null || echo "NO_LABELS"
```

## 2. Plan features with Software Architect

Launch a **Software Architect** agent (`subagent_type: "Software Architect"`) with:

- The user's feature descriptions from `$ARGUMENTS`
- The Style Profile from step 1a
- The recent activity context from step 1b

The architect should:

1. **Break down** each feature into concrete, implementable units of work. A "notification system" might become: notification model + storage, notification triggers, notification delivery, notification UI, notification preferences.

2. **Order by dependency** — which pieces need to exist before others? Group into phases if there are natural boundaries.

3. **Map to the codebase** — for each unit of work, specify:
   - Which existing files need modification (with reasons)
   - Which new files need creation (following the repo's naming/location patterns)
   - Which existing patterns to replicate (reference specific files as templates)
   - Database changes needed (migrations, new tables/columns)
   - New dependencies required (if any)

4. **Identify risks** — flag anything that might conflict with existing code, require breaking changes, or need architectural decisions the user should weigh in on.

5. **Estimate scope** — small / medium / large per item, following the same scale as `/issue`

## 3. Present the plan

Show the user a clear, scannable plan. Format:

```
═══════════════════════════════════════
  Feature Plan
═══════════════════════════════════════

  Project: <project name>
  Style: <one-line summary, e.g. "Next.js App Router + Prisma + Tailwind, feature-grouped modules">
  Features: N units of work across M phases

  Phase 1: <name>
    1. <title> [scope] — <one-line description>
       Files: <key files touched/created>
    2. ...

  Phase 2: <name>
    3. ...

  ── Risks / decisions needed ──
  • <risk or question>
  • ...

═══════════════════════════════════════
  Proceed? [yes / adjust / stop]
═══════════════════════════════════════
```

Wait for user confirmation. If they say "adjust", ask what to change and revise the plan.

## 4. Create GitHub issues

Once the user confirms, create a detailed issue for each unit of work using `gh issue create`. Follow the same issue format as `/issue`:

```markdown
## Context

[Why this feature matters and how it fits into the existing system. Reference
the style profile — explain which existing patterns this follows.]

## Proposed approach

[Concrete implementation plan that matches the repo's conventions.
Name specific files, functions, patterns to replicate. Reference
existing code as templates where applicable.]

### Style notes

[Explicit callouts for how this feature should match existing conventions:
- File location and naming
- Pattern to follow (reference existing file as template)
- Testing approach (match existing test style)
- Error handling (match existing pattern)]

## Tasks

- [ ] [Concrete step following the repo's patterns]
- [ ] [Another step]
- [ ] [Write tests matching existing test style]
- [ ] [Update docs if needed]

## Acceptance criteria

- [ ] [Specific verifiable outcome]
- [ ] [Integrates with existing X without breaking Y]
- [ ] [Tests pass, lint clean]

## References

- Style template: `path/to/similar/existing/feature`
- Related: #N (dependency)
```

Create labels for phases if they don't exist (`phase-N` pattern). Apply relevant existing labels too.

Create issues in dependency order and cross-reference dependencies in each issue body.

## 5. Update the roadmap

### If ROADMAP.md exists:

Append a new phase (or phases) to the existing roadmap, following the existing format exactly. Use the same markdown structure, checkbox style, and link format as the existing items.

```markdown
## Phase N: <Feature Name>

- [ ] [#X <Title>](https://github.com/<user>/<repo>/issues/X)
- [ ] [#Y <Title>](https://github.com/<user>/<repo>/issues/Y)
```

### If no ROADMAP.md:

Create one from scratch following the `/setup` format:

```markdown
# Roadmap

This roadmap orders open work by leverage and dependency. Each item links to a GitHub issue with full details. Checkboxes are updated automatically when issues are closed or reopened.

## Phase 1: <name>

- [ ] [#X <Title>](https://github.com/<user>/<repo>/issues/X)
- [ ] [#Y <Title>](https://github.com/<user>/<repo>/issues/Y)
```

### If AGENTS.md exists:

Check if the planned features change the project structure or conventions significantly enough to warrant an AGENTS.md update. If so, note it — the actual update happens when features are implemented and `/docs` runs.

## 6. Commit and push

```bash
git add ROADMAP.md
git commit -m "$(cat <<'EOF'
roadmap: add <feature-area> items (#X, #Y, #Z)

Planned N new features across M phases based on codebase analysis.
Each item has a detailed issue spec following existing project conventions.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
git push
```

## 7. Summary and next steps

```
═══════════════════════════════════════
  Features Planned
═══════════════════════════════════════

  Issues created: N
    • #X — <title> [scope]
    • #Y — <title> [scope]
    • ...

  Roadmap: updated (Phase N added)
  Style matched: <key conventions identified>

  Start building:
    /next <first-issue-number>

  Or let autopilot handle it:
    /autopilot phase-<N>

═══════════════════════════════════════
```

## Style guidelines

- The entire point of this skill is **style matching**. Every issue created must reference specific existing files as templates. Generic issues that ignore the repo's conventions defeat the purpose.
- When in doubt about a convention, find 3 examples in the codebase and follow the majority pattern.
- Don't over-scope. If the user says "add dark mode", that's one feature with a few tasks — not a 15-issue epic.
- Keep the plan practical. If the repo is a simple Express app, don't propose a microservices architecture for the new features.
- The style profile is the foundation. If the explore agent returns a weak profile (missing key areas), run targeted follow-up searches before planning.
- Reference real files, not abstract patterns. "Follow the pattern in `src/features/auth/AuthProvider.tsx`" is useful. "Follow React best practices" is not.
