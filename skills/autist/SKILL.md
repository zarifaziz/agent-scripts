---
name: autist
description: Consult a project-focused agent with deep reasoning capabilities for code analysis, refactoring, and development tasks.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Autist

A project-aware agent with deep reasoning capabilities for code analysis, refactoring, planning, and complex development tasks.

## CLI Paths

- Primary: `autist`
- Fallback: `$HOME/.local/bin/autist`

## Usage

```bash
# Basic
autist <<'EOF'
[your prompt]
EOF

# Resume session
autist -s <session-id> <<'EOF'
[follow-up prompt]
EOF

```

Prompts must come via **stdin** (heredoc, pipe, or redirect). The tool uses your current working directory as context.

## When to Use

**✅ USE FOR:**

- Code reviews and architecture feedback
- Planning complex implementations or refactoring
- Bug analysis across multiple files
- Deep technical questions requiring reasoning

**❌ DON'T USE FOR:**

- Simple file reading (use `cat`, `less`)
- Basic searches (use `grep`, `rg`, `local-librarian`)
- Trivial code changes (just edit it)

## Example: Detailed Analysis

```bash
cd ~/project/backend

autist <<'EOF'
## Context

Building user authentication with email/password + OAuth (Google, GitHub).
Basic implementation done, need expert review before committing.

Files: auth/handler.go, auth/service.go, auth/jwt.go, auth/middleware.go

## Current Situation

- Email/password auth works
- Concerned about JWT security (token storage, expiration handling)
- OAuth integration feels messy (code duplication per provider)
- Rate limiting is per-IP, might need per-user too

## Questions

1. Review architecture - is separation of concerns good?
2. Security issues with JWT implementation?
3. How to structure OAuth providers better?
4. Rate limiting best practices here?
5. What tests should I add?

## Constraints

- Must support horizontal scaling (stateless)
- OWASP compliance required
EOF
```

## Example: Quick Questions

```bash
# Planning
autist <<'EOF'
Need to implement real-time collaboration for document editor.
What's the best architecture? WebSocket vs SSE?
How to handle conflicts (OT vs CRDT)?
EOF

# Bug hunting
autist <<'EOF'
Intermittent 500 errors on /api/orders (1 in 50 requests).
Logs show "database connection lost" but DB is healthy.
Only happens in production under load. Started after connection pooling changes.

Files: handlers/orders.go, services/order_service.go, repositories/order_repository.go

Where's the bug?
EOF

# Refactoring
autist <<'EOF'
user_service.go is 2000+ lines - unmaintainable.
How to break it into smaller services/packages?
Big bang vs incremental refactor?
EOF
```

## Handling Timeouts (300 Second Limit)

**⚠️ CORE ISSUE:** Autist may take longer than 300 seconds for complex analysis/refactoring tasks, causing tool execution to timeout.

**✅ SOLUTION:** The session ID is displayed **immediately** at the start of execution (not just at completion). If a timeout occurs:

1. **Copy the session ID** shown at the beginning of the output
2. **Continue the session** with a simple "continue" prompt:

```bash
autist -s <session-id> <<'EOF'
continue
EOF
```

This allows autist to complete its work without losing context or progress.

### Example Timeout Recovery

```bash
# Initial command times out after 300 seconds
autist <<'EOF'
Refactor the entire auth system for better testability
EOF

# Output shows:
# --------------
# To continue this session (in case of timeout), use: autist -s abc-123-def
# --------------
# [... timeout occurs ...]

# Simply continue the session:
autist -s abc-123-def <<'EOF'
continue
EOF

# Autist resumes and completes the work
```

**When to expect timeouts:**
- Large-scale refactoring across many files
- Deep code analysis with extensive reasoning
- Complex multi-step architectural changes
- Comprehensive bug hunting across large codebases

**Best practice:** For very large tasks, break them into smaller subtasks or be prepared to continue the session if timeout occurs.

## Notes

- Automatically uses current working directory
- Session IDs shown **twice**: immediately (for timeout recovery) and at completion (for follow-ups)
- Can read files, search code, execute commands in your project
- Always monitor the initial session ID output for timeout recovery
