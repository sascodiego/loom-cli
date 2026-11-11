# loom-cli

A portable extraction of four development methods plus the router that picks
between them. Nothing here is installed into any runtime: these are the
deliverables, and each package is meant to be consumed by whatever harness you
run.

## Repository map

| Path | What it is |
| --- | --- |
| `gentle-sdd/` | Gentle AI Spec-Driven Development (v3.7.0 extraction): the artifact-chain method, already Engram-native. `system/` is the portable, tool-agnostic system; `reference/` is the raw extraction organized by adapter, for diffing and provenance. |
| `rsc-asdd/` | Agent Spec-Driven Development: spec-that-proves-itself, review inside every task scaled by risk, decision queue for long autonomous runs. |
| `rsc-ftd/` | Fast-Track Development: one feature document, one minimal loop, one session. |
| `rsc-tools/` | The `rsc` CLI and its targets/schema that ASDD and FTD call for risk classification. |
| `egm/` | The Elephant-Goldfish Model: brainstorm, draft, review and resolve skills with fresh-context adversarial reviewers. |
| `router/` | The method router (Go): classifies a declared pre-flight with Clef and records the human-confirmed decision. |
| `skills/method-router/` | The harness skill the router needs: detect, pre-flight interview, evidence gate, Engram mirror, dispatch. |
| `docs/designs/` | The Reviewed design for the router and its EGM review artifacts. |
| `docs/analysis/` | The Engram unification analysis the design builds on. |
| `docs/engram-contract.md` | The canonical persistence contract shared by every package. |
| `odd/` | ODD feature documents and their tasks (`odd/tasks/`). |

## The five methods

Every request has a method; ODD is the floor, so there is no "none".

| Method | Serves | Entry artifact |
| --- | --- | --- |
| `odd` | Bounded changes and investigations that fit one session and one review sitting. | None: the request is the artifact. |
| `ftd` | An authorised change whose scope is settled, carried by one feature document. | Feature document in `02-DOCS`. |
| `asdd` | Long autonomous runs against a spec that proves itself, with per-task review and a decision queue. | `constitution.md` plus an approved spec. |
| `gentle-sdd` | An approved artifact chain: proposal → spec → design → tasks → apply → archive. | An OpenSpec-style change directory. |
| `egm-review-loop` | Document review by fresh-context adversarial auditors. Registered, `available: false` in v1. | A target document plus its standards. |

The classification axis is complexity only; criticality is handled inside each
method's own gates.

## How the router consumes the methods

`router/registry.json` enumerates all five methods with their criteria path,
skill path, Engram namespace and availability. At decision time the router:

1. validates the declared pre-flight against the repository itself;
2. reads each dispatchable method's `router/criteria/<method>/CRITERIA.md` and
   hashes the whole set into `criteria_hash`;
3. sends the declaration, the session facts and the criteria excerpts to Clef;
4. prints a gate-ready decision the human confirms.

The registry is the only place that binds a method name to its skill and its
criteria; the router never infers either.

## Memory

Every package mirrors its artifacts to Engram under a deterministic topic key,
with the files as the source of truth and the mirror as a search index. The
contract is `docs/engram-contract.md`, reproduced per package under
`<package>/system/skills/_shared/engram-contract.md`. The `rsc` ledger (`.rsc/`)
is never mirrored, and adversarial reviewers never call memory tools.

## Status

The router design is Reviewed (`docs/designs/method-router.md`). Implementation
progress and evidence live in `odd/tasks/method-router-v1.md`. Building and
running the router: `router/README.md`.
