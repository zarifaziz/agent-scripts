---
name: inspect-resource-question
description: Extract and inspect AI-generated questions from lesson plans or worksheet plans. Formats quillDocument questions into readable plaintext.
allowed-tools:
  - bash
  - read
  - grep
  - glob
metadata:
  version: "1.0"
---

# Inspect Resource Questions

## Overview

Given a lesson plan or worksheet plan ID or URL, this skill:

1. Extracts the plan ID from URL or uses provided ID
2. Determines the correct database environment (prod, dev, staging)
3. Verifies the node exists via existence check
4. Extracts all AI-generated questions from the plan
5. Formats quillDocument questions into readable plaintext using `quill-plain`

## Usage

The user will provide either:

- **Lesson Plan URL**: `https://schools.tutero.com/lesson-plan/lp_01K76BE7AM487533R1GEK8M343?id&tab=slides`
- **Worksheet Plan URL**: `https://schools.tutero.dev/worksheets/wp_01K86J68WJD4EPD1D5RBQJ64GS`
- **Direct ID**: `lp_01K76BE7AM487533R1GEK8M343` or `wp_01K86J68WJD4EPD1D5RBQJ64GS`

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
    # If direct ID, determine by prefix
    if [[ "$id" =~ ^wp_ ]]; then
        plan_type="worksheet"
    elif [[ "$id" =~ ^lp_ ]]; then
        plan_type="lesson"
    fi
fi
```

### Step 2: Determine Database Environment

Map domain to preset:

- `.com` → prod preset (try `resources-prod`, fallback to `resources`)
- `.dev` → dev preset (`resources-dev`)
- `.staging-dev` → staging preset (`resources-staging`)

```bash
# Extract domain from URL
if [[ "$url" =~ schools\.tutero\.com ]]; then
    preset="resources-prod"
elif [[ "$url" =~ schools\.tutero\.dev ]]; then
    preset="resources-dev"
elif [[ "$url" =~ schools\.tutero\.staging-dev ]]; then
    preset="resources-staging"
else
    # Default to dev or ask user
    preset="resources-dev"
fi
```

### Step 3: Verify Node Exists

Run existence check using cypher-safe:

```bash
cypher-safe --preset <preset-name> "MATCH (n {id: '<plan-id>'}) RETURN labels(n) as labels"
```

Verify the node is a LessonPlan. If not found or wrong type, report error.

### Step 4: Extract Questions

Run the following query using cypher-safe to extract all AI-generated questions:

```cypher
MATCH (lp:LessonPlan {id: "<plan-id>"})
      -[:PLANS]->(l:Lesson)
      -[:CONTAINS]->(ls:LessonSection)
      -[:INCLUDES]->(lpn:LessonPlanNode)
WHERE lpn.enabled = true AND lpn.ready = true
OPTIONAL MATCH (lpn)-[pr:PROMPTS]->(q:QuillDocumentNode)

WITH lpn, l, ls,
     [q IN collect({
         questionId: q.id,
         question: q.quillDocument,
         answer: q.answer,
         difficulty: coalesce(q.lessonPlanDifficulty, q.difficulty),
         index: pr.index
     }) WHERE q.question IS NOT NULL] AS questions
WHERE size(questions) > 0

RETURN collect({
    label: labels(lpn),
    type: lpn.type,
    id: lpn.id,
    lessonId: l.id,
    sectionId: ls.id,
    questions: questions
}) AS result
```

### Step 5: Format Questions with quill-plain

Pipe the query results through `jq` and `quill-plain` to format quillDocument questions:

```bash
# Execute query and pipe to quill-plain
cypher-safe --preset <preset-name> "<query>" | ~/.local/bin/quill-plain

# Or with jq processing first
cypher-safe --preset <preset-name> "<query>" | jq '.' | ~/.local/bin/quill-plain
```

The `quill-plain` binary:

- **Location**: `~/.local/bin/quill-plain`
- **Input**: JSON via pipe or heredoc
- **Output**: Formatted plaintext questions
- **Purpose**: Converts quillDocument format to readable plaintext

## Complete Flow Example

```bash
#!/bin/bash

# Example: https://schools.tutero.dev/worksheets/wp_01K86J68WJD4EPD1D5RBQJ64GS

# 1. Extract ID
plan_id="wp_01K86J68WJD4EPD1D5RBQJ64GS"

# 2. Determine environment
preset="resources-dev"

# 3. Verify existence
cypher-safe --preset "$preset" "MATCH (n {id: '$plan_id'}) RETURN labels(n)"

# 4. Extract questions
cypher-safe --preset "$preset" "
MATCH (lp:LessonPlan {id: \"$plan_id\"})
      -[:PLANS]->(l:Lesson)
      -[:CONTAINS]->(ls:LessonSection)
      -[:INCLUDES]->(lpn:LessonPlanNode)
WHERE lpn.enabled = true AND lpn.ready = true
OPTIONAL MATCH (lpn)-[pr:PROMPTS]->(q:QuillDocumentNode)

WITH lpn, l, ls,
     [q IN collect({
         questionId: q.id,
         question: q.quillDocument,
         answer: q.answer,
         difficulty: coalesce(q.lessonPlanDifficulty, q.difficulty),
         index: pr.index
     }) WHERE q.question IS NOT NULL] AS questions
WHERE size(questions) > 0

RETURN collect({
    label: labels(lpn),
    type: lpn.type,
    id: lpn.id,
    lessonId: l.id,
    sectionId: ls.id,
    questions: questions
}) AS result
" > /tmp/questions.json

# 5. Format questions with quill-plain
jq -r '.[0].result[] | @json' /tmp/questions.json | while read -r node; do
    echo "==================================================================="
    echo "Node Type: $(echo "$node" | jq -r '.type // "N/A"')"
    echo "Node ID: $(echo "$node" | jq -r '.id')"
    echo "==================================================================="

    echo "$node" | jq -c '.questions | sort_by(.index) | .[]' | while read -r q; do
        echo "--- Question #$(echo "$q" | jq -r '.index') ---"
        echo "$q" | jq -r '.question' | ~/.local/bin/quill-plain
        echo
    done
done
```

## Output Format

The skill will present:

1. **Plan metadata**: ID, type, environment
2. **Node information**: For each node containing questions:
   - Node type and labels
   - Node ID
   - Lesson ID
   - Section ID
3. **Formatted questions**: Plaintext version of each question with:
   - Question index (sorted order)
   - Question ID
   - Question text (from quillDocument, converted via quill-plain)
   - Answer (if available)
   - Difficulty level (if available)

## Example Output

```
===================================================================
Node Type:
Node Labels: LessonPlanNode, LessonPlanScaffoldedQuestion
Node ID: 01JZ0B8SY43WXRE8K11K9N19PG
Lesson ID: lsn_01JZ0B8SY43WXRE8K12WSJP92W
Section ID: 01JZ0B8SY43WXRE8K13152SDRR
Question Count: 4
===================================================================

--- Question #0 (ID: lpuserq_01JZ0C8573GZ9RRFK1AFJTWBQ7) ---
Difficulty: N/A

Explore a composite shape made by joining a rectangle and a triangle to calculate
its perimeter and area.

--- Question #1 (ID: 01JZ0C8573GZ9RRFK1AHD7GKAJ) ---
Difficulty: N/A

A rectangle has a length of $10$ cm and a width of $6$ cm. What is its perimeter?

--- Question #2 (ID: 01JZ0C8573GZ9RRFK1AJV5DEAP) ---
Difficulty: N/A

A triangle is attached to one side of the rectangle. If the triangle's base is
$10$ cm and its other two sides are $8$ cm each, find the total perimeter.
```

## Error Handling

1. **ID not found**: Report that plan doesn't exist in the database
2. **Wrong node type**: Report that the ID exists but is not a LessonPlan
3. **Connection error**: Report database connection issues
4. **Invalid URL format**: Report parsing error
5. **No questions found**: Report that the plan has no AI-generated questions
6. **quill-plain not found**: Check if binary exists at `~/.local/bin/quill-plain`

## Environment Detection Logic

```bash
# Full detection logic
function detect_environment() {
    local url="$1"

    if [[ "$url" =~ schools\.tutero\.com ]]; then
        echo "resources-prod"
    elif [[ "$url" =~ schools\.tutero\.dev ]]; then
        echo "resources-dev"
    elif [[ "$url" =~ schools\.tutero\.staging-dev ]]; then
        echo "resources-staging"
    else
        echo "resources-dev"  # default
    fi
}

function detect_plan_type() {
    local input="$1"

    if [[ "$input" =~ /worksheets/ ]] || [[ "$input" =~ ^wp_ ]]; then
        echo "worksheet"
    elif [[ "$input" =~ /lesson-plan/ ]] || [[ "$input" =~ ^lp_ ]]; then
        echo "lesson"
    else
        echo "unknown"
    fi
}
```

## Testing the Flow

Before using this skill, test the pipeline:

```bash
# Test 1: Check if quill-plain exists
ls -la ~/.local/bin/quill-plain

# Test 2: Verify cypher-safe connection
cypher-safe --preset resources-dev "MATCH (n:LessonPlan) RETURN n.id LIMIT 1"

# Test 3: Extract questions (sample ID)
cypher-safe --preset resources-dev "
MATCH (lp:LessonPlan {id: 'lp_01JZ0B8RQZHG7SH17H884DHSAW'})
      -[:PLANS]->(l:Lesson)
      -[:CONTAINS]->(ls:LessonSection)
      -[:INCLUDES]->(lpn:LessonPlanNode)
WHERE lpn.enabled = true AND lpn.ready = true
OPTIONAL MATCH (lpn)-[pr:PROMPTS]->(q:QuillDocumentNode)

WITH lpn, l, ls,
     [q IN collect({
         questionId: q.id,
         question: q.quillDocument,
         answer: q.answer,
         difficulty: coalesce(q.lessonPlanDifficulty, q.difficulty),
         index: pr.index
     }) WHERE q.question IS NOT NULL] AS questions
WHERE size(questions) > 0

RETURN collect({
    label: labels(lpn),
    type: lpn.type,
    id: lpn.id,
    lessonId: l.id,
    sectionId: ls.id,
    questions: questions
}) AS result
"

# Test 4: Full pipeline with quill-plain
cypher-safe --preset resources-dev "<query>" | \
  jq -r '.[0].result[0].questions[0].question' | \
  ~/.local/bin/quill-plain
```

## Notes

- Always use cypher-safe skill for database queries
- Query works for both lesson plans and worksheet plans
- quill-plain binary must be available at `~/.local/bin/quill-plain`
- The query filters nodes to only include those with questions (`WHERE size(questions) > 0`)
- Only enabled and ready nodes are included
- QuillDocumentNode contains the formatted question content
- Results are collected into a single `result` array for easier processing
- Questions within each node should be sorted by index for proper ordering
- The query automatically filters out null questions using list comprehension
- Report progress to user at each step
