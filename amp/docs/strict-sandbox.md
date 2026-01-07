# Strict Sandbox (`samp`)

Use this if you're mostly operating/editing files under a project folder + want simple security enforced at the OS layer.

```bash
samp [amp args...]
```

- Kernel blocks ALL\* writes outside working directory (pwd).
- Any child process/script run by amp is also subject to same sandbox.
- Based direcly off of [OpenAI Codex CLI](https://github.com/openai/codex) sandbox policies (Uses macos `sandbox-exec` under the hood).

\*: /dev/null, /tmp, and some amp specific dirs are writable for amp to function properly

| Writable              | Blocked              |
| --------------------- | -------------------- |
| PWD                   | Everything else      |
| TMPDIR, /tmp, cache   | ~/.ssh, ~/.aws, /etc |
| ~/.amp, ~/.config/amp |                      |
