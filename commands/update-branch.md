---
allowed-tools: Bash(git *), Bash(cargo *), Bash(just *), Bash(dart *), Bash(flutter *), Bash(flutter_rust_bridge_codegen *), Read, Edit, Write, Glob, Grep, Agent
description: Merge main into the current feature branch
---

## Context

- Current branch: !`git branch --show-current`
- Current status: !`git status --short`
- Main branch latest: !`git log origin/main --oneline -1`
- Branch divergence: !`git rev-list --left-right --count origin/main...HEAD`

## Your task

Merge `origin/main` into the current feature branch, resolving all conflicts and ensuring the project builds.

## Strategy

### Phase 1: Prepare

1. Fetch latest from origin: `git fetch origin`
2. Check if merge is needed: `git merge-base --is-ancestor origin/main HEAD` — if true, already up to date.
3. **Format first, merge second** — before merging, pull the `justfile` from main and run `just fmt` (or equivalent formatter). Commit this as a separate formatting commit. This eliminates format-only conflicts and dramatically reduces conflict count.
   ```bash
   git show origin/main:justfile > justfile
   just fmt  # or: dart format ., cargo fmt, etc.
   git add -A && git commit -m "style: format code before merge"
   ```

### Phase 2: Merge

4. Run `git merge origin/main` — if no conflicts, skip to Phase 4.
5. If conflicts exist, categorize them:
   ```bash
   git diff --name-only --diff-filter=U
   ```
   Sort conflicts into three buckets:
   - **Auto-generated files** (FRB codegen, protobuf, `frb_generated.*`, `third_party/`) — accept theirs, regenerate after: `git checkout --theirs <file> && git add <file>`
   - **Format-only conflicts** — accept theirs (main's formatting is canonical): `git checkout --theirs <file> && git add <file>`
   - **Semantic conflicts** — resolve manually, preserving branch's intent on top of main's changes

### Phase 3: Resolve semantic conflicts

6. For each semantic conflict:
   - If the file is badly mangled (orphaned code blocks, unclosed delimiters), reset and re-apply:
     ```bash
     git show origin/main:<file> > <file>
     # Then manually re-apply your branch's changes on top
     ```
   - Otherwise, resolve conflict markers normally, keeping both sides' intent.

7. After all conflicts resolved: `git add -A && git commit --no-edit` (uses merge commit message).

### Phase 4: Regenerate and build

8. **Regenerate codegen** (if applicable):
   ```bash
   cd packages/modality && flutter_rust_bridge_codegen generate
   ```
   After FRB codegen, check for these known issues:
   - `crates/api/Cargo.toml`: FRB may overwrite `flutter_rust_bridge = { workspace = true }` with a pinned version — revert it
   - `frb_generated.rs`: check for `::HashMap` (missing `std::collections` prefix)

9. **Build in layers** — fix errors iteratively between each step:
   ```bash
   cargo check --workspace    # Fast Rust compile check
   cargo test --workspace     # Rust tests
   just whiteboard            # Full Flutter+Rust native build (if applicable)
   ```

10. **Typed wrapper migration** — when main introduces wrapper types (e.g., `String` → `Id`, `String` → `Url`), all Dart consumer code must be updated:
    - Wrap: `Id(field0: value)`, `PropertyKey(field0: key)`, `Url(field0: url)`
    - Unwrap: `id.field0`, `key.field0`, `url.field0`
    - FRB wrapper types can't be `const` in Dart (remove `const` from constructors)
    - Check **both production code AND test files** — tests often have hardcoded string literals that need wrapping
    - Check map key types — `Map<String, X>` may become `Map<Id, X>`

### Phase 5: Commit fixes

11. Commit all post-merge fixes as a single commit:
    ```
    fix: update code for post-merge type/API changes
    ```

## Error recovery

- **Merge fails with "not possible because you have unmerged files"**: You have uncommitted conflict resolutions. Run `git add -A` then `git commit --no-edit`.
- **Build errors after merge**: These are expected. Fix them iteratively — the most common are missing imports, type mismatches from API changes, and stale codegen.
- **FRB codegen crashes**: Check `flutter_rust_bridge.yaml` — new crates with FRB-exposed types need their modules added to `rust_input`.
- **Too many conflicts to resolve manually**: Consider `git merge --abort`, then try the format-first approach from Phase 1 if you haven't already.
