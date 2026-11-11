# EGM — Elephant-Goldfish Model (extracted)

Source: `/src/elephant-goldfish-model` (github.com/google/elephant-goldfish-model),
commit `8f16812` (v0.1.1), Google, Apache-2.0 (LICENSE included; upstream states it is
not an officially supported Google product). Extraction only — nothing installed.

The method: **Elephant** is the context-holder — the main agent develops the idea with
the full conversation and drafts the specification or design document. **Goldfish**
reviewers then examine the draft **without the earlier conversation** (fresh eyes),
exposing unstated assumptions, undefined terms, and blind spots. The human makes the
architectural trade-offs and approves before implementation.

```
system/
├── skills/       7 skills (byte-identical): SKILL.md + references + agents/openai.yaml
│                 + TEST.md + test_scenarios.yaml + assets per skill
├── agents/       10 reviewer/orchestrator persona definitions
└── resources/    document_standards/ (feature-spec, heilmeier, implementation-readiness,
                  re-implementation, zoom-out) + execution_instructions.md
reference/        raw copies + plugin.json + adapters/opencode (per-runtime packaging)
```

`system/` mirrors the repo-root layout, so references like `agents/synthesizer.md` and
`resources/document_standards/…` resolve exactly as upstream.

## Skills → agents

| Skill | Role | Agents it names |
|---|---|---|
| `egm-bootstrap` | document a codebase (README per subtree) | `egm-bootstrap-orchestrator`, `egm-zoom-out` (workers: `egm-codebase-details`) |
| `egm-brainstorm` | develop an idea with context | catalyst, synthesizer, codebase-details + reviewer validation |
| `egm-draft` | elephant drafts the design doc | — (elephant's own work; judged later by reviewers) |
| `egm-review` | design review before building | the goldfish panel: comprehension, structural, style, design, implementation-readiness, inventive-catalyst, codebase-details, synthesizer, zoom-out |
| `egm-review-loop` | review any document (chooses reviewers + standard) | synthesizer, zoom-out |
| `egm-resolve` | merge review feedback into decisions | — |
| `egm-drift-report` | check codebase drift vs documented design | — |

Agent files: `bootstrap_orchestrator` (egm-bootstrap-orchestrator), `codebase_details`,
`comprehension`, `design_critic` (named `egm-design`), `implementation_readiness`,
`inventive_catalyst`, `structural`, `style`, `synthesizer`, `zoom_out`.

## Bindings to the runtime (substitute when adapting)

- **Isolated subagents + local executor** — the core requirement (plugin.json:
  `requires_local_executor`; cloud support partial). Goldfish only works if the runtime
  can spawn subagents that start without the parent conversation.
- **`agents/openai.yaml`** per skill — manifest for the `com.openai` plugin extension;
  adapt or drop per runtime.
- **`resources/document_standards/`** — the rubrics reviewers judge against; for
  non-software documents `/egm-review-loop` lets you pick reviewers and standards.
- **Packaging** (`plugin.json`, `adapters/opencode`, `pyproject.toml` upstream tests)
  stays in `reference/`; the method itself is pure markdown.

## Verification done

- `diff -r` 1:1 against the source for skills, agents, resources.
- Every internal `agents/…` and `resources/…` reference resolves inside `system/`.
- No install-time placeholders.
- Upstream `tests/` (pytest, TESTING.md) not brought — repo-level test suite; each skill
  carries its own `TEST.md` + `test_scenarios.yaml` instead (included).
