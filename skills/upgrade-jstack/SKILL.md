---
name: upgrade-jstack
description: >
  Check for and install jstack updates. Compares local version against remote,
  pulls latest changes, and re-runs the installer to update symlinks and settings.
user-invocable: true
argument-hint: >
  ["check" to only check without installing | "force" to reinstall even if up to date]
allowed-tools: Bash, Read, Glob, Grep
effort: low
---

# Upgrade jstack

Check for and install updates to the jstack skill/agent stack.

## 1. Locate installation

Check for the jstack installation:

```bash
JSTACK_DIR="${JSTACK_DIR:-$HOME/.jstack}"
```

Verify it exists and is a git repo:
- If `$JSTACK_DIR` doesn't exist or isn't a git repo, tell the user:
  > jstack is not installed. Install it with:
  > ```
  > git clone https://github.com/oobagi/jstack.git ~/.jstack && ~/.jstack/install.sh
  > ```
  Then stop.

Read the current version:
```bash
cat "$JSTACK_DIR/VERSION"
```

## 2. Check for updates

Fetch the latest remote state without modifying the working tree:

```bash
git -C "$JSTACK_DIR" fetch origin main
```

Compare versions:
```bash
LOCAL_VERSION=$(cat "$JSTACK_DIR/VERSION")
REMOTE_VERSION=$(git -C "$JSTACK_DIR" show origin/main:VERSION)
```

Check commit distance:
```bash
git -C "$JSTACK_DIR" log --oneline HEAD..origin/main
```

If the local version matches the remote and there are no new commits:
- Report: `jstack v$LOCAL_VERSION is up to date.`
- If `force` was NOT specified, stop here.

## 3. Show what changed

If there are updates available (or `force` was specified):

```bash
# Commit summary
git -C "$JSTACK_DIR" log --oneline HEAD..origin/main

# Changed skills
git -C "$JSTACK_DIR" diff --stat HEAD..origin/main -- skills/

# Changed agents
git -C "$JSTACK_DIR" diff --stat HEAD..origin/main -- agents/

# Changed hooks
git -C "$JSTACK_DIR" diff --stat HEAD..origin/main -- hooks/

# Changed settings template
git -C "$JSTACK_DIR" diff --stat HEAD..origin/main -- settings/

# Changed installer
git -C "$JSTACK_DIR" diff --stat HEAD..origin/main -- install.sh
```

Present a summary:

```
jstack update available: v0.1.0 → v0.2.0

  Changes:
    • 2 new skills: /docs, /design
    • 1 updated skill: /setup
    • 1 updated hook: rtk-rewrite.sh
    • Settings template updated

  Commits:
    abc1234 Enhance /setup with architecture review phase
    def5678 Add /docs skill for documentation sync
    ...
```

If `$ARGUMENTS` contains `check`, stop here — don't install.

## 4. Pull and reinstall

Pull the latest changes:

```bash
git -C "$JSTACK_DIR" pull --ff-only origin main
```

If the pull fails (e.g., local modifications), report the error and suggest:
```
Pull failed — you may have local modifications in ~/.jstack.
To force update: cd ~/.jstack && git reset --hard origin/main
Or to keep your changes: cd ~/.jstack && git stash && git pull && git stash pop
```

If the pull succeeds, re-run the installer:

```bash
bash "$JSTACK_DIR/install.sh"
```

## 5. Verify

After installation, verify that everything is linked correctly:

```bash
# Check all skill symlinks are valid
for skill in "$JSTACK_DIR"/skills/*/; do
  name=$(basename "$skill")
  link="$HOME/.claude/skills/$name/SKILL.md"
  if [ -L "$link" ] && [ -e "$link" ]; then
    echo "  ✓ $name"
  else
    echo "  ✗ $name — broken or missing symlink"
  fi
done

# Check all agent symlinks are valid
for agent in "$JSTACK_DIR"/agents/*.md; do
  name=$(basename "$agent")
  link="$HOME/.claude/agents/$name"
  if [ -L "$link" ] && [ -e "$link" ]; then
    echo "  ✓ $name"
  else
    echo "  ✗ $name — broken or missing symlink"
  fi
done
```

Report the final state:

```
jstack upgraded: v0.1.0 → v0.2.0

  Skills:  10 linked (2 new, 1 updated)
  Agents:  8 linked
  Hooks:   1 installed
  Settings: merged

  All symlinks verified ✓
```
