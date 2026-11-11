# Feature: method-router v1

Ships the `router` CLI (Go, stdlib-only) plus registry, criteria, harness routing
skill and the unified Engram-contract adaptations. Authoritative design:
`docs/designs/method-router.md` (Status: Reviewed; Stage 9 re-audit READY WITH
CONDITIONS — conditions applied). Milestones M1–M8, in that order; every
milestone leaves the tree buildable and tested.

## Specs

- **S1** Method floor is ODD, never `none`: "ODD is the floor — no `none`
  method". Registry order: `odd, ftd, asdd, gentle-sdd, egm-review-loop`.
- **S2** Routing is one-shot per new work request: "continuations keep method,
  never re-routed". Continuation detection reads working-tree state
  (`.router/open/`), never git commit events.
- **S3** Classification axis is complexity only: "complexity only, never
  criticality". Criticality stays inside each method's own gates.
- **S4** Pre-flight v1 is a declaration, not a diff: fields `request_summary`
  (user verbatim), `intent`, `scope_summary`, `subsystems []string`,
  `scope_estimate {steps, sessions}`. Output slots (`effort_tier`, `method`)
  never appear in input. Every declared subsystem must match a top-level
  repository directory; mismatch is exit 2. Validation rejects unknown fields
  and empty prose.
- **S5** State keying: `.router/open/<branch-encoded>/<routing_id>.json` with
  `/` → `--` (`fix/339-cedula` → `fix--339-cedula`) and a CLI-minted UUIDv4
  `routing_id`. Same branch name reused gets a distinct `routing_id`.
- **S6** Record schema v1 carries exactly: `routing_id`, `slug`, `branch`,
  `branch_encoded`, `method`, `tier`, `decided_at`, `request_id`,
  `provider`, `model`, `criteria_hash`, `decided_by` (`clef |
  user_gate_override | joint_fallback | override_re_route`), `human_approved`
  (meaningful only for `decided_by: clef`), `suggested_method`,
  `suggested_tier`, `user_choice`, `payload_hash`, `status` (`open | closed |
  superseded`), optional `superseded_by`, `requested_by`.
- **S7** Output contract: JSON on stdout, human text on stderr; exit 0 success,
  2 usage or validation error, 3 provider failure after retry. Per-subcommand:
  `detect`, `record`, `close` → 2 on validation; `decide` → 3 on provider
  failure after retry, 2 on validation; `re-route` → 2 on closed or unknown
  records. Exit 4 deferred to v2.
- **S8** Registry is enumerated, not schema-only: `odd`, `ftd`, `asdd`,
  `gentle-sdd` → `available: true`; `egm-review-loop` → `available: false`
  (excluded from v1 dispatch).
- **S9** `criteria_hash` = SHA-256 over the concatenation of the five criteria
  files in registry order, including the `available: false` file.
- **S10** Each `CRITERIA.md` is under 2 KB (hard bound enforced at `decide`
  time) and states: work shapes served, explicit NOT-statements, the entry
  artifact required, and for `asdd`/`gentle-sdd` a mutual-exclusion section.
- **S11** Confidence policy: top-2 within 0.05 → `ambiguous: true` with both
  options shown; unavailable method → next-best substitution with
  `substituted: true`.
- **S12** Provider retry: 2 retries, 3 attempts total, exponential backoff;
  typed failure envelope for the joint-fallback path.
- **S13** `router metrics`: agreement rate = share of `decided_by: clef` with
  `human_approved: true`, stratified by `decided_by` and method. Clean =
  `decided_by: clef` at the current `schema_version`; gate overrides count in
  the denominator as disagreements (`human_approved: false`); `joint_fallback`
  and `override_re_route` are excluded from the denominator and counted
  separately. Suppressed below the 30-decision sample floor.
- **S14** Every writer path funnels through `internal/state/history.go`; no
  other component writes `.router/`. History is append-only JSONL
  (`.router/history/<routing_id>.jsonl`), every event carries the full record
  snapshot; `reconciled` is v2-reserved and never emitted by v1.
- **S15** Unified Engram contract: master at `docs/engram-contract.md`, adapted
  from gentle's persistence contract with its five rules verbatim, plus
  per-method naming tables; copies in `_shared/` carry a pointer header. The
  `.rsc/` ledger is excluded from Engram. Goldfish reviewer subagents never
  call `mem_*`; the document reaches them via prompt or locator only.
- **S16** Rename `genlte-sdd/` → `gentle-sdd/` happens first, in M1
  housekeeping (`git mv`); M7 verifies `grep -rn genlte-sdd` over living files
  returns zero hits (historical documents exempt by declaration).
- **S17** v1 scope is exactly `detect / decide / record / re-route / close /
  metrics`; `status`, `doctor`, the reconciliation engine and exit 4 are
  deferred to v2.
- **S18** Evidence presentation gate shows: suggested method + tier,
  probability per method option, the criteria excerpt that matched, declaration
  recap, and confirm / override / re-route options — "nothing is shown that the
  record does not capture".
- **S19** Engram mirror write contract: `mem_save(topic_key: "router/{slug}/
  decision", ...)`, upsert acceptable, never authority, `capture_prompt:
  false`. The CLI never writes Engram; the harness skill does.

## Tasks

| ID | Task | Specs | Route | Commit |
| --- | --- | --- | --- | --- |
| T1 | Repo housekeeping: `git mv genlte-sdd gentle-sdd` + README self-path edits | S16 | inline | 2375225 (baseline) |
| T2 | `router/go.mod`, `cmd/router/main.go`, `internal/cli/common.go` — entrypoint, flag parsing, repo-root discovery, exit-code contract | S7 | inline | 2ecff7c |
| T3 | `internal/state/open.go` + `history.go` — record read/write, append-only JSONL, branch encoding, UUID minting | S5, S6, S14 | inline | 4c53dcd |
| T4 | `internal/state/detect.go` + `config.go` + `cli/{detect,record,close}.go` + state tests | S2, S5, S7 | inline | 3287031 |
| T5 | `internal/preflight/schema.go` + `validate.go` + tests — declaration schema, subsystem validation | S4, S7 | inline | |
| T6 | `router/registry.json` + 5 `criteria/<method>/CRITERIA.md` + registry tests | S1, S8, S9, S10 | inline | |
| T7 | `internal/provider/{provider,request,cloudflare}.go` + `cloudflare_test.go` — Clef adapter, retry, payload | S11, S12 | inline | |
| T8 | `internal/preflight/derive.go` + `cli/decide.go` + `decide_test.go` | S4, S9, S11, S18 | inline | |
| T9 | `cli/reroute.go` + `metrics.go` + `main_test.go` + `e2e_test.go` | S7, S13, S17 | inline | |
| T10 | `skills/method-router/SKILL.md` + `references/preflight-schema.md` + `references/engram-mirror.md` | S18, S19, S2 | inline | |
| T11 | `docs/engram-contract.md` master + `_shared/` copies (ftd, asdd, egm) | S15 | inline | |
| T12 | `rsc-ftd` + `rsc-asdd` SKILL.md persistence edits (1 + 7 files) | S15 | inline | |
| T13 | `egm` SKILL.md persistence edits (3 files) + goldfish rule | S15 | inline | |
| T14 | M7 verification: closed-set `genlte-sdd` reference check | S16 | inline | |
| T15 | `router/README.md` + root `README.md` + analysis supersession line | S17 | inline | |

Delivery: no git remote in this repository → the PR-chaining menu does not
apply. Work-unit commits land on `feat/method-router-v1`; push/PR/merge remain
human decisions. Forecast: > 400 authored lines; a single local branch, work
units in commit order per milestone.

## Log

- **L1** (user, verbatim) `Ahira usando ODD vamis a empezar con el desarrollo de
  ese plan`
- **L2** (decision, planning) Delivery strategy: repository has no remote
  (`git remote -v` empty), so `ask-on-risk`/chaining is inapplicable; local
  feature branch `feat/method-router-v1` with work-unit commits.
- **L4** (evidence) T2–T4 checks: `go vet ./...` clean; `go test ./...` green
  (cmd/router, internal/cli, internal/state). Because M1 tests were written
  after their implementation, two load-bearing guards were mutation-checked:
  removing the clef/`human_approved` rule fails
  `TestRecordRejectsInconsistentGateOutcomes`, and removing the true-branch
  guard fails `TestDetectWarnsOnMalformedAndForeignRecords` — the second check
  caught a weak assertion that was then strengthened to pin the collision
  guard exactly. From T5 onward tests are written first and RED is observed.
- **L5** (decision, implementation) Encoded-branch collisions (`fix/339` and
  `fix--339` encode to the same component) are resolved by the record's true
  `branch` field, not by changing the encoding the design pins: readers filter
  on `MatchesBranch` and report a foreign record as a warning. Losslessness
  comes from the record content, exactly as the design states.
- **L3** (decision, planning) Authoritative spec source: `docs/designs/method-
  router.md` (Reviewed). Specs S1–S19 quote it verbatim; where the design is
  silent, the design wins over inference and the gap is reported, never
  invented.
