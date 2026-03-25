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

For each item in ROADMAP.md, create a GitHub issue using `gh issue create`. The issue title should match the roadmap item text. Add a brief body describing the task. Assign a label matching the roadmap phase (create labels first if needed).

```bash
# Example: create an issue and capture its number
ISSUE_NUM=$(gh issue create \
  --title "Core architecture" \
  --body "Set up the core architecture for the project." \
  --label "phase-1" \
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
# Enable auto-merge on the repo
gh api repos/<user>/<repo> \
  -X PATCH \
  -f allow_auto_merge=true

# Create branch protection rule
gh api repos/<user>/<repo>/branches/main/protection \
  -X PUT \
  --input - <<'EOF'
{
  "required_status_checks": {
    "strict": true,
    "contexts": ["test"]
  },
  "enforce_admins": false,
  "required_pull_request_reviews": null,
  "restrictions": null
}
EOF
```

Adapt the `contexts` array to match the actual job name(s) in `ci.yml`. If the CI uses a matrix, use the job name (e.g., `"test"` — GitHub Actions expands matrix jobs under the job name).

**Note:** Branch protection requires the repo to be on a GitHub plan that supports it (Pro, Team, or Enterprise for private repos; free for public repos). If the API call fails because of plan limitations, warn the user and skip — don't block the rest of setup.

### 5d. Commit and push automation updates

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

## 6. Summary

After everything is created, show the user:

- The repo URL (`gh repo view --json url --jq .url`)
- A tree view of what was created (`find . -not -path './.git/*' -not -name '.git' | head -40 | sort`)
- Automation status:
  - Number of GitHub issues created
  - CI workflow (what it runs)
  - Roadmap sync (auto-checkbox updates)
  - Branch protection (enabled or skipped)
- **ROADMAP_PAT setup**: The roadmap-sync workflow requires a `ROADMAP_PAT` repository secret — a GitHub Personal Access Token with `repo` scope. Tell the user:

  > **Action needed:** Create a `ROADMAP_PAT` secret for roadmap auto-sync.
  >
  > 1. Go to https://github.com/settings/tokens and create a fine-grained token with **Contents** and **Pull requests** read+write access for this repo
  > 2. Set it from the terminal:
  >    ```
  >    echo "<your-token>" | gh secret set ROADMAP_PAT --repo <user>/<repo>
  >    ```
  >    Or add it manually at the repo's **Settings > Secrets and variables > Actions**.
  >
  > Without this, the roadmap-sync workflow won't be able to create PRs or push changes.

- Next steps (e.g., "Drop an icon at `assets/<project-name>-icon.png`", "Run `/next` to start working through the roadmap")
