#!/bin/bash
set -e

DOTFILES_DIR="$(cd "$(dirname "$0")" && pwd)"

link() {
  local src="$1" dst="$2"
  mkdir -p "$(dirname "$dst")"
  if [ -L "$dst" ]; then
    rm "$dst"
  elif [ -e "$dst" ]; then
    echo "  backing up $dst -> $dst.bak"
    mv "$dst" "$dst.bak"
  fi
  ln -s "$src" "$dst"
  echo "  $dst -> $src"
}

echo "Linking Claude Code configs..."
for item in skills agents hooks CLAUDE.md RTK.md settings.json; do
  link "$DOTFILES_DIR/claude/$item" "$HOME/.claude/$item"
done

echo "Done!"
