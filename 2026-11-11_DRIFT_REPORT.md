# Drift Report — 2026-11-11

Scope: the four documents in this repository that specify its own behaviour —
`docs/designs/method-router.md`, `router/README.md`, `README.md` (root) and
`docs/engram-contract.md`. The package READMEs (`gentle-sdd/`, `rsc-ftd/`,
`rsc-asdd/`, `rsc-tools/`, `egm/`) are excluded: they describe extracted
third-party copies, not code this repository implements, so a difference there is
provenance, not drift.

Method: each document was read in full and every concrete claim it makes about
files, symbols, fields, enums, commands, flags, exit codes and paths was checked
against the current source. Only divergences are reported.

## docs/designs/method-router.md

- **Design/code divergence — classifier output vocabulary.** The design fixes the
  tier enum as `effort_tier (trivial | small | medium | large)`
  (`docs/designs/method-router.md:331`). The code offers
  `low | medium | high` (`router/internal/provider/request.go:42-44`). Two of the
  four designed options (`trivial`, `large`) do not exist, and two of the three
  emitted ones are not in the design. `tier` is persisted on every record
  (`router/internal/state/open.go:64`) and printed by `decide`
  (`router/internal/cli/decide.go:31`), so a reader of the design cannot predict
  a single recorded value. Nothing enforces the vocabulary at read time, so an
  in-tree `tier` value is never validated against either list.
- **Missing file — `router/internal/state/config.go`.** M1 lists it as a
  `(create)` target holding `{provider, model, endpoint}` loading and the
  credential environment names (`docs/designs/method-router.md:520-524`). The
  file does not exist; that content lives in `router/internal/state/detect.go:88-128`
  alongside `detect`. Behaviour is present, the designed module boundary is not.
- **Misplaced test — `router/internal/cli/main_test.go`.** M4 names that file for
  the CLI-layer exit-code contract (`docs/designs/method-router.md:637`). It exists
  as `router/cmd/router/main_test.go`, next to the binary it drives. The contract
  is tested; the designed path is not.
- **Logic divergence — `re-route` does not produce a fresh decision.** The design
  says an approved re-route "emits the new decision" and that "the change
  restarts under the user-approved target method with a fresh decision"
  (`docs/designs/method-router.md:215-222`, `:254-257`). The implementation copies
  the predecessor record and changes only the method and the decision origin, so
  `request_id`, `payload_hash`, `criteria_hash`, `provider` and `model` are
  inherited rather than re-derived (`router/internal/cli/reroute.go:75-90`). The
  new record therefore claims a classification it never ran. (Recorded as an
  implementation decision in `odd/tasks/method-router-v1.md` L7, but the design
  text still states otherwise.)

## router/README.md

- **Documented credential that no code reads.**
  `router/README.md:67` documents `CLOUDFLARE_ACCOUNT_ID` as "Account id, when the
  endpoint URL needs it". The constant is declared
  (`router/internal/state/detect.go:93`) but never read: the provider factory
  reads only `CLOUDFLARE_API_TOKEN` (`router/internal/cli/decide.go:22-33`). An
  endpoint URL that genuinely needs the account id cannot be assembled from the
  environment as documented, and the declared constant is dead.
- **Registry path claim vs. reality.** The README states the criteria live at
  `router/criteria/<method>/CRITERIA.md` and that `criteria_hash` covers "every
  criteria file in registry order, including the method marked unavailable"
  (`router/README.md:22-24`, `:196-199`) — that holds. No drift.

## README.md

- **Overbroad claim about the shared contract copies.** The root README states the
  contract is "reproduced per package under
  `<package>/system/skills/_shared/engram-contract.md`" (`README.md:63`). Copies
  exist for `rsc-ftd`, `rsc-asdd` and `egm`. `gentle-sdd/system/skills/_shared/`
  holds `engram-convention.md` and no `engram-contract.md`, so the sentence is
  false for one of the four packages it names. `docs/engram-contract.md`
  qualifies gentle-sdd as unchanged/reference-only, but the root README does not
  carry that qualification.

## docs/engram-contract.md

No drift detected.

- The five rules match `docs/analysis/engram-unification-analysis.md:209-221`
  without a meaning-changing difference.
- Every naming-table row is named by the skill that must write it
  (`rsc-ftd/system/skills/ftd/SKILL.md:67`;
  `rsc-asdd/system/skills/asdd/SKILL.md:144-147` plus the six phase skills;
  `egm-draft/SKILL.md:325`, `egm-review-loop/SKILL.md:153`,
  `egm-resolve/SKILL.md:187`), and no skill invents a key the contract omits.
- The exclusions hold: the `.rsc/` ledger is never mirrored
  (`rsc-asdd/system/skills/asdd/SKILL.md:148`) and no reviewer persona in
  `egm/system/agents/` calls `mem_*`
  (`egm/system/skills/egm-review-loop/SKILL.md:46-47` states the hard rule).
- Each existing copy points at the canonical document with a relative path that
  resolves correctly from its own location
  (`<pkg>/system/skills/_shared/` → four levels up → repository root).

## Ranked summary

1. Classifier tier vocabulary differs from the design in both directions
   (`trivial|small|medium|large` designed, `low|medium|high` emitted), and no
   validation pins either list.
2. `re-route` inherits the predecessor's `request_id`/`payload_hash` while the
   design promises a fresh decision, so a record can claim a classification that
   was never run.
3. `CLOUDFLARE_ACCOUNT_ID` is documented and declared but never read.
4. The designed module boundary `internal/state/config.go` does not exist.
5. The root README's "per package" contract-copy claim is false for `gentle-sdd`.
6. The M4 test path in the design (`internal/cli/main_test.go`) is actually
   `cmd/router/main_test.go`.
