#!/bin/bash
set -euo pipefail

# jstack installer — symlink-based deployment with settings merging
# Usage: ./install.sh [--uninstall] [--dry-run] [--no-settings]

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
JSTACK_DIR="${JSTACK_DIR:-$HOME/.jstack}"
CLAUDE_DIR="$HOME/.claude"
VERSION=$(cat "$SCRIPT_DIR/VERSION")

# --- Argument parsing ---
UNINSTALL=false
DRY_RUN=false
NO_SETTINGS=false
for arg in "$@"; do
  case "$arg" in
    --uninstall)   UNINSTALL=true ;;
    --dry-run)     DRY_RUN=true ;;
    --no-settings) NO_SETTINGS=true ;;
    --help|-h)
      echo "jstack v$VERSION — Claude Code skill/agent stack"
      echo ""
      echo "Usage: ./install.sh [OPTIONS]"
      echo ""
      echo "Options:"
      echo "  --uninstall     Remove jstack symlinks, hooks, and settings entries"
      echo "  --dry-run       Show what would happen without making changes"
      echo "  --no-settings   Skip settings.json merge"
      echo "  --help          Show this help"
      exit 0
      ;;
  esac
done

# --- Helpers ---
info()  { echo "  $1"; }
warn()  { echo "  ⚠ $1"; }
err()   { echo "  ✗ $1" >&2; }

run() {
  if $DRY_RUN; then
    echo "  [dry-run] $*"
  else
    "$@"
  fi
}

# --- Dependency check ---
ensure_deps() {
  local missing=()
  for cmd in git jq; do
    if ! command -v "$cmd" &>/dev/null; then
      missing+=("$cmd")
    fi
  done
  if [ ${#missing[@]} -gt 0 ]; then
    err "Missing required tools: ${missing[*]}"
    err "Install them and re-run."
    exit 1
  fi
}

# --- Install repo to ~/.jstack ---
install_repo() {
  if [ "$SCRIPT_DIR" = "$JSTACK_DIR" ]; then
    info "Running from $JSTACK_DIR — skipping repo copy"
    return
  fi

  if [ -d "$JSTACK_DIR/.git" ]; then
    info "Updating existing installation at $JSTACK_DIR..."
    run git -C "$JSTACK_DIR" pull --ff-only origin main 2>/dev/null || true
  else
    info "Installing jstack to $JSTACK_DIR..."
    if [ -d "$JSTACK_DIR" ]; then
      warn "$JSTACK_DIR exists but is not a git repo — backing up to ${JSTACK_DIR}.bak"
      run mv "$JSTACK_DIR" "${JSTACK_DIR}.bak"
    fi
    run cp -R "$SCRIPT_DIR" "$JSTACK_DIR"
    # Initialize as a git repo pointing to the remote
    run git -C "$JSTACK_DIR" remote set-url origin "https://github.com/oobagi/jstack.git" 2>/dev/null || true
  fi
}

# --- Symlink skills ---
install_skills() {
  local count=0
  run mkdir -p "$CLAUDE_DIR/skills"

  for skill_dir in "$JSTACK_DIR"/skills/*/; do
    [ -d "$skill_dir" ] || continue
    local name=$(basename "$skill_dir")
    local target="$CLAUDE_DIR/skills/$name"

    run mkdir -p "$target"
    run ln -sf "$JSTACK_DIR/skills/$name/SKILL.md" "$target/SKILL.md"
    count=$((count + 1))
  done
  info "Linked $count skills"
}

# --- Symlink agents ---
install_agents() {
  local count=0
  run mkdir -p "$CLAUDE_DIR/agents"

  for agent_file in "$JSTACK_DIR"/agents/*.md; do
    [ -f "$agent_file" ] || continue
    local name=$(basename "$agent_file")

    run ln -sf "$JSTACK_DIR/agents/$name" "$CLAUDE_DIR/agents/$name"
    count=$((count + 1))
  done
  info "Linked $count agents"
}

# --- Copy hooks ---
install_hooks() {
  local count=0
  run mkdir -p "$CLAUDE_DIR/hooks"

  for hook_file in "$JSTACK_DIR"/hooks/*.sh; do
    [ -f "$hook_file" ] || continue
    local name=$(basename "$hook_file")

    run cp "$JSTACK_DIR/hooks/$name" "$CLAUDE_DIR/hooks/$name"
    run chmod +x "$CLAUDE_DIR/hooks/$name"
    count=$((count + 1))
  done
  info "Installed $count hooks"
}

# --- Merge settings ---
merge_settings() {
  if $NO_SETTINGS; then
    info "Skipping settings merge (--no-settings)"
    return
  fi

  local base="$JSTACK_DIR/settings/base.json"
  local target="$CLAUDE_DIR/settings.json"

  if [ ! -f "$base" ]; then
    warn "No base settings found at $base — skipping"
    return
  fi

  # Create target if missing
  if [ ! -f "$target" ]; then
    run cp "$base" "$target"
    info "Created settings.json from jstack defaults"
    return
  fi

  if $DRY_RUN; then
    info "[dry-run] Would merge settings from $base into $target"
    return
  fi

  # Backup existing settings
  cp "$target" "$CLAUDE_DIR/settings.json.pre-jstack"
  info "Backed up settings.json → settings.json.pre-jstack"

  # Merge using jq:
  # 1. Clean legacy artifacts from user settings
  # 2. Additive merge for hooks, env, plugins
  # 3. User scalar settings win if already set
  local merged
  merged=$(jq -s '
    .[0] as $base | .[1] as $user |

    # Step 1: Clean legacy artifacts from user config
    ($user
      # Remove sync-dotfiles PostToolUse hooks
      | if .hooks.PostToolUse then
          .hooks.PostToolUse |= map(
            select(.hooks | all(.command | test("sync-dotfiles") | not))
          )
          | if .hooks.PostToolUse | length == 0 then del(.hooks.PostToolUse) else . end
        else . end
      # Remove old rtk-rewrite PreToolUse hooks (will be re-added from base)
      | if .hooks.PreToolUse then
          .hooks.PreToolUse |= map(
            select(.hooks | all(.command | test("rtk-rewrite") | not))
          )
          | if .hooks.PreToolUse | length == 0 then del(.hooks.PreToolUse) else . end
        else . end
      # Remove legacy env var
      | if .env.DOTFILES_REPO then del(.env.DOTFILES_REPO) else . end
    ) as $cleaned |

    # Step 2: Merge hooks (concatenate arrays per hook type, deduplicate by matcher)
    (
      (($cleaned.hooks // {}) | keys) + (($base.hooks // {}) | keys) | unique
    ) as $hook_types |
    (
      $hook_types | map({
        key: .,
        value: ((($cleaned.hooks[.] // []) + ($base.hooks[.] // [])) | unique_by(.matcher))
      }) | from_entries
    ) as $merged_hooks |

    # Step 3: Build final object — user settings as base, overlay jstack additions
    $cleaned * {
      hooks: $merged_hooks,
      env: (($cleaned.env // {}) + ($base.env // {})),
      enabledPlugins: (($cleaned.enabledPlugins // {}) + ($base.enabledPlugins // {}))
    }
    # Set scalar defaults only if not already set by user
    | if .effortLevel == null then .effortLevel = $base.effortLevel else . end
    | if .promptSuggestionEnabled == null then .promptSuggestionEnabled = $base.promptSuggestionEnabled else . end
    | if .skipDangerousModePermissionPrompt == null then .skipDangerousModePermissionPrompt = $base.skipDangerousModePermissionPrompt else . end
    # Store jstack version for upgrade tracking
    | ._jstack_version = $base._jstack_version
  ' "$base" "$target")

  echo "$merged" | jq '.' > "$target"
  info "Merged settings (jstack v$VERSION)"
}

# --- Uninstall ---
uninstall() {
  echo "Uninstalling jstack..."

  # Remove skill symlinks pointing to jstack
  if [ -d "$CLAUDE_DIR/skills" ]; then
    for skill_dir in "$CLAUDE_DIR"/skills/*/; do
      [ -d "$skill_dir" ] || continue
      local link="$skill_dir/SKILL.md"
      if [ -L "$link" ] && readlink "$link" | grep -q "\.jstack\|jstack" 2>/dev/null; then
        run rm "$link"
        run rmdir "$skill_dir" 2>/dev/null || true
        info "Removed skill: $(basename "$skill_dir")"
      fi
    done
  fi

  # Remove agent symlinks pointing to jstack
  if [ -d "$CLAUDE_DIR/agents" ]; then
    for agent_file in "$CLAUDE_DIR"/agents/*.md; do
      [ -f "$agent_file" ] || [ -L "$agent_file" ] || continue
      if [ -L "$agent_file" ] && readlink "$agent_file" | grep -q "\.jstack\|jstack" 2>/dev/null; then
        run rm "$agent_file"
        info "Removed agent: $(basename "$agent_file")"
      fi
    done
  fi

  # Remove jstack-installed hooks
  for hook in rtk-rewrite.sh; do
    if [ -f "$CLAUDE_DIR/hooks/$hook" ]; then
      run rm "$CLAUDE_DIR/hooks/$hook"
      info "Removed hook: $hook"
    fi
  done

  # Restore settings backup if it exists
  if [ -f "$CLAUDE_DIR/settings.json.pre-jstack" ]; then
    run cp "$CLAUDE_DIR/settings.json.pre-jstack" "$CLAUDE_DIR/settings.json"
    info "Restored settings.json from pre-jstack backup"
  fi

  echo ""
  echo "jstack uninstalled. ~/.jstack/ was left in place."
  echo "To fully remove: rm -rf ~/.jstack"
}

# --- Main ---
main() {
  echo "jstack v$VERSION"
  echo ""

  ensure_deps

  if $UNINSTALL; then
    uninstall
    exit 0
  fi

  install_repo
  install_skills
  install_agents
  install_hooks
  merge_settings

  echo ""
  echo "jstack v$VERSION installed successfully!"
  echo ""
  echo "  Skills:   $(ls -1d "$JSTACK_DIR"/skills/*/ 2>/dev/null | wc -l | tr -d ' ') skills linked"
  echo "  Agents:   $(ls -1 "$JSTACK_DIR"/agents/*.md 2>/dev/null | wc -l | tr -d ' ') agents linked"
  echo "  Hooks:    rtk-rewrite.sh"
  echo "  Settings: merged into ~/.claude/settings.json"
  echo ""
  echo "  Install location: $JSTACK_DIR"
  echo "  Upgrade: /upgrade-jstack"
}

main
