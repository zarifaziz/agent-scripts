#!/usr/bin/env bash
# Wire agent-scripts skills and commands into ~/.claude/, one symlink per item:
#
#   ~/.claude/skills/<name>      -> <repo>/skills/<name>
#   ~/.claude/commands/<name>.md -> <repo>/commands/<name>.md
#
# Per-item links, not whole directories: that lets ~/.claude/skills hold skills
# from several repos at once (this one, peer-agent-scripts, a plugin), so they
# compose instead of one repo owning the directory.
#
# Idempotent: safe to run repeatedly. Refuses to clobber anything it doesn't own.
#
# Usage:
#   ./setup.sh            # link everything, report conflicts
#   ./setup.sh --force    # replace conflicting symlinks (never real files/dirs)

set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLAUDE_DIR="${CLAUDE_CONFIG_DIR:-$HOME/.claude}"

FORCE=0
for arg in "$@"; do
  case "$arg" in
    --force) FORCE=1 ;;
    *) echo "unknown argument: $arg" >&2; exit 2 ;;
  esac
done

linked=0 ok=0 replaced=0 conflicts=0

# ensure_dir <path> — the parent must be a real directory we can link into.
# A whole-directory symlink (the older setup style) is a hard stop: linking into
# it would silently write items into whatever repo it points at.
ensure_dir() {
  local dir="$1"
  if [[ -L "$dir" ]]; then
    cat >&2 <<EOF
error: $dir is a symlink to $(readlink "$dir")

Per-item linking needs a real directory. Migrate it:

  target=\$(readlink "$dir")
  rm "$dir" && mkdir -p "$dir"
  \$(dirname "\$target")/setup.sh   # re-link the other repo, per-item
  $REPO/setup.sh                   # then this one

EOF
    exit 1
  fi
  if [[ -e "$dir" && ! -d "$dir" ]]; then
    echo "error: $dir exists and is not a directory" >&2
    exit 1
  fi
  mkdir -p "$dir"
}

# ours <path> — true if an existing symlink points into this repo or dangles.
# A dangling link is already broken, so taking it over is a repair.
ours() {
  [[ -e "$1" ]] || return 0
  [[ "$(readlink "$1")" == "$REPO"/* ]]
}

# link <src> <dst>
link() {
  local src="$1" dst="$2" name="${2##*/}"
  if [[ -L "$dst" ]]; then
    local current
    current="$(readlink "$dst")"
    if [[ "$current" == "$src" ]]; then
      echo "    ok       $name"
      ok=$((ok + 1))
      return
    fi
    if (( FORCE )) || ours "$dst"; then
      rm "$dst"
      ln -s "$src" "$dst"
      echo "    replaced $name (was -> $current)"
      replaced=$((replaced + 1))
      return
    fi
    echo "    CONFLICT $name -> $current (use --force to take it over)" >&2
    conflicts=$((conflicts + 1))
    return
  fi
  if [[ -e "$dst" ]]; then
    echo "    CONFLICT $name exists and is not a symlink — move it aside by hand" >&2
    conflicts=$((conflicts + 1))
    return
  fi
  ln -s "$src" "$dst"
  echo "    linked   $name"
  linked=$((linked + 1))
}

echo "agent-scripts -> $CLAUDE_DIR"

ensure_dir "$CLAUDE_DIR/skills"
echo "  skills:"
shopt -s nullglob
for skill in "$REPO"/skills/*/; do
  skill="${skill%/}"
  name="${skill##*/}"
  if [[ ! -f "$skill/SKILL.md" ]]; then
    echo "    skipped  $name (no SKILL.md)" >&2
    continue
  fi
  link "$skill" "$CLAUDE_DIR/skills/$name"
done

ensure_dir "$CLAUDE_DIR/commands"
echo "  commands:"
for cmd in "$REPO"/commands/*.md; do
  link "$cmd" "$CLAUDE_DIR/commands/${cmd##*/}"
done
shopt -u nullglob

echo "done: $linked linked, $ok already ok, $replaced replaced, $conflicts conflicts"
(( conflicts == 0 )) || exit 1
