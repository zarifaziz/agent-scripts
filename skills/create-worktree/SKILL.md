---
name: create-worktree
description: Create a git worktree for any repo under `.worktrees/<BRANCH_NAME>/`, run detected setup steps (copy .env/CLAUDE.local.md, install deps), and push an init commit. Use when asked to "make a worktree", "create a worktree", "set up a worktree", or "new worktree for this branch".
---

# Create Worktree

Creates a git worktree for any repo under `.worktrees/<BRANCH_NAME>/`, copies gitignored env files if found, installs dependencies based on the detected language/toolchain, and pushes an empty init commit so the branch exists on origin.

For the modality repo specifically, use [`create-modality-worktree`](../create-modality-worktree/skill.md) instead — it handles modality-specific paths.

## When to Use

- Starting a feature or fix without leaving the current branch
- Need an isolated working directory for any git repo
- Want the worktree pre-wired with env files and installed deps so it's immediately usable

## Required Inputs

| Input | Source |
|-------|--------|
| `<REPO_PATH>` | Provided by caller, or CWD if already inside a git repo |
| `<BRANCH_NAME>` | Must be provided by caller — prompt if missing, never invent one |

## Worktree Location

`.worktrees/<BRANCH_NAME>/` relative to the repo root.

## One-Shot Command

Run from anywhere. Substitute `<REPO_PATH>` and `<BRANCH_NAME>`:

```bash
cd <REPO_PATH> && \
git fetch origin main && \
mkdir -p .worktrees && \
git worktree add .worktrees/<BRANCH_NAME> -b <BRANCH_NAME> origin/main && \
cd .worktrees/<BRANCH_NAME> && \
git commit --allow-empty -m "chore: Init commit" && \
git push -u origin <BRANCH_NAME>
```

Setup steps (env files, deps) are detected and run separately — see below.

## Step-by-Step

```bash
cd <REPO_PATH>
git fetch origin main
mkdir -p .worktrees
git worktree add .worktrees/<BRANCH_NAME> -b <BRANCH_NAME> origin/main
cd .worktrees/<BRANCH_NAME>
```

### Copy env files (only if they exist at the repo root or known subdirs)

```bash
# Repo root
[ -f ../../.env ] && cp ../../.env .

# Common subdirs — create the dir first if needed
for d in server backend app; do
  [ -f "../../$d/.env" ] && mkdir -p "$d" && cp "../../$d/.env" "$d/.env"
done

# CLAUDE.local.md
[ -f ../../CLAUDE.local.md ] && cp ../../CLAUDE.local.md .
```

### Run setup scripts (only if present)

```bash
# Makefile targets
if [ -f Makefile ]; then
  grep -q '^install:' Makefile && make install
  grep -q '^setup:'   Makefile && make setup
fi

# Shell setup scripts
[ -x ./setup.sh   ] && ./setup.sh
[ -x ./install.sh ] && ./install.sh
```

### Language-specific dep install (only for detected toolchains)

```bash
# Dart / Flutter
[ -f pubspec.yaml ] && flutter pub get

# Node
[ -f package.json ] && {
  [ -f pnpm-lock.yaml ] && pnpm install ||
  [ -f yarn.lock ]      && yarn         ||
  npm install
}

# Python
[ -f requirements.txt ] && pip install -r requirements.txt

# Go
[ -f go.mod ] && go mod download

# Ruby
[ -f Gemfile ] && bundle install

# Rust
[ -f Cargo.toml ] && cargo fetch
```

### Push init commit

```bash
git commit --allow-empty -m "chore: Init commit"
git push -u origin <BRANCH_NAME>
```

## Verification

After running, confirm:

1. `.worktrees/<BRANCH_NAME>/` directory exists inside the repo root
2. `git -C .worktrees/<BRANCH_NAME> rev-parse --abbrev-ref HEAD` prints `<BRANCH_NAME>`
3. Any `.env` files that exist at the root are present in the worktree
4. `CLAUDE.local.md` exists in the worktree if it existed at the root
5. `git -C .worktrees/<BRANCH_NAME> log --oneline -1` shows the init commit
6. `git ls-remote origin <BRANCH_NAME>` returns a ref (push succeeded)

## Cleanup

When done with a worktree:

```bash
cd <REPO_PATH>
git worktree remove .worktrees/<BRANCH_NAME>
git branch -D <BRANCH_NAME>             # local branch (optional)
git push origin --delete <BRANCH_NAME>  # remote branch (optional, after PR merged)
```

---

## Subagent Mode

When invoked via the Task tool as a subagent:

1. Require `<BRANCH_NAME>` and `<REPO_PATH>` (or confirm CWD is inside a git repo). If branch name is missing, return `STATUS: failure` with `ERROR: branch name required`.
2. Run the one-shot command, then the setup detection block.
3. If any dep install step fails, abort before committing/pushing and report the failure — do not push a half-installed worktree.
4. Return structured output:
   ```
   STATUS: success | failure
   WORKTREE_PATH: <absolute path to worktree>
   BRANCH: <BRANCH_NAME>
   SETUP_STEPS_RUN: <comma-separated list of steps that ran, e.g. "copy .env, npm install">
   ERROR: <error message if failed>
   ```
5. Do not ask clarifying questions beyond the two required inputs. Use defaults for everything else.
6. Do not open a PR — that is the caller's responsibility.
