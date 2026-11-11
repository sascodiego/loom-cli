# Engram persistence contract — rsc-ftd copy

> **Canonical contract:** `docs/engram-contract.md` at the repository root
> (`../../../../docs/engram-contract.md` from this file). This copy carries only the
> rsc-ftd naming table and a condensed statement of the five rules; the master is
> authoritative.

## The five rules (condensed)

1. **Reads go to resolved per-store locators** — never a freshest-copy fallback; store mode and locators are declared, never inferred.
2. **Dual write with read-back** — write both stores per each store's convention and read each successful write back.
3. **Non-atomicity declared honestly** — preserve what landed; report a failed mirror as a partial write naming the outstanding locator.
4. **Conflict: preserve both, escalate decision-scoped** — the human resolves it only when it affects the next decision.
5. **No second authority** — mirrors are recovery hints, never authority; landed evidence outranks snapshots.

## rsc-ftd naming table

| topic_key (= title) | type | when to write |
| --- | --- | --- |
| `ftd/{slug}/feature-doc` | architecture | One per feature. Upsert after **every** feature-doc mutation (intent, scope, checklist, evidence, next step). |
