---
name: jflow
description: >
  End-to-end app builder. Quick interview to catch blockers (API keys, services, design),
  then chains /setup → /autopilot to build the app.
user-invocable: true
argument-hint: >
  [app idea, e.g. "daily briefing app with Stripe" | "skip-setup" to resume on existing project | "dry-run" to preview]
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion, Skill, TaskCreate, TaskUpdate, TaskList, WebFetch
effort: high
---

# jflow

Build an app from a rough idea — scaffolding, implementation, and shipping — in one command.

## 0. Parse arguments

- Extract the app idea from `$ARGUMENTS`
- Flags:
  - `skip-setup` — project already exists, skip to bridge + autopilot
  - `dry-run` — show the plan without executing
  - `interactive` — pass through to `/autopilot`
- If empty and no flags, ask: "What do you want to build?"

## 1. Interview

One question. Cover what's needed, skip what's obvious from the app idea.

### 1a. Intake

Parse the app idea to pre-fill what you can infer. Then present ONE `AskUserQuestion` with only the gaps:

> **Quick setup — answer what you know, skip what you don't.**
>
> **Project:** name, stack, key features, platform (web/mobile/CLI)
> **Services** (if any): database, auth, payments, AI, email, storage
> **Deploy:** where? (Vercel, Fly, Railway, etc.)
> **Design** (if UI): reference URL, colors, component library — or "surprise me"
>
> I'll use sensible defaults for anything left blank.

If the app idea already answers most of this, only ask about the actual gaps — don't re-ask what's clear. For simple projects (CLI tool, library, API), skip design and deployment questions entirely.

### 1b. Follow-up (0-1 questions max)

Only if there's a critical ambiguity (e.g., they said "auth" but not which provider). Otherwise skip.

### 1c. Environment check (silent)

```bash
gh auth status
# Check relevant CLI tools and env vars based on stated stack
```

Build ready/missing lists.

## 2. Resolve blockers

If anything is missing, present ONE grouped question:

> **Before we start:**
>
> **Need from you:** [credentials/keys with links to get them]
> **I'll install:** [packages/tools during setup]
> **Optional (can do later):** [non-critical items]

For credentials the user provides: store for `.env`, never echo back.

**Hard gate:** Stack tools must be installed, `gh` must be authenticated, Phase 1 API keys must exist.
**Soft gate:** Design preferences, domains, production keys — defer.

If nothing is missing, skip.

## 3. Show the plan

```
═══════════════════════════════════════
  jflow — Starting
═══════════════════════════════════════

  App: <name> — "<tagline>"
  Stack: <stack>
  Deploy: <target>

  Plan:
    1. /setup — scaffold repo, CI, roadmap, issues
    2. Bridge — deps, .env, verify build
    3. /autopilot — build all items

═══════════════════════════════════════
```

If `dry-run`, stop here.

## 4. Run `/setup`

Invoke the `setup` skill with all gathered context. Be explicit: "Do NOT re-ask these questions."

## 5. Bridge

In the new project directory:

### 5a. Install dependencies
Run the appropriate install command for the stack.

### 5b. Write `.env`
Create `.env` with collected credentials. Verify `.env` is in `.gitignore`.

### 5c. Verify build
Run a quick build/lint. If it fails, diagnose and fix before continuing.

### 5d. Design system (if UI project)
If design context was captured, invoke `/design create`.

### 5e. Clean up todo.txt
Check off resolved items from `/setup`'s `todo.txt`. Delete if all done.

## 6. Run `/autopilot`

Invoke `/autopilot`. Pass `interactive` flag if the user requested it.

## 7. Summary

```
═══════════════════════════════════════
  jflow — Complete
═══════════════════════════════════════

  App: <name>
  Repo: <url>

  Items shipped: N / M
  PRs merged: N

  Deferred:
    • <anything skipped>

  Next:
    1. /release preview to publish a test build
    2. Deploy to <target>
    3. /autopilot to continue remaining items

═══════════════════════════════════════
```

## 8. Error recovery

| Failure | Resume |
|---|---|
| Interview aborted | Re-run `/jflow` |
| `/setup` fails | Fix, then `/jflow skip-setup` |
| Bridge fails | Fix manually, then `/autopilot` |
| `/autopilot` fails | `/autopilot` to resume |

## Style guidelines

- Follow the standard output format in `_output-format.md`
- Interview should feel fast — one question, not a wizard
- Batch blockers into one question
- Never echo credentials back
