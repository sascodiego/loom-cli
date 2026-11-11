# Stage 9 — Implementation Readiness Report (Consolidated)

> **RE-AUDIT ADDENDUM (2026-11-10).** The resolution pass was applied to the
> design (declaration-based pre-flight replacing diff derivation;
> `user_gate_override`; OQ4 closed; milestones reordered; `router metrics`
> added; status/doctor/reconcile deferred to v2; rename moved to M1
> housekeeping). All three reviewers re-audited the revised DetailedRange:
> **READY WITH CONDITIONS** (unanimous). All first-audit BLOCKERs and 21
> MAJORs verified resolved. The 13 residual conditions (reroute ownership,
> rename window, exit-code mapping and test, detect verdict enum,
> substitution rule, criteria-excerpt rule, e2e test, clean-decision
> definition, criteria-hash scope, reconciled event, asdd scope list, M7
> verification) were applied as mechanical edits to the document. The single
> re-audit permitted by the convergence cap is consumed; the review is
> concluded. See below for the first-audit record.

Document: `docs/designs/method-router.md` — DetailedRange ("## Detailed Implementation", M1–M8)
Reviewers: egm-implementation-readiness, egm-codebase-details, egm-style (parallel, fresh context, implementation-readiness standard)
GoldfishContext: engram-unification-analysis.md, concepts_brief.md (+ critic_report.md, intermediate_zoom_out_report.md for continuity)
Date: 2026-11-10

## Verdict: NOT READY

egm-codebase-details states it outright and the other two audits agree in
substance: three load-bearing implementation inputs are undefined, one cited file
is never created, later milestones have no test plans, and the plan implements
FeatureRange decisions that were never made. Every milestone is individually
legible; the plan as a sequence cannot be executed without the implementer
making design decisions mid-build — the exact failure the readiness standard
prohibits.

## Convergence map

- **Empty-diff derivation** (all 3; 5th stage confirmation of the FeatureRange
  BLOCKER): M2 `derive.go` implements a mechanism the design never defined. The
  plan must state what facts exist at request time or the milestone cannot ship.
- **First-gate override has no lifecycle** (all 3): `record` cannot be
  implemented — no subcommand, no `decided_by` value for a user rejecting Clef's
  suggestion at the confirm gate.
- **OQ4 decided at code time** (implementation-readiness #2, design #6): M3
  hardcodes the privacy choice the document declares pre-production-blocking.
- **M3/M4 ordering inverted** (style #3, implementation-readiness #7): decide.go
  loads criteria files that M4 creates later; M3 cannot compile or test.
- **Config schema contradictions** (implementation-readiness #3, codebase-details
  #2): `size_class` thresholds are declared "in config" but the config schema
  has no such field, values, or boundary definitions.
- **Measurement unimplemented** (implementation-readiness #4): no milestone
  computes the agreement rate from `.router/history/` — the number the entire
  project justifies itself with has no producer.
- **Registry/naming drift** (implementation-readiness #5, codebase-details #8,
  style #1): registry shape without values; M4 namespaces inconsistent with M6;
  "four methods" prose vs five IDs; `gentle-sdd` status unresolved.
- **Test gaps** (codebase-details #7, style #4/#10): no tests for M4–M8, no
  end-to-end CUJ coverage, exit codes declared but never wired per subcommand.

---

## egm-implementation-readiness

1. **BLOCKER — M2 implements a mechanism that cannot produce its inputs.** No
   stated behavior when the diff is empty (prose-only request? explicit base-ref
   interview? skip facts?). The implementer must invent the fallback behavior
   for the plan's core classification input — violates the Zero-Decision Rule.
2. **BLOCKER — OQ4 is unresolved and M3 hardcodes a choice anyway.** Whether
   `scope_summary` travels raw or as digest is a privacy-relevant design
   decision made at code time.
3. **MAJOR — Config schema contradiction between M1 and M2.** Thresholds
   declared "in config" have no schema field, shape, or defaults.
4. **MAJOR — The delegation-earning measurement has no implementation item.** No
   command, script, or report computes agreement rate; the threshold is
   undefined.
5. **MAJOR — Registry contents are enumerated as fields, not values.** Engram
   namespace strings undefined; M6 defines them inconsistently with M4.
6. **MAJOR — M7 reference updates are unbounded** ("any cross-references") —
   not allowlist-suitable; "create or extend" leaves root README undecided.
7. **MAJOR — Milestone ordering breaks buildability: M3 depends on M4
   artifacts** (criteria files, registry). Each phase must leave the tree
   buildable; M3 as ordered does not.
8. **MINOR — M6 `engram-contract.md` has no canonical home** — three adapted
   copies with no owner is a drift hazard.
9. **MINOR — Exit-code semantics are global, not per subcommand** — the harness
   joint-fallback branching depends on the distinction.
10. **MINOR — Test plan covers state and provider, but not CLI wiring or CUJs** —
    no end-to-end `decide → record → detect → close` or exit-4 conflict tests.

**Question:** What exactly should `derive.go` emit for a brand-new request, and
what input does Clef classify on in that case?

---

## egm-codebase-details

**Verdict: NOT READY.** Cited edit targets verified to exist on disk; but three
load-bearing inputs undefined, one cited file never created, M4–M8 without test
plans.

1. **BLOCKER — M2 `derive.go` implements a mechanism the design never defined.**
   Base ref undefined at first request; the implementer must choose it —
   zero-decision failure.
2. **BLOCKER — The config schema cannot support `size_class`.** No thresholds
   key, no numeric defaults, no boundary definitions anywhere in the range.
3. **BLOCKER — `record` cannot be implemented: first-gate user override has no
   lifecycle path.** No subcommand, no `decided_by` value; `human_approved`
   semantics reference the open confirm-gate contract.
4. **MAJOR — M3 "extends" a file no milestone creates** (`internal/cli/
   decide.go` absent from M1's enumeration; M1's write allowlist wrong).
5. **MAJOR — Branch-name path collision unspecified in every state operation**
   (slugify? escaped segments? `slug` derivation rule not stated).
6. **MAJOR — M7 reference-update enumeration is open-ended**, violating the
   file-boundary requirement.
7. **MAJOR — M4–M8 have no test or verification plan** — no acceptance criteria
   for half the plan (criteria compliance, registry gating, skill script,
   persistence paragraphs, rename reference check).
8. **MAJOR — M4 pre-empts the open OQ2 decision** (`egm-review-loop
   available: false` bakes an unratified choice into code; if OQ2 resolves
   differently, M4/M5 are rework).
9. **MINOR — Criteria hashing is underspecified** (which files, what order,
   registry included?).
10. **MINOR — M6's egm file list looks under-enumerated** (`egm-review`,
    `egm-brainstorm`, `egm-drift-report`, `egm-bootstrap` not marked
    out-of-scope).

**Question:** What exact diff does M2's `derive.go` take for a brand-new
request — and if "no diff," what replaces `diff_stats`/`size_class` concretely
enough that `schema.go`, the config schema, and the thresholds can be written
without further design choices?

---

## egm-style

Persona: naive reader, copy-editor lens. Plan clarity, ordering, completeness,
actionability.

1. **BLOCKER — M2's fact derivation is not actionable for the path that consumes
   it** — reproduces the four-stage FeatureRange BLOCKER inside the plan; M2 as
   written cannot produce a valid decide payload.
2. **MAJOR — M3 depends on M2's output shape but does not state how empty facts
   are handled** (fail, warn, or proceed?).
3. **MAJOR — Milestone ordering breaks dependency direction between M3 and M4**
   (criteria files created in M4; M3 untestable until M4 lands — reorder or
   fixture).
4. **MAJOR — No tests enumerated for M3's CLI layer** beyond the provider
   (output shape, retry wiring, exit-code mapping 0/2/3/4 unverified).
5. **MAJOR — M5 references subcommands and "the lossless evidence presentation
   format" without restating the contract** — not self-contained.
6. **MAJOR — Exit-code semantics declared but never wired** to enumerated
   behaviors; reconcile classes have no severity/code assignment.
7. **MINOR — M6 edit paths assume structures the plan never creates or
   verifies**; edit items without anchors.
8. **MINOR — M7 renames after M4 already references the canonical name** —
   known-broken intermediate window, unacknowledged.
9. **MINOR — "genlte-sdd" typo persists inside M7's own bullet** (inconsistent
   reference).
10. **MINOR — M8's historical-document edit is not self-contained** (superseded
    item IDs not enumerated).

**Question:** What concrete facts does the CLI derive for a `new` request where
no branch diff exists — and should M2 specify a request-time derivation source
instead of a diff?
