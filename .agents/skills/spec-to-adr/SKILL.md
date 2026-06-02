---
name: spec-to-adr
description: Extract the architectural decisions from a GitHub Spec Kit spec or plan and record them as ADR(s) in the project's MADR format. Use after running /speckit.specify or /speckit.plan, or when the user wants to turn a spec/plan/feature document into Architecture Decision Records.
---

# spec-to-adr

Bridge the project's spec-driven workflow (GitHub Spec Kit) to its ADRs: read a spec/plan
and capture the *decisions* it implies as ADRs. Specs describe *what* to build; ADRs record
*why* a particular design was chosen — this skill turns the latter out of the former.

This skill composes with [`write-adr`](../write-adr/SKILL.md); follow its numbering, MADR
sections, index update, and conventions for every ADR you create.

## Inputs

- A Spec Kit artifact — typically `specs/<feature>/spec.md` and/or `specs/<feature>/plan.md`
  (produced by [`.agents/commands/speckit.specify.md`](../../commands/speckit.specify.md) and
  `speckit.plan.md`), or any feature/design doc the user points to.
- The governing principles in [`.specify/memory/constitution.md`](../../../.specify/memory/constitution.md)
  — decisions must not contradict it; if one does, call that out.

## Steps

1. **Read the spec/plan.** If the user didn't name one, list candidates
   (`ls specs/*/spec.md specs/*/plan.md` and `.specify/`) and ask which to use.
2. **Identify decisions.** Pull out the genuine architectural choices — not every requirement.
   A line deserves an ADR when it (a) is costly to reverse, (b) chooses between real
   alternatives, or (c) constrains the architecture. Examples: a storage/transport choice, a
   new cross-cutting pattern, a dependency, a public contract. Skip pure implementation detail.
3. **Confirm scope.** Show the user the candidate decision list and how you'd split it (one
   ADR per decision; merge tightly-coupled ones). Let them trim before writing.
4. **Author each ADR** via the `write-adr` conventions: next number, `docs/adr/NNNN-*.md` from
   `docs/adr/template.md`, MADR sections, index row. In **More Information**, link back to the
   source spec/plan (`specs/<feature>/...`) so the ADR and spec stay connected.
5. **Set status honestly.** `Proposed` if the spec is approved but the work isn't done;
   `Accepted` only once the decision is settled/implemented.

## Notes

- Reuse the spec's own "alternatives considered" / "rationale" sections as the ADR's
  *Considered Options* and *Decision Drivers* where present — don't reinvent them.
- One spec often yields several ADRs; keep each ADR single-decision and scannable.
- Don't copy requirement lists into ADRs; an ADR is the *decision and its rationale*, with a
  link to the spec for the full requirements.

$ARGUMENTS
