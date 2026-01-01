---
name: handoff
description: Spawn a parallel coding agent in a prise PTY when the user says "Run an agent", "Spawn an agent", "handoff", or similar. Use this instead of spawn-servant when running in prise (not tmux).
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Handoff Skill (Prise)

Use this skill to spawn a parallel subagent in a new prise PTY for independent tasks.

## Valid Subagents

- **Amp**: `amp`
- **Claude**: `claude --dangerously-skip-permissions`

## Companion Script

Use `handoff` command (symlinked to `~/.local/bin`):

### Script Usage

```bash
# Spawn a servant
handoff "<agent-cmd>" "<task-prompt>"

# Report back to parent (run from servant PTY)
handoff report <parent-pty-id> "<message>"
```

The script automatically:

- Captures parent PTY ID for return communication
- Spawns servant in new PTY (no focus switch)
- Appends return protocol instructions to the prompt
- Outputs spawn info (PTY ID) for follow-up tasks

### Script Output

```
=> PTY 14 spawned (parent: 1)
   kill: prise pty kill 14
HANDOFF_PTY=14
```

## Large Prompts

For multiline/large prompts, use a variable first (avoids quoting issues):

```bash
PROMPT=$(cat <<'EOF'
following: @T-019b08fb-e9d0-7032-905c-da18fcc2b7f8

Fix all ESLint errors in /Users/mac/project/src/

CONTEXT: CI is failing due to lint errors introduced in recent commits.

TASK:
1. Run eslint and capture all errors
2. Fix each error following existing code patterns
3. Do NOT disable rules or use eslint-ignore comments

VERIFY: Run 'npm run lint' - should exit 0 with no errors
EOF
)
handoff "amp" "$PROMPT"
```

Why variable-first is better:
- Avoids nested quoting hell
- Easier to debug (`echo "$PROMPT"` to verify)
- Prise has no length limit on send (unlike tmux)

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
handoff "amp" "Fix the authentication bug"
```

**GOOD prompt (comprehensive):**

```bash
handoff "amp" "following: @T-019b08fb-e9d0-7032-905c-da18fcc2b7f8 working off of that thread (use read_thread if any questions/confusions)

Fix the JWT token expiration bug in /Users/mac/project/src/auth/jwt.go

CONTEXT: Users are getting logged out unexpectedly. The issue is in the token refresh logic around line 145-160.

TASK:
1. Read /Users/mac/project/src/auth/jwt.go and understand the current refresh flow
2. The refreshToken function is not checking token expiry correctly - it compares timestamps wrong
3. Fix the comparison logic to use time.Before() instead of direct comparison

VERIFY: Run 'go test ./src/auth/...' - all tests should pass

REFERENCE: See similar fix pattern in /Users/mac/project/src/auth/session.go lines 80-95"
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

The script appends return instructions telling the servant to use:

```bash
handoff report <parent-pty-id> "<message>"
```

The report command:
- Reads parent PTY ID from the return instructions
- Sends report directly to parent PTY via `prise pty send`

Example servant report:
```bash
handoff report 1 "## Files Modified:
/Users/mac/project/src/auth.ts
## Changes Summary:
Fixed token validation bug
## Verification:
All tests pass"
```

## Follow-up Commands

After spawning, use the output variables for follow-up:

```bash
# Send additional instructions
prise pty send "$HANDOFF_PTY" "Also fix the warnings please"
prise pty send "$HANDOFF_PTY" Enter

# Check PTY status
prise pty list

# Capture PTY screen
prise pty capture "$HANDOFF_PTY"

# Kill the servant
prise pty kill "$HANDOFF_PTY"
```

## Prise vs Tmux

| Feature | spawn-servant (tmux) | handoff (prise) |
|---------|---------------------|-----------------|
| Identification | session:window | PTY ID |
| Send text | paste-buffer (line-by-line) | direct send (no limit) |
| Create window | `new-window -d -n name` | `pty spawn --cwd` |
| List | `list-windows` | `pty list` |
| Kill | `kill-window -t name` | `pty kill <id>` |

## Guidelines

- PTYs spawn in background (no focus switch)
- Keep prompts detailed - spawned agent has no parent context
- Use `prise pty kill <id>` to stop a subagent if needed
- Check status with `prise pty list`
- Capture output with `prise pty capture <id>`

## When to Use

- Parallel independent tasks (e.g., run tests while implementing feature)
- Long-running operations that don't need immediate results
- Multiple file/directory operations that don't overlap
- When running in prise instead of tmux
