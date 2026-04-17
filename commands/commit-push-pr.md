---
allowed-tools: Bash(git checkout --branch:*), Bash(git add:*), Bash(git status:*), Bash(git push:*), Bash(git commit:*), Bash(gh pr create:*)
description: Commit, push, and open a PR
---

## Context

- Current git status: !`git status`
- Current git diff (staged and unstaged changes): !`git diff HEAD`
- Current branch: !`git branch --show-current`

## Your task

Based on the above changes:

1. Create a new branch if on main
2. Create a single commit with an appropriate message
3. Push the branch to origin
4. Create a draft pull request using `gh pr create`. Assign to 'zarifaziz'. Use a simple two paragraph description of the changes made. Make sure all software terms are wrapped in codefences.
   - The PR title **must** follow conventional commit format: `type(scope): description` or `type: description`
   - Valid types: `feat`, `fix`, `refactor`, `test`, `chore`, `docs`, `ci`, `style`, `perf`, `build`
   - If a ticket number is provided as $1, use format: `type: $2 [$1]` (e.g., `feat: AI layouts awareness implementation [ISSUE-123]`)
   - **NEVER use `!` for breaking changes** (e.g., `feat!:` is forbidden). If a change is breaking, mention it in the PR description instead.
   - The type is determined by the nature of the changes, not the ticket. Read the diff to decide.
   - This is critical because PRs are squash-merged and the PR title becomes the commit message used for automated version bumps.
5. You can use the "create pr descroption" skill to create a description for the PR.
6. You have the capability to call multiple tools in a single response. You MUST do all of the above in a single message. Do not use any other tools or do anything else. Do not send any other text or messages besides these tool calls.
7. Do not add extra contributors, such as Claude Code.
