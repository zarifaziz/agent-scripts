#!/usr/bin/env bash
# install.sh - Install amp thread picker scripts and indexer service
# Usage: ./install.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="$HOME/.local/bin"
LAUNCHAGENTS_DIR="$HOME/Library/LaunchAgents"
PLIST_NAME="com.amp.thread-indexer.plist"

echo "Installing amp-thread-picker tools..."

# Create directories
mkdir -p "$BIN_DIR"
mkdir -p "$LAUNCHAGENTS_DIR"

# Symlink scripts
for script in amp-thread-picker amp-live amp-thread-indexer; do
    src="$SCRIPT_DIR/$script"
    dest="$BIN_DIR/$script"
    
    # Force overwrite with -f and -n to not follow existing symlink
    ln -sfn "$src" "$dest"
    echo "  Linked: $dest -> $src"
done

# Install plist (with HOME substitution)
plist_src="$SCRIPT_DIR/$PLIST_NAME"
plist_dest="$LAUNCHAGENTS_DIR/$PLIST_NAME"

# Unload existing service if running
if launchctl list | grep -q "com.amp.thread-indexer"; then
    echo "  Unloading existing service..."
    launchctl bootout "gui/$(id -u)/com.amp.thread-indexer" 2>/dev/null || true
fi

# Create plist with HOME expanded
sed "s|__HOME__|$HOME|g" "$plist_src" > "$plist_dest"
echo "  Installed: $plist_dest"

# Load the service
echo "  Loading indexer service..."
launchctl bootstrap "gui/$(id -u)" "$plist_dest" 2>/dev/null || true

# Initial index if DB doesn't exist
if [[ ! -f "$HOME/.cache/amp_threads.db" ]]; then
    echo "  Running initial index (this may take a moment)..."
    "$BIN_DIR/amp-thread-indexer" --force
else
    echo "  Database exists, running incremental sync..."
    "$BIN_DIR/amp-thread-indexer" &
fi

echo ""
echo "Installation complete!"
echo ""
echo "Scripts installed:"
echo "  - amp-thread-picker : Search all amp threads (bind to prefix-@)"
echo "  - amp-live          : Switch to live amp sessions (bind to prefix-a)"
echo "  - amp-thread-indexer: Background indexer (runs every 5 min)"
echo ""
echo "Add to ~/.tmux.conf:"
echo '  bind-key @ popup -w 70% -h 70% -E "$HOME/.local/bin/amp-thread-picker"'
echo '  bind-key a popup -w 85% -h 75% -E "$HOME/.local/bin/amp-live"'
echo ""
echo "Force re-index: amp-thread-indexer --force"
