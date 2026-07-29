---
name: e2e-test
description: Verify a feature works end-to-end on the PeerAI local stack by driving Chrome and cross-checking the browser UI against backend logs and Postgres. Derives a definition-of-done checklist from the conversation and confirms it before testing. Use when asked to "e2e test", "end-to-end test", "verify this works", "test it for real", or "prove the feature works".
---

# E2E Test

Prove a feature works by exercising it as a user would — clicking through Chrome against the local stack — then confirming the same claim in independent places: the browser UI, the backend log, and Postgres. A green unit test says the code does what its author thought. This says the product does what the user asked for.

Specific to the PeerAI metarepo at `/Users/zaz/Coding/metarepo`. The strongest verification applies to chat-agent work, which emits a structured event stream; see Step 6 for what is and is not provable elsewhere.

## When to Use

- After building a feature and before opening a PR, when "the tests pass" is not the same as "it works"
- When AI-generated code needs proving rather than trusting
- To reproduce a bug against the real stack before fixing it

Do NOT use for pure refactors with no behavioural change, or when the backend is not involved — a unit test is cheaper and repeatable.

## Required Inputs

| Input | Source |
|---|---|
| Feature under test | Read from the conversation. If ambiguous, ASK — never guess. |
| Definition of done | Derived in Step 1, confirmed by the user before any clicking. |
| Running local stack | Detected in Step 2. Offer to start what is missing; never start it silently. |

**Shell state does not persist between Bash calls.** Every snippet below is self-contained for that reason. Do not refactor a value into a variable and use it in a later call — write it to a file.

## Workflow

### Step 1 — Derive the definition of done, then confirm it

Read back over the conversation for what was built. Turn it into a checklist of observable outcomes — visible in the UI, the log, or the DB. Not implementation details.

Present it and get agreement before touching the browser:

```
Definition of done for <feature>:
- [ ] <observable outcome in the UI>
- [ ] <event, or absence of error, in the backend log>
- [ ] <row / column state in the DB>
Testing against: org=<name> study=<name> document=<name>
Proceed, or amend?
```

Use AskUserQuestion when the feature is unclear or more than one reading is plausible. A checklist the user corrected beats one you inferred.

### Step 2 — Preflight the stack

Infra first — Postgres, cognito-local, elasticmq and minio are all required:

```bash
docker ps --format '{{.Names}}\t{{.Status}}'   # expect postgres-db-v1, cognito-local, elasticmq, minio
```

Missing: `cd /Users/zaz/Coding/metarepo/tooling/peer-local-setup && docker-compose up -d`.

Then confirm the app on :3000 is the one you mean. A port that answers is not the app you want:

```bash
for pid in $(lsof -t -nP -iTCP:3000 -sTCP:LISTEN | sort -u); do
  ps -o pid,etime,command -p $pid | tail -1 | cut -c1-110
  echo "  cwd: $(lsof -a -p $pid -d cwd -Fn | grep ^n | cut -c2-)"
done
curl -s -m 5 http://localhost:8081/health
```

Three failure modes, all seen in practice:

1. **Another project owns the port.** Confirm a listener's `cwd` is the repo you mean. Two projects can coexist on :3000 — one on `*:3000`, one on `[::1]:3000` — and `localhost` resolves to whichever holds `::1`.
2. **The dev server is stale.** Compare `etime` against `git log -1 --date=relative`. A server older than the last pull reports phantom `Module not found` errors for files that exist on disk. Restart before believing any compile error.
3. **The backend is down.** An empty `/health` is the normal resting state; it is not auto-started.

Starting either — capture the pid, and poll rather than sleeping a magic number:

```bash
cd /Users/zaz/Coding/metarepo/backend/peer-api-svc
rm -f /tmp/peerai-be.log
(nohup poetry run start > /tmp/peerai-be.log 2>&1 & echo $! > /tmp/peerai-be.pid)
for i in $(seq 1 40); do curl -sf -m2 http://localhost:8081/health && break; sleep 2; done
```

Frontend is the same shape from `frontend/peer-fe` (`npm start > /tmp/peerai-fe.log`), but wait on the log, not the port: `grep -q "compiled successfully" /tmp/peerai-fe.log`. A returning `npm start` is not readiness.

**If the backend was already running, `/tmp/peerai-be.log` may be stale or from another branch.** Reading it would manufacture a green trace from an unrelated session. Assert the binding first:

```bash
lsof -p $(lsof -t -nP -iTCP:8081 -sTCP:LISTEN | head -1) 2>/dev/null | grep -c peerai-be.log
```

Zero means the log plane is unavailable — say so and rely on UI plus DB, or restart the backend with the redirect. Note :8081 legitimately shows **two** pids (uvicorn parent plus a `multiprocessing.spawn` child); that is not a port collision.

### Step 3 — Authenticate

The roadmap route is gated by `ProtectedComponent`; a cold Chrome profile lands on `/login` and everything after this fails.

Load the browser tools in one batched call first — they are deferred and error if called cold:

```
ToolSearch: select:mcp__claude-in-chrome__tabs_context_mcp,mcp__claude-in-chrome__tabs_create_mcp,mcp__claude-in-chrome__navigate,mcp__claude-in-chrome__computer,mcp__claude-in-chrome__find,mcp__claude-in-chrome__read_page,mcp__claude-in-chrome__javascript_tool,mcp__claude-in-chrome__read_console_messages
```

Then `tabs_context_mcp`, open a NEW tab (never hijack one the user is working in), and navigate to `http://localhost:3000`. If the studies list renders, the session is warm — skip ahead. If `/login` renders, sign in with the local cognito-local super admin: `superadmin@example.com` / `superadmin`. Confirm by screenshot that the studies list rendered before continuing.

### Step 4 — Choose a target with real data

The default org is often the empty one. Rank by actual content and pick the top row:

```bash
PGPASSWORD=postgres psql -w -h localhost -U postgres -d peerai -tA -F' | ' -c "
select o.display_name, d.display_name, d.org_id, d.study_id, d.id,
  (select count(*) from subcontents_t sc
   where sc.document_id=d.id and sc.deleted_at is null) as cnt
from documents_t d
join orgs_t o on o.id=d.org_id
join user_org_roles_t r on r.org_id=d.org_id and r.deleted_at is null
join users_t u on u.id=r.user_id and u.email='superadmin@example.com'
where d.deleted_at is null and d.status='DOCUMENT_GENERATED'
group by o.display_name, d.display_name, d.org_id, d.study_id, d.id
order by cnt desc limit 5;"
```

The `user_org_roles_t` join matters: without it the query can hand you a document the acting user cannot open, which renders an ErrorPage with no clue why.

Order is load-bearing. Navigate to `http://localhost:3000` and log in FIRST, then set the org (localStorage needs a loaded page on that origin — use `javascript_tool`):

```javascript
localStorage.setItem('selectedOrgId', '<org_id>')
```

Then navigate to the target. Append `?chatAgent=1` — the chat panel is disabled by default and the query param self-persists, so no second localStorage write is needed:

```
http://localhost:3000/studies/<study_id>/documents/<doc_id>/roadmap?chatAgent=1
```

### Step 5 — Mark the log position, then drive the browser

Persist the offset to a file; a shell variable will not survive to Step 6:

```bash
wc -l < /tmp/peerai-be.log | tr -d ' ' > /tmp/peerai-be.offset
```

`wc -l < file` pads with leading spaces on macOS — the `tr -d ' '` is required, not defensive.

Now drive it. Screenshot after each meaningful action; the screenshot is the evidence, not your recollection. Prefer `mcp__claude-in-chrome__find` over guessed coordinates. After an action that triggers backend work, wait and re-screenshot rather than asserting immediately.

Then check for errors the UI swallowed, via `mcp__claude-in-chrome__read_console_messages` with `onlyErrors: true`.

### Step 6 — Verify in the available planes

A claim is confirmed when the planes agree. Any one alone can lie: the UI can render an optimistic update that never persisted; the log can show a tool succeeding whose result the UI dropped; the DB can hold a row the user never saw.

**Backend log — chat-agent turns only.** Events are compact JSON with a `chat_agent_event` anchor. Grep the anchor, not the logger name; SQL echo puts ~11k lines of noise in the same stream. `grep -o` is what strips the `[ts] INFO:chat_agent.events:` prefix, so keep that form. `+1` on the offset avoids replaying the last pre-existing line:

```bash
tail -n +$(( $(cat /tmp/peerai-be.offset) + 1 )) /tmp/peerai-be.log \
  | grep -o '{"chat_agent_event".*}' \
  | jq -c '{event, iteration, tool_name, success, terminated_reason, finish_reason}'
```

A healthy read-only turn:

```
chat_turn_received -> chat_turn_dispatched -> agent_turn_start
agent_iteration_start / llm_call_start / llm_call_complete   (per iteration)
tool_call_start -> tool_call_complete   success=true
agent_turn_complete    terminated_reason="natural"
chat_turn_completed    terminated_reason="natural"
```

`terminated_reason` is the highest-value field. `natural` is success. `max_iterations`, `loop_trap`, `error`, `exception` and `stream_complete_missing` all still render plausible text in the UI — exactly the class of bug this skill exists to catch.

**Non-chat features have no event stream.** There is no offset-based trace and no per-feature log convention; `USE_JSON_LOGS=true` in the backend `.env` makes non-event logs whole-line JSON, which is the only way to filter them reliably. Be honest in the report about which planes actually ran — two planes stated plainly beats three planes implied.

**Database — always available.** Filter by the document under test; an unfiltered `order by created_at desc` will happily assert on a row from another tab or a background job:

```bash
PGPASSWORD=postgres psql -w -h localhost -U postgres -d peerai -P pager=off -c "
select role, status, text,
       token_usage->>'total_tokens' as tokens,
       metadata->>'model' as model,
       metadata->>'terminated_reason' as terminated,
       jsonb_path_query_array(tool_calls, '\$[*].name') as tools
from messages_t
where document_id='<doc_id>' and deleted_at is null
order by created_at desc limit 2;"
```

Select `text` in full — a truncated answer cannot be checked against the document's real content. The `\$` escape is required inside a double-quoted `-c`; unescaped it is `'$[*].name'`. Do NOT use `jsonb_array_length` on `tool_calls` — it errors when the column holds a scalar. The SQL column is `metadata` (`message_metadata` is the Python attribute only).

### Step 7 — Report

Walk the Step 1 checklist item by item: PASS or FAIL, plus the specific evidence — which screenshot, which log line, which DB row, and which plane proved it. Name any plane that was unavailable.

If an item failed, say so and stop. Do not fix and re-test in the same run unless asked. A fix folded into a verification run leaves neither trustworthy.

## Verification

The run is complete when:

1. Every Step 1 checklist item has an explicit PASS or FAIL with named evidence.
2. At least one assertion came from the log or DB, not the UI alone.
3. `terminated_reason` was checked for any chat-agent turn.
4. The browser console was checked for errors.
5. Log-plane provenance was confirmed, or its absence declared.
6. Anything started in Step 2 is either cleaned up or reported as still running.

## Cleanup

```bash
kill $(cat /tmp/peerai-be.pid) 2>/dev/null && rm -f /tmp/peerai-be.pid
```

Offer this; do not do it unprompted — the user may want the stack up. **Never kill a process you did not start.** Check its `cwd` first: a listener on a port you care about may belong to an entirely different project.

## Gotchas

- **A port that answers is not the app you want.** Check the listener's `cwd`.
- **`Module not found` for a file that exists means a stale dev server**, not a broken import. Check `etime` before debugging code.
- **Shell state dies between Bash calls.** No `export PGPASSWORD` in one call and `psql` in the next; no `OFFSET=` reused later. Inline it or write it to a file. Pass `-w` to psql so a missing password fails fast instead of hanging on a prompt for the full tool timeout.
- **`wc -l < file` pads with spaces** on macOS. Always `| tr -d ' '`.
- **Chat is disabled by default.** `REACT_APP_CHAT_AGENT_ENABLED` is usually absent from `peer-fe/.env`; use `?chatAgent=1`.
- **`peer-fe/.env` is build-time.** Webpack inlines it via DefinePlugin, so edits need a dev-server restart.
- **A plausible answer in the UI is not a pass.** Check it against the document's real content.

## Rules

- **Confirm the checklist before clicking.** The definition of done is the user's, not yours.
- **Ask before starting or killing anything.**
- **Report failures as failures.** A run that always passes is worthless.
- **Evidence, not assertion.** Every PASS cites a screenshot, log line, or DB row.
