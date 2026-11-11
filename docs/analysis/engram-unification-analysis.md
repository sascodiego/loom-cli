# Unifying the three methods on Engram — analysis (proposal only)

Status: design analysis, no implementation. Covers: what memory each extracted method
uses, how each would adapt to Engram, pros/cons of Engram vs the original system, a
unified schema, and how the Cloudflare Clef decision model fits as the method router.

Source trees: `genlte-sdd/` (Gentle AI SDD v3.7.0), `rsc-ftd/` + `rsc-asdd/` + `rsc-tools/`
(RSC harness v4.1.0), `egm/` (Elephant-Goldfish Model v0.1.1).

---

## 1. What memory each method uses today

### Gentle SDD — Engram-native already
- Contract: `skills/_shared/engram-convention.md` + `persistence-contract.md` + `engram/protocol.md`.
- Deterministic naming: `title = topic_key = sdd/{change-name}/{artifact-type}`,
  `type: architecture`, `scope: project`, `capture_prompt: false`. Upsert semantics —
  overwrite, no iteration history.
- Artifact types: explore, proposal, spec, design, tasks, apply-progress, verify-report,
  archive-report, research, optional `state` recovery hint.
- Mode resolution (`engram | openspec | hybrid | none`): orchestrator asks once per
  session; default engram if available. `hybrid` = files as team source of truth +
  Engram as recovery working memory.
- Recovery: `mem_search` by topic_key → `mem_get_observation`; `mem_review` lifecycle
  (`active` / `needs_review` = stale, never trusted as fact).

### RSC FTD — the feature document IS the memory
- Feature doc per feature in `02-DOCS/wiki/ftd/` (intent, scope, checklist, evidence,
  next step). Tasks checked off only against observed proof recorded in the doc.
- Plus: `02-DOCS/wiki/harness/user-profile.md` (tone/technical level), worklogs in
  `02-DOCS/raw/worklog/`, `.rsc/` JSON state (`state.js`), and RSC session memory
  (`rsc memory on|off|save|resume|learn|metrics`, wired per runtime via hooks;
  `.rsc/memory`).
- Characteristics: file-based, git-versioned, team-shareable, human-readable, evidence
  inline. No memory tool required.

### RSC ASDD — a file wiki plus a machine-enforced ledger
- `02-DOCS/wiki/sdd/`: `specs/`, `plans/`, `decisions/` (the decision queue),
  `progress/`, `config`, `constitution`, `sessions/`, `index.md` (knowledge map).
- `.rsc/review-ledger.jsonl` — enforcement state (max 3 review rounds, no repeated
  findings, tier-2 panel records). `.rsc/memory` + worklogs for session recovery.
- The CLI (`rsc risk`, `rsc spec gate`, `rsc review ledger`, ship/branch guards) reads
  and writes this state; tier-2 branches legally leave the machine only with a
  gate-passing spec + recorded panel. The memory is load-bearing for enforcement.

### EGM — deliberately stateless
- No persistence layer at all. State = the documents produced (draft, findings,
  resolved decisions) + live conversation context. Between sessions, only the
  documents survive. Standards are static files in `resources/`.

---

## 2. Unified Engram schema (one memory model for all methods)

Namespaces prevent collisions; every method keeps its file artifacts as the authority
where it already had them (see §3), Engram carries working/recovery memory.

```
Topic-key namespaces
  sdd/{change}/{artifact}            gentle SDD (already defined upstream — unchanged)
  ftd/{feature-slug}/state           FTD feature-doc mirror (checklist progress, next step)
  ftd/{feature-slug}/evidence        evidence pointers + proof status
  asdd/{change}/{artifact}           ASDD phase artifacts mirror (constitution, spec, plan,
                                     tasks, progress)
  asdd/{change}/decision/{id}        decision-queue entry       type: decision
  asdd/{change}/review-state         read-only ledger SUMMARY (mirror, never authority)
  egm/{doc-slug}/draft               EGM draft spec/design
  egm/{doc-slug}/review/{round}      review findings + resolution status
  router/{request-slug}/decision     method-router output (Clef): chosen method,
                                     probabilities, state digest   type: decision
  user-profile                       tone/level preferences       scope: personal
Shared conventions
  scope: project (method state) | personal (user profile) | global (cross-project prefs)
  capture_prompt: false; upsert via stable topic_key; mem_review lifecycle respected
  (needs_review = stale context, never a fact)
```

Clef decision model (Cloudflare Workers AI, `@cf/cloudflare/clef`): state = project
digest (`mem_context` + repo manifest of the four methods) ; questions = typed schema:
`{ method: [none, ftd, asdd, gentle-sdd, egm-review-loop], risk_tier: [0,1,2],
store_mode: [engram, hybrid, openspec] }` → calibrated probabilities in one forward
pass. The router logs its decision as a `router/…` observation for audit. Clef routes;
it never bypasses human gates (tier-2 approval stays human per ASDD rules).

---

## 3. Per-method adaptation design

### Gentle SDD — zero adaptation
Already Engram-native. Pin `store_mode` to `engram` or `hybrid` in the unified tool and
reuse its contract verbatim. Its mode table becomes the reference model for the others.

### RSC FTD — dual-write adapter ("doc authority, Engram mirror")
- Keep the feature doc in `02-DOCS/wiki/ftd/` as the single source of truth (git,
  evidence, team).
- Add an adapter section to `SKILL.md`: on every checklist mutation, upsert
  `ftd/{slug}/state` (progress + next step) and append evidence pointers; on session
  start, `mem_search("ftd/{slug}")` instead of re-reading the repo.
- rsc session-memory (`rsc memory`) is replaced by the Engram path; worklogs stay as
  files (they are git evidence).
- Effort: small — one convention file (like gentle's `engram-convention.md`) + edits to
  one SKILL.md.

### RSC ASDD — hybrid with strict authority separation
- `02-DOCS/wiki/sdd/` and `.rsc/review-ledger.jsonl` stay authoritative: tier-2 gates
  and audits require git-tracked artifacts and a machine-checked ledger.
- Engram mirrors phase artifacts for recovery (`asdd/{change}/{artifact}`) and indexes
  decision-queue entries (`type: decision`) so past decisions are searchable when
  writing new proposals.
- The ledger mirror is explicitly read-only in the convention: a reviewer verifying a
  gate reads the JSONL file, never the observation.
- Effort: medium — one convention file + persistence sections across ~6 phase
  SKILL.mds (specify, plan, tasks, implement, review, ship).

### EGM — gains a memory it never had (with a contamination guard)
- Elephant side persists: `egm/{doc}/draft`, `egm/{doc}/review/{round}` findings +
  resolution status, so a review loop survives compaction/session restart — today it
  just dies.
- **Goldfish reviewers must stay memory-blind**: their persona instructions already
  mandate fresh eyes; the convention must forbid reviewer subagents from calling
  `mem_search`/`mem_context` (or scope their reads to the document content passed in
  the prompt only). This is the one method where memory can defeat the method.
- Effort: small-medium — one convention file + `egm-review-loop` persistence sections.

---

## 4. Pros / cons: Engram vs each method's original memory

### General (Engram)
- ✅ Cross-session recovery; survives compaction; semantic search ("what did we decide
  about X?"); stable topic_key upserts; lifecycle metadata flags stale memory; scopes
  personal/global fit user-profile data.
- ❌ Local DB — not team-shareable, not git-auditable; upsert overwrites (no iteration
  history); a second source of truth risks drift; mutable, so never an enforcement
  authority; availability depends on the host runtime.

### gentle SDD (Engram vs its Engram)
- Keeping Engram: ✅ native contract, recovery protocol, zero work.
- Cons of Engram-only: upsert loses iteration history → its own `hybrid` mode exists for
  that. No change recommended beyond pinning the mode.

### FTD (Engram vs feature-doc-in-repo)
- Keep original: ✅ single source of truth (no drift); evidence lives inline next to the
  checkbox it justifies (the doc's core rule); git audit + team shareable; works with
  zero memory tooling.
- Engram: ✅ resume without re-reading the repo; survives compaction; cross-feature
  search; user-profile travels across projects (personal scope).
- Verdict: **hybrid** — doc authority, Engram mirror. Engram-only would break the
  evidence rule and team visibility; doc-only is what FTD ships today and is fine, just
  not unified.

### ASDD (Engram vs 02-DOCS wiki + .rsc ledger)
- Keep original: ✅ tier-2 enforcement is *defined* on auditable files + machine-checked
  JSONL; decision queue is a legal record; CI/ship guards read files, not a local DB.
- Engram: ✅ in-flight change recovery after session loss; past-decision search feeding
  `specify`; per-task progress without re-parsing the wiki; user-profile.
- Verdict: **hybrid with authority separation** — Engram = recovery/search mirror;
  files + ledger = authority. Engram-only would be a compliance regression (auditable
  tier-2 record replaced by a mutable local observation: unacceptable).

### EGM (Engram vs none)
- Keep none: ✅ maximal simplicity; documents are portable and shareable by nature; no
  stale-memory risk; fresh-eyes property trivially preserved.
- Engram: ✅ multi-round review loops survive interruption (today they die with the
  session); findings become searchable knowledge for future drafts; brainstorm outcomes
  persist; drafts resumable.
- Cons: ⚠️ memory contamination — if a goldfish reads prior session memory it is no
  longer a goldfish; needs an explicit reviewer-blind rule. New dependency for a
  plugin that markets itself on simplicity.
- Verdict: **adopt, elephant-side only, with reviewer-blind enforcement.**

---

## 5. Recommended target state

1. One `engram-convention` per method (FTD, ASDD, EGM), all following the gentle
   contract's naming/lifecycle rules; gentle reused as-is.
2. Authority rule everywhere: file artifacts (02-DOCS, openspec/) stay the source of
   truth; Engram = working/recovery/search memory. Enforcement state (`.rsc/` ledger)
   never mirrored as authority.
3. A single `method-router` (Clef on Workers AI) reads the unified memory as its state
   and routes to `none | ftd | asdd | gentle-sdd | egm-review-loop`, logging
   `router/{slug}/decision` observations. Human gates (ASDD tier-2 approval, spec
   approval) stay human regardless of router confidence.
4. Store mode pinned per project (`engram` solo, or `hybrid` when team/git matters —
   default `hybrid` mirrors gentle's own default recommendation for real projects).

---

## 6. File-standard convergence and divergence control (decided)

**File layer:** ASDD's `02-DOCS/wiki/sdd/` structure is the common file standard for
the file-based methods — it is a strict superset of FTD's feature document (which
becomes the lightweight profile of the same standard). EGM adopts the feature-doc
pattern for its review loops (it has no file convention today). Gentle SDD keeps its
own authority model (Engram-native in `engram` mode, `openspec/` files in `hybrid`);
forcing it into `02-DOCS` would be a destructive migration.

**Dual-write is mandatory:** every disk artifact mutation carries an Engram leg. The
divergence policy is gentle's own contract, adopted verbatim — proven in production,
no detection machinery added (no hashes, no sweeps):

1. **Reads go to resolved per-store locators** — never a freshest-copy fallback, never
   silently substituting another store's copy. Store mode and locators are declared by
   the orchestrator/router, not inferred by the agent.
2. **Dual write with read-back** — attempt both declared writes following each store's
   convention, and read back each successful write.
3. **Non-atomicity declared honestly** — hybrid writes are not atomic: preserve
   successful writes, report any failed mirror as partial, and name the outstanding
   locator. Never claim synchronization; never roll back valid progress.
4. **Conflict: preserve both, escalate decision-scoped** — a genuine content conflict
   is kept in both stores; it is resolved (by the human) only when it affects the next
   decision.
5. **No second authority** — snapshots (progress mirrors, state observations) are
   recovery hints, never authority; stale snapshots do not replace actual progress.
   Reporting ranks landed evidence above snapshots (archive rule), but nothing
   auto-overwrites anything.

The router's state builder follows rule 1: it reads each method's resolved locators
(files for file-based methods, Engram topic keys for gentle SDD) — both stores stay
canonical, so no reconciliation layer is needed.

Answer to "what if they diverge?" per gentle: it is reported, preserved, and named —
never silently auto-healed. A failed mirror is drift with a name on it, and the next
decision that depends on it goes to the human.

### 6a. Relation to gentle's hybrid contract (v3.7.0)

The five rules above ARE gentle's hybrid contract (resolved locators, dual write with
read-back, no-atomicity honesty, preserve-and-escalate conflicts, snapshots as hints),
adopted verbatim because it is proven upstream. The earlier draft of this section
proposed hash-based reconciliation; that machinery is dropped — gentle's rule 4 covers
the failure mode (Engram fresher than files after a failed file write) by preserving
both versions and escalating, without any detection layer. Gentle's residual gap
(no post-write detection of a skipped or lying mirror) is accepted as-is: the
contract's honesty rules are the mechanism.

---

## 7. Per-method Engram adaptation work plan (gentle's contract, verbatim)

Shared base (once): port gentle's `engram-convention.md` + `persistence-contract.md`
as the common contract; each method adds its naming table. Session mode: **`hybrid` always — no asking, no fallback.** When Engram is
unavailable, the file write still lands and the Engram mirror is reported partial
with its outstanding locator named (gentle's rule 3), never silently skipped.

### FTD — smallest (validates the pattern)
- Naming: one topic key per feature: `ftd/{slug}/feature-doc` (`type: architecture`,
  `capture_prompt: false`).
- Touchpoint: `skills/ftd/SKILL.md` — the feature doc at `02-DOCS/wiki/ftd/<slug>.md`
  is mutated in place; add a persistence section: after every doc mutation, dual write
  (file + `mem_save` upsert) and read back; on one-sided failure report partial and
  name the missing locator.
- Recovery path: new session → `mem_search("ftd/{slug}")` + `mem_get_observation`, then
  verify against the file locator before acting.
- Effort: one persistence section + one naming table. 1 SKILL.md edited, 1 file added.

### ASDD — medium (dispatcher + six phases)
- Naming: `asdd/{change}/constitution|spec|plan|tasks|progress` (upsert per artifact)
  and `asdd/{change}/decisions/{id}` (`type: decision`).
- Touchpoints:
  - `skills/asdd/SKILL.md` (dispatcher): resolve store mode once per change, pass
    `artifactStore` + locators into every phase launch (mirrors gentle's orchestrator
    behavior).
  - Phase persistence paragraphs: `constitution`, `specify`, `plan`, `tasks`,
    `implement` (progress + decision queue), `ship` (closing report). Each: dual
    write + read-back + partial reporting.
  - EXCLUDED from Engram: `.rsc/review-ledger.jsonl` (CLI-owned enforcement state),
    `rsc risk` / `spec gate` inputs (disk-native CLI, unchanged), worklogs (git
    evidence).
- Effort: 7 SKILL.mds touched + shared convention. No CLI changes.

### EGM — small-medium (three skills + one rule)
- Naming: `egm/{doc-slug}/draft`, `egm/{doc-slug}/review/{round}`,
  `egm/{doc-slug}/resolution`.
- Touchpoints: `egm-draft` (write `docs/designs/<name>.md` + dual write),
  `egm-review-loop` (findings per round + dual write), `egm-resolve` (resolution +
  dual write).
- The goldfish rule: reviewer subagents never call `mem_*`; the document reaches them
  via prompt/locator only (they already launch with fresh context — this pins it).
- EXCLUDED: `egm-bootstrap` codebase READMEs (generated artifacts, not method state);
  per-runtime integration docs (`docs/claude.md`, `docs/codex.md` — packaging surface).
- Effort: 3 SKILL.mds touched + convention.

### Implementation order (when authorized)
1. Shared contract files (port from gentle).
2. FTD (validates end-to-end with the smallest surface).
3. EGM.
4. ASDD (dispatcher + phases).
5. Method-router (Clef) last — it reads the unified Engram layer only once methods
   write it.
