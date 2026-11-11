# Method Router: Clef-Classified Work Routing for Gentle-Shell

* **Author:** Diego Sasco
* **Date:** 2026-11-10
* **Status:** Brainstorm brief — 7-lens Goldfish review complete, objections open
* **Reviewers:** egm-design, egm-codebase-details, egm-zoom-out, egm-comprehension, egm-implementation-readiness, egm-structural, egm-inventive-catalyst (parallel, fresh-context, BRAINSTORM mode)

## Overview

A provider-agnostic CLI (Go, living in `/src/loom-cli`) routes incoming software work to
the right development method using Clef, Cloudflare's decision model (state + typed
questions → calibrated probabilities in one forward pass). When the user requests work,
the agent fills a lightweight pre-flight schema; the CLI sends it plus per-method
`CRITERIA.md` files to a Clef endpoint; Clef returns an effort tier and a recommended
method (`none/inline`, `ftd`, `asdd`, `gentle-sdd`, `egm-review-loop`). Routing is
one-shot per new work request: the decided method runs unchanged to completion — no
mid-flight method switching. Clef knows whether a request is new work or a
continuation of an already-decided change; continuations keep their decided method
and are never re-routed. In phase 1 the human always
makes the final call; suggestion-vs-choice is measured to decide later whether routing
can be delegated. Project scope includes the Engram unification of the four methods
(gentle's persistence contract, hybrid always).

Target user: a developer working in gentle-shell (Pi) who wants method discipline
selected and enforced per task without manually choosing a process each time.

Core value proposition: the right amount of process for each piece of work — no
spec ceremony for a typo, no freestyle for an auth change — with the classification
made by a fast decision model instead of the agent grading its own homework, and the
human holding the final call until the router earns trust.

## Analysis

The seven lenses converged on a small number of load-bearing problems. Every HIGH
below was raised independently by three or more lenses.

### 1. The router has no proven baseline to beat (egm-zoom-out, egm-inventive-catalyst, egm-comprehension)

The decision space is five options and the inputs are five fields. A deterministic
rubric in `CRITERIA.md` ("tier ≥ 2 or touches auth/billing/CI → `asdd`") produces
debuggable, offline, zero-latency decisions today. RSC already ships offline
classifiers (`rsc-tools/scripts/lib/risk.js`, `recommend.js`) and a routing table
(`asdd/references/model-routing.md`); FTD's evals contain `routing.json`. No lens
found evidence that probabilistic ranking beats these, and FTD's own doctrine states
"size never selects escalation" — a per-work-unit re-classification on size inputs
contradicts the method it routes to. The concept must define the rubric baseline and
the measured error rate Clef must beat, or the project's core justification is
assertion. Candidate resolution: rubric as the default path, Clef justified (and
measured) for the residual cases the rubric cannot separate.

### 2. "No artifact migration" is the load-bearing undefined term (6 lenses)

Bidirectional escalation away from a method that already wrote authority artifacts is
where the design breaks: ASDD's tier-2 gate legally requires the wiki + ledger record;
gentle-SDD artifacts live in Engram/openspec; FTD's promotion rule says a promoted
feature doc becomes the spec input with no completed work discarded. "Run the native
flow from the decision moment" leaves the boundary undefined — what each method
REQUIRES on entry, what it LEAVES behind on exit, and who owns stranded invariants.
egm-inventive-catalyst's pattern (Beforehand Cushioning): a mandatory handoff stub —
one file each method must leave describing the invariants it established, which any
successor method inherits or explicitly disclaims. Also unresolved: whether
de-escalation is blocked while gates are open (tier-2 panel pending), and the locator
handoff when the new method reads from a different store (gentle rule 1 forbids
fresh-copy fallbacks).

### 3. The pre-flight schema is self-attested input to a grader (6 lenses)

`{intent, scope_summary, touched_areas, size_class, effort_tier}` is filled by the
same agent that benefits from a lighter method; a lazy `size_class: small` silently
routes a tier-2 change to `none/inline` — the exact case FTD hard-enforces at push.
egm-inventive-catalyst's pattern (Intermediary): the CLI derives objective fields from
`git diff` / repo manifest instead of trusting agent prose. Related circularity
(egm-implementation-readiness): `effort_tier` appears as both agent input and Clef
output — the schema must settle as a versioned contract where the agent reports
observable facts and Clef outputs the tier + method classification.

### 4. Clef endpoint failure has no declared semantics (all 7 lenses)

The single most converged finding. Every work unit would gate on a network call with
quotas, latency, and outages; the concept specifies neither degraded mode (cached last
decision, deterministic local heuristic, fail-closed with user prompt) nor who is
accountable for a misroute surfacing units later. Related: provider-agnosticism hides
a Clef-shaped interface (typed questions, single-pass probabilities) that generic LLM
providers do not honor — "provider" must be defined as Clef-compatible endpoints or
the abstraction owns a calibration path; and per-project provider switches can flip
the method for the same request (determinism stance needed).

### 5. Routing decisions need an append-only record, not an Engram upsert (egm-codebase-details, egm-implementation-readiness)

Engram upserts overwrite: re-routing the same change erases the prior decision and its
probability margin, and the override ledger — the most important audit record in
phase 1 — would live in the system's weakest store. A `router/{slug}/decision` history
should be append-only (JSONL, file-based, like `.rsc/`), with Engram as the search
mirror only. Correlation requires a request ID spanning pre-flight → Clef call →
method execution, plus criteria-version capture so historical behavior is reproducible
after a CRITERIA.md edit.

### 6. Oscillation without hysteresis (4 lenses)

Re-evaluating before every unit with bidirectional switching invites a borderline
change to flip methods across units — the exact migration cost the design avoids.
egm-inventive-catalyst's pattern (Dynamics with Bounds): sticky routing — escalate
immediately on any high-risk read, de-escalate only after N consecutive low-risk
reads. Related: phase 1 decision fatigue (ten confirmations for a ten-task change) —
confirm only when the recommendation changes; and the routing state itself (current
method, dwell count, iteration classification) needs a named store.

### 7. Scope: two products riding as one (4 lenses)

The Engram unification (analysis §7) is well-scoped, incremental, and independently
shippable; the router is speculative and — per the analysis's own ordering — comes
last. Bundling them inverts the order and blocks the unification on the riskier
deliverable. Related scope notes: gate method availability on adapter completion
(routing to `asdd` before its Engram adapter lands reads degraded state); the
goldfish reviewer-blind rule must be enforced mechanically (memory tools stripped
from reviewer tool surfaces, not forbidden in prose); enforcement state (`.rsc/`
ledger) stays out of Engram at concept level; and whether tier-0/inline work owes the
full dual-write tax is open (egm-inventive-catalyst's Segmentation pattern).

### Smaller findings

- Method identifiers need one canonical scheme (`genlte-sdd` on disk vs `gentle-sdd`
  in prose; `ftd` vs `rsc-ftd`) and a method registry (id, criteria ref, naming table)
  instead of a hard-coded enum.
- The CLI needs a `doctor` command (provider reachable, schema valid, criteria
  parseable) and a defined blocking-vs-advisory stance — advisory suggestions nobody
  can reject measure nothing.
- Pre-flight payloads leak uncommitted intent to a third party; the schema shape
  (digests vs raw summaries) is a privacy decision belonging now.
- EGM routing granularity: `egm-review-loop` is one phase of a larger method; routing
  into it without `egm-draft` upstream is undefined.

## Author Rulings (2026-11-10, after the Goldfish round)

1. **Between-methods routing is a new layer.** Existing classifiers (`risk.js`,
   `recommend.js`, `model-routing.md`, FTD's decisor) operate inside the flow of an
   already-chosen method and are blind to other methods — they are not baselines for
   this router and no precedence conflict exists. Analysis §1's "beat the baseline"
   requirement is reframed: phase 1's suggestion-vs-choice measurement IS the
   baseline-building mechanism (the baseline today is the user choosing manually).
   The rubric alternative stays in Alternatives Weighed as the degradation path —
   CRITERIA.md already contains its material.
2. **Routing is one-shot; no mid-flight switching in any direction.** Resolves the
   de-escalation-with-open-authority findings (Analysis §2), the oscillation findings
   (§6), and most of the migration debt: the method is decided before artifacts
   exist, and continuations of a decided change are recognized and keep their
   method. The handoff-stub proposal is unnecessary for v1.
3. **Endpoint failure: retry, then agent + user decide together with evidence.**
   The CLI retries the Clef call; if still unavailable, the agent presents the
   pre-flight schema and the compared CRITERIA to the user and they pick the method
   jointly, with justification recorded like any routing decision. No silent
   heuristic fallback, no hard block.

## Round 2 Review Outcomes (2026-11-10, concept v2)

All seven lenses re-reviewed the revised concept. Outcome against round 1:

### Resolved by the rulings (round-1 findings that died)

Mid-flight switching semantics, oscillation/hysteresis, de-escalation stranding
authority artifacts, and the handoff-stub necessity — all eliminated by one-shot
routing (Ruling 2). Silent fallback on endpoint failure — replaced by retry → joint
agent+user decision with evidence (Ruling 3).

### Surviving findings (convergent in round 2 — the open design debt)

1. **Continuation detection is the load-bearing undefined mechanism (7/7 lenses).**
   "Clef knows" has no signal, no store, no error budget, no audit trail; a misread
   continuation silently bypasses routing forever. Candidate: durable local routing
   state (`router/{slug}/decision` keyed by change, checked before any classification
   call).
2. **Self-attested schema + tier circularity (5/7).** `effort_tier` is both agent
   input and Clef output; a lazy `size_class: small` routes tier-2 work to inline.
   Candidate: split the schema — CLI derives observable facts from `git diff`/repo
   manifest (input); Clef owns the classification, tier + method (output); the agent
   contributes only `intent`/`scope_summary`.
3. **Append-only decision record (5/7).** Override ledger and probability margins in
   an upsert store are erasable and unreproducible after a CRITERIA.md edit.
   Candidate: append-only JSONL as authority (request ID, criteria hash,
   provider/model, `decided_by`, probabilities); Engram as search mirror only.
4. **One-shot lock-in has no escape hatch (2/7, HIGH).** A misroute discovered at
   unit 3 of 10 has no exit; restarting as "new work" may itself be classified a
   continuation and re-lock the same method. Candidate: user-override re-route with
   recorded justification.
5. **Measurement validity (5/7).** Agreement ≠ correctness; there is no ground
   truth; degraded-mode decisions pollute the baseline. Candidate: shadow rubric in
   phase 1 — record the triple (rubric, Clef, human) per decision; pre-declare the
   metric/threshold that would retire Clef in favor of the rubric.
6. **Provider contract unstated (5/7).** "Provider-agnostic" vs Clef-shaped interface
   is unresolved; no determinism stance (criteria+provider+model version recorded per
   decision). Candidate: written endpoint spec (typed questions in, calibrated
   probabilities out, criteria hash) before any code.
7. **MEDIUM cluster:** schema `schema_version` + privacy shape (digests vs raw
   payloads); canonical method registry (`genlte-sdd` is a typo frozen into a path;
   `egm-review-loop` entry contract without `egm-draft` upstream); method-availability
   gating (route only to methods with completed adapters); `doctor`/`explain`/
   dry-run tooling; outage ergonomics (per-session memoization of the joint
   decision); scope sequencing (unification ships first — already the analysis's
   order).

### Standing author decisions (the seven round-2 questions, consolidated)

RESOLVED 2026-11-10 — see Author Rulings II below (rulings 4–7).

## Author Rulings II (2026-11-10, post round 2)

**4. Classification axis is complexity only.** The router classifies expected
duration, step count, breadth of touched areas, and context-degradation risk for the
agent — never criticality. `risk_tier` is removed from the router vocabulary; risk
gates stay inside each method's own flow after selection (ASDD tier-2 panel, FTD
risk loop). Examples: style tweak → ODD; full redesign → heavier; simple auth bug →
ODD; auth UI+backend integration → SDD.

**5. Escape hatch is user-invoked re-route only.** At any time the user may re-route
a change; the CLI records `superseded_by` + justification append-only, and the new
method applies from that moment. The router has no automatic escalation or
 de-escalation triggers; the user holds decision power.

**6. Continuation detection is deterministic and git-anchored.** Routing state is
tracked in the repo at `.router/open/<slug>.json` — facts only: slug, method,
decided_at, request_id, provider, model, criteria_hash, decided_by, payload_hash
(no prose payloads in git). The CLI finds open changes from the working tree plus
`git log --diff-filter=A -- .router/open/` reachable from HEAD. The agent selects
the slug from a CLI-provided candidate list or declares `new`; free-text slugs are
never accepted. Continuations never reach Clef. Closure commits `status: closed`.

**7. Decision records.** `.router/history/<slug>.jsonl` (tracked, append-only)
records every event with `decided_by ∈ {clef, joint_fallback, override_re_route}`;
Engram `router/{slug}/decision` is a search mirror only, never authority. Clef is
the primary classifier — classification must not depend on agent context or bias
(contaminated context, malformed sends, per-method bias are agent-side risks); the
rubric material in CRITERIA.md is the degradation path only. Phase-1 action
threshold: sustained override rate > 30% with a consistent-method pattern → revise
that method's CRITERIA.md (refine the criterion, not retire Clef).

## Author Rulings III (2026-11-10, post round 3)

**8. Reconciliation is human-decided.** When working tree, `history/<slug>.jsonl`,
and git reachability disagree, the CLI stops, presents the evidence (the three
sources side by side), and the user decides once; the resolution is recorded in
history. No silent auto-reconciliation.

**9. Routing state is keyed by branch first, then by a minted ID.** Path:
`.router/open/<branch>/<routing_id>.json`, where `routing_id` is a UUID minted by
the CLI at decision time — it disambiguates a future branch reusing the same name.
Branch name is the strongest continuation signal; the slug remains a human-readable
label inside the record, never the key.

**10. There is no `none`.** The method set becomes `{odd, ftd, asdd, gentle-sdd,
egm-review-loop}`. ODD inline is the mandatory floor — every request has a method,
and the floor carries the harness's own risk line, so the low-complexity/high-
criticality blind spot of ruling 4 dissolves by design. The common criticality
checklist is dropped.

**11. Agent-initiated re-route request, with discard-and-restart.** The agent may
and should propose a re-route when observed scope outgrows the decided method
(evidence recorded, `requested_by: agent`). On user approval, the in-flight work is
discarded — not migrated — and the change restarts under the more robust method,
consistent with the no-migration principle.

## Round 3 Review Outcomes (2026-11-10, concept v3)

All seven lenses re-reviewed. The concept's core (one-shot routing, complexity axis,
Clef-primary, human final call) survived; the git-anchored continuation mechanism
(ruling 6) concentrated nearly all new HIGH findings.

### Finding 1 — the git anchor is the fragile part (7/7 lenses)

`git log --diff-filter=A -- .router/open/` reachable from HEAD keys identity on
commit events, and commit events are exactly what git rewrites:

- **Squash merges** (the default PR workflow) collapse add+close commits; orphaned
  open records land on main, or finished changes reappear as candidates forever.
- **Rebase/cherry-pick** rewrite or duplicate anchors — a named contradiction between
  binding (1) (no re-routing) and binding (6) (anchor in rewritable history).
- **Fresh clones / unpushed commits** make open changes invisible → silent `new`
  declaration and duplicate routing.
- **Worktrees** split working tree (local) from history (shared); precedence is
  undefined. Two parallel open slugs on one branch have no deterministic resolution;
  the agent picking a slug is the same bias ruling (7) excludes from classification.

**Convergent candidate resolution (to adopt in drafting):** detection reads STATE,
not commit events — working-tree presence of `.router/open/<slug>.json` plus the
slug's own open/closed status is authoritative; `history/<slug>.jsonl` wins on any
disagreement; git history is corroboration only. Schema additions: owning branch/ref,
`decided_by` gains a human-confirmed value, `criteria_hash` pinned as historical fact
(never a re-evaluation trigger), `payload_hash` purpose defined or dropped. Ops
surface: `router status` reconciliation command, force-close stale slug operation,
CLI-issued `request_id` as integrity marker, cold-start diagnostic, validation rule
rejecting `.router/` edits without a lifecycle event, and a lighter-layout option
(append-only history + pointer file) if commit churn hurts.

### Finding 2 — the complexity axis has two blind spots (6/7 lenses)

- **Low-complexity / high-criticality:** `none` has no internal gate — binding (4)'s
  "gates stay inside each method" fails precisely at the lowest method. Candidate
  fix: a mandatory minimal criticality checklist common to ALL methods (one line:
  touched_areas against a guardrail list → forces a review step post-hoc), or an
  explicit criticality floor added to pre-flight (deliberate exception to ruling 4).
- **High-complexity / low-risk:** mechanical refactors (200-file rename) inherit
  design ceremony they don't need; the agent rationally mis-reports to escape it,
  polluting the override metric. Candidate fixes: methods may internally downgrade
  ceremony for high-confidence mechanical work; CRITERIA.md distinguishes mechanical
  vs design complexity; and an Inversion pattern — the agent may *request* a re-route
  (with evidence) subject to user approval, instead of being structurally forbidden
  from surfacing drift.

### Measurement refinements (carried from round 2, sharpened)

`decided_by` must distinguish "Clef suggested, human approved" from "Clef decided"
(that distinction IS phase 1's measurement); stratify the 30% override metric by
`decided_by` and separate input error from classifier error (post-hoc actuals vs
declared); minimum sample floor + cooldown before any CRITERIA revision; record the
joint-fallback inputs and the Clef request/trace ID.

### Standing author decisions after round 3

1. The reconciliation rule: which source is authoritative when open-state, history,
   and working tree disagree (lenses propose: history file wins, working-tree
   presence required, git = corroboration).
2. Slug scoping: global vs branch-scoped, and the owning-branch field.
3. The `none`-method criticality floor: minimal common checklist vs pre-flight
   criticality signal vs accept the hole.
4. Agent-initiated re-route request: adopt the Inversion channel or keep user-only.

## Goals vs. Non-Goals

* **In-Scope Goals:**
  - Provider-agnostic Go CLI in `/src/loom-cli` with JSON output; Cloudflare first;
    provider add/remove via per-project config.
  - Pre-flight schema filled by the agent; CRITERIA.md per method bootstrapped from
    each method's own NOT-statements and sent to Clef with every request.
  - Method routing one-shot at request time; continuations of a decided change are
    recognized and keep their method (no mid-flight re-routing, no artifact
    conversion).
  - Human final call in phase 1; suggestion-vs-choice measured into Engram.
  - Engram unification of the four methods (gentle's contract, hybrid always).
* **Out-of-Scope Non-Goals:**
  - Artifact conversion tools between methods.
  - Fully delegated routing (no human gate) — phase 2, only if phase 1 measurements
    justify it.
  - Replacing ASDD/FTD's internal enforcement (risk tiers, spec gate, ledger) — the
    router selects methods; existing gates stay.
  - Mid-flight method switching in any direction — routing is one-shot per new work
    request; continuations of a decided change keep their method.
  - A generic-LLM adapter with its own calibration path (v1 providers are
    Clef-compatible endpoints only — to be confirmed).

## Alternatives Weighed

- **Cloudflare Worker as the router** — Rejected by author: the CLI must consume any
  Clef endpoint; Cloudflare is one provider, not the architecture.
- **Upfront ODD-style pre-flight gating every request** — Rejected during drilling:
  taxes trivial work ("fix a typo" pays a full explore); replaced by one-shot
  request-time routing (agent-filled schema) with continuation recognition — no
  per-work-unit re-evaluation.
- **Hash-based mirror reconciliation for the unified memory** — Rejected by author:
  gentle's honesty contract (never claim sync, report partial, preserve conflicts)
  worked in production; detection machinery added complexity without a proven need.
- **Engram-only or files-only memory for the unified layer** — Rejected: files-only
  loses cross-session recovery and search; Engram-only loses auditability and team
  sharing; ODD-pattern dual-write (file authority + mirror) chosen.
- **Migration/conversion of artifacts on method escalation** — Rejected by author:
  detect the method as early as possible instead; the chosen method runs its native
  flow from the decision moment. (Open design debt: see Analysis §2 — entry/exit
  state contract per method still required.)

## Key Decisions & Staging for Next Steps

- Go CLI in this repo, JSON output; provider pattern: Cloudflare first, per-project
  config, Clef-compatible endpoint contract.
- Divergence policy for the unified memory: gentle's five rules verbatim; hybrid
  always; no asking.
- Measurement: suggestion-vs-user-choice recorded by the harness skill into Engram;
  the CLI does not persist.
- The seven Goldfish questions below are the open design debt this brief hands to
  drafting; each maps to a CRITERIA/contract decision, not to code.

**Goldfish questions for the author (one per lens):**

1. egm-design: when re-evaluation de-escalates away from a method with authority
   artifacts mid-change — archived as record, blocked while gates are open, or
   inherited without migration?
2. egm-codebase-details: what is the append-only, human-auditable record of every
   routing decision and human override, if Engram upserts overwrite?
3. egm-comprehension: does an FTD feature doc with checked tasks and evidence survive
   escalation to ASDD as the spec input (per FTD's promotion rule), and does the
   `.rsc/review-ledger.jsonl` persist across the transition?
4. egm-implementation-readiness: does an in-flight change keep its original method
   when a provider switch or outage flips the recommendation — and what exact
   fallback does the CLI use?
5. egm-zoom-out: what concrete mistakes did the existing offline classifiers make
   that justify a network endpoint — and what measured error rate proves Clef better?
6. egm-structural: what minimum state does each method require to start from the
   decision moment, and who authors/versions CRITERIA.md against method drift?
7. egm-inventive-catalyst: when Clef is unreachable mid-change — block, cached
   decision, or local heuristic — and who is accountable for a late-surfacing
   misroute?
