---
allowed-tools: Bash(git checkout --branch:*), Bash(git add:*), Bash(git status:*), Bash(git push:*), Bash(git commit:*), Bash(git log:*), Bash(git diff:*), Bash(git branch:*), Bash(gh pr create:*), Bash(gh pr view:*), Bash(gh pr edit:*)
description: Commit, push, and open a PR
---

## Context

- Current git status: !`git status`
- Current git diff (staged and unstaged changes): !`git diff HEAD`
- Current branch: !`git branch --show-current`

## Your task

Based on the above changes:

1. Create a new branch if on main.
2. Create a single commit with an appropriate message.
3. Push the branch to origin.
4. Create a draft pull request using `gh pr create`. Assign to `zarifaziz`.

### PR title rules (REQUIRED)

PRs are squash-merged. The title becomes the commit message used for automated version bumps.

- **Format:** `type(scope): description` or `type: description`
- **Valid types:** `feat`, `fix`, `refactor`, `test`, `chore`, `docs`, `ci`, `style`, `perf`, `build`
- **The type is determined by the nature of the changes**, not the ticket. Read the diff to decide.
- **NEVER use `!` for breaking changes** (e.g., `feat!:` is forbidden). Mention breaking changes in the PR description instead.

#### Ticket number preservation

- If `$1` is provided as a ticket id (e.g. `ISSUE-123`, `TASK-4567`), append it: `type: description [$1]`.
- If no `$1` but the **branch name** contains a ticket pattern like `ISSUE-XXX`, `TASK-XXXX`, `BUG-XXX`, `FEAT-XXX`, etc., extract it and append `[TICKET-XXX]` to the title.
- If a ticket tag already exists in any commit message or branch name, **preserve it exactly** — do not strip, rename, or reformat it.
- Examples:
  - Branch `ISSUE-123-fix-auth` + auth bug fix → `fix: correct token expiry check [ISSUE-123]`
  - `$1=TASK-9` + new layout feature → `feat: add AI layouts awareness [TASK-9]`

### PR description rules

Follow the `make-pr-descriptions` skill at `~/.claude/skills/make-pr-descriptions/SKILL.md`. Summary:

- **One-line purpose** at the top — what & why for reviewers.
- **`**Changes:**` bullet list** — max 5 bullets, one per logical change. Reference actual files/functions.
- **Call stack / ASCII flowchart** — include for any change touching 2+ files or non-obvious execution paths. Skip only for trivial single-file edits.
- Wrap all software terms (functions, files, vars, types) in backticks.
- No fluff: skip "This PR...", "In this change...", etc.
- No `Claude Code` / co-author trailers.

Pass description via `--body-file -` heredoc to preserve newlines (avoid `\n` escapes).

Template:

```
gh pr create --draft --assignee zarifaziz --title "type: description [TICKET-XXX]" --body-file - <<'EOF'
One sentence purpose for reviewers.

**Changes:**
- Changed `X` in `path/to/file.ts` to fix `Z`
- Added `funcName()` for edge case `D`

**Call Stack:**
entrypoint()
├── helperA()
└── helperB()
EOF
```

### Execution

5. Call all required tools in a **single message** (parallel where possible). No extra prose, no other tools.
6. Do not add `Claude Code` or other co-author trailers.
