# SDD System (tool-agnostic) — distilled from gentle-ai v3.7.0

Portable Spec-Driven Development system: one orchestrator, one executor agent per
phase, and the skill each agent executes. No runtime binding: skills are byte-identical
to the canonical assets in the `gentle-ai` repo at tag `v3.7.0`; agents carry only
`name` + `description` + a non-binding `suggested-tools` capability hint.

## Layout (paths relative to this directory)

```
orchestrator.md            coordinator: routes phases, meta-commands (sdd-new/continue/ff/status),
                           gatekeeper, ODD default workflow — self-contained, no placeholders
agents/sdd-<phase>.md      11 executor agents (init, explore, research, propose, spec, design,
                           tasks, apply, verify, archive, onboard) — never delegate further
skills/sdd-<phase>/        the procedure each agent executes (SKILL.md + references/)
skills/_shared/            contracts referenced by skills and the orchestrator:
                           sdd-phase-common, openspec-convention, engram-convention,
                           research-lifecycle, sdd-status-contract, sdd-orchestrator-sections
engram/protocol.md         persistence protocol (referenced by sdd-phase-common)
```

## How it fits together

1. The **orchestrator** is the only delegate: it validates readiness, launches exactly
   one phase agent at a time in dependency order
   (`init → explore → research? → propose → spec → design → tasks → apply* → verify? → archive`),
   and runs the meta-commands `sdd-new`, `sdd-continue`, `sdd-ff`, `sdd-status`.
2. Each **agent** opens its skill file, opens the shared contracts, does the phase work
   itself in-context, and persists artifacts to the active store.
3. **Skills** are the source of truth for behavior; agents are thin, per-runtime wrappers.

## Adapting to your runtime

- **Agents**: add your runtime's frontmatter (model, tools) around `name`/`description`.
  `suggested-tools` is the capability contract per phase, not a binding — map each entry
  to your runtime's equivalents. Rewrite nothing else.
- **Skill paths**: agents reference `skills/...` relative to this directory. Point them
  at wherever you install the tree.
- **Engram integration** (kept deliberately): agents/orchestrator call `mem_search`,
  `mem_get_observation`, `mem_save`, `mem_update` and read `engram/protocol.md`. These
  names are the Engram contract; bind them to your runtime's memory tools. Persistence
  backends are declared by the workspace and resolved natively — do not hardcode one.
- **Artifacts**: specs/design/tasks live under the OpenSpec convention
  (`openspec/changes/<change>/`, see `skills/_shared/openspec-convention.md`).

## Verification done

- No install-time placeholders remain (`{{CLAUDE_*}}`, `{{ENGRAM_TOOL_PREFIX}}`;
  the two surviving `{{...}}` strings are self-documenting text in
  `skills/_shared/sdd-orchestrator-sections.md`).
- No Claude paths or "Task tool" references; the delegate guard is generic.
- Orchestrator section placeholders (`{{GENTLE_AI_SDD_SECTION:*}}`,
  `{{GENTLE_AI_RESEARCH_LIFECYCLE}}`) were resolved inline from
  `skills/_shared/sdd-orchestrator-sections.md` and `research-lifecycle.md`.
- All internal document references resolve to existing files.
- `skills/` is byte-identical to `reference/skills/sdd-*` (the v3.7.0 tag content).
