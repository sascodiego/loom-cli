# Engram persistence contract — egm copy

> **Canonical contract:** `docs/engram-contract.md` at the repository root
> (`../../../../docs/engram-contract.md` from this file). This copy carries only the
> egm naming table and a condensed statement of the five rules; the master is
> authoritative.

## The five rules (condensed)

1. **Reads go to resolved per-store locators** — never a freshest-copy fallback; store mode and locators are declared, never inferred.
2. **Dual write with read-back** — write both stores per each store's convention and read each successful write back.
3. **Non-atomicity declared honestly** — preserve what landed; report a failed mirror as a partial write naming the outstanding locator.
4. **Conflict: preserve both, escalate decision-scoped** — the human resolves it only when it affects the next decision.
5. **No second authority** — mirrors are recovery hints, never authority; landed evidence outranks snapshots.

## egm naming table

| topic_key (= title) | type | when to write |
| --- | --- | --- |
| `egm/{doc-slug}/draft` | architecture | Design draft written or a section updated (`egm-draft`). |
| `egm/{doc-slug}/review/{round}` | architecture | A review round's findings consolidated (`egm-review-loop`). |
| `egm/{doc-slug}/resolution` | architecture | Resolution applied and the report marked resolved (`egm-resolve`). |
