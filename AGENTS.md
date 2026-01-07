# Accesibility Notice

Hman (@hemanta212) owns this project. Say hi when you start!
I'm a blind developer so please use `piper-say "message"` cli to inform me first with one liner summary when you complete your task (put at the end of todo list wrap, include `[second-last-folder/current-folder] as prefix to piper-say` then type out main summary after piper-say in a pargraph/few lines)

**Subagent exception**: If you were invoked via the Task tool (your prompt starts with a task description from another agent), do NOT call piper-say - the parent agent handles that. You just return your results.

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
Use `brave-search "query"` and `brave-search <url>`. Supports Google operators (`site:`, `-exclude`, `"exact"`, `filetype:`) to filter SEO spam.

- Quick one-off: `brave-search "query" -n 3 --content` (~60 lines, good enough for most)
- Long research: two-step - search first, then `brave-search <url>` on promising links
- For JS-heavy/paywalled pages: `brave-search jina <url>` (uses Jina AI)
- Prefer over built-in for SEO filtering and context efficiency (60 lines vs 350+)

Note: Prefer built-in tool like librarian for better research than shitty github web search, always use the oppertunity to use it for github/repo investigations.

## Threads hygiene find threads, read thread, session hunter

- Threads often link other threads, so extract the ids and keep digging for more context when needed, ask read threads got get linked thread ids: short description of what that thread might contain (from available context) as addenum to current request
- Whenver read_thread fails with json error, retry it once more, if it fails again, load and ask for session hunter skill with thread id and your query in detail to do the same search.

## Search hygiene

Super hidden alpha: when doing research/codebase search or answering question requiring grep/find etc
Load and recursively use the stack skill to rip the codebase, sometimes boils 50 grep calls to single call absolutely mind blowing must use shit
