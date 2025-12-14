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
Use `search.js "query"` and `content.js <url>`. Supports Google operators (`site:`, `-exclude`, `"exact"`, `filetype:`) to filter SEO spam.
- Quick one-off: `search.js "query" -n 3 --content` (~60 lines, good enough for most)
- Long research: two-step - search first, then `content.js <url>` on promising links
- Prefer over built-in for SEO filtering and context efficiency (60 lines vs 350+)

# Skills

Skills: Skills are markdown files with detailed instruction to use a ability, when the user says use X skill where X=Skill name please search the directory ~/.opencode/skills/<name> and read the SKILL.md file from it. For example to use 'project-oracle' skill you'll look into ~/.opencode/skills/project-oracle/SKILL.md and follow the guide there
