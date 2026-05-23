---
name: make-pr-descriptions
description: Produces a high-quality PR title + description for the current branch, ready to paste into GitHub or run via `gh pr create`. Use when asked to "write a PR description", "draft a PR", "make a PR description", or "generate PR title and body".
---

# Make PR Descriptions

Drafts a polished PR title and description for the current branch. Presents the draft to the user for review before creating or editing anything on GitHub.

## Title Rules

- Format: `type(scope): short description` or `type: short description`
- Valid types: `feat`, `fix`, `refactor`, `test`, `chore`, `docs`, `ci`, `style`, `perf`, `build`
- **Never use `!`** — `feat!:` is forbidden
- Max ~70 chars, no trailing period

## Description Format

Write the sections in this exact order:

1. **Opening paragraph** — 2–4 sentences of concise prose covering *what* changed and *why*. The reviewer should understand the motivation without reading the diff.
2. **Highlights** — tight bullet list of the most important changes; written so a reviewer can skim and know exactly where to focus.
3. **Flow diagram** — ASCII box/arrow diagram (or fenced code block) showing the key flow, data path, or component interaction introduced or changed by this PR.
4. **Testing** — brief checklist of what was tested / how to verify.
5. Footer (always include):
   ```
   🤖 Generated with [Claude Code](https://claude.ai/code)
   ```

### Style rules

- No filler openers — never start with "This PR introduces…" or "In this PR…"
- ASCII diagrams use box/arrow style, not just indented bullets
- Before/after code snippets are encouraged where helpful
- Keep the whole description under ~40 lines so it fits on screen without scrolling

## Workflow

### Step 1 — Gather context

```bash
# See all commits on this branch vs main
git log --oneline origin/main..HEAD

# Understand the scope of changes
gh pr diff --stat 2>/dev/null || git diff origin/main --stat

# Read the full diff to understand the *why*
gh pr diff 2>/dev/null || git diff origin/main
```

### Step 2 — Draft

Write the title and description following the format above. Present the full draft to the user for review — **do not run `gh pr create` or `gh pr edit` yet**.

### Step 3 — Iterate

If the user requests changes, revise and re-present the draft. Repeat until the user explicitly says to create or update the PR.

### Step 4 — Create or update

Only after explicit user approval:

```bash
# Create new PR
gh pr create --title "<title>" --body "$(cat <<'EOF'
<description body>
EOF
)"

# Or edit an existing PR
gh pr edit --title "<title>" --body "$(cat <<'EOF'
<description body>
EOF
)"
```

## Example Output

**Title:**
```
feat(auth): add OAuth2 PKCE flow for mobile clients
```

**Description:**
```
Mobile clients require PKCE to perform the OAuth2 authorization code flow
without a client secret. This adds PKCE support to the auth service and
updates the mobile SDK to pass the code_verifier on token exchange.

- `AuthService.authorize()` now generates and stores a `code_verifier`
- Token exchange endpoint validates `code_verifier` against stored challenge
- `MobileAuthClient` updated to pass `code_verifier` in exchange request
- Legacy desktop flow unchanged — PKCE is opt-in via `usePkce: true`

┌─────────────┐   /authorize?code_challenge=…   ┌─────────────┐
│ Mobile App  │ ─────────────────────────────► │ Auth Server │
│             │                                 │             │
│             │   /token + code_verifier=…      │             │
│             │ ─────────────────────────────► │  validates  │
│             │ ◄───────────────────────────── │  challenge  │
│             │        access_token             └─────────────┘
└─────────────┘

- [ ] Tested PKCE flow end-to-end with the iOS simulator
- [ ] Confirmed legacy desktop flow still works (no regression)
- [ ] Unit tests added for `generateCodeVerifier` and `verifyChallenge`

🤖 Generated with [Claude Code](https://claude.ai/code)
```

## Rules

- **Never create or edit the PR without explicit user approval** — always present the draft first.
- **One draft, then wait** — don't loop re-generating unless the user asks for a revision.
- **Read the diff, not just the commit messages** — commit messages are often terse; the diff reveals the real motivation.
- **Diagrams are mandatory** — every description must include a flow diagram. If no data flow exists, diagram the component interaction or before/after state.
