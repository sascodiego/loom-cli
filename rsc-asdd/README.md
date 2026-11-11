# ASDD — Agent Spec-Driven Development (extracted from rsc-harness)

Source: `/src/rsc-harness`, commit `e4c6be1` (v4.1.0). Extraction only — nothing installed.

The full method (successor of `sdd`): an agent goes from idea to shipped, verified
software over a long autonomous run. Classic SDD reviewed at the end; ASDD changes two
things — **the spec must prove itself** and **review moves inside every task**, scaled
by risk tier. The human takes only the decisions the agent must not take alone
(decision queue). Routes: constitution → specify → clarify → plan → tasks → analyze →
implement (verify + review per task) → closing → ship.

```
system/skills/
├── asdd/SKILL.md          the dispatcher and method definition (byte-identical)
├── asdd/references/       model-routing.md (phase model routing table)
├── asdd/evals/            eval cases
└── <phase>/               the 11 phase skills ASDD routes to:
    sdd-init · constitution · specify · clarify · plan · tasks · analyze ·
    implement · review · verify · ship
reference/                 raw copies from the tag
```

Relative skill references inside ASDD (`../specify/SKILL.md`, …) resolve inside
`system/skills/` exactly as in the harness.

## Bindings to the harness (substitute when adapting)

- **`rsc risk`** (the CLI itself is extracted in `../rsc-tools/`: `node scripts/rsc.js risk`) — tierer: tier 0 docs → no reviewer,
  tier 1 ordinary code → one fresh-context reviewer, tier 2 (auth, billing, CI,
  migrations, secrets) → three-refuter panel **and human approval**. The agent never
  picks or argues down its own tier.
- **`rsc spec gate`** — spec-quality gate; a tier-2 branch only leaves the machine with
  a linked spec passing it.
- **`rsc review ledger panel`** — records the review panel verdicts; `rsc review escapes`
  feeds `review-evals/<sha>.md`.
- **`rsc main status`** — branch guard (`closed: true` denies local merges; park instead).
- **`02-DOCS/wiki/sdd/`** — decision queue and artifacts; `02-DOCS/wiki/index.md` is the
  knowledge map. The root `CLAUDE.md` is owned by `harness`, never edited by a phase.
- **Escalation/lane targets outside this tree**: `ftd` (small authorised changes,
  kept in `../rsc-ftd/`), plus auxiliary skills available in the harness if needed:
  `code-review`, `debug`, `deployment`, `design`, `idea-refinement`, `init`, `parallel`,
  `security-audit`, `testing-go/py/web`, `worktrees`, and stack skills (`go`, `nextjs`,
  `fastapi`, `flutter`, `postgresdb`, …) loaded per detected stack.

## Adapting

`asdd/SKILL.md` defines the method; the phase skills are self-contained procedures.
Point your runtime at `system/skills/`, bind the `rsc` CLI suite to your equivalents
(or run the tiers by hand — the method states who reviews what), and keep the
`02-DOCS` layout or remap it consistently across phases.
