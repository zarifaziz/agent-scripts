# amp sandbox

Kernel-level sandbox + permission helper for Amp on macOS.

Based off of [amp-permission-docs](https://ampcode.com/manual#permissions).

## Install

```bash
git clone -b hman https://github.com/MathGaps/agent-scripts.git
cd agent-scripts/amp


make install-sandbox  # go builds binary to ~/.local/bin/amps (sandbox only)
# OR
make install          # installs amp sandbox + amp-permission-helper + config files
```

If you encounter any problem installing it, point your agent to this folder, give it the docs link @ https://ampcode.com/manual#permissions and ask it to install it for you :)

## Usage

- Choose sandbox if you don't do much outside your pwd or if you're paranoid/want maximum security...
- Two for more freedom while having cherry-picked protections

| Command                    | Mode              | Protection                                 |
| -------------------------- | ----------------- | ------------------------------------------ |
| `amps`                     | sandbox-exec      | Kernel blocks writes outside PWD           |
| `amp` (with settings.json) | permission-helper | Allow everything except dangerous footguns |

## Tools

### Amps (Amp with Sandbox)

Kernel-level sandbox using macOS sandbox-exec (Apple Seatbelt). Based on [OpenAI Codex CLI](https://github.com/openai/codex) sandbox policies (forked and loosened a little).

```bash
amps [amp args...]
```

**Protection:**

- File writes blocked everywhere except:
  - PWD (folder that amp was lauched in) (with `.git` read-only)
  - TMPDIR, user cache/var folders
  - `~/.amp`, `~/.config/amp`, `~/.cache/amp`, `~/.local/share/amp` (needed for amp to function)
- Network unrestricted (amp needs internet)
- Child processes inherit sandbox (can't escape write restrictions)

### amp-permission-helper

Prompt-based delegate for Amp's permission system. Intercepts commands and prompts for approval when accessing sensitive paths (configurable).
Works with regular amp command (via amp setting.json configuration)
Amp calls our script `amp-permission-helper`for allow/block decisions.

```bash
amp-permission-helper --test '{"cmd": "rm -rf /"}'  # test without running
amp-permission-helper --log                          # view decisions
amp-permission-helper --edit <config>                # edit config file
```

**Decision pipeline (exit codes: 0=allow, 1=ask, 2=reject):**

| Check                         | Result | Examples                     |
| ----------------------------- | ------ | ---------------------------- |
| Catastrophic pattern          | BLOCK  | `rm -rf /`, fork bombs       |
| Always-ask pattern            | ASK    | `brew install`, `git reset`  |
| Sensitive path                | ASK    | `~/.ssh`, `~/.aws`           |
| Outside PWD                   | ASK    | Anything outside working dir |
| Safe command in allowed paths | ALLOW  | Normal operations            |

## Config

Config files in `~/.config/amp-permissions/`:

| File                          | Purpose                              |
| ----------------------------- | ------------------------------------ |
| `reject-patterns.txt`         | Catastrophic patterns - hard blocked |
| `always-ask-patterns.txt`     | Patterns that always prompt          |
| `sensitive-paths.txt`         | High-value paths (~/.ssh, ~/.aws)    |
| `always-allowed-paths.txt`    | Directories that never prompt        |
| `always-allowed-commands.txt` | Trusted commands (ssh, ln)           |
| `readonly-commands.txt`       | Zero side-effect commands            |

## Uninstall

```bash
make uninstall
```
