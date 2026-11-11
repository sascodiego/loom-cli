# CRITERIA — gentle-sdd (Gentle AI Spec-Driven Development)

The artifact-chain method: every change travels through approved documents —
proposal, specification, design, tasks — before and while it is applied. The
ledger, not the conversation, is the authority.

## Work shapes this method serves

- A change delivered through an explicit artifact chain: proposal → spec →
  design → tasks → apply → archive, each step reviewable alone.
- Work where a human approves artifacts at their boundaries: the spec before
  design, the design before tasks, the tasks before apply.
- Multi-session work whose continuation must be reconstructable from the ledger
  alone, by a later session.
- Work in any language and against any toolchain: the method owns documents, not
  build steps.
- Ambiguous requirements that must converge into a spec.

## Work shapes this method does NOT serve

- A small authorised change with no artifact chain: that is `ftd`, or ODD when
  smaller still.
- A long unattended run whose authority is a spec plus per-task review and a
  decision queue: that is `asdd`.
- The workspace harness itself, or tool registration.
- A document review with fresh-context adversarial auditors: that is
  `egm-review-loop`.
- A request for information, which writes nothing.

## Entry artifact required

A change directory in the project's OpenSpec-style ledger holding the proposal
and specification, approved before design starts. No approved spec, no
implementation: the chain exists so code cannot outrun its justification.

## Mutual exclusion — against `asdd`

Both are spec-driven; the deciding factor separates them.

Choose **gentle-sdd** when the artifact chain is the point: documents approved at
their boundaries, the ledger the authority a later session resumes from.

Choose **asdd** when autonomy over time is the point: long unattended runs,
review inside every task scaled by risk, a human decision queue.

Approved artifact chain, any run length: `gentle-sdd`. Long run, light artifact
chain: `asdd`.
