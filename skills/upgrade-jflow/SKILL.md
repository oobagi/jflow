---
name: upgrade-jflow
description: >
  Check for and install jflow updates. Compares local version against remote,
  pulls latest changes, and re-runs the installer to update symlinks and settings.
user-invocable: true
argument-hint: >
  ["check" to only check without installing | "force" to reinstall even if up to date]
allowed-tools: Bash, Read, Glob, Grep
effort: low
---

# Upgrade jflow

Check for and install updates to the jflow skill/agent stack.

## 1. Locate installation

Check for the jflow installation:

```bash
JFLOW_DIR="${JFLOW_DIR:-$HOME/.jflow}"
```

Verify it exists and is a git repo:
- If `$JFLOW_DIR` doesn't exist or isn't a git repo, tell the user:
  > jflow is not installed. Install it with:
  > ```
  > git clone https://github.com/oobagi/jflow.git ~/.jflow && ~/.jflow/install.sh
  > ```
  Then stop.

Read the current version:
```bash
cat "$JFLOW_DIR/VERSION"
```

## 2. Check for updates

Fetch the latest remote state without modifying the working tree:

```bash
git -C "$JFLOW_DIR" fetch origin main
```

Compare versions:
```bash
LOCAL_VERSION=$(cat "$JFLOW_DIR/VERSION")
REMOTE_VERSION=$(git -C "$JFLOW_DIR" show origin/main:VERSION)
```

Check commit distance:
```bash
git -C "$JFLOW_DIR" log --oneline HEAD..origin/main
```

If the local version matches the remote and there are no new commits:
- Report using the standard output format:
```
═══════════════════════════════════════
  Upgrade jflow — Complete
═══════════════════════════════════════

  jflow v$LOCAL_VERSION is up to date.

  Next: no action needed

═══════════════════════════════════════
```
- If `force` was NOT specified, stop here.

## 3. Show what changed

If there are updates available (or `force` was specified):

```bash
# Commit summary
git -C "$JFLOW_DIR" log --oneline HEAD..origin/main

# Changed skills
git -C "$JFLOW_DIR" diff --stat HEAD..origin/main -- skills/

# Changed agents
git -C "$JFLOW_DIR" diff --stat HEAD..origin/main -- agents/

# Changed hooks
git -C "$JFLOW_DIR" diff --stat HEAD..origin/main -- hooks/

# Changed settings template
git -C "$JFLOW_DIR" diff --stat HEAD..origin/main -- settings/

# Changed installer
git -C "$JFLOW_DIR" diff --stat HEAD..origin/main -- install.sh
```

Present a summary using the standard output format:

```
═══════════════════════════════════════
  Upgrade jflow — Update Available
═══════════════════════════════════════

  Version: v0.1.0 → v0.2.0

  Changes:
    • 2 new skills: /docs, /design
    • 1 updated skill: /setup
    • 1 updated hook: rtk-rewrite.sh

  Commits:
    abc1234 Enhance /setup with architecture review phase
    def5678 Add /docs skill for documentation sync

  Next: run /upgrade-jflow to install

═══════════════════════════════════════
```

If `$ARGUMENTS` contains `check`, stop here — don't install.

## 4. Pull and reinstall

Pull the latest changes:

```bash
git -C "$JFLOW_DIR" pull --ff-only origin main
```

If the pull fails (e.g., local modifications), report the error and suggest:
```
Pull failed — you may have local modifications in ~/.jflow.
To force update: cd ~/.jflow && git reset --hard origin/main
Or to keep your changes: cd ~/.jflow && git stash && git pull && git stash pop
```

If the pull succeeds, re-run the installer:

```bash
bash "$JFLOW_DIR/install.sh"
```

## 5. Verify

After installation, verify that everything is linked correctly:

```bash
# Check all skill symlinks are valid
for skill in "$JFLOW_DIR"/skills/*/; do
  name=$(basename "$skill")
  link="$HOME/.claude/skills/$name/SKILL.md"
  if [ -L "$link" ] && [ -e "$link" ]; then
    echo "  ✓ $name"
  else
    echo "  ✗ $name — broken or missing symlink"
  fi
done

# Check all agent symlinks are valid
for agent in "$JFLOW_DIR"/agents/*.md; do
  name=$(basename "$agent")
  link="$HOME/.claude/agents/$name"
  if [ -L "$link" ] && [ -e "$link" ]; then
    echo "  ✓ $name"
  else
    echo "  ✗ $name — broken or missing symlink"
  fi
done
```

Report the final state using the standard output format:

```
═══════════════════════════════════════
  Upgrade jflow — Complete
═══════════════════════════════════════

  Version: v0.1.0 → v0.2.0

  Done:
    ✓ Pulled latest from origin/main
    ✓ Ran install.sh
    ✓ Verified all symlinks

  Skills:  10 linked (2 new, 1 updated)
  Agents:  8 linked
  Hooks:   1 installed

  Next: all skills updated — no action needed

═══════════════════════════════════════
```
