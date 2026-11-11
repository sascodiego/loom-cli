# Pre-flight declaration — schema v1

The document `router decide` and `router record` accept. It is a **declaration**,
not a measurement: at decision time the change usually does not exist yet, so
nothing about the diff is read, derived or estimated by the CLI.

## Shape

```json
{
  "schema_version": 1,
  "request_summary": "The user's request, verbatim.",
  "intent": "The outcome the user wants.",
  "scope_summary": "The parts of the system you expect to touch.",
  "subsystems": ["router", "docs"],
  "scope_estimate": {"steps": 4, "sessions": 1}
}
```

| Field | Type | Rule |
| --- | --- | --- |
| `schema_version` | integer | Must be `1`. |
| `request_summary` | string | The user's words verbatim. Non-empty prose. |
| `intent` | string | What outcome they want. Non-empty prose. |
| `scope_summary` | string | Expected surface. Non-empty prose. |
| `subsystems` | string[] | Each must be a tracked top-level directory of the repository, or the command exits 2. An empty list is allowed (a root-file-only change) and only reduces the breadth signal. |
| `scope_estimate` | object | `steps` and `sessions`, both at least 1. Your estimate, never the classifier's answer. |

## What must never appear

- `effort_tier` — the classifier's output.
- `method` — the classifier's output.
- Any other unknown field.

Decoding rejects unknown fields, so a caller cannot smuggle a classification
answer into the input to bias the endpoint. There is no option to relax this.

## Failure modes

| Condition | Exit | Message shape |
| --- | --- | --- |
| unknown field, whitespace-only prose, bad schema version, estimate below 1 | 2 | `invalid pre-flight: …` |
| a subsystem that is not a tracked top-level directory | 2 | `unknown subsystem(s) billing: declared subsystems must be tracked top-level directories …` |
| provider unreachable after 3 attempts | 3 | `provider failure: <kind>, status <n>, after <k> attempt(s)` |

## Why there is no diff

An earlier draft derived facts from the change itself. That is impossible at
decision time: routing happens *before* the work, when the diff is empty. The
facts that do exist are the declaration and the repository structure, so those
are what travel.
