# Method Router: Clef-Classified Work Routing for Gentle-Shell

*   **Author:** Diego Sasco
*   **Status:** Reviewed (Stage 9 re-audit: READY WITH CONDITIONS — conditions resolved)
*   **Date:** 2026-11-10

--------------------------------------------------------------------------------

## Overview

This design specifies a small Go CLI ("the router") and the memory contract that
lets an AI coding harness pick the right development method for each piece of work
automatically — with the human holding the final call in phase 1, and
delegation of that call earned by measurement.

The harness (gentle-shell, a Pi-based coding agent) routes work to five
registered methods: `odd` (its own inline workflow and the floor), `gentle-sdd`
(its native spec-driven method), and three from extracted packages — RSC-FTD,
RSC-ASDD, and EGM. Four are routable in version 1; `egm-review-loop` ships
unavailable pending an entry contract. Each
method is a different amount of process: a checklist for a small change, a feature
document, a full spec-driven flow with review inside every task, or a multi-round
document review. Choosing between them by hand every time is inconsistent, and
choosing wrong in either direction costs real work: too little process on a long,
many-step change degrades the agent's context and produces results that drift
from the request without verification; too much process on a trivial change
adds overhead the work does not need.

The router replaces that manual choice with a classification. When the user requests
work, the agent fills a short pre-flight (a declared task definition: what the work
is, which areas it will touch, an estimated scope), the
CLI sends it — together with each method's written criteria — to Clef, a decision
model served by Cloudflare that returns calibrated probabilities instead of prose.
The CLI prints the classification as JSON; the harness shows the suggestion and the
user confirms. From that moment the work runs under the decided method's complete
native flow. Every routing decision is written to tracked files in the repository
and mirrored into Engram, the harness's memory, under one unified contract shared
by all routable methods.

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
the planning structure that keeps them coherent, and trivial changes carry process overhead
until the user stops using the discipline at all.

The problem this design solves: make method selection (1) a fast, explicit step
with evidence the user can see and override, (2) based on the observable shape of
the work rather than on agent self-assessment, and (3) durable — the decision,
its inputs, and its probabilities are recorded so that phase 1 experience can be
measured and, later, delegation of the decision can be earned with data.

## Goals

*   **G1 — One-shot routing by complexity.** Each new work request is routed
    exactly once, on a complexity axis of three measured dimensions: expected
    duration, step count, and breadth of the declared touched areas. Protecting
    the agent's context from long unstructured runs is why those three are the
    axis. The decided
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
    the user with its evidence. The user may accept, override at the
    confirmation gate, or request a re-route at any time; an approved re-route
    restarts the change under the target method the user approves (the agent's
    request proposes the target with evidence), and the discard of in-flight
    artifacts is performed by the user and their orchestration agent — the CLI
    only records the supersede event. No artifact migration.
*   **G6 — Durable, auditable decisions.** Routing facts are tracked in the
    repository (`.router/`), every event is append-only in per-slug history
    files, and Engram mirrors the decisions for search. The measured baseline —
    suggestion versus user choice — decides later whether routing can be
    delegated. The delegation criterion is pre-declared: at least 30 clean
    `decided_by: clef` decisions with a sustained agreement rate of 90% or
    higher and a stable criteria hash over the window. Below the sample floor,
    the rate is not reported at all.
*   **G7 — One memory contract.** All routable methods write their working state
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
*   **NG6 — Authoring the methods.** FTD, ASDD, and EGM already exist as
    extracted skill packages in this repository; ODD and `gentle-sdd` are the
    harness's native workflows. This design adapts the external packages'
    persistence to the unified contract; it does not redefine their workflows.

## Open Questions

All four questions from the first draft were resolved during review:

*   **OQ1 (method registry)** — resolved. `router/registry.json` schema defined
    in Detailed Implementation (M3), with canonical IDs and per-method values
    enumerated.
*   **OQ2 (`egm-review-loop` entry contract)** — resolved by design: the method
    ships `available: false` and is excluded from v1 dispatch until an entry
    contract exists.
*   **OQ3 (availability gating)** — resolved: registry `available` flags gate
    dispatch; a suggestion for an unavailable method is replaced by the
    next-best available method and the substitution is recorded in the decision
    event.
*   **OQ4 (payload privacy)** — resolved by author: the pre-flight prose
    (request summary, intent, scope summary) and the criteria content travel
    raw to the Clef endpoint. Recorded decision for phase 1 (single user; the
    endpoint declares zero data retention). Revisit if routing is shared with
    a team.

## Terminology

*   **Clef** — Cloudflare's decision model (see "Clef — definition and endpoint
    contract" in the Technical Plan). Consumed as an API-key-authenticated HTTP
    endpoint; returns calibrated probabilities, never prose.
*   **NOT-statements** — the boundary sentences in each method's own description
    ("NOT the full method, that is `asdd`"), written by the method authors;
    bootstrapping source for CRITERIA.md.
*   **Goldfish rule** — reviewer subagents never call memory tools; the document
    reaches them via prompt or locator only (preserves fresh-eyes review).
*   **Rulings** — the binding author decisions from the brainstorm, recorded in
    `docs/designs/method-router-artifacts/concepts_brief.md` under
    Author Rulings I–III. Rulings cited by number
    refer to that list.
*   **Phase 1** — the rollout stage where every routing suggestion requires user
    confirmation and suggestion-vs-choice is measured to decide whether
    delegation is earned.
*   **Gentle's persistence contract (the five rules)** — (1) reads go to resolved
    per-store locators, never freshest-copy fallbacks; (2) dual write with
    read-back; (3) non-atomicity declared honestly — report partial writes and
    name the outstanding locator, never claim sync; (4) conflicts are preserved
    in both stores and escalated only when they affect the next decision;
    (5) snapshots are recovery hints, never a second authority.

## Critical User Journeys (CUJs)

### CUJ 1: New work request (happy path)

*   **Given**: A repository with the router installed and `.router/` initialized;
    no open routing record on the current branch.
*   **When**: The user asks the agent for work ("Integrate auth UI with the
    backend").
*   **Then**: The pre-flight interview produces a task declaration (request
    summary, intent, scope summary, subsystems, scope estimate) that is shown
    to the user; the CLI sends the declaration plus criteria to Clef and prints
    JSON with tier + method + probabilities; the harness shows the suggestion
    with evidence; the user confirms; `.router/open/<branch>/`
    and `.router/history/` are written; the harness mirrors the decision into
    Engram; the method's native flow starts.

### CUJ 2: Continuation of a decided change

*   **Given**: An open `.router/open/<branch>/<routing_id>.json` for the current
    branch and an already-decided method.
*   **When**: The user asks for more work on the same change.
*   **Then**: The CLI presents the candidate record; the harness echoes the
    selected record (slug, method, decided_at) to the user for confirmation;
    no Clef call happens; work continues under the decided method.

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
    `requested_by: agent`); the discard of in-flight artifacts is performed by
    the user and their orchestration agent — the CLI never touches the working
    tree, it only records the supersede event; the change restarts under the
    user-approved target method with a fresh decision.

## Technical Plan (Architecture)

Four components, three of them new and one adapted:

**1. The CLI (`router`, Go, this repository).** Single binary, JSON output on
stdout, human text on stderr. Subcommands:

*   `router detect` — continuation check. Reads `.router/open/<branch-encoded>/`
    in the working tree (state is authority), lists candidate records for the
    current branch, prints `{candidates: [...], verdict: "continuation" |
    "new"}` — with multiple open records, `candidates` lists all and
    resolution follows Ruling 8. Never
    calls the network. The continuation verdict is computed by state alone:
    the agent is shown the candidate list (slug, method, decided_at) and either
    selects one — by its key, never free text — or declares `new`; declaring
    `new` while open records exist makes the CLI warn and, per Ruling 8, a
    disagreeing source is resolved by the user with evidence.
*   `router decide` — classification. Input: the declared pre-flight JSON
    (request summary, agent declaration, validated subsystems). Loads criteria
    files, builds the Clef request (state +
    typed questions: `effort_tier` enum, `method` enum), calls the configured
    provider, prints `{tier, method, probabilities, request_id, provider, model,
    criteria_hash, ambiguous}`. Mints `request_id` for this call.
*   `router record` — persists the decision: writes
    `.router/open/<branch-encoded>/<routing_id>.json` (facts only, including
    `payload_hash` of the exact pre-flight JSON sent to the endpoint) and
    appends the first event to
    `.router/history/<routing_id>.jsonl`. Mints `routing_id`; carries the
    `request_id` minted by `decide`. Branch names are filesystem-encoded in
    the directory component (`/` becomes `--`, so `fix/339-cedula` becomes
    `fix--339-cedula`); the true branch name is stored inside the record, so
    the encoding is lossless.
*   `router re-route` — supersedes an open record (`superseded_by`, justification,
    `requested_by: agent | user`) and emits the new decision. It **re-classifies**:
    it takes the re-declared pre-flight (`--input`), runs the same pipeline as
    `decide`, and records the fresh `request_id`, `payload_hash`, `criteria_hash`,
    `tier` and `suggested_*`. The recorded `method` stays the user-approved
    target — the call supplies provenance and evidence, not authority. A
    classification failure persists nothing and leaves the predecessor open.
*   `router close` — marks `status: closed` at ship; appends history.
*   `router metrics` — computes the phase-1 agreement rate from
    `.router/history/`: share of `decided_by: clef` decisions with
    `human_approved: true`, stratified by `decided_by` and by method;
    suppressed below the 30-decision sample floor.

(`router status`, `router doctor`, and the reconciliation engine are deferred
to v2 — see "Deferred to v2" in the Detailed Implementation.)

### Clef — definition and endpoint contract

Clef is an existing product, not a hidden deliverable: a Cloudflare decision
model (`@cf/cloudflare/clef`, hosted on Workers AI; open-weights release
available for self-hosting) that consumes a `state` plus a schema of typed
questions and returns a calibrated probability for every allowed option of
every question in a single forward pass. It generates no prose.

Binding v1 assumption: Clef is consumed as an **HTTP endpoint authenticated
with an API key**. The CLI owns the query; the adapter
(`internal/provider/`) is the single place that knows the endpoint shape,
auth scheme, and request/response mapping. Swapping providers means
implementing the adapter interface against a different endpoint; nothing else
in the design changes.

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
     │  agent declares: intent + scope + subsystems + scope estimate
     │  CLI: validates the declaration against repo structure
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

(`egm-review-loop` appears in the diagram for completeness; it ships
`available: false` pending OQ2 and is excluded from v1 dispatch.)

**Pre-flight schema (v1 contract).** The pre-flight is the definition phase:
at decision time nothing else exists — no diff, no work — so the input is the
declared task definition, shown to the user for approval before it is sent.

*   `request_summary` — the user's own words for the request (verbatim; the
    least bias-prone signal in the payload).
*   Agent declaration: `intent`, `scope_summary`, `subsystems` (areas the work
    will touch, validated by the CLI against the repository's top-level
    structure — a declared subsystem that does not exist is rejected with
    exit 2), `scope_estimate` (expected step count and session count,
    agent-declared).
*   Clef output: `effort_tier` (low | medium | high) and `method`.
    The agent never reports a tier. The vocabulary is closed: `internal/state`
    declares it once and validates it on record write and read, so a value
    outside it fails loudly instead of persisting.

Residual bias: the agent frames the declaration. Mitigations: the user sees
and approves the declaration at the confirmation gate before it reaches Clef,
and phase-1 disagreement is the measured signal that tunes CRITERIA.md. There
is no diff-based measurement at decision time; that pretense was removed.

**Operational rules.**

*   **Confidence policy**: if the top two methods sit within 0.05 probability,
    the CLI marks `ambiguous: true`; the harness presents both options to the
    user instead of a single suggestion.
*   **Unavailable-method rule**: if the top method is `available: false`, the
    CLI substitutes the next-best available method, marks
    `substituted: true`, and the harness shows the substitution.
*   **Concurrency assumption (v1)**: one routing session per working tree.
    Concurrent `record` calls create independent records distinguished by
    `routing_id`; ambiguity surfaces through `detect` and resolves per
    Ruling 8. No locking.

**Method dispatch.** After the user confirms, the harness starts the decided
method's own skill flow. `odd` is the floor: ODD inline with its mandatory risk
line. Dispatch contracts per method (what each needs on entry) are defined in
the method packages and referenced from the method registry.

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

## Non-Functional Requirements

*   **Security & Privacy**: pre-flight prose (`request_summary`, `intent`,
    `scope_summary`) and criteria content leave the machine only to the
    configured Clef endpoint — a recorded author decision for phase 1 (single
    user; the endpoint declares zero data retention; see OQ4 resolution).
    Credentials live in the
    environment, never in tracked files. `.router/` records carry facts and
    hashes only — no prose in git; `payload_hash` is the SHA-256 of the exact
    pre-flight JSON sent to the endpoint.
*   **Performance & Latency**: one network round trip per new work request (2
    retries, 3 attempts total — see the Technical Plan, Provider interface);
    continuation detection
    is fully offline. Criteria excerpts travel with every `decide` call;
    CRITERIA.md files must stay under 2 KB each (hard bound — the CLI rejects
    larger files at `decide` time).
*   **Rollout & Rollback**: method availability is gated by the registry
    `available` flags — the Engram adapters ship before any method is marked
    available. Disabling the router per project is removing
    `.router/config.json`;
    work then continues under the ODD floor. All routing state is plain tracked
    files — rollback is a `git revert` of `.router/`. Engram mirror records
    survive a revert as stale entries; per the persistence contract (rule 5)
    they remain recovery hints, never authority, and the history files stay the
    audit trail.

--------------------------------------------------------------------------------

## Detailed Implementation

Everything below ships in this repository (`/src/loom-cli`). Milestone order:
M1 (state core) -> M2 (pre-flight schema) -> M3 (criteria + registry) -> M4
(provider, decide, re-route, metrics) -> M5 (harness skill) -> M6 (method
adaptations) -> M7 (naming) -> M8 (docs). Each milestone leaves the tree
buildable and tested: M3 lands criteria and the registry before M4's `decide`
needs them. No production code is written during the design workflow; this
section is the plan the implementation will follow.

### M1 — CLI core: state store, detection, recording

*   **Repo housekeeping** (execute first) — `git mv genlte-sdd gentle-sdd` and
    update `gentle-sdd/README.md` self-path references. Everything downstream
    references the canonical name from birth.
*   **`router/go.mod`** (create) — Go module `loom/router`; stdlib-only for v1
    (UUID minting via `crypto/rand`; HTTP client via `net/http`).
*   **`router/cmd/router/main.go`** (create) — binary entrypoint; parses the
    subcommand (`detect`, `decide`, `record`, `re-route`, `close`), dispatches
    to `internal/cli`, enforces the output contract: JSON on stdout, human
    text on stderr, exit codes 0 (success), 2 (usage or validation error), 3
    (provider failure after retry). Exit 4 (state conflict) arrives with the
    deferred v2 reconciliation engine. Per-subcommand mapping: `detect`,
    `record`, `close` emit 2 on validation failures; `decide` emits 3 on
    provider failure after retry and 2 on validation; `re-route` emits 2 on
    closed or unknown records.
*   **`router/internal/state/open.go`** (create) — read/write of
    `.router/open/<branch-encoded>/<routing_id>.json`. Record schema (v1):
    `routing_id` (UUIDv4, CLI-minted), `slug`, `branch` (true name),
    `branch_encoded`, `method`, `tier`, `decided_at`, `request_id` (minted by
    `decide`), `provider`, `model`, `criteria_hash`, `decided_by` (`clef |
    user_gate_override | joint_fallback | override_re_route`), `human_approved`
    (meaningful only for `decided_by: clef` — true only when the final method
    equals the suggestion and the user accepted it), `suggested_method`,
    `suggested_tier`, `user_choice` (when it differs from the suggestion),
    `payload_hash` (SHA-256 of the exact pre-flight JSON sent to the endpoint;
    pre-flight prose is never persisted in tracked files), `status` (`open |
    closed | superseded`), optional `superseded_by` and `requested_by`.
*   **`router/internal/state/history.go`** (create) — append-only JSONL writer
    for `.router/history/<routing_id>.jsonl`. Events: `timestamp`, `event`
    (`decided | gate_overridden | joint_fallback | re_route_requested |
    superseded | closed`), `decided_by`, `human_approved`,
    `user_choice`, full record snapshot, justification when present. The
    `reconciled` event is v2-reserved; no v1 writer emits it. Every
    writer path funnels through this file — no other component writes
    `.router/`.
*   **`router/internal/state/detect.go`** (create) — `detect` logic: list
    `.router/open/<encoded-branch>/*.json` with `status: open` in the working
    tree; validate candidates (`request_id` present, `decided_by` valid,
    schema version known); print `{candidates, verdict}`. Working tree only;
    zero network, zero git-log reachability.
*   **`router/internal/state/config.go`** (create) — load/validate
    `.router/config.json`: `{provider, model, endpoint}`; credentials from
    environment (`CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID`), never
    tracked. Note: the size-class threshold fields from earlier drafts are
    removed along with `size_class` itself.
*   **`router/internal/cli/*.go`** (create, 4 files: `detect.go`, `record.go`,
    `close.go`, `common.go`) — one file per subcommand;
    `common.go` carries shared flag parsing and repo-root discovery
    (`git rev-parse --show-toplevel`). `decide.go` and `reroute.go` are
    created in M4, after their dependencies exist.
*   **`router/internal/state/*_test.go`** (create) — table tests against real
    temp git repositories (`git init` in `t.TempDir()`): branch keying and
    encoding (`fix/339-cedula` -> `fix--339-cedula`), same-name-branch reuse
    (different `routing_id`), orphan open file, superseded chains, close and
    reopen sequences, detect verdicts.

### M2 — Pre-flight schema v1 and declaration validation

*   **`router/internal/preflight/schema.go`** (create) — schema v1 types with
    `schema_version`: `request_summary`, `intent`, `scope_summary`,
    `subsystems []string`, `scope_estimate {steps, sessions}`. Output slots
    (`effort_tier`, `method`) never appear in input. Validation rejects
    unknown fields and empty prose.
*   **`router/internal/preflight/validate.go`** (create) — subsystem
    validation: every declared subsystem must match a top-level directory of
    the repository (`git ls-files | top-level directories`); mismatch is exit
    2. No diff derivation, no LLM calls, no network.
*   **`router/internal/preflight/schema_test.go`** (create) — validation,
    subsystem acceptance/rejection against fixture trees, empty-field tests.

### M3 — Criteria files and method registry (before `decide`, so the tree stays buildable)

*   **`router/registry.json`** (create) — canonical method registry, fully
    enumerated values, not just fields:
    *   `odd` -> criteria `router/criteria/odd/CRITERIA.md`, skill "harness
        native (ODD inline)", Engram namespace `odd`, `available: true`.
    *   `ftd` -> criteria `router/criteria/ftd/CRITERIA.md`, skill
        `rsc-ftd/system/skills/ftd`, namespace `ftd`, `available: true`.
    *   `asdd` -> criteria `router/criteria/asdd/CRITERIA.md`, skill
        `rsc-asdd/system/skills/asdd`, namespace `asdd`, `available: true`.
    *   `gentle-sdd` -> criteria
        `router/criteria/gentle-sdd/CRITERIA.md`, skill `gentle-sdd/` (native
        package), namespace `sdd`, `available: true`.
    *   `egm-review-loop` -> criteria
        `router/criteria/egm-review-loop/CRITERIA.md`, skill
        `egm/system/skills/egm-review-loop`, namespace `egm`,
        `available: false` (pending OQ2 entry contract; excluded from v1
        dispatch — recorded decision, closes the OQ2/OQ3 pre-emption finding).
*   **`router/criteria/<method>/CRITERIA.md`** (create, 5 files) — authoring
    rule: (a) work shapes the method serves, (b) explicit shapes it does NOT
    serve (the NOT-statements), (c) the entry artifact it requires, (d) the
    `asdd` and `gentle-sdd` files carry a mutual-exclusion section naming the
    other method and the boundary between them. Each file under 2 KB (hard
    bound enforced at `decide` time).
*   **`criteria_hash` definition** (normative): SHA-256 over the concatenation
    of the five criteria files in registry order — including the file marked
    `available: false`. Pinned here so two implementers produce identical
    hashes.
*   **`router/internal/registry/registry_test.go`** (create) — schema
    validation, availability gating (`available: false` methods are
    substituted and flagged), criteria parse and hash stability.

### M4 — Provider interface, `decide`, `re-route`, metrics

*   **`router/internal/provider/provider.go`** (create) — `Decide(ctx, state,
    questions) (Decision, error)`; `Decision` carries per-option probabilities
    and the endpoint's request/trace ID.
*   **`router/internal/provider/request.go`** (create) — builds the Clef-shaped
    payload: `state` (declared pre-flight JSON + criteria excerpts) and typed
    questions (`effort_tier` enum, `method` enum) with option descriptions
    taken verbatim from CRITERIA.md. Criteria excerpts are the full file
    content of the available methods (each under the 2 KB bound); unavailable
    methods contribute nothing.
*   **`router/internal/provider/cloudflare.go`** (create) — Workers AI REST
    client for `@cf/cloudflare/clef`; retry with backoff (2 retries, 3
    attempts total, per Ruling 3); typed failure envelope for the
    joint-fallback path.
*   **`router/internal/preflight/derive.go`** (create, replaces the earlier
    draft's diff derivation) — derives nothing from diffs; assembles the final
    payload from the validated declaration plus observable session facts that
    exist at request time: current branch, working-tree cleanliness summary
    (count of modified files only), criteria excerpts. This is the concrete
    answer to the empty-diff finding: the facts that exist at decision time
    are the declaration and the repo structure; nothing is measured that does
    not exist.
*   **`router/internal/cli/decide.go`** (create) — schema -> derive -> request
    -> provider -> output JSON (`tier`, `method`, `probabilities`,
    `request_id`, `provider`, `model`, `criteria_hash`, `ambiguous`,
    `substituted` where the availability rule fired).
*   **`router/internal/cli/reroute.go`** (create) — supersedes an open record;
    re-classifies the re-declared pre-flight (`--input`) so the new record carries
    fresh provenance; appends history; never touches the working tree beyond
    `.router/`.
*   **`router/internal/cli/metrics.go`** (create) — agreement-rate report from
    `.router/history/`: share of `decided_by: clef` with `human_approved:
    true`, stratified by `decided_by` and method. Clean = `decided_by: clef`
    at the current `schema_version`; gate overrides count in the denominator
    as disagreements (`human_approved: false`); `joint_fallback` and
    `override_re_route` are excluded from the denominator entirely and counted
    separately. Suppressed below the 30-decision sample floor.
*   **`router/internal/provider/cloudflare_test.go`** (create) — `httptest`
    fixtures: success, quota error, timeout, malformed response; retry
    counting; request-shape golden test.
*   **`router/internal/cli/decide_test.go`** (create) — output-shape golden
    test, gate-override path (`decided_by: user_gate_override`),
    ambiguous-marks, substitution rule.
*   **`router/cmd/router/main_test.go`** (create) — CLI-layer exit-code
    contract: per-subcommand 0/2/3 mapping including provider failure and
    validation failures.
*   **`router/internal/cli/e2e_test.go`** (create) — end-to-end CUJ coverage
    against a temp repo: decide → record → detect → close (CUJ 1/2), gate
    override (`user_gate_override`), re-route supersede (CUJ 4).

### M5 — Harness routing skill (gentle-shell)

*   **`skills/method-router/SKILL.md`** (create) — the routing skill,
    self-contained: Step 1 `router detect`; continuation -> user echo of the
    selected record -> dispatch; new -> pre-flight interview (request summary
    verbatim from the user, agent fills declaration fields), `router decide`,
    evidence presentation, confirmation gate, `router record`, Engram mirror,
    dispatch. Defines the **evidence presentation format** inline (so the
    skill is self-contained): the gate shows {suggested method + tier,
    probability per method option, the criteria excerpt that matched,
    declaration recap, confirm / override / re-route options} — nothing is
    shown that the record does not capture. Includes the joint-fallback
    conversation script (Ruling 3) and the ambiguity script
    (`ambiguous: true` -> both options shown).
*   **`skills/method-router/references/preflight-schema.md`** (create) — the
    schema v1 contract exactly as `internal/preflight` defines it.
*   **`skills/method-router/references/engram-mirror.md`** (create) — mirror
    write contract: `mem_save(topic_key: "router/{slug}/decision", ...)`,
    upsert acceptable, never authority, `capture_prompt: false`.

### M6 — Method adaptations to the unified Engram contract

*   **Canonical home**: the master contract lives at
    `docs/engram-contract.md` (create) — one file, adapted from gentle's
    persistence contract (five rules verbatim) plus the per-method naming
    tables. Copies go to `rsc-ftd/system/skills/_shared/`,
    `rsc-asdd/system/skills/_shared/` (directory created),
    `egm/system/skills/_shared/` (directory created). The master is the
    canonical home; copies carry a header pointing at it.
*   **`rsc-ftd/system/skills/ftd/SKILL.md`** (edit) — persistence section:
    after every feature-doc mutation, dual write (file +
    `ftd/{slug}/feature-doc` upsert) with read-back; partial report on
    one-sided failure.
*   **`rsc-asdd/system/skills/asdd/SKILL.md`** (edit) — dispatcher section:
    store mode fixed `hybrid`; artifact locators passed into every phase
    launch.
*   **`rsc-asdd/system/skills/{constitution,specify,plan,tasks,implement,ship}/SKILL.md`**
    (edit, 6 files; anchors: the persistence/artifact paragraphs of each
    phase) — artifact upserts under `asdd/{change}/…`, decision entries
    `asdd/{change}/decisions/{id}` (`type: decision`), read-back, honest
    partial reporting. The ledger (`.rsc/`) is explicitly excluded from
    Engram. Declared out of scope for v1 persistence edits: `clarify`,
    `analyze`, `review`, `verify`, `sdd-init` (their artifacts are covered by
    the dispatcher-level locators).
*   **`egm/system/skills/{egm-draft,egm-review-loop,egm-resolve}/SKILL.md`**
    (edit, 3 files) — persistence paragraphs (`egm/{doc-slug}/draft`,
    `/review/{round}`, `/resolution`) + the goldfish rule: reviewer subagents
    never call `mem_*`; the document reaches them via prompt/locator only.
    Out of scope, declared: `egm-review`, `egm-brainstorm`, `egm-drift-report`,
    `egm-bootstrap` (no persistence paragraphs in v1).
*   **`genlte-sdd/`** — no changes; already Engram-native.
*   **Verification** (M6 exit criteria): each edited SKILL.md names its topic
    keys exactly as the contract table defines; a grep checklist verifies
    topic-key conventions per package.

### M7 — Naming consolidation (verification only)

The rename itself executed in M1 (repo housekeeping). This milestone runs the
    closed-set reference check: `grep -rn genlte-sdd` over living files
    returns zero hits (historical documents — `concepts_brief.md`,
    `engram-unification-analysis.md`, the review reports — are exempt by
    declaration).

### M8 — Documentation

*   **`router/README.md`** (create) — CLI usage, config schema, exit codes,
    provider onboarding (Cloudflare), the `.router/` layout, the phase-1
    measurement and its delegation criterion (30+ clean decisions, >= 90%
    sustained agreement, stable criteria hash).
*   **Root `README.md`** (create) — repository map: the four method packages,
    `rsc-tools/`, `router/`, `docs/designs/`, and how the router consumes
    criteria from the registry.
*   **`docs/analysis/engram-unification-analysis.md`** (edit) — one supersession
    line at the top pointing at `docs/designs/method-router.md` (enumerated single edit;
    the body stays historical).

### Deferred to v2

`router status`, `router doctor`, and the reconciliation engine (land after
observed divergence, per the zoom-out YAGNI finding); exit code 4 (state
conflicts); `egm-review-loop` dispatch (pending OQ2 entry contract); any
second provider (lands when one exists). The deferred surface is the
explicit answer to the YAGNI finding: v1 ships `detect / decide / record /
re-route / close / metrics` and nothing else.