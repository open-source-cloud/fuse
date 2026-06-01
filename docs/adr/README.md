# Architecture Decision Records

This directory holds FUSE's Architecture Decision Records (ADRs): short documents
that capture a significant architectural choice, the context that forced it, the
options considered, and the consequences.

We use the [MADR 3.0](https://adr.github.io/madr/) format. Start a new ADR by
copying [`template.md`](template.md).

## Process

- **When to write one.** Any decision that is costly to reverse, shapes the
  architecture, or that a future contributor would otherwise have to reverse-engineer
  from the code: a new subsystem, a cross-cutting pattern, a dependency/framework
  choice, a public contract, a data-model or persistence decision.
- **Numbering.** Zero-padded, monotonically increasing: `NNNN-kebab-title.md`
  (`0001`, `0002`, …). Numbers are never reused.
- **Status lifecycle.** `Proposed` → `Accepted` → (`Deprecated` | `Superseded`).
  A `Proposed` ADR records a decision we intend to make or have not finalized.
- **Immutability.** Once an ADR is `Accepted`, treat it as immutable. To change a
  decision, write a **new** ADR and mark the old one `Superseded by ADR-XXXX`
  (and link forward from the old one). Fixing typos/links is fine.
- **Diagrams.** Author diagrams in Mermaid to match the existing convention in
  [`../images/`](../images) (dark theme via an `%%{init}%%` header).

## Index

| #    | Title                                                                                   | Status   | Date       |
| ---- | --------------------------------------------------------------------------------------- | -------- | ---------- |
| 0001 | [Record architecture decisions using MADR](0001-record-architecture-decisions-using-madr.md) | Accepted | 2026-06-01 |
| 0002 | [Ergo actor model for workflow execution](0002-ergo-actor-model-for-workflow-execution.md)   | Accepted | 2026-06-01 |
| 0003 | [In-memory repositories by default, Postgres optional](0003-in-memory-repositories-by-default.md) | Accepted | 2026-06-01 |
| 0004 | [Multi-trigger workflow initiation](0004-multi-trigger-workflow-initiation.md)               | Accepted | 2026-06-01 |
| 0005 | [AI agents as workflow nodes; phased roadmap](0005-ai-agents-as-workflow-nodes-phased-roadmap.md) | Accepted | 2026-06-01 |
| 0006 | [LLM provider abstraction & multi-provider strategy](0006-llm-provider-abstraction-and-multi-provider-strategy.md) | Accepted | 2026-06-01 |
| 0007 | [Agent reasoning loop & tools-from-functions](0007-agent-reasoning-loop-and-tools-from-functions.md) | Proposed | 2026-06-01 |
| 0008 | [Settings, environments & secrets management](0008-settings-environments-and-secrets-management.md) | Proposed | 2026-06-01 |
