---
name: qa
description: >
  Generate a hands-on manual testing guide for the current project, or run the tests automatically.
  Analyzes the codebase, docs, and git history to produce step-by-step instructions the user can
  follow to exercise every feature. Pass "latest" to scope to the last featureset. Pass "auto" to
  execute the tests autonomously using Playwright, CLI, and API calls — then report results.
user-invocable: true
argument-hint: >
  [latest — last featureset only | auto — execute tests autonomously | combine: "auto latest"]
allowed-tools: Bash, Read, Glob, Grep, Agent, mcp__playwright__browser_navigate, mcp__playwright__browser_snapshot, mcp__playwright__browser_click, mcp__playwright__browser_fill_form, mcp__playwright__browser_take_screenshot, mcp__playwright__browser_press_key, mcp__playwright__browser_select_option, mcp__playwright__browser_hover, mcp__playwright__browser_evaluate, mcp__playwright__browser_console_messages, mcp__playwright__browser_network_requests, mcp__playwright__browser_tabs, mcp__playwright__browser_navigate_back, mcp__playwright__browser_close, mcp__playwright__browser_resize, mcp__playwright__browser_wait_for, mcp__playwright__browser_install, mcp__playwright__browser_run_code, mcp__playwright__browser_drag, mcp__playwright__browser_handle_dialog, mcp__playwright__browser_file_upload
effort: high
---

# QA — Testing Guide & Automated Execution

Produce a clear, actionable testing guide that verifies the software works. In **manual mode** (default), write the guide for the user to follow. In **auto mode**, execute every test yourself using Playwright for UI, Bash for CLI/API, and report pass/fail results.

## 0. Parse arguments

Check `$ARGUMENTS` for flags (combine freely):

| Flag | Effect |
|------|--------|
| `latest` or `last` | Scope to the most recent featureset only |
| `auto` | Execute tests autonomously instead of writing a guide |

Examples: `/qa`, `/qa latest`, `/qa auto`, `/qa auto latest`

---

## 1. Gather project context (run in parallel where possible)

### 1a. Always gather

- **README / CLAUDE.md** — read for project overview, tech stack, and setup instructions.
- **Package manifest** — read `package.json`, `Cargo.toml`, `pyproject.toml`, `go.mod`, or equivalent to understand dependencies, available scripts, and entry points.
- **Source structure** — use Glob to map the top-level directory tree (2 levels deep). Identify key directories (routes, components, services, models, CLI entrypoints, etc.).
- **Existing tests** — Glob for test files (`**/*test*`, `**/*spec*`, `**/tests/**`). Read a few to understand what's already covered and how tests are structured.
- **ROADMAP / CHANGELOG** — if present, read them to understand feature groupings and milestones.

### 1b. If scope is "latest"

- Run `git log --oneline -30` to see recent history.
- Identify the most recent logical featureset: look for the last merge commit, or the last cluster of related commits on the current branch. Read the commit messages and diffs (`git diff <boundary>..HEAD --stat` then `git diff <boundary>..HEAD` for changed files) to understand exactly what was added or changed.
- Read every file that was modified in that featureset.

### 1c. If scope is "full"

- Use an **Explore agent** (`subagent_type: "Explore"`) to do a thorough analysis of the codebase:
  - All user-facing features (CLI commands, API endpoints, UI screens, exported functions)
  - Configuration options and environment variables
  - Integration points (databases, external APIs, file I/O)
  - Edge cases visible in the code (error handling paths, fallback behaviors, feature flags)

## 2. Identify features to test

Organize discovered features into logical groups. Each group should represent a cohesive area of functionality (e.g., "Authentication", "Data export", "CLI commands", "Webhook handling").

For each feature, note:
- What it does (one line)
- Where the code lives (file paths)
- Any setup or prerequisites (env vars, database state, test fixtures)
- Whether existing automated tests cover it (and what they miss)

**Classify each test case by execution method:**

| Type | How to test |
|------|-------------|
| `cli` | Run a shell command, check exit code and stdout/stderr |
| `api` | Send an HTTP request via `curl`, check status code and response body |
| `ui` | Navigate with Playwright, interact with elements, verify visual state |
| `db` | Run a query or check file/state after an action |

## 3. Build the test plan

Structure the plan as a list of test cases grouped by feature. Each test case must include:

1. **ID** — short identifier (e.g., `AUTH-01`, `CLI-03`)
2. **What to test** — one-line description
3. **Type** — `cli`, `api`, `ui`, or `db`
4. **Steps** — exact commands, URLs, or actions
5. **Expected result** — what success looks like (exit codes, output patterns, UI state)
6. **Edge cases** — variations to try

Prioritize by risk: most critical and most likely-to-break first.

### If scope is "latest"

- Only include test cases for features touched by the latest featureset.
- Add a **Regression Checks** section: smoke tests for adjacent features that could be affected.

---

## 4. Manual mode (default — no `auto` flag)

Write the full test plan to the user as formatted output, following these rules:

### Header
- Project name, scope (full or latest), date, tech stack, prerequisites

### Setup
- How to get running locally, seed data, run existing test suite as baseline

### Test Sections (one per feature group)
- Numbered test cases with steps, expected results, and edge cases
- Code blocks for every command or expected output

### Automated Test Gaps
- Brief list of features with no automated coverage

### Formatting
- Write for a human at their terminal
- Code blocks for commands and outputs
- Bold critical items (required env vars, destructive commands, gotchas)
- Relative file paths from project root
- Bullets and numbered steps, not paragraphs

**Stop here in manual mode. Do not proceed to step 5.**

---

## 5. Auto mode (`auto` flag present)

Execute every test case from the plan yourself. You are now the tester.

### 5a. Environment setup

1. Determine the project's local dev server command (from package.json scripts, README, or convention — e.g., `npm run dev`, `cargo run`, `python manage.py runserver`).
2. Start the dev server in the background via Bash (`run_in_background: true`). Wait a few seconds for it to be ready.
3. Identify the local URL (typically `http://localhost:3000`, `http://localhost:8080`, etc. — check the dev server output or config).
4. For UI tests: ensure Playwright MCP is available. If `mcp__playwright__browser_navigate` is not accessible, fall back to `curl` for any HTTP-based checks and note that UI visual tests were skipped.

### 5b. Execute test cases

Work through each test case in order. For each one:

**CLI tests (`cli` type):**
- Run the command via Bash
- Capture stdout, stderr, and exit code
- Compare against expected result

**API tests (`api` type):**
- Run `curl` commands via Bash (use `-s -w "\n%{http_code}"` to capture status codes)
- Parse and validate response body and status code against expected result

**UI tests (`ui` type):**
- Use `mcp__playwright__browser_navigate` to open the target URL
- Use `mcp__playwright__browser_snapshot` to get the accessibility tree and verify page structure, text content, and element presence
- Use `mcp__playwright__browser_click`, `mcp__playwright__browser_fill_form`, `mcp__playwright__browser_press_key` etc. to interact with the UI
- Use `mcp__playwright__browser_take_screenshot` to capture visual state for layout/theme verification
- Use `mcp__playwright__browser_console_messages` to check for JS errors
- Use `mcp__playwright__browser_network_requests` to verify API calls made by the UI
- After interactions, use `mcp__playwright__browser_snapshot` again to verify the UI updated correctly

**DB/state tests (`db` type):**
- Run the triggering action first, then verify the side effect (check file contents, run a DB query, read logs)

**For each test case, record:**
- **Status**: PASS, FAIL, or SKIP (with reason)
- **Actual result**: what actually happened (brief)
- **Evidence**: relevant output snippet, screenshot reference, or error message
- **Notes**: anything unexpected, even if the test passed

### 5c. Edge case execution

For each test case that has edge cases listed:
- Execute at least the highest-risk edge case variation
- Record results the same way

### 5d. Visual and theme checks (UI projects only)

If the project has a UI:
1. Navigate to each major screen/page
2. Take a screenshot via `mcp__playwright__browser_take_screenshot`
3. Check the accessibility snapshot for:
   - Correct heading hierarchy
   - All interactive elements are labeled
   - No broken/empty elements
4. If the project has dark/light theme support, toggle it and screenshot both
5. Use `mcp__playwright__browser_resize` to check responsive behavior at:
   - Desktop: 1280x800
   - Tablet: 768x1024
   - Mobile: 375x812
6. Report any layout breaks, overflow issues, or missing responsive behavior

### 5e. Cleanup

- Stop the dev server if you started one (kill the background process)
- Close the Playwright browser via `mcp__playwright__browser_close`
- Undo any test data or state changes if possible

## 6. Auto mode results report

After all tests execute, present the results:

### Summary

```
Total: XX | Passed: XX | Failed: XX | Skipped: XX
```

### Results by feature group

For each group, show a table:

| ID | Test | Status | Notes |
|----|------|--------|-------|
| AUTH-01 | Login with valid creds | PASS | |
| AUTH-02 | Login with bad password | FAIL | Got 500 instead of 401 |

### Failures (detail)

For each FAIL, show:
- Test ID and description
- Expected vs. actual result
- Error output or screenshot reference
- Suggested fix or investigation path (file path + what to look at)

### Visual report (if UI tests ran)

- List of screenshots taken and what they show
- Any layout/theme/responsive issues found

### Verdict

- **Ship it** — all tests pass, no visual issues
- **Needs work** — list the blockers that must be fixed
- **Partial** — some tests skipped (explain why), passing tests look good
