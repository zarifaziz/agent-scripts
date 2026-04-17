---
allowed-tools: ["Bash(gh api*)","Bash(gh pr view*)","Bash(gh pr diff*)","Bash(git diff*)","Bash(git log*)","Bash(git status*)","Bash(git add*)","Bash(git commit*)","Bash(git push*)","Bash(cargo*)","Bash(flutter*)","Glob","Grep","Read","Edit","Agent"]
---

Address review comments on the current PR.

Arguments: $ARGUMENTS

**Parse arguments:**
- Extract `--pr <number>` if present — use it in all `gh` commands. If absent, use the current branch's PR.
- Remaining arguments are treated as additional context.

---

## Step 1 — Fetch PR and review comments

1. Run `gh pr view --json number,title,headRefName,baseRefName` to get PR metadata
2. Fetch all review comments:
   ```
   gh api repos/{owner}/{repo}/pulls/{number}/comments --paginate
   ```
3. Also fetch PR review bodies (top-level review summaries):
   ```
   gh api repos/{owner}/{repo}/pulls/{number}/reviews --paginate
   ```
4. Collect all unresolved review comments regardless of who left them (human reviewers, ClaudeBot, other bots).

## Step 2 — Triage each comment

For each comment, critically evaluate whether it is legitimate:

1. **Read the relevant code** — don't just trust the comment. Open the file at the referenced line and understand the full context.
2. **Verify the claim** — check if the issue actually exists. Grep for related code, read surrounding functions, check if the concern is already handled elsewhere.
3. **Classify** into one of:
   - **Legit fix needed** — the comment identifies a real bug, missing error handling, incorrect behavior, or valid improvement
   - **Legit but already handled** — the concern is valid in general but is already addressed by other code the bot didn't see
   - **Not applicable** — the suggestion doesn't apply to this codebase (wrong language idioms, misunderstood architecture, outdated advice)
   - **Stylistic/subjective** — the comment is about style preferences, not correctness

Output a summary table:

```
| # | File:Line | Issue | Verdict | Action |
|---|-----------|-------|---------|--------|
```

## Step 3 — Address legitimate comments

For each "Legit fix needed" comment:
- Make the code change
- Keep changes minimal and focused — fix exactly what the comment identifies, nothing more

## Step 4 — Respond to comments on the PR

For each comment, post a reply on the PR explaining what you did:

- **Legit fix needed** → reply with: "Fixed — [brief description of what changed]"
- **Legit but already handled** → reply with: "This is already handled by [explanation with file:line reference]"
- **Not applicable** → reply with: "Pushing back — [brief explanation of why this doesn't apply]"
- **Stylistic/subjective** → reply with: "Skipping — this is a style preference, not a correctness issue. [brief rationale if needed]"

Use this to post replies:
```
gh api repos/{owner}/{repo}/pulls/{number}/comments/{comment_id}/replies -f body="..."
```

## Step 5 — Commit and push

If any code changes were made:
1. Stage only the changed files
2. Commit with message: `fix: address PR review comments`
3. Push to the PR branch

## Step 6 — Summary

Output a brief summary:
- How many comments were addressed with code changes
- How many were pushed back on
- How many were already handled
- Link to the PR
