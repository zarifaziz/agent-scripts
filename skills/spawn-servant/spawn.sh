#!/usr/bin/env bash
# spawn.sh - Spawn a servant agent in a detached tmux window
# Usage: spawn.sh <window-name> <agent-command> <task-prompt>
# Example: spawn.sh "fix-lint" "amp" "Fix all ESLint errors..."

set -euo pipefail

WINDOW_NAME="${1:-}"
AGENT_CMD="${2:-amp}"
TASK_PROMPT="${3:-}"

if [[ -z "$WINDOW_NAME" || -z "$TASK_PROMPT" ]]; then
    echo "Usage: spawn.sh <window-name> <agent-command> <task-prompt>"
    echo "Example: spawn.sh 'fix-lint' 'amp' 'Fix all ESLint errors in src/'"
    exit 1
fi

# Sanitize window name
WINDOW_NAME=$(echo "$WINDOW_NAME" | tr ' ' '-' | tr -cd '[:alnum:]-_' | cut -c1-30)

# Capture parent tmux context (if in tmux)
PARENT_SESSION=""
PARENT_WINDOW=""
IN_TMUX=false

if [[ -n "${TMUX:-}" ]]; then
    IN_TMUX=true
    # Use TMUX_PANE to get invoking pane context, not currently active one
    if [[ -n "${TMUX_PANE:-}" ]]; then
        PARENT_SESSION=$(tmux display-message -t "$TMUX_PANE" -p '#S')
        PARENT_WINDOW=$(tmux display-message -t "$TMUX_PANE" -p '#I')
    else
        PARENT_SESSION=$(tmux display-message -p '#S')
        PARENT_WINDOW=$(tmux display-message -p '#I')
    fi
fi

# Build return instructions (only if in tmux)
RETURN_INSTRUCTIONS=""
if $IN_TMUX; then
    RETURN_INSTRUCTIONS="

---
RETURN PROTOCOL:
When you complete this task, send your results back to parent session.
Run this command at the END of your work:

tmux send-keys -t \"${PARENT_SESSION}:${PARENT_WINDOW}\" \"# SERVANT REPORT from ${WINDOW_NAME}
## Status: [COMPLETED/FAILED/PARTIAL]
## Files Modified:
[list all files with absolute paths]
## Changes Summary:
[brief description of what was done]
## Verification:
[test results, build output, etc.]
## Issues/Notes:
[any blockers or things to note]
\" C-m

After sending the report, you may exit."
fi

# Check if window already exists
EXISTING_WINDOW=$(tmux list-windows -F '#I:#W' 2>/dev/null | grep ":${WINDOW_NAME}$" | cut -d: -f1 || true)
CURRENT_SESSION=$(tmux display-message -p '#S')

if [[ -n "$EXISTING_WINDOW" ]]; then
    echo "Reusing existing window:"
    CREATED_WINDOW="$EXISTING_WINDOW"
    REUSED=true
else
    # Create the window in detached mode
    tmux new-window -d -n "$WINDOW_NAME" "$AGENT_CMD"
    CREATED_WINDOW=$(tmux list-windows -F '#I:#W' | grep ":${WINDOW_NAME}$" | cut -d: -f1)
    REUSED=false
fi

if $REUSED; then
    echo "Reusing servant:"
else
    echo "Spawned servant:"
fi
echo "  Session: $CURRENT_SESSION"
echo "  Window: $CREATED_WINDOW ($WINDOW_NAME)"
if $IN_TMUX; then
    echo "  Parent: ${PARENT_SESSION}:${PARENT_WINDOW}"
fi
echo ""
echo "Commands:"
echo "  Switch to:  tmux select-window -t '$WINDOW_NAME'"
echo "  Kill:       tmux kill-window -t '$WINDOW_NAME'"
echo "  Send keys:  tmux send-keys -t '$WINDOW_NAME' 'message' C-m"

# Wait for agent to initialize (only if new window)
if ! $REUSED; then
    sleep 3
fi

# Send the task with return instructions appended
FULL_PROMPT="${TASK_PROMPT}${RETURN_INSTRUCTIONS}"
tmux send-keys -t "$WINDOW_NAME" "$FULL_PROMPT" C-m

# Output reference info for follow-up
echo ""
echo "---"
echo "SPAWN_SESSION=$CURRENT_SESSION"
echo "SPAWN_WINDOW=$CREATED_WINDOW"
echo "SPAWN_NAME=$WINDOW_NAME"
echo "SPAWN_REUSED=$REUSED"
