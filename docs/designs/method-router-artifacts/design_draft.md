> **SUPERSEDED** — working copy del drafting, reemplazado por
> `docs/designs/method-router.md` (no incluye los fixes de la etapa de review).

# Method Router: Clef-Classified Work Routing for Gentle-Shell

*   **Author:** Diego Sasco
*   **Status:** Draft
*   **Date:** 2026-11-10

--------------------------------------------------------------------------------

## Overview

This design specifies a small Go CLI ("the router") and the memory contract that
lets an AI coding harness pick the right development method for each piece of work
automatically — with the human always holding the final call.

The harness (gentle-shell, a Pi-based coding agent) already carries four extracted
development methods: ODD (its own inline workflow), RSC-FTD, RSC-ASDD, and EGM. Each
method is a different amount of process: a checklist for a small change, a feature
document, a full spec-driven flow with review inside every task, or a multi-round
document review. Choosing between them by hand every time is inconsistent, and
choosing wrong in either direction costs real work: too little process on a long,
many-step change degrades the agent's context and ships unverified drift; too much
process on a trivial change buries the work in ceremony.

The router replaces that manual choice with a classification. When the user requests
work, the agent fills a short pre-flight (what the work is, where it touches), the
CLI sends it — together with each method's written criteria — to Clef, a decision
model served by Cloudflare that returns calibrated probabilities instead of prose.
The CLI prints the classification as JSON; the harness shows the suggestion and the
user confirms. From that moment the work runs under the decided method's complete
native flow. Every routing decision is written to tracked files in the repository
and mirrored into Engram, the harness's memory, under one unified contract shared
by all four methods.

The classification axis is deliberately narrow: how much planning and context
management the work needs — expected duration, step count, breadth of touched
areas. Not how critical the change is. A one-line fix to an auth handler and a
one-line copy edit both stay small; a full redesign of either is heavy. Methods
keep their own internal risk gates after selection; the router only measures size
of effort, because its purpose is to protect the agent's context, not to grade
danger.

## The Problem

An agent working in a coding harness must decide, for every request, how much
process to apply. There is no stable answer: the same harness serves typo fixes
and multi-day refactors. Today the choice is made by the user (who must remember
which method exists and when it pays off) or by the agent (whose judgment is
shaped by whatever is in its context, and who benefits from choosing less
process). Both routes fail in predictable ways: long autonomous runs start without
the planning structure that keeps them coherent, and trivial changes get wrapped
in ceremony until the user stops using the discipline at all.

The problem this design solves: make method selection (1) a fast, explicit step
with evidence the user can see and override, (2) based on the observable shape of
the work rather than on agent self-assessment, and (3) durable — the decision,
its inputs, and its probabilities are recorded so that phase 1 experience can be
measured and, later, delegation of the decision can be earned with data.

## Goals

*   **G1 — One-shot routing by complexity.** Each new work request is routed
    exactly once, on a complexity axis only (expected duration, step count,
    breadth of touched areas, context-degradation risk for the agent). The decided
    method runs to completion; continuations of a decided change are recognized
    and never re-routed.
*   **G2 — Provider-agnostic CLI.** A Go CLI in this repository, JSON output,
    consuming Clef-shaped endpoints. Cloudflare Workers AI is the first provider;
    providers are added or removed through per-project configuration.
*   **G3 — Classification independent of agent context.** The classification step
    is executed by Clef, not by the agent: agent context can be contaminated,
    incomplete, or biased toward less process. The agent reports what the work is;
    the model decides what it requires.
*   **G4 — Deterministic continuation.** Routing state lives in tracked files
    keyed by branch plus a CLI-minted routing ID; a request that continues an
    already-decided change keeps its method and never reaches the classifier
    again.
*   **G5 — Human final call, phase 1.** Every routing suggestion is presented to
    the user with its evidence. The user may accept, override, or ask for a
    re-route at any time; an approved re-route discards in-flight work and
    restarts under the more robust method (no artifact migration).
*   **G6 — Durable, auditable decisions.** Routing facts are tracked in the
    repository (`.router/`), every event is append-only in per-slug history
    files, and Engram mirrors the decisions for search. The measured baseline —
    suggestion versus user choice — decides later whether routing can be
    delegated.
*   **G7 — One memory contract.** All four methods write their working state
    through the same Engram contract (gentle's persistence contract: dual write,
    read-back, honest partial reporting; hybrid always), so the router reads one
    memory layer instead of four.

## Non-Goals

*   **NG1 — Mid-flight method switching.** No escalation or de-escalation while a
    change is in flight, in any direction. The remedy for a wrong decision is the
    user-invoked re-route with discard-and-restart, not conversion.
*   **NG2 — Artifact migration or conversion** between methods. Methods always
    run their native flow from the moment they are decided.
*   **NG3 — Fully delegated routing.** Phase 1 keeps the human in the loop.
    Automatic routing is a phase-2 decision that phase-1 measurements must earn.
*   **NG4 — Replacing methods' internal gates.** Risk tiers, spec gates, review
    ledgers, and push guards belong to each method's flow and stay there. The
    router selects; it does not enforce.
*   **NG5 — Generic-LLM providers.** V1 providers are Clef-compatible endpoints
    (typed questions in, calibrated probabilities out). An adapter that simulates
    calibration over chat models is out of scope.
*   **NG6 — Authoring the methods.** ODD, FTD, ASDD, and EGM already exist as
    extracted skill packages in this repository. This design adapts their
    persistence to the unified contract; it does not redefine their workflows.

## Open Questions

*   **OQ1 — Method registry and canonical IDs.** One registry file maps method id
    (`odd`, `ftd`, `asdd`, `gentle-sdd`, `egm-review-loop`) to criteria path,
    skill location, and Engram namespace. Resolves the `genlte-sdd` on-disk typo
    versus prose inconsistency. Exact schema to settle at implementation.
*   **OQ2 — `egm-review-loop` entry contract.** Routing into the review loop
    requires an upstream document (`egm-draft` output). Rule to define: either the
    loop is routable only with a document path argument, or it is excluded from
    v1 routing options.
*   **OQ3 — Method availability gating.** Whether the router may suggest methods
    whose Engram adapter is not yet implemented, and how that is surfaced.
*   **OQ4 — Payload privacy shape.** Whether `scope_summary` prose is sent raw to
    the Clef endpoint or as a digest plus structured facts; decides the schema
    before data accumulates.

## Critical User Journeys (CUJs)

### CUJ 1: New work request (happy path)

*   **Given**: A repository with the router installed and `.router/` initialized;
    no open routing record on the current branch.
*   **When**: The user asks the agent for work ("integrá auth de UI con backend").
*   **Then**: The agent fills intent and scope; the CLI derives observable facts,
    calls Clef, prints JSON with tier + method + probabilities; the harness shows
    the suggestion with evidence; the user confirms; `.router/open/<branch>/`
    and `.router/history/` are written; the harness mirrors the decision into
    Engram; the method's native flow starts.

### CUJ 2: Continuation of a decided change

*   **Given**: An open `.router/open/<branch>/<routing_id>.json` for the current
    branch and an already-decided method.
*   **When**: The user asks for more work on the same change.
*   **Then**: The CLI presents the candidate record; the agent confirms the
    continuation; no Clef call happens; work continues under the decided method.

### CUJ 3: Endpoint failure

*   **Given**: The configured Clef endpoint is unreachable or erroring.
*   **When**: A new work request is routed and the call fails after retry.
*   **Then**: The CLI reports the failure; the agent presents the pre-flight and
    compared criteria to the user; they pick the method jointly; the decision is
    recorded with `decided_by: joint_fallback` and mirrored like any other.

### CUJ 4: Agent-initiated re-route

*   **Given**: Work routed to `odd` that has grown to many steps across several
    subsystems.
*   **When**: The agent emits a re-route request with evidence, and the user
    approves.
*   **Then**: The current record is superseded (`decided_by: override_re_route`,
    `requested_by: agent`); in-flight artifacts are discarded; the change restarts
    under the more robust method with a fresh decision.

## Technical Plan (Architecture)

Four components, three of them new and one adapted:

**1. The CLI (`router`, Go, this repository).** Single binary, JSON output on
stdout, human text on stderr. Subcommands:

*   `router detect` — continuation check. Reads `.router/open/<branch>/` in the
    working tree (state is authority), lists candidate records for the current
    branch, prints `{candidates: [...], verdict: "continuation" | "new"}`. Never
    calls the network.
*   `router decide` — classification. Input: pre-flight JSON (agent prose fields +
    CLI-derived facts). Loads criteria files, builds the Clef request (state +
    typed questions: `effort_tier` enum, `method` enum), calls the configured
    provider, prints `{tier, method, probabilities, request_id, provider, model,
    criteria_hash}`.
*   `router record` — persists the decision: writes
    `.router/open/<branch>/<routing_id>.json` (facts only, including
    `payload_hash` of the prose payload) and appends the first event to
    `.router/history/<routing_id>.jsonl`. Mints `routing_id` and `request_id`.
*   `router re-route` — supersedes an open record (`superseded_by`, justification,
    `requested_by: agent | user`), appends history, emits the new decision.
*   `router close` — marks `status: closed` at ship; appends history.
*   `router status` — reconciliation surface: open records versus working tree
    versus history discrepancies, provider health, last-success timestamp.
*   `router doctor` — provider reachable, schema valid, criteria parseable.

**2. Provider interface.** One Go interface:
`Decide(ctx, state, questions) (Decision, error)`. V1 implementation:
`cloudflare` (Workers AI REST, `@cf/cloudflare/clef`). Active provider, model,
and endpoint come from per-project config (`.router/config.json`, credentials
from environment). Adding a provider = implementing the interface; removing =
deleting its config entry. The interface contract is the Clef shape: typed
questions in, calibrated probabilities out.

**3. Criteria files.** One `CRITERIA.md` per method, in plain English: what work
the method is worth its ceremony for, expressed on the complexity axis (duration,
steps, breadth, context load), bootstrapped from each method's own NOT-statements
and owned with the method package. The CLI hashes them (`criteria_hash`) and
sends their content with every `decide` call.

**4. The harness skill (`routing` skill in gentle-shell).** Owns everything the
CLI must not do: pre-flight interview with the user, agent prose fields, showing
the suggestion with evidence, the confirmation gate, the joint fallback after
CLI-reported endpoint failure, and the Engram mirror
(`router/{slug}/decision`, upsert — a search index, never the authority).

```text
user request
     │
     ▼
harness routing skill
     │  router detect ──► continuation? ──► keep method, run native flow
     │        │ new
     │        ▼
     │  agent: intent + scope_summary
     │  CLI: derives observable facts (touched areas, diff stats)
     │        │
     │        ▼
     │  router decide ─► provider ─► Clef endpoint
     │        │              └─ failure: retry ─► joint agent+user decision
     │        ▼
     │  suggestion + probabilities ─► user confirms / overrides
     │        │
     │        ├─ router record ─► .router/open/ + .router/history/  (authority)
     │        ├─ harness skill ─► Engram router/{slug}/decision    (mirror)
     │        └─ dispatch ─► method native flow (odd | ftd | asdd |
     │                                             gentle-sdd | egm-review-loop)
     ▼
work runs ─► ship ─► router close ─► status: closed
```

**Pre-flight schema (v1 contract).** Split by author:

*   Agent prose: `intent`, `scope_summary`.
*   CLI-derived facts: `touched_areas` (paths), `diff_stats` (files, insertions,
    deletions), `size_class` (from diff thresholds).
*   Clef output: `effort_tier` (trivial | small | medium | large) and `method`.
    The agent never reports a tier; that circularity is closed by construction.

**Method dispatch.** After the user confirms, the harness starts the decided
method's own skill flow. `odd` is the floor: ODD inline with its mandatory risk
line. Dispatch contracts per method (what each needs on entry) are defined in
the method packages and referenced from the method registry (OQ1).

## Alternatives Considered

### Alternative A: Cloudflare Worker as the routing service

*   **Pros**: Central deployment; no local binary to distribute; updates roll out
    to every consumer at once.
*   **Cons**: Couples the router's existence to one vendor; adds a deployed
    service to own and operate; routing breaks for offline or self-hosted setups.
*   **Reason for rejection**: Rejected by author — the CLI must consume Clef from
    any provider. Cloudflare is one provider (always supported), not the
    architecture. The worker-shaped deployment was never the requirement.

### Alternative B: Full pre-flight exploration before every request

*   **Pros**: Maximum information at decision time; the agent reads the codebase
    before the method is chosen.
*   **Cons**: Taxes every request with an explore pass — a typo fix pays a full
    investigation to learn it is a typo fix; latency and token cost on the
    smallest, most frequent work.
*   **Reason for rejection**: Rejected by author during brainstorming — replaced
    by one-shot request-time routing with a lightweight schema and deterministic
    continuation recognition. The exploration budget belongs to the decided
    method's own flow, not to the router.

### Alternative C: Deterministic rubric as the primary classifier

*   **Pros**: Offline, zero latency, fully debuggable; criteria material already
    exists in CRITERIA.md; no account or network dependency.
*   **Cons**: Static thresholds cannot match prose criteria against prose
    pre-flights; every criteria refinement is a code change; no calibrated
    confidence to measure; executed over inputs the agent shaped.
*   **Reason for rejection**: Rejected by author — classification must not depend
    on agent context, bias, or malformed self-report; Clef exists precisely to
    take that judgment out of the agent's hands. The rubric's material survives
    as the degradation path (joint agent+user decision on endpoint failure) and
    CRITERIA.md remains the tuning surface: a sustained override rate above 30%
    with a consistent-method pattern triggers a criteria revision, not a
    classifier swap.

### Alternative D: Bidirectional mid-flight escalation/de-escalation

*   **Pros**: Adapts to discovered complexity without user intervention; matches
    the intuition that a method can "grow" with the work.
*   **Cons**: Oscillation without hysteresis; de-escalation strands authority
    artifacts (ASDD tier-2 gates legally require wiki and ledger); every switch
    is a migration problem; a misread continuation locks the wrong method.
*   **Reason for rejection**: Rejected by author (Ruling 2) — routing is one-shot
    per new work request; continuations keep their method. The remedy for a
    wrong decision is the user-approved re-route with discard-and-restart.

### Alternative E: Hash-based mirror reconciliation for the unified memory

*   **Pros**: Catches stale Engram mirrors mechanically; no reliance on agent
    honesty for mirror maintenance.
*   **Cons**: Adds detection machinery gentle-ai never needed; its contract
    (never claim sync, report partial, preserve conflicts, escalate by decision)
    already covered the failure modes in production.
*   **Reason for rejection**: Rejected by author — gentle's persistence contract
    is adopted verbatim instead. Honesty at write time replaces detection after
    the fact.

### Alternative F: Engram-only or files-only unified memory

*   **Pros**: One store — simpler mental model; no dual-write discipline.
*   **Cons**: Engram-only loses git auditability, team sharing, and iteration
    history (upserts overwrite); files-only loses cross-session recovery,
    compaction survival, and semantic search — and starves the router of a
    readable state layer.
*   **Reason for rejection**: Rejected by author — the ODD pattern (file
    authority + Engram mirror) with gentle's hybrid contract is the only option
    that serves both the team and the agent, and the router's state builder
    depends on it.

### Alternative G: Agent self-reported effort tier

*   **Pros**: Zero derivation code; the agent knows its own plan best.
*   **Cons**: The agent that benefits from a lighter method grades its own
    homework; `effort_tier` appeared as both input and output — a circular
    contract that corrupts the phase-1 baseline (five of seven review lenses
    flagged it).
*   **Reason for rejection**: Rejected by author — the schema is split: the agent
    reports prose (intent, scope), the CLI derives observable facts, Clef owns
    the tier and method classification.

## Detailed Implementation

Everything below ships in this repository (`/src/loom-cli`). Implementation is
ordered: M1–M3 are the CLI, M4 criteria, M5 the harness skill, M6 the method
adaptations, M7 housekeeping, M8 docs. No production code is written during the
design workflow; this section is the plan the implementation will follow.

### M1 — CLI core: state store, detection, recording

*   **`router/go.mod`** (create) — Go module `loom/router`; declares the module
    and the single direct dependency set (HTTP client is stdlib; UUID minting via
    `crypto/rand`, no external UUID lib needed).
*   **`router/cmd/router/main.go`** (create) — binary entrypoint; parses the
    subcommand (`detect`, `decide`, `record`, `re-route`, `close`, `status`,
    `doctor`), dispatches to `internal/cli`, enforces the global output contract:
    JSON on stdout, human text on stderr, exit code 0 success / 2 usage / 3
    provider failure after retry / 4 state conflict requiring user resolution.
*   **`router/internal/state/open.go`** (create) — read/write of
    `.router/open/<branch>/<routing_id>.json`. Record schema (v1):
    `routing_id` (UUIDv4, CLI-minted), `slug`, `branch`, `method`, `tier`,
    `decided_at`, `request_id`, `provider`, `model`, `criteria_hash`,
    `decided_by` (`clef | joint_fallback | override_re_route`), `human_approved`
    (boolean — true only when the final method equals Clef's suggestion and the
    user accepted it; false is the phase-1 disagreement signal),
    `suggested_method`, `suggested_tier`, `payload_hash`, `status`
    (`open | closed | superseded`), optional `superseded_by` and
    `requested_by`. Fact: file presence + `status` field is the authority for
    "open"; git history is never consulted for liveness.
*   **`router/internal/state/history.go`** (create) — append-only JSONL writer for
    `.router/history/<routing_id>.jsonl`. Event schema: `timestamp`, `event`
    (`decided | confirmed | joint_fallback | re_route_requested | superseded |
    closed | reconciled`), `decided_by`, `human_approved`, `user_choice` (when it differs from the
    suggestion), full record snapshot, justification when present. Every writer path in the CLI funnels through this file — no
    other component writes `.router/`.
*   **`router/internal/state/detect.go`** (create) — `detect` logic: list
    `.router/open/<current-branch>/*.json` with `status: open`; validate each
    candidate (`request_id` present, `decided_by` valid, schema version known);
    print `{candidates, verdict}`. Zero network, zero git-log reachability —
    working tree only, per Ruling 8 (divergence between what the tree shows and
    what history says is resolved by the user through `router status`, not
    silently).
*   **`router/internal/state/config.go`** (create) — load/validate
    `.router/config.json`: `{provider, model, endpoint, criteria_paths?}`;
    credentials resolved from environment (`CLOUDFLARE_API_TOKEN`,
    `CLOUDFLARE_ACCOUNT_ID`) — never stored in tracked files.
*   **`router/internal/state/reconcile.go`** (create) — logic behind `router
    status`: compares open files versus history events versus working tree;
    reports each discrepancy class as a typed diagnostic (orphan open record,
    closed record still in `open/`, file deleted locally while history says
    open, criteria-hash of an open record older than current CRITERIA.md).
*   **`router/internal/cli/*.go`** (create, 7 files: `detect.go`, `record.go`,
    `close.go`, `reroute.go`, `status.go`, `doctor.go`, `common.go`) — one file
    per subcommand wiring state/provider packages; `common.go` carries shared
    flag parsing and the repo-root discovery (`git rev-parse --show-toplevel`).
*   **`router/internal/state/*_test.go`** (create) — table tests against real
    temp git repositories (created with `git init` in `t.TempDir()`): branch
    keying, same-name-branch reuse (different `routing_id`), orphan open file,
    `status: superseded` chains, close/reopen sequences, detect verdicts.

### M2 — Pre-flight schema and fact derivation

*   **`router/internal/preflight/schema.go`** (create) — schema v1 types with
    `schema_version` field: agent prose (`intent`, `scope_summary`), derived
    facts (`touched_areas []string`, `diff_stats {files, insertions, deletions}`,
    `size_class`), output slots (`effort_tier`, `method` — never present in
    input). Validation rejects unknown fields and empty prose.
*   **`router/internal/preflight/derive.go`** (create) — CLI derivation: `git
    diff --numstat` against the base ref for `diff_stats` and `touched_areas`;
    `size_class` from thresholds declared in config (so projects can calibrate).
    No LLM calls.
*   **`router/internal/preflight/schema_test.go`** (create) — validation,
    derivation against fixture repos, threshold boundary tests.

### M3 — Provider interface, Clef client, decide/status/doctor

*   **`router/internal/provider/provider.go`** (create) — `Decide(ctx, state,
    questions) (Decision, error)` interface; `Decision` carries per-option
    probabilities and the endpoint's request/trace ID.
*   **`router/internal/provider/request.go`** (create) — builds the Clef-shaped
    payload: `state` (pre-flight JSON + criteria excerpts) and typed questions
    (`effort_tier` enum, `method` enum) with option descriptions taken verbatim
    from CRITERIA.md.
*   **`router/internal/provider/cloudflare.go`** (create) — Workers AI REST
    client for `@cf/cloudflare/clef`; retry with backoff (2 attempts, per
    Ruling 3); typed failure envelope for the harness joint-fallback path.
*   **`router/internal/cli/decide.go`** (extends) — wires schema → request →
    provider → output JSON (`tier`, `method`, `probabilities`, `request_id`,
    `provider`, `model`, `criteria_hash`).
*   **`router/internal/cli/status.go` + `doctor.go`** (extend) — `status`: run
    reconcile, report provider health and last success; `doctor`: provider
    round-trip probe, config validation, criteria parse check.
*   **`router/internal/provider/cloudflare_test.go`** (create) — `httptest`
    server fixtures: success, quota error, timeout, malformed response; retry
    counting; request-shape golden test.

### M4 — Criteria files and method registry

*   **`router/registry.json`** (create) — canonical method registry: id →
    `{criteria_path, skill_location, engram_namespace, available: bool}` for
    `odd`, `ftd`, `asdd`, `gentle-sdd`, `egm-review-loop`. Resolves OQ1; the
    `available` flag implements availability gating (OQ3) until adapters land.
*   **`router/criteria/odd/CRITERIA.md`** (create) — ODD floor: when inline ODD
    is enough (bounded scope, single area, risk line sufficient).
*   **`router/criteria/ftd/CRITERIA.md`** (create) — bootstrapped from
    `rsc-ftd/system/skills/ftd/SKILL.md` frontmatter NOT-statements, rewritten
    on the complexity axis.
*   **`router/criteria/asdd/CRITERIA.md`** (create) — bootstrapped from
    `rsc-asdd/system/skills/asdd/SKILL.md`: long multi-step runs, cross-area
    integration, high context-degradation risk.
*   **`router/criteria/gentle-sdd/CRITERIA.md`** (create) — bootstrapped from
    `genlte-sdd/system/skills/_shared/sdd-orchestrator-sections.md` and the
    gentle skill descriptions: spec-driven work with per-task review.
*   **`router/criteria/egm-review-loop/CRITERIA.md`** (create) — document review
    work with an existing document argument; marked dependent on OQ2.
*   Criteria authoring rule: each file states (a) the work shapes the method
    serves, (b) explicit shapes it does NOT serve (the NOT-statements), (c) the
    entry artifact it requires. Hashed by the CLI at `decide` time.

### M5 — Harness routing skill (gentle-shell)

*   **`skills/method-router/SKILL.md`** (create) — the routing skill: Step 1
    `router detect`; continuation → confirm and dispatch; new → pre-flight
    interview (agent fills prose fields only), `router decide`, evidence
    presentation to the user, confirmation gate, `router record`, Engram mirror
    write, method dispatch. Includes the lossless evidence presentation format
    and the joint-fallback conversation script (Ruling 3).
*   **`skills/method-router/references/preflight-schema.md`** (create) — the
    schema v1 contract exactly as `internal/preflight` defines it, for the
    agent-facing documentation.
*   **`skills/method-router/references/engram-mirror.md`** (create) — mirror
    write contract: `mem_save(topic_key: "router/{slug}/decision", ...)`,
    upsert acceptable, never authority, `capture_prompt: false`.

### M6 — Method adaptations to the unified Engram contract

*   **Contract file** `engram-contract.md` (create, same content adapted per
    package naming table) copied to: `rsc-ftd/system/skills/_shared/`,
    `rsc-asdd/system/skills/_shared/` (directory created),
    `egm/system/skills/_shared/` (directory created). Content: gentle's
    persistence contract (five rules verbatim) + per-method naming table.
*   **`rsc-ftd/system/skills/ftd/SKILL.md`** (edit) — persistence section:
    after every feature-doc mutation, dual write (file +
    `ftd/{slug}/feature-doc` upsert) with read-back; partial report on
    one-sided failure.
*   **`rsc-asdd/system/skills/asdd/SKILL.md`** (edit) — dispatcher section:
    store mode fixed `hybrid`; artifact locators passed into every phase launch.
*   **`rsc-asdd/system/skills/{constitution,specify,plan,tasks,implement,ship}/SKILL.md`**
    (edit, 6 files) — phase persistence paragraphs: artifact upserts under
    `asdd/{change}/…`, decision entries `asdd/{change}/decisions/{id}`
    (`type: decision`), read-back, honest partial reporting. Ledger
    (`.rsc/`) explicitly excluded from Engram.
*   **`egm/system/skills/{egm-draft,egm-review-loop,egm-resolve}/SKILL.md`**
    (edit, 3 files) — persistence paragraphs (`egm/{doc-slug}/draft`,
    `/review/{round}`, `/resolution`) + the goldfish rule: reviewer subagents
    never call `mem_*`; the document reaches them via prompt/locator only.
*   **`genlte-sdd/`** — no changes; already Engram-native.

### M7 — Naming consolidation

*   **Rename `genlte-sdd/` → `gentle-sdd/`** (directory move via `git mv`) —
    fixes the typo frozen into the path; the router registry and criteria
    reference the canonical name from day one.
*   **Reference updates** (edit): `genlte-sdd/README.md` path references, and
    any cross-references in `engram-unification-analysis.md` and
    `concepts_brief.md` headers (historical documents keep prose mentions; only
    living path references update).

### M8 — Documentation

*   **`router/README.md`** (create) — CLI usage, config schema, exit codes,
    provider onboarding (Cloudflare), the `.router/` layout, phase-1 measurement
    explanation.
*   **Root `README.md`** (create or extend) — repository map: the four method
    packages, `rsc-tools/`, `router/`, and how the router consumes criteria from
    the registry.
*   **`engram-unification-analysis.md`** (edit) — mark §7 items as superseded by
    this design where the design decides them (point to `design_draft.md`).

### Explicit non-items

*   No changes to `rsc-tools/` (the CLI reads nothing from it; ASDD's gates stay
    disk-native).
*   No skill installation into any runtime — `skills/method-router/` ships in
    this repository; installing it into gentle-shell is a separate, user-run
    step.
*   No provider abstraction beyond the single interface + one implementation
    (YAGNI; second provider lands when one exists).
