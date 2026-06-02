# 0021. Deployment and delivery architecture (CI/CD, Docker, Helm)

- Status: Accepted
- Date: 2026-06-01
- Deciders: FUSE maintainers

> Backfill ADR: records the existing delivery and deployment architecture.

## Context and Problem Statement

FUSE must ship reliably from commit to a running, highly-available deployment. We need to define
how code is gated and released, how the artifact is built, how it is packaged for Kubernetes, and
how the HA topology ([ADR-0018](0018-high-availability-and-clustering.md)) is expressed in
infrastructure — in a way that's reproducible for contributors and operators.

## Decision Drivers

- Enforce the quality gate (lint → build → test) and integration (E2E) before release.
- Reproducible, small, secure runtime image.
- Automated, semantic versioning of both the app image and the deployable chart.
- A Kubernetes deployment that supports both standalone and clustered/HA modes, plus a
  one-command local stack mirroring it.

## Considered Options

- **GitHub Actions (CI → E2E → CD) + semantic-release + multi-arch distroless image +
  Helm/OCI chart, with profile-based docker-compose for local/E2E.**
- **Manual/tag-driven releases.**
- **Plain Kubernetes manifests (kustomize) instead of Helm.**

## Decision Outcome

Chosen option: **a fully automated GitHub Actions pipeline producing a distroless image and an
OCI Helm chart, with a profile-based docker-compose mirror for local dev and E2E.**

- **CI** (`.github/workflows/ci.yml`): on push/PR — Go 1.26, `make swagger/lint/build/test`
  (golangci-lint v2, gotestsum) + benchmarks, `helm lint`/`template` validation, then functional
  tests against a real Postgres 17 service; PRs additionally build and push a `pr-<n>` image
  (DockerHub, gated by `environment: prod`).
- **E2E** (`e2e.yml`): after CI — builds `fuse-app:test`, brings up the `e2e` docker-compose
  profile (3 nodes + Postgres + S3/rustfs + etcd), health-checks, runs `-tags=e2e` tests; a
  `e2e-slow` tier runs on `main`.
- **CD** (`cd.yml`): after E2E on `main` — `go-semantic-release` derives the version from
  conventional commits; on a release it builds a **multi-arch (amd64/arm64)** image, tags
  `:latest`/`:<version>`, Trivy-scans (SARIF → GitHub Security), and **publishes the Helm chart
  to a separate OCI repo** (`fuse-chart`) after bumping its version. App image and chart are
  distinct repositories.
- **Image** (`Dockerfile`): multi-stage — `golang:1.26` builds a static `CGO_ENABLED=0` binary
  (with swagger gen); runtime is `gcr.io/distroless/static-debian12`, non-root, `EXPOSE 9090`.
- **Helm chart** (`deploy/helm/fuse/`): `mode: standalone` → Deployment, `mode: cluster` →
  StatefulSet (headless service, `POD_NAME`/`POD_IP`/`CLUSTER_NODE_NAME` injected, acceptor port
  for ergo). Values cover image, cluster (static or etcd discovery, cookie, acceptor), database,
  object store, HA intervals, service, `/health` probes, and HPA (cluster + etcd discovery).
- **Local/E2E** (`docker-compose.yml`): profiles `infra` (PG + rustfs S3 + etcd), `ha` (3 nodes
  built from source + nginx round-robin LB), `e2e` (3 nodes from the prebuilt image + migrate
  init). This mirrors the K8s HA topology for `make infra-up`/`ha-up`.

### Consequences

- Good: every release passes the same gates (lint/build/test → functional → E2E); versions are
  automated and traceable; the runtime image is small, static, non-root, and scanned.
- Good: one chart serves standalone and HA; the compose stack reproduces the cluster locally.
- Good: separating app image and chart repos lets them version independently.
- Bad: the pipeline is GitHub-Actions- and DockerHub-specific (CI provider / registry lock-in);
  schema replication under HA remains best-effort ([ADR-0018](0018-high-availability-and-clustering.md)).
- Neutral: in-memory state means multi-replica standalone needs sticky sessions or `mode: cluster`;
  HA correctness needs Postgres + shared object store.

## Pros and Cons of the Options

### Actions + semantic-release + distroless + Helm/OCI (chosen)

- Good: automated, gated, reproducible, secure image, K8s-native, local parity.
- Bad: CI/registry-specific; more moving parts to maintain.

### Manual/tag-driven releases

- Good: simple, full control.
- Bad: error-prone, inconsistent gating, no automated semver.

### Raw manifests / kustomize

- Good: no templating engine.
- Bad: harder to parameterize standalone-vs-cluster, HA, and external deps; no chart distribution.

## More Information

- CI/CD: `.github/workflows/{ci,cd,e2e}.yml`; release via `go-semantic-release`;
  `scripts/bump-version.sh`. Image: `Dockerfile`. Chart: `deploy/helm/fuse/` (+ `deploy/k3s`,
  `deploy/kind`, `deploy/nginx`). Local: `docker-compose.yml` profiles; `docs/DEPLOYMENT.md`,
  `docs/SETUP.md`.
- Related: [ADR-0018](0018-high-availability-and-clustering.md),
  [ADR-0003](0003-in-memory-repositories-by-default.md),
  [ADR-0019](0019-object-store-payload-externalization.md).
