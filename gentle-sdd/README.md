# Gentle AI SDD — v3.7.0 extraction

Source: `Gentleman-Programming/gentle-ai`, tag `v3.7.0`. Extraction only — nothing installed.

```
system/       the portable, tool-agnostic SDD system (start here — see system/README.md)
reference/    raw extraction from the tag, organized by adapter (for diffing and provenance)
```

`reference/` mirrors the repo layout at `v3.7.0`:

- `skills/` — canonical SDD skills + `_shared/` contracts (source of `system/skills/`)
- `agents/`, `commands/`, `orchestrators/` — per-adapter agents (`claude`, `cursor`, `kimi`, `kiro`),
  commands (`claude` `gentle-sdd-*`, `opencode` `sdd-*`), orchestrators (12 adapters)
- `plugins/`, `overlays/`, `workflows/`, `misc/` — opencode plugins/overlays, windsurf workflows
- `contracts/sdd-integration/` — edit-authority consent schema + fixture
- `engram/protocol.md` — persistence protocol

## Measured differences between adapters (v3.7.0)

- **Skills (`internal/assets/skills/`)**: single canonical set, shared by every adapter —
  byte-identical everywhere. Skills are already tool-agnostic.
- **Executor agents**: same phase procedure in every adapter; bodies differ only in
  tool phrasing ("Task tool" vs "task/delegate"), home-path conventions
  (`~/.claude/skills/` vs `~/.kiro/skills/` + Windows variants) and minor step
  renumbering. The `claude` set is the reference implementation; `reference/agents/`
  keeps all four for comparison.
- **Commands**: thin launchers that just map a slash-command to the orchestrator;
  `claude` names them `gentle-sdd-*`, `opencode` `sdd-*`.
- **Orchestrators**: vary by runtime capability — sub-agent-capable runtimes
  (claude, cursor, opencode) delegate phases to agents; rules-based runtimes
  (codex, gemini, antigravity, hermes, qwen, windsurf, generic) inline the same
  sections without sub-agents. Shared sections come verbatim from
  `skills/_shared/sdd-orchestrator-sections.md` (drift is test-enforced upstream).
- **kimi** uniquely ships `.yaml` frontmatter companions per agent.
- No Pi-specific assets exist at this tag; the `generic` orchestrator is the one to
  use for runtimes without a dedicated adapter.
