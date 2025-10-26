---
name: local-librarian
description: Search and analyze local repositories with a specialized agent that explores codebases, understands architecture, and explains implementations.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Local Librarian

A specialized codebase understanding agent that helps search and analyze local repositories. It acts as your personal multi-repository code expert, exploring local filesystems to answer questions about architecture, functionality, and implementation patterns.

## CLI Paths

- Primary: `local-librarian`
- Fallback: `$HOME/.local/bin/local-librarian`

## Usage

The tool accepts prompts via stdin pipes or heredocs. **You must always specify the directory/scope to search in your prompt.**

### Basic Syntax

```bash
# Heredoc (recommended)
local-librarian <<EOF
Search ~/Coding/mathgaps-org to find how spans and logs from backend 
applications are processed and uploaded to Grafana. Look for OpenTelemetry
integration and Grafana Loki/Tempo configuration.
EOF

# Pipe from echo
echo "Search ~/Coding/my-project to find authentication middleware implementation" | local-librarian

# Pipe from file
cat question.txt | local-librarian
```

## Search Directory

When using local-librarian, you can either:
1. **Specify the directory** to search in your prompt (recommended for clarity)
2. **Let it default** to your current working directory (where you invoke the command)

✅ **Good examples:**
```bash
# Explicit directory (recommended)
local-librarian <<EOF
Search ~/Coding/mathgaps-org to find how the learning platform 
handles user enrollment and course progress tracking.
EOF
```

```bash
# Using current directory as default
cd ~/Coding/backend-services
local-librarian <<EOF
Find how the API gateway routes requests to microservices.
EOF
```

```bash
local-librarian <<EOF
Explore ~/Coding/frontend-app to find how the React components
handle state management and API calls.
EOF
```

⚠️ **Works but less clear:**
```bash
# Will search current directory - might be confusing if you forget where you are
local-librarian <<EOF
Find how authentication works
EOF
```

## How It Works

The local-librarian has access to local filesystem tools:
- List repositories (including git submodules)
- List directory contents
- Read file contents
- Find files by pattern (glob)
- Search code with regex
- Search git commit history
- View diffs and changes

It operates from your home directory (`~`) with read-only access.

## Example Queries

### Find specific functionality
```bash
local-librarian <<EOF
Search ~/Coding/mathgaps-org repositories to find how OpenTelemetry 
spans and logs are collected and exported to Grafana. Include:
- Span creation and attribute setting
- Log processing and formatting
- Grafana Loki/Tempo configuration
- Export/upload mechanisms
EOF
```

### Understand architecture
```bash
local-librarian <<EOF
Analyze ~/Coding/microservices-app to explain the overall architecture:
- How services communicate
- Database connections
- Message queue usage
- API endpoints and routing
EOF
```

### Trace implementation
```bash
local-librarian <<EOF
In ~/Coding/web-app, trace how user authentication flows from login 
form submission through JWT validation to protected route access.
EOF
```

### Debug or troubleshoot
```bash
local-librarian <<EOF
Search ~/Coding/api-service to find all error handling patterns,
specifically looking for how database errors are caught and reported.
EOF
```

## Tips for Best Results

1. **Be specific** about what you're looking for
2. **Always include** the directory to search
3. **Provide context** about what you're trying to achieve
4. **Mention specific technologies** if relevant (e.g., "OpenTelemetry", "Redis", "PostgreSQL")
5. **Ask for examples** if you want to see actual code snippets

## Notes

- The agent explores **local repositories only** (not GitHub)
- It has **read-only access** to your filesystem
- Works with git repositories, including **submodules**
- Best for **multi-step analysis** across multiple repositories
- Provides detailed explanations with code examples and file paths
