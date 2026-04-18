<p align="center">
  <img src="assets/logo.png" alt="jflow" width="420">
  <br><br>
  Skills and agents for <a href="https://docs.anthropic.com/en/docs/claude-code">Claude Code</a> that automate the full dev loop (planning, implementation, code review, security hardening, and shipping) so you can go from a rough idea to a merged PR in one command.
  <br><br>
  <a href="https://github.com/oobagi/jflow/releases"><img src="https://img.shields.io/github/v/release/oobagi/jflow?label=version" alt="Version" /></a>
  <a href="https://github.com/oobagi/jflow/blob/main/LICENSE"><img src="https://img.shields.io/github/license/oobagi/jflow" alt="License" /></a>
  <a href="https://github.com/oobagi/jflow/stargazers"><img src="https://img.shields.io/github/stars/oobagi/jflow" alt="Stars" /></a>
  <a href="https://github.com/oobagi/jflow/issues"><img src="https://img.shields.io/github/issues/oobagi/jflow" alt="Issues" /></a>
</p>

<p align="center">
  <a href="#install"><strong>Install</strong></a>
  ·
  <a href="#skills"><strong>Skills</strong></a>
  ·
  <a href="#flows"><strong>Flows</strong></a>
  ·
  <a href="#agents"><strong>Agents</strong></a>
</p>

---

## How it works

[gstack](https://github.com/garrytan/gstack) proved that Claude Code gets dramatically better with structured roles and sequential workflows. But gstack is massively over-engineered — 20 roles simulating an entire company (CEO reviews, office hours, design consultations, retros, standups). It models how an *org* makes decisions. jflow models *the code*. No meetings, no management layer, no org chart. Just the loop every developer actually runs: plan it, build it in a worktree, review it, harden it, ship it, move on. Same results, fraction of the complexity, way faster.

The power is in the flows. `/jflow "your app idea"` runs an upfront interview that catches every blocker (API keys, secrets, services, design preferences) then chains `/setup` → `/autopilot` to scaffold, implement, review, and ship the entire app without stopping. `/autopilot` works through your roadmap: picks up the next item, implements it, dispatches parallel review agents, opens a PR, waits for CI, merges, and loops, running `/docs`, `/simplify`, and `/checkup` at phase boundaries. `/polish` does the same for ad-hoc work you've already built. One command in, merged PRs out.

The individual skills are useful on their own too. `/issue` turns a rough complaint into a well-scoped ticket, and auto-scales to a multi-issue breakdown with a roadmap when the idea is bigger. `/qa auto` opens the app in a real browser, clicks through every feature, takes screenshots, and reports what's broken. `/test` dispatches up to 6 review agents in parallel against your uncommitted changes and spawns fixers for anything they flag.

## Install

```bash
git clone https://github.com/oobagi/jflow.git && cd jflow && ./install.sh
```

Symlinks skills/agents into `~/.claude/`, copies hooks, merges settings (preserves your config). Requires `git` and `jq`.

Flags: `--dry-run` (preview) | `--no-settings` (skip settings merge) | `--no-rtk` (skip token proxy) | `--uninstall`

Settings merge is additive — your existing values are preserved. Backup saved to `settings.json.pre-jflow`.

## Skills

Invoked via `/skill-name`. Orchestrate agents, manage branches, ship code.

| Skill | What it does | Flags |
|-------|-------------|-------|
| `/jflow` | End-to-end app builder. Interview, scaffold, implement, review, ship | `skip-setup` `dry-run` `interactive` |
| `/autopilot` | Full dev loop, iterates ROADMAP items via `/next` → `/test` → `/ship` | `roadmap` `issues` `interactive` `phase-N` `dry-run` `compact-N%` |
| `/polish` | Quality pipeline: `/simplify` → `/harden` → `/test` → `/ship` | `dry-run` `no-ship` `skip-simplify` `skip-harden` |
| `/next` | Pick up next ROADMAP item, work in a worktree | `parallel` `<issue number>` |
| `/test` | Dispatch review agents against uncommitted changes | `security` `performance` `accessibility` `api` |
| `/ship` | Branch, commit, PR, CI, merge, cleanup | — |
| `/harden` | Security audit + input validation + error boundaries | `audit` `fix` `logging` `validation` `errors` `boundaries` |
| `/simplify` | Parallel agents fix DRY violations, dead code, complexity | `full` `scope:PATH` `dry-only` `dead-only` `logic-only` |
| `/issue` | Turn a rough idea into GitHub issues, auto-scaling to a multi-issue breakdown for bigger work | *problem description* |
| `/setup` | Scaffold new project: repo, CI/CD, docs, roadmap, issues | *project description* |
| `/docs` | Sync README, ROADMAP, CHANGELOG, docs/ with codebase state | `changelog` `full` |
| `/design` | Design system creation or audit (colors, typography, components) | `create` `audit` |
| `/scrape-design` | Playwright-based website design extraction | *url* |
| `/sitrep` | Branch health, stale worktrees, uncommitted work, recent activity | — |
| `/qa` | Walk through testing every feature, or `auto` for Playwright-driven testing | `auto` `latest` |
| `/release` | Cut a release — triggers the project's release mechanism and monitors to completion | `preview` `production` `--screenshots` |
| `/checkup` | Git hygiene: prune remotes, remove stale branches/worktrees, gc | `now` |
| `/upgrade-jflow` | Pull latest jflow, re-run installer | `check` `force` |

Flags are passed as arguments: `/autopilot issues interactive`, `/polish no-ship skip-harden`, `/harden audit logging`.

## Flows

Three skills orchestrate nested skill chains:

### `/jflow` — idea to finished app

> **When to use:** You have an app idea and nothing else. No repo, no scaffold, no roadmap. You want to go from zero to merged PRs in one session.

Flags: `skip-setup` (resume on existing project) | `dry-run` (preview plan) | `interactive` (confirm before each autopilot item)

```
"daily briefing app with Stripe billing"
  │
  ├─> interview      Catch all blockers upfront (keys, services, design)
  ├─> resolve        Install tools, collect credentials, validate env
  ├─> /setup         Scaffold repo, CI/CD, roadmap, issues
  ├─> bridge         Install deps, write .env, verify build, /design
  ├─> /autopilot     Build every roadmap item (loop below)
  └─> summary        What shipped, what's deferred, what to deploy
```

### `/autopilot` — ROADMAP to shipped PRs

> **When to use:** You already have a repo with a ROADMAP or open GitHub issues. You want Claude to work through them one by one — implement, review, ship, repeat — without babysitting.

Flags: `roadmap` | `issues` (source filter) | `interactive` (confirm each item) | `phase-N` (start at phase) | `dry-run` | `compact-N%` (context threshold)

```
ROADMAP.md item
  │
  ├─> /next       Create worktree, plan implementation
  ├─> implement   Dispatch architects + developers
  ├─> /test       Review agents validate
  ├─> /ship       Branch, PR, CI, merge
  │
  ├─> (repeat for next item)
  │
  └─> /harden + /docs + /simplify + /checkup  (at phase boundaries)
```

### `/polish` — cleanup to ship

> **When to use:** You've been hacking on something ad-hoc — it works but it's messy. You want it cleaned up, hardened, reviewed, and shipped without thinking about the steps.

Flags: `dry-run` | `no-ship` (stop before PR) | `skip-simplify` | `skip-harden`

```
uncommitted changes
  │
  ├─> /simplify   DRY, dead code, complexity
  ├─> /harden     Security + validation
  ├─> /test       Review agents validate
  └─> /ship       Branch, PR, CI, merge
```


## Agents

Specialized AI personas dispatched by skills. Based on [agency-agents](https://github.com/msitarzewski/agency-agents).

### Engineering

| Agent | Focus |
|-------|-------|
| Backend Architect | System design, databases, APIs, cloud infra |
| Code Reviewer | Correctness, maintainability (blocker/suggestion/nit) |
| Frontend Developer | React/Vue/Angular, UI, performance, a11y |
| Security Engineer | Threat modeling, vuln assessment, secure architecture |
| Software Architect | DDD, architectural patterns, ADRs |
| Technical Writer | Dev docs, API refs, tutorials |

### Game Development

| Agent | Focus |
|-------|-------|
| Game Audio Engineer | FMOD/Wwise, adaptive music, spatial audio |
| Game Designer | Gameplay loops, player psychology, economy balancing |
| Level Designer | Spatial storytelling, encounter design, pacing |
| Narrative Designer | Branching dialogue, lore, environmental storytelling |
| Technical Artist | Shaders, VFX, LOD pipelines, asset budgets |

### Product & QA

| Agent | Focus |
|-------|-------|
| Product Manager | Discovery, roadmap, PRD, go-to-market |
| UX Researcher | Usability testing, behavior analysis |
| Accessibility Auditor | WCAG 2.2 AA, assistive tech testing |
| API Tester | API validation, integration testing |
| Performance Benchmarker | Load testing, Web Vitals, capacity planning |
| Reality Checker | Evidence-based certification, defaults to "NEEDS WORK" |

## Bundled Tools

Installed and configured automatically. Skip any with install flags (`--no-rtk`) or by editing `settings.json` after install.

### RTK — token compression proxy

60–90% savings on dev operations. A `PreToolUse` hook transparently rewrites Bash commands before they execute:

```
git log --oneline -20  →  rtk git log --oneline -20
```

Covers: git, gh, grep, cat, curl, docker, kubectl, vitest, pytest, cargo, go, and more. Strips verbose formatting, trims whitespace, and compresses output so Claude consumes fewer tokens per tool call.

Installed to `~/.local/bin/rtk`. Skip with `--no-rtk` during install.

### code-simplifier — Claude Code plugin

Automated code quality analysis. Detects DRY violations, dead code, and overly complex logic, then suggests or applies fixes. Enabled as a Claude Code plugin in `settings.json`.

Used by `/simplify` and `/polish`, and runs automatically at phase boundaries during `/autopilot`.

### context7 — Claude Code plugin

Retrieves up-to-date documentation and code examples for any library or framework directly from source. Enabled as a Claude Code plugin in `settings.json`.

Used across skills whenever framework-specific guidance is needed — `/setup` for scaffolding best practices, `/harden` for validation library APIs, `/design` for design framework docs.

## Architecture

```
~/.jflow/                                ~/.claude/
├── skills/ ─────────── symlink ──────> ├── skills/
├── agents/ ─────────── symlink ──────> ├── agents/
├── hooks/rtk-rewrite.sh ── copy ─────> ├── hooks/rtk-rewrite.sh
└── settings/base.json ──── merge ────> └── settings.json
```

