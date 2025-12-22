# Amp Permission System

Kernel-level sandbox + permission evaluator for Amp on macOS.

Based on [Amp permission docs](https://ampcode.com/manual#permissions) and [OpenAI Codex CLI](https://github.com/openai/codex) sandbox policies.

## Commands

| Command | Sandbox             | Protection                                                         |
| ------- | ------------------- | ------------------------------------------------------------------ |
| `amp`   | None                | Permission helper integration only (via settings.json)             |
| `amps`  | Lenient (blocklist) | Permission helper + cherry-picked sandbox protection on some paths |
| `samp`  | Strict (allowlist)  | Blocks ALL writes outside PWD                                      |

## What should I Use?

- If you mostly work inside a project folder, want simple and maximum security enforced at OS layer, use strict sandbox only with
  ```
  make install-strict
  ```
- If you want `--dangerouslyAllowAll` but block accidental footguns, approve brew/package installs etc, use permission system with lenient sandbox
  - You might need to tinker with the `permissions.yaml` config to get it to work for you if you choose this
  - Install with
  ```
  make install
  ```

See the [docs](docs/) section below for more info on implmentation of each

## Install

```bash
git clone -b hman https://github.com/MathGaps/agent-scripts.git
cd agent-scripts/amp

make install-strict   # Strict sandbox only (samp)

make install          # Permission system + lenient sandbox
```

In case of any issues, point your agent to this folder + the [amp docs](https://ampcode.com/manual#permissions) and ask it to install it for you.

## Strict Sandbox (`samp`)

For more info, see [docs/strict-sandbox.md](docs/strict-sandbox.md).

## Lenient Sandbox (`amps`)

For more info, see [docs/lenient-sandbox.md](docs/lenient-sandbox.md).

## Uninstalling

```bash
make uninstall
```

## Inspiration

- Brilliant amp docs/permission system and this famous fckup

<img width="465" height="720" alt="image" src="https://github.com/user-attachments/assets/157dfe08-b24e-48a6-9f4f-c9c04694354e" />
