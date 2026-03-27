---
name: jflow
description: >
  End-to-end app builder. Runs an upfront interview to catch all blockers (API keys, secrets,
  services, design, deployment), resolves them, then chains /setup → /autopilot to build
  the entire app from scaffolding to merged PRs in one command.
user-invocable: true
argument-hint: >
  [app idea, e.g. "daily briefing app with Stripe billing" | "skip-setup" to resume from autopilot on existing project | "dry-run" to preview plan without executing]
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion, Skill, TaskCreate, TaskUpdate, TaskList, WebFetch
effort: high
---

# jflow

Build an entire app from a rough idea — scaffolding, implementation, review, and shipping — in one command. The interview phase catches every blocker upfront so nothing surfaces hours into the build.

## 0. Parse arguments

- Extract the app idea from `$ARGUMENTS`
- Check for flags:
  - `skip-setup` — project already exists, skip to the bridge + autopilot
  - `dry-run` — show the full plan (interview results, roadmap preview, blocker list) without executing
  - `interactive` — pass through to `/autopilot` (confirm before each item)
- If `$ARGUMENTS` is empty and no flags, prompt with `AskUserQuestion`: "What do you want to build?"

## 1. Interview — single-pass blocker detection

The goal: one comprehensive question that covers everything `/setup` asks PLUS everything that would otherwise surface in `todo.txt` and block the build 2 hours in. The user answers what they can and skips what they're unsure about.

### 1a. Comprehensive intake

Parse the app idea from `$ARGUMENTS` to pre-fill what you can infer (e.g., "Next.js app with Stripe" implies web platform, Node stack, payment processing). Then present ONE `AskUserQuestion` with everything still needed:

> **Let's get everything upfront so the build runs clean.**
>
> **Project:**
> 1. Project name (kebab-case)
> 2. One-sentence tagline
> 3. Tech stack — language, framework, libraries
> 4. Key features / goals
> 5. Target platform — web, iOS, CLI, server, desktop
> 6. Repo visibility — public or private? (default: private)
>
> **Services** (skip any that don't apply):
> 7. Database — which one? Do you already have an instance running?
> 8. Auth — provider and method? (e.g., NextAuth, Clerk, Supabase Auth) Which social logins?
> 9. Payments — Stripe? Do you have API keys (even test keys)?
> 10. AI/LLM — which provider? Do you have an API key?
> 11. Email — transactional provider? (Resend, SendGrid, etc.)
> 12. File storage — S3, R2, Supabase Storage, local?
> 13. Any other third-party APIs or services?
>
> **Deployment:**
> 14. Where will this deploy? (Vercel, Fly.io, Railway, AWS, Cloudflare, etc.)
> 15. Custom domain? Do you own one?
>
> **Design** (skip for CLI/API-only projects):
> 16. Design reference — URL of a site you like, screenshot, or "surprise me"
> 17. Color scheme / brand colors / dark mode preference
> 18. Component library — shadcn/ui, Radix, MUI, Tailwind-only, etc.
>
> Answer what you know, skip what you don't. I'll use sensible defaults for anything left blank.

If the app idea in `$ARGUMENTS` is rich enough to answer most of these, only ask about the gaps — don't re-ask what's already clear.

### 1b. Targeted follow-ups (0-2 questions max)

After parsing the intake, identify critical ambiguities only:
- They said "auth" but not which provider — ask
- They said "database" but not which one — recommend based on stack and confirm
- Features suggest a service they didn't mention (e.g., "user profiles with avatars" implies file storage)

If the answers from 1a are clear, skip follow-ups entirely.

### 1c. Environment validation (automated — no questions)

Silently check what's already available on the machine:

```bash
# CLI tools
gh auth status                          # GitHub CLI
node --version 2>/dev/null              # Node.js (if JS/TS stack)
python3 --version 2>/dev/null           # Python (if Python stack)
cargo --version 2>/dev/null             # Rust (if Rust stack)
docker --version 2>/dev/null            # Docker (if they need it)

# Existing credentials
echo "${ROADMAP_PAT:-(not set)}"
echo "${STRIPE_SECRET_KEY:-(not set)}"
echo "${OPENAI_API_KEY:-(not set)}"
echo "${SUPABASE_URL:-(not set)}"
echo "${DATABASE_URL:-(not set)}"
# ... check any other env vars relevant to their stated services
```

Build two lists:
- **Ready** — tools installed, credentials found
- **Missing** — tools not installed, credentials not set, services not configured

## 2. Resolve blockers

Present all missing items grouped into one `AskUserQuestion` — not one question per blocker. Categorize clearly:

> **A few things to set up before the build starts:**
>
> **Need from you** (paste values or "skip" to defer):
> - Stripe test key (get one at https://dashboard.stripe.com/test/apikeys)
> - Supabase project URL + anon key (create at https://supabase.com/dashboard)
>
> **I'll install** (just confirming):
> - `shadcn/ui` — will set up during scaffolding
>
> **Optional** (can do later):
> - Custom domain DNS setup
> - Production API keys (test keys are fine for now)

For each credential the user provides:
- Store it in memory for writing to `.env` after `/setup` creates the project directory
- Never log, commit, or echo credentials back

For missing CLI tools, provide the install command and ask the user to run it (or offer to run it):
- `brew install node` / `brew install python` / etc.

**Hard gate:** Do not proceed past this step if critical blockers remain — tools required by the stated stack must be installed, `gh` must be authenticated. API keys for services used in Phase 1 roadmap items are critical; keys for later phases can be deferred.

**Soft gate:** Design preferences, custom domains, production keys — defer these with a note.

If nothing is missing, skip this step entirely.

## 3. Design capture (conditional)

Only for projects with a UI (web app, mobile app, desktop — not CLI, API, or library).

- If they provided a design reference URL: invoke the `scrape-design` skill with that URL. Capture the output — it will inform `/design` later.
- If they stated color/component preferences: record them.
- If they said "surprise me" or skipped: note that `/design create` will generate a default system.
- If no UI: skip entirely.

## 4. Show the plan

Before executing anything, present a summary of what's about to happen:

```
═══════════════════════════════════════
  jflow — Starting
═══════════════════════════════════════

  App: <project-name> — "<tagline>"
  Stack: <stack>
  Platform: <platform>
  Deploy: <target>

  Blockers resolved:
    ✓ Stripe test key — provided
    ✓ Supabase credentials — provided
    ✓ Node.js v22 — installed
    • Custom domain — deferred

  Plan:
    1. /setup — scaffold repo, CI/CD, roadmap, issues
    2. Bridge — install deps, write .env, verify build
    3. /design — create design system (from reference)
    4. /autopilot — build all roadmap items

═══════════════════════════════════════
```

If `dry-run` flag is set, stop here.

## 5. Invoke `/setup`

Invoke the `setup` skill via the Skill tool with all gathered context as the argument string. Include:
- Project name, tagline, tech stack, features, services, platform, visibility
- Be explicit: "Do NOT re-ask these questions — they've already been answered"

Example invocation:
```
skill: "setup", args: "Project: daily-briefing. Tagline: AI-powered daily briefing with billing. Stack: Next.js 14 + Supabase + Stripe. Features: 1) AI-generated daily briefings from RSS/news APIs, 2) Stripe subscription billing, 3) Email delivery via Resend, 4) User dashboard. Services: Supabase (auth + db + storage), Stripe (payments), OpenAI (summarization), Resend (email). Platform: web. Visibility: private. License: MIT. Do NOT re-ask these questions — they've already been answered in the jflow interview."
```

Wait for `/setup` to complete. Capture the repo URL and issue count from its output.

## 6. Bridge — post-setup, pre-autopilot

This fills the gap between a scaffolded repo and a buildable project. Run these steps in the newly created project directory (`~/Developer/<project-name>`):

### 6a. Install dependencies

```bash
# Run the appropriate install command based on stack
npm install          # Node.js / Next.js / React
pip install -e '.[dev]'  # Python
cargo build          # Rust
go mod tidy          # Go
bundle install       # Ruby
```

### 6b. Write `.env`

Create a `.env` file with all credentials collected in step 2:

```bash
# Verify .env is in .gitignore (it should be from /setup)
grep -q '.env' .gitignore || echo '.env' >> .gitignore
```

Write the `.env` with all collected values. Use comments to label sections:

```
# Database
SUPABASE_URL=<value>
SUPABASE_ANON_KEY=<value>

# Payments
STRIPE_SECRET_KEY=<value>
STRIPE_PUBLISHABLE_KEY=<value>

# AI
OPENAI_API_KEY=<value>
```

### 6c. Verify build

Run a quick build/lint to confirm the scaffolded project compiles before autopilot starts:

```bash
npm run build 2>&1 || npm run lint 2>&1    # Node.js
cargo check 2>&1                            # Rust
python -m py_compile <main-file> 2>&1       # Python
```

If the build fails, diagnose and fix the issue. Common causes:
- Missing dependencies — run install again
- TypeScript config issues — fix `tsconfig.json`
- Missing env vars at build time — adjust config to defer env checks to runtime

Do NOT proceed to `/autopilot` if the project doesn't build.

### 6d. Design system (conditional)

If design context was captured in step 3:

- Invoke the `design` skill: `skill: "design", args: "create"`
- If a `/scrape-design` output exists, reference it so the design system matches the reference
- This creates `DESIGN.md` and framework-specific token files that implementation agents will follow

If no UI or no design context, skip.

### 6e. Update `todo.txt`

Read the `todo.txt` that `/setup` generated. Check off any items that were already resolved in step 2 (API keys provided, secrets configured, tools installed). If all items are checked, delete the file.

### 6f. Compact context

The interview + setup phases consume significant context. Run `/compact` (built-in CLI command, not a Skill invocation) before starting autopilot to free up space for the build loop.

After compact, re-state the current position:

> **Resuming jflow.** Setup complete for `<project-name>`. Starting `/autopilot` — N roadmap items across M phases.

## 7. Invoke `/autopilot`

Invoke the `autopilot` skill via the Skill tool:

```
skill: "autopilot"
```

If the `interactive` flag was set, pass it: `skill: "autopilot", args: "interactive"`

`/autopilot` handles the full loop: `/next` → `/harden` → `/test` → `/ship` per item, with `/docs` + `/simplify` + `/checkup` at phase boundaries and `/qa auto` at the end.

Wait for it to complete.

## 8. Summary

Combine outputs from all phases into one final summary:

```
═══════════════════════════════════════
  jflow — Complete
═══════════════════════════════════════

  App: <project-name> — "<tagline>"
  Repo: https://github.com/<user>/<project-name>
  Stack: <stack>

  Setup:
    • Scaffolded N files
    • Created N GitHub issues across M phases
    • CI/CD, branch protection, roadmap sync

  Blockers resolved:
    ✓ Stripe test key
    ✓ Supabase credentials
    ✓ Design system created from reference

  Build:
    • Items shipped: N / M
    • PRs merged: N
    • Phases cleared: N

  Quality:
    • Docs syncs: N
    • Simplify passes: N
    • QA: N passed, N failed

  Deferred:
    • Custom domain setup
    • Production API keys

  Next:
    1. Review any /qa failures above
    2. Set up deferred items
    3. Deploy to <target>
    4. /autopilot to continue remaining items (if any)

═══════════════════════════════════════
```

## 9. Error recovery

| Failure point | What happens | How to resume |
|---|---|---|
| Interview aborted | Nothing was created | Re-run `/jflow` |
| `/setup` fails | Partial scaffolding | Fix the issue, then `/jflow skip-setup` |
| Bridge fails (deps/build) | Repo exists but doesn't build | Fix manually, then `/autopilot` |
| `/autopilot` fails mid-run | Some items shipped, some not | `/autopilot` to resume from next uncompleted item |

## Style guidelines

- Follow the standard output format in `_output-format.md`
- The interview should feel fast — one big question, not a wizard
- Batch blocker resolution into one question, not one per credential
- Show clear phase transitions: Interview → Setup → Bridge → Build
- Never echo credentials back to the user after they provide them
- Be explicit about what's blocking vs. what's optional
