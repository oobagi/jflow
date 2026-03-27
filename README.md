# jstack

Skill & agent stack for Claude Code. 15 skills, 17 agents — issue triage to shipped PR.

## Install

```bash
git clone https://github.com/oobagi/jstack.git && cd jstack && ./install.sh
```

Symlinks skills/agents into `~/.claude/`, copies hooks, merges settings (preserves your config).

Flags: `--dry-run` | `--no-settings` | `--no-rtk` | `--uninstall`

## Skills

Invoked via `/skill-name`. Orchestrate agents, manage branches, ship code.

| Skill | What it does |
|-------|-------------|
| `/autopilot` | Full dev loop — iterates ROADMAP items via `/next` -> `/harden` -> `/test` -> `/ship` |
| `/polish` | Quality pipeline — `/simplify` -> `/harden` -> `/test` -> `/ship` |
| `/next` | Pick up next ROADMAP item, work in a worktree |
| `/test` | Dispatch review agents against uncommitted changes |
| `/ship` | Branch, commit, PR, CI, merge, cleanup |
| `/harden` | Security audit + input validation + error boundaries. `audit` for report-only |
| `/simplify` | Parallel agents fix DRY violations, dead code, complexity |
| `/issue` | Turn a rough idea into a well-scoped GitHub issue |
| `/setup` | Scaffold new project — repo, CI/CD, docs, roadmap, issues |
| `/docs` | Sync README, ROADMAP, CHANGELOG, docs/ with codebase state |
| `/design` | Design system creation or audit (colors, typography, components) |
| `/scrape-design` | Playwright-based website design extraction |
| `/sitrep` | Branch health, stale worktrees, uncommitted work, recent activity |
| `/checkup` | Git hygiene — prune remotes, remove stale branches/worktrees, gc |
| `/upgrade-jstack` | Pull latest jstack, re-run installer |

## Flows

Two skills orchestrate nested skill chains:

### `/autopilot` — ROADMAP to shipped PRs

```
ROADMAP.md item
  │
  ├─> /next       Create worktree, plan implementation
  ├─> implement    Dispatch architects + developers
  ├─> /harden     Security audit + fixes
  ├─> /test       Review agents validate
  ├─> /ship       Branch, PR, CI, merge
  │
  ├─> (repeat for next item)
  │
  └─> /docs + /simplify + /checkup   (at phase boundaries)
```

### `/polish` — cleanup to ship

```
uncommitted changes
  │
  ├─> /simplify   DRY, dead code, complexity
  ├─> /harden     Security + validation
  ├─> /test       Review agents validate
  └─> /ship       Branch, PR, CI, merge
```

Supports `dry-run`, `no-ship`, `skip-simplify`, `skip-harden`.

### `/test` — agent dispatch

```
uncommitted changes
  │
  ├─> lint + tests (baseline)
  │
  ├─> parallel agents:
  │   ├── Code Reviewer       (always)
  │   ├── Reality Checker      (always)
  │   ├── Security Engineer    (always)
  │   ├── Accessibility Auditor (if UI changes)
  │   ├── API Tester           (if API changes)
  │   └── Perf Benchmarker     (if perf-sensitive)
  │
  ├─> deduplicate findings
  ├─> spawn fixers for blockers
  └─> verdict: ready to /ship or needs work
```

## Agents

Specialized AI personas dispatched by skills.

### Engineering

| Agent | Focus |
|-------|-------|
| Backend Architect | System design, databases, APIs, cloud infra |
| Code Reviewer | Correctness, maintainability — blocker/suggestion/nit |
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
| Reality Checker | Evidence-based certification — defaults to "NEEDS WORK" |

## Architecture

```
~/.jstack/                          ~/.claude/
├── skills/ ─── symlink ──────────> ├── skills/
├── agents/ ─── symlink ──────────> ├── agents/
├── hooks/rtk-rewrite.sh ─ copy ─> ├── hooks/rtk-rewrite.sh
└── settings/base.json ─── merge ─> └── settings.json
```

## RTK

Token compression proxy — 60-90% savings on dev operations. A `PreToolUse` hook transparently rewrites Bash commands:

```
git log --oneline -20  →  rtk git log --oneline -20
```

Covers: git, gh, grep, cat, curl, docker, kubectl, vitest, pytest, cargo, go, and more.

## Configuration

Merged from `settings/base.json` during install. Your existing values are preserved.

| Setting | Default | Purpose |
|---------|---------|---------|
| `effortLevel` | `high` | Thorough agent output |
| `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE` | `40` | Aggressive context compaction |
| Plugins | code-simplifier, context7, swift-lsp, rust-analyzer-lsp | Enabled by default |
