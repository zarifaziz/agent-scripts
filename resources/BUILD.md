# Build & Debug Guide

## Running the Service

- Launch with log filtering via `./dist/auto-run`, it quickly exits and gives you log file to tail with
- Rerunning the server handles killing+restarting the process.

## Worksheet Plan Scripts

- The shell harnesses under `testing/worksheet_plan/` drive end-to-end flows. Common entry points:
  - `./create_worksheet_plan_test.sh` – smoke test that creates a plan and waits until it’s ready.
  - `./test_update_worksheet_plan.sh`, `./verify_v2_spec.sh`, and `./test_all_cases.sh` for broader regression coverage.
- Each script assumes the local server is running and uses helpers from the same folder; check `testing/worksheet_plan/README.md` for payload examples (`--skill`, `--node-count`, difficulty flags) and environment prerequisites.

## Database Inspection

- Run ad-hoc queries with `cypher-safe --preset resources-dev`. Invoke `cypher-safe` skill for detailed instructions.
- When a Cypher query of roughly 10+ lines fails, dump the full statement plus parameters to the log and invoke `db-oracle` skill for troubleshooting before retrying.

## Preset queries for quick testing

Modify these to suit your needs.

```cypher
// Lesson plan inspect query
MATCH (lp:LessonPlan{id:"3139ee5f-8984-4141-868a-24886753935e"})-[:PLANS]-(l:Lesson)
OPTIONAL MATCH (l)-[:CONTAINS]-(ls:LessonSection)
OPTIONAL MATCH (ls)-[:HAS]-(sk:Skill)
WITH lp, l, ls, sk

CALL {
    WITH ls
    OPTIONAL MATCH (ls)-[:INCLUDES]-(lpn:LessonPlanNode)
    RETURN lpn
}
RETURN lp, l, ls, sk, lpn
```

```cypher
// WSPLAN:: Worksheet plan node inspect query
MATCH (lp:LessonPlan {id:"lp_01K81C3SYPYKXXDX7NB1E3M7E0"})-[*1..3]->(n:LessonPlanNode)
OPTIONAL MATCH (n)--(q:Question|UserQuestion)
WITH DISTINCT n, lp, q
WHERE NOT n:LessonPlanSkill AND NOT n:LessonPlanSkills AND NOT n:LessonPlanLessonAgenda AND NOT n:LessonPlanLearningGoals AND NOT n:LessonPlanSkillSlide
//RETURN labels(n), n.enabled, n.ready
//RETURN count(q)
RETURN lp, n, q
```
