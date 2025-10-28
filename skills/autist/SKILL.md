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

# Control reasoning depth
autist -r high <<'EOF'  # low|medium|high
[complex prompt needing deep analysis]
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

## Notes

- Automatically uses current working directory
- Session IDs shown after each run for follow-ups
- Can read files, search code, execute commands in your project
