# clean-up-storage

MacBook disk cleanup guide for Zarif's dev environment. Run when "storage almost full" hits.

## Quick Start

```bash
df -h /  # Check current free space
```

## Storage Profile (as of 2026-04-17)

Total disk: 460GB. Dev tools dominate usage. Biggest hogs in order:

### 1. Flutter Example Build Dirs (~37GB) — SAFE TO NUKE

Each Flutter example app (whiteboard, lesson_plan, worksheet, wizard, prose) and studio compiles Rust + Dart into native/web binaries. Rebuilds automatically on `flutter run`.

```bash
# Check sizes
du -sh ~/Coding/metarepo/frontend/library/modality/packages/*/example/build \
       ~/Coding/metarepo/frontend/library/modality/apps/studio/build 2>/dev/null | sort -rh

# Clean
rm -rf ~/Coding/metarepo/frontend/library/modality/packages/*/example/build \
       ~/Coding/metarepo/frontend/library/modality/apps/studio/build
```

### 2. Rust Shared Target (~35GB) — SAFE TO NUKE

Cargo workspace compile cache. Slower next build but rebuilds fine.

```bash
du -sh ~/Coding/metarepo/frontend/library/modality/.shared-target
rm -rf ~/Coding/metarepo/frontend/library/modality/.shared-target
```

### 3. Git Worktrees (~13GB) — ASK FIRST

Zarif uses worktrees actively. **Never delete without asking.**

```bash
du -sh ~/Coding/metarepo/frontend/library/modality/.worktrees/modality/*/
# If user approves specific ones:
# git worktree remove <name>
```

### 4. Notion Cache (~11GB) — SAFE, QUIT FIRST

Notion's Partitions folder stores offline page renders. Re-downloads on relaunch.

```bash
# Must quit Notion first
pgrep -x Notion && osascript -e 'quit app "Notion"' && sleep 2
rm -rf ~/Library/Application\ Support/Notion/Partitions
```

### 5. Google Chrome Cache (~8GB) — MANUAL

Can't safely delete from CLI. User must do: Chrome Settings > Privacy > Clear browsing data > Cached images and files.

### 6. Flutter SDK + FVM (~7.5GB)

```bash
du -sh ~/flutter ~/fvm
fvm list  # Check which versions installed, remove unused
```

### 7. Application Support (various)

| App | Typical Size | Notes |
|-----|-------------|-------|
| Claude Desktop | ~13GB | claude-code-vm, sessions, cache |
| Google Chrome | ~8GB | Profile data, clear from Chrome UI |
| Cursor | ~2GB | Extensions, cached VSIXs |
| VoiceInk | ~2GB | Audio models |
| Slack | ~900MB | Workspace cache |
| Superwhisper | ~500MB | Audio models |

### 8. Homebrew (~8GB)

```bash
brew cleanup --prune=0  # Remove old formula versions
# Usually only frees ~10-100MB unless many outdated packages
```

### 9. Rustup Toolchains (~3.7GB)

Zarif uses pinned nightly + stable. Rolling nightly is often redundant.

```bash
rustup toolchain list
# Remove redundant ones (keep active + stable):
# rustup toolchain remove nightly-aarch64-apple-darwin
```

### 10. Misc Caches (~5GB total)

```bash
# UV Python cache (~1.6GB)
rm -rf ~/.cache/uv

# Puppeteer/Chromium (~500MB)
rm -rf ~/.cache/puppeteer

# HuggingFace models (~500MB)
rm -rf ~/.cache/huggingface

# Dart pub cache (~1.7GB) — re-downloads on `flutter pub get`
dart pub cache clean --force

# CockroachDB local data (~1.5GB) — loses local DB
rm -rf ~/Coding/metarepo/frontend/library/modality/server/data/crdb
```

### 11. Nvim Data (~1.6GB) — DO NOT TOUCH

Zarif uses this actively. Located at `~/.local/share/nvim/`.

## Scan Commands

Full audit when unsure where space went:

```bash
# Top-level breakdown
du -sh ~/Library/Application\ Support ~/Library/Caches ~/Coding ~/.cargo ~/.rustup ~/.cache ~/.local ~/flutter ~/fvm 2>/dev/null | sort -rh

# Find large hidden dirs
du -sh ~/.[!.]* 2>/dev/null | sort -rh | head -20

# Find Rust target dirs anywhere
find ~/Coding -name "target" -type d -maxdepth 6 2>/dev/null | xargs -I{} du -sh {} 2>/dev/null | sort -rh | head -10

# Find Flutter build dirs
find ~/Coding -name "build" -type d -maxdepth 7 2>/dev/null | xargs -I{} du -sh {} 2>/dev/null | sort -rh | head -15

# Find node_modules
find ~ -name "node_modules" -type d -maxdepth 5 2>/dev/null | head -20 | xargs -I{} du -sh {} 2>/dev/null | sort -rh
```

## Safety Rules

- **Never delete git worktrees without asking** — may have uncommitted work
- **Never delete nvim data** — active config + plugins
- **Quit Electron apps before cleaning their cache** (Notion, Slack, etc.)
- **Chrome cache: always clean from Chrome UI**, not CLI
- **Flutter builds, .shared-target, pub cache: always safe** — deterministically rebuilt
- Run `df -h /` before and after to verify gains
