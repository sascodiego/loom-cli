# Feature: drift fixes 2026-11-11

Closes the six divergences reported in `2026-11-11_DRIFT_REPORT.md` between the
documents and the shipped router. Two of them change recorded semantics and were
decided by the author: the code's `low|medium|high` tier vocabulary becomes the
contract, and `re-route` performs a real classification call. Branch
`fix/drift-fixes`, chained on `fix/review-warnings` (PR #1) because it touches the
same supersede sequence.

## Specs

- **S1** The tier vocabulary is `low | medium | high`, adopted from the
  implementation as the contract; the design's `trivial | small | medium | large`
  (`docs/designs/method-router.md:331`) is corrected to it. The vocabulary is
  declared once in the state package and validated on record write and read, so a
  value outside the enum fails loudly instead of persisting.
- **S2** `re-route` performs a **real classification call**: it takes the
  re-declared pre-flight (`--input`), runs the same derive + provider path as
  `decide`, and persists a fresh decision carrying the new `request_id`,
  `payload_hash`, `criteria_hash`, `tier`, `suggested_method` and
  `suggested_tier`. The recorded `method` stays the user-approved `--method` and
  `decided_by` stays `override_re_route`. A provider failure is exit 3 and
  persists nothing: the classification runs before any record is touched.
- **S3** `CLOUDFLARE_ACCOUNT_ID` is read. When the configured endpoint contains
  the placeholder `{account_id}`, it is substituted from the environment;
  a placeholder with the variable unset is exit 2 naming the variable. An
  endpoint without the placeholder ignores the variable, so its absence is not an
  error.
- **S4** Config loading lives in `router/internal/state/config.go`, the module
  boundary the design's M1 names; `state/detect.go` keeps only detection.
- **S5** The root README's contract-copy claim names the three packages that
  carry a copy (`rsc-ftd`, `rsc-asdd`, `egm`) and says `gentle-sdd` keeps its own
  Engram convention unchanged, matching `docs/engram-contract.md`.
- **S6** The design's M4 test path is `router/cmd/router/main_test.go`; the
  `internal/cli/main_test.go` reference is corrected.
- **S7** `skills/method-router/SKILL.md` and `router/README.md` document the new
  `re-route` signature (with `--input`) and the `low|medium|high` vocabulary.
- **S8** Every existing test stays green; each new behavior is written test-first
  with RED observed before the implementation.

## Tasks

| ID | Task | Specs | Route | Commit |
| --- | --- | --- | --- | --- |
| T1 | Tier enum in `internal/state` (constants + validation on write and read) + tests | S1, S8 | inline | |
| T2 | `re-route` real classification: `--input`, derive + provider, fresh provenance, exit 3 on failure + tests | S2, S8 | worker | |
| T3 | `{account_id}` substitution in the endpoint + tests | S3, S8 | inline | |
| T4 | Split `internal/state/config.go` out of `detect.go` (behaviour-neutral) | S4, S8 | inline | |
| T5 | Docs: design corrections (tier enum, re-route, M4 path), root README, router/README, routing skill | S1, S2, S5, S6, S7 | inline | |

Delivery: `fix/drift-fixes` chained on PR #1's branch. Push and PR are the
human's decision; nothing merges without it. Forecast: ~350 authored lines, one
commit per task.

## Log

- **L1** (user, verbatim) `Drift 1 acepto tu recomendacion, drift 2: llamada real.
  Hacemos un ODD para los fixes`
- **L2** (decision) Drift 1: adopt `low|medium|high` and correct the design, not
  the reverse. Rationale: `CRITERIA.md`, the routing skill and the gate already
  speak in those terms; changing the code would rewrite the evidence path for no
  gain.
- **L3** (decision) Drift 2: `re-route` re-classifies for real. The recorded
  method remains the user-approved target, so the call supplies provenance and
  evidence, not authority — the user's choice is what the record says, and
  `suggested_method`/`suggested_tier` preserve what Clef answered.
- **L6** (evidence, close) Each drift in `2026-11-11_DRIFT_REPORT.md` mapped to its
  fix: (1) tier vocabulary → T1, design:336 corrected to `low | medium | high`;
  (2) `re-route` fresh decision → T2, design:246-253 and M4's reroute bullet now
  say it re-classifies, and the skill documents that `--input` carries the
  re-declared pre-flight; (3) `CLOUDFLARE_ACCOUNT_ID` → T3, read through the
  `{account_id}` endpoint placeholder; (4) `internal/state/config.go` → T4, the
  file now exists and `detect.go` keeps detection only; (5) root README copy
  claim → T5, names the three packages and states `gentle-sdd` keeps its own
  convention; (6) M4 test path → T5, `router/cmd/router/main_test.go`.
  Checks: `go vet ./...` clean, `go test -count=1 ./...` green in all six
  packages, real-binary runs for the placeholder (exit 2 without the variable,
  exit 3 with it, URL substituted) and for re-route's failure path (exit 3 after
  three attempts, predecessor open, history unchanged). A grep for the obsolete
  vocabulary over living docs returns only this document's own citation of it.
- **L5** (delivery) Slice boundary recorded: PR #1 holds commits 2fb9215 (supersede
  atomicity + stale-schema naming); this PR holds everything after it on
  `fix/drift-fixes`. The 427-line count reported against `main` includes PR #1's
  268, so this PR's own slice starts at ~159 lines.
- **L4** (decision) Branch strategy: `fix/drift-fixes` is chained on
  `fix/review-warnings` (PR #1) because both touch `reroute.go`; the PR will say so.
