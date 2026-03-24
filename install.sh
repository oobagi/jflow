#!/bin/bash
set -e

DOTFILES_DIR="$(cd "$(dirname "$0")" && pwd)"

mkdir -p "$HOME/.claude"

echo "Installing Claude Code configs..."

# Copy directories (merge into existing, overwrite matching files)
for dir in skills agents hooks; do
  if [ -d "$DOTFILES_DIR/claude/$dir" ]; then
    cp -R "$DOTFILES_DIR/claude/$dir" "$HOME/.claude/"
    echo "  copied $dir/"
  fi
done

# Make hooks executable
if [ -d "$HOME/.claude/hooks" ]; then
  chmod +x "$HOME/.claude/hooks"/*.sh 2>/dev/null || true
fi

# Copy settings.json (back up existing if it differs)
if [ -f "$HOME/.claude/settings.json" ]; then
  if ! diff -q "$DOTFILES_DIR/claude/settings.json" "$HOME/.claude/settings.json" > /dev/null 2>&1; then
    echo "  backing up settings.json -> settings.json.bak"
    cp "$HOME/.claude/settings.json" "$HOME/.claude/settings.json.bak"
  fi
fi
cp "$DOTFILES_DIR/claude/settings.json" "$HOME/.claude/settings.json"
echo "  copied settings.json"

echo "Done!"
