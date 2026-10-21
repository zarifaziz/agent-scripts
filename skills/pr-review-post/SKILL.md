---
name: pr-review-post
description: Draft Zarif's "PRs needing review :pray:" post for the PeerAI #eng Slack channel from his open peertechbio PRs, grouped by topic with raw links, and flag any PR that a reviewer can't act on yet. Use when asked for "my PRs that need review", "PRs needing review", "the eng channel post", "PRs to post on #eng", "bump my PRs", or "draft my review post".
---

# PR review post

Builds the post Zarif puts in #eng (`C0AADR7490S`) to ask for reviews. The output
is a draft for him to paste. Never post it: he sends it himself.

## 1. Collect the PRs

```bash
gh search prs --author @me --owner peertechbio --state open --limit 100 \
  --json repository,number,title,isDraft,url
```

Leave out **peer-agent-scripts** PRs: Zarif merges those himself after an agent
review. Leave out **drafts** too, and list them in the report.

For each remaining PR, get its state:

```bash
gh pr view N -R peertechbio/REPO --json \
  reviewDecision,mergeable,reviews,commits,body,baseRefName,headRefName
gh pr checks N -R peertechbio/REPO --json name,state,bucket
gh api graphql -f query='query{viewer{login}
  repository(owner:"peertechbio",name:"REPO"){pullRequest(number:N){
  reviewThreads(first:100){nodes{isResolved
    first:comments(first:1){nodes{author{login} body}}
    last:comments(last:1){nodes{author{login}}}}}}}}'
```

- **Human** means not `claude`, not a login ending in `[bot]`, and not you
  (`viewer.login`).
- **`mergeable: UNKNOWN`** means GitHub hasn't finished computing it. Ask again
  before you report it.
- **Stacked:** a `baseRefName` other than `main` is another PR's branch. Find it
  with `gh pr list -R peertechbio/REPO --head <baseRefName>`.

## 2. Sort each PR

Put each PR in the first group it matches:

| PR | Group |
|---|---|
| Approved: `reviewDecision` is `APPROVED`, or every human reviewer's latest review is `APPROVED` | Left out of the post. Offer to merge it |
| First line of the description matches `^\[PR made by the .* agent from the #ai-collective\]` | "AI collective PRs", last |
| Its reviewers' requests are done (below) | "Re-review", first |
| Anything else | A topic group |

Sort a stacked PR on its own first. If its parent is in the same group, list it
right after the parent.

**Requests are done** when, for each human reviewer's latest `CHANGES_REQUESTED`
or `COMMENTED` review (skip empty-body `COMMENTED` reviews: those are thread
replies, and rule 3 covers threads):
1. A commit after its `submittedAt` changes the code. Commits whose headline
   starts with `Merge` don't count.
2. Every request in the review body has been dealt with. Read the body and the
   new commits, and check each request yourself.
3. Every unresolved thread a human started ends with a comment from you.

With no such review, the PR is not a re-review. For a re-review PR, write what
was addressed after the URL, e.g. `(re-review: Box check done)`. An AI collective
PR that meets these 3 gets the same note.

**Flags** apply in any group, including approved PRs. List them under the post:
- conflicts (`mergeable: CONFLICTING`)
- any check with `bucket` `fail` (name it)
- `evals_gate` still `PENDING`: evals haven't run, so it can't merge

**Topics:** PRs that share a `PEE-` ticket go together first. Then PRs in the
same feature. Everything left goes under "General fixes". Name each topic in
plain words, never by repo.

## 3. Check the last post

Search Slack for Zarif's last post in #eng: `from:<@U0BJ007A03V> in:<#C0AADR7490S>`,
with keywords `PRs needing review` or `ready for review`. Read its thread.

- **Bump or new post.** If the post is under a week old and any of its PRs is
  still open, suggest a **reply in that thread** with the full list of PRs that go
  in the post, old and new. Remind the user to tick "Also send to #eng". Otherwise, suggest a
  new post.
- **Holds.** If someone in the thread asked to hold PRs (a release freeze, say),
  respect it and say so.

## 4. Write the post

Format, exactly:

```
PRs needing review :pray:

Re-review
https://github.com/peertechbio/peer-api-svc/pull/889  (re-review: Box check done)

<Topic in plain words>
https://github.com/peertechbio/peer-fe/pull/1064
https://github.com/peertechbio/peer-fe/pull/1072  (stacked on #1064)

AI collective PRs
https://github.com/peertechbio/peer-api-svc/pull/909
```

- Raw URLs, one per line. No tables, no markdown links, no titles after the URL.
- Mark a stacked PR with `(stacked on #N)`.
- No version labels like "v0", and never the words "seam" or "nits".

## 5. Report

Put the post in a code block. Below it, give one line each for:
- Approved PRs left out, ready to merge
- Flagged PRs, with the flag
- Drafts

Then offer the next step: merge the approved PRs, fix the flagged ones, or save the
post as a Slack draft (`slack_send_message_draft`, channel `C0AADR7490S`).
