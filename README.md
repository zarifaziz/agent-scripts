# Using Agents and Skills

- Symlink or copy the skills folder to your ~/.config/opencode/skills folder

- Symlink or copy the agent folder to your ~/.config/opencode/agent folder
- If agent/ folder contains .sh scripts, you have to make it available in the PATH, easiest way is to symlink it to $HOME/.local/bin/

```
ln -s ~/.config/opencode/agent/agent.sh ~/.local/bin/agent

## Actual example
ln -s ~/.config/opencode/agent/web-search.sh ~/.local/bin/web-search
```

- Now test using web-search in your cli directly (internally it calls `opencode run --agent web-search` which internally looks up for web-search.md inside opencode agent folder)
- The sh script wraps opencode agent invocation, so other agents can use it predictively and token-efficently
