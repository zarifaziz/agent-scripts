# amp sandbox

Permission helper + kernel-level sandbox for Amp.

Based off of [amp-permission-docs](https://ampcode.com/manual#permissions).

## Install

```bash
git clone -b hman https://github.com/MathGaps/agent-scripts.git
cd agent-scripts/amp
bash ./install.sh
```

If it fails, point your agent to this folder, give it the docs link @ https://ampcode.com/manual#permissions and ask it to install it for you :)

**What it does:**

- Symlinks `amp-permission-helper`, `amp-sandboxed`, `amp.sb` to `~/.local/bin/`
- Symlinks config files to `~/.config/amp-permissions/`
- Backs up existing `~/.config/amp/settings.json` to `.bak`, symlinks ours

## Usage

Regular `amp` uses permission-helper by default. For stricter kernel sandbox: `amp-sandboxed` (or `alias amps='amp-sandboxed'`).

| Command         | Mode              | Protection                          |
| --------------- | ----------------- | ----------------------------------- |
| `amp`           | permission helper | macOS popup prompts for outside-PWD |
| `amp-sandboxed` | sandbox-exec      | kernel blocks writes outside PWD    |

## Files

```
amp-permission-helper  # Amp delegate - prompts, logging, configurable
amp-sandboxed          # Wrapper - runs amp in sandbox-exec
amp.sb                 # macOS sandbox profile
config/                # Editable allow/block lists
```

## Commands

```bash
amp-permission-helper --test '{"cmd": "rm -rf /"}'  # test without running
amp-permission-helper --log                          # view decisions
amp-permission-helper --edit <config>                # edit config file
```

### Edit subcommands

| Subcommand | Config File | Purpose |
| ---------- | ----------- | ------- |
| `--edit readonly` | `readonly-commands.txt` | Commands auto-allowed (ls, cat, etc.) |
| `--edit sensitive` | `sensitive-paths.txt` | Extra sensitive paths |
| `--edit reject` | `reject-patterns.txt` | Hard-blocked patterns (rm -rf, dd, etc.) |
| `--edit paths` | `always-allowed-paths.txt` | Directories that never prompt |
| `--edit commands` | `always-allowed-commands.txt` | Commands that never prompt (ssh, ln) |
| `--edit ask` | `always-ask-patterns.txt` | Patterns that always prompt (git reset, brew) |
| `--edit interpreters` | `interpreters.txt` | Script interpreters to scan |

## Always-Allowed Paths

Directories that never prompt (edit via `amp-permission-helper --edit paths`):

| Path | Purpose |
| ---- | ------- |
| `/tmp`, `/private/tmp` | System temp directory |
| `/var/folders`, `/private/var/folders` | macOS per-user cache/temp |
| `~/Coding/metarepo` | Main workspace |

## Always-Allowed Commands

Commands that never prompt regardless of paths (edit via `amp-permission-helper --edit commands`):

| Command | Reason |
| ------- | ------ |
| `ssh` | Paths are remote, not local |
| `ln` | Symlinks are harmless |

## Special Cases

- **brew install/uninstall/upgrade**: Always prompts
- **Readonly commands**: Auto-allowed (see `amp-permission-helper --edit readonly`)
