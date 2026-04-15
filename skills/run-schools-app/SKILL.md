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

### 2. Update `schools-app/pubspec_overrides.yaml`

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

### 3. Update `schools-app/justfile`

Update the `TARGET` variable in the `setup` recipe and the `cd` path in the `build-modality` recipe.

**For main:**

```
TARGET="../../library/modality/packages/modality/web/pkg"
```

and build-modality:

```
cd ../../library/modality && just build --web
```

**For a worktree named `<branch>`:**

```
TARGET="../../library/modality/.worktrees/modality/<branch>/packages/modality/web/pkg"
```

and build-modality:

```
cd ../../library/modality/.worktrees/modality/<branch> && just build --web
```

### 4. Run `just run`

```bash
cd ~/Coding/metarepo/frontend/app/schools-app && just run
```

`just run` handles everything: setup (symlinks WASM, flutter pub get), WASM build, and launching Chrome with COOP/COEP headers on localhost.

## Verification

After running, confirm:

1. `pubspec_overrides.yaml` paths all point to the correct worktree (or main)
2. `justfile` TARGET and build-modality paths point to the correct location
3. The app launches in Chrome

## Notes

- The schools-app uses the `main` branch.
