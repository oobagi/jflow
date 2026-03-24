---
name: test
description: Run Code Reviewer and Reality Checker agents against uncommitted changes to validate before shipping.
user_invocable: true
argument-hint: [optional focus area, e.g. "security" or "performance"]
---

# Test

Validate the current uncommitted changes by running review agents in parallel. Stop and report findings so the user can decide whether to fix or ship.

## 1. Identify what changed

Run `git diff --stat` and `git diff` to understand the full scope of changes. Also check `git status` for new untracked files — read those too.

## 2. Run lint and tests

Run the project's linter and test suite. If either fails, report the failures immediately — do not proceed to agent reviews until the basics pass.

## 3. Run review agents in parallel

Launch these agents **in parallel** (single message, multiple Agent tool calls):

### Code Reviewer (`subagent_type: "Code Reviewer"`)

Tell it to review all uncommitted changes. It should:
- Read every modified and new file
- Focus on correctness, security, maintainability, performance, and test coverage
- Use its priority markers: blocker, suggestion, nit
- Report findings only — do NOT make changes

### Reality Checker (`subagent_type: "Reality Checker"`)

Tell it to evaluate the implementation against the issue spec or design docs. It should:
- Read the relevant issue (check conversation context or recent git log for issue numbers) and any design docs
- Read the changed files
- Assess: does the implementation match the spec? Any gaps?
- Give an honest verdict — do NOT make changes

If `$ARGUMENTS` contains a focus area (e.g., "security"), tell both agents to weight that area more heavily.

## 4. Summarize findings

Collect results from both agents and present a unified summary:

- **Blockers** — must fix before shipping (from either agent)
- **Suggestions** — should fix, but not blocking
- **Verdict** — ready to ship or needs work

If there are blockers, fix them. Then tell the user to run `/test` again or `/ship` if clean.

If no blockers, tell the user to `/ship` when ready.
