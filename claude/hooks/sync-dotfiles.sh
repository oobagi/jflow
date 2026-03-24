#!/bin/bash
# Auto-sync ~/.claude/skills and ~/.claude/agents to the dotfiles repo
# when Claude Code edits them. Runs as a PostToolUse hook — zero LLM cost.
#
# Reads JSON from stdin with the edited file path, checks if it's in
# skills/ or agents/, and if so copies it to the dotfiles repo and
# auto-commits + pushes.

DOTFILES_REPO="${DOTFILES_REPO:-$HOME/dotfiles}"
CLAUDE_DIR="$HOME/.claude"

# Read hook input (JSON on stdin)
INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# Bail if no file path or jq isn't available
[ -z "$FILE_PATH" ] && exit 0

# Only sync skills and agents directories
case "$FILE_PATH" in
  "$CLAUDE_DIR/skills/"*|"$CLAUDE_DIR/agents/"*)
    ;;
  *)
    exit 0
    ;;
esac

# Compute the relative path under ~/.claude/ (e.g., skills/next/SKILL.md)
REL_PATH="${FILE_PATH#$CLAUDE_DIR/}"
DEST="$DOTFILES_REPO/claude/$REL_PATH"

# Create parent dirs and copy
mkdir -p "$(dirname "$DEST")"
cp "$FILE_PATH" "$DEST"

# Auto-commit and push (silently, don't block Claude)
cd "$DOTFILES_REPO"
git add "claude/$REL_PATH" 2>/dev/null
if ! git diff --cached --quiet 2>/dev/null; then
  git commit -m "Auto-sync $REL_PATH" --no-gpg-sign -q 2>/dev/null
  git push -q 2>/dev/null &
fi

exit 0
