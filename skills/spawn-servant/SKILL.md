---
name: spawn-servant
description: Spawn a parallel coding agent in a tmux window when the user says "Run an agent", "Spawn an agent", or similar.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Spawn Servant Skill

Use this skill to spawn a parallel subagent in a new tmux window for independent tasks.

## Valid Subagents

- **Amp**: `amp`
- **Claude**: `claude --dangerously-skip-permissions`

## Companion Script

Use `spawn-servant` command (symlinked to `~/.local/bin`):

### Script Usage

```bash
spawn-servant "<window-name>" "<agent-cmd>" "<task-prompt>"
```

The script automatically:

- Captures parent tmux session/window for return communication
- Spawns servant in detached window (no focus switch)
- Appends return protocol instructions to the prompt
- Outputs spawn info (session, window number) for follow-up tasks

### Script Output

```
Spawned servant:
  Session: main
  Window: 5 (fix-lint)
  Parent: main:2

Commands:
  Switch to:  tmux select-window -t 'fix-lint'
  Kill:       tmux kill-window -t 'fix-lint'
  Send keys:  tmux send-keys -t 'fix-lint' 'message' C-m

---
SPAWN_SESSION=main
SPAWN_WINDOW=5
SPAWN_NAME=fix-lint
```

Use `SPAWN_WINDOW` for follow-up commands to the servant.

## Example

Spawn using heredoc for complex prompts:

```bash
spawn-servant "fix-lint" "amp" "$(cat <<'EOF'
following: @T-019b08fb-e9d0-7032-905c-da18fcc2b7f8 working off of that thread (use read_thread if any questions/confusions)

Fix all ESLint errors in /Users/mac/project/src/

CONTEXT: CI is failing due to lint errors introduced in recent commits.

TASK:
1. Run eslint and capture all errors
2. Fix each error following existing code patterns
3. Do NOT disable rules or use eslint-ignore comments

VERIFY: Run 'npm run lint' - should exit 0 with no errors
EOF
)"
```

## CRITICAL: Prompt Quality Requirements

When spawning a servant, you MUST provide comprehensive, detailed prompts. The spawned agent has NO context from the parent session - it starts fresh.

**Your prompt MUST include:**

1. **following: @<thread_id>** - MANDATORY: Link to parent thread at the START so servant can read context
2. **Full context** - What problem is being solved? What's the background?
3. **Specific file paths** - Absolute paths to all relevant files
4. **Clear success criteria** - How does the agent know when it's done?
5. **Relevant code patterns** - If following existing patterns, specify which files to reference
6. **Verification steps** - How to test/verify the work (test commands, build commands)

**BAD prompt (incomplete):**

```bash
tmux send-keys -t "fix-auth" "Fix the authentication bug" C-m
```

**GOOD prompt (comprehensive):**

```bash
tmux send-keys -t "fix-auth" "following: @T-019b08fb-e9d0-7032-905c-da18fcc2b7f8 working off of that thread (use read_thread if any questions/confusions)

Fix the JWT token expiration bug in /Users/mac/project/src/auth/jwt.go

CONTEXT: Users are getting logged out unexpectedly. The issue is in the token refresh logic around line 145-160.

TASK:
1. Read /Users/mac/project/src/auth/jwt.go and understand the current refresh flow
2. The refreshToken function is not checking token expiry correctly - it compares timestamps wrong
3. Fix the comparison logic to use time.Before() instead of direct comparison

VERIFY: Run 'go test ./src/auth/...' - all tests should pass

REFERENCE: See similar fix pattern in /Users/mac/project/src/auth/session.go lines 80-95" C-m
```

**Note:** Get your current thread ID from the environment context (Amp Thread URL) and include it so the servant can use `read_thread` to understand the full conversation history if needed.

## QA/Validation/CodeReview Tasks

For review-type tasks:

- Instruct servant to **load the `web-qa` skill** for web/UI testing tasks
- **P0/Major errors**: STOP immediately and report back - don't keep hunting
- **Minor errors**: Collect and batch report at the end
- Focus on finding the first critical blocker for tight iteration cycles

Example addition to prompt:

```
Load the web-qa skill for browser testing.
If you find a critical bug or blocker, STOP and report immediately. Batch minor issues for end report.
```

## Return Protocol

The spawn.sh script automatically appends return instructions to the servant's prompt. When the servant completes, it will send a report back to your tmux window containing:

- **Status**: COMPLETED/FAILED/PARTIAL
- **Files Modified**: All files with absolute paths
- **Changes Summary**: What was done
- **Verification**: Test results, build output
- **Issues/Notes**: Any blockers or observations

This only works when you're running inside tmux. If not in tmux, return instructions are skipped.

## Follow-up Commands

After spawning, use the output variables for follow-up:

```bash
# Send additional instructions
tmux send-keys -t "$SPAWN_NAME" "Also fix the warnings please" C-m

# Check if window still exists
tmux list-windows | grep "$SPAWN_NAME"

# Kill the servant
tmux kill-window -t "$SPAWN_NAME"
```

## Guidelines

- Use `-d` flag to spawn quietly in background (no focus switch)
- Sanitize window titles: replace spaces with dashes, remove special characters
- Keep window titles short and descriptive (max 30 characters)
- Use `tmux kill-window -t "window-name"` to stop a subagent if needed
- Check status with `tmux list-windows`
- Switch to servant with `tmux select-window -t "window-name"` when needed

## When to Use

- Parallel independent tasks (e.g., run tests while implementing feature)
- Long-running operations that don't need immediate results
- Multiple file/directory operations that don't overlap
