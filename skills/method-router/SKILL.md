---
name: method-router
description: >-
  Routes a new work request to one of five methods (odd, ftd, asdd, gentle-sdd,
  egm-review-loop) using the router CLI, and detects continuations so a resumed
  request keeps its method. Use at the start of any work request, when a session
  resumes on a branch, or when the agent wants to propose a re-route because the
  observed scope outgrew the decided method.
---

# Method Router

The router decides *which method* the work runs under. It classifies. It does
not implement. Everything the CLI must not do — talking to the user, showing
evidence, writing the Engram mirror — is your job in this skill.

Five methods exist and ODD is the floor: every request has a method, and low
complexity never means unchecked. The classification axis is **complexity
only**; criticality lives inside each method's own gates and is never a routing
input.

## Step 1 — detect a continuation

Run `router detect` before anything else.

```bash
router detect   # stdout: {"candidates": [...], "verdict": "continuation" | "new"}
```

**Routing is one-shot per work request.** A continuation keeps its method and is
never re-routed. State on disk is the authority: `detect` reads the working
tree, never git history and never the network.

- `verdict: "new"` → go to Step 2.
- `verdict: "continuation"` → echo the selected record back to the user (slug,
  method, decided_at), then dispatch into that method. Do not call `decide`. If
  several candidates are listed, show all of them and ask the user which one
  this session continues — by its key, never free text. If the user says this is
  new work while open records exist, say so explicitly and let them decide;
  never silently discard an open record.
- `warnings` present → tell the user what the CLI reported (a malformed record,
  or a record belonging to a colliding branch name). Never repair it silently.

## Step 2 — pre-flight interview (new request)

Write a declaration with the user, not for them.

- `request_summary` — the user's request **verbatim**. Quote them; do not improve
  their words. This is the one field that must not be paraphrased.
- `intent` — what outcome they want, in one or two sentences.
- `scope_summary` — the parts of the system you expect to touch.
- `subsystems` — the repository's top-level directories this work touches. The
  CLI validates each one against tracked files and exits 2 on a mismatch, so
  never guess: if unsure, ask or omit the uncertain one.
- `scope_estimate` — your own guess at steps and sessions. It is input, never
  the answer.

Never put `effort_tier` or `method` in the declaration: those are the
classifier's output, and the CLI refuses unknown input fields precisely so a
model answer cannot be smuggled in. Schema: `references/preflight-schema.md`.

## Step 3 — decide

```bash
router decide --input <file|->   # exit 0 JSON, 2 invalid declaration, 3 provider failure
```

Output: `tier` (`low`, `medium`, `high`), `method`, `probabilities`,
`request_id`, `provider`, `model`,
`criteria_hash`, `payload_hash`, `ambiguous`, and `substituted` when the
availability rule fired.

## Step 4 — present the evidence and gate

Show, in the user's language:

1. The suggested **method and tier**.
2. The **probability of every method option**, not only the winner.
3. The **criterion that matched** — the part of the winning method's CRITERIA.md
   that justifies it (the work shape it serves).
4. A recap of the declaration: summary, intent, scope, subsystems, estimate.
5. The three choices: **confirm**, **override**, **re-route**.

Nothing is shown that the record does not capture. If the gate cannot display a
fact from the decide output, it is not evidence to the user.

**Ambiguity** (`ambiguous: true`): the top two methods are within 0.05. Present
both as live options with their probabilities; do not imply a winner.

**Substitution** (`substituted: true`): the classifier's top choice is not
dispatchable in v1. Say which method was substituted and why, and offer the next
alternative too.

**Provider failure** (exit 3): switch to the joint fallback below.

## Step 5 — record the decision

```bash
router record --input <file|->   # the decide output plus the gate outcome
```

The gate outcome fixes `decided_by`:

| Gate outcome | decided_by | human_approved | user_choice |
| --- | --- | --- | --- |
| user accepts the suggestion | `clef` | `true` | omit |
| user picks a different method | `user_gate_override` | `false` | the chosen method |
| joint fallback after failure | `joint_fallback` | `false` | the agreed method |

`record` mints the `routing_id` and refuses an inconsistent combination: `clef`
with `human_approved: false`, `clef` with a method different from the
suggestion, an override with no `user_choice`, or `override_re_route` (that one
belongs to `re-route`).

## Step 6 — mirror to Engram, then dispatch

The CLI never writes Engram. You do:

```
mem_save(topic_key: "router/{slug}/decision", capture_prompt: false, ...)
```

The mirror is an index for later recall — never authority. The tracked record is.
Contract: `references/engram-mirror.md`.

Then dispatch into the recorded method using the registry's skill path.

## Joint fallback (provider failure)

When `decide` exits 3, the classifier is unavailable. Say so plainly, with the
failure message as evidence. Then:

1. Present the pre-flight declaration again.
2. Compare, side by side, what each method's CRITERIA.md says it serves.
3. Ask the user to choose. You may recommend; you may not decide.
4. `router record` with `decided_by: joint_fallback`, `human_approved: false`,
   `user_choice: <chosen method>`, and the decide fields you do have. If
   `decide` failed before producing any output, use the last successful
   decision's `request_id` only when it belongs to this same request; otherwise
   ask the user to re-run `decide` once before falling back.

Never record a decision the user did not make, and never invent a `request_id`.

## Ambiguity script

```
router decide: ambiguous — odd 0.50, ftd 0.47 (gap 0.03, threshold 0.05)
```

Say: the classifier cannot separate these two. Show both criteria excerpts. Ask
which one. Record the answer as `clef` with `human_approved: true` only if the
user confirms the classifier's own top choice; otherwise it is a
`user_gate_override`.

## Agent-initiated re-route

When observed scope outgrows the decided method, propose a re-route with
evidence. Never re-route unilaterally.

```bash
router re-route --routing-id <id> --method <target> --input <file|-> \
  --justification "<observed evidence>" --requested-by agent
```

`--input` carries the **re-declared pre-flight**: the scope has grown, so the
declaration is rewritten for what the work is now. The CLI re-classifies it with
Clef before touching any record, so the new record's `request_id`, `payload_hash`,
`criteria_hash`, `tier` and `suggested_*` come from a real call rather than from
the predecessor. The recorded `method` stays the target *you* passed — the
classification supplies evidence, not authority.

On approval this supersedes the current record, opens a fresh one under the
target method with `decided_by: override_re_route`, and records the whole trail.
If the classification fails (bad declaration, provider failure) nothing is
persisted and the current record stays open: fix the declaration and retry, and
say so to the user instead of silently keeping the old method.

The in-flight work is **discarded, not migrated**: the change restarts under the
new method. Discarding artifacts is your and the user's job — the CLI only
records.

## Close

```bash
router close --routing-id <id>
```

At ship. Not before: `detect` reports every open record on the branch.

## Metrics

```bash
router metrics            # agreement rate; suppressed below 30 decisions
router metrics --json
```

Report it when the user asks how the routing is doing. Do not act on a suppressed
rate, and do not treat the rate as permission to change anything.
