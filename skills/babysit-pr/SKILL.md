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

### Check 0: PR Still Open?

```bash
# Check if PR is still open
PR_STATE=$(gh pr view --json state -q '.state')
```

If the PR state is `MERGED` or `CLOSED`:
1. Report the final state to the user (e.g., "PR #123 has been merged. Stopping babysit loop.")
2. Call the `signal_loop_success` tool to stop the loop
3. Exit immediately — no further checks needed

### Check 0.5: PR Title Follows Conventional Commits

```bash
# Get the current PR title
PR_TITLE=$(gh pr view --json title -q '.title')
```

Validate the title matches conventional commit format: `type(scope): description` or `type: description`

- **Valid types:** `feat`, `fix`, `refactor`, `test`, `chore`, `docs`, `ci`, `style`, `perf`, `build`
- **NEVER use `!`** (e.g., `feat!:` is forbidden)

If the title is malformed (missing type prefix, uses `!`, wrong format):
1. Determine the correct type by reading the PR diff: `gh pr diff --name-only`
2. Fix the title: `gh pr edit --title "type: corrected title"`
3. Exit — next loop iteration handles remaining checks

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
1. Read the file and understand the reviewer's feedback
2. Analyze the code in question and formulate the best solution
3. **Present the comment to the developer in the session** with:
   - Who left the comment and what they said
   - The relevant code snippet
   - Your recommended fix (concrete code suggestion)
4. **Stop and wait for the developer** — do NOT make code changes, commit, push, or reply on GitHub until the developer has reviewed your suggestion and confirmed or implemented the fix
5. After the developer has addressed the comment (confirmed the fix, made the change, or told you to proceed), you may reply to the review thread to let the reviewer know it's been addressed

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

If more than 5 commits behind main and no conflicts exist:
1. Merge main to keep the branch fresh
2. Push

Additional checks you might want:
- Auto-request re-review after the developer addresses all comments
- Check if dependent PRs have merged and rebase accordingly
- Label PR with status (e.g., "needs-review", "changes-requested", "ready-to-merge")

## Rules

- **One fix per run** — fix the first problem found, push, and exit. Don't try to fix everything at once. The next loop iteration handles the rest.
- **Never force push** — always use regular `git push`. If push is rejected, pull first.
- **Never merge the PR** — babysitting maintains the PR, it doesn't merge it. The human decides when to merge.
- **Review comments need developer approval first** — never auto-reply to or auto-fix review comments. Present them to the developer with a suggested fix, then wait. Only after the developer has reviewed, confirmed, or implemented the fix may you reply to the reviewer. Autonomously replying before the developer has seen the comment looks like AI slop.
- **Be conservative with conflict resolution** — if a conflict is ambiguous (both sides made meaningful changes to the same logic), report it to the developer instead of guessing.
- **Commit messages** — prefix with `fix:` or `chore:` and mention what triggered the change (e.g., `fix: address review comment on error handling`).
