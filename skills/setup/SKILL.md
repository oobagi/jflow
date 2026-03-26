---
name: setup
description: >
  Scaffold a new project — creates a local directory, initializes a GitHub repo,
  and generates README, ROADMAP, AGENTS.md, LICENSE, docs/, CI/CD workflows,
  and optional language-specific scaffolding. Sets up GitHub issues for every
  roadmap item, auto-syncing roadmap checkboxes, branch protection, and
  auto-merge. Private repo by default.
user-invocable: true
argument-hint: "[project description or context — leave blank for guided setup]"
allowed-tools: Bash, Read, Write, Edit, Glob, Grep, Agent, AskUserQuestion
---

# Setup

Scaffold a new project from scratch. Gather requirements conversationally, then create everything in one shot — including full CI/CD automation.

## 1. Gather project info — batch intake

Start with a single, comprehensive intake question. This avoids slow back-and-forth and lets the user think holistically about their project upfront.

### 1a. First question — the essentials

If `$ARGUMENTS` is empty or sparse, present **all** of these questions in one `AskUserQuestion` call. Format them as a numbered list so the user can answer inline:

> **Let's set up your project.** Answer as many of these as you can — skip any you're unsure about and I'll use sensible defaults.
>
> 1. **Project name** — what should the repo be called? (kebab-case, e.g. `my-cool-app`)
> 2. **What does it do?** — one sentence tagline
> 3. **Tech stack** — language, framework, major libraries (e.g. "Rust CLI", "Next.js + Postgres", "Python FastAPI")
> 4. **Key features or goals** — what should it do when it's done? (bullet points are fine)
> 5. **External services** — any APIs, databases, auth providers, or third-party services it needs? (e.g. Stripe, Supabase, OpenAI, Redis)
> 6. **Target platform** — where does it run? (e.g. web, iOS, CLI, server, desktop, embedded)
> 7. **Repo visibility** — public or private? (default: private)
> 8. **License** — MIT, Apache-2.0, or something else? (default: MIT)

If `$ARGUMENTS` already contains rich context, parse it for as many of these fields as possible and **only ask about the remaining gaps** — but still ask them all in a single question.

### 1b. Follow-up — defining questions

After the user answers the essentials, review their answers and improv 1-3 follow-up questions **only if needed** to resolve ambiguities or make critical decisions. Examples of when follow-ups are warranted:

- They said "web app" but didn't mention auth — ask if they need authentication and what kind
- They mentioned a database but not which one — ask their preference
- The features suggest a complex architecture — ask about their preferred patterns (monorepo vs multi-repo, monolith vs microservices, etc.)
- They mentioned an API — ask if it's REST, GraphQL, or gRPC

If the answers from 1a are clear and complete, **skip follow-ups entirely** and proceed to step 2. Don't ask questions just to ask questions.

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

### ROADMAP.md (draft — issue links added in step 5)

Generate the roadmap with phases/milestones based on the user's goals. Use this format for each item — **without issue links for now** (they get added after issues are created in step 5):

```markdown
# Roadmap

This roadmap orders open work by leverage and dependency. Each item links to a GitHub issue with full details. Checkboxes are updated automatically when issues are closed or reopened.

## Phase 1: Foundation

- [ ] Project scaffolding and CI setup
- [ ] Core architecture
- [ ] ...

## Phase 2: ...

- [ ] ...
```

Keep items concrete and actionable — each one should be a meaningful unit of work that maps well to a single GitHub issue. Avoid vague items like "improve performance" — instead, specify what gets improved.

### LICENSE

Generate the full license text. Default to MIT with the current year and the user's GitHub username (get it from `gh api user --jq .login`). Use a different license only if the user specified one.

### .gitignore

Generate a `.gitignore` appropriate for the tech stack. Always include `todo.txt` (used by /setup for user action items). If the stack isn't clear, create a minimal one:

```
.DS_Store
.env
*.log
todo.txt
```

For known stacks, include the standard ignores (e.g., `node_modules/`, `target/`, `__pycache__/`, `.venv/`, `build/`, etc.) plus `todo.txt`.

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

### .github/workflows/ci.yml

Generate a CI workflow appropriate for the tech stack. Trigger on push to `main` and pull requests to `main`. Include lint, test, and build steps.

**Templates by stack:**

**Python:**
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        python-version: ["3.12", "3.13"]

    steps:
      - uses: actions/checkout@v4

      - name: Set up Python
        uses: actions/setup-python@v5
        with:
          python-version: ${{ matrix.python-version }}

      - name: Install dependencies
        run: |
          python -m pip install --upgrade pip
          python -m pip install -e '.[dev]'

      - name: Lint
        run: ruff check .

      - name: Test
        run: pytest -q

      - name: Build
        run: python -m build
```

**Rust:**
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install Rust
        uses: dtolnay/rust-toolchain@stable
        with:
          components: clippy, rustfmt

      - name: Check formatting
        run: cargo fmt --check

      - name: Lint
        run: cargo clippy -- -D warnings

      - name: Test
        run: cargo test

      - name: Build
        run: cargo build --release
```

**Node/TypeScript:**
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        node-version: [20, 22]

    steps:
      - uses: actions/checkout@v4

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: ${{ matrix.node-version }}

      - name: Install dependencies
        run: npm ci

      - name: Lint
        run: npm run lint

      - name: Test
        run: npm test

      - name: Build
        run: npm run build
```

**Swift:**
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4

      - name: Build
        run: swift build

      - name: Test
        run: swift test
```

**Go:**
```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: 'stable'

      - name: Lint
        uses: golangci/golangci-lint-action@v4

      - name: Test
        run: go test ./...

      - name: Build
        run: go build ./...
```

Adapt the template to match the project's actual setup (e.g., if pyproject.toml has specific extras, use them in the install step). If the stack doesn't match any template, create a reasonable CI workflow based on the stack's conventions.

### .github/workflows/roadmap-sync.yml

Generate the roadmap auto-sync workflow. This is stack-independent — it's the same for every project:

```yaml
name: Sync Roadmap Checkboxes

on:
  issues:
    types: [closed, reopened]

jobs:
  update-roadmap:
    runs-on: ubuntu-latest
    env:
      ISSUE_NUMBER: ${{ github.event.issue.number }}
      ISSUE_STATE: ${{ github.event.action }}
    steps:
      - name: Check out repository
        uses: actions/checkout@v4
        with:
          token: ${{ secrets.ROADMAP_PAT }}

      - name: Update ROADMAP.md checkbox
        run: |
          if [ "$ISSUE_STATE" = "closed" ]; then
            sed -i "s/^- \[ \] \[#${ISSUE_NUMBER} /- [x] [#${ISSUE_NUMBER} /" ROADMAP.md
          else
            sed -i "s/^- \[x\] \[#${ISSUE_NUMBER} /- [ ] [#${ISSUE_NUMBER} /" ROADMAP.md
          fi

      - name: Commit and push
        env:
          GH_TOKEN: ${{ secrets.ROADMAP_PAT }}
        run: |
          git diff --quiet ROADMAP.md && exit 0
          BRANCH="roadmap/mark-${ISSUE_NUMBER}-${ISSUE_STATE}"
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git checkout -b "$BRANCH"
          git add ROADMAP.md
          git commit -m "roadmap: mark #${ISSUE_NUMBER} as ${ISSUE_STATE}"
          git push origin "$BRANCH"
          PR_URL=$(gh pr create \
            --title "roadmap: mark #${ISSUE_NUMBER} as ${ISSUE_STATE}" \
            --body "Auto-generated: update ROADMAP.md checkbox for #${ISSUE_NUMBER}." \
            --head "$BRANCH" \
            --base main)
          gh pr merge "$PR_URL" --auto --merge --delete-branch
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

Generated with /setup — includes README, ROADMAP, AGENTS.md, LICENSE, CI/CD, docs/, and project skeleton.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

Then create the repo and push.

## 5. Set up automation

After the repo exists on GitHub, set up the full automation pipeline.

### 5a. Create GitHub issues for every roadmap item

For each item in ROADMAP.md, create a GitHub issue using `gh issue create`. The issue title should match the roadmap item text. Assign a label matching the roadmap phase (create labels first if needed).

**Each issue must be a proper spec, not a one-liner.** Write the issue body as a structured document that someone could implement without asking clarifying questions. Use this format:

```markdown
## Context

Why this work matters — what problem it solves, what depends on it, or what user need it addresses. 2-3 sentences grounding the reader.

## Proposed approach

Concrete steps, commands, APIs, data models, or architectural decisions relevant to implementing this item. Be specific — name files, functions, endpoints, schemas where applicable. If there are multiple valid approaches, briefly note the recommended one and why.

## Tasks

- [ ] First concrete subtask
- [ ] Second concrete subtask
- [ ] Write tests for X
- [ ] Update docs if needed

## Acceptance criteria

- [ ] Specific condition that must be true when this is done
- [ ] Another measurable outcome
- [ ] Tests pass, lint clean

## References

- Related issues: #N, #M
- Relevant docs: `docs/foo.md`, external links
- Design decisions: any ADRs or notes from the architecture review
```

Adapt the template to each item — a small bug fix needs less structure than a core architecture decision. But every issue should have at minimum: **Context** (why), **Tasks** (what to do), and **Acceptance criteria** (how to verify). The goal is issues detailed enough to hand to `/next` for autonomous implementation.

```bash
# Example: create an issue with a proper body and capture its number
ISSUE_NUM=$(gh issue create \
  --title "Core architecture" \
  --label "phase-1" \
  --body "$(cat <<'ISSUE_EOF'
## Context

The project needs a foundational architecture before feature work can begin.
This establishes the module structure, error handling patterns, and data flow
that all subsequent issues build on.

## Proposed approach

- Set up the directory structure: `src/`, `tests/`, `config/`
- Define the core types and traits/interfaces
- Establish error handling pattern (Result types, custom errors)
- Add a basic CLI entry point or server skeleton

## Tasks

- [ ] Create module/package structure
- [ ] Define core types
- [ ] Set up error handling
- [ ] Add entry point with basic arg parsing
- [ ] Verify `cargo build` / `npm run build` succeeds

## Acceptance criteria

- [ ] Project builds with zero warnings
- [ ] Module structure matches AGENTS.md description
- [ ] CI passes on this foundation
ISSUE_EOF
)" \
  | grep -o '[0-9]*$')
```

Create all issues and capture their numbers.

### 5b. Rewrite ROADMAP.md with issue links

Once all issues are created, rewrite ROADMAP.md so every item links to its GitHub issue. Use the same format as the user's established projects:

```markdown
- [ ] [#1 Project scaffolding and CI setup](https://github.com/<user>/<repo>/issues/1)
- [ ] [#2 Core architecture](https://github.com/<user>/<repo>/issues/2)
```

This format is what the `roadmap-sync.yml` workflow matches against with its `sed` pattern. The format must be exact: `- [ ] [#N Title](URL)`.

### 5c. Set up branch protection

Enable branch protection on `main` to require PRs and status checks:

```bash
# Enable auto-merge and auto-delete branches on merge
gh api repos/<user>/<repo> \
  -X PATCH \
  -f allow_auto_merge=true \
  -f delete_branch_on_merge=true

# Create branch protection rule
gh api repos/<user>/<repo>/branches/main/protection \
  -X PUT \
  --input - <<'EOF'
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["test"]
  },
  "enforce_admins": true,
  "required_pull_request_reviews": null,
  "restrictions": null
}
EOF
```

Adapt the `contexts` array to match the actual job name(s) in `ci.yml`. If the CI uses a matrix, use the job name (e.g., `"test"` — GitHub Actions expands matrix jobs under the job name).

**Note:** Branch protection requires the repo to be on a GitHub plan that supports it (Pro, Team, or Enterprise for private repos; free for public repos). If the API call fails because of plan limitations, warn the user and skip — don't block the rest of setup.

### 5d. Set up ROADMAP_PAT secret

The roadmap-sync workflow needs a `ROADMAP_PAT` secret (a fine-grained GitHub PAT with repo contents/issues/PRs permissions) to create PRs when issues are closed/reopened.

**Step 1 — Check if already set on this repo:**

```bash
gh secret list --repo <user>/<repo> | grep ROADMAP_PAT
```

If it exists, skip — done.

**Step 2 — Check the `ROADMAP_PAT` environment variable:**

```bash
echo "${ROADMAP_PAT:-(not set)}"
```

- **If the env var is set:** use it automatically — no user interaction needed:

```bash
echo "$ROADMAP_PAT" | gh secret set ROADMAP_PAT --repo <user>/<repo>
```

Verify with `gh secret list --repo <user>/<repo> | grep ROADMAP_PAT`, then move on silently.

- **If the env var is not set:** ask the user with `AskUserQuestion`:

> **Roadmap auto-sync needs a GitHub fine-grained PAT.** If you already have one you use across repos, paste it here. Otherwise:
>
> 1. Go to https://github.com/settings/tokens?type=beta → **Generate new token**
> 2. Name it `roadmap-sync`, select **All repositories**, and grant: **Contents** (Read and write), **Issues** (Read and write), **Pull requests** (Read and write)
> 3. Copy the token
>
> **Tip:** To skip this prompt on future repos, add `export ROADMAP_PAT=<your-token>` to your shell profile.
>
> Paste your token here (it will be stored as a repo secret, not logged):

Then set it:

```bash
echo "<pasted-token>" | gh secret set ROADMAP_PAT --repo <user>/<repo>
```

If the user declines or wants to do it later, that's fine — it'll be in `todo.txt`.

### 5e. Commit and push automation updates

```bash
git add ROADMAP.md
git commit -m "$(cat <<'EOF'
Link roadmap items to GitHub issues

Each ROADMAP.md checkbox now links to a tracked GitHub issue.
Roadmap checkboxes auto-update when issues are closed or reopened.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
git push
```

## 6. Architecture review

Before declaring setup complete, validate the scaffolded project with review agents. This catches structural issues before `/autopilot` starts building on a weak foundation.

### 6a. Run review agents in parallel

Launch these agents **in parallel** (single message, multiple Agent tool calls):

**Software Architect** (`subagent_type: "Software Architect"`):
- Review `ROADMAP.md`: Are phases ordered by dependency? Are items concrete enough to implement? Any missing foundational work?
- Review project structure: Does the directory layout match the tech stack's conventions? Any missing standard directories?
- Review `ci.yml`: Does it test what matters? Any missing steps for the stack?
- Produce a brief assessment with: approved items, suggested changes, and questions

**Code Reviewer** (`subagent_type: "Code Reviewer"`):
- Review all generated code files (not docs): `Cargo.toml`, `package.json`, `pyproject.toml`, CI workflow YAML, `.gitignore`, language scaffolding
- Flag blockers (incorrect configurations that would cause build/CI failures) and suggestions (improvements)

### 6b. Present findings

Show the combined results from both agents grouped by severity:

1. **Blockers** — must fix now (incorrect configs, missing dependencies, CI that won't pass)
2. **Suggestions** — recommended changes (roadmap reordering, missing scaffolding, convention mismatches)
3. **Questions** — ambiguities that should be resolved before building

### 6c. Apply fixes

If there are blockers or the user approves suggestions:
- Apply the changes directly (edit ROADMAP.md, adjust scaffolding, fix CI)
- If fixes change the roadmap structure or issue descriptions, update the corresponding GitHub issues too
- Commit and push:

```bash
git add -A
git commit -m "$(cat <<'EOF'
Refine project scaffolding based on architecture review

Applied fixes from Software Architect and Code Reviewer agents.

Co-Authored-By: Claude Opus 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
git push
```

If there are questions, ask the user with `AskUserQuestion` and adjust based on their answers.

If no blockers or suggestions — skip straight to the summary.

## 7. Generate todo.txt

Create a `todo.txt` in the project root with all the manual steps the user needs to complete to get fully operational. This file is **untracked** (already in `.gitignore`) — it's a personal checklist, not project documentation.

Build the list dynamically based on what the project actually needs. Only include items that weren't already completed during setup.

```
# TODO — manual setup steps for <project-name>
# This file is untracked (.gitignore) — it's your personal checklist.
# Delete it when you're done.

## Required

- [ ] Drop a project icon at assets/<project-name>-icon.png (referenced in README)

## Project-specific

<dynamically generated based on the project — examples below>
```

**Note:** Only include the ROADMAP_PAT item if the user skipped it during step 5d:
```
- [ ] Create ROADMAP_PAT secret for roadmap auto-sync:
      1. Go to https://github.com/settings/tokens?type=beta → Generate new fine-grained token
         Permissions: Contents (RW), Issues (RW), Pull requests (RW) — scope to All repositories
      2. Run: echo "<your-token>" | gh secret set ROADMAP_PAT --repo <user>/<repo>
      Tip: add `export ROADMAP_PAT=<token>` to your shell profile so future /setup runs set it automatically.
```

**Examples of project-specific items** (include only what's relevant):

- `- [ ] Create Stripe API keys and set STRIPE_SECRET_KEY in .env` — if they mentioned Stripe
- `- [ ] Set up Supabase project and add SUPABASE_URL + SUPABASE_ANON_KEY to .env` — if they mentioned Supabase
- `- [ ] Create OpenAI API key and set OPENAI_API_KEY in .env` — if they mentioned OpenAI/LLM
- `- [ ] Set up database: create Postgres instance and add DATABASE_URL to .env` — if they mentioned a database
- `- [ ] Register OAuth app and configure client ID/secret` — if they mentioned auth
- `- [ ] Set up DNS and configure custom domain` — if they mentioned deployment
- `- [ ] Create Apple Developer account for App Store distribution` — if it's an iOS app
- `- [ ] Install system dependencies: <list>` — if the stack requires system-level installs

End the file with:

```
## Get started

- [ ] Run /next to start working through the roadmap
```

## 8. Summary

After everything is created, show the user:

- The repo URL (`gh repo view --json url --jq .url`)
- A tree view of what was created (`find . -not -path './.git/*' -not -name '.git' | head -40 | sort`)
- Automation status:
  - Number of GitHub issues created
  - CI workflow (what it runs)
  - Roadmap sync (auto-checkbox updates)
  - Branch protection (enabled or skipped)
- **todo.txt** — remind the user: `Check todo.txt in the project root for manual setup steps (API keys, secrets, etc.)`
- Architecture review results (if step 6 ran): blockers fixed, suggestions applied, questions resolved
- Next steps (e.g., "Work through `todo.txt`", "Run `/next` to start building")
