# CRITERIA — ftd (Fast-Track Development)

The ordinary authorised-change lane. One feature document, one minimal loop, one
session. FTD exists so a change that already has its scope settled does not pay
for the full spec method.

## Work shapes this method serves

- An authorised change whose intent and scope are already clear, needing one
  feature document (intent, scope, checklist, evidence, next step) rather than a
  specification.
- A change a single session can carry: a handful of tasks, one subsystem, a
  reviewable diff, checked off only against observed proof.
- Work that wants a minimal risk loop — self-check, `rsc risk`, one reviewer
  from tier 1 — and nothing heavier.
- Multi-step but bounded work where the checklist itself is the coordination
  artifact.

## Work shapes this method does NOT serve

- The full method with a spec that must prove itself, per-task review scaled by
  risk, and a decision queue: that is `asdd`, not this lane.
- The classification itself. Deciding which method a request gets is never FTD's
  job; the decisor (this router) does that.
- A request for information. Reading, explaining or comparing writes nothing and
  needs no feature document.
- Work that must run unattended for hours across several subsystems, or that
  needs human approval gates on artifacts before implementation.

## Entry artifact required

A feature document in the project's `02-DOCS` (or the repository's documented
equivalent) holding `Intent`, `Scope`, `Checklist`, `Evidence` and `Next`. No
feature document, no FTD: if the change cannot state its scope up front, the
effort tier belongs above this lane.
