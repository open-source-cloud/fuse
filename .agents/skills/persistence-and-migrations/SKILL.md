---
name: persistence-and-migrations
description: Explains how FUSE persists state — the repository interface/memory/postgres triad and its parity contract, the real table shapes, the exact idiom for adding a migration and how `fuse migrate` applies it, the object-store ref convention, and the driver combinations that are silently unrecoverable. Use when adding or changing a repository, writing a migration, touching Config.Validate or the object store, or working F-03, A-05 or A-06.
---

# persistence-and-migrations

Backlog ids below are from the repo-root backlog (`BACKLOG_V2.md`). *If it has been renamed to
`BACKLOG.md`, the canonical file is the one titled "FUSE — product shape and backlog".*

## When to use

- Adding a column, a table, or a repository method.
- Anything that reads or writes a payload through the object store.
- F-03 (config validation for driver combinations), A-05 (pin runs to a schema version),
  A-06 (stop the shared-pointer live edit).

## Ground truth

| File | Authoritative for |
| --- | --- |
| `internal/repositories/<name>.go` | The interface + its `Err…NotFound` sentinels |
| `internal/repositories/<name>_memory.go` | The memory driver |
| `internal/repositories/postgres/<name>.go` | The Postgres driver + its object-store key builder |
| `internal/repositories/postgres/migrations/` | 11 embedded migrations, `000001`…`000011` |
| `internal/repositories/postgres/db.go` | `RunMigrations` (L19) + `convertDSNForMigrate` (L50) |
| `internal/app/di/repos.go` | Driver selection for all 9 repositories |
| `internal/app/di/objectstore.go` | Object-store driver switch |
| `internal/app/config/config.go` | Every env var; `Validate()` at L175 |
| `tests/functional/` | The driver-parity contract harness |
| `docs/adr/0003-in-memory-repositories-by-default.md`, `docs/adr/0019-object-store-payload-externalization.md` | Why there are two of everything, and the ref convention |

⚠️ `.agents/rules/05-repositories.mdc` is **stale**: it still states FUSE has in-memory
implementations only ("New types can satisfy the same interfaces later if a durable store is
introduced"). The Postgres driver has existed since migration `000001`. Treat `tests/functional/` as
the real convention.

## The triad and the parity contract

Every repository is **interface + `Memory*` + `postgres.*`**, selected in `internal/app/di/repos.go`
by one guard repeated for each provider:

```go
if p.Config.Database.Driver == config.DBDriverPostgres && p.Pool != nil { /* postgres */ }
// else memory
```

`config.DBDriverPostgres` (`config.go:13`) is the **only** driver constant; `"memory"`,
`"filesystem"` and `"s3"` are bare string literals at their use sites.

Parity is enforced by shared contract functions, not duplicated tests:

```go
// tests/functional/graph_repository_test.go  (UNTAGGED — runs in `make test`)
func contractTestGraphRepository(t *testing.T, newRepo func() repositories.GraphRepository, reset func())
func TestMemoryGraphRepository_Contract(t *testing.T) { contractTestGraphRepository(t, repositories.NewMemoryGraphRepository, func(){}) }

// tests/functional/postgres_test.go  (//go:build functional — runs in `make test-functional`)
func TestPostgresGraphRepository_Contract(t *testing.T) { /* same body, TRUNCATE reset closure */ }
```

Eight contracts exist today: Graph, Journal, Workflow (+ SubWorkflowRefs), Package, Environment,
Credential, Claim (Postgres-only), Awakeable. `postgres_test.go` creates schema `fuse_functional`,
runs the migrations into it via a `search_path` DSN (`withSearchPath`, L35) and truncates between
tests. **Behaviour you add belongs in the shared body**, so both drivers are held to it.

⚠️ **Embedding the interface in the struct defeats compile-time completeness checks.**
`MemoryGraphRepository`, `MemoryWorkflowRepository`, `postgres.WorkflowRepository`,
`postgres.GraphRepository` and `postgres.PackageRepository` all embed their interface (the pattern
`.agents/rules/05-repositories.mdc` prescribes). Add a method to an interface, implement it in one
driver only, and everything still compiles — the other driver nil-panics at runtime. Only
`postgres/environment.go` and `postgres/credential.go` use `var _ repositories.X = (*T)(nil)`.
**Implement both drivers in the same commit and add the case to the shared contract test.**

## Real table shapes (read from the migration SQL)

`workflows` — `000001` L34-44, plus `snapshot_ref` (`000003`) and `environment` (`000009`):

```sql
id BIGSERIAL PRIMARY KEY, workflow_id VARCHAR(36) NOT NULL UNIQUE,
schema_id VARCHAR(128) NOT NULL, state workflow_state NOT NULL DEFAULT 'untriggered',
output_ref VARCHAR(512), claimed_by VARCHAR(128), claimed_at TIMESTAMPTZ,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
-- + snapshot_ref VARCHAR(512)          (000003)
-- + environment VARCHAR(128) NOT NULL DEFAULT 'default'  (000009, indexed)
```

**There is no version column.** That is exactly A-05.

`journal_entries` — `000001` L69-90: `workflow_id VARCHAR(36)`, `sequence BIGINT`,
`entry_type journal_entry_type`, `thread_id SMALLINT`, `function_node_id VARCHAR(128)`,
`exec_id VARCHAR(64)`, `state workflow_state`, `parent_threads SMALLINT[]`,
`input_ref` / `result_ref` / `data_ref VARCHAR(512)`, FK to `workflows(workflow_id)`, and
**`CREATE UNIQUE INDEX idx_journal_wf_seq ON journal_entries (workflow_id, sequence)`**.

`graph_schema_versions` — `000007`: `(schema_id, version, definition_ref, created_by, comment,
is_active)` with `uq_schema_version UNIQUE (schema_id, version)`, plus
`graph_schemas.active_version INTEGER NOT NULL DEFAULT 1` and a back-fill `INSERT … SELECT`.
A-05's retrieval half is already built on top of this (see below).

`awakeables` — `000001` L139-154: already carries `deadline_at TIMESTAMPTZ`, which is what the
expiry sweeper **proposed** by C-06 would read. No sweeper exists today.

Column-type conventions: `VARCHAR(36)` workflow/awakeable ids, `VARCHAR(64)` exec ids, `VARCHAR(128)`
schema/node/package ids and environment names, `VARCHAR(512)` **every** object-store ref,
`SMALLINT` thread ids, `SMALLINT[]` parent threads, `BIGINT` sequences and ns durations, `BYTEA` ciphertext.

## Adding a migration — copy the newest existing one

Files are `internal/repositories/postgres/migrations/NNNNNN_snake_title.{up,down}.sql`, 6-digit
zero-padded, strictly sequential, **both files always present** even when the down is a comment
(`000004`). They are picked up by `//go:embed migrations/*.sql` (`db.go:15`) — there is no list to update.
**The current head is `000011`, so the next number is `000012`.**

The canonical ADD COLUMN idiom is `000009_add_workflow_environment`, verbatim:

```sql
-- ADR-0031 Phase 3: environment becomes a per-execution scoping dimension.
-- ... 2-4 lines saying WHY, and why the default backfills in-flight rows.
ALTER TABLE workflows ADD COLUMN environment VARCHAR(128) NOT NULL DEFAULT 'default';
CREATE INDEX idx_workflows_environment ON workflows (environment);
```

```sql
DROP INDEX IF EXISTS idx_workflows_environment;          -- index first
ALTER TABLE workflows DROP COLUMN IF EXISTS environment;  -- IF EXISTS on both
```

`NOT NULL` + `DEFAULT` is mandatory when the table can already hold in-flight rows. CREATE TABLE
follows `000001`/`000010`: `id BIGSERIAL PRIMARY KEY`, business key `VARCHAR(n) NOT NULL UNIQUE`,
`created_at`/`updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`, named constraints `fk_<short>` /
`uq_<short>`, indexes `idx_<table>_<discriminator>`. (`000011_create_credentials` deliberately breaks
the BIGSERIAL rule — its `id` is the business `VARCHAR(128)` PK. Not the norm.)

**How it is applied.** `postgres.RunMigrations(dsn)` (`db.go:19`) opens the embedded FS via `iofs`,
runs golang-migrate with the pgx/v5 driver, tolerates `migrate.ErrNoChange`. It is reached from two
places: eagerly at fx **Provide** time inside `providePgxPool` (`internal/app/di/database.go`, after
`pool.Ping`, with a comment that it must precede package registration), and from
`fuse migrate` (`internal/app/cli/migrate.go`), which starts neither fx, nor ergo, nor HTTP.

⚠️ Migrations run **on every node** in the compose `ha` profile (all three run the server). The `e2e`
profile deliberately uses a dedicated `fuse-migrate` init container and gates the nodes on
`service_completed_successfully`. Prefer the e2e shape when concurrent DDL matters.

⚠️ Migrations target whatever `search_path` the DSN carries (`fuse_functional` for functional tests,
`fuse_e2e` for the e2e stack). A migration that hard-codes a schema-qualified name breaks both.

⚠️ **Enum migrations are effectively irreversible.** `000004` adds a `journal_entry_type` value and its
`.down.sql` is a comment explaining that PostgreSQL cannot remove enum values. A new journal entry
type needs its own `ALTER TYPE … ADD VALUE` migration.

## Memory-driver pointer-sharing hazards (A-06)

**Memory repos hand out live pointers; Postgres repos hand out fresh objects.**

| Site | Behaviour |
| --- | --- |
| `internal/repositories/graph_memory.go:31-39` `FindByID` | `return graph, nil` — the exact `*workflow.Graph` that `Save` stored (L48) and that `workflow_handler.go:168` hands to `internalworkflow.New` (L177) |
| `internal/repositories/postgres/graph.go:37-67` `FindByID` | rebuilds a `Graph` from `schemas/{id}/definition.json` every call — structurally immune |
| `workflow_memory.go` `Get` | returns the stored `*Workflow` |
| `awakeable_memory.go` `FindByID` | returns the stored `*Awakeable` |

So `GraphService.updateVersioned` calling `graph.UpdateSchema(schema)`
(`internal/services/graph_service.go:260` → `internal/workflow/graph.go:122-133`, which replaces
`g.schema` and re-runs `compute()` in place) rewrites a **running** run's topology under the memory
driver. That is F-01.3 / **A-06**.

The fix idiom already exists further down the same file, in `FindByIDAndVersion`
(`graph_memory.go:78-99`): lines 89-90 do
`schema := sv.Schema.Clone(); workflow.NewGraph(&schema)`. For `FindByID`, `graph.Schema()`
(`internal/workflow/graph.go:148-150`) already returns a `Clone`. Expect
`IsNodesMetadataPopulated()` to go false afterwards so `GraphService.FindByID` repopulates — that is
the same cost Postgres already pays, not a regression.

⚠️ **`GraphSchema.Clone()` is not a total deep copy** (`internal/workflow/graph_schema.go:47-77`): it
deep-copies `Nodes`, `Edges`, `Metadata`, `Tags`, and value-copies `Concurrency`/`TriggerConfig`, but
assigns `Timeout: f.Timeout` — the `*GraphTimeoutConfig` pointer is **shared**. It is correct for
topology, not for everything.

## Object store

`pkg/objectstore/store.go` — `Put/Get/Delete/Exists`, keyed by a hierarchical string path; `Delete` is
documented idempotent; `Get` returns `ErrObjectNotFound`. Three drivers, chosen by
`OBJECT_STORE_DRIVER` in `internal/app/di/objectstore.go`: `"filesystem"`, `"s3"`, and a `default`
branch that falls through to **memory with no validation** — a typo yields the memory store silently.

**The ref convention (ADR-0019): the relational row never holds a payload.** The blob goes to the
object store under a hierarchical key and the row stores that key in a `*_ref VARCHAR(512)` column.
Key builders are unexported functions at the top of the owning Postgres file — add a new one there
rather than inlining a `Sprintf`:

| Builder | File | Key |
| --- | --- | --- |
| `journalObjectKey` | `postgres/journal.go:28` | `workflows/%s/journal/%d/%s` (`input.json`, `result.json`, `data.json`) |
| `workflowOutputKey` | `postgres/workflow.go:28` | `workflows/%s/output.json` |
| `graphObjectKey` / `graphVersionObjectKey` | `postgres/graph.go:32` / `:16` | `schemas/%s/definition.json`, `schemas/%s/v%d/definition.json` |
| `awakeableResultKey` | `postgres/awakeable.go:29` | `awakeables/%s/result.json` |
| `traceObjectKey` | `postgres/trace.go:29` | `workflows/%s/trace/%s/%s` |
| `packageObjectKey` | `postgres/package.go:31` | (see file) |

⚠️ Payload PUTs happen **inside** the DB transaction and are not rolled back — `postgres/journal.go`
`Append` writes up to three objects per entry before the INSERT. ADR-0019 accepts the orphans and
notes there is no GC. Journal keys are **sequence-addressed, not content-addressed**; A-09's
content-addressing idea changes this key contract, so dedup must happen at `Put` time.

## Driver combinations that are silently unrecoverable (F-03)

`Config.Validate()` (`config.go:175-182`) today checks **one** thing: etcd endpoints when
`CLUSTER_ENABLED=true` and `CLUSTER_DISCOVERY_MODE=etcd`. Everything below is unguarded.

| Combination | What happens | Evidence |
| --- | --- | --- |
| `DB_DRIVER=postgres` + `OBJECT_STORE_DRIVER=memory` (**the default object store**) | Durable journal rows whose `input_ref`/`result_ref`/`data_ref` point into a process-local map. Every payload fetch fails on the next boot; nothing is recoverable. This is F-03's headline case. | `postgres/journal.go` Append PUTs / LoadAll GETs; `di/repos.go` injects `p.Store` into the Postgres repos with no cross-check |
| `DB_DRIVER=postgres` + empty `DB_POSTGRES_DSN` | `providePgxPool` logs `Warn` and returns a **nil pool**; every provider takes the `Pool != nil` false branch and hands back a `Memory*` repo. The engine boots, looks healthy, persists nothing. | `internal/app/di/database.go` (`if cfg.Database.PostgresDSN == ""` → warn + empty result) |
| `HA_ENABLED=true` + `DB_DRIVER != postgres` | `MemoryClaimRepository.ClaimWorkflow` returns `true` unconditionally — zero mutual exclusion. | `internal/repositories/claim_memory.go:20-21` |

⚠️ **Coverage hole you must design around: `Config.Validate` is only reached by `fuse server`.** Its
sole caller is `internal/app/fuse.go:47` inside `app.NewApp`, provided by `FuseAppModule`.
`fuse migrate` (`cli/migrate.go`) and `fuse seed examples` (`cli/seed.go`, whose module list omits
`FuseAppModule`) never call it. Either call `Validate` in those `RunE` bodies too, or hoist the check
into a provider inside `DatabaseModule`/`ObjectStoreModule`, which both CLIs do include.

Style to follow (`config.go:175-182`): a flat sequence of `if <bad combination> { return
fmt.Errorf("<VAR> is required when <VAR>=<v> and <VAR>=<v>") }`, returning on the first failure,
naming **environment variables**, not Go field names. The table test belongs in
`internal/app/config/config_test.go` next to `TestConfig_Validate_EtcdRequiresEndpoints` (L39).

## A-05 landing map (PROPOSED — none of this exists yet)

What **exists today**: `FindByIDAndVersion` is fully implemented on `GraphService` and on both
repositories (`graph_memory.go:78-99`, `postgres/graph.go:180-211`) and has **no handler caller** —
A-05 gets the retrieval half for free. What does **not** exist: any record of the version a run
started on. `postgres/workflow.go:Get` → `loadGraph(ctx, schemaID)` (L241) resolves
`SELECT definition_ref FROM graph_schemas WHERE schema_id=$1` — the **active** definition — so a run
recovered after an edit replays against current topology.

Mechanical shape: new `000012_add_workflow_schema_version.{up,down}.sql` following the `000009` idiom;
add the column to the `Get` SELECT list and to the `Save` INSERT, **excluded from the
`ON CONFLICT … DO UPDATE SET`** exactly like `environment` and with the same style of comment
(`postgres/workflow.go` Save carries that comment today); resolve `loadGraph` by `(schema_id, version)`
against `graph_schema_versions.definition_ref`; mirror in `workflow_memory.go`; extend
`contractTestWorkflowRepository`.

## Traps

| Trap | Evidence |
| --- | --- |
| **Postgres enforces `UNIQUE (workflow_id, sequence)` on the journal; memory does not.** A replay bug that re-appends an existing sequence (A-07) errors loudly under Postgres and passes silently under memory. Never conclude "it works" from `make test` alone for journal changes. | `idx_journal_wf_seq` (`000001`); `journal_memory.go` Append is a plain slice append |
| **Awakeable `Resolve` semantics diverge.** Postgres only resolves rows still `pending` (`WHERE … AND status='pending'`, `RowsAffected()==0` → `ErrAwakeableNotFound`); memory overwrites unconditionally. Double-resolve is rejected under one driver and accepted under the other. | `postgres/awakeable.go:133-158` vs `awakeable_memory.go:55-65` |
| **`MemoryWorkflowRepository.FindExecutions` ignores `From`/`To` and returns zero timestamps**, while Postgres filters on `created_at`. | `workflow_memory.go` vs `postgres/workflow.go` FindExecutions |
| **`OBJECT_STORE_KEY_PREFIX` is dead config.** `ObjectStoreConfig.KeyPrefix` (`config.go:128`) is declared and referenced nowhere else — no driver, no key builder applies it. No multi-tenant key isolation comes from it. | grep `KeyPrefix` |
| **`LOG_LEVEL` is dead config.** `ParamsConfig.LogLevel` has **no** `env` tag; the level comes only from the cobra `-l/--loglevel` flag. `LOG_FORMAT` **is** tagged and works. | `config.go:77-82` |
| **`convertDSNForMigrate` never converts `postgresql://`.** `db.go:54` compares `dsn[:14]` against the 13-character literal `"postgresql://"`, which can never be equal, so such a DSN reaches golang-migrate unconverted and fails there rather than at config parse time. `postgres://` (L51) is fine. **No backlog task owns this** — file it with evidence, don't fix it in passing. | `internal/repositories/postgres/db.go:50-57` |
| **`setupTestPool`'s TRUNCATE list is incomplete** — it omits `environments`, `credentials`, `secrets`, `idempotency_keys`, `execution_traces`, `execution_trace_steps` and `graph_schema_versions`. A new table needs an explicit decision about that list and the per-test `reset` closures. | `tests/functional/postgres_test.go:48-92` |
| **The S3 driver has no contract test.** `pkg/objectstore/store_test.go:testObjectStore` runs for memory and filesystem only; `s3_test.go` covers endpoint normalisation alone. S3 behaviour is exercised only by the e2e stack against rustfs. | `pkg/objectstore/*_test.go` |
| **`make test-functional` passes vacuously without a DSN** — `testDSN` calls `t.Skip` when `DB_POSTGRES_DSN` is unset, so every Postgres contract test silently skips. | `tests/functional/postgres_test.go:25-32` |
| **`pkg/store` is not persistence.** It is the in-memory KV backing `Workflow.aggregatedOutput` and dot-notation input mapping. Do not confuse it with `pkg/objectstore`. | `pkg/store/kv.go` |

## Commands (confirmed to exist)

```bash
make lint && make build && make test        # the mandated gate, in this order
make swagger                                # REQUIRED on a fresh clone before build/test
make infra-up                               # docker compose --profile infra (PG 17, rustfs S3, etcd)
DB_POSTGRES_DSN='postgres://fuse:fuse@localhost:5432/fuse?sslmode=disable' make test-functional
go test -count=1 -run 'TestMemory.*_Contract' ./tests/functional/...
go test -tags=functional -count=1 -run TestPostgresGraphRepository_Contract ./tests/functional/...
go test -count=1 ./internal/app/config/...  # where the F-03 table test belongs
go test -count=1 ./pkg/objectstore/...
make migrate                                # go build then ./bin/fuse migrate (needs DB_POSTGRES_DSN)
make seed                                   # ./bin/fuse seed examples -l debug
make e2e-local                              # builds fuse-app:test, compose --profile e2e, -tags=e2e, down -v
psql "$DB_POSTGRES_DSN" -c '\d workflows'   # confirm the live column set before writing a migration
```

A schema change additionally needs `make test-functional` against a live PG — that is what actually
exercises the migration, and CI runs it as a separate job after the lint/build/test job.

$ARGUMENTS
