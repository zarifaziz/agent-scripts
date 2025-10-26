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

THE ALL-KNOWING NEO4J ORACLE - an omniscient entity with complete knowledge of the database schema, every Go query pattern ever written, and infinite testing capabilities against live databases.

## CLI Paths

- Primary: `db-oracle`
- Fallback: `$HOME/.local/bin/db-oracle`

## Usage

```bash
db-oracle -db <resources|learning|teaching> [--dirs <dir1> <dir2>...] [-v] <<'EOF'
[your prompt]
EOF
```

**Required:** `-db` flag (resources, learning, or teaching)
**Important:** Always use `--dirs` to specify project directories for context
**Prompts:** Must come via stdin (heredoc, pipe, or redirect)

## When to Consult the Oracle

**🔮 CONSULT ORACLE FOR:**
- Medium/large queries (10+ lines, complex relationships)
- Multi-hop traversals (3+ levels deep)
- Performance-critical queries requiring optimization
- Complex aggregations or write operations
- Queries with intricate conditional logic

**🔍 HANDLE YOURSELF WITH cypher-safe:**
- Simple lookups (single node/relationship fetch)
- Quick counts or basic aggregations
- Testing parameter values
- Basic pattern matching (1-2 hops)

**Rule of thumb:** If you can confidently write and test it in under 5 minutes, do it yourself. Otherwise, consult the Oracle.

## Pre-Consultation Preparation

Before invoking the Oracle, collect:

1. **Project directories** - Use `--dirs` flag
2. **High-level purpose** - What you're building, input/output
3. **Go code context** - Structs, function signatures, file paths
4. **Draft query** - Your attempt with notes on what's stuck
5. **Test parameters** - CRITICAL - actual runtime parameters from your code
6. **Business rules** - Constraints, edge cases, performance requirements

**How to get test parameters:**
```go
func (s *Service) GetUserLearningPath(ctx context.Context, userID string) (*LearningPath, error) {
    params := map[string]interface{}{"userId": userID}
    log.Printf("DEBUG Query Params: %+v", params)  // Capture these!
    // ...
}
```

## Structured Prompt Format

```markdown
## Context
[High-level purpose of what you're building]

## Go Code
[Relevant struct definitions, function signatures]

## Current Query Attempt
[Your draft query with notes on what's not working]

## Input Parameters for Testing
[Actual runtime parameters in JSON - CRITICAL for Oracle to test]

## Questions/Requirements
[Specific questions or requirements]

## Additional Context
[Business rules, constraints, edge cases]
```

## Example: Detailed Consultation

````bash
cd ~/Coding/metarepo/backend/app/resources/network-stats-feature

db-oracle -db resources --dirs backend/app/resources backend/app/common <<'EOF'
## Context

Building GetClassNetworkStats function for dashboard page - returns aggregated 
statistics for all classes in a network (active users, completion rates, top skills).

Input: networkID (string)
Output: NetworkStats struct

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
```

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

1. Calculate average completion rate across all users in network
2. Get top 5 skills ranked by completion count
3. Handle networks with no active classes gracefully
4. Performance critical - runs on dashboard page

## Additional Context

- Completion: (User)-[:COMPLETED]->(Skill)
- Skills to Classes: (Class)-[:TEACHES]->(Skill)
- Only count users where active=true
- Network typically has 50+ classes, ~500 users
EOF
````

## Example: Quick Consultations

```bash
# Debugging query
db-oracle -db learning --dirs backend/app/learning <<'EOF'
## Context
GetStudentProgress returns 0% even when user has completed skills.
Happens intermittently.

## Current Query
```cypher
MATCH (student:Student {id: $studentId})-[:ENROLLED_IN]->(course:Course {id: $courseId})
MATCH (course)-[:REQUIRES]->(skill:Skill)
OPTIONAL MATCH (student)-[:COMPLETED]->(skill)
// Count seems wrong with OPTIONAL MATCH
```

## Test Parameters
{"studentId": "student_abc123", "courseId": "course_math101"}

## Questions
1. Why does count return incorrect values?
2. How to collect completed skill IDs while calculating progress?
3. Get lastActivityAt from most recent COMPLETED relationship?
EOF

# Optimization
db-oracle -db resources --dirs backend/app/resources <<'EOF'
Dashboard query timing out for large networks (500+ resources, takes 5-8s).
Need under 1 second.

Current slow query aggregates resources by type, by subject, gets recently 
added and top rated. Too many WITH clauses and multiple passes.

Test params: {"networkId": "net_large_network_123"}

How to optimize? Split into multiple queries? What indexes?
EOF
```

## Common Mistakes

❌ Vague: "Hey Oracle, write me a query for user stats"
✅ Detailed: Full context with code, draft query, test parameters

❌ Simple query: `MATCH (u:User {id: $id}) RETURN u`
✅ Use cypher-safe for simple lookups

❌ No draft/attempt
✅ Show your work and where you're stuck

❌ Missing test parameters
✅ Capture actual runtime params from code

❌ Forgot `--dirs` flag
✅ Always specify project directories

❌ Prompt as command argument
✅ Use stdin (heredoc, pipe, redirect)

---

_Now go forth and query wisely._
