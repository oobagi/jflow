#!/bin/bash
# Convenience wrapper — runs the installer in uninstall mode
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
exec "$SCRIPT_DIR/install.sh" --uninstall "$@"
