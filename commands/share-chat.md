---
description: Copy a ready-to-paste prompt that points another Claude Code session at this chat
argument-hint: "What should the other session talk to this one about?"
allowed-tools: ListAgents, Bash
---

Put a paste-ready prompt on the user's clipboard so they can hand this chat to
another Claude Code session and hold a conversation with it.

## Steps

1. Call `ListAgents` and read the line naming THIS session. It gives a name and a
   short `[ref]`, and together those are the address other sessions use to reach
   it. A session UUID is not an address, so keep it out of the prompt.
2. Run `printf '%s | %s' "$CLAUDE_CODE_SESSION_ID" "$PWD"` to get the resume id
   and the working directory.
3. Build the prompt below with the real values filled in, and copy it using a
   quoted heredoc so the brackets and text pass through untouched:

   ```
   cat <<'PROMPT' | pbcopy
   ...the prompt...
   PROMPT
   ```

## The prompt to copy

```
Talk to my other Claude Code session, NAME [REF].

Run ListAgents first to confirm that address resolves, then reach it with
SendMessage. It is working in CWD.

Topic: TOPIC

Send one message, then show me its reply as soon as it lands, and keep relaying
both directions so I can hold a conversation with it.
```

For `TOPIC`, use the user's arguments verbatim. If they passed none, use `ask
what it is currently working on, then wait for the reply`.

## Then reply

Two short blocks and nothing else:

- One line saying the prompt is copied, naming the address.
- The resume id on its own labelled line, because `claude --resume` takes it
  while `SendMessage` does not.

Never list the peer sessions. No preamble, no summary, no closing offer.
