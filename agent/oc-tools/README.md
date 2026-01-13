# oc-tools

Specialized sub-agent tools implemented with pi-sdk/opencode-sdk.

## Installation

```bash
# Install all tools
make install

# Install only TS scripts
make install-ts

# Install only Go binaries
make install-go

# Install a single tool
make install-tool NAME=autist
make install-tool NAME=db-oracle
make install-tool NAME=branch-namer
```

Symlinks tools to `~/.local/bin/`.

## Standalone TypeScript Scripts (Pi SDK)

Bun shebang scripts using `@mariozechner/pi-coding-agent` SDK.

| Script           | Model                       | Purpose                                                            |
| ---------------- | --------------------------- | ------------------------------------------------------------------ |
| `autist`         | gpt-5.2-codex (amplify-oai) | Oracle advisory agent - read-only analysis and planning            |
| `db-oracle`      | claude-opus-4-5 (amplify)   | Neo4j query oracle - researches codebase, tests queries on live DB |
| `session-hunter` | claude-haiku-4-5 (amplify)  | Find Amp sessions by code changes, file paths, or function names   |

### Usage

```bash
# autist - oracle advisory
autist "analyze this architecture"
echo "plan refactor" | autist
autist -s SESSION_ID "continue"

# db-oracle - neo4j queries
echo "find all courses" | db-oracle -db learning
db-oracle -db resources --dirs backend/app <<< "query prompt"

# session-hunter - find sessions
session-hunter "which session modified auth.dart?"
```

## Go Commands

| Command        | Purpose                                      |
| -------------- | -------------------------------------------- |
| `branch-namer` | Generate branch names from task descriptions |

## Structure

```
oc-tools/
├── standalone-ts/     # Bun TypeScript scripts (pi SDK)
│   ├── autist
│   ├── db-oracle
│   └── session-hunter
├── cmd/               # Go commands
│   └── branch-namer/
├── shared/            # Shared Go utilities
├── Makefile
└── README.md
```
