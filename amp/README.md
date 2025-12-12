# amp sandbox

Permission helper + kernel-level sandbox for Amp.

Based off of [amp-permission-docs](https://ampcode.com/manual#permissions).

## Install

```bash
git clone -b hman https://github.com/MathGaps/agent-scripts.git
cd agent-scripts/amp
make install
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

## Decision Pipeline

Exit codes: `0` = allow, `1` = ask (prompt), `2` = reject (hard block)

| Check | Result | Scope |
| ----- | ------ | ----- |
| Catastrophic pattern in command | BLOCK (exit 2) | rm -rf /, dd, fork bombs |
| Catastrophic pattern in script | BLOCK (exit 2) | Scripts calling bash/source with reject patterns |
| Always-ask pattern in script | ASK (prompt) | Scripts with brew, git reset, etc. |
| STDIN to interpreter | ASK (prompt) | Pipes to bash/python, heredocs |
| Safe command (no metacharacters) | ALLOW | readonly + always-allowed commands |
| Always-ask pattern | ASK (prompt) | brew install, git reset, terraform destroy |
| All paths in always-allowed dirs | ALLOW | /tmp, metarepo |
| No paths or all under PWD | ALLOW | Normal operations |
| Sensitive path | ASK (prompt) | ~/.ssh, ~/.aws, /etc |
| Outside PWD | ASK (prompt) | Anything else outside working dir |

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
| `--edit readonly` | `readonly-commands.txt` | Truly read-only commands (ls, cat, grep) |
| `--edit commands` | `always-allowed-commands.txt` | Side-effecty but trusted (ssh, ln) |
| `--edit sensitive` | `sensitive-paths.txt` | High-value paths (~/.ssh, ~/.aws, /etc) |
| `--edit reject` | `reject-patterns.txt` | Catastrophic patterns - always hard-blocked |
| `--edit paths` | `always-allowed-paths.txt` | Directories that never prompt |
| `--edit ask` | `always-ask-patterns.txt` | Patterns that always prompt (git reset, brew) |
| `--edit interpreters` | `interpreters.txt` | Script interpreters to scan |

## Config Philosophy

- `readonly-commands.txt`: Only commands with zero side effects (no writes, no network)
- `always-allowed-commands.txt`: Commands you trust but have side effects (ssh, ln)
- `reject-patterns.txt`: Catastrophic patterns - blocked in commands AND scripts
- `sensitive-paths.txt`: High-value paths only (~/.ssh, ~/.aws, not /usr or /var)

## Always-Allowed Paths

| Path | Purpose |
| ---- | ------- |
| `/tmp`, `/private/tmp` | System temp directory |
| `/var/folders`, `/private/var/folders` | macOS per-user cache/temp |
| `~/Coding/metarepo` | Main workspace |

## Always-Allowed Commands

| Command | Reason |
| ------- | ------ |
| `ssh` | Paths are remote, not local |
| `ln` | Symlinks are harmless |
| `psql-safe` | Sandboxed DB access |
| `cypher-safe` | Sandboxed DB access |

## Sensitive Paths

| Path | Risk |
| ---- | ---- |
| `~/.ssh`, `~/.gnupg` | Keys |
| `~/.aws`, `~/.kube` | Cloud credentials |
| `~/.config` | App configs |
| `~/Library/Keychains` | macOS keychain |
| `/etc`, `/System` | System config |

## Script Scanning

Scripts invoked via `bash script.sh`, `./script`, `source script` are scanned:
- Catastrophic patterns (reject-patterns.txt) → hard BLOCK
- Always-ask patterns → prompt user
- Recursive scanning up to depth 3
