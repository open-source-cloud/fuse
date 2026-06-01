# 0009. Portable AI agent guidance in `.agents/`

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

## Context and Problem Statement

FUSE's AI/editor guidance was Cursor-specific: `.cursor/rules/` (13 `.mdc` coding rules),
`.cursor/skills/` (7 architecture skill packs), and `.cursor/commands/` (9 GitHub Spec Kit
`speckit.*` commands), alongside `.specify/` and a root `AGENTS.md`. Claude Code does not read
`.cursor/` at all — it loads `CLAUDE.md` (with `@path` imports) and skills under
`.claude/skills/`. So a large, valuable body of guidance was usable by Cursor but invisible to
Claude Code and any other agentic tool.

How do we make this guidance tool-agnostic and reusable across Claude Code, Cursor, and future
tools, without duplicating content or breaking any tool that exists today?

## Decision Drivers

- One **source of truth** for rules/skills/commands — no copies to keep in sync.
- **Portability**: usable by Claude Code, Cursor, and other agents.
- **Non-breaking**: Cursor must keep working; Claude Code must discover skills/rules.
- Minimal moving parts; plain files, no build step.

## Considered Options

- **Stay on `.cursor/`** — keep Cursor as the home; other tools get nothing.
- **Move everything into `.claude/`** — Claude-native, but Cursor-breaking and just as
  tool-locked in the other direction.
- **Generic `.agents/` as source of truth + symlink shims** — tool dirs point into `.agents/`.

## Decision Outcome

Chosen option: **a generic `.agents/` directory as the single source of truth, with each
tool's directory symlinked into it.** Content moved (history-preserving `git mv`) to
`.agents/{rules,skills,commands}`. Tool wiring:

- `.cursor/{rules,skills,commands}` → symlinks to `../.agents/*` (Cursor reads the same
  `.mdc`/`SKILL.md`/`.md` files through the link).
- `.claude/{skills,commands,rules}` → symlinks to `../.agents/*` (Claude Code discovers skills
  under `.claude/skills/`; rules/commands likewise).
- `AGENTS.md` is the canonical, tool-agnostic entry (overview + links into `.agents/` +
  preserved learned facts); `CLAUDE.md` imports it via `@AGENTS.md`.

Rule files keep the `.mdc` extension (plain markdown + YAML frontmatter) so Cursor still
recognizes them; every other tool reads them as ordinary markdown. Editors edit content under
`.agents/`, never the symlinks.

### Consequences

- Good: rules/skills/commands live once and are shared by all tools; Claude Code now sees the
  full guidance set; adding a new tool is one more symlink.
- Good: no duplication, no build step; `git mv` preserved file history.
- Bad: **symlinks need `git config core.symlinks true`** — Windows contributors without it (or
  without Developer Mode) get plain-text placeholder files instead of working links. Documented
  in `docs/CONTRIBUTE.md`.
- Neutral: depends on tools resolving symlinked directories (verified for Cursor and Claude
  Code on this Linux/WSL2 repo). If a future tool can't, it gets its own pointer/import instead.

## Pros and Cons of the Options

### Generic `.agents/` + symlinks (chosen)

- Good: single source of truth; every tool works; non-breaking; trivial to extend.
- Bad: symlink portability caveat on Windows.

### Stay on `.cursor/`

- Good: zero work.
- Bad: guidance stays invisible to Claude Code and other tools — the problem we are solving.

### Move into `.claude/`

- Good: Claude-native discovery with no symlinks.
- Bad: breaks Cursor; merely swaps one tool lock-in for another.

## More Information

- Layout & wiring: `.agents/` (source of truth); `.cursor/*` and `.claude/*` symlinks;
  `AGENTS.md` (canonical entry) imported by `CLAUDE.md`.
- ADR-authoring lives here too: `.agents/skills/write-adr/` and `.agents/skills/spec-to-adr/`.
- Related: [ADR-0001](0001-record-architecture-decisions-using-madr.md) (the ADR convention
  this guidance documents). Cross-references in `.specify/memory/constitution.md` updated to
  `.agents/rules/`.
