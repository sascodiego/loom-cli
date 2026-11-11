# CRITERIA — asdd (Agent Spec-Driven Development)

The method for long unattended agent runs against a spec that proves itself. The
cost is real: a constitution, an approved spec, a plan, per-task review, and a
decision queue for what the agent must not decide.

## Work shapes this method serves

- A feature or substantial change from idea to shipped, verified software, in
  any language.
- A run long enough that the agent works for hours on its own: many tasks,
  several subsystems, more than one session.
- Work where green tests are not proof: the spec states what the user can do,
  what result they get against an approved reference, what it must respect, and
  how it proves it.
- Work that wants review inside every task, scaled by `rsc risk`, with a
  decision queue for what the agent must not decide alone.

## Work shapes this method does NOT serve

- A single phase. ASDD dispatches; running one phase is not this method.
- A small authorised change whose scope is already settled: that is `ftd`.
- The workspace harness itself, or tool registration.
- A document review with independent fresh-context auditors: that is
  `egm-review-loop`.
- A request for information, which writes nothing.

## Entry artifact required

`constitution.md` plus a spec under the project's ASDD ledger (default
`02-DOCS/wiki/sdd/`), approved by the human before planning. The spec must pass
its gate: the four questions answered and an explicit out of scope.

## Mutual exclusion — against `gentle-sdd`

Both are spec-driven; the deciding factor separates them.

Choose **asdd** when autonomy over time is the point: long unattended runs,
review inside every task scaled by risk, a human decision queue.

Choose **gentle-sdd** when the artifact chain is the point: proposal → spec →
design → tasks → apply → archive, with the human approving artifacts at their
boundaries and the ledger as the authority for a later session.

Long run, light artifact chain: `asdd`. Approved artifact chain, any run length:
`gentle-sdd`.
