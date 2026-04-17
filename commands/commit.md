Commit the relevant changes you worked on during this session using semantic commit messages.

## Instructions

1. Run `git status` and `git diff` to understand all current changes.
2. Group related changes into logical commits. If there are many files changed across different concerns, commit them in separate chunks rather than one giant commit. For example:
   - Prompt template changes in one commit
   - Test changes in another
   - Config/dependency changes in another
   - Refactors separate from new features
3. Use **conventional commit** format for messages:
   - `feat:` for new features or capabilities
   - `fix:` for bug fixes
   - `refactor:` for code restructuring without behavior change
   - `test:` for adding or updating tests
   - `chore:` for config, dependencies, tooling changes
   - `docs:` for documentation changes
   - **NEVER use `!` for breaking changes** (e.g., `feat!:` is forbidden). Mention breaking changes in the commit body instead.
4. Keep commit messages concise (1-2 sentences) and focused on the **why**, not the **what**.
5. Do NOT commit files that contain secrets (`.env`, credentials, API keys).
6. Stage specific files for each commit — avoid `git add -A` or `git add .`.
7. After all commits, run `git status` to confirm a clean working tree.
