# 0001. Record architecture decisions using MADR

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

## Context and Problem Statement

FUSE's architectural rationale has so far lived implicitly across `CLAUDE.md`,
the `.agents/rules/*.mdc` files (formerly `.cursor/rules/`), code comments, and commit messages. As the
project takes on larger, cross-cutting initiatives (notably the AI-agent work),
contributors increasingly need to know *why* a structure exists, not just what it
is. There is no durable, reviewable record of significant decisions, so rationale
gets re-litigated or lost.

How should we capture significant architectural decisions so they are durable,
discoverable, and reviewable alongside the code?

## Decision Drivers

- Decisions should be versioned with the code and reviewed via the normal PR flow.
- Lightweight, markdown-first (the repo has no docs site; docs are plain markdown).
- Enough structure to capture the *alternatives* and *trade-offs*, not just the outcome.
- Low ceremony so writing one is not a barrier.

## Considered Options

- **MADR 3.0** (Markdown Any Decision Records) — structured template with explicit
  options and pros/cons.
- **Nygard ADRs** — minimal: Context / Decision / Consequences.
- **No formal ADRs** — keep documenting in `CLAUDE.md` and cursor rules.

## Decision Outcome

Chosen option: **MADR 3.0**, stored as numbered markdown files in `docs/adr/`.
MADR's explicit "Considered Options" and "Pros and Cons" sections fit FUSE's needs
well — several decisions (e.g. the LLM framework choice) hinge on comparing
alternatives, which Nygard's leaner format does not surface. It remains plain
markdown, so it reviews and renders like the rest of `docs/`.

### Consequences

- Good: durable, greppable decision history; alternatives and trade-offs are explicit;
  reviewed in PRs.
- Good: a stable target to link to from `CLAUDE.md`, cursor rules, and code comments.
- Bad: a small per-decision writing cost.
- Neutral: ADRs are immutable once Accepted — changing a decision means a new,
  superseding ADR (see [`README.md`](README.md)).

## Pros and Cons of the Options

### MADR 3.0

- Good: explicit options/pros-cons; widely recognized; still markdown.
- Bad: slightly more verbose than Nygard.

### Nygard ADRs

- Good: minimal, fast to write.
- Bad: little room to compare alternatives, which is central to several FUSE decisions.

### No formal ADRs

- Good: zero new process.
- Bad: rationale stays scattered and gets lost — the status quo we want to fix.

## More Information

- Format reference: <https://adr.github.io/madr/>
- Process, numbering, and the index: [`README.md`](README.md); skeleton: [`template.md`](template.md).
- This convention is referenced from the repository `CLAUDE.md`.
