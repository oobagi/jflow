#!/bin/bash
set -e

DOTFILES_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "Saving Claude Code configs to dotfiles..."

for dir in skills agents; do
  if [ -d "$HOME/.claude/$dir" ]; then
    rm -rf "$DOTFILES_DIR/claude/$dir"
    cp -R "$HOME/.claude/$dir" "$DOTFILES_DIR/claude/$dir"
    echo "  saved $dir/"
  fi
done

cp "$HOME/.claude/settings.json" "$DOTFILES_DIR/claude/settings.json"
echo "  saved settings.json"

echo ""
git -C "$DOTFILES_DIR" status --short
echo ""
echo "Review the changes above, then commit and push."
