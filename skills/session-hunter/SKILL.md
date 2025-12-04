---
name: session-hunter
description: Hunt down Amp sessions by code changes, file paths, or function names.
allowed-tools:
  - bash
metadata:
  version: "1.0"
  model: "anthropic/claude-haiku-4-5"
---

# Session Hunter

Find Amp sessions where specific code changes happened. Searches `~/.amp/file-changes/` by file paths, function names, or code patterns.

## Usage

```bash
session-hunter "Find session where validateToken was added"
session-hunter "Which session modified bump.go in updog?"
session-hunter -s ses_abc123 "follow-up query"  # continue session
```

## Limitations

Session-hunter finds _where_ and _what_ changed, but not _why_ or whether it's correct.

**Idiomatic workflow:**

1. `session-hunter` → finds thread ID (e.g., `T-abc123...`)
2. `read_thread` → extracts full context: problem, rationale, edge cases

This combo distinguishes **intended behavior** from **actual behavior** - crucial for debugging flawed implementations.
