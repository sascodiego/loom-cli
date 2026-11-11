# Engram mirror contract

The mirror is a **search index**, never the authority. The tracked files under
`.router/` are the record of what happened; Engram holds a copy so a later
session can recall *why* without reading the repository.

## Who writes

The harness skill writes the mirror. The CLI never calls Engram: it has no
memory client and no credentials, and a classification must not depend on
agent-side context.

## What to write

```
mem_save(
  topic_key: "router/{slug}/decision",
  title:     "Routing decision: {slug} -> {method}",
  type:      "decision",
  scope:     "project",
  capture_prompt: false,
  content:   <the block below>
)
```

```markdown
**What**: Routed "{request_summary}" to {method} at tier {tier}.
**Why**: {one sentence drawn from the criteria that matched, plus the deciding
factor when the user overrode or a joint fallback fired}.
**Where**: branch {branch}; routing_id {routing_id}; record
.router/open/{branch_encoded}/{routing_id}.json
**Learned**: {anything the user clarified during pre-flight that the declaration
does not capture — or omit the line}.
```

`topic_key` is stable per slug, so a re-decision upserts instead of piling up
entries. Slug comes from the record; do not invent one.

## What never goes in

- The raw pre-flight prose in bulk. The record keeps `payload_hash`; the mirror
  keeps the summary and the reason, not the wall of text.
- Credentials, tokens, endpoint URLs.
- Anything you inferred but the user did not confirm.

## Multiple records on one branch

One `topic_key` per slug. When a record is superseded by a re-route, write to
**the same** `topic_key` and say so in `content` (`superseded by <new id>`): the
mirror tracks the current decision for that slug, not every historical record —
history lives in `.router/history/`.

## Failure handling

If `mem_save` fails, say so in one line and continue the routing. The mirror is
not part of the decision path: a missing mirror costs recall, never correctness.
Never retry in a loop and never block the work on it.

## Reverting

Routing state is plain tracked files; rolling back the router means reverting
`.router/`. Mirror records written before the revert survive as stale entries.
They are recovery hints only — when the record says one thing and memory says
another, the record wins and the stale memory is corrected.
