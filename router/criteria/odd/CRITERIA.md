# CRITERIA — odd (Organic Driven Development, the floor)

The default lane. Every request has a method, and this is the method that always
exists. Choose ODD when the work is small enough to hold in one head and finish
without a written specification.

## Work shapes this method serves

- A bounded change with a known outcome: a bug with a reproducible failure, a
  test that pins a rule, a rename or a single-file refactor.
- Investigations that end in an explanation or a proposal, not in code.
- Documentation edits, config tweaks, and dependency bumps with no migration.
- Work whose complete diff would fit in one review sitting (roughly 400 authored
  lines) and whose author could finish it from the request plus the diff.
- Work spanning one subsystem, one session, a handful of steps.

## Work shapes this method does NOT serve

- A change that spans several subsystems across several sessions, where a
  different session could not resume from the request and the diff alone.
- Anything that must be delivered against a written, human-approved
  specification before code exists.
- Long unattended runs where an agent must keep working for hours and prove its
  own progress against a spec.
- Document review that needs independent, fresh-context auditors.

When one of these holds, the request is not ODD: escalate the effort tier and let
the registry offer the method that carries the ceremony.

## Entry artifact required

None. The request text is the artifact. ODD still carries the harness's own risk
line, so low complexity never means unchecked: criticality lives inside the
method's checks, not in this classification.
