#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$HOME/.local/bin"
CONFIG_DIR="$HOME/.config/amp-permissions"
AMP_SETTINGS="$HOME/.config/amp/settings.json"

echo "Installing amp sandbox tools..."

# 1. Create directories
mkdir -p "$BIN_DIR" "$CONFIG_DIR" "$(dirname "$AMP_SETTINGS")"

# 2. Symlink binaries
ln -sf "$SCRIPT_DIR/amp-permission-helper" "$BIN_DIR/"
ln -sf "$SCRIPT_DIR/amp-sandboxed" "$BIN_DIR/"
ln -sf "$SCRIPT_DIR/amp.sb" "$BIN_DIR/"
chmod +x "$SCRIPT_DIR/amp-permission-helper" "$SCRIPT_DIR/amp-sandboxed"
echo "Linked: amp-permission-helper, amp-sandboxed, amp.sb -> $BIN_DIR/"

# 3. Symlink config files
for f in readonly-commands.txt sensitive-paths.txt reject-patterns.txt always-allowed-paths.txt always-allowed-commands.txt always-ask-patterns.txt interpreters.txt; do
  ln -sf "$SCRIPT_DIR/config/$f" "$CONFIG_DIR/"
done
echo "Linked: config files -> $CONFIG_DIR/"

# 4. Backup and symlink amp settings.json
if [[ -f "$AMP_SETTINGS" && ! -L "$AMP_SETTINGS" ]]; then
  mv "$AMP_SETTINGS" "$AMP_SETTINGS.bak"
  echo "Backed up: $AMP_SETTINGS -> $AMP_SETTINGS.bak"
fi
ln -sf "$SCRIPT_DIR/config/settings.json" "$AMP_SETTINGS"
echo "Linked: settings.json -> $AMP_SETTINGS"

echo ""
echo "Done! Regular amp now uses permission-helper by default."
echo "For stricter sandbox mode: amp-sandboxed"
echo "Or add alias: alias amps='amp-sandboxed'"
[[ ":$PATH:" != *":$BIN_DIR:"* && ":$PATH:" != *":$BIN_DIR/:"* ]] && echo "" && echo "Note: Add to PATH: export PATH=\"\$HOME/.local/bin:\$PATH\""
