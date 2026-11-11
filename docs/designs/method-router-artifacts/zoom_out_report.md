# Stage 5 — Initial Zoom Out Report

Document: `docs/designs/method-router.md` (full file)
Reviewers: egm-zoom-out, egm-inventive-catalyst (parallel, fresh context, zoom-out standard)
GoldfishContext: engram-unification-analysis.md, concepts_brief.md
Date: 2026-11-10

---

## egm-zoom-out

Context sufficiency check (§0): **PASS.** Objective and motivation are clear; the
document is self-contained and traceable to the brief's rulings.

1. **BLOCKER — The observable facts may not exist at decision time.** The
   pre-flight derives `touched_areas`, `diff_stats`, and `size_class` from
   `git diff --numstat` against the base ref (M2, `derive.go`; CUJ 1). But
   routing happens at *request* time, before any work is done — the diff is
   empty. If the intent is to derive facts *after* initial exploration, that
   contradicts Alternative B's rejection ("no pre-flight exploration") and adds
   latency to every request; if it is at request time, the CLI-derived half of
   the schema — the very fix for the self-attested-input problem (brief Ruling 7
   / round-2 finding 2) — is vacuous. The design's central anti-bias mechanism
   appears architecturally undefined. This must be resolved before
   implementation.

2. **MAJOR — Clef has no baseline to beat, and the design forbids ever finding
   one.** The brief's strongest round-1 finding (all lenses) asked what the
   offline rubric gets wrong and what error rate justifies a network dependency.
   Ruling 1 reframed phase-1 measurement as "baseline-building," but measuring
   *Clef vs. user* while Clef is deployed never measures *Clef vs. rubric* — and
   the round-2 candidate (record the triple: rubric, Clef, human) was dropped
   without a ruling. The design's own escalation rule (30% override → revise
   CRITERIA.md, never retire Clef) means the network classifier can never lose.
   The IFR here may be: rubric decides, Clef only handles the cases the rubric
   marks ambiguous. The design asserts Clef's necessity rather than earning it.

3. **MAJOR — "Provider-agnostic" is a one-vendor contract; this is the two-year
   obsolescence risk.** NG5 locks the provider interface to the Clef shape
   (typed questions, calibrated probabilities), with exactly one implementation
   (`@cf/cloudflare/clef`). If Cloudflare retires, repackages, or reprices Clef —
   plausible on a two-year horizon — the entire routing layer has no fallback
   except the joint-fallback conversation, which is the manual status quo the
   design exists to remove. A chat-model adapter is called out of scope, so the
   "agnosticism" is architectural vocabulary, not actual optionality. Either
   accept this honestly (it's a Cloudflare-coupled router) or define the
   degradation path as a real path.

4. **MAJOR — Discard-and-restart (Ruling 11) makes the escape hatch more
   expensive than the misroute.** Restarting under ASDD throws away verified
   in-flight work, including work the user watched land. A user three tasks into
   an ODD change that outgrew itself faces a choice between abandoning real
   progress or continuing under-capacity — most will continue, which pollutes
   the phase-1 override metric and quietly reverts routing to agent
   self-assessment. "No artifact migration" is a fine ruling for method
   *selection*; applying it to *user-approved* re-routes where the user
   explicitly wants to preserve work is over-rigid. The design should at minimum
   declare what "discard" means when the file artifacts are still on disk.

5. **MAJOR — Phase-1 machinery is heavy for an unproven feature (YAGNI).** Seven
   subcommands, JSONL event history, a reconciliation engine comparing three
   sources with typed discrepancy classes (`reconcile.go`), criteria hashing,
   `doctor` probing, UUID-keyed per-branch records — all before anyone has shown
   the router beats manual choice. `router status`/`reconcile.go` is the
   clearest weed: it exists to police a state format that hasn't shipped yet. A
   v1 with `detect`/`decide`/`record`/`close` and a "read the open file; if it
   looks wrong, ask the user" rule would validate the concept with roughly half
   the surface. Reconciliation earns its place once real divergence is observed.

6. **MEDIUM — The design bundles two products against the analysis's own
   ordering.** `engram-unification-analysis.md` §7 explicitly orders the router
   *last*, only after methods write the unified layer; this design's plan runs
   M1–M3 (router CLI) before M6 (method adapters), patching the inversion with
   the `available` registry flag and OQ3. The gating is a real mitigation, but
   the plan still invites building and testing the classifier before its
   consumers exist. The rollout paragraph says adapters ship before routing
   opens — make the *plan* match that, not just the rollout note.

7. **MEDIUM — `.router/` in git is a permanent tax on every repository using
   this.** Every work item adds open/close/supersede commits under `.router/`,
   cluttering `git log`, PR diffs, and squash merges (the brief's own round-3
   Finding 1 documented how commit churn breaks anchors). The design moved
   detection to working-tree-only, which fixes liveness but leaves the
   write-tax: history files accumulate forever with no pruning or retention
   policy. For a phase-1 experiment whose value is unproven (finding 2),
   untracked-local state with optional export is a simpler default.

8. **NIT — Agent bias re-enters through `intent`/`scope_summary` prose.** The
   circularity fix moved the tier out of agent hands, but the agent still writes
   the prose Clef classifies over; a lazily framed scope summary steers the
   classifier exactly as a lazy `size_class` would have. The design closed the
   mechanical hole and left the rhetorical one. Worth one sentence acknowledging
   this residual, since the phase-1 measurement interprets disagreement as
   signal.

9. **NIT — Registry/criteria reuse is good; the design correctly leverages
   existing material.** Positive note: CRITERIA.md bootstrapped from each
   method's own NOT-statements, gentle's persistence contract adopted verbatim,
   ODD as floor, explicit non-items section — the reuse and anti-YAGNI
   discipline in M4–M6 and "Explicit non-items" is genuinely strong and
   consistent with the codebase.

10. **NIT — OQ2 and OQ4 are implementation-blocking, not "to settle later."**
    Routing into `egm-review-loop` without an upstream document, and the
    raw-prose-vs-digest privacy shape, both change the pre-flight schema and the
    registry — the v1 contract this design freezes. Flag them as
    must-resolve-before-M1, not as a trailing Open Questions list.

**Most important question for the author:** At the moment `router decide` runs —
before any work exists — what do `touched_areas`, `diff_stats`, and `size_class`
actually contain, and if the answer is "nothing until the agent explores,"
doesn't that reintroduce the exact pre-flight exploration tax that Alternative B
rejected?

---

## egm-inventive-catalyst

Persona: egm-inventive-catalyst (TRIZ / contradictions / IFR). Standard:
zoom-out. Evidence: design doc + `concepts_brief.md` (11 rulings) +
`engram-unification-analysis.md`.

1. **MAJOR — The design never justifies Clef against its own fallback path,
   which is a complete classifier.** The joint agent+user decision (CUJ 3,
   Ruling 3) uses the identical inputs (pre-flight prose + compared
   CRITERIA.md) and produces a human-approved decision with zero latency,
   credentials, quota, or privacy exposure. If that path is good enough for
   outages, it is good enough for phase 1; Clef must beat *that*, not the
   strawman "user choosing manually with no evidence." The doc measures
   `human_approved` but never proposes running fallback-mode as the primary for
   a comparison window — the cheapest experiment that could kill or justify the
   entire network dependency is absent.

2. **MAJOR — Measurement validity debt from round 2 was dropped, not resolved.**
   The brief's round-2 finding 5 (5/7 lenses: "agreement ≠ correctness; no
   ground truth; candidate: shadow rubric recording the triple (rubric, Clef,
   human)") does not appear in the design. The 30% override threshold (NFR
   section) is also unstratified, despite the brief explicitly demanding
   stratification by `decided_by` and separation of input error from classifier
   error. As written, phase 1 can "succeed" while Clef adds nothing over a
   rubric — and the design would have no data proving otherwise.

3. **MAJOR — Ruling 11's discard-and-restart makes the escape hatch economically
   irrational, so the misroute cost is unbounded.** A change that outgrew `odd`
   at unit 3 of 10 has accumulated exactly the context the heavier method needs;
   discarding it (NG2, CUJ 4) means the agent will rationally never propose a
   re-route and users will rationally never approve one. This is a hidden
   contradiction — "no migration" vs "preserve accumulated work" — resolved by
   compromise instead of elimination; the rejected handoff-stub pattern (brief,
   lens finding 2) was the TRIZ-resolving shape and its rejection rests on
   Ruling 2's one-shot premise, which re-route itself already violates.

4. **MEDIUM — Two products still ride in one design.** M6 (Engram unification)
   and M7 are independently shippable and — per the analysis's own ordering and
   brief finding 7 — come first; the router (M1–M5) is the speculative
   deliverable that depends on them. Bundling means a failure or redesign of the
   router's classification core leaves the unification hostage to it. The
   rollout gating (M6 before routing opens) mitigates execution order but not
   review, approval, and rollback coupling.

5. **MEDIUM — Two-year obsolescence concentrates in the Clef-shaped interface,
   not the router.** The durable value is provider-independent: continuation
   detection, the append-only decision record, the criteria files. The bet that
   calibrated one-forward-pass decision endpoints remain the right
   classification substrate is the fragile part (NG5 explicitly forbids the
   generic-LLM adapter that would otherwise be the hedge). If harnesses gain
   native structured classification, the provider layer and its config/retry/
   doctor machinery become the dead weight. The doc should state which
   components survive a Clef retirement and which are sacrificial.

6. **MEDIUM — Tracked machine state (`.router/`) in git is unexamined at the
   workflow level.** Every decision, re-route, and close is a commit to a shared
   tracked directory; squash/rebase churn — the exact failure mode round 3
   (Finding 1, 7/7 lenses) killed ruling 6's git anchor for — still applies to
   the history JSONLs' audit value, and multi-checkout/worktree flows (this
   machine runs a worktree-per-branch convention) silently fork or lose open
   records keyed to branch names. The design resolved detection to
   working-tree authority but never asked whether the tracked audit trail earns
   its merge-conflict and churn cost versus a tracked summary + local detail.

7. **MEDIUM — OQ2 and OQ3 are load-bearing for the M4 registry schema but
   deferred past design freeze.** Whether `egm-review-loop` is routable and how
   `available` gating surfaces are schema decisions (entry-artifact field?
   per-method status enum?), yet they are marked "to settle at implementation."
   Deferring them risks reworking the registry and criteria hashing contract
   after M4 ships.

8. **LOW — OQ4 (payload privacy) is sequenced after exposure begins.** CUJ 1
   sends raw `intent`/`scope_summary` prose to Cloudflare on day one; "settle
   before production data accumulates" is the wrong gate — the gate is first
   rollout, since phase-1 data is itself production-adjacent intent leakage. The
   brief flagged this as a "decide now" item.

9. **NIT — Altitude control is good overall, but the Detailed Implementation
   drifts below design altitude** (per-file Go layout, exit-code tables,
   test-fixture mechanics). Acceptable for an implementation-ready doc; flagging
   only that these sections will rot fastest and should be marked non-normative.

**Most important question for the author:** What measured evidence would justify
Clef over the joint agent+user fallback — and will phase 1 be instrumented
(shadow rubric + stratified override data) so that question can actually be
answered, or is Clef's adoption an unfalsifiable premise of the design?
