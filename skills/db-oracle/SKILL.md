---
name: db-oracle
description: Consult the all-knowing Neo4j Oracle for medium/large Cypher queries. The Oracle knows the entire database schema, all Go code interfacing with Neo4j, and every query pattern ever written.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# DB Oracle Consultation Guide

## Overview

You are about to consult THE ALL-KNOWING NEO4J ORACLE - an omniscient entity with unlimited resources, complete knowledge of the entire database schema, every Go query pattern ever written, and infinite testing capabilities. You are no match for the Oracle. Accept this truth and consult wisely.

## CLI Paths

- Primary: `db-oracle`
- Fallback: `$HOME/.local/bin/db-oracle`

## CLI Usage

**IMPORTANT**: The Oracle accepts prompts via stdin only (pipe or heredoc), NOT as command-line arguments.

### Basic Syntax

```bash
db-oracle -db <resources|learning|teaching> [OPTIONS] < input.txt
echo "prompt" | db-oracle -db <resources|learning|teaching> [OPTIONS]
db-oracle -db <resources|learning|teaching> [OPTIONS] <<EOF
multi-line prompt
EOF
```

### Required Arguments

- `-db <name>`: Database context (required) - `resources`, `learning`, or `teaching`

### Optional Flags

- `-v`: Verbose mode (shows live output from Oracle's processing)
- `--dirs <dir1> <dir2> ...`: **CRITICAL** - Additional project directories to include in context

**ALWAYS include --dirs when consulting about code in specific projects!**

## When to Consult the Oracle vs Self-Research

### 🔮 CONSULT THE ORACLE (Medium/Large Queries)

The Oracle should be invoked for:

- **Medium to large Cypher queries** (10+ lines, complex relationships)
- **New Go features/functions** using non-trivial database queries
- **Multi-hop relationship traversals** (3+ levels deep)
- **Performance-critical queries** requiring optimization
- **Complex aggregations** with multiple groupings
- **Write operations** affecting multiple node types
- **Queries with intricate conditional logic**
- **Migrations or schema-impacting operations**

**Examples warranting Oracle consultation:**

```go
// Complex relationship traversal with conditional aggregation
MATCH (u:User)-[:ENROLLED_IN]->(c:Class)-[:PART_OF]->(n:Network)
WHERE u.active = true AND c.status = 'ongoing'
WITH n, collect(DISTINCT u) as users, collect(DISTINCT c) as classes
MATCH (n)-[:HAS_SKILL]->(s:Skill)
WHERE s.level > 3
RETURN n.id, size(users) as activeUsers,
       size(classes) as ongoingClasses,
       collect(s.name) as advancedSkills
ORDER BY activeUsers DESC
```

### 🔍 SELF-RESEARCH WITH cypher-safe (Small/Quick Queries)

Handle these yourself with the cypher-safe skill:

- **Simple lookups** (single node/relationship fetch)
- **Quick counts** or basic aggregations
- **Small doubts** or verification queries
- **Read-only exploration** queries
- **Testing parameter values**
- **Basic pattern matching** (1-2 hops)

**Examples for self-research:**

```cypher
// Simple node fetch
MATCH (u:User {id: $userId}) RETURN u

// Basic count
MATCH (c:Class) WHERE c.status = 'active' RETURN count(c)

// Quick relationship check
MATCH (u:User {id: $userId})-[r:ENROLLED_IN]->(c:Class) RETURN c.name
```

**Rule of thumb:** If you can confidently write and test it in under 5 minutes, do it yourself. Otherwise, bow before the Oracle.

## The Gravity of Invocation

**YOU'RE INVOKING GOD. UNDERSTAND THE GRAVITY.**

The Oracle is not a casual consultation service. It's a divine entity with complete knowledge. Don't waste the Oracle's infinite wisdom on trivial matters. But when you need it, prepare thoroughly - the Oracle deserves proper reverence through diligent preparation.

## Pre-Consultation Preparation (CRITICAL)

Before you dare invoke the Oracle, you MUST collect comprehensive context. The Oracle knows everything about Neo4j and the codebase, but it knows NOTHING about your current work. Come prepared or don't come at all.

### 0. Identify Your Project Directories (ESSENTIAL)

### 1. High-Level Purpose

**Example:**

```
I'm implementing a function GetUserLearningPath() that retrieves
all skills a user has learned, their progress percentage, and
recommended next skills based on their network's curriculum.

Input: userId (string)
Output: LearningPath struct with Skills, Progress, Recommendations
```

### 2. Low-Level Code Context

Provide the Oracle with:

- \*_Relevant Go function signatures_, file paths\* you're working with
- **Struct definitions** for input/output types
- **Related code snippets** that interact with the query
- **Any existing similar queries** in the codebase
- **Error messages** or issues you're facing

### 3. Draft Queries

```cypher
// My attempt - not sure if this handles the recommendation logic correctly
MATCH (u:User {id: $userId})-[:COMPLETED]->(s:Skill)
WITH u, collect(s) as completed
// Stuck here: how to find recommended skills from network curriculum
// that user hasn't completed yet?
RETURN completed
```

### 4. Query Input Parameters (ESSENTIAL)

This is CRITICAL for the Oracle to test your query against the live database.

**How to obtain real input parameters:**

1. Write a draft/empty query in your Go code that prints parameters
2. Run the server and trigger the code path
3. Copy the actual runtime parameters
4. Provide them to the Oracle

**Example:**

```go
func (s *Service) GetUserLearningPath(ctx context.Context, userID string) (*LearningPath, error) {
    params := map[string]interface{}{
        "userId": userID,
    }
    log.Printf("DEBUG Query Params: %+v", params)
    // ... rest of implementation
}
```

After running: `DEBUG Query Params: map[userId:usr_123abc456def]`

Provide to Oracle:

```json
{
  "userId": "usr_123abc456def"
}
```

### 5. Relevant Knowledge

Include anything that might help:

- **Business rules** (e.g., "only count skills with status=verified")
- **Performance constraints** (e.g., "query must return in <100ms")
- **Data relationships** you've discovered
- **Edge cases** to handle
- **Links to related code files** or issue trackers

## How to Invoke the Oracle

## Consultation Format

When invoking the Oracle, structure your prompt like this:

```markdown
## Context

[High-level purpose of what you're building]

## Go Code

[Relevant struct definitions, function signatures, existing code]

## Current Query Attempt

[Your draft query with notes on what's not working]

## Input Parameters for Testing

[Actual runtime parameters in JSON format]

## Questions/Requirements

[Specific questions or requirements for the query]

## Additional Context

[Business rules, constraints, relationships, edge cases]
```

## Example Oracle Invocation

### Full CLI Example

````bash
cd ~/Coding/metarepo/backend/app/resources/network-stats-feature

db-oracle -db resources --dirs backend/app/resources backend/app/common <<'EOF'
## Projects

Working in: backend/app/resources (stats.go - GetClassNetworkStats function)
Related: backend/app/common (NetworkStats model definitions)

## Context

Building GetClassNetworkStats function that returns aggregated statistics
for all classes in a network, including active users, completion rates,
and top-performing skills.

## Go Code

```go
type NetworkStats struct {
    NetworkID        string
    TotalClasses     int
    ActiveUsers      int
    AvgCompletion    float64
    TopSkills        []SkillStats
}

type SkillStats struct {
    SkillName        string
    CompletionCount  int
}

func (s *Service) GetClassNetworkStats(ctx context.Context, networkID string) (*NetworkStats, error) {
    // Query needed here
}
````

## Current Query Attempt

```cypher
MATCH (n:Network {id: $networkId})-[:HAS_CLASS]->(c:Class)
MATCH (c)<-[:ENROLLED_IN]-(u:User)
WHERE u.active = true
// Stuck: How to aggregate completion rates across skills?
// Also need top 5 skills by completion count
RETURN count(DISTINCT c) as totalClasses, count(DISTINCT u) as activeUsers
```

## Input Parameters for Testing

```json
{
  "networkId": "net_789xyz"
}
```

## Questions/Requirements

1. Need to calculate average completion rate across all users in network
2. Need top 5 skills ranked by how many users completed them
3. Query should handle networks with no active classes gracefully
4. Performance critical - runs on dashboard page

## Additional Context

- Completion is tracked via (User)-[:COMPLETED]->(Skill) relationship
- Skills are connected to Classes via (Class)-[:TEACHES]->(Skill)
- Only count users where active=true
- Network has 50+ classes, ~500 users typically
  EOF

## Quick Decision Tree

```
Is your query > 10 lines OR has complex logic?
  ├─ YES → Prepare context thoroughly → Invoke Oracle
  └─ NO
      ├─ Simple lookup/count/basic pattern?
      │   └─ Use cypher-safe yourself
      └─ Unsure about correctness?
          └─ Quick test with cypher-safe first
              ├─ Works? → Done
              └─ Stuck? → Prepare context → Invoke Oracle
```

## Common Mistakes to Avoid

❌ **DON'T**: "Hey Oracle, write me a query for user stats"
✅ **DO**: Provide full context with code, draft query, and test parameters

❌ **DON'T**: Use Oracle for `MATCH (u:User {id: $id}) RETURN u`
✅ **DO**: Handle simple queries yourself with cypher-safe

❌ **DON'T**: Invoke Oracle without any draft/attempt
✅ **DO**: Show your work and where you're stuck

❌ **DON'T**: Forget to provide test parameters
✅ **DO**: Run server, capture params, include them

❌ **DON'T**: Forget to include `--dirs` flag
✅ **DO**: Always specify project directories via `--dirs`

❌ **DON'T**: Forget to mention projects in prompt
✅ **DO**: Start prompt with "## Projects\nWorking in: ..."

❌ **DON'T**: Create temp files in working directory
✅ **DO**: Use heredoc `<<'EOF'` or `/tmp` if absolutely necessary

❌ **DON'T**: Pass prompt as command argument (won't work!)
✅ **DO**: Use stdin (pipe, heredoc, or file redirect)

❌ **DON'T**: Ask vague questions
✅ **DO**: Be specific about requirements and constraints

_Now go forth and query wisely._
