---
name: local-librarian
description: Search and analyze local repositories with a specialized agent that explores codebases, understands architecture, and explains implementations.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Local Librarian

A specialized codebase understanding agent that searches and analyzes local repositories. Acts as your personal code expert for exploring filesystems, understanding architecture, and explaining implementations.

## CLI Paths

- Primary: `local-librarian`
- Fallback: `$HOME/.local/bin/local-librarian`

## Usage

```bash
# Specify directory in prompt (recommended)
local-librarian <<'EOF'
Search ~/Coding/mathgaps-org to find...
EOF

# Or use current directory as default
cd ~/Coding/my-project
local-librarian <<'EOF'
Find how authentication works
EOF
```

Prompts via **stdin only**. Operates from `~` with read-only access. Defaults to current working directory if no directory specified.

## How It Works

Local equivalent of GitHub Librarian with tools for:
- Finding repositories (including git submodules)
- Listing directories and reading files
- Searching code patterns (ripgrep/grep)
- Searching git commits and viewing diffs
- Glob pattern file matching

## Example: Detailed Search

```bash
local-librarian <<'EOF'
Search ~/Coding/mathgaps-org to find how OpenTelemetry spans and logs 
are collected and exported to Grafana.

## Context
Need to understand the complete observability pipeline - how traces and logs 
get from application code to Grafana dashboards.

## Looking For
- Where spans are created in application code
- Log formatting and structure
- OpenTelemetry configuration
- Grafana Loki and Tempo setup
- Data flow: app → collector → storage → visualization

## Questions
1. How are spans created and enriched with attributes?
2. What log format is used and how are logs correlated with traces?
3. Where is the OpenTelemetry collector configured?
4. How is Grafana configured to query Loki and Tempo?
EOF
```

## Example: Quick Searches

```bash
# Architecture overview
local-librarian <<'EOF'
Search ~/Coding/backend-services to explain the microservices architecture.
What services exist? How do they communicate? Database per service?
EOF

# Trace implementation
local-librarian <<'EOF'
Search ~/Coding/web-app to trace authentication flow from login form 
submission through JWT validation to protected routes.
EOF

# Find patterns
cd ~/Coding/api-service
local-librarian <<'EOF'
Find all error handling patterns. How are database errors caught and reported?
EOF

# Configuration
local-librarian <<'EOF'
Search ~/Coding/infrastructure to find how environment config works.
What format? Where are values defined? How to add new config?
EOF
```

## Best Practices

**✅ DO:**
- Specify directory explicitly or cd there first
- Provide context about what you're trying to understand
- Ask specific questions
- Use structured format (Context, Looking For, Questions)

**❌ DON'T:**
- Be vague ("Find auth stuff")
- Skip context
- Use command-line arguments for prompts
