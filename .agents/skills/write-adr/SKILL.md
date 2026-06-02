---
name: write-adr
description: Author an Architecture Decision Record (ADR) in this project's MADR format under docs/adr/. Use when the user wants to record, document, or propose a significant architectural decision, says "write/add an ADR", or asks to capture the rationale for a design choice.
---

# write-adr

Scaffold and author a new Architecture Decision Record in the project's MADR format.

ADRs live in `docs/adr/`. The format and process are defined by:

- Template: [`docs/adr/template.md`](../../../docs/adr/template.md) — the MADR 3.0 skeleton (copy this).
- Index + process: [`docs/adr/README.md`](../../../docs/adr/README.md) — statuses, numbering, immutability.
- Convention: [ADR-0001](../../../docs/adr/0001-record-architecture-decisions-using-madr.md).

## Steps

1. **Pick the next number.** Find the highest existing ADR number and add one (zero-padded, 4 digits):

   ```bash
   ls docs/adr/[0-9][0-9][0-9][0-9]-*.md | sed -E 's#.*/([0-9]{4})-.*#\1#' | sort -n | tail -1
   ```

   Next = that + 1 (e.g. `0009` → `0010`). Numbers are never reused.

2. **Create the file** `docs/adr/NNNN-kebab-title.md` by copying `docs/adr/template.md`, then fill it in:
   - `# NNNN. Title` — imperative, specific (e.g. "Use Postgres LISTEN/NOTIFY for schema replication").
   - `Status` — `Proposed` for a decision not yet finalized/implemented; `Accepted` once it is. (`Deprecated`/`Superseded` only when retiring one.)
   - `Date` — today's date (`YYYY-MM-DD`). Ask the user if unknown rather than guessing.
   - **Context and Problem Statement**, **Decision Drivers**, **Considered Options** (2–4, each with honest pros/cons), **Decision Outcome** (chosen option + why), **Consequences** (good / bad / neutral), **More Information** (links to related ADRs and the code/PR it describes).

3. **Ground it in the code.** Reference real file paths/types the decision touches; don't invent rationale. For a backfill ADR (documenting an existing decision), note that in a short italic line under the title and reconstruct rationale from the code and `.agents/rules/`.

4. **Update the index.** Add a row to the table in `docs/adr/README.md`:

   ```
   | NNNN | [Title](NNNN-kebab-title.md) | Status | YYYY-MM-DD |
   ```

5. **Cross-link.** In the new ADR's "More Information", link related ADRs; if it supersedes one, set the old ADR's status to `Superseded by ADR-NNNN` and link forward.

## Conventions

- Keep each ADR ~1 page and scannable. Reserve diagrams for ADRs where they add clarity; author them in Mermaid matching the dark theme in `docs/images/`.
- Never put secrets, keys, or tokens in an ADR.
- ADRs are immutable once `Accepted` — to change a decision, write a new superseding ADR rather than editing the old one (fixing typos/links is fine).
- Commit with a `docs(adr):` conventional-commit message.

$ARGUMENTS
