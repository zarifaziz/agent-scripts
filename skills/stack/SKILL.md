---
name: stack
description: Trace Go function call graphs. Use for "how does X work" or "who calls Y" questions instead of chaining greps.
allowed-tools:
  - bash
metadata:
  version: "1.0"
---

# Stack - Call Graph Tracer

Traces Go function calls. Replaces manual grep chains for code flow analysis.

## Core Commands

```bash
stack funcName -depth 10      # trace DOWN from function
stack funcName -ref           # trace UP (who calls this?)
stack funcName -format mermaid # diagram output
```

## When to Use

| Question                          | Command                              |
| --------------------------------- | ------------------------------------ |
| "How does this mutation work?"    | `stack editLessonPlanNode -depth 10` |
| "Who calls this helper?"          | `stack helperFunc -ref`              |
| "What implements this interface?" | `stack Repository.Method`            |

## Patterns

```bash
# Entry point → full flow
stack editLessonPlanNode -depth 12

# Find all callers of a function
stack generateFromLLM -ref

# Interface auto-resolves to impl
stack LessonPlanNodeRepository.Edit
# → nodeRepositoryImpl.Edit

# Multiple impls? Pick specific one
stack warmUpQuestionsNodeRepositoryImpl.RegenerateWithInput -depth 8
```

## Direction

| Flag       | Direction | Use Case           |
| ---------- | --------- | ------------------ |
| `-depth N` | Top-down  | "How does X work?" |
| `-ref`     | Bottom-up | "Who calls X?"     |

## Don't Use Stack For

- Text/string search → `rg`
- Regex patterns → `rg -e`
- Non-Go files → `rg`
- Structural patterns → `ast-grep`

## Fallback

Non-functions (types, structs) auto-fallback to ripgrep:

```bash
stack LessonPlanEditInput  # → ripgrep results
```

## Combo Workflows

**rg → stack** (error hunting):

```bash
rg "failed to regenerate" internal/   # find error location
stack generateFromLLM -ref            # trace who calls it
```

**stack → rg** (find related strings):

```bash
stack editLessonPlanNode -depth 10    # trace the flow
rg "oops.Wrapf" internal/lesson_plan/ # find error handling in those files
```

**ast-grep → stack** (pattern → specific impl):

```bash
ast-grep -p 'func ($R) RegenerateWithInput($$$) ($$$) { $$$ }' -l go internal/
stack warmUpQuestionsNodeRepositoryImpl.RegenerateWithInput -ref
```
