# rsc-tools — the enforcement CLI + hooks (extracted from rsc-harness)

Source: `/src/rsc-harness`, commit `e4c6be1` (v4.1.0, npm package `@ericrisco/rsc`).
Extraction only — nothing installed. MIT licensed (LICENSE included).

This is the tooling layer the ASDD/FTD skills invoke for enforcement:
`risk` (tier 0/1/2 classifier), `spec gate`, `review ledger|consolidate|escapes`
(`.rsc/review-ledger.jsonl`, max 3 rounds), branch/ship guards (a tier-2 branch does
not leave the machine without a spec passing the gate and a recorded panel), and the
install/onboarding machinery.

```
scripts/        the CLI (bin: rsc.js, install-apply.js) + lib/ (risk, review-ledger,
                spec-gate, state, registry, onboarding…) + the guard hooks
                (asdd-gate, ship-guard, branch-guard, copy-guard, danger-guard,
                gitmoji-guard, userprompt-gate, spec-completeness, worktree-reaper,
                session-memory, sello, …) + skill rubrics/workflows
targets/        runtime adapters the scripts require (claude, codex, cursor, …):
                agents.js, index.js, sello.mjs, commands.js, wiring, opt-outs
schema/         frontmatter.schema.json (validated with ajv)
manifest.json   skill catalog manifest (read by registry/audit/doctor/backups)
package.json    bin entries, node >=18, deps: ajv + @iarna/toml (lazy-loaded)
```

## Running

```bash
cd rsc-tools
npm install            # optional for --version/basic; needed where ajv/toml kick in
node scripts/rsc.js risk [--base <ref>] [paths…]
node scripts/rsc.js review ledger …        # or: npm link && npx rsc …
```

The CLI operates on the repository in the current working directory: state in `.rsc/`,
artifacts under `02-DOCS/`, git-based (branches, ledger, guards wired into the agent
runtime at `rsc install` time via `targets/`).

## Coupling notes

- `scripts/` requires `targets/` heavily — both trees are needed together (verified:
  no `../` reference escapes this directory).
- It reads its own `skills/` catalog at runtime for install/capabilities; that full
  catalog (~200 skills) is upstream-only. The extracted methods live in `../rsc-ftd/`
  and `../rsc-asdd/`.
- The upstream test suite (`tests/`, 1.7M) needs the full catalog as fixture and is not
  brought; each method skill carries its own `evals/` instead.
- Hook wiring (`settings.json` hook entries per runtime) happens through
  `rsc install`/`onboard` against a chosen target assistant.

## Verification done

- `diff -r` 1:1 against the source for every brought item.
- Smoke test from this copy: `rsc 4.1.0`; `risk --help` lists `risk`, `spec gate`,
  `review consolidate|ledger|escapes`.
