# router

The method router: it classifies a work request onto one of five methods
(`odd`, `ftd`, `asdd`, `gentle-sdd`, `egm-review-loop`) and records the decision
the human confirmed. It never implements the work and never writes memory.

- [Install and build](#install-and-build)
- [Commands](#commands)
- [Configuration](#configuration)
- [State layout](#state-layout)
- [Exit codes](#exit-codes)
- [Classification: the declared pre-flight](#classification-the-declared-pre-flight)
- [Clef and the provider contract](#clef-and-the-provider-contract)
- [Phase-1 measurement and the delegation criterion](#phase-1-measurement-and-the-delegation-criterion)
- [What v1 deliberately does not do](#what-v1-deliberately-does-not-do)

## Install and build

Go 1.23 or newer, stdlib only, no third-party dependencies.

```bash
cd router
go build -o bin/router ./cmd/router     # or: go run ./cmd/router <command>
go test ./...
```

## Commands

| Command | What it does |
| --- | --- |
| `router detect` | Lists open routing records for the current branch. Working tree only: no network, no git history. |
| `router decide --input <file\|->` | Classifies a declared pre-flight through Clef and prints the gate-ready decision. |
| `router record --input <file\|->` | Persists the confirmed decision: mints the `routing_id`, writes the record, appends the first history event. |
| `router re-route --routing-id <id> --method <m> [--justification <text>] [--requested-by agent\|user]` | Supersedes an open record with a fresh one under the target method. Never touches the working tree. |
| `router close --routing-id <id>` | Marks an open record closed at ship. |
| `router metrics [--json]` | Agreement rate over recorded decisions, stratified, suppressed below 30 decisions. |

Stdout is always one JSON document. Human-readable text goes to stderr.

`detect` prints `{"candidates": [...], "verdict": "continuation" | "new"}`.
Multiple open records are listed in full: the human resolves which one the
session continues.

`decide` prints `tier`, `method`, `probabilities` (every option, not only the
winner), `request_id`, `provider`, `model`, `criteria_hash`, `payload_hash`,
`ambiguous`, and `substituted` when the availability rule fired. The confidence
policy: when the top two methods are within 0.05, `ambiguous` is true and the
gate must present both.

## Configuration

Per project: `.router/config.json`, tracked.

```json
{
  "provider": "cloudflare",
  "model": "@cf/cloudflare/clef",
  "endpoint": "https://api.cloudflare.com/client/v4/accounts/<account>/ai/run/@cf/cloudflare/clef"
}
```

Credentials come from the environment and are never persisted:

| Variable | Purpose |
| --- | --- |
| `CLOUDFLARE_API_TOKEN` | API key for the endpoint. Required. |
| `CLOUDFLARE_ACCOUNT_ID` | Account id, when the endpoint URL needs it. |

Unknown fields are rejected in both the config and the pre-flight declaration,
so a stale or hand-edited file fails loudly instead of silently misclassifying.

## State layout

All routing state is plain tracked files. Rollback is a `git revert` of
`.router/`.

```
.router/
├── config.json                          provider, model, endpoint
├── open/
│   └── <branch-encoded>/
│       └── <routing_id>.json            one record per routing decision
└── history/
    └── <routing_id>.jsonl               append-only audit trail
```

`<branch-encoded>` replaces `/` with `--`, so `fix/339-cedula` becomes
`fix--339-cedula`. The encoding alone is not injective (a branch may contain a
literal `--`); the true branch name lives inside each record and readers compare
it, so a collision is reported as a warning instead of being trusted.

The record carries the facts of the decision, including `payload_hash` — the
SHA-256 of the exact pre-flight declaration that was classified. The prose
itself is never stored in tracked files.

History events: `decided`, `gate_overridden`, `joint_fallback`,
`re_route_requested`, `superseded`, `closed`. Every event carries the full
record snapshot. `reconciled` is reserved for v2 and has no v1 writer.

## Exit codes

| Code | Meaning |
| --- | --- |
| 0 | Success. |
| 2 | Usage or validation error: malformed declaration, unknown subsystem, missing config, unknown or closed record, unknown or unavailable method. |
| 3 | Provider failure after retry. The harness switches to the joint fallback. |
| 4 | State conflict — reserved for the v2 reconciliation engine, never returned by v1. |

## Classification: the declared pre-flight

Routing happens before the work, so no diff exists to measure. What travels to
the classifier is the declaration:

```json
{
  "schema_version": 1,
  "request_summary": "the user's request, verbatim",
  "intent": "the outcome they want",
  "scope_summary": "the parts of the system expected to change",
  "subsystems": ["router", "docs"],
  "scope_estimate": {"steps": 4, "sessions": 1}
}
```

- `subsystems` are validated against the repository's tracked top-level
  directories; a mismatch is exit 2.
- `effort_tier` and `method` never appear in the input: they are the
  classifier's output, and unknown input fields are rejected so a model answer
  cannot be smuggled in.

The classification axis is **complexity only**. Criticality lives inside each
method's own gates and is never a routing input.

## Clef and the provider contract

Clef (`@cf/cloudflare/clef`) is an existing Cloudflare decision model served
over Workers AI: it consumes a `state` plus a schema of typed questions and
returns a calibrated probability for every allowed option of every question in
one forward pass. It generates no prose.

The adapter is the single place that knows the endpoint:

```text
POST {endpoint}
Authorization: Bearer {CLOUDFLARE_API_TOKEN}
{"model":"@cf/cloudflare/clef",
 "input":{"state":<declaration + session facts + criteria excerpts>,
          "questions":[<effort_tier enum>, <method enum>]}}
```

Response: `{"success":true,"result":{"request_id":"…","answers":{…}}}`.

Retry: 2 retries, 3 attempts total, exponential backoff. Auth failures and
malformed answers are never retried; 429 and 5xx are. Every failure is typed
(`auth`, `quota`, `server`, `timeout`, `transport`, `malformed_response`) so the
joint-fallback conversation has evidence instead of a string.

Swapping classifiers means implementing `provider.Provider`; nothing else
changes.

## Phase-1 measurement and the delegation criterion

The router is the primary classifier for phase 1. The criteria files are the
degradation path, not the baseline.

`router metrics` computes the agreement rate from the history:

- **clean** — `decided_by: clef` at the current schema version.
- **disagreement** — `decided_by: user_gate_override`, which counts in the
  denominator.
- **excluded** — `joint_fallback` and `override_re_route`, counted separately and
  never blended into the rate, plus `stale_schema`: a clef decision recorded under
  an obsolete pre-flight schema. Stale decisions stay visible in `by_decided_by`
  (they are real decisions) and are excluded from the denominator by name, so
  `clef = clean + stale_schema` always reconciles.
- **suppressed** below 30 decisions, because a rate over a handful of decisions
  means nothing.

The criterion for delegating classification to the rubric was declared before
the measurement started: at least 30 clean `decided_by: clef` decisions, at least
90% sustained human agreement, and a stable `criteria_hash` throughout the
window. `criteria_hash` is SHA-256 over the concatenation of every criteria file
in registry order, including the method marked unavailable, so any change to the
classifier's material is visible.

## What v1 deliberately does not do

`router status`, `router doctor` and the reconciliation engine land after
observed divergence, not before. `egm-review-loop` is registered but marked
`available: false` until its entry contract exists, so it is never offered as an
option. Exit code 4 arrives with the reconciliation engine. There is one
provider.

Registry: `router/registry.json`. Criteria: `router/criteria/<method>/CRITERIA.md`,
each under 2 KB, stating the work shapes the method serves, the ones it does
not, and the entry artifact it requires.
