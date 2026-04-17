# agent-scripts

Zarif's personal toolbox for Claude Code: slash commands, skills, and supporting
scripts. Git-tracked so it survives machine resets and syncs across boxes.

## Layout

```
.
├── commands/    # Claude Code slash commands (one .md per command)
├── skills/      # Claude Code skills (one dir per skill, with SKILL.md)
├── agent/       # Standalone agent binaries and helpers
├── amp/         # Amp CLI sandbox + tooling
├── cypher-safe/ # Neo4j query REPL
├── psql-safe/   # Postgres/CockroachDB query REPL
├── exports/     # Cached schema/API exports used by skills
├── resources/   # Shared reference docs
└── setup.sh     # Idempotent symlink installer
```

## Install

Clone anywhere (this README assumes `~/Coding/agent-scripts`), then:

```bash
cd ~/Coding/agent-scripts
./setup.sh
```

`setup.sh` symlinks the repo into `~/.claude/`:

| Symlink | Target |
|---------|--------|
| `~/.claude/skills`   | `agent-scripts/skills`   |
| `~/.claude/commands` | `agent-scripts/commands` |

The script is idempotent and refuses to clobber existing non-matching paths —
re-running on an already-configured machine prints `ok:` for each link and exits
cleanly.

Restart Claude Code after first install so it re-scans both dirs.

## Adding a slash command

Drop a `.md` file into `commands/`. The filename (without `.md`) becomes the
slash name — e.g. `commands/foo.md` → `/foo`. First line is the summary shown
in the command picker; the rest is the instruction the agent follows.

## Adding a skill

Create `skills/<name>/SKILL.md` with frontmatter:

```markdown
---
name: <name>
description: <when-to-use trigger — keep it specific>
---

<body: the instructions the skill executes>
```

Single-file skills (no supporting scripts) can also live as
`skills/<name>.md` directly — see `skills/clean-up-storage.md`.

## Other subdirs

- **agent/oc-tools** — Go binaries (`big-brain`, `branch-namer`, `db-oracle`,
  `local-librarian`, `session-hunter`, `web-search`). Build with the
  per-tool `Makefile`.
- **amp/** — Amp CLI sandbox configs and permission handler. See
  `amp/README.md`.
- **cypher-safe/**, **psql-safe/** — Query REPLs used by the matching skills.

## Conventions

- No README/SUMMARY spam inside task dirs — keep the root doc authoritative.
- Conventional Commits for all commits.
- Never touch `agent-scripts/CLAUDE.md` (symlink to `AGENTS.md`); edit
  `AGENTS.md` directly.
