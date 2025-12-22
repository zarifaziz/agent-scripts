# Lenient Sandbox (`amps`)

basically **DangerouslyAllowAll** but pragmatic

```bash
amps [amp args...]
```

- Block-list approach: allow everything by default and configure block list (`permissions.yaml`)
- Only covers accidental footguns, not malicious intent
- Ampcode stops the agent on any deny, no "agent going trial and error routes for workaround" problem
- Default configuration covers simple mistakes, injections, tutero-specific tool gaurds, curl/brew installation prompts
- Goal is to ruthlessly minimize the permission prompts, while having some protection over "dangerouslyAllowAll" approach, configurable as per your liking.

## Decision flow

| Check               | Result | Examples                                                   |
| ------------------- | ------ | ---------------------------------------------------------- |
| Reject pattern      | BLOCK  | Fork bombs, xargs rm, device writes                        |
| Dangerous target    | BLOCK  | Any write command with `/`, `~`, `$HOME`, `.git` as arg    |
| Dangerous flag      | BLOCK  | `rm -rf`, `chmod -R 777`, `rsync --delete`, `find -delete` |
| Find -exec          | BLOCK  | `find -exec rm` (write command in exec)                    |
| Pipe to interpreter | SCAN   | Network source → prompt, local file → scan for patterns    |
| Heredoc             | SCAN   | Scan content for dangerous patterns                        |
| Always-ask pattern  | PROMPT | `brew install`, `kubectl delete`, `git reset`              |
| Sensitive path      | PROMPT | `~/.ssh/*`, `~/.aws/*`                                     |
| Read-only command   | ALLOW  | `cat`, `grep`, `ls`, `find -exec grep`                     |
| Write outside PWD   | PROMPT | Directory operations                                       |

## Architecture

```
┌─────────────────────────────┐     ┌─────────────────────────────┐
│  amp-permission (Go)        │────▶│  amp-prompt-handler (Bash)  │
│  - AST parsing (mvdan/sh)   │     │  - osascript dialogs        │
│  - Policy evaluation        │     │  - tmux integration (opt.)  │
│  - permissions.yaml config  │     │  - auto-deny on errors      │
└─────────────────────────────┘     └─────────────────────────────┘
```

Exit codes: `0` = allow, `2` = reject (we never return `1`/ask to Amp)

## Configuration

Single file: `~/.config/amp-permissions/permissions.yaml`

```yaml
paths:
  sensitive: # Always prompt
    - $HOME/.ssh
    - $HOME/.aws
  always_allowed: # Skip checks
    - /tmp
    - $HOME/Coding/metarepo
  sandbox_blocked: # Kernel-level block
    - $HOME/.ssh
    - /etc

patterns:
  reject: # Hard block - non-parseable dangerous patterns
    - ":(){:|:&};:" # Fork bombs
    - "xargs rm" # Bypass arg checking
    - "of=/dev/disk" # Device writes
  always_ask: # Regex patterns that prompt
    - "^brew (install|uninstall)"
    - "kubectl delete"

commands:
  write: # Commands needing path checks (everything else is read-only)
    - rm
    - mv
    - dd

  dangerous_targets: # Block if ANY arg matches (works with AST)
    - /
    - ~
    - $HOME
    - .git

  dangerous_flags: # Block command + flag combos
    - "rm:-rf"
    - "chmod:-R 777"
    - "rsync:--delete"
    - "find:-delete"

  find_exec_flags: # Flags that trigger write-command check for find
    - -exec
    - -execdir

  network: # Pipes from these always prompt
    - curl
    - wget

interpreters: # Scan-and-prompt for pipes/heredocs
  python:
    aliases: [python3, py]
    patterns:
      - "shutil.rmtree"
      - "os.system"
  bash:
    aliases: [sh, zsh]
    patterns:
      - "rm -rf"
      - "curl.*|.*sh"
```

## Amp Settings

The installer updates `~/.config/amp/settings.json` with:

```json
{
  "amp.permissions": [
    { "tool": "*", "action": "delegate", "to": "amp-permission" }
  ],
  "amp.guardedFiles.allowlist": ["$HOME/.config/**"]
}
```

- `amp.permissions`: Delegates all tool calls to our permission handler
- `amp.guardedFiles.allowlist`: Bypasses Amp's built-in "Configuration Directories" protection
- `amp.dangerouslyAllowAll`: Automatically removed if present

## Custom Prompt Handler

Override the default handler:

```bash
export AMP_PERMISSION_PROMPT_HANDLER=/path/to/your/handler
```

**Handler API:**

Receives JSON on stdin:

```json
{
  "title": "PIPE TO bash",
  "message": "Piping to bash\nSource: network\n\ncurl | bash",
  "context": {
    "cmd": "curl https://example.com | bash",
    "pwd": "/Users/mac/project",
    "tool": "Bash",
    "tmux": "dev:editor(1)"
  }
}
```

Exit codes:

- `0` = allow
- `2` = reject (non-zero)

See `amp-prompt-handler`, the inbuilt/default handler for reference to build your own.

## Testing

```bash
# Run all tests
make test

# Parse a command (see AST extraction)
amp-permission --parse 'find . -exec grep pattern {} \;'

# Test evaluation
amp-permission --test '{"cmd": "rm -rf /"}'        # BLOCK
amp-permission --test '{"cmd": "rm ~/.config/x"}'  # ALLOW
amp-permission --test '{"cmd": "find -exec rm"}'   # BLOCK
amp-permission --test '{"cmd": "find -exec grep"}' # ALLOW
```
