---
name: harden
description: >
  Analyze and implement code safety systems — structured error logging, input validation,
  error boundaries, and graceful degradation. Use "audit" for report-only mode.
user-invocable: true
argument-hint: >
  ["audit" for report only | "fix" to implement (default) | focus: "logging", "validation", "errors", "boundaries"]
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion, TaskCreate, TaskUpdate, TaskList
effort: high
---

# Harden

Analyze the codebase for safety gaps and implement robust error handling, structured logging, input validation, and graceful degradation. Designed to run standalone or as part of `/autopilot`.

## 0. Parse arguments

Check `$ARGUMENTS` for:

- **`audit`** — scan and report only, make no changes. Present findings as a prioritized checklist.
- **`fix`** — scan, report, and implement fixes (this is the default if no mode is specified).
- **A focus area** — narrow the scope to a specific domain:
  - `logging` — structured error/event logging, log levels, contextual metadata
  - `validation` — input validation at system boundaries (API routes, form handlers, CLI args, env vars)
  - `errors` — error handling patterns (try/catch, Result types, error propagation, custom error types)
  - `boundaries` — error boundaries (React), middleware (Express/Fastify), panic recovery (Go/Rust), global handlers

These can be combined: `audit logging` means "report logging issues only, don't fix."

## 1. Detect stack and context

Read project files to determine the tech stack and what safety patterns are idiomatic:

- Check `package.json`, `Cargo.toml`, `go.mod`, `pyproject.toml`, `Gemfile`, etc.
- Read `AGENTS.md`, `README.md`, or equivalent for project context
- Identify the framework (React, Next.js, Express, Fastify, Actix, Axum, Gin, Django, FastAPI, Rails, etc.)
- Note existing error handling and logging libraries already in use (winston, pino, slog, tracing, sentry, etc.)

**Do not introduce a new logging library if one is already in use.** Build on what exists.

## 2. Audit the codebase

Run a systematic scan across four safety domains. Use `Grep` and `Glob` extensively, and spawn an **Explore agent** for deeper analysis if the codebase is large.

**In parallel with the manual scan below**, launch a **Security Engineer** agent (`subagent_type: "Security Engineer"`). Give it the list of in-scope files and tell it to focus on areas the manual Grep/Glob scan **cannot catch well**: threat modeling, business logic vulnerabilities, auth/authz design flaws, and contextual security review that requires understanding data flow across files. The manual scan (2a–2d below) handles the mechanical pattern-matching — the Security Engineer handles the judgment calls. Deduplicate overlapping findings in step 3, keeping the more detailed version.

### 2a. Error handling gaps

Search for:

- **Unhandled promises** — `.then()` without `.catch()`, `async` functions without try/catch, missing `onRejectionHandled`
- **Empty catch blocks** — `catch (e) {}` or `catch (_)` with no logging or re-throw
- **Swallowed errors** — catch blocks that log to `console.log` instead of `console.error` or a proper logger
- **Missing error propagation** — functions that silently return `null`/`undefined`/default on failure instead of surfacing the error
- **Unchecked `.unwrap()`/`.expect()`** in Rust, unchecked `err` in Go, bare `except:` in Python
- **Missing finally blocks** — resource cleanup (DB connections, file handles, streams) without `finally` or RAII

### 2b. Logging gaps

Search for:

- **No logging at all** — API routes, background jobs, or critical paths with zero log statements
- **Console.log in production code** — `console.log` used where a structured logger should be
- **Missing error context** — errors logged without request ID, user ID, operation name, or relevant state
- **No log levels** — everything at the same level (no distinction between info, warn, error, fatal)
- **Missing audit trail** — mutations (create, update, delete) with no log of who did what
- **Sensitive data in logs** — passwords, tokens, PII logged in plaintext

### 2c. Input validation gaps

Search for:

- **Unvalidated API inputs** — route handlers that use `req.body`/`req.params`/`req.query` directly without schema validation
- **Missing environment variable validation** — `process.env.X` used without existence check or default
- **Type coercion risks** — comparisons with `==` instead of `===`, parseInt without radix, unvalidated JSON.parse
- **SQL/NoSQL injection surface** — string concatenation in queries instead of parameterized queries
- **Path traversal** — user input used in file paths without sanitization
- **Missing Content-Type / CORS / rate limiting** — at the middleware level

### 2d. Boundary protection gaps

Search for:

- **No global error handler** — missing Express error middleware, missing React error boundaries, missing panic recovery
- **No graceful shutdown** — server doesn't handle SIGTERM/SIGINT for connection draining
- **No health check endpoint** — no `/health` or `/ready` route for load balancers
- **No circuit breakers** — external service calls without timeout or retry limits
- **No request timeout** — HTTP server without request timeout configuration
- **Crash-on-error** — unhandled exceptions crashing the process instead of being caught at the boundary

## 3. Report findings

Present findings as a prioritized table, grouped by severity:

```
═══════════════════════════════════════
  Harden — Audit Results
═══════════════════════════════════════

  Critical (fix before shipping):
    1. No global error handler — server crashes on unhandled route errors
       → src/server.ts
    2. SQL injection via string concatenation
       → src/db/queries.ts:42, src/db/queries.ts:87

  High (fix soon):
    3. 12 empty catch blocks swallowing errors silently
       → src/api/users.ts:33, src/api/orders.ts:55, ...
    4. No structured logging — all output via console.log
       → 34 files affected

  Medium (improve when convenient):
    5. Environment variables used without validation
       → src/config.ts (8 vars unchecked)
    6. No graceful shutdown handler
       → src/index.ts

  Low (nice to have):
    7. No health check endpoint
    8. No request ID propagation in logs

  Summary: 2 critical · 2 high · 2 medium · 2 low
═══════════════════════════════════════
```

**If mode is `audit`, stop here.** Present the report and suggest `harden fix` to implement.

## 4. Implement fixes

Work through findings from highest to lowest severity. For each fix:

### 4a. Plan the fix

Before changing code, briefly state what you'll do and why. Prefer the **minimal correct fix** — don't over-engineer or add unnecessary abstractions.

### 4b. Apply the fix

Use `Edit` to modify existing files. Follow these principles:

- **Use existing libraries** — if pino/winston/tracing/slog is already installed, use it. Don't add a new one.
- **If no logger exists** — add a lightweight structured logger appropriate to the stack:
  - **Node.js**: pino (fast, structured JSON)
  - **Python**: stdlib `logging` with `structlog` if the project already uses it
  - **Go**: `slog` (stdlib) or `zerolog` if already present
  - **Rust**: `tracing` crate
- **Error types** — create a single error module/file with typed errors if one doesn't exist. Keep it minimal: a base error class/enum with `code`, `message`, `cause`, and `context` fields.
- **Validation** — use the project's existing validation library (zod, joi, valibot, pydantic, etc.) or add zod/pydantic if none exists. Validate at the boundary, trust internally.
- **Error boundaries** — add at the outermost layer only (one global handler, not per-component).
- **Logging** — add structured log calls at:
  - Request entry/exit (middleware)
  - Error catch points (with full context)
  - Mutation operations (who did what)
  - External service calls (with timing)
- **Never log secrets** — strip Authorization headers, mask tokens, omit passwords.

### 4c. Preserve behavior

Safety fixes must not change business logic. If a function previously returned `null` on error and callers depend on that, keep the return value but add logging — don't change the contract without updating all callers.

## 5. Validate fixes

After implementing:

1. Run the project's linter — fix any issues introduced.
2. Run the project's test suite — all existing tests must still pass.
3. If tests fail, the fix introduced a regression — revert and try a less invasive approach.

## 6. Summary

Present what was done:

```
═══════════════════════════════════════
  Harden — Complete
═══════════════════════════════════════

  Fixed:
    ✓ Added global error handler middleware (src/middleware/error.ts)
    ✓ Replaced 12 empty catch blocks with structured error logging
    ✓ Added pino structured logger (src/lib/logger.ts)
    ✓ Added request ID middleware for log correlation
    ✓ Added zod validation to 4 API routes
    ✓ Added graceful shutdown handler
    ✓ Added /health endpoint

  Deferred (needs manual review):
    • SQL query in src/db/legacy.ts:42 — uses dynamic table name,
      needs business context to fix safely

  Files changed: 14
  New files: 2 (src/lib/logger.ts, src/middleware/error.ts)

  Next: /test to validate, then /ship
═══════════════════════════════════════
```

## Integration with /autopilot

When invoked from `/autopilot thorough`, this skill runs at **phase boundaries** (not per-item). Phase boundary maintenance is opt-in — autopilot only runs it when the `thorough` flag is set.

In this context:
- Always run in `fix` mode (not audit)
- Scope to files changed during the entire phase (not just one item)
- Skip stack detection (already known from earlier in the loop)
- Only flag critical and high severity issues, defer medium/low
- If no issues found, report clean and move on immediately

## Style guidelines

- Follow the standard output format in `_output-format.md`
- Be direct — list findings, don't editorialize
- Show file paths and line numbers for every finding
- Group by severity, not by category
- When fixing, state what you're doing in one line, then do it
- Don't add comments like `// Added by harden` — the git history is the record
