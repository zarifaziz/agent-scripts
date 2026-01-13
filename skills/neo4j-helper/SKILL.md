---
name: neo4j-helper
description: Get Cypher query help using pre-exported Neo4j context (schema, usage patterns, entities, edges) and test queries with cypher-safe. Lighter alternative to db-oracle.
allowed-tools:
  - Bash
  - Read
  - glob
  - Grep
metadata:
  version: "1.0"
---

# Neo4j Helper

## Overview

Provides Cypher query assistance by:
1. Reading pre-exported Neo4j schema + Go usage patterns
2. Understanding the data model from exports
3. Testing queries with cypher-safe

## Exports Location

```
~/workspace/agent-scripts/exports/
  resources/   # Resources repo exports
  learning/    # Learning repo exports
```

Each folder contains:
- `neo4j_database_schema_*.txt` - Live schema dump (node/relationship types, properties)
- `all_edges.txt` - Edge relationship definitions (if present)
- `all_entities.txt` - Node/entity definitions (if present)
- `*_impl.txt` - Neo4j usage patterns from Go code
- `*_api.txt` - API layer code with GraphQL resolvers

## Workflow

### 1. Check Exports Freshness

```bash
# Check when exports were last generated
ls -la ~/workspace/agent-scripts/exports/<resources|learning>/
```

If exports are missing or stale (>1 week old), regenerate:

```bash
# For resources
cd ~/workspace/metarepo/backend/app/resources
go run ~/workspace/agent-scripts/neo4j-usage-export.go --export ~/workspace/agent-scripts/exports/resources resources

# For learning
cd ~/workspace/metarepo/backend/app/learning
go run ~/workspace/agent-scripts/neo4j-usage-export.go --export ~/workspace/agent-scripts/exports/learning learning
```

### 2. Read Context

Load relevant export files based on the query domain:

```bash
# Schema (always start here)
cat ~/workspace/agent-scripts/exports/<db>/neo4j_database_schema_<db>.txt

# Find relevant API patterns
ls ~/workspace/agent-scripts/exports/<db>/
cat ~/workspace/agent-scripts/exports/<db>/<relevant>_api.txt
```

### 3. Draft Query

Based on user request + context from exports, draft the Cypher query.

### 4. Test with cypher-safe

```bash
# Read query (default)
cypher-safe --preset <resources-dev|learning-dev> "MATCH (n:Label) RETURN n LIMIT 5"

# With parameters
cypher-safe --preset resources-dev --params '{"id":"abc123"}' \
  "MATCH (n:Node {id: \$id}) RETURN n"

# Write query (session mode - auto-rollback)
cypher-safe --preset resources-dev --session test-session \
  "CREATE (n:Test {name: 'test'}) RETURN n"

# Drop session when done
cypher-safe --drop test-session
```

## Quick Reference

### Available Presets

- `resources-dev` - Resources database
- `learning-dev` - Learning database
- `learning-admin-dev` - Learning admin database

### Common Patterns

```bash
# Explore schema
cypher-safe --preset resources-dev "MATCH (n) RETURN labels(n) as label, count(*) as count"

# Find relationships
cypher-safe --preset resources-dev "MATCH ()-[r]->() RETURN type(r), count(*) ORDER BY count(*) DESC"

# Process with jq
cypher-safe --preset resources-dev "MATCH (t:Topic) RETURN t.name" | jq -r '.[].t.name'
```

## When to Use This Skill

- Writing new Cypher queries
- Debugging existing queries
- Understanding the Neo4j data model
- Exploring relationships between entities
- Optimizing query performance

## Key Points

1. **Schema first**: Always check the schema export before writing queries
2. **Pattern matching**: Search `*_api.txt` files for existing query patterns
3. **Test safely**: Use session mode for write operations (auto-rollback)
4. **Preset required**: Remote queries need `--preset` flag
