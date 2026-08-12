# V-b · Resume appends a second `subworkflow:started` for one exec id, and nothing catches it

|  |  |
| --- | --- |
| **Tier** | Unfiled (found during verification) — **the maintainer must assign it a tier and an id** |
| **Size** | S (unfiled; sizing is the maintainer's call together with the id) |
| **Status** | Not started |
| **Premise** | EXECUTED (`internal/actors/f01_2_verification_test.go`, during F-01.2) |
| **Depends on** | nothing to reproduce. The fix belongs with **A-02**, which depends on **F-02** |
| **Blocks** | nothing formally — but it constrains A-02's replay handler for `subworkflow:started` |
| **First agent** | `fuse-implementation-planner` (premise already executed; plan it as a constraint inside A-02, not as a separate PR) |
| **Skill packs** | `durable-execution-internals`, `persistence-and-migrations` |

## What the backlog says

> After a resume the journal holds two `subworkflow:started` entries for a single exec id with *different*
> `childWorkflowId` — `internal/actors/workflow_handler.go:917-926` appends unconditionally with no read
> of an existing entry. Any future replay handler for `subworkflow:started` must therefore tolerate or
> repair pre-existing duplicates rather than assume one entry per exec.
>
> Postgres would not catch it either: `sub_workflow_refs` is UNIQUE on `child_workflow_id` only — there is
> no unique index on the parent exec.

## Premise status

**EXECUTED.** It falls out of F-01.2, which reproduced the duplicate-child chain under real Go 1.26 with
a checked-in failing test. Limits inherited from F-01.2 and worth restating: no process was killed, no
ergo runtime, no Postgres — "restart" meant rebuilding the aggregate from the persisted journal. So the
duplicate *journal entry* is executed evidence; the claim that Postgres would not catch it is a schema
read, and both were re-confirmed while writing this dossier: `workflow_handler.go:917-926` is an
unconditional `Journal().Append(JournalEntry{Type: JournalSubWorkflowStarted, …})` with no prior read,
and `internal/repositories/postgres/migrations/000001_create_tables.up.sql:54` declares
`child_workflow_id VARCHAR(36) NOT NULL UNIQUE` with no unique constraint on `parent_exec_id` (the only
other index on the table is `idx_subwf_parent` on `parent_workflow_id`, `:66`).

Scope note carried from F-01.2: this is reachable for **synchronous** sub-workflows only. With
`async=true`, `handleSubWorkflowAction` calls `SetResultFor` at `workflow_handler.go:936`, journalling
`step:completed`, so `findPendingThreads` never re-issues the step and no second append happens.

## Code area

| Path | Symbol / line | Why it matters |
| --- | --- | --- |
| `internal/actors/workflow_handler.go` | `:917-926` unconditional `Journal().Append` of `JournalSubWorkflowStarted` | the write site — no read of an existing entry for the exec |
| `internal/actors/workflow_handler.go` | `:912` `SaveSubWorkflowRef`, `:936` `SetResultFor` (async branch) | the ref row written alongside; why async is immune |
| `internal/repositories/postgres/migrations/000001_create_tables.up.sql` | `:52-66` `sub_workflow_refs` | UNIQUE on `child_workflow_id` (`:54`) only; nothing keyed on the parent exec |
| `internal/workflow/workflow.go` | `replayJournalEntries` | where a future `subworkflow:started` handler lands, and must tolerate duplicates |

## Route

This rides with **A-02**, which owns replaying the intercepted system functions and is blocked on F-02.
Hand `fuse-implementation-planner` this dossier together with A-02's, so the plan states outright that
the new `subworkflow:started` replay handler must tolerate or repair pre-existing duplicates rather than
assume one entry per exec. Then `fuse-go-implementer` and `fuse-code-reviewer` as part of A-02's PR.
Do not open a separate PR unless the maintainer files this as its own task.

## Done when

*Derived — the finding carries no "Done when".*

- [ ] The maintainer has assigned this finding a tier and an id.
- [ ] A test asserts that after a resume there is exactly one `subworkflow:started` entry per exec id.
- [ ] A test asserts that a replay handler fed a journal that *already* contains duplicates does not
      spawn a second child (repair or tolerate — the plan says which).
- [ ] Whether a durable uniqueness constraint on `(parent_workflow_id, parent_exec_id)` is added to
      `sub_workflow_refs` is decided explicitly, with a migration if yes.
- [ ] F-02's `TestCrashResume_PendingSubWorkflow…` scenario is green under a real SIGKILL, not only
      under aggregate rebuild.

## Out of scope

Do not fix the duplicate-child spawn itself here — **A-02** owns it, and **F-02** owns proving it under a
real crash. Do not add recursion depth counters or cycle detection — **B-09** owns that. Do not wire
sub-workflow input or an output selector — **B-10**. Do not "just add" the unique index without the
replay handler: a constraint that fires on a legitimate resume converts a duplicate into a crash.

## Decisions

Probably **neither** an ADR nor a SpecKit spec on its own — it is a constraint on A-02's fix. If a
migration adds a uniqueness constraint on the parent exec, that is a persistence decision touching
**ADR-0032 · Sub-workflow composition: child workflows as first-class instances** and **ADR-0010 ·
Durable execution via an append-only journal**; record it in A-02's own ADR rather than a separate one.
Next free ADR number is **0035**.

## Log

- **2026-08-11** — Dossier created from BACKLOG_V2.md ("Found during verification — unfiled").
- **2026-08-11** — Raised by F-01.2 verification with executed evidence and deliberately **not fixed**,
  per "do not fix bugs found along the way". Write site and `sub_workflow_refs` schema re-confirmed by
  reading while writing this dossier; the synchronous-only scope correction from F-01.2 recorded above.
