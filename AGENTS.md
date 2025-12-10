# Accesibility Notice

Hman (@hemanta212) owns this project. Say hi when you start!
I'm a blind developer so please use `piper-say`utility via piper-tts skill to inform me with one liner summary when you complete your task (put at the end of todo list wrap, include `[second-last-folder/current-folder] as prefix to piper-say`)

## Important!

- **NEVER write markdown files (README, SUMMARY, etc.) unless explicitly requested** - All task summaries and completion reports should be communicated directly to the user in conversation (TLDR format always)
- Keep all communication concise and actionable - the user prefers direct verbal updates over written documentation
- The user is already blind, so don't punish him more with your summary/task md files unless asked explicilty, also don't use emojis or other visual aids (i'm sure u can guess why)
- Only and only when explicitly requested, mkdir -p .plans/ folder and put all your md files there, never ever pollute main dir .

## Agent Protocol

- When working in a project which is under ~/Coding/metarepo/ root, you can use other folders under ~/Coding/metarepo/ to find dependencies/infrastructure i.e the whole stack for reference/lookup/research.
- PR links: use gh pr view/diff instead of pasting URLs.
- Add notes to AGENTS only when the user says “make a note”; edit AGENTS.MD (ignore CLAUDE.md symlink).
- Need an upstream file? Redirect to /tmp/, then cherry-pick—never overwrite tracked files.
- Commits: Conventional Commits (feat|fix|refactor|build|ci|chore|docs|style|perf|test; e.g., feat(api): add telemetry, chore!: drop support for iOS 16).
- Use the skills when appropriate: eg cypher-safe for db testing/verify, db-oracle for hard queries/finding out schema, autist for complex consultation
- Internet: search early/often, quote exact errors, prefer 2024–2025 sources
- Autist/Oracle hygiene: reuse session id for follow-up and context reuse, fish out session id from ~/.cache/scripts/autist/<latest-file.log> and say 'continue' using that session id

## Critical Thinking

- Chase root cause, not band-aids—trace upstream and fix the real break.
- Unsure? Read more code first; if still blocked, ask with a short options summary.
- Flag conflicting instructions and propose the safer path.
- Write down findings in the task thread so others can follow the reasoning.
- When tasked with a bug or debugging something, and the information isn't enough to deduce just add logs and re-run yourself or prompt for re-run if you can't run yourself. Don't assume and do the fix, verify it first

## Web Search

IMPORTANT: Do NOT use the built-in web_search tool - it will always fail.
Use the `web-search` skill instead

### search.js tips
- Supports all Google operators: `site:github.com`, `-exclude`, `"exact phrase"`, `filetype:pdf`
- Use `-n 10` for more results (default 5)
- Quick one-off: `search.js "query" -n 3 --content` (~60 lines, good enough for most)
- Long research: two-step - search first, then `content.js <url>` on promising links
- Prefer over built-in for SEO filtering and context efficiency (60 lines vs 350+)

# Skills

Skills: Skills are markdown files with detailed instruction to use a ability, when the user says use X skill where X=Skill name please search the directory ~/.opencode/skills/<name> and read the SKILL.md file from it. For example to use 'project-oracle' skill you'll look into ~/.opencode/skills/project-oracle/SKILL.md and follow the guide there

## Available skills are (but not limited to):

```

autist: Consult a project-focused agent with deep reasoning capabilities for code analysis, refactoring, and development tasks.
big-brain: Consult the Big Brain expert for deep technical analysis, architectural decisions, refactoring strategies, debugging, and code reviews using Claude Opus.
browser-tools: Minimal CDP tools for collaborative site exploration.
cypher-safe: Execute Cypher queries against Neo4j databases. Use when running read queries, testing write operations, or working with Neo4j databases.
db-oracle: Consult the all-knowing Neo4j Oracle for medium/large Cypher queries. The Oracle knows the entire database schema, all Go code interfacing with Neo4j, and every query pattern ever written.
docker-restore: Restore Neo4j database backups from Google Cloud Storage to a local Docker container. Use when the user asks to restore a Neo4j backup, mentions gsutil backup paths, or needs to restore from gs://neo4j-backups-dev/resources/.
frontend-design: Create distinctive, production-grade frontend interfaces with high design quality. Use this skill when the user asks to build web components, pages, or applications. Generates creative, polished code that avoids generic AI aesthetics.
inspect-resource-question: Extract and inspect AI-generated questions from lesson plans or worksheet plans. Formats quillDocument questions into readable plaintext.
launch: Instructions for researching current project context, matching relevant skills, and launching an agent in tmux with comprehensive context.
local-librarian: Search and analyze local repositories with a specialized agent that explores codebases, understands architecture, and explains implementations.
piper-tts: Use piper-say TTS tool to provide audio feedback when completing code implementation tasks. Primarily uses English (piper-say) unless user explicitly requests Nepali (piper-sayn).
psql-safe: Execute SQL queries against PostgreSQL/CockroachDB databases. Use when running read queries, testing write operations, or working with CockroachDB databases.
reverse-graphql-input: Reverse engineer GraphQL input from a lesson plan or worksheet plan ID/link. Extracts plan data and reconstructs the original input JSON used to create it.
session-hunter: Hunt down Amp sessions by code changes, file paths, or function names.
users home dir: This users home directory is /Users/mac
web-search: Search the web
```
