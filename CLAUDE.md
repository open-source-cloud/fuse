# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

FUSE (FUSE Utility for Stateful Events) is a workflow engine built on the **ergo actor model** (ergo.services) for creating automation pipelines. Each workflow node runs as an independent actor with message passing, supervision, and fault tolerance. The project is pre-production and under active development.

## Build & Development Commands

```bash
make build              # Build binary to bin/fuse
make test               # Run tests (requires gotestsum, outputs testdox format)
make lint               # Run golangci-lint (auto-installs if missing)
make lint-fix           # Auto-fix lint issues
make run                # Build + run server on port 9090 with debug logging
make swagger            # Generate Swagger docs (requires swag)
make test-benchmark     # Run benchmark tests

# Run a single test
go test -v -run TestName ./internal/workflow/

# Clear test cache
go clean -testcache

# Docker
make dkb                # Build Docker image (fuse-app:dev)
make dkx                # Run Docker container

# Docker Compose (single file, profile-based)
make infra-up           # PG + S3 (rustfs) + etcd (local dev, fuse on host)
make ha-up              # Full HA: 3 Fuse nodes + infra (built from source)
make ha-down            # Tear down HA stack
```

**Quality gate order before committing: `make lint && make build && make test`**

## Architecture

### Layered Structure

```
cmd/fuse/main.go          → CLI entrypoint (cobra commands: server, workflow, mermaid)
internal/app/di/           → uber-go/fx dependency injection modules (AllModules composes everything)
internal/app/config/       → Config via caarlos0/env (environment variables)
internal/actors/           → ergo actor implementations (supervisors, pools, workers)
internal/handlers/         → HTTP handlers (extend base Handler, actor-based WebWorker pattern)
internal/services/         → Business logic layer
internal/repositories/     → Data access (interfaces + in-memory implementations)
internal/workflow/         → Core workflow logic (graph, nodes, edges)
internal/packages/         → Function package registry
internal/messaging/        → Actor message type definitions
internal/dtos/             → Data transfer objects
pkg/                       → Public importable libraries (workflow metadata, transport, uuid, http, store)
```

### Key Patterns

- **Actor Factory**: All actors use `ActorFactory[T gen.ProcessBehavior]` for creation via DI
- **Repository Pattern**: Interfaces in `internal/repositories/` with `Memory*` implementations (`*_memory.go`). Memory repos use `sync.RWMutex` for thread safety
- **DI Modules**: Each layer has an fx.Module in `internal/app/di/` — composed into `AllModules`
- **HTTP Handlers**: Extend base `Handler` struct in `internal/handlers/handler.go`, implement verb methods (`HandleGet`, `HandlePost`, etc.)
- **Config**: Environment variables with struct tags (`env:"VAR_NAME" envDefault:"value"`)

### Core Domain

- **Workflow**: Complete automation definition with a graph of nodes and edges
- **Node**: Actor that executes a function from a registered package; has input/output metadata
- **Edge**: Connection between nodes, optionally conditional (uses expr-lang/expr for expressions)
- **Graph**: Contains nodes, edges, trigger node, and thread IDs for execution ordering
- **Package**: Registry of available functions that nodes can reference

### REST API (default port 9090)

- `GET /health` — Health check
- `POST /v1/workflows/trigger` — Execute workflow
- `PUT/GET /v1/schemas/{schemaID}` — Manage workflow schemas
- `GET /v1/packages`, `PUT/GET /v1/packages/{packageID}` — Manage packages
- `POST /v1/workflows/{workflowID}/execs/{execID}` — Submit async function results
- Swagger UI at `/docs`

## Tech Stack

- **Go 1.26**, **ergo.services** (actor model), **uber-go/fx** (DI), **cobra** (CLI)
- **In-memory repositories** for workflow/graph/package state (dev and default runtime)
- **zerolog** (structured logging), **go-playground/validator** (validation)
- **gorilla/mux** (routing), **swag** (Swagger generation)
- **stretchr/testify** (test assertions), **gotestsum** (test runner)
- **golangci-lint v2** (linting, config in `.golangci.yml`)

## Conventions

- **Conventional commits**: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`, `perf:`
- **Branch naming**: `feat/`, `fix/`, `docs/`, `refactor/`, `test/`, `chore/`
- **Test files**: Co-located as `*_test.go`, use table-driven tests and Arrange-Act-Assert
- **Cyclomatic complexity**: Max 15 (enforced by golangci-lint)
- **Actor logging**: Use `a.Log()` inside actors, not the global logger
- Detailed coding rules are in `.cursor/rules/` (13 files covering Go conventions, actors, repos, handlers, testing, DI, concurrency, etc.)
