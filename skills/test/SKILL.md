---
name: test
description: >
  Run review agents (Code Reviewer, Reality Checker, Security Engineer, and conditional specialists)
  against uncommitted changes to validate before shipping.
user-invocable: true
argument-hint: [optional focus area, e.g. "security" or "performance"]
allowed-tools: Bash, Read, Glob, Grep, Agent, AskUserQuestion
effort: high
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
- Focus on correctness, maintainability, and test coverage
- **Defer security to the Security Engineer** — do not duplicate security findings
- **Defer performance to the Performance Benchmarker** (if launched) — flag only obvious perf bugs, not deep analysis
- Use its priority markers: blocker, suggestion, nit
- Report findings only — do NOT make changes

### Reality Checker (`subagent_type: "Reality Checker"`)

Tell it to evaluate the implementation against the issue spec or design docs. It should:
- Read the relevant issue (check conversation context or recent git log for issue numbers) and any design docs
- Read the changed files
- Assess: does the implementation match the spec? Any gaps?
- Give an honest verdict — do NOT make changes

### Security Engineer (`subagent_type: "Security Engineer"`)

Tell it to review all uncommitted changes for security vulnerabilities. It should:
- Read every modified and new file
- Focus on: injection flaws, auth/authz issues, data exposure, insecure defaults, dependency risks
- Flag findings as blocker or suggestion
- Report findings only — do NOT make changes

### Conditional agents (launch in the same parallel batch if applicable)

Check the changed files to determine which of these additional agents to include:

- **Accessibility Auditor** (`subagent_type: "Accessibility Auditor"`) — launch if changes touch UI/frontend files (`.tsx`, `.jsx`, `.vue`, `.svelte`, `.html`, CSS, SwiftUI views). Tell it to audit the changed components for WCAG compliance, focus management, and screen reader compatibility.
- **API Tester** (`subagent_type: "API Tester"`) — launch if changes touch API routes, handlers, or endpoint definitions. Tell it to validate request/response contracts, error handling, status codes, and edge cases.
- **Performance Benchmarker** (`subagent_type: "Performance Benchmarker"`) — launch if changes touch database queries, hot loops, data processing, or rendering logic. Tell it to identify potential performance regressions.

If `$ARGUMENTS` contains a focus area (e.g., "security"), tell all agents to weight that area more heavily.

## 4. Summarize findings

Collect results from all agents and present a unified summary. **Deduplicate** — if multiple agents flagged the same issue, keep the most detailed finding and drop the rest.

- **Blockers** — must fix before shipping (from either agent)
- **Suggestions** — should fix, but not blocking
- **Verdict** — ready to ship or needs work

## 5. Fix blockers

If there are blockers, spawn the most appropriate agent(s) to implement the fixes. Choose based on the nature of the issues — e.g., a Software Architect for design/structural problems, a Code Reviewer for correctness issues, or a general-purpose agent for straightforward bug fixes. Use multiple agents in parallel if the blockers span different domains.

Pass the chosen agent(s):

- The full list of blockers and suggestions from both reviewers
- The relevant file paths and what needs to change
- Instructions to fix each issue while preserving existing behavior

After the agent(s) complete, re-run lint and tests to verify the fixes don't introduce regressions.

If no blockers, skip to step 6.

## 6. Final verdict

If fixes were applied in step 5, tell the user to run `/test` again to re-validate.

If no blockers were found (or only suggestions remain), tell the user to `/ship` when ready.
