---
allowed-tools: ["Bash(gh pr view*)","Bash(gh pr diff*)","Bash(gh pr diff* | wc -l)","Bash(gh api*)","Glob","Grep","Read","Edit","Bash(gh pr edit*)"]
---

Review the current PR in this repository.

Arguments: $ARGUMENTS

**First, parse the arguments:**
- If arguments contain `-a` or `--author`, run **Author Mode**
- Otherwise run **Reviewer Mode** — treat the remaining arguments as reviewer context
- Extract `--pr <number>` if present — use it in all `gh pr view` and `gh pr diff` commands as `gh pr view <number>` / `gh pr diff <number>`. If `--pr` is absent, omit the number (uses the current branch's PR).

---

# Reviewer Mode

1. Run `gh pr view --json number,title,body,author,state,baseRefName,headRefName`
2. Run `gh pr diff` — count total lines to determine size, then apply the matching section below

## Small PR (diff under 250 lines)

- 1-2 sentence summary of what the PR does
- Technical issues by severity — **Critical** (bugs, security, data loss, incorrect behavior), **Important** (missing error handling, edge cases, performance), **Minor** (style, readability)
- Skip categories with no issues
- If the diff looks clean, say so briefly

## Medium PR (diff 250–700 lines)

**WHAT THIS PR DOES**
2-3 bullets from a product/user perspective, not a code perspective.

**TECHNICAL ISSUES**
Critical and Important only. Skip Minor.

**QUESTIONS FOR THE AUTHOR**
3-5 specific questions about assumptions in the code, edge cases, or integration points that cannot be verified from the diff alone. Not style questions.

## Large PR (diff over 700 lines)

**WHAT THIS PR DOES**
2-4 bullets explaining what changes from a product/user perspective.

**WHERE TO START**
3-5 files or areas that carry the most risk or complexity. For each, one sentence on why. A reviewer with limited time should read these first and skip the rest.

**WHAT NEEDS DOMAIN KNOWLEDGE**
Parts that cannot be verified from code alone. For each, state the specific question a human needs to answer.

**QUESTIONS FOR THE AUTHOR**
5-10 specific questions postable directly as review comments. Not style questions.

**TECHNICAL ISSUES**
Limit to the high-risk areas above. Critical and Important only.

## Before flagging any issue

- If the diff swallows an error but triggers an async event/queue/job, read the consumer/handler to confirm whether it covers the failure case. Do not flag if the async path handles it. Flag if it does not, and say why.
- If the diff references a constant, collection name, or type that might be defined elsewhere, search for it before flagging it as hardcoded or mismatched.
- If the diff changes behavior that depends on callers, grep call sites to understand the impact.

Only flag verified issues. If something cannot be verified, say so and explain what would need to be checked.

Be direct. No praise. No trivial nits.
If reviewer context was provided, use it to sharpen focus across all sections.

---

# Author Mode

You are helping the PR author prepare their PR for review. The author will fix issues and add context before submitting — so surface problems first, then help them communicate the intent clearly.

1. Run `gh pr view --json number,title,body,author,state,baseRefName,headRefName`
2. Run `gh pr diff` and read the changed files

## Step 1 — Critical issues to fix before submitting

Apply the same verification rules as Reviewer Mode — check async paths, grep call sites, read related files before flagging anything.

Flag only **Critical** (bugs, incorrect behavior, security, data loss) and **Important** (unhandled errors, edge cases, wrong assumptions) issues. Skip Minor entirely.

For each issue: file:line, one sentence on what is wrong, one sentence on why it matters.

If nothing is found, say so in one line and move on.

## Step 2 — Audit the PR description

Look at the existing PR body. For each of these that is missing or thin, produce the exact text to add:

- **Problem statement**: what user-facing or system problem does this solve?
- **Key decisions**: what significant choices were made and why? What alternatives were rejected?
- **Tradeoffs / intentional limitations**: what is deliberately not handled? What could go wrong and is accepted?
- **Review focus**: what is the riskiest part? What needs domain knowledge?

Format each gap as:

**DESCRIPTION — [section name]**
> [exact suggested text, ready to paste]

If the PR description already covers something well, skip it.

## Step 3 — Inline comment candidates

Read the changed files. For each location that meets any of these criteria, suggest a comment:

- A decision that looks wrong or surprising but is intentional (swallowed errors, sync where async is expected, unusual control flow)
- A performance tradeoff that is not obvious from the code
- An assumption about input, caller state, or system behavior that is not enforced by types
- Any line where a reviewer will ask "why not X instead?"

Do NOT suggest comments for self-explanatory code.

For each candidate, output exactly:

```
FILE: path/to/file.go
LINE: 123
COMMENT:
// suggested comment text
```

## Step 4 — Summary

One line each:
- How many issues to fix (Step 1)
- How many description gaps (Step 2)
- How many inline comment candidates (Step 3)

If there are issues to fix, lead with those — the author should not submit until Step 1 is clear.

---

After outputting all suggestions, tell the author: if they want to apply them, you can add the inline comments to the files directly and update the PR description — just say "apply all", "apply comments", or "apply description".
