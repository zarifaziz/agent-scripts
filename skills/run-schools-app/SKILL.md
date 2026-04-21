---
name: run-schools-app
description: Point the schools-app at the current modality worktree (or main) and launch it. Auto-detects the active worktree from the current working directory. Use when asked to "run schools app", "run the schools app", "launch schools", "test in schools app", or "run-schools-app".
---

# Run Schools App

Automates testing modality features in the schools-app frontend by detecting the active worktree, updating config files to point at it, and launching the app.

## Schools App Location

```
~/Coding/metarepo/frontend/app/schools-app
```

## Steps

### 1. Auto-detect the worktree

Determine the modality source from the current working directory:

- **If CWD contains `.worktrees/modality/<name>`** → extract `<name>` as the worktree branch. The modality root is `.worktrees/modality/<name>/`.
- **If CWD is the main modality dir** (`~/Coding/metarepo/frontend/library/modality`) → use main paths (no worktree).

### 2. Sync `schools-app` with `origin/main`

The schools-app checkout can drift behind origin/main, causing stale `justfile`/`pubspec` patterns that differ from the skill's expected values. Always sync before editing:

```bash
cd ~/Coding/metarepo/frontend/app/schools-app
git fetch origin main
# If behind origin/main, stash local edits (if any), fast-forward, then restore:
git stash push -m "run-schools-app skill edits" -- justfile pubspec_overrides.yaml pubspec.lock 2>/dev/null || true
git pull --ff-only origin main
# If stash had changes, pop. If pubspec.lock conflicts, take upstream
# (flutter pub get regenerates it):
git stash pop 2>/dev/null || true
if git status --short | grep -q "^UU pubspec.lock"; then
  git checkout --theirs -- pubspec.lock && git reset HEAD pubspec.lock
fi
```

Verify the branch is `main` and up to date with `origin/main` before continuing.

**Merge conflicts in code files (not pubspec.lock) must be surfaced to the user.** Only auto-resolve `pubspec.lock` since `flutter pub get` regenerates it.

### 3. Update `schools-app/pubspec_overrides.yaml`

Update all 10 modality package paths in `pubspec_overrides.yaml` to point at the detected source.

**Modality packages to update:**

- `modality`
- `drive`
- `whiteboard`
- `autogrid`
- `worksheet`
- `lesson_plan`
- `document`
- `ds`
- `prose`
- `wizard`

**Path patterns (relative to schools-app directory):**

For **main**:

```yaml
modality:
  path: ../../library/modality/packages/modality/
drive:
  path: ../../library/modality/packages/drive/
# ... same pattern for all 10 packages
```

For a **worktree** named `<branch>`:

```yaml
modality:
  path: ../../library/modality/.worktrees/modality/<branch>/packages/modality/
drive:
  path: ../../library/modality/.worktrees/modality/<branch>/packages/drive/
# ... same pattern for all 10 packages
```

**Important:** Only update the 10 modality package paths listed above. Do not touch other dependency overrides in the file (e.g., `learning_library`, `whiteboard_legacy`, `graphql_codegen`, etc.).

### 4. Update `schools-app/justfile`

Update the `TARGET` variable in the `setup` recipe and the `cd` path in the `build-modality` recipe.

**Path-depth gotcha:**
- `TARGET` is a symlink target resolved **from `schools-app/web/`** (the symlink's own directory), so it needs **three** `../` segments to climb out of `schools-app/web/` → `schools-app/` → `app/` → `frontend/`.
- `build-modality`'s `cd` runs **from `schools-app/`**, so it only needs **two** `../` segments (`schools-app/` → `app/` → `frontend/`).
- `pubspec_overrides.yaml` paths are resolved **from `schools-app/`**, so they use **two** `../` segments.

Don't conflate these — they are not all the same depth.

**For main:**

```
TARGET="../../../library/modality/packages/modality/web/pkg"
```

and build-modality:

```
cd ../../library/modality && just build --web
```

**For a worktree named `<branch>`:**

```
TARGET="../../../library/modality/.worktrees/modality/<branch>/packages/modality/web/pkg"
```

and build-modality:

```
cd ../../library/modality/.worktrees/modality/<branch> && just build --web
```

### 5. Ensure `--web-port=5001` in `schools-app/justfile`

The `run` recipe must pass `--web-port=5001`, not `5000`. macOS's ControlCenter/AirPlay Receiver binds `*:5000` on both IPv4 and IPv6 by default (since macOS Monterey), which collides with Flutter's dev server even when `--web-hostname=localhost` is set. Port `5001` avoids this.

If the synced `justfile` from origin/main still has `--web-port=5000`, change it to `5001`.

### 6. Run `just run`

```bash
cd ~/Coding/metarepo/frontend/app/schools-app && just run
```

`just run` handles everything: setup (symlinks WASM, flutter pub get), WASM build, and launching Chrome with COOP/COEP headers on localhost.

## Verification

After running, confirm:

1. `schools-app` is on `main` and fast-forwarded to `origin/main` (no unresolved conflicts except auto-handled `pubspec.lock`)
2. `pubspec_overrides.yaml` paths all point to the correct worktree (or main) with **two** `../` segments
3. `justfile` `TARGET` uses **three** `../` segments, `build-modality` `cd` uses **two**
4. `justfile` `run` recipe passes `--web-port=5001`
5. The app launches in Chrome on http://localhost:5001

## Notes

- The schools-app uses the `main` branch.
- If Chrome fails to bind the port, inspect with `lsof -nP -iTCP:5001 -sTCP:LISTEN`. If a stale `dartvm` from a prior run is holding the port, kill it with `kill -9 <pid>`. If macOS ControlCenter (AirPlay Receiver) is holding port 5000, port 5001 avoids the conflict.
