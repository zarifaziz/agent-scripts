---
name: project-oracle
description: Consult a project-focused agent with deep reasoning capabilities for code analysis, refactoring, and development tasks.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Project Oracle

A project-aware agent that helps with code analysis, refactoring, and development tasks in your current working directory.

## CLI Paths

- Primary: `project-oracle`
- Fallback: `$HOME/.local/bin/project-oracle`

## Usage

The tool accepts prompts via command-line arguments, stdin pipes, or heredocs.

### Basic Syntax

```bash
# Heredoc
project-oracle <<EOF
Review the database schema migrations
and suggest improvements
# Context
..
# Questions
..
# Requirements
..
# Constraints
...
# <So on>..
EOF
```

### Session Management

Use the `-s` or `--session` flag to continue previous conversations:

## NOTE: If not related to previous conversation, just omit -s id for fresh start

```bash
# With heredoc
project-oracle -s abc123 <<EOF
Wait.. about the thing you recommended..
I tried but...
.....<Whole context/question/etc>
EOF

```

## Notes

- The tool automatically uses your current working directory as context
- Session IDs are displayed after each invocation for follow-up
