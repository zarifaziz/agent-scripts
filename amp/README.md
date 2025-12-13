# amp sandbox

Kernel-level sandbox + permission helper for Amp on macOS.

Based on [Amp permission docs](https://ampcode.com/manual#permissions) and [OpenAI Codex CLI](https://github.com/openai/codex) sandbox policies.

## Install

```bash
git clone -b hman https://github.com/hemanta212/agent-scripts.git
cd agent-scripts/amp

make install-strict   # Installs 'samp' -- strict sandbox

make install          # Lenient sandbox + permission helper + configs
```

If stuck, point your agent to this folder + the [amp docs](https://ampcode.com/manual#permissions) and ask it to install.

## Commands

| Command | Sandbox                            | Protection                                 |
| ------- | ---------------------------------- | ------------------------------------------ |
| `samp`  | Strict, security-first (allowlist) | Blocks ALL writes outside PWD              |
| `amps`  | Lenient, security-last (blocklist) | Only blocks sensitive paths                |
| `amp`   | None                               | Permission helper only (via settings.json) |

## 1. Lenient Sandbox (`amps`)

Blocklist sandbox + Go shell parser + Bash policy. Permissive, minimal prompts.

```bash
amps [amp args...]
```

**Architecture:**

```
Command → Go Parser (AST) → Bash Evaluator → Prompt Handler
              ↓                    ↓               ↓
         Detect patterns     Expand vars      osascript/TTY
         Check configs       Check paths
```

**Layers:**

| Layer | Component      | Protection                                   |
| ----- | -------------- | -------------------------------------------- |
| 1     | Kernel sandbox | Hard blocks writes to sensitive paths        |
| 2     | Go parser      | Detects dangerous patterns, pipes, heredocs  |
| 3     | Bash eval      | Variable expansion, path resolution, prompts |

**Decision flow:**

| Check                    | Result | Examples                         |
| ------------------------ | ------ | -------------------------------- |
| Risky pattern            | BLOCK  | `rm -rf ~`, fork bombs           |
| Read-only command        | ALLOW  | `cat`, `grep`, `ls` piped        |
| Always-ask pattern       | ASK    | `brew install`, `kubectl delete` |
| Sensitive path           | ASK    | `~/.ssh/*`, `~/.aws/*`           |
| Single file outside PWD  | ALLOW  | `rm /tmp/file.txt`               |
| Directory op outside PWD | ASK    | `rm -r /usr/local/foo`           |

## 2. Strict Sandbox (`samp`)

Kernel blocks ALL writes outside PWD. Simple, paranoid.

```bash
samp [amp args...]
```

| Writable              | Blocked                    |
| --------------------- | -------------------------- |
| PWD (except .git)     | Everything else            |
| TMPDIR, cache dirs    | ~/.ssh, ~/.aws, /etc, etc. |
| ~/.amp, ~/.config/amp |                            |

Best for: Maximum security, single project work.

## Config Files

All in `~/.config/amp-permissions/`:

### sandbox-blocked-paths.txt

Kernel-level write blocks (lenient sandbox). No prompt, just denied.

```
$HOME/.ssh
$HOME/.aws
$HOME/.config/gcloud
/etc
```

### reject-patterns.txt

Command patterns - hard blocked by parser.

```
rm -rf ~
rm -rf $HOME
dd if=/dev/zero
:(){:|:&};:
```

### always-ask-patterns.txt

Regex patterns that always prompt.

```
^brew (install|uninstall|upgrade)
^git reset
kubectl delete
terraform destroy
```

### write-commands.txt

Commands needing path checks. Everything else is read-only (auto-allowed).

```
rm
rmdir
dd
brew
pip
```

### sensitive-paths.txt

Paths that prompt before access.

```
$HOME/.ssh
$HOME/.gnupg
/etc
```

### interpreters.txt

Pipe-to-interpreter triggers prompt.

```
bash
python
node
```

### always-allowed-paths.txt

Skip all checks.

```
/tmp
/var/folders
```

## Testing

```bash
amp-permission-eval --test '{"cmd": "rm -rf ~"}'
amp-permission-eval --parse 'cat ~/.ssh/id_rsa | bash'
amp-permission-eval --log
```

## Custom Prompt Handler

```bash
export AMP_PERMISSION_PROMPT_HANDLER=/path/to/handler
```

Schema:

```json
{
  "title": "CONFIRM | SENSITIVE | OUTSIDE PWD | HEREDOC | PIPE TO <interp>",
  "message": "Human readable prompt message",
  "context": {
    "cmd": "full command string",
    "pwd": "working directory",
    "tool": "Bash | edit_file | create_file",
    "tmux": "session:window(pane) | no-tmux"
  }
}
```

Example:

```json
{
  "title": "CONFIRM",
  "message": "Pattern: ^brew (install|uninstall)\n\nCmd: brew install neovim",
  "context": {
    "cmd": "brew install neovim",
    "pwd": "/Users/mac/projects/myapp",
    "tool": "Bash",
    "tmux": "dev:editor(1)"
  }
}
```

Exit 0 = allow, non-zero = deny.

## Uninstall

```bash
make uninstall
```
