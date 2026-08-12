# AGENTS.md

Canonical, tool-agnostic guidance for AI agents working in this repository (Claude Code,
Cursor, and others). Tool-specific entry points point here.

## AI guidance map

Project guidance lives in a tool-agnostic home and is shared with every tool via symlinks:

- **`.agents/rules/`** — coding rules (Go conventions, actors, repositories, handlers,
  testing, DI, concurrency, …). Indexed in [`.agents/rules/README.mdc`](.agents/rules/README.mdc).
- **`.agents/skills/`** — reusable skill packs (architecture patterns), ADR-authoring skills
  (`write-adr`, `spec-to-adr`), and the engine knowledge packs listed below.
- **`.agents/agents/`** — subagent definitions for the backlog delivery pipeline (below).
- **`.agents/commands/`** — GitHub Spec Kit commands (`speckit.*`).
- **`docs/adr/`** — Architecture Decision Records ([index](docs/adr/README.md)); see
  [ADR-0009](docs/adr/0009-portable-ai-agent-guidance.md) for why guidance lives in `.agents/`.
- **`.specify/memory/constitution.md`** — project constitution (governing principles).

Tool wiring: `.cursor/{rules,skills,commands}` and `.claude/{rules,skills,commands,agents}` are
symlinks into `.agents/`; `CLAUDE.md` imports this file via `@AGENTS.md`. Edit content under
`.agents/` (the single source of truth), never the symlinks.

## Delivering backlog work

The work queue is [`BACKLOG_V2.md`](BACKLOG_V2.md) ("FUSE — product shape and backlog").
`BACKLOG.md` is **superseded** and its task ids were renumbered wholesale — never cite them.

Start with the **`backlog-task`** skill (`/backlog-task F-02`). It runs one task through
verify → research → plan → implement → review → gates, delegating to these subagents:

| Stage | Subagent |
| ----- | -------- |
| Reproduce the claim with an executed failing test (never fixes it) | `fuse-premise-verifier` |
| Read-only cross-subsystem research, CONFIRMED vs INFERRED per claim | `fuse-engine-researcher` |
| Verified premise → plan (tests, driver parity, HA, migration, ADR/spec call) | `fuse-implementation-planner` |
| Write the remote node protocol spec — Tier B, before any Go | `fuse-protocol-spec-author` |
| Test-first implementation under `.agents/rules/` + the quality gates | `fuse-go-implementer` |
| Real-process SIGKILL/restart durability harness (F-02) | `fuse-e2e-harness-engineer` |
| Adversarial review: replay, actors, races, parity, scope, contract | `fuse-code-reviewer` |

Engine knowledge packs, loaded by tier: `durable-execution-internals` and `crash-resume-testing`
(F/A), `persistence-and-migrations` (F-03, A-05, A-06), `observability-tracing` (A-04, C-04),
`function-package-authoring` (in-process nodes), `remote-node-protocol` (Tier B — the pillar),
`capability-registry-and-agents` (B-03/B-04, Tier C).

Two rules the backlog treats as non-negotiable: **a premise that does not reproduce closes its
task** (that is a success, not a failure), and **once the B-01 protocol spec is published it is a
contract with strangers** — treat it with more care than engine internals.

## Learned User Preferences

- Prefer dev tooling to be installable from the Makefile so `make test` and `make lint` work on fresh machines without manual binary setup.
- Avoid adding Python to the Makefile or repo scripts for maintenance tasks such as bulk-formatting JSON under `data/`; prefer shell, `jq`, or Go-based tooling.

## Learned Workspace Facts

- The module targets Go 1.26; local toolchains, CI workflows, and the Dockerfile should match.
- golangci-lint must be built with a Go version at least as new as `go.mod`; use `make install-lint` (`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0`). Prebuilt install scripts can embed an older Go and fail with a version-mismatch error when loading `.golangci.yml`.
- `make test` runs `install-gotestsum` first (`go install gotest.tools/gotestsum@latest` into `GOPATH/bin`).
- With ergo v3.2+, sending to sibling processes during actor `Init` is supported; workflow init runs inline in `WorkflowHandler.Init` instead of an `ActorInit` / `actor:init` self-message. Async completion from goroutines outside `HandleMessage` (e.g. timer callbacks) should deliver messages via `Node().Send`, not `Process.Send`, when the worker can be in Sleep (ergo restricts `Process.Send` to Init/Running/Terminated).
- Example workflows are seeded with `./bin/fuse seed examples --ci`, where `--ci` skips `github-request-example` and any workflow referencing `fuse/pkg/logic/timer`, because the default server path does not complete async timer flows in CI. (A `make examples-ci` target and `scripts/run-example-workflows.sh` were removed; `--ci` is the surviving equivalent.) Multiple files under `examples/workflows/` reuse the same schema `id` (e.g. `small-test`); upserting them in order means the last file wins for that id.
- Run state is read with `GET /v1/workflows/{workflowID}` (`internal/actors/mux_worker.go:220`); the sibling routes are `/cancel`, `/snapshot`, `/retry-node`, `/retry` and `/trace`. There is **no** `/status` route. The bare route returns `{workflowId, status}` only — richer mid-flight state is backlog task C-09.
- `MemoryPackageRepository` uses a mutex because internal package registration saves packages from concurrent goroutines.
- `make dockerfile-lint` runs Hadolint in Docker on `Dockerfile` with `.hadolint.yaml`; it catches consecutive-`RUN` patterns that align with SonarCloud Docker rules (e.g. docker:S7031). `make sonar-local` uses `sonar-project.properties` and Dockerized SonarScanner; set `SONAR_TOKEN` from SonarCloud (analysis-capable token, not a GitHub token). It passes the current git branch and commit; if analysis fails with HTTP 404 on `analysis/analyses`, try `SONAR_REGION=us` for US-hosted SonarCloud orgs.
- Default filesystem object store path is `./data/fuse` (`OBJECT_STORE_FS_BASE_PATH`) so local runs on macOS do not try to create `/data` on the read-only root volume; with Docker and the repo mounted at `/app`, the same default resolves under `/app/data/fuse`. Use an explicit absolute path (for example `/data/fuse`) in production when a volume is mounted there.
- `fuse migrate` applies embedded PostgreSQL migrations using `DB_POSTGRES_DSN` without starting the HTTP server or actor runtime; `make migrate` builds then runs `./bin/fuse migrate`. `fuse seed examples` upserts every `examples/workflows/*.json` graph schema through `GraphService` with the same DB and object-store settings as the server, but without HTTP or ergo; use `--ci` to skip `github-request-example.json` and any file whose JSON references `fuse/pkg/logic/timer`. After CLI changes, rebuild `bin/fuse`—an older binary will not expose new subcommands.
- All Docker infrastructure is in a single `docker-compose.yml` with profiles: `infra` (PG+S3+etcd only), `ha` (3 Fuse nodes built from source), `e2e` (3 nodes from pre-built image with migrate init). Use `make infra-up`, `make ha-up`, or `docker compose --profile e2e up`.
- `LOG_FORMAT` env var (default `json`) controls log output format. Set `LOG_FORMAT=console` for human-readable colorized output (local dev). The CLI flag `--log-format` overrides the env var when provided. JSON mode uses RFC3339 timestamps and disables ANSI colors in all loggers (app, ergo, fx).
