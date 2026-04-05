---
name: test
description: >
  Validate uncommitted changes with review agents. Default is light mode (lint + tests + Code Reviewer).
  Use "full" for all agents. Auto-escalates for security-sensitive or API changes.
user-invocable: true
argument-hint: >
  ["full" for all agents | optional focus area like "security" or "performance"]
allowed-tools: Bash, Read, Glob, Grep, Agent, AskUserQuestion
effort: high
---

# Test

Validate the current uncommitted changes by running review agents. Has two modes:

- **Light** (default): lint, tests, and a Code Reviewer. Fast, sufficient for most changes.
- **Full** (explicit or auto-escalated): adds Reality Checker, Security Engineer, and conditional specialists.

## 1. Identify what changed

Run `git diff --stat` and `git diff` to understand the full scope of changes. Also check `git status` for new untracked files — read those too.

## 2. Determine mode

**Use full mode if ANY of these are true:**
- `$ARGUMENTS` contains "full"
- `$ARGUMENTS` contains a focus area like "security" or "performance"
- Changed files touch auth, payments, crypto, or API key handling
- New API routes or endpoint handlers were added
- Database migrations or schema changes
- Changes span 8+ files

**Otherwise, use light mode.**

## 3. Run lint and tests

Run the project's linter and test suite. If either fails, report the failures immediately — do not proceed to agent reviews until the basics pass.

## 4. Run review agents

### Light mode

Launch one agent:

**Code Reviewer** (`subagent_type: "Code Reviewer"`)
- Read every modified and new file
- Focus on correctness, maintainability, and obvious bugs
- Use priority markers: blocker, suggestion, nit
- Report findings only — do NOT make changes

### Full mode

Launch these agents **in parallel** (single message, multiple Agent tool calls):

**Code Reviewer** (`subagent_type: "Code Reviewer"`)
- Same as light mode, but defer security to the Security Engineer and performance to the Performance Benchmarker

**Reality Checker** (`subagent_type: "Reality Checker"`)
- Read the relevant issue (check conversation context or recent git log for issue numbers) and any design docs
- Assess: does the implementation match the spec?
- Give an honest verdict — do NOT make changes

**Security Engineer** (`subagent_type: "Security Engineer"`)
- Review all uncommitted changes for security vulnerabilities
- Focus on: injection flaws, auth/authz issues, data exposure, insecure defaults
- Flag findings as blocker or suggestion
- Report findings only — do NOT make changes

**Conditional agents** (launch in the same parallel batch if applicable):
- **Accessibility Auditor** — if changes touch UI files (`.tsx`, `.jsx`, `.vue`, `.svelte`, `.html`, CSS)
- **API Tester** — if changes touch API routes or endpoint definitions
- **Performance Benchmarker** — if changes touch database queries, hot loops, or rendering logic

If `$ARGUMENTS` contains a focus area, tell all agents to weight that area more heavily.

## 5. Summarize findings

Collect results and present a unified summary. **Deduplicate** — if multiple agents flagged the same issue, keep the most detailed finding.

```
═══════════════════════════════════════
  Test — Complete (or Needs Work)
═══════════════════════════════════════

  Mode: light (or full)
  Agents: Code Reviewer [, others if full]

  Blockers:
    ✗ <blocker description> — <file:line>

  Suggestions:
    • <suggestion> — <file:line>

  Done:
    ✓ Lint — clean
    ✓ Tests — 42 passing
    ✓ Code review — 2 suggestions
    [✓ Security review — no issues]
    [✓ Reality check — matches spec]

  Next: /ship — ready to ship
  (or)
  Next: fix N blockers, then re-run /test

═══════════════════════════════════════
```

## 6. Fix blockers

If there are blockers, spawn the most appropriate agent(s) to implement fixes. Pass:
- The full list of blockers
- Relevant file paths and what needs to change
- Instructions to fix while preserving existing behavior

After fixes, re-run lint and tests to verify no regressions.

If no blockers, skip to step 7.

## 7. Final verdict

If fixes were applied:

```
═══════════════════════════════════════
  Test — Complete
═══════════════════════════════════════

  Done:
    ✓ Fixed N blockers
    ✓ Re-ran lint and tests — clean

  Next: /ship when ready

═══════════════════════════════════════
```

## Style guidelines
- Follow the standard output format in `_output-format.md`
