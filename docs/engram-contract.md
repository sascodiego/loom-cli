# Unified Engram persistence contract

This contract is identical for every extracted method package — `rsc-ftd`, `rsc-asdd`,
`egm` — and the copies in each package's `system/skills/_shared/engram-contract.md` point
here as canonical, carrying only that package's naming table. `gentle-sdd` is Engram-native
and already follows it (`gentle-sdd/system/skills/_shared/engram-convention.md`); it is
listed for reference and is not edited by this contract.

The file artifacts stay the source of truth: the `02-DOCS/` wikis, the feature documents,
the design documents, the decisions logs. Engram is a mirror — a recovery and search layer
that survives compaction and session loss, never an authority and never an enforcement
record. Source of this contract: `docs/analysis/engram-unification-analysis.md` (§5–§7),
spec S15 of the method-router feature (`odd/tasks/method-router-v1.md`).

## The five rules

The divergence policy is gentle's own hybrid contract, adopted verbatim — proven in
production, with no detection machinery added:

1. **Reads go to resolved per-store locators** — never a freshest-copy fallback, never silently substituting another store's copy. Store mode and locators are declared by the orchestrator/router, not inferred by the agent.
2. **Dual write with read-back** — attempt both declared writes following each store's convention, and read back each successful write.
3. **Non-atomicity declared honestly** — hybrid writes are not atomic: preserve successful writes, report any failed mirror as partial, and name the outstanding locator. Never claim synchronization; never roll back valid progress.
4. **Conflict: preserve both, escalate decision-scoped** — a genuine content conflict is kept in both stores; it is resolved (by the human) only when it affects the next decision.
5. **No second authority** — snapshots (progress mirrors, state observations) are recovery hints, never authority; stale snapshots do not replace actual progress. Reporting ranks landed evidence above snapshots (archive rule), but nothing auto-overwrites anything.

## Naming tables

One table per method package. `title` equals `topic_key`, deterministic, so recovery works
by exact match (gentle's naming rule, applied everywhere).

### rsc-ftd — feature-document lane

| topic_key (= title) | type | when to write |
| --- | --- | --- |
| `ftd/{slug}/feature-doc` | architecture | One per feature. Upsert after **every** feature-doc mutation (intent, scope, checklist, evidence, next step). |

### rsc-asdd — spec chain

| topic_key (= title) | type | when to write |
| --- | --- | --- |
| `asdd/{change}/constitution` | architecture | Constitution drafted or amended (base stage). |
| `asdd/{change}/spec` | architecture | Spec written or revised (`specify`). |
| `asdd/{change}/plan` | architecture | Plan written or revised (`plan`). |
| `asdd/{change}/tasks` | architecture | Task table appended or updated (`tasks`). |
| `asdd/{change}/progress` | architecture | Progress entry appended (`implement`). |
| `asdd/{change}/verification` | architecture | Verification record written (closing). |
| `asdd/{change}/review` | architecture | Closing review record / archive state written (closing). |
| `asdd/{change}/decisions/{id}` | decision | A decision logged in the decisions trail. |

### egm — review loop

| topic_key (= title) | type | when to write |
| --- | --- | --- |
| `egm/{doc-slug}/draft` | architecture | Design draft written or a section updated (`egm-draft`). |
| `egm/{doc-slug}/review/{round}` | architecture | A review round's findings consolidated (`egm-review-loop`). |
| `egm/{doc-slug}/resolution` | architecture | Resolution applied and the report marked resolved (`egm-resolve`). |

### gentle-sdd — reference only

Unchanged: `sdd/{change-name}/{artifact-type}` per its existing convention
(`gentle-sdd/system/skills/_shared/engram-convention.md`). Listed for reference only;
nothing in this contract edits it.

## Capture and lifecycle

- `capture_prompt: false` for every automated artifact write — these are pipeline outputs,
  not human/proactive saves. If an older schema rejects or does not expose the field, omit
  it rather than failing.
- `scope: project` for all method state.
- Upsert by writing again with the same `topic_key` (same topic_key + project + scope
  overwrites — that is the design; Engram is working memory, not an audit trail).
- `mem_update` when the observation id is already known; `mem_save` with the same
  `topic_key` otherwise.

## Exclusions

- The rsc ledger (`.rsc/`) is **never** mirrored to Engram. It is CLI-owned enforcement
  state; a reviewer verifying a gate reads the JSONL file, never an observation.
- Goldfish reviewer subagents **never call `mem_*`** — the document reaches them via
  prompt or locator only. Their fresh eyes are the method; prior-session memory would
  defeat it.
- Secrets and credentials never enter Engram.

## Failure handling

A failed mirror is reported honestly as a **partial write**, naming the outstanding
locator. It is never retried in a loop and never allowed to block the work: the file write
still lands (rule 3), the partial state is named, and the next decision that depends on
the mirror goes to the human (rule 4). Never claim the two stores are synchronized.
