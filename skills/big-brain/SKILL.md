---
name: big-brain
description: Consult the Big Brain expert for deep technical analysis, architectural decisions, refactoring strategies, debugging, and code reviews using Claude Opus.
allowed-tools:
  - bash
metadata:
  version: "1.0"
  model: "anthropic/claude-opus-4-5"
---

# Big Brain

The Big Brain is an expert technical consultant powered by Claude Opus 4, specialized in deep thinking, comprehensive analysis, and strategic technical decisions. Use this when you need thorough, well-reasoned recommendations.

## CLI Paths

- Primary: `big-brain`
- Fallback: `$HOME/.local/bin/big-brain`

## Usage

```bash
# Basic consultation
big-brain <<'EOF'
[your question or problem]
EOF

# From command line
big-brain "How should we implement caching?"

# Piped input
echo "What's the best approach for rate limiting?" | big-brain

# Continue a session (for follow-ups or timeout recovery)
big-brain -s ses_abc123 <<'EOF'
continue with implementation details
EOF
```

Prompts can come via **stdin** (heredoc, pipe, redirect) or command-line arguments. The tool uses your current working directory as context and will research the codebase thoroughly.

## Session Continuation

The tool displays the session ID at start and completion. Use `-s SESSION_ID` to:
- Continue a conversation with follow-up questions
- Recover from timeouts (session ID shown before timeout)
- Build on previous analysis

**Oracle Mode**: Big-brain operates as a read-only advisory oracle - it analyzes and advises but does not make changes itself.

## When to Use

**✅ USE FOR:**
- Feature implementation planning and architectural decisions
- Refactoring strategies and design pattern selection
- Complex debugging and root cause analysis
- Code review and quality assessment
- Performance optimization recommendations
- Technical tradeoff analysis
- Design decisions requiring deep reasoning

**❌ DON'T USE FOR:**
- Simple file reading (use `cat`, `less`)
- Basic searches (use `grep`, `rg`, `local-librarian`)
- Quick factual lookups
- Simple code changes (just edit it)

## Example: Architecture Decision

```bash
cd ~/project

big-brain <<'EOF'
## Context

We're building a notification system that needs to:
- Send emails, SMS, push notifications, webhooks
- Handle 100k+ notifications/day
- Support scheduling and retries
- Track delivery status

## Question

Should we build a custom notification service or use a third-party like SendGrid/Twilio?
What architecture would you recommend?

## Constraints

- Must be reliable (99.9% uptime)
- Budget: $500/month
- Team: 3 backend engineers (Go experience)
- Timeline: 2 months

What are the tradeoffs and your recommendation?
EOF
```

## Example: Refactoring Consultation

```bash
big-brain <<'EOF'
We have a monolithic user service (3000+ lines) that handles:
- Authentication (JWT, OAuth, sessions)
- User profiles and preferences
- Permissions and roles
- Account lifecycle (signup, deletion, suspension)
- Email verification and password reset

It's becoming hard to maintain and test. How should we refactor this?

Options we're considering:
1. Split into microservices (auth, profile, permissions)
2. Keep monolith but modularize into packages
3. Extract just auth into separate service

What do you recommend and why?
EOF
```

## Example: Debugging Strategy

```bash
cd backend/api

big-brain <<'EOF'
## Problem

Production API showing intermittent 20s+ response times (normally <100ms).
Happens randomly, can't reproduce locally.

## What We Know

- Started after deploying new caching layer (Redis)
- Only affects /api/v2/dashboard endpoint
- Database queries are fast (<50ms)
- Redis connection pool: 10 max connections
- Traffic: ~500 req/min peak

## Suspects

- Redis connection exhaustion?
- Slow cache deserialization?
- Lock contention somewhere?

What debugging approach would you take? What should we look for?
EOF
```

## Example: Code Review

```bash
big-brain <<'EOF'
Review this authentication middleware implementation:

Files to review:
- middleware/auth.go (150 lines)
- services/jwt_service.go (200 lines)
- handlers/login.go (100 lines)

Concerns:
1. Security - are we handling tokens safely?
2. Performance - too many DB calls per request?
3. Error handling - proper messages without leaking info?
4. Testing - what coverage gaps exist?

Please provide a comprehensive review with specific recommendations.
EOF
```

## Example: Technology Selection

```bash
big-brain <<'EOF'
Choosing a database for our analytics system:

Requirements:
- Store 10M+ events/day (user actions, system logs)
- Query patterns: time-range filters, aggregations, grouping
- Retention: 2 years (older data archived)
- Real-time dashboards (sub-second queries)

Options:
1. PostgreSQL + TimescaleDB
2. ClickHouse
3. Elasticsearch
4. Custom solution (Kafka + Druid)

We have PostgreSQL expertise but no experience with others.
What would you recommend considering team skills, scalability, and operational complexity?
EOF
```

## Example: Performance Optimization

```bash
big-brain <<'EOF'
Our GraphQL API has performance issues:

Problem:
- Dashboard query takes 5-8 seconds
- Fetches: user, courses (20+), progress for each course, recommendations
- 80+ database queries per request (N+1 problem)
- DataLoader implemented but still slow

Current stack:
- Go GraphQL (gqlgen)
- PostgreSQL
- Redis for caching

What optimization strategies would you recommend?
Should we denormalize? Add materialized views? Cache at which layer?
EOF
```

## Output Format

The Big Brain provides structured, comprehensive responses including:
- **Problem Summary**: Clear understanding of your request
- **Analysis**: Deep investigation of the current state and constraints
- **Recommendations**: Specific, actionable advice with rationale
- **Tradeoffs**: Pros and cons of different approaches
- **Implementation Notes**: Key considerations and gotchas
- **Next Steps**: Clear path forward

## Notes

- Automatically uses current working directory as context
- Will research your codebase thoroughly before responding
- Provides thoughtful, well-reasoned analysis (not quick answers)
- Output is clean - intermediate research logs are filtered out
- Full logs saved to `~/.cache/scripts/big-brain/` for debugging
