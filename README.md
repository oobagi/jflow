# jstack

A skill & agent stack for Claude Code. Ships 15 task-orchestration skills and 17 specialized AI agents that handle the full development lifecycle — from issue triage to shipped PR.

## Install

```bash
git clone https://github.com/oobagi/jstack.git
cd jstack
./install.sh
```

The installer symlinks skills and agents into `~/.claude/`, copies hooks, and merges settings into `settings.json` (preserving your existing config).

```bash
./install.sh --dry-run        # Preview without changes
./install.sh --no-settings    # Skip settings merge
./install.sh --no-rtk         # Skip rtk (token proxy) install
./install.sh --uninstall      # Remove symlinks and hooks
```

## Skills

Skills are task workflows invoked via `/skill-name` in Claude Code. They orchestrate agents, manage branches, run tests, and ship code.

| Skill | Description |
|-------|-------------|
| `/autopilot` | End-to-end feature loop — iterates through ROADMAP.md items running `/next` -> `/harden` -> `/test` -> `/ship` for each. Runs `/docs`, `/simplify`, `/checkup` at phase boundaries. |
| `/checkup` | Git repo health — removes stale worktrees/branches, prunes remotes, runs gc, flags hygiene issues. |
| `/design` | Design system creation and audit. Create mode generates colors, typography, components, layout patterns. Audit mode reviews for consistency and accessibility. |
| `/docs` | Syncs documentation with codebase state. Updates README, ROADMAP, AGENTS, CHANGELOG, and `docs/` to fix stale references. |
| `/harden` | Analyze and implement safety systems — structured error logging, input validation, error boundaries, graceful degradation. Use `audit` for report-only. |
| `/issue` | Turn a rough idea into a well-scoped GitHub issue with context, solution avenues, task list, and test plan. |
| `/next` | Pick up the next ROADMAP item (or a specific issue) and work in a worktree. Use `parallel` to suggest concurrent issues. |
| `/polish` | Post-implementation quality pipeline — runs `/simplify` -> `/harden` -> `/test` -> `/ship` in sequence. Use `dry-run` to preview, `no-ship` to stop before shipping. |
| `/scrape-design` | Scrape a website via Playwright and produce a high-fidelity design doc — colors, typography, layout, component inventory, design philosophy. |
| `/setup` | Scaffold a new project — GitHub repo, README, ROADMAP, AGENTS.md, LICENSE, CI/CD, docs/, and language-specific scaffolding. Sets up issues for every roadmap item. |
| `/ship` | Branch, commit, PR, merge, cleanup. Handles auto-merge, CI monitoring, and branch protection. |
| `/simplify` | Deep codebase simplification — parallel agents fix DRY violations, remove dead code, simplify logic, create reusable helpers. |
| `/sitrep` | Situation report — branch health, stale worktrees, uncommitted work, in-progress plans, recent activity. |
| `/test` | Dispatch review agents (Code Reviewer, Reality Checker, Security Engineer, + conditional specialists) against uncommitted changes. |
| `/upgrade-jstack` | Check for and install jstack updates. Compares local vs. remote, pulls latest, re-runs installer. |

## Agents

Agents are specialized AI personas dispatched by skills. Each has a defined role, core mission, critical rules, deliverables, and success metrics.

### Engineering

| Agent | Specialization |
|-------|---------------|
| **Backend Architect** | Scalable system design, database architecture, API development, cloud infrastructure |
| **Code Reviewer** | Correctness, maintainability, security, performance — prioritized as blocker/suggestion/nit |
| **Frontend Developer** | Modern web (React/Vue/Angular), UI implementation, performance, accessibility |
| **Security Engineer** | Threat modeling (STRIDE), vulnerability assessment, secure code review, security architecture |
| **Software Architect** | System design, domain-driven design, architectural patterns, ADRs |
| **Technical Writer** | Developer docs, API references, README files, tutorials, docs-as-code |

### Game Development

| Agent | Specialization |
|-------|---------------|
| **Game Audio Engineer** | FMOD/Wwise integration, adaptive music, spatial audio, audio performance budgeting |
| **Game Designer** | Systems/mechanics architecture, gameplay loops, player psychology, economy balancing |
| **Level Designer** | Spatial storytelling, flow architecture, encounter design, environmental narrative |
| **Narrative Designer** | Story systems, branching dialogue, lore architecture, environmental storytelling |
| **Technical Artist** | Shaders, VFX systems, LOD pipelines, asset budgeting, cross-engine optimization |

### Product & UX

| Agent | Specialization |
|-------|---------------|
| **Product Manager** | Full product lifecycle — discovery, roadmap, PRD, go-to-market, outcome measurement |
| **UX Researcher** | User behavior analysis, usability testing, data-driven design insights |

### Testing & QA

| Agent | Specialization |
|-------|---------------|
| **Accessibility Auditor** | WCAG 2.2 AA compliance, assistive tech testing, inclusive design — defaults to finding barriers |
| **API Tester** | API validation, performance testing, integration testing — 95%+ coverage target |
| **Performance Benchmarker** | Load testing, Web Vitals optimization, capacity planning, regression detection |
| **Reality Checker** | Evidence-based certification — defaults to "NEEDS WORK", requires proof for production readiness |

## How It Works

### Architecture

```
~/.jstack/                          ~/.claude/
├── skills/ ─── symlink ──────────> ├── skills/
├── agents/ ─── symlink ──────────> ├── agents/
├── hooks/                          ├── hooks/
│   └── rtk-rewrite.sh ── copy ──> │   └── rtk-rewrite.sh
├── settings/                       └── settings.json  <── merged
│   └── base.json ── merge ────────────────┘
├── install.sh
└── VERSION
```

### Skill -> Agent Dispatch

Skills orchestrate agents based on what changed. For example, `/test` on uncommitted changes:

1. Identifies changed files and runs linter/tests as a baseline
2. Dispatches agents in parallel:
   - **Code Reviewer** — always
   - **Reality Checker** — always
   - **Security Engineer** — always
   - **Accessibility Auditor** — if UI changes detected
   - **API Tester** — if API changes detected
   - **Performance Benchmarker** — if perf-sensitive changes detected
3. Collects findings, deduplicates across agents
4. Marks blockers vs. suggestions
5. Optionally spawns fixer agents for blockers
6. Final verdict: ready to `/ship` or needs work

### Full Lifecycle: `/autopilot`

```
ROADMAP.md item
  │
  ├─> /next          Create worktree, plan implementation
  ├─> implement       Dispatch Backend Architect, Frontend Developer, etc.
  ├─> /harden        Security audit + fixes
  ├─> /test          Review agents validate
  ├─> /ship          Branch, PR, CI, merge
  │
  ├─> (repeat for next item)
  │
  └─> /docs + /simplify + /checkup   (at phase boundaries)
```

## RTK Integration

jstack includes [rtk](https://github.com/oobagi/rtk), a token compression proxy that reduces Claude Code token usage by 60-90% on dev operations.

A `PreToolUse` hook transparently rewrites Bash commands to rtk equivalents:

```
git log --oneline -20  →  rtk git log --oneline -20
```

Supported: git, gh, cat, grep, rg, ls, tree, find, curl, docker, kubectl, vitest, pytest, cargo, go, and more.

```bash
rtk gain              # Show token savings analytics
rtk gain --history    # Command usage history with savings
rtk discover          # Find missed optimization opportunities
```

## Configuration

Default settings are merged from `settings/base.json` during install:

| Setting | Default | Purpose |
|---------|---------|---------|
| `effortLevel` | `"high"` | Agents aim for thorough, detailed work |
| `CLAUDE_AUTOCOMPACT_PCT_OVERRIDE` | `40` | Aggressive context compaction at 40% usage |
| `promptSuggestionEnabled` | `false` | No prompt suggestions — user directs flow |
| `skipDangerousModePermissionPrompt` | `true` | Agents can use tools without per-command prompts |

Enabled plugins: code-simplifier, context7 (live docs), swift-lsp, rust-analyzer-lsp.

Your existing settings are preserved — jstack merges additively and never overwrites scalar values you've set.

## Project Structure

```
~/.jstack/
├── install.sh
├── VERSION
├── settings/
│   └── base.json
├── hooks/
│   └── rtk-rewrite.sh
├── skills/
│   ├── autopilot/SKILL.md
│   ├── checkup/SKILL.md
│   ├── design/SKILL.md
│   ├── docs/SKILL.md
│   ├── harden/SKILL.md
│   ├── issue/SKILL.md
│   ├── next/SKILL.md
│   ├── polish/SKILL.md
│   ├── scrape-design/SKILL.md
│   ├── setup/SKILL.md
│   ├── ship/SKILL.md
│   ├── simplify/SKILL.md
│   ├── sitrep/SKILL.md
│   ├── test/SKILL.md
│   └── upgrade-jstack/SKILL.md
└── agents/
    ├── engineering-backend-architect.md
    ├── engineering-code-reviewer.md
    ├── engineering-frontend-developer.md
    ├── engineering-security-engineer.md
    ├── engineering-software-architect.md
    ├── engineering-technical-writer.md
    ├── game-audio-engineer.md
    ├── game-designer.md
    ├── level-designer.md
    ├── narrative-designer.md
    ├── product-manager.md
    ├── technical-artist.md
    ├── design-ux-researcher.md
    ├── testing-accessibility-auditor.md
    ├── testing-api-tester.md
    ├── testing-performance-benchmarker.md
    └── testing-reality-checker.md
```

## Version

0.1.0
