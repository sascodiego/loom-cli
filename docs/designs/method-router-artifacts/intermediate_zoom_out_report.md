# Stage 8 — Intermediate Zoom Out Report

Document: `docs/designs/method-router.md`
Reviewer: egm-zoom-out (fresh context, zoom-out standard)
Prior reports in context: zoom_out_report.md (Stage 5), comprehension_report.md (Stage 6), critic_report.md (Stage 7)
GoldfishContext: engram-unification-analysis.md, concepts_brief.md
Date: 2026-11-10

**Overall read:** the document visibly absorbed parts of the critique (Clef
definition, asdd/gentle-sdd boundary, continuation mechanics, discard ownership,
retry counts, human_approved stratification). But the single most-validated defect
in the entire review chain — the empty-diff-at-decision-time BLOCKER, now
confirmed by four stages — is still in the text unchanged, and several structural
gaps (delegation criterion, Risks section, method count) were not touched.

## Findings

1. **BLOCKER — (b) prior, still unaddressed (4th stage).** CLI-derived facts are
   still empty at decision time. M2 `derive.go` still computes
   `touched_areas`/`diff_stats`/`size_class` from `git diff --numstat` against a
   base ref, and CUJ 1 runs `decide` at request time before work exists — yet
   the doc now asserts the circularity is "closed by construction" (Pre-flight
   schema) because the CLI owns the facts. Half the input to Clef is vacuous;
   the fix for agent bias is itself architecturally undefined. Nothing in
   Stages 5–7 was incorporated here.

2. **MAJOR — (b) prior, still unaddressed.** Method count identity: the Overview
   still says "four extracted development methods" while the registry, enum, and
   diagram carry five (`odd, ftd, asdd, gentle-sdd, egm-review-loop`), and G7
   says "all four methods write" through the unified contract. Style's BLOCKER
   stands; a fresh reader still cannot tell if `gentle-sdd` is routable v1 or
   covered by G7.

3. **MAJOR — (b) prior, still unaddressed.** The delegation-earning criterion
   remains undefined. G6/NG3 defer phase 2 to "phase-1 measurements," but the
   only quantified trigger is the 30% override tripwire; no threshold, sample
   floor, or success definition exists (the `human_approved` stratification
   added in M1 helps measurement quality but not the decision rule). Phase 2 is
   still undecidable from the document's own terms.

4. **MAJOR — (b) prior, still unaddressed.** Discard-and-restart economics
   unchanged. G5/CUJ4/NG1 still mandate discard-and-restart with no artifact
   migration; the doc now clearly assigns the discard to user + orchestration
   agent (good), but never says what "discard" means for on-disk authority
   artifacts, nor acknowledges that a user three tasks in will rationally keep
   an under-capacity method — polluting the phase-1 override baseline.

5. **MAJOR — (b) prior, still unaddressed.** No Risks & Mitigations section.
   Risk content remains scattered across OQ4, NFR paragraphs, and Alternatives
   cons; misroute cost, criteria drift, Clef retirement, and mirror staleness
   are still never named as risks. Stage 7's structural #3 stands.

6. **MAJOR — (b) prior, still unaddressed (cluster of design findings).**
   Operational edges still undefined: branch-name `/` collision in
   `.router/open/<branch>/` (design #2), no confidence-threshold policy for
   near-uniform Clef distributions (design #7), no concurrency rule for two
   sessions deciding on one branch (design #5), and rollback (`git revert` of
   `.router/`) still leaves Engram mirrors permanently stale (structural #6).
   None appear in the current text.

7. **MAJOR — (b) prior, still unaddressed.** OQ4 privacy gate is declared but
   not enforced: `decide` sends raw `intent`/`scope_summary` from day one, and
   the gate remains "before production data accumulates," not before first
   rollout. Criteria prose traveling with every call is likewise still outside
   the privacy decision (structural #7).

8. **MAJOR — (a) NEW — YAGNI surface persists despite the Stage 5 callout.**
   Seven subcommands, `reconcile.go` with typed discrepancy classes, `doctor`,
   criteria hashing, UUID-keyed per-branch records are all still in scope before
   anyone has shown the router beats the joint fallback. The Stage 5
   recommendation (v1 = detect/decide/record/close; reconciliation after
   observed divergence) was not adopted, and nothing in Stages 6–7 contradicts
   it. This is the largest remaining cuttable surface at altitude.

9. **MAJOR — (b) prior, partially mitigated.** Clef justification vs. joint
   fallback is still asserted rather than earned. Alternative C's rejection now
   names the 30% override tripwire and the rubric-as-degradation-path, but the
   inventive-catalyst's cheapest experiment — run joint-fallback mode as primary
   for a comparison window — is still absent, and the design still forbids Clef
   ever losing (NG5 blocks the adapter that would retire it). Mitigated in
   framing, not in substance.

10. **NIT — (c) adequately mitigated (positive notes).** Several Stage 6/7
    findings are now properly handled: Clef is defined with an endpoint contract
    (comprehension #1); asdd vs gentle-sdd gets a mutual-exclusion criteria rule
    with both probabilities shown (comprehension #3); continuation matching is
    now a by-key candidate selection with user resolution per Ruling 8
    (comprehension #2); discard ownership is correctly scoped in CUJ 4
    (structural #9); retry counts reconciled to "2 retries, 3 attempts total"
    (comprehension #7); `human_approved` semantics for joint_fallback defined
    and stratified (comprehension #5); `egm-review-loop` is pinned
    `available: false` with the diagram footnote (structural #4). Residual
    NITs: `request_id` still reads as minted in two places (style #3); G1 still
    lists a fourth axis (`context-degradation risk`) nothing measures (design
    #3); formulaic "Rejected by author" tails and the Spanish example persist
    (style #8/#6).

**Question for the author:** At the moment `router decide` runs — before any
work exists — what do `touched_areas`, `diff_stats`, and `size_class` actually
contain, and if the honest answer is "nothing until the agent explores," does
that not reintroduce the exact pre-flight exploration tax that Alternative B
rejected?
