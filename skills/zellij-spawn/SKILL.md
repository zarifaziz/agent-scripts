---
name: zellij-spawn
description: Spawn parallel coding agents in Zellij tabs. Use when asked to "run an agent", "spawn an agent", "handoff", or for parallel tasks in Zellij.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Zellij Spawn Skill

Spawn parallel subagents in new Zellij tabs for independent tasks.

## Valid Subagents

- **Amp**: `amp`
- **Claude**: `claude --dangerously-skip-permissions`

## Companion Script

Use `zellij-spawn` command (symlink to `~/.local/bin`):

```bash
# Symlink setup (one-time)
ln -sf ~/.config/agents/skills/zellij-spawn/scripts/zellij-spawn ~/.local/bin/zellij-spawn
```

### Script Usage

```bash
# Spawn a servant
zellij-spawn "<tab-name>" "<agent-cmd>" "<task-prompt>"

# Check status of all spawned agents
zellij-spawn status

# Report back to parent (run from spawned agent)
zellij-spawn report <STATUS> "<message>"
```

The script automatically:

- Captures parent Zellij tab for return communication
- Spawns servant in new tab (switches back to parent)
- Appends return protocol instructions to the prompt
- Outputs spawn info for follow-up tasks

### Script Output

```
=> fix-lint (parent: amp2)
   kill: zellij action go-to-tab-name 'fix-lint' && zellij action close-tab
   report: /tmp/zellij-spawn/amp2.report
SPAWN_NAME=fix-lint
```

## Large Prompts

For multiline/large prompts, use a variable first:

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
zellij-spawn "fix-lint" "amp" "$PROMPT"
```

## CRITICAL: Prompt Quality Requirements

The spawned agent has NO context from the parent session.

**Your prompt MUST include:**

1. **following: @<thread_id>** - MANDATORY: Link to parent thread at the START
2. **Full context** - What problem is being solved?
3. **Specific file paths** - Absolute paths to all relevant files
4. **Clear success criteria** - How does the agent know when it's done?
5. **Relevant code patterns** - If following existing patterns, specify which files
6. **Verification steps** - How to test/verify the work

**BAD prompt:**

```bash
zellij-spawn "fix-auth" "amp" "Fix the authentication bug"
```

**GOOD prompt:**

```bash
zellij-spawn "fix-auth" "amp" "following: @T-019b08fb-e9d0-7032-905c-da18fcc2b7f8

Fix the JWT token expiration bug in /Users/mac/project/src/auth/jwt.go

CONTEXT: Users are getting logged out unexpectedly. Issue in refresh logic lines 145-160.

TASK:
1. Read /Users/mac/project/src/auth/jwt.go
2. Fix refreshToken - use time.Before() instead of direct comparison

VERIFY: Run 'go test ./src/auth/...' - all tests should pass

REFERENCE: Similar fix in /Users/mac/project/src/auth/session.go lines 80-95"
```

**Note:** Get your thread ID from the Amp Thread URL in environment context.

## Return Protocol

The script appends return instructions. Spawned agent reports via:

```bash
zellij-spawn report <STATUS> "<message>"
```

Where STATUS is: `COMPLETED`, `FAILED`, or `PARTIAL`

Reports are written to `/tmp/zellij-spawn/<parent-tab>.report`

**Parent agent checks reports:**

```bash
cat /tmp/zellij-spawn/amp2.report
```

Example servant report:
```bash
zellij-spawn report COMPLETED "## Files Modified:
/Users/mac/project/src/auth.ts
## Changes Summary:
Fixed token validation bug
## Verification:
All tests pass"
```

## Checking Status

Run `zellij-spawn status` to see all spawned agents and their reports:

```bash
zellij-spawn status

# Output:
# === Spawned Agents Status ===
# Tab: fix-lint
#   Running: YES
#   Report: NO
#   Spawned: 2026-01-14T12:30:41+11:00
# 
# === Reports ===
# --- amp2.report ---
# STATUS: COMPLETED
# MESSAGE: Fixed all lint errors...
```

## Follow-up Commands

```bash
# Check if tab exists
zellij action query-tab-names | grep "$SPAWN_NAME"

# Switch to servant tab
zellij action go-to-tab-name "$SPAWN_NAME"

# Kill the servant (must switch first)
zellij action go-to-tab-name "$SPAWN_NAME" && zellij action close-tab

# Read reports from spawned agents
cat /tmp/zellij-spawn/*.report
```

## Zellij vs Tmux Comparison

| Feature | spawn-servant (tmux) | zellij-spawn |
|---------|---------------------|--------------|
| Identification | session:window | tab name |
| Send text | paste-buffer | write-chars |
| Create window | `new-window -d -n` | `new-tab --name` |
| List | `list-windows` | `query-tab-names` |
| Kill | `kill-window -t name` | switch + `close-tab` |
| Return protocol | send-keys to parent | file-based |

## Guidelines

- Tabs spawn and script switches back to parent automatically
- Keep prompts detailed - spawned agent has no parent context
- Check reports at `/tmp/zellij-spawn/<parent>.report`
- Use `query-tab-names` to see running servants

## When to Use

- Parallel independent tasks (e.g., run tests while implementing feature)
- Long-running operations that don't need immediate results
- Multiple file/directory operations that don't overlap
- QA/validation tasks in parallel with development
