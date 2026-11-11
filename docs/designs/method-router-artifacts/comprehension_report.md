# Stage 6 — Comprehension Report

Document: `docs/designs/method-router.md` (FeatureRange 1..352 & DetailedRange 353..539)
Reviewer: egm-comprehension (fresh context, document isolated — no GoldfishContext, per Stage 6 protocol)
Date: 2026-11-10

---

## What this proposes

The document proposes a small Go CLI ("the router") living in this repository that
removes the manual decision of which development method (ODD, RSC-FTD, RSC-ASDD, or
EGM) an AI coding harness ("gentle-shell") should apply to each new piece of work.
When the user requests work, the agent fills a short pre-flight (prose intent +
scope), the CLI derives observable facts (paths touched, diff stats, a size class),
and sends all of it — plus each method's written criteria — to Clef, a decision
model served by Cloudflare Workers AI that returns calibrated probabilities for an
effort tier (trivial/small/medium/large) and a method. The suggestion is shown to
the user, who confirms or overrides; the decision is persisted as tracked files
under `.router/` (an open record keyed by branch plus a minted routing ID, plus an
append-only per-ID history) and mirrored into Engram, the harness's memory system,
as a non-authoritative search index. Routing is one-shot: continuations of a
decided change are detected offline from the open record and never re-classified;
the only remedy for a wrong decision is a user-approved re-route that discards
in-flight artifacts and restarts under the new method with no artifact conversion.
The router only measures size of effort, never risk — each method keeps its own
internal risk gates. Implementation is staged M1–M8: CLI core and state store,
pre-flight schema and fact derivation, the provider interface and Clef client,
criteria files and a method registry, the harness routing skill, adaptations of the
four methods to one unified Engram persistence contract, a directory rename fixing
a `genlte-sdd` typo, and documentation.

## How the existing system works, per this document

The harness (gentle-shell, described as a Pi-based coding agent) already carries
four extracted development methods, each a different amount of process: ODD is its
own inline workflow (a checklist for small changes, treated as the "floor"),
RSC-FTD produces a feature document, RSC-ASDD is a full spec-driven flow with
review inside every task, and EGM is a multi-round document review. These exist as
skill packages in the repository with their own workflows intact — the design does
not author or redefine them. Today the method choice is made by the user (who must
remember which method exists and when it pays off) or by the agent (whose judgment
is biased toward less process), and both routes fail predictably: long runs start
without planning structure, and trivial changes get buried in ceremony until users
abandon the discipline. Each method currently persists its working state its own
way; the design adapts them all to one shared Engram contract — "gentle's
persistence contract": dual write (file + Engram upsert), read-back verification,
honest partial reporting, hybrid store mode always — with files in the repo
remaining the authority and Engram only a mirror. Clef is presented as an existing
decision model served by Cloudflare (`@cf/cloudflare/clef`) that consumes typed
questions and returns calibrated probabilities instead of prose. The router reads
each method's complexity-axis criteria from a `CRITERIA.md` per method
(bootstrapped from the method packages' own NOT-statements), hashed and sent with
every classification call.

---

## Findings

1. **HIGH — Clef is never defined.** The entire architecture hinges on "Clef, a
   decision model served by Cloudflare that returns calibrated probabilities
   instead of prose," but the document never explains what Clef is, whether it is
   an existing product or a model to be built, what its input contract looks like
   beyond "typed questions in," or how a Workers AI endpoint produces "calibrated
   probabilities." A fresh reader cannot tell if this is off-the-shelf or a hidden
   deliverable.

2. **HIGH — Continuation recognition has no matching mechanism.** G4/CUJ2 say a
   request that "continues an already-decided change" keeps its method, and
   `router detect` simply lists open records for the branch. The mechanism for
   deciding that a *new* user request is a continuation of the same logical
   change — versus genuinely new work on the same branch — is asserted but never
   described (does the agent judge? is there a similarity check? what if the
   branch legitimately hosts two changes?). The verdict enum is printed, but who
   or what computes "continuation" beyond file presence is unstated.

3. **HIGH — Duplicate-method ambiguity never addressed.** The registry contains
   both `asdd` and `gentle-sdd`, and the criteria for ASDD ("long multi-step
   runs, cross-area integration") and gentle-sdd ("spec-driven work with per-task
   review") describe heavily overlapping work shapes. Nothing in the document
   explains how Clef, the criteria, or the user distinguishes between them, nor
   what happens when probabilities split across near-identical methods.

4. **MEDIUM — Re-route discard is asserted without a mechanism.** G5/CUJ4 state
   that an approved re-route "discards in-flight work and restarts" with "no
   artifact migration." *How* in-flight artifacts are discarded (git reset? file
   deletion? which files?) is never specified, and discarding is arguably a
   destructive operation performed on the user's working tree — the safety
   mechanics are absent.

5. **MEDIUM — `human_approved` semantics leave gaps.** The field is true "only
   when the final method equals Clef's suggestion and the user accepted it." But
   the record also has `decided_by: joint_fallback`, where there is no Clef
   suggestion at all; whether joint decisions can ever be `human_approved: true`,
   and what the phase-1 baseline then measures, is undefined.

6. **MEDIUM — Diff-based derivation conflicts with pre-flight timing.**
   `derive.go` computes `touched_areas` and `diff_stats` from `git diff --numstat`
   against a base ref — but CUJ 1 and the pre-flight schema imply facts are
   derived at *request time*, before any work has been done. For a brand-new
   request there is no diff yet. The document never resolves whether facts are
   derived from a plan the agent declares, from work already started, or
   something else; this is the core input to the classifier.

7. **MEDIUM — "Failure after retry" is inconsistent across the document.** The
   architecture section says "plus at most two retries" and the cloudflare client
   says "retry with backoff (2 attempts)," which is compatible only with a
   generous reading; meanwhile exit code 3 is "provider failure after retry."
   Minor, but the retry count is stated two different ways without
   reconciliation.

8. **MEDIUM — Contradiction on where criteria live and what is hashed.** NFR says
   "`.router/` records carry facts and hashes only — no prose in git," yet the
   pre-flight schema includes agent prose fields, and CUJ 1 says the pre-flight
   goes to the CLI. Presumably prose stays out of `.router/` records (hence
   `payload_hash`), but `scope_summary` is still "sent raw or digested" (OQ4) —
   whether the prose ever touches a tracked file, and what `payload_hash` hashes,
   is left ambiguous against the "no prose in git" rule.

9. **LOW — Undefined project-internal terms.** Terms used without inline
   definition: "NOT-statements" (defined only by usage), "goldfish rule" (M6),
   "lossless evidence presentation format," "Ruling 2/3/8" (referenced as if the
   reader has access to a rulings list not in this document), "phase-1," "gentle's
   persistence contract (five rules verbatim)" — the five rules are summarized
   but never enumerated.

10. **LOW — `egm-review-loop` left open but wired everywhere.** OQ2 admits the
    entry contract for the review loop is undecided (document-path argument
    required, or excluded from v1), yet the diagram, registry, and dispatch all
    include it unconditionally. Shipping the design with an undecided dispatch
    target in the happy-path diagram is an internal inconsistency the document
    acknowledges but does not resolve.
