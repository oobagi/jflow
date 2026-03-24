---
name: setup
description: >
  Scaffold a new project — creates a local directory, initializes a GitHub repo,
  and generates README, ROADMAP, AGENTS.md, LICENSE, docs/, and optional
  language-specific scaffolding. Private repo by default.
user-invocable: true
argument-hint: "[project description or context — leave blank for guided setup]"
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion
---

# Setup

Scaffold a new project from scratch. Gather requirements conversationally, then create everything in one shot.

## 1. Gather project info

If `$ARGUMENTS` is empty, ask the user:

> What's the project? Give me a name, a one-liner on what it does, and any tech/language preferences.

If `$ARGUMENTS` has content, parse it for as much info as possible (name, description, tech stack, goals).

Either way, use `AskUserQuestion` to fill in any gaps. Ask all unknowns in a **single** multi-part question to avoid back-and-forth. The fields you need:

| Field | Required | Default |
|---|---|---|
| **Project name** (kebab-case, used for dir + repo) | Yes | — |
| **Tagline** (one sentence, used in README header) | Yes | — |
| **Tech stack / language** | Yes | — |
| **Key features or goals** (seeds README body + ROADMAP) | No | — |
| **Repo visibility** | No | `private` |
| **License** | No | `MIT` |

Only ask about fields you genuinely can't infer from the context. If the user said "a Rust CLI for X", you already know the language — don't ask again.

If the license is non-obvious for the project type (e.g., a library that might want Apache-2.0, or a game that might want a custom license), briefly ask. Otherwise default to MIT silently.

## 2. Create project directory

```bash
mkdir -p ~/Developer/<project-name>
cd ~/Developer/<project-name>
git init
```

## 3. Generate files

Create all files from the project root (`~/Developer/<project-name>/`). Generate content — never leave files empty or placeholder-only.

### AGENTS.md

Write an `AGENTS.md` at the project root. This is read by Claude, Gemini, Codex, and other AI coding agents. Include:

- Project name and what it does (1-2 sentences)
- Tech stack, language version, key dependencies
- How to build, run, test, and lint
- Project structure overview (brief — just top-level dirs and what they contain)
- Any conventions or patterns to follow (naming, architecture style, etc.)

Keep it factual and concise. This is a reference doc, not marketing copy.

### README.md

Follow the user's established README style. The structure is:

```html
<p align="center">
  <img src="assets/<project-name>-icon.png" width="128" height="128" alt="<project-name> icon">
</p>

<h1 align="center"><Project Name></h1>

<p align="center">
  <strong>Tagline goes here — one sentence describing what this does and why.</strong>
</p>

<p align="center">
  <!-- shields: language version, license, platform, etc. — pick what's relevant -->
  <img src="https://img.shields.io/badge/..." alt="...">
</p>

<p align="center">
  <a href="#quickstart"><strong>Quickstart</strong></a>
  ·
  <a href="#docs"><strong>Docs</strong></a>
  ·
  <a href="ROADMAP.md"><strong>Roadmap</strong></a>
</p>

---
```

Then the body, which varies by project but generally follows:

1. **What it is / How it works** — brief explanation of the project, its purpose, why it's useful
2. **Features** — bold linked titles with short descriptions (only if there are known features)
3. **Quickstart** — clone, install, run. Language-appropriate commands.
4. **Docs** — table mapping doc files to topics (only if docs exist to link to)
5. **License** — one line, e.g. "MIT"

Adapt sections to what makes sense for the project. Don't force sections that have nothing to say yet. A README for a brand new project should be lean — it will grow.

For the image reference in the header, always include it pointed at `assets/<project-name>-icon.png` — the file won't exist yet, but it gives the user a clear place to drop an icon later.

### ROADMAP.md

If the user provided goals, features, or a project vision, create a real roadmap with phases/milestones. Use GitHub issue references where it makes sense (the issues won't exist yet — that's fine, they'll be created later).

If the user gave minimal context, create a simple roadmap with:

```markdown
# Roadmap

## Phase 1: Foundation
- [ ] Project scaffolding and CI setup
- [ ] Core architecture
- [ ] ...
```

Seed it with whatever is reasonable based on what you know about the project.

### LICENSE

Generate the full license text. Default to MIT with the current year and the user's GitHub username (get it from `gh api user --jq .login`). Use a different license only if the user specified one.

### .gitignore

Generate a `.gitignore` appropriate for the tech stack. If the stack isn't clear, create a minimal one:

```
.DS_Store
.env
*.log
```

For known stacks, include the standard ignores (e.g., `node_modules/`, `target/`, `__pycache__/`, `.venv/`, `build/`, etc.).

### docs/

Create a `docs/` directory. Only pre-create subdirectories or files if the project context makes it clear what's needed. Otherwise, just create the directory with a minimal `docs/README.md`:

```markdown
# Docs

Documentation for <project-name>.
```

### assets/

Create `assets/` directory (this is where the README header image will go):

```bash
mkdir -p assets
```

### Language-specific scaffolding

If the tech stack is clear, also generate the standard project files:

| Stack | Files |
|---|---|
| **Python** | `pyproject.toml`, `src/<package>/__init__.py`, `src/<package>/__main__.py` (if CLI) |
| **Rust** | `Cargo.toml`, `src/main.rs` or `src/lib.rs` |
| **Node/TypeScript** | `package.json`, `tsconfig.json` (if TS), `src/index.ts` or `src/index.js` |
| **Swift** | `Package.swift`, `Sources/<Name>/<Name>.swift` |
| **Go** | `go.mod`, `main.go` or `cmd/<name>/main.go` |

Use sensible defaults. The user can always change them.

## 4. Create GitHub repo and push

```bash
# Get GitHub username
gh_user=$(gh api user --jq .login)

# Create the repo (private by default)
gh repo create "$gh_user/<project-name>" --private --source=. --push

# Or if user requested public:
# gh repo create "$gh_user/<project-name>" --public --source=. --push
```

Make the initial commit before creating the repo:

```bash
git add -A
git commit -m "$(cat <<'EOF'
Initial project scaffolding

Generated with /setup — includes README, ROADMAP, AGENTS.md, LICENSE, docs/, and project skeleton.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

Then create the repo and push.

## 5. Summary

After everything is created, show the user:

- The repo URL (`gh repo view --json url --jq .url`)
- A tree view of what was created (`find . -not -path './.git/*' -not -name '.git' | head -30 | sort`)
- Any next steps (e.g., "Drop an icon at `assets/<project-name>-icon.png`", "Run `/next` to start working through the roadmap")
