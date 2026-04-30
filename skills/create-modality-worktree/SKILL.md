---
name: create-modality-worktree
description: Create a git worktree for the modality repo with CLAUDE.local.md, server/.env, and flutter pub get pre-installed, then commit and push the branch. Use when asked to "make a modality worktree", "set up a modality worktree", "create a modality worktree", "start a modality worktree", or "worktree for this modality issue".
---

# Create Modality Worktree

Creates a git worktree for the modality library with the modality-specific environment files copied in, dependencies installed, and an init commit pushed to origin.

For non-modality repos use [`creating-worktrees`](../creating-worktrees/SKILL.md) instead.

## When to Use

- Starting work on a modality issue without leaving the current branch
- Need an isolated working directory for a modality feature/fix
- Want the worktree pre-wired with `CLAUDE.local.md`, `server/.env`, and resolved Dart deps so it boots immediately

## Repo Location

```
~/Coding/metarepo/frontend/library/modality
```

## Worktree Location

`.worktrees/<BRANCH_NAME>/` — two levels deep from the modality root. Path math in the copy commands assumes this layout.

> Note: `.superset/config.json` uses three-level paths (`../../../`) because superset places worktrees at `.worktrees/modality/<branch>/`. This skill deliberately uses two-level paths and writes to `.worktrees/<branch>/` directly.

## Branch Name

The caller should pass `<BRANCH_NAME>`. If they don't, prompt the user for one before running anything. Do not invent a branch name.

## Files Copied From Modality Root

| File | Why |
|------|-----|
| `CLAUDE.local.md` | Zarif's per-project Claude instructions; overrides repo `CLAUDE.md` rules (e.g. merge-conflict handling) |
| `server/.env` | Backend secrets/config; modality server won't boot without it |

`flutter pub get` is run after the copies so package overrides resolve against the worktree's own checkout.

## One-Shot Command

Run from anywhere. Substitute `<BRANCH_NAME>`:

```bash
cd ~/Coding/metarepo/frontend/library/modality && \
git fetch origin main && \
mkdir -p .worktrees && \
git worktree add .worktrees/<BRANCH_NAME> -b <BRANCH_NAME> origin/main && \
cd .worktrees/<BRANCH_NAME> && \
cp ../../CLAUDE.local.md . && \
mkdir -p server && cp ../../server/.env server/.env && \
flutter pub get && \
git commit --allow-empty -m "chore: Init commit" && \
git push -u origin <BRANCH_NAME>
```

## Step-by-Step (Manual)

```bash
cd ~/Coding/metarepo/frontend/library/modality
git fetch origin main
mkdir -p .worktrees
git worktree add .worktrees/<BRANCH_NAME> -b <BRANCH_NAME> origin/main
cd .worktrees/<BRANCH_NAME>

# Copy gitignored env files from modality root
cp ../../CLAUDE.local.md .
mkdir -p server && cp ../../server/.env server/.env

# Install Dart deps
flutter pub get

# Init commit and push so the branch exists on origin
git commit --allow-empty -m "chore: Init commit"
git push -u origin <BRANCH_NAME>
```

## Verification

After running, confirm:

1. `.worktrees/<BRANCH_NAME>/CLAUDE.local.md` exists and matches the file at the modality root
2. `.worktrees/<BRANCH_NAME>/server/.env` exists
3. `.worktrees/<BRANCH_NAME>/.dart_tool/` exists (created by `flutter pub get`)
4. `git -C .worktrees/<BRANCH_NAME> rev-parse --abbrev-ref HEAD` prints `<BRANCH_NAME>`
5. `git -C .worktrees/<BRANCH_NAME> log --oneline -1` shows the init commit
6. `git ls-remote origin <BRANCH_NAME>` returns a ref (push succeeded)

## Cleanup

When done with a worktree:

```bash
cd ~/Coding/metarepo/frontend/library/modality
git worktree remove .worktrees/<BRANCH_NAME>
git branch -D <BRANCH_NAME>      # local branch (optional)
git push origin --delete <BRANCH_NAME>  # remote branch (optional, after PR merged)
```

---

## Subagent Mode

When invoked via the Task tool as a subagent:

1. Require `<BRANCH_NAME>` as input. If missing, return `STATUS: failure` with `ERROR: branch name required`.
2. Run the one-shot command above.
3. If `flutter pub get` fails, abort the rest and report — do not commit/push a half-installed worktree.
4. Return structured output:
   ```
   STATUS: success | failure
   WORKTREE_PATH: /Users/zariftutero/Coding/metarepo/frontend/library/modality/.worktrees/<BRANCH_NAME>
   BRANCH: <BRANCH_NAME>
   ERROR: <error message if failed>
   ```
5. Do not ask clarifying questions beyond the branch name. Use defaults for everything else.
6. Do not open a PR — that is the caller's responsibility.
