<p align="center">
  <img src="assets/logo.png" alt="jflow" width="420">
  <br><br>
  A CLI/TUI harness that drives <a href="https://docs.anthropic.com/en/docs/claude-code">Claude Code</a> through real dev loops — workspaces, sessions, controlled context, and a todo pane the agent shares with you.
  <br><br>
  <a href="https://github.com/oobagi/jflow/releases"><img src="https://img.shields.io/github/v/release/oobagi/jflow?label=version" alt="Version" /></a>
  <a href="https://github.com/oobagi/jflow/blob/main/LICENSE"><img src="https://img.shields.io/github/license/oobagi/jflow" alt="License" /></a>
  <a href="https://github.com/oobagi/jflow/stargazers"><img src="https://img.shields.io/github/stars/oobagi/jflow" alt="Stars" /></a>
  <a href="https://github.com/oobagi/jflow/issues"><img src="https://img.shields.io/github/issues/oobagi/jflow" alt="Issues" /></a>
</p>

<p align="center">
  <a href="#what-this-is"><strong>What this is</strong></a>
  ·
  <a href="#install"><strong>Install</strong></a>
  ·
  <a href="#skills-v1-still-shipped"><strong>Skills</strong></a>
  ·
  <a href="#agents"><strong>Agents</strong></a>
  ·
  <a href="ROADMAP.md"><strong>Roadmap</strong></a>
</p>

---

## What this is

**jflow is becoming a Go CLI + Bubble Tea TUI** — a three-pane harness that drives `claude -p` as a subprocess. The mental model: **`claude` is the worker, `jflow` is the foreman.**

```
┌──────────────┬───────────────────────────────────┬────────────────────┐
│ workspaces   │ chat                              │ todo               │
│              │                                   │                    │
│ ▸ ~/code/app │  you ▸ work through next 3 issues │ [x]  Read tests    │
│   ~/.jflow   │                                   │ ▸    Patch regex   │
│   /tmp/scrtch│  claude ▸ Starting on issue #41…  │ [ ]  Run npm test  │
│              │  ⚙ Read("validate.test.ts")       │ [ ]  Open PR       │
│ + new        │  ⚙ Edit("validate.ts")            │                    │
│              │  > _                              │ a add · ⏎ activate │
└──────────────┴───────────────────────────────────┴────────────────────┘
```

**Why a harness?** The old `/jflow` and `/autopilot` skills degraded fast inside one Claude session because the same context window had to hold the orchestration playbook, plan the work, execute every step, and carry results between steps. By the third issue, context was bloated and the model started cutting corners.

Putting orchestration in code means jflow can:

- decide *when* to compact and *what* to carry forward (configurable per tool, e.g. `compact_at = 0.15`)
- run each phase in its own `claude -p` subprocess with a focused prompt
- share a **flat todo list** between the worker (via a bundled MCP server) and the user (via a focusable right pane) — no more guessing what the agent thinks it's doing
- spawn cheap Sonnet meta-calls for orchestration decisions ("is the worker stuck?", "grade this output") without polluting the worker's context

Status: Phase 1 prototype is working end-to-end (single-pane chat with streaming, status bar, per-turn driver lifecycle). Phase 2 (workspaces / sessions / todo pane) is next. Track progress in [`ROADMAP.md`](ROADMAP.md).

## Install

```bash
git clone https://github.com/oobagi/agency-skills.git ~/.jflow && cd ~/.jflow && ./install.sh
```

Symlinks skills/agents into `~/.claude/`, copies hooks, merges settings (preserves your config). Requires `git` and `jq`.

Flags: `--dry-run` (preview) | `--no-settings` (skip settings merge) | `--no-rtk` (skip token proxy) | `--uninstall`

> The Go binary install (`go install ./cmd/jflow/`) is wired into Phase 5 of the roadmap. Until then, run the prototype directly: `cd ~/.jflow && go run ./cmd/jflow/`.

## Skills (v1, still shipped)

The skill bundle is the current way to use jflow inside Claude Code. The CLI rewrite ports the **`jflow` suite** (`autopilot`, `next`, `ship`, `polish`, `qa`, `release`, `jflow`, `setup`, `issue`) into deterministic Go tool programs. The standalone skills below stay as Claude Code skills — they don't need a harness.

Invoked via `/skill-name`. Orchestrate agents, manage branches, ship code.

| Skill | What it does | Status |
|-------|-------------|-------|
| `/jflow` | End-to-end app builder. Interview, scaffold, implement, review, ship | porting → CLI |
| `/autopilot` | Full dev loop, iterates ROADMAP items via `/next` → `/ship` | porting → CLI |
| `/next` | Pick up next ROADMAP item, work in a worktree | porting → CLI |
| `/ship` | Branch, commit, PR, CI, merge, cleanup | porting → CLI |
| `/polish` | Quality pipeline: simplify → harden → test → ship | porting → CLI |
| `/qa` | Walk through testing every feature, or `auto` for Playwright-driven testing | porting → CLI |
| `/release` | Cut a release — triggers the project's release mechanism and monitors to completion | porting → CLI |
| `/setup` | Scaffold new project: repo, CI/CD, docs, roadmap, issues | porting → CLI |
| `/issue` | Turn a rough idea into GitHub issues, auto-scaling to multi-issue breakdown | porting → CLI |
| `/test` | Dispatch review agents against uncommitted changes | stays as skill |
| `/simplify` | Parallel agents fix DRY violations, dead code, complexity | stays as skill |
| `/harden` | Security audit + input validation + error boundaries | stays as skill |
| `/docs` | Sync README, ROADMAP, CHANGELOG, docs/ with codebase state | stays as skill |
| `/sitrep` | Branch health, stale worktrees, uncommitted work, recent activity | stays as skill |
| `/checkup` | Git hygiene: prune remotes, remove stale branches/worktrees, gc | stays as skill |
| `/design` | Design system creation or audit | stays as skill |
| `/scrape-design` | Playwright-based website design extraction | stays as skill |
| `/upgrade-jflow` | Pull latest jflow, re-run installer | stays as skill |

Flags are passed as arguments: `/autopilot issues interactive`, `/polish no-ship skip-harden`, `/harden audit logging`.

### Flows

Three skills orchestrate nested skill chains:

#### `/jflow` — idea to finished app

> **When to use:** App idea, no repo. Go from zero to merged PRs in one session.

```
"daily briefing app with Stripe billing"
  ├─> interview      Catch all blockers upfront (keys, services, design)
  ├─> /setup         Scaffold repo, CI/CD, roadmap, issues
  ├─> /autopilot     Build every roadmap item
  └─> summary        What shipped, what's deferred, what to deploy
```

#### `/autopilot` — ROADMAP to shipped PRs

> **When to use:** Repo with a ROADMAP or open issues. Work through them one by one without babysitting.

```
ROADMAP.md item
  ├─> /next       Create worktree, plan implementation
  ├─> implement   Dispatch architects + developers
  ├─> /test       Review agents validate (when warranted)
  ├─> /ship       Branch, PR, CI, merge
  └─> (loop)
```

#### `/polish` — cleanup to ship

> **When to use:** You've been hacking ad-hoc and want it cleaned up, hardened, reviewed, shipped.

```
uncommitted changes
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

Today (skill bundle):

```
~/.jflow/                                ~/.claude/
├── skills/ ─────────── symlink ──────> ├── skills/
├── agents/ ─────────── symlink ──────> ├── agents/
├── hooks/rtk-rewrite.sh ── copy ─────> ├── hooks/rtk-rewrite.sh
└── settings/base.json ──── merge ────> └── settings.json
```

Coming (CLI harness):

```
~/.jflow/
├── cmd/jflow/                  cobra root, subcommands
├── internal/
│   ├── claude/                 driver — spawns claude -p, decodes JSONL
│   ├── ui/                     bubbletea v2 three-pane TUI
│   │   ├── todopane/           flat todo list, active indicator
│   │   └── ...
│   ├── workspace/              cwd-keyed registry
│   ├── session/                per-session state (transcript, todos, usage)
│   ├── tool/                   Tool interface + autopilot/next/ship/...
│   ├── mcp/todo/               bundled MCP server: todo_* tools
│   └── meta/                   cheap-Sonnet meta-loop for orchestration decisions
└── ~/.jflow/state/             workspaces.json + sessions/<uuid>.json + logs/
```

See [`docs/`](docs/) for the full design — overview, architecture, TUI, context management, build order, meta-model loop.
