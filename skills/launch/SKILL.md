---
name: launch
description: Instructions for researching current project context, matching relevant skills, and launching an agent in tmux with comprehensive context.
metadata:
  version: "1.0"
---

# Launch Skill - Instructions for LLM

## What You Must Do

Given a user prompt, you must:

1. Research the current project to gather context (file paths, implementations, patterns)
2. Optionally search web for external API docs if needed
3. Run `skill-info` command and identify relevant skills with explanations
4. Assemble comprehensive context with file path citations and highlighted skills
5. Spawn a tmux window in session 'agents' (create if doesn't exist)
6. Launch the `am` command with all context via heredoc

## Step 1: Research Current Project

**Primary Focus: Project Research**

Use available tools to understand the current project:

- **finder**: Search for relevant code by concepts/functionality
- **Grep**: Find specific patterns, function names, imports
- **Read**: Examine configuration files, existing implementations
- **glob**: Locate files by patterns (_.go, _.ts, etc.)

**Research Goals:**

- Understand existing implementations related to the task
- Find configuration files and patterns
- Locate relevant modules/packages
- Identify architectural patterns in use
- Discover similar existing features

**Citation Format:**
Always cite actual file paths from the project:

```
Project Context:
- Authentication implementation: /src/auth/handler.go
- Database models: /models/user.go
- Configuration: /config/database.yml
- Similar feature: /features/payments/processor.ts
```

**Secondary: Web Research (Optional)**

Only use web search when:

- Need external API documentation (official docs)
- Integrating with third-party services
- Verifying compatibility or versions

Format web citations:

```
External References:
- Official API docs: [https://example.com/api/docs]
- Integration guide: [https://docs.service.com/integration]
```

## Step 2: Identify Relevant Skills

Run `skill-info` command to get all available skills with descriptions:

```bash
SKILLS_LIST=$(skill-info)
```

Analyze the output and match skills based on:

- **Task keywords**: database, refactor, search, analyze, etc.
- **Technology mentions**: Neo4j, PostgreSQL, GraphQL, etc.
- **Task type**: implementation, planning, debugging, querying

**Matching Strategy:**

- Identify 1-3 most relevant skills for the main task
- Note any supporting skills that might be useful
- Understand what each matched skill does

**Format for Context:**

```
## Relevant Skills (Highlighted)

### PRIMARY: skill-name
[Description]
Why relevant: [Explain how it applies to this task]
Location: ~/.opencode/skills/skill-name/SKILL.md

### SUPPORTING: another-skill
[Description]
Why relevant: [Explain supporting role]
Location: ~/.opencode/skills/another-skill/SKILL.md
```

## Step 3: Assemble Context Document

```markdown
# Task Context for Agent

## Original Prompt

[User's exact request]

## Project Context

### Current Implementation

[What exists in the project related to this task]

### Citations (File Paths)

- /path/to/relevant/file.ext: [what it contains]
- /path/to/config/file.yml: [relevant config]
- /path/to/similar/feature.go: [similar implementation]

### External References (if any)

- [URL]: [description]

## Relevant Skills (Highlighted)

### PRIMARY: skill-name

[Description]
Why relevant: [Clear explanation of relevance to this task]
Location: ~/.opencode/skills/skill-name/SKILL.md

### SUPPORTING: skill-name-2

[Description]
Why relevant: [How it supports the main task]
Location: ~/.opencode/skills/skill-name-2/SKILL.md

---

## Task Instructions

1. Review project context and citations
2. Focus on PRIMARY skills highlighted above
3. Use SUPPORTING skills as needed
4. [Specific task instructions based on prompt]
```

## Step 4: Launch in Tmux

```bash
# Create session if doesn't exist
tmux has-session -t agents 2>/dev/null || tmux new-session -d -s agents

# Launch am command with context via heredoc
tmux send-keys -t agents "am <<'EOF'
[Insert full context document here]
EOF
" C-m
```
