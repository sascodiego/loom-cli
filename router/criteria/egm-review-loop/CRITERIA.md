# CRITERIA — egm-review-loop (Elephant-Goldfish review)

The document-review method: an orchestration driver spawns fresh-context
adversarial auditors (Goldfish) against qualitative standards and structural
templates, while the primary agent (Elephant) holds institutional knowledge and
applies only the findings it approves.

## Work shapes this method serves

- Reviewing a document that must survive adversarial reading: a design doc, a
  README, a feature specification, an implementation plan.
- Work where unstated assumptions are the risk and a reviewer with no
  conversation history is the countermeasure.
- Multi-round review that must converge: findings resolved, approved fixes
  applied, and the loop stopped by a declared convergence cap rather than by
  exhaustion.
- Verification of document completeness and tone against an explicit standard.

## Work shapes this method does NOT serve

- Implementing anything. The loop reviews artifacts; it does not write code and
  does not replace a method that does.
- Code changes, tests or builds. Those belong to the method that owns the
  implementation.
- A review of a diff or a pull request. This method reads documents against
  standards, not patches against behavior.
- A quick sanity read of a note, where one pass by the author is enough.

## Entry artifact required

A target document (with its line range if partial), the qualitative standards or
templates it is audited against, and the review context the auditors need. The
reviewers never call Engram: they receive the document by prompt or locator
only, so their context stays fresh.
