#!/usr/bin/env bash
# Wire agent-scripts dirs into ~/.claude/ via symlinks.
# Idempotent: safe to run multiple times.

set -euo pipefail

REPO="$(cd "$(dirname "$0")" && pwd)"
mkdir -p "$HOME/.claude"

# link <src> <dst> — symlink src -> dst, refuse to clobber non-matching state.
link() {
  local src="$1" dst="$2"
  if [[ -L "$dst" ]]; then
    if [[ "$(readlink "$dst")" == "$src" ]]; then
      echo "ok: $dst"
      return
    fi
    echo "error: $dst points to $(readlink "$dst"), expected $src" >&2
    exit 1
  fi
  if [[ -e "$dst" ]]; then
    echo "error: $dst exists and is not a symlink" >&2
    exit 1
  fi
  ln -s "$src" "$dst"
  echo "linked: $dst -> $src"
}

link "$REPO/skills"   "$HOME/.claude/skills"
link "$REPO/commands" "$HOME/.claude/commands"
