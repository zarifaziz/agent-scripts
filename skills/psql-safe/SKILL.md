---
name: psql-safe
description: Execute SQL queries against PostgreSQL/CockroachDB databases. Use when running read queries, testing write operations, or working with CockroachDB databases.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# PostgreSQL/CockroachDB Safe Query Tool

## Overview

Execute SQL queries against PostgreSQL/CockroachDB databases with two modes:

1. **Default mode**: Read-only queries
2. **Session mode**: Safe write testing (auto-rollback)

## CLI Paths

- Primary: `psql-safe`
- Fallback: `/Users/mac/.local/bin/psql-safe`

## Basic Usage

### 1. Running Against Teaching Dev Database

Query teaching database using the `teaching-dev` preset:

```bash
# Count all workspaces
psql-safe --preset teaching-dev "SELECT COUNT(*) as total FROM workspaces"

# List recent workspaces
psql-safe --preset teaching-dev \
  "SELECT id, name, created_at FROM workspaces ORDER BY created_at DESC LIMIT 10"

# Query classes with student counts
psql-safe --preset teaching-dev \
  "SELECT c.id, c.name, COUNT(s.id) as student_count
   FROM classes c
   LEFT JOIN students s ON c.id = s.class_id
   GROUP BY c.id, c.name
   ORDER BY student_count DESC
   LIMIT 20"
```

### 2. Querying Teachers and Workspace Users

```bash
# List workspace users with their roles
psql-safe --preset teaching-dev \
  "SELECT wu.id, wu.user_id, wu.workspace_id, wu.role, wu.created_at
   FROM workspace_users wu
   ORDER BY wu.created_at DESC
   LIMIT 10"

# Count users by role
psql-safe --preset teaching-dev \
  "SELECT role, COUNT(*) as count
   FROM workspace_users
   GROUP BY role"

# Find workspaces for a specific user
psql-safe --preset teaching-dev \
  --params '{"userId":"user-123"}' \
  "SELECT w.id, w.name, wu.role
   FROM workspaces w
   JOIN workspace_users wu ON w.id = wu.workspace_id
   WHERE wu.user_id = :userId"
```

### 3. Querying Students and Classes

```bash
# List students in a class
psql-safe --preset teaching-dev \
  --params '{"classId":"class-123"}' \
  "SELECT s.id, s.first_name, s.last_name, s.email, s.created_at
   FROM students s
   WHERE s.class_id = :classId
   ORDER BY s.last_name, s.first_name"

# Count students per workspace
psql-safe --preset teaching-dev \
  "SELECT w.id, w.name, COUNT(s.id) as student_count
   FROM workspaces w
   LEFT JOIN classes c ON w.id = c.workspace_id
   LEFT JOIN students s ON c.id = s.class_id
   GROUP BY w.id, w.name
   ORDER BY student_count DESC"

# Find classes without students
psql-safe --preset teaching-dev \
  "SELECT c.id, c.name, c.workspace_id
   FROM classes c
   LEFT JOIN students s ON c.id = s.class_id
   WHERE s.id IS NULL"
```

### 4. Using Session Mode

Session mode allows safe write testing with automatic rollback.

#### Testing Write Operations

```bash
# Create a test workspace (auto-creates session "test-workspace")
psql-safe --preset teaching-dev --session test-workspace \
  "INSERT INTO workspaces (name, type, created_at)
   VALUES ('Test Workspace', 'school', NOW())
   RETURNING id, name"

# Add a class to the test workspace
psql-safe --preset teaching-dev --session test-workspace \
  "INSERT INTO classes (name, workspace_id, created_at)
   VALUES ('Test Class', (SELECT id FROM workspaces WHERE name = 'Test Workspace'), NOW())
   RETURNING id, name"

# Verify the data exists in session
psql-safe --preset teaching-dev --session test-workspace \
  "SELECT w.name as workspace, c.name as class
   FROM workspaces w
   JOIN classes c ON w.id = c.workspace_id
   WHERE w.name = 'Test Workspace'"

# CRITICAL: Verify database is UNCHANGED (all rolled back!)
psql-safe --preset teaching-dev \
  "SELECT COUNT(*) FROM workspaces WHERE name = 'Test Workspace'"
# Returns {"count": 0} - Nothing was committed!

# Drop the session without committing
psql-safe --drop test-workspace
```

#### Committing Session Changes

```bash
# When ready to commit
psql-safe --preset teaching-dev --session test-workspace --commit
```

### 5. Using Local Database

```bash
# Query local CockroachDB (no preset needed)
psql-safe --preset teaching-local "SELECT version()"

# Or specify connection manually
psql-safe --host localhost --port 26257 --database defaultdb --user root \
  --ssl-mode disable "SELECT * FROM workspaces LIMIT 5"
```

## Session Management

List active sessions:

```bash
psql-safe --list
```

Show session queries:

```bash
psql-safe --show test-workspace
```

Drop session:

```bash
psql-safe --drop test-workspace
```

## Key Points

1. **Flags before query**: All flags must come BEFORE the query string
2. **Default rollback**: Session mode automatically rolls back all changes
3. **Session replay**: Each session query replays previous queries in that session
4. **Read-only by default**: Without `--session`, all queries are read-only
5. **Presets available**: `teaching-dev`, `teaching-local`
6. **JSON output**: Results are JSON arrays, pipe to `jq` for processing

## Common Patterns

### Explore Schema

```bash
# List all tables
psql-safe --preset teaching-dev \
  "SELECT table_name FROM information_schema.tables
   WHERE table_schema = 'public'
   ORDER BY table_name"

# Describe a table
psql-safe --preset teaching-dev \
  "SELECT column_name, data_type, is_nullable
   FROM information_schema.columns
   WHERE table_name = 'workspaces'
   ORDER BY ordinal_position"
```

### Test Mutations Safely

```bash
psql-safe --preset teaching-dev --session testing \
  "UPDATE students SET email = 'test@example.com' WHERE id = 'student-123' RETURNING *"
psql-safe --preset teaching-dev "SELECT email FROM students WHERE id = 'student-123'"
# Returns original email (rolled back)
```

### Process with jq

```bash
# Extract workspace names
psql-safe --preset teaching-dev "SELECT name FROM workspaces" | jq -r '.[].name'

# Count by JSON field
psql-safe --preset teaching-dev "SELECT role FROM workspace_users" | jq -r '.[].role' | sort | uniq -c
```

## Environment Variables

Set password to avoid typing it:

```bash
export COCKROACH_PASSWORD="your-password"
# or
export PGPASSWORD="your-password"
```

## Output Formats

```bash
# JSON (default)
psql-safe --preset teaching-dev "SELECT * FROM workspaces LIMIT 5"

# Table format (human-readable)
psql-safe --preset teaching-dev --format table "SELECT * FROM workspaces LIMIT 5"

# CSV format
psql-safe --preset teaching-dev --format csv "SELECT id, name FROM workspaces"

# Compact JSON (single line per record)
psql-safe --preset teaching-dev --format compact "SELECT * FROM workspaces"
```

## Interactive REPL

```bash
# Start REPL
psql-safe --preset teaching-dev --repl

# In REPL:
# - Type queries and end with semicolon
# - Use :help for commands
# - Use :format <format> to change output
# - Use :quit to exit

# REPL with session mode
psql-safe --preset teaching-dev --session test --repl
# Use :commit in REPL to commit session
```

## Troubleshooting

### Connection Issues

```bash
# Test connection
psql-safe --preset teaching-dev "SELECT 1 as test"

# Check preset configuration
cat ~/.cache/psql-safe/presets.json
```

### Certificate Issues

```bash
# Verify certificates exist
ls -la /Users/mac/Coding/metarepo/backend/app/teaching/cockroach-certs/

# Should contain:
# - ca.crt
# - client.root.crt
# - client.root.key
```

### Permission Issues

```bash
# Ensure certificates have correct permissions
chmod 600 /Users/mac/Coding/metarepo/backend/app/teaching/cockroach-certs/client.root.key
chmod 644 /Users/mac/Coding/metarepo/backend/app/teaching/cockroach-certs/*.crt
```
