# FTD — Fast-Track Development (extracted from rsc-harness)

Source: `/src/rsc-harness` (Gentleman-Programming RSC harness), commit `e4c6be1` (v4.1.0).
Extraction only — nothing installed.

The default lane for ordinary authorised change: one feature document per feature in
`02-DOCS` (intent, scope, checklist, evidence, next step), tasks checked off only against
observed proof, branch question before code, minimal risk loop (self-check, `rsc risk`,
one reviewer from tier 1). Deliberately NOT the full method — that is ASDD (`rsc-asdd/`).

```
system/
├── skills/ftd/SKILL.md              the method (byte-identical to the harness skill)
├── skills/ftd/evals/                cases.yaml + routing.json (its eval harness)
└── skills/orient/references/orientation-contract.md   declared dependency (voice protocol)
reference/                           raw copies from the tag
```

## Bindings to the harness (substitute when adapting)

- **`rsc risk`** — the risk tierer (path table + content patterns → tier 0/1/2, decides
  reviewer count). The method trusts it and never lets the agent pick its own tier.
  The CLI itself is extracted in `../rsc-tools/` (run it from there). Substitute with your own risk classification or run without tiers (loses the
  reviewer-scaling rule).
- **`02-DOCS/`** — the workspace convention (feature docs, `02-DOCS/wiki/`, evidence).
  This is the method's artifact contract, like `openspec/` in other systems.
- **Sibling skills by routing** (kept in the harness, not part of this method):
  `suggest` (the decisor/classifier), `asdd` (escalation when tier 2 or contradicting
  a recorded decision), `review`, `verify`, `debug`, `worktrees`, `specify`.
- **`orient` voice protocol** — included here (`skills/orient/references/`), since the
  skill declares it as its communication contract.
- **`user-profile.md`** — read from `02-DOCS/wiki/harness/` for tone/technical level.

## Adapting

`skills/ftd/SKILL.md` is the whole method: point your runtime's skill loader at the
`system/skills/` tree and bind the routing names above to whatever exists in your
environment. All internal file references resolve inside this tree.
