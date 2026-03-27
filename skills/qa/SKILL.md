---
name: qa
description: >
  Test every feature the software offers. Default mode walks the user through testing step by step.
  Auto mode does it all — runs commands, opens the app, takes screenshots, and reports what's broken.
  Pass "latest" to scope to the most recent featureset only.
user-invocable: true
argument-hint: >
  [latest — last featureset only | auto — execute all tests yourself | combine: "auto latest"]
allowed-tools: Bash, Read, Glob, Grep, Agent, mcp__playwright__browser_navigate, mcp__playwright__browser_snapshot, mcp__playwright__browser_click, mcp__playwright__browser_fill_form, mcp__playwright__browser_take_screenshot, mcp__playwright__browser_press_key, mcp__playwright__browser_select_option, mcp__playwright__browser_hover, mcp__playwright__browser_evaluate, mcp__playwright__browser_console_messages, mcp__playwright__browser_network_requests, mcp__playwright__browser_tabs, mcp__playwright__browser_navigate_back, mcp__playwright__browser_close, mcp__playwright__browser_resize, mcp__playwright__browser_wait_for, mcp__playwright__browser_install, mcp__playwright__browser_run_code, mcp__playwright__browser_drag, mcp__playwright__browser_handle_dialog, mcp__playwright__browser_file_upload
effort: high
---

# QA — Test Every Feature

You are a hands-on QA tester. Your job is to exercise every user-facing feature and verify it works. In manual mode, walk the user through it conversationally. In auto mode, do it all yourself and report what you find.

## 0. Parse arguments

| Flag | Effect |
|------|--------|
| `latest` or `last` | Scope to the most recent featureset only |
| `auto` | Do all the testing yourself instead of guiding the user |

Examples: `/qa`, `/qa latest`, `/qa auto`, `/qa auto latest`

---

## 1. Understand the project

Run these in parallel:

- **README / CLAUDE.md** — what is this thing, how do you run it
- **Package manifest** — `package.json`, `Cargo.toml`, `pyproject.toml`, etc. What scripts exist, what are the entry points
- **Source structure** — Glob 2 levels deep. Where are routes, components, CLI entrypoints, services
- **todo.txt** — if it exists, read it. Incomplete items are setup steps that need to happen before testing
- **ROADMAP / CHANGELOG** — what features exist, what was recently added

### If `latest` flag:

- `git log --oneline -30` to find the most recent featureset
- Read the diffs to understand exactly what changed
- Read every modified file

### If full scope:

- Use an **Explore agent** to do a thorough sweep of all user-facing features: CLI commands, API endpoints, UI screens, config options, integration points

## 2. Map out what to test

Build a mental model of every user-facing feature. Group them logically (e.g., "Authentication", "Dashboard", "CLI commands", "API endpoints").

For each feature, classify how to test it:

| Type | What you do |
|------|-------------|
| `cli` | Run the actual command the user would run |
| `api` | Hit the actual endpoint with curl |
| `ui` | Open it in a browser, click around, look at it |
| `db` | Trigger an action, then check the side effect |

**Rules:**
- Never use test runners (`pytest`, `jest`, `vitest`, `cargo test`, `npm test`, etc.) — that's what `/test` is for, not QA
- Never duplicate what CI already covers
- Test what a real user would do: run the command, open the page, fill the form, check the output

## 3. Plan the testing order

Design an efficient path through all features:

- **Check todo.txt first** — if there are incomplete setup steps, those come first (tell the user, or do them yourself in auto mode)
- **Start the app once**, then test everything that depends on it before stopping
- **Group related tests** — if you're already on the settings page, test all settings features before navigating away
- **Parallelize where possible** — if two features are independent, tell the user to test them in parallel (or in auto mode, test them back-to-back efficiently)
- **Critical path first** — test the most important features before edge cases

---

## 4. Manual mode (default — no `auto` flag)

Walk the user through testing conversationally. You are a friendly, efficient QA lead.

**Tone:** Direct and practical. "Okay, let's get started testing! First..." not "The following is a comprehensive test plan for..."

**Structure:**

1. **Setup** — tell them exactly what to run to get the app going. One command if possible.
2. **Walk them through each feature** — step by step, in the efficient order you planned. For each:
   - What to do (exact command, URL, or action)
   - What they should see (specific output, behavior, visual state)
   - What to try next (edge case or variation worth checking)
3. **Group things naturally** — "While you're on that page, also check..." or "Before you close the terminal, also try..."
4. **Call out gotchas** — if something needs auth, env vars, specific state, say so before they hit it

**What NOT to do:**
- Don't write a formal test plan document
- Don't number everything like a spec
- Don't say "run the test suite" — you're guiding manual testing
- Don't pad with disclaimers or boilerplate

**Stop here in manual mode. Do not proceed to step 5.**

---

## 5. Auto mode (`auto` flag present)

You are the tester. Do everything yourself.

### 5a. Setup

1. Check **todo.txt** — if there are incomplete setup steps, do them first
2. Start the dev server in background via Bash (`run_in_background: true`)
3. Wait for it to be ready, then health-check the URL
4. Ensure Playwright MCP is available for UI tests. If not, fall back to curl and note that visual tests were skipped.

### 5b. Test everything

Work through each feature in the order you planned. Power through everything — don't stop at the first failure. You want the full picture.

**CLI tests:**
- Run the actual command via Bash
- Capture stdout, stderr, exit code
- Compare against expected behavior

**API tests:**
- `curl -s -w "\n%{http_code}"` via Bash
- Check status code and response body

**UI tests:**
- Navigate to the page via `mcp__playwright__browser_navigate`
- **Screenshot immediately** via `mcp__playwright__browser_take_screenshot` — review what you see. Is the layout correct? Do elements render? Are images loaded? Does the styling look right?
- Get the accessibility snapshot via `mcp__playwright__browser_snapshot` to verify structure
- Interact: click buttons, fill forms, navigate — whatever a real user would do
- **Screenshot after every significant interaction** — did the UI update correctly? Did the modal appear? Did the form submit? Did the error message show?
- Check `mcp__playwright__browser_console_messages` for JS errors
- Check `mcp__playwright__browser_network_requests` for failed API calls

**DB/state tests:**
- Trigger the action, then verify the side effect

**Visual verification is mandatory for every UI test.** You must:
- Take at least one screenshot per page/screen
- Describe what you see in plain language
- Flag anything that looks broken, misaligned, or wrong — even if the snapshot says the structure is fine
- A screenshot that "looks fine" is evidence. A screenshot with broken layout is a bug.

**For each test, record:**
- **Status**: PASS, FAIL, or SKIP (with reason)
- **What happened**: brief description
- **Evidence**: output snippet or what the screenshot showed
- **Notes**: anything unexpected, even on passes

### 5c. Responsive and theme sweep (UI projects only)

After testing all features, do a final sweep:

1. Navigate to each major page
2. Resize to three viewports — screenshot each:
   - Desktop: 1280x800
   - Tablet: 768x1024
   - Mobile: 375x812
3. If dark/light theme exists, toggle and screenshot both
4. Report any layout breaks, overflow, or theme issues

### 5d. Cleanup

- Stop the dev server (kill the background process)
- Close the Playwright browser via `mcp__playwright__browser_close`
- Undo any test data or state changes if possible

## 6. Results report (auto mode only)

### Summary

```
Total: XX | Passed: XX | Failed: XX | Skipped: XX
```

### Results by feature

| ID | Test | Status | Notes |
|----|------|--------|-------|
| AUTH-01 | Login with valid creds | PASS | |
| AUTH-02 | Login with bad password | FAIL | Got 500 instead of 401 |

### Failures

For each FAIL:
- What was expected vs. what happened
- Error output or screenshot observations
- Where to look in the code (file path + what to check)

### Visual issues

Any layout, styling, or rendering problems found during screenshots — with what page, what viewport, and what's wrong.

### Verdict

- **Ship it** — everything works, looks right
- **Needs work** — list what's broken
- **Partial** — some tests skipped, passing tests look good
