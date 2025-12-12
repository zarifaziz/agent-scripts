---
name: cypher-safe
description: Execute Cypher queries against Neo4j databases. Use when running read queries, testing write operations, or working with Neo4j databases.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Cypher Safe Query Tool

## Overview

Execute Cypher queries against Neo4j databases with two modes:

1. **Default mode**: Read-only queries
2. **Session mode**: Safe write testing (auto-rollback)

## CLI Paths

- Primary: `cypher-safe`
- Fallback: `/Users/mac/.local/bin/cypher-safe`

---

## BEFORE ANY QUERY (Required)

**DO NOT skip to examples below - follow these steps first or waste time trial-erroring.**

Skipping protocol = 3-5 wasted queries guessing relationships. Following it = 1 precise query.

### Step 1: Export schema
```bash
cd ~/Coding/metarepo/backend/app/resources/<your-worktree>
mkdir -p dist/ && neo4j-usage-export resources --export dist/
```

### Step 2: Search codebase for existing patterns
```bash
# Find how entities relate in existing code
Grep "WorksheetPlan" internal/
Grep "LessonSection" internal/
```

### Step 3: THEN write your informed query

### Anti-pattern (wastes time)
```
❌ Jump straight to querying → guess relationships → trial-error 3+ times
   Example: "Does WorksheetPlan have HAS_SECTION? No... CONTAINS? No... let me try through Lesson..."
```

### Correct approach
```
✅ Check schema export → find existing code pattern → write 1 informed query
   Example: See code uses (wp)-[:PLANS]->(l)-[:CONTAINS]->(ls) → query works first try
```

---

## Examples (only after completing protocol above)

### 1. Running Against Local Database (Default)

Query local Neo4j without any connection flags:

```bash
cypher-safe "MATCH (n) RETURN n LIMIT 5"

cypher-safe "MATCH (n) RETURN count(n) as total"

cypher-safe "MATCH (p:Person) RETURN p.name, p.age ORDER BY p.age DESC"
```

With parameters:

```bash
cypher-safe --params '{"name":"Alice","age":30}' \
  "MATCH (p:Person {name: \$name}) RETURN p"
```

### 2. Running Against Preset (resources-dev)

Use `--preset resources-dev` to connect to remote database:

```bash
cypher-safe --preset resources-dev "MATCH (n) RETURN count(n) as total"

cypher-safe --preset resources-dev \
  "MATCH (t:Topic) RETURN t.name as name LIMIT 10"

cypher-safe --preset resources-dev \
  "MATCH (r:Resource) RETURN r.title, r.created ORDER BY r.created DESC LIMIT 20"
```

With parameters:

```bash
cypher-safe --preset resources-dev \
  --params '{"topicId":"123"}' \
  "MATCH (t:Topic {id: \$topicId}) RETURN t"
```

### 3. Using Session Mode

Session mode allows safe write testing with automatic rollback.

#### Local Database Sessions

```bash
cypher-safe --session test-users \
  "CREATE (p:Person {name: 'Alice', age: 30}) RETURN p"

cypher-safe --session test-users \
  "CREATE (p:Person {name: 'Bob', age: 25}) RETURN p"

cypher-safe --session test-users \
  "MATCH (p:Person) RETURN p.name, p.age ORDER BY p.age"

cypher-safe --session test-users \
  "MATCH (a:Person {name: 'Alice'}), (b:Person {name: 'Bob'})
   CREATE (a)-[r:KNOWS]->(b) RETURN type(r)"

cypher-safe --drop test-users
```

#### Preset Database Sessions

```bash
cypher-safe --preset resources-dev --session feature-test \
  "CREATE (t:Topic {name: 'Test Topic', id: '999'}) RETURN t"

cypher-safe --preset resources-dev --session feature-test \
  "MATCH (t:Topic {id: '999'}) SET t.verified = true RETURN t"

cypher-safe --preset resources-dev --session feature-test \
  "MATCH (t:Topic) WHERE t.verified = true RETURN t.name"

cypher-safe --drop feature-test
```

## Session Management

List active sessions:

```bash
cypher-safe --list
```

Show session queries:

```bash
cypher-safe --show test-users
```

Drop session:

```bash
cypher-safe --drop test-users
```

## Key Points

1. **Flags before query**: All flags must come BEFORE the query string
2. **Default rollback**: Session mode automatically rolls back all changes
3. **Session replay**: Each session query replays previous queries in that session
4. **Read-only by default**: Without `--session`, all queries are read-only
5. **Presets available**: `resources-dev`, `learning-dev`, `learning-admin-dev`
6. **JSON output**: Results are JSON arrays, pipe to `jq` for processing

## Common Patterns

Explore data:

```bash
cypher-safe "MATCH (n) RETURN labels(n) as label, count(*) as count"
```

Test mutations safely:

```bash
cypher-safe --session testing "CREATE (n:Test) RETURN n"
cypher-safe "MATCH (n:Test) RETURN count(n)"  # Returns 0 (rolled back)
```

Verify before production:

```bash
cypher-safe --preset resources-dev --session verify \
  "MATCH (t:Topic) SET t.status = 'active' RETURN count(t)"
```

Process with jq:

```bash
cypher-safe "MATCH (t:Topic) RETURN t.name" | jq -r '.[].t.name'
```
