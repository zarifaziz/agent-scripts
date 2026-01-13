#!/usr/bin/env bash
# spawn.sh - Spawn a servant agent in a detached tmux window
# Usage: spawn-servant <window-name> <agent-command> <task-prompt>
#        spawn-servant report <status> <message>

set -euo pipefail

SPAWN_STATE_DIR="$HOME/.cache/spawn-servant"
mkdir -p "$SPAWN_STATE_DIR"

# ============================================================
# REPORT SUBCOMMAND
# ============================================================
if [[ "${1:-}" == "report" ]]; then
    TARGET="${2:-}"
    STATUS="${3:-COMPLETED}"
    MESSAGE="${4:-No details provided}"
    
    if [[ -z "$TARGET" || ! "$TARGET" =~ : ]]; then
        echo "ERROR: Missing or invalid target (format: session:window)"
        echo "Usage: spawn-servant report <session:window> <status> <message>"
        exit 1
    fi
    
    # Parse target
    PARENT_SESSION="${TARGET%%:*}"
    PARENT_WINDOW="${TARGET##*:}"
    
    # Get current window name for report header (use TMUX_PANE to get actual window, not active)
    if [[ -n "${TMUX_PANE:-}" ]]; then
        CURRENT_WINDOW=$(tmux display-message -t "$TMUX_PANE" -p '#W' 2>/dev/null || echo "servant")
    else
        CURRENT_WINDOW=$(tmux display-message -p '#W' 2>/dev/null || echo "servant")
    fi
    
    # Build report
    REPORT="# SERVANT REPORT from $CURRENT_WINDOW
## Status: $STATUS
$MESSAGE"
    
    # Send via tmux buffer with bracketed paste mode (-p) to avoid line-by-line issues
    REPORT_FILE=$(mktemp)
    printf '%s' "$REPORT" > "$REPORT_FILE"
    tmux load-buffer "$REPORT_FILE"
    tmux paste-buffer -p -t "${PARENT_SESSION}:${PARENT_WINDOW}"
    sleep 0.3
    tmux send-keys -t "${PARENT_SESSION}:${PARENT_WINDOW}" Enter
    rm -f "$REPORT_FILE"
    
    echo "Report sent to ${PARENT_SESSION}:${PARENT_WINDOW}"
    exit 0
fi

# ============================================================
# SPAWN SUBCOMMAND (default)
# ============================================================
WINDOW_NAME="${1:-}"
AGENT_CMD="${2:-amp}"
TASK_PROMPT="${3:-}"

# Validation with helpful errors
if [[ -z "$WINDOW_NAME" ]]; then
    echo "ERROR: Missing window name"
    echo "Usage: spawn-servant <name> <agent> <prompt>"
    echo "       spawn-servant report <status> <message>"
    exit 1
fi

if [[ -z "$TASK_PROMPT" ]]; then
    echo "ERROR: Missing task prompt"
    echo "Usage: spawn-servant <name> <agent> <prompt>"
    exit 1
fi

if ! command -v tmux &>/dev/null; then
    echo "ERROR: tmux not installed"
    exit 1
fi

if ! command -v "$AGENT_CMD" &>/dev/null; then
    echo "ERROR: Agent '$AGENT_CMD' not found in PATH"
    exit 1
fi

if [[ -z "${TMUX:-}" ]]; then
    echo "ERROR: Not in tmux session"
    echo "Run: tmux new -s main"
    exit 1
fi

# Sanitize window name
WINDOW_NAME=$(echo "$WINDOW_NAME" | tr ' ' '-' | tr -cd '[:alnum:]-_' | cut -c1-30)

# Capture parent tmux context
if [[ -n "${TMUX_PANE:-}" ]]; then
    PARENT_SESSION=$(tmux display-message -t "$TMUX_PANE" -p '#S')
    PARENT_WINDOW=$(tmux display-message -t "$TMUX_PANE" -p '#I')
else
    PARENT_SESSION=$(tmux display-message -p '#S')
    PARENT_WINDOW=$(tmux display-message -p '#I')
fi

# Save parent info for report subcommand
STATE_FILE="$SPAWN_STATE_DIR/$WINDOW_NAME"
cat > "$STATE_FILE" <<EOF
PARENT_SESSION="$PARENT_SESSION"
PARENT_WINDOW="$PARENT_WINDOW"
EOF

# Build return instructions with embedded parent info
RETURN_INSTRUCTIONS="---
RETURN PROTOCOL:
When you complete this task, report back using:

spawn-servant report ${PARENT_SESSION}:${PARENT_WINDOW} <STATUS> <MESSAGE>

Where STATUS is: COMPLETED, FAILED, or PARTIAL
And MESSAGE is a summary including:
- Files Modified (absolute paths)
- Changes Summary
- Verification results
- Any issues or blockers

Example:
spawn-servant report ${PARENT_SESSION}:${PARENT_WINDOW} COMPLETED \"## Files Modified:
/path/to/file.ts
## Changes Summary:
Fixed the auth bug by updating token validation
## Verification:
All tests pass
## Issues:
None\"

After sending the report, you may exit."

# Save return instructions to file
RETURN_FILE="$SPAWN_STATE_DIR/${WINDOW_NAME}-return.txt"
printf '%s' "$RETURN_INSTRUCTIONS" > "$RETURN_FILE"

# Check if window already exists
EXISTING_WINDOW=$(tmux list-windows -F '#I:#W' 2>/dev/null | grep ":${WINDOW_NAME}$" | cut -d: -f1 || true)
CURRENT_SESSION=$(tmux display-message -p '#S')

if [[ -n "$EXISTING_WINDOW" ]]; then
    CREATED_WINDOW="$EXISTING_WINDOW"
    REUSED=true
else
    tmux new-window -d -n "$WINDOW_NAME" "$AGENT_CMD"
    CREATED_WINDOW=$(tmux list-windows -F '#I:#W' | grep ":${WINDOW_NAME}$" | cut -d: -f1)
    REUSED=false
fi

# Wait for agent to initialize (only if new window)
if ! $REUSED; then
    sleep 3
fi

# Send the task prompt via paste-buffer, then @file for return instructions
PROMPT_FILE=$(mktemp)
printf '%s' "$TASK_PROMPT" > "$PROMPT_FILE"
tmux load-buffer "$PROMPT_FILE"
tmux paste-buffer -t "$WINDOW_NAME"
rm -f "$PROMPT_FILE"
sleep 0.3
# Add return instructions via @file reference (avoids control char issues)
tmux send-keys -t "$WINDOW_NAME" " @${RETURN_FILE}"
sleep 0.3
tmux send-keys -t "$WINDOW_NAME" Enter

# Compact output
if $REUSED; then
    echo "=> $WINDOW_NAME (reused, window $CREATED_WINDOW)"
else
    echo "=> $WINDOW_NAME (window $CREATED_WINDOW)"
fi
echo "   kill: tmux kill-window -t '$WINDOW_NAME'"
echo "SPAWN_NAME=$WINDOW_NAME"
