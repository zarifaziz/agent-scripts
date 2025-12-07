---
name: reverse-graphql-input
description: Reverse engineer GraphQL input from a lesson plan or worksheet plan ID/link. Extracts plan data and reconstructs the original input JSON used to create it.
allowed-tools:
  - bash
  - read
  - grep
  - glob
metadata:
  version: "1.0"
---

# Reverse GraphQL Input from Lesson Plan or Worksheet Plan

## Overview

Given a lesson plan or worksheet plan ID or URL, this skill reverse engineers the original GraphQL input that was used to create the plan. It:

1. Extracts the plan ID from URL or uses provided ID
2. Determines the correct database environment (prod, dev, staging)
3. Detects the plan type (Lesson Plan vs Worksheet Plan)
4. Verifies the node exists and is a LessonPlan
5. Executes the appropriate query to reconstruct the input JSON

## Usage

The user will provide either:
- **Lesson Plan URL**: `https://schools.tutero.com/lesson-plan/lp_01K76BE7AM487533R1GEK8M343?id&tab=slides`
- **Worksheet Plan URL**: `https://schools.tutero.dev/worksheets/lp_01K8079CTG53MZYH6M622HNWP3`
- **Direct ID**: `lp_01K76BE7AM487533R1GEK8M343`

## Implementation Steps

### Step 1: Extract Plan ID and Detect Plan Type

Extract the ID from the URL or use the provided ID directly:

```bash
# If URL provided, extract ID using regex
# Lesson Plan Pattern: /lesson-plan/([a-zA-Z0-9_]+)
# Worksheet Plan Pattern: /worksheets/([a-zA-Z0-9_]+)

# Detect plan type:
if [[ "$url" =~ /worksheets/ ]]; then
    plan_type="worksheet"
elif [[ "$url" =~ /lesson-plan/ ]]; then
    plan_type="lesson"
else
    # If direct ID, ask user or check both
    plan_type="unknown"
fi
```

### Step 2: Determine Database Environment

Map domain to preset:
- `.com` → prod preset (check if `resources-prod` exists, otherwise `resources`)
- `.dev` → dev preset (`resources-dev`)
- `.staging-dev` → staging preset (`resources-staging`)

### Step 3: Verify Node Exists

Run existence check using cypher-safe:

```bash
cypher-safe --preset <preset-name> "MATCH (n {id: '<lesson-plan-id>'}) RETURN labels(n) as labels"
```

Verify the node is a LessonPlan. If not found or wrong type, report error.

### Step 4: Execute Reverse Engineering Query

Choose the appropriate query based on plan type:

#### 4A. For Lesson Plans (Standard)

Run the following query using cypher-safe to reconstruct the input:

```cypher
WITH "<lesson-plan-id>" AS id
MATCH (lp:LessonPlan {id: id})

            // Get the class ID from IN relationship
            CALL {
                WITH lp
                OPTIONAL MATCH (lp)<-[:CREATED]-(u:User)
                RETURN u.id AS classId
            }

            // Get the subtopic ID from FOR relationship
            CALL {
                WITH lp
                OPTIONAL MATCH (lp)-[:FOR]->(st:Subtopic)
                RETURN st.id AS subtopicID
            }

            // Get the context from SELECTED relationship or direct property
            CALL {
                WITH lp
                OPTIONAL MATCH (lp)-[:SELECTED]->(lpc:LessonPlanContext)
                RETURN coalesce(lpc.context, lp.context) AS contextValue
            }

            // Get lesson plan name
            WITH lp, classId, subtopicID, contextValue, lp.name AS lessonPlanName

            // Get all lessons with their durations, ordered by index
            CALL {
                WITH lp
                OPTIONAL MATCH (lp)-[plansRel:PLANS]->(l:Lesson)
                WITH l, plansRel
                ORDER BY plansRel.index ASC
                RETURN collect(l.duration) AS lessonDurations
            }

            // Extract all unique skill IDs from enabled SkillSections
            CALL {
                WITH lp
                OPTIONAL MATCH (lp)-[:PLANS]->(lesson:Lesson)-[:CONTAINS]->(ss:SkillSection)-[:HAS]->(sk:Skill)
                WHERE ss.enabled = true OR ss.enabled IS NULL
                WITH sk
                ORDER BY sk.id
                RETURN collect(DISTINCT sk.id) AS skillIDs
            }

            // Get enabled node types from SELECTED or lesson sections
            CALL {
                WITH lp

                MATCH (lp)-[:PLANS]->(l:Lesson)-[:CONTAINS]->(section:LessonSection)-[:INCLUDES]->(lpn:LessonPlanNode)
                WHERE lpn.enabled = true AND lpn.type <> ""
                RETURN collect(DISTINCT lpn.type) AS nodeTypes

            }

            // Build the final reconstructed input structure
            WITH {
                classId: classId,
                input: {
                    lessonDurations: lessonDurations,
                    input: {
                        includeNodes: CASE WHEN size(nodeTypes) > 0 THEN nodeTypes ELSE [] END,
                        skillIDs: skillIDs,
                        subtopicID: subtopicID,
                        context: contextValue,
                        name: coalesce(lessonPlanName, "")
                    }
                }
            } AS reconstructedInput

            RETURN reconstructedInput
```

#### 4B. For Worksheet Plans

Run the following query using cypher-safe to reconstruct the worksheet plan input.
This query properly extracts nodeCounts in the correct input format:
- Only `SCAFFOLDED_QUESTION` uses `nodeType`
- All others collapse to `questionType`: MULTI, SHORT, OPEN_ENDED

(based on `getWorksheetState` in `internal/lesson_plan/update_worksheet_plan_v2.go`):

```cypher
MATCH (lp:WorksheetPlan {id: "<worksheet-plan-id>"})
OPTIONAL MATCH (lp)-[:FOR]->(st:Subtopic)
OPTIONAL MATCH (lp)<-[:FOR]-(wc:WorksheetPlanConfiguration)
OPTIONAL MATCH (u:User)-[:CREATED]->(lp)

// Get skillIDs from SkillSections
CALL {
    WITH lp
    OPTIONAL MATCH (lp)-[:PLANS]->(:Lesson)-[:CONTAINS]->(ss:SkillSection)-[:HAS]->(skill:Skill)
    WHERE ss.enabled = true OR ss.enabled IS NULL
    RETURN collect(DISTINCT skill.id) AS skillIDs
}

// Get nodeCounts in input format (SCAFFOLDED uses nodeType, others use questionType)
// Also calculate total questionCount from actual nodes
CALL {
    WITH lp
    OPTIONAL MATCH (lp)-[:PLANS]->(l:Lesson)-[:CONTAINS]->(ls:LessonSection)-[includesRel:INCLUDES]->(lpn:LessonPlanNode)
    WHERE (lpn.enabled = true OR lpn.enabled IS NULL) AND (ls.enabled = true OR ls.enabled IS NULL)
    
    OPTIONAL MATCH (lpn:StudentQuestionsNode)-[qi:INCLUDES]->(q:Question)
    
    // Map LPN types to question types
    WITH lpn, CASE
        WHEN lpn.type = "SCAFFOLDED_QUESTION" THEN "SCAFFOLDED"
        WHEN lpn.type = "STUDENT_QUESTIONS" THEN qi.type
        WHEN lpn.type = "MULTIPLE_CHOICE_QUESTION" THEN "MULTIPLE_CHOICE"
        WHEN lpn.type = "SHORT_ANSWER_QUESTION" THEN "SHORT_ANSWER"
        WHEN lpn.type = "CONTEMPLATIVE_QUESTION" THEN "OPEN_ENDED"
        ELSE null
    END AS qType,
    lpn.type AS nodeType
    WHERE lpn IS NOT NULL AND qType IS NOT NULL
    
    WITH nodeType, qType, count(*) AS cnt
    
    // Format: SCAFFOLDED gets nodeType, others get questionType (MULTI, SHORT, OPEN_ENDED)
    WITH CASE
        WHEN nodeType = "SCAFFOLDED_QUESTION" THEN {nodeType: "SCAFFOLDED_QUESTION", count: cnt}
        WHEN qType = "MULTIPLE_CHOICE" THEN {questionType: "MULTI", count: cnt}
        WHEN qType = "SHORT_ANSWER" THEN {questionType: "SHORT", count: cnt}
        WHEN qType = "OPEN_ENDED" THEN {questionType: "OPEN_ENDED", count: cnt}
        ELSE null
    END AS nodeCount
    WHERE nodeCount IS NOT NULL
    
    // Aggregate same types together
    WITH nodeCount.nodeType AS nt, nodeCount.questionType AS qt, sum(nodeCount.count) AS total
    WITH collect(
        CASE 
            WHEN nt IS NOT NULL THEN {nodeType: nt, count: total}
            ELSE {questionType: qt, count: total}
        END
    ) AS nodeCounts
    
    // Sum all counts for questionCount
    WITH nodeCounts, reduce(sum = 0, nc IN nodeCounts | sum + nc.count) AS questionCount
    RETURN nodeCounts, questionCount
}

// Get enabled node types
CALL {
    WITH lp
    OPTIONAL MATCH (lp)-[:PLANS]->(:Lesson)-[:CONTAINS]->(:LessonSection)-[:INCLUDES]->(lpn:LessonPlanNode)
    WHERE lpn.enabled = true OR lpn.enabled IS NULL
    RETURN collect(DISTINCT lpn.type) AS includeNodes
}

RETURN {
    input: {
        name: lp.name,
        subtopicID: st.id,
        context: coalesce(lp.context, ""),
        skillIDs: skillIDs,
        includeNodes: [n IN includeNodes WHERE n IS NOT NULL AND n <> ""],
        configuration: {
            type: coalesce(wc.type, "STANDARD"),
            questionCount: questionCount,
            averageQuestionDifficulty: coalesce(wc.averageQuestionDifficulty, -1),
            minDifficulty: coalesce(wc.minDifficulty, 0),
            maxDifficulty: coalesce(wc.maxDifficulty, 4),
            workingOutSpaceEnabled: coalesce(wc.workingOutSpaceEnabled, true),
            reducePaperEnabled: coalesce(wc.reducePaperEnabled, true),
            nodeCounts: nodeCounts
        }
    },
    classId: u.id
} AS creationParams
```

**nodeCounts format** (input format, not internal):
- `SCAFFOLDED_QUESTION` → `{nodeType: "SCAFFOLDED_QUESTION", count: X}`
- `MULTIPLE_CHOICE_QUESTION` / `STUDENT_QUESTIONS(MULTIPLE_CHOICE)` → `{questionType: "MULTI", count: X}`
- `SHORT_ANSWER_QUESTION` / `STUDENT_QUESTIONS(SHORT_ANSWER)` → `{questionType: "SHORT", count: X}`
- `CONTEMPLATIVE_QUESTION` / `STUDENT_QUESTIONS(OPEN_ENDED)` → `{questionType: "OPEN_ENDED", count: X}`
- `LEARNING_GOALS` → not included in nodeCounts

### Step 5: Format Output

Present the reconstructed input JSON in a clear, formatted manner using `jq` or similar:

```bash
echo '<result>' | jq '.'
```

## Environment Detection Logic

```bash
# Extract domain from URL
if [[ "$url" =~ schools\.tutero\.com ]]; then
    preset="resources-prod"  # or just "resources"
elif [[ "$url" =~ schools\.tutero\.dev ]]; then
    preset="resources-dev"
elif [[ "$url" =~ schools\.tutero\.staging-dev ]]; then
    preset="resources-staging"
else
    # Default to dev or ask user
    preset="resources-dev"
fi
```

## Error Handling

1. **ID not found**: Report that lesson plan doesn't exist in the database
2. **Wrong node type**: Report that the ID exists but is not a LessonPlan
3. **Connection error**: Report database connection issues
4. **Invalid URL format**: Report parsing error

## Example Flows

### Example 1: Lesson Plan

```bash
# Input: https://schools.tutero.com/lesson-plan/lp_01K76BE7AM487533R1GEK8M343?id&tab=slides

# 1. Extract ID: lp_01K76BE7AM487533R1GEK8M343
# 2. Detect plan type: lesson-plan → lesson
# 3. Detect environment: .com → prod
# 4. Verify: cypher-safe --preset resources-prod "MATCH (n {id: 'lp_01K76BE7AM487533R1GEK8M343'}) RETURN labels(n)"
# 5. Execute lesson plan query (4A)
# 6. Format and present result
```

### Example 2: Worksheet Plan

```bash
# Input: https://schools.tutero.dev/worksheets/lp_01K8079CTG53MZYH6M622HNWP3

# 1. Extract ID: lp_01K8079CTG53MZYH6M622HNWP3
# 2. Detect plan type: worksheets → worksheet
# 3. Detect environment: .dev → dev
# 4. Verify: cypher-safe --preset resources-dev "MATCH (n {id: 'lp_01K8079CTG53MZYH6M622HNWP3'}) RETURN labels(n)"
# 5. Execute worksheet plan query (4B)
# 6. Format and present result
```

#### 4C. For Lesson Plans V3 (New Structure)

For the new `createLessonPlanV3` with nested lesson structures, skill sections, and practice nodes:

```bash
cypher-safe --preset resources-dev --params '{"lpID":"lp_xxx"}' "$(cat ~/.opencode/skills/reverse-graphql-input/reverse_v3.cypher)"
```

See `reverse_v3.cypher` in this directory for the full query. Handles introductionNodes, skillSections (with templateConfig), and practiceNodes (with cardConfig).

## Notes

- Always use cypher-safe skill for database queries
- Handle cases where preset names might vary (try with/without suffixes)
- Validate lesson plan ID format before querying
- Present results in readable JSON format
- Report progress to user at each step
- Worksheet plans use the same LessonPlan node type but have different structure
- URL pattern determines whether to use lesson or worksheet query
