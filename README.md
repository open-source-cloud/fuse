# FUSE

[![CI](https://github.com/open-source-cloud/fuse/actions/workflows/ci.yml/badge.svg)](https://github.com/open-source-cloud/fuse/actions/workflows/ci.yml)
[![E2E](https://github.com/open-source-cloud/fuse/actions/workflows/e2e.yml/badge.svg)](https://github.com/open-source-cloud/fuse/actions/workflows/e2e.yml)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/open-source-cloud/fuse)](go.mod)

**FUSE** is an open-source workflow engine built on the [ergo actor model](https://docs.ergo.services/). Each workflow step runs as an isolated actor with message passing, supervision, and fault tolerance — designed for **AI workflows**, **agentic orchestration**, and **event-driven automation**.

Built and maintained by [Uranus Technologies](https://uranus.com.br).

## Features

- **Graph-based workflows** — Define automation pipelines as directed graphs of nodes and edges
- **Actor-per-node execution** — Each node runs as an independent ergo actor with supervision and fault isolation
- **Control flow** — Conditional branching (expr-lang), parallel execution, join/merge strategies, graph-level loops, sub-workflows
- **Resilience** — Configurable retries with backoff, per-node and workflow-level timeouts, error edges for recovery paths
- **Durable execution** — Journal-based state persistence, crash recovery, and resume from last checkpoint
- **Async nodes** — External completion via HTTP callback for human-in-the-loop and long-running steps
- **Awakeables** — Durable sleep and wait-for-event primitives
- **Schema versioning** — Version, activate, and rollback workflow definitions
- **High availability** — Multi-node clustering with ergo, claim-based workflow distribution, etcd or static peer discovery
- **Helm chart** — Production-ready Kubernetes deployment (standalone or clustered StatefulSet)

## Architecture

```
                    ┌─────────────────────────────────────────────┐
                    │                 FUSE Node                   │
                    │                                             │
  HTTP Request ───> │  HTTP Mux ──> Workflow Supervisor           │
                    │                    │                        │
                    │               Instance Sup                  │
                    │              /            \                 │
                    │    Workflow Handler    Function Pool         │
                    │         │                  │                │
                    │    Graph Engine       Actor Workers         │
                    │                                             │
                    │  ┌───────────┐  ┌──────┐  ┌────────────┐   │
                    │  │ PostgreSQL│  │  S3  │  │    etcd     │   │
                    │  │ (state)   │  │(data)│  │ (discovery) │   │
                    │  └───────────┘  └──────┘  └────────────┘   │
                    └─────────────────────────────────────────────┘
```

### How it works

1. **Define** a workflow schema as a JSON graph of nodes and edges
2. **Register** function packages that nodes reference (internal Go functions or external HTTP services)
3. **Trigger** a workflow instance via the REST API
4. The engine walks the graph: each node spawns as an actor, executes its function, and routes output along edges
5. Conditional edges evaluate expressions (expr-lang) to choose the next path
6. Parallel branches fan out to concurrent actors, then merge at join nodes
7. Failed nodes retry or route through error edges to recovery paths
8. Async nodes pause execution and resume when an external system calls back

## Quick start

**Prerequisites:** Go 1.26+, Make

```bash
# Clone
git clone https://github.com/open-source-cloud/fuse.git
cd fuse

# Build and run
make build
make run        # Starts server on port 9090 with debug logging
```

The API is available at `http://localhost:9090`. Interactive Swagger docs at `http://localhost:9090/docs`.

### Run with infrastructure

For PostgreSQL, S3 (rustfs), and etcd:

```bash
make infra-up   # Start PG + S3 + etcd
make run        # Run FUSE on host against local infra
```

### Docker

```bash
make dkb        # Build image (fuse-app:dev)
make dkx        # Run container on port 9090
```

### HA cluster (3 nodes)

```bash
make ha-up      # Build + start 3 FUSE nodes + PG + S3 + etcd
make ha-down    # Tear down
```

## API

| Method | Path | Description |
| ------ | ---- | ----------- |
| `GET` | `/health` | Health check |
| `POST` | `/v1/workflows/trigger` | Start a workflow instance |
| `PUT` | `/v1/schemas/{schemaID}` | Create or update a workflow schema |
| `GET` | `/v1/schemas/{schemaID}` | Get a workflow schema |
| `GET` | `/v1/packages` | List function packages |
| `GET` | `/v1/packages/{packageID}` | Get a package |
| `PUT` | `/v1/packages/{packageID}` | Register or update a package |
| `POST` | `/v1/workflows/{workflowID}/execs/{execID}` | Submit async function result |

Full API documentation: [docs/API.md](docs/API.md) | Swagger UI: `http://localhost:9090/docs`

### Example: trigger a workflow

```bash
# 1. Upload a schema
curl -X PUT http://localhost:9090/v1/schemas/my-workflow \
  -H "Content-Type: application/json" \
  -d @examples/workflows/small-test.json

# 2. Trigger execution
curl -X POST http://localhost:9090/v1/workflows/trigger \
  -H "Content-Type: application/json" \
  -d '{"schemaID": "my-workflow"}'
```

See [`examples/workflows/`](examples/workflows/) for more schema examples including conditional branching, parallel execution, retries, error edges, timeouts, sub-workflows, and awakeables.

## Testing

FUSE has a comprehensive 4-layer test strategy:

```
Unit tests          100 test files across pkg/ and internal/
                    Table-driven tests, Arrange-Act-Assert pattern

Functional tests    Contract tests against real PostgreSQL
                    Repository behavior validation

E2E fast tier       Full stack (3 FUSE nodes + PG + S3 + etcd)
                    Workflow execution, resilience, orchestration

E2E slow tier       Long-running scenarios (main branch only)
                    Persistence, integration, stress testing
```

```bash
make test               # Unit tests
make test-functional    # Functional tests (requires PostgreSQL)
make test-benchmark     # Benchmark tests
make e2e-local          # Full E2E suite (builds Docker, starts infra)
```

## CI/CD Pipeline

Every change goes through a multi-stage pipeline:

```
CI                          E2E                         CD
├── Lint (golangci-lint)    ├── Fast tier (always)      ├── Semantic release
├── Build                   └── Slow tier (main only)   ├── Multi-arch Docker image
├── Unit tests                                          │   (linux/amd64, linux/arm64)
├── Benchmarks                                          ├── Trivy vulnerability scan
├── Helm lint                                           └── Helm chart publish (OCI)
└── Functional tests (PG)
```

Container images are published to [DockerHub](https://hub.docker.com/r/uranustechnologies/fuse) on every release.

## Deployment

FUSE ships with a Helm chart for Kubernetes. Supports `standalone` (Deployment) and `cluster` (StatefulSet with ergo clustering) modes.

```bash
helm install fuse ./deploy/helm/fuse \
  --namespace fuse --create-namespace \
  --set image.tag=latest
```

Full deployment guide: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)

## Tech stack

| Component | Technology |
| --------- | ---------- |
| Language | Go 1.26 |
| Actor model | [ergo.services](https://docs.ergo.services/) |
| Dependency injection | [uber-go/fx](https://github.com/uber-go/fx) |
| CLI | [cobra](https://github.com/spf13/cobra) |
| HTTP routing | [gorilla/mux](https://github.com/gorilla/mux) |
| Database | PostgreSQL 17 (pluggable, in-memory for dev) |
| Object store | S3-compatible (pluggable, in-memory for dev) |
| Service discovery | etcd (for cluster mode) |
| Logging | [zerolog](https://github.com/rs/zerolog) |
| API docs | [swag](https://github.com/swaggo/swag) (Swagger 2.0) |
| Testing | [testify](https://github.com/stretchr/testify), [gotestsum](https://github.com/gotestyourself/gotestsum) |
| Linting | [golangci-lint v2](https://golangci-lint.run/) |
| Container | Distroless (gcr.io/distroless/static-debian12) |

## Project structure

```
cmd/fuse/                  CLI entrypoint (server, workflow, mermaid, migrate, seed)
internal/
  app/config/              Configuration (env vars with struct tags)
  app/di/                  Dependency injection modules (uber-go/fx)
  actors/                  ergo actor implementations
  handlers/                HTTP handlers (WebWorker pattern)
  services/                Business logic layer
  repositories/            Data access (interfaces + PostgreSQL/memory implementations)
  workflow/                Core workflow engine (graph, nodes, edges, execution)
  packages/                Function package registry
  messaging/               Actor message types
  dtos/                    Data transfer objects
pkg/                       Public libraries (workflow types, transport, uuid, http, store)
tests/
  e2e/                     End-to-end test suites
  functional/              Contract tests (PostgreSQL)
deploy/helm/fuse/          Helm chart for Kubernetes
examples/workflows/        Example workflow schemas
```

## Contributing

See [docs/CONTRIBUTE.md](docs/CONTRIBUTE.md) for guidelines.

Quality gate before every commit:

```bash
make lint && make build && make test
```

## License

Copyright 2025 [Uranus Technologies](https://uranus.com.br)

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.
