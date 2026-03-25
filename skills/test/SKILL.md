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
