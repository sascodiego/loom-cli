# Stage 7 — Critic Report (Consolidated)

Document: `docs/designs/method-router.md` — FeatureRange 1..400
Reviewers: egm-design, egm-structural, egm-style (parallel, fresh context, feature-spec standard)
GoldfishContext: engram-unification-analysis.md, concepts_brief.md (11 binding rulings)
Date: 2026-11-10

## Convergence map

- **Diff at request time** (structural #1): independently confirmed by Stage 5
  (zoom-out BLOCKER) and Stage 6 (comprehension #6). Third stage in a row — the
  most validated defect in the document.
- **Method identity: "four methods" vs five registry IDs, `gentle-sdd` status
  undefined** (style #1 BLOCKER, structural #8 MAJOR, design #3 partially).
- **Delegation criterion missing** (design #1 BLOCKER, structural #5 MAJOR).
- **Schema gaps at the confirm gate**: user override at first gate has no
  `decided_by` value (structural #2 BLOCKER).
- **Path encoding**: branch-name keying collides on `/` (design #2).
- **Privacy**: OQ4 declared blocking but unenforced (design #6); criteria prose
  travels unexamined (structural #7).
- **Undefined terms**: "more robust method" ordering (style #2), discard
  contract for authority artifacts (design #4), concurrency (design #5),
  confidence threshold (design #7), rollback/mirror staleness (structural #6),
  missing Risks section (structural #3), OQ2 contradiction (structural #4).

---

## egm-structural

Persona: egm-structural (REVIEW mode) · Standard: feature-spec-standard.

1. **BLOCKER — The core classification inputs are temporally impossible as
   specified.** The pre-flight schema says the CLI derives
   `diff_stats (files, insertions, deletions)` and `size_class (from diff
   thresholds)` (Technical Plan, Pre-flight schema). But routing happens at
   request time, before any work exists: at CUJ 1 there is no diff to measure
   except pre-existing uncommitted changes. The document never defines what the
   diff is taken against, when it is sampled, or the numeric thresholds for
   `size_class`. The load-bearing input to Clef is undefined.

2. **BLOCKER — `decided_by` has no value for a first-decision user override.**
   Ruling 7 defines `decided_by ∈ {clef, joint_fallback, override_re_route}`;
   the only override event described is a re-route superseding an *open* record.
   In phase 1 the user can override a Clef suggestion at the very first confirm
   gate (G5). That event has no lifecycle subcommand and no `decided_by` value.
   Round 3 explicitly required distinguishing "Clef suggested, human approved"
   from "Clef decided" — the schema cannot record it.

3. **MAJOR — Required "Risks & Mitigations" section is missing.** The standard
   mandates failure modes, security implications, limits. Risk content is
   scattered (OQ4 privacy, NFR paragraphs, Alternatives cons). Misroute cost,
   criteria drift, and Engram-mirror staleness are never named as risks.

4. **MAJOR — Contradiction between Open Questions and the Technical Plan on
   `egm-review-loop`.** OQ2 says "to define"; the Technical Plan already decides
   (`available: false`, excluded from v1 dispatch). Stale-state defect: either
   OQ2 is closed with the ruling that closed it, or the plan pre-empts an
   unresolved decision.

5. **MAJOR — The delegation-earning criterion is asserted but never defined.**
   G6/NG3 hinge on it. No metric, threshold, sample floor, or stratum definition.
   Round 2's measurement-validity finding is silently dropped.

6. **MAJOR — Rollback path leaves the Engram mirror permanently stale.** NFR:
   "rollback is a `git revert` of `.router/`." No cleanup, tombstone, or honest
   degradation for Engram records after a revert — the mirror reports open
   routings for decisions that no longer exist as authority.

7. **MAJOR — CRITERIA.md prose leaves the machine but is excluded from the
   privacy decision.** Five criteria files of method-authors' prose travel on
   every classification; their privacy treatment is unaddressed (OQ4 covers only
   pre-flight prose).

8. **MAJOR — Method identity remains ambiguous: "four methods," five registry
   IDs.** Overview names four; registry carries five; `ftd` vs `rsc-ftd` not
   acknowledged; OQ1 defers an identity contract that keys Engram namespaces and
   dispatch to "implementation."

9. **MINOR — G5's "an approved re-route discards in-flight work" misassigns
   ownership.** CUJ 4 correctly scopes discard to user + orchestration agent;
   G5 reads as if the re-route itself discards.

10. **MINOR — Method ID mismatch between prose and diagram.** `gentle-sdd` vs
    "RSC-ASDD"/"RSC-FTD" prose; a fresh reader cannot tell the canonical routing
    ID string G4 depends on.

**Question:** What exactly does the CLI diff, and against what baseline, to
derive `diff_stats` and `size_class` at the moment a user first requests work
that has not been started?

---

## egm-design

Persona: egm-design (hostile critic). Standard: feature-spec-standard.

1. **BLOCKER — Phase-1 success is unmeasurable: no threshold for earning
   delegation.** NG3/G6 defer to "phase-1 measurements" but the only quantified
   trigger is the CRITERIA-revision tripwire (>30% override). The round-3
   refinements (sample floor, cooldown, stratification, input-vs-classifier
   error separation) appear nowhere. Phase 2 can never be decided from this
   document's own terms.

2. **MAJOR — Branch-name paths are broken by design: `.router/open/<branch>/`
   collides with `/` in branch names.** `fix/339-cedula` produces nested
   directories — `fix/339/id.json` is indistinguishable from branch `fix`
   containing subdirectory `339`. Ruling 9 fixed slug collision but introduced
   path-encoding ambiguity; detection enumerates this directory.

3. **MAJOR — G1's classification axis is internally inconsistent: four
   dimensions in Goals, three everywhere else.** `context-degradation risk` is
   never an observable input; if the fourth dimension is real, nothing measures
   it; if it isn't, G1 overclaims.

4. **MAJOR — Re-route discard semantics undefined for already-written authority
   artifacts.** CUJ 4 defers mechanics to the user/orchestration agent, but for
   an `asdd` change the wiki and ledger are legal authority: deleted, archived,
   or preserved? A fresh implementer cannot write this code path.

5. **MAJOR — No concurrency story for simultaneous routing on one branch.** Two
   sessions (or worktrees) can both run `decide` and `record`; `routing_id`
   disambiguates records, not decisions; no lock, last-write-wins, or
   reconciliation rule.

6. **MAJOR — The OQ4 privacy gate is declared but unenforced.** `decide` sends
   `intent` + `scope_summary` from day one; nothing gates the feature on OQ4's
   resolution, which the NFR itself calls pre-production-blocking.

7. **MAJOR — No confidence-threshold policy for Clef output.** No behavior for
   low-confidence or near-uniform distributions; is 0.34-vs-0.33 auto-suggested?
   Every phase-1 measurement inherits whatever silent policy the implementation
   picks.

8. **MINOR — Unavailable-method recommendation is undefined while
   `egm-review-loop` ships `available: false`.** If Clef returns it with highest
   probability, no defined next action (block? next-best? joint fallback?).

9. **MINOR — Dangling dependency: "identical wording in M3"** cites milestones
   undefined within the reviewed range.

10. **MINOR — Continuation recognition's human check is asymmetric.** Declaring
    `new` triggers warning + user resolution, but the agent selecting a wrong
    candidate is only confirmed by the same biased agent; a wrong selection
    bypasses routing forever. A one-line user echo of the selected record would
    close it.

**Question:** What concrete, quantified phase-1 result earns delegation in
phase 2 — and until that is written, what stops `router decide` from shipping
with the OQ4 privacy decision still open?

---

## egm-style

Persona: egm-style. Standard: feature-spec-standard.

1. **BLOCKER — The document cannot state how many methods exist.** Overview and
   NG6 say four (ODD, RSC-FTD, RSC-ASDD, EGM); the registry, diagram, and enum
   contain five, and `gentle-sdd` appears in neither four-method enumeration. A
   fresh implementer cannot tell whether `gentle-sdd` is routable v1, where its
   CRITERIA.md comes from, or whether G7's "all four methods" covers it.

2. **MAJOR — "the more robust method" is used as if it were a defined ordering,
   and it is not.** No robustness ranking exists among {odd, ftd, asdd,
   gentle-sdd}; the dispatch rule for the most consequential event (Ruling 11)
   is unwritable.

3. **MAJOR — `request_id` is minted in two places.** `decide` prints it;
   `record` "mints routing_id and request_id." Both readings are supported;
   the ID spans pre-flight → Clef → history.

4. **MAJOR — NFRs reference milestones M3 and M6 never defined in the reviewed
   range.** Undefined symbols inside normative requirements.

5. **MINOR — "the human always holding the final call" overstates the design** —
   unconditional phrasing contradicts the staged phase-1/phase-2 stance.

6. **MINOR — CUJ 1's example is in Spanish** (`"integrá auth de UI con
   backend"`) in an otherwise English document.

7. **MINOR — Broken sentence + soft bound in the Performance NFR** (mid-thought
   wrap; "should stay small" vs a hard number).

8. **MINOR — Every alternative rationale ends with "Rejected by author —"**;
   formulaic tail, no information under a heading already named "Reason for
   rejection."

9. **MINOR — Colloquial phrasing in normative prose** ("grades its own
   homework," "buried in ceremony," "ships unverified drift").

10. **MINOR — Overview sentences exceed 50 words with stacked em-dashes.**

**Question:** Is `gentle-sdd` a routable v1 method whose CRITERIA.md and Engram
adapter must be built like the others — or a fifth, already-Engram-native method
that merely inherits the unified contract — and why does the document say "four
methods" everywhere?
