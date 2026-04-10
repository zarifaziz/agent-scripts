---
name: babysit-pr
description: Babysit the current PR by checking for merge conflicts, unaddressed review comments, CI failures, and stale branches. Designed to be looped with `/loop 10m /babysit-pr`. Use when the user says "babysit", "watch my PR", "babysit PR", or "loop babysit".
---

# Babysit PR

Continuously monitors and maintains the current PR. Designed to be run on a loop via `/loop`.

## Usage

```bash
# One-shot check
/babysit-pr

# Continuous babysitting (recommended)
/loop 10m /babysit-pr

# Slower cadence for less active PRs
/loop 30m /babysit-pr
```

## Workflow

On each run, perform these checks **in order**. Stop at the first check that requires action, fix it, commit, push, then exit. The next loop iteration will pick up remaining items.

### Check 1: Merge Conflicts

```bash
# Fetch latest main and check if current branch can merge cleanly
git fetch origin main
git merge-tree $(git merge-base HEAD origin/main) HEAD origin/main
```

If there are conflicts:
1. Run `git merge origin/main`
2. Resolve conflicts intelligently (read both sides, pick the right resolution)
3. Commit the merge and push

### Check 2: Unaddressed Review Comments

```bash
# Get the current PR number
PR_NUMBER=$(gh pr view --json number -q '.number')

# Fetch review comments
gh api repos/{owner}/{repo}/pulls/${PR_NUMBER}/comments --jq '.[] | select(.in_reply_to_id == null) | {id: .id, path: .path, body: .body, line: .line}'

# Fetch review threads to find unresolved ones
gh pr view ${PR_NUMBER} --json reviewDecision,reviews,comments
```

For each unaddressed comment:
1. Read the file and understand the reviewer's request
2. Make the requested change
3. Commit with message referencing the review comment
4. Push
5. Reply to the comment thread explaining what was changed

### Check 3: CI Compilation/Test Failures

```bash
# Check CI status
gh pr checks ${PR_NUMBER}
```

**Only act on compilation and test failures** (e.g. `rust-test`, `flutter-test`, `cargo check` errors). These are the checks that actually block merging and are most likely caused by this PR or a merge.

**Ignore formatting and linting failures** (e.g. `rust-format`, `rust-lint`, `flutter-analyze`, `dart-autoformat`). These are often pre-existing on main and fixing them in this PR adds scope creep. Report them to the user but do not fix them.

If a compilation or test check is failing:
1. Read the failure logs: `gh run view <run-id> --log-failed`
2. Check if the failure is in a file changed by this PR: `gh pr diff --name-only`
3. If the failure is in an unrelated file, **report it but do not fix it** — it's likely pre-existing
4. If the failure is related to this PR's changes, diagnose the root cause, fix, commit, and push

### Check 4: Branch Staleness

```bash
# How far behind main?
git rev-list --count HEAD..origin/main
```

If more than 10 commits behind main and no conflicts exist:
1. Merge main to keep the branch fresh
2. Push

TODO(human): Add or remove checks below based on your workflow priorities.
Additional checks you might want:
- Post a Slack update when PR is ready for re-review
- Auto-request re-review after addressing all comments
- Check if dependent PRs have merged and rebase accordingly
- Label PR with status (e.g., "needs-review", "changes-requested", "ready-to-merge")

## Rules

- **One fix per run** — fix the first problem found, push, and exit. Don't try to fix everything at once. The next loop iteration handles the rest.
- **Never force push** — always use regular `git push`. If push is rejected, pull first.
- **Never merge the PR** — babysitting maintains the PR, it doesn't merge it. The human decides when to merge.
- **Be conservative with conflict resolution** — if a conflict is ambiguous (both sides made meaningful changes to the same logic), leave a PR comment asking the author instead of guessing.
- **Commit messages** — prefix with `fix:` or `chore:` and mention what triggered the change (e.g., `fix: address review comment on error handling`).
