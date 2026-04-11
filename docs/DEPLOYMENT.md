# Deploying FUSE on Kubernetes

FUSE ships a Helm chart under [`deploy/helm/fuse/`](../deploy/helm/fuse/). It targets any CNCF-compatible cluster (EKS, GKE, AKS, on-prem, etc.); wire your own Ingress or cloud load balancer in front of the Service.

## Prerequisites

- Kubernetes 1.24+ (typical; align with your platform)
- Helm 3
- A container image (chart default: `ghcr.io/open-source-cloud/fuse` — override `image.repository` / `image.tag` in `values.yaml`)

## Install

From the repository root:

```bash
helm install fuse ./deploy/helm/fuse \
  --namespace fuse --create-namespace \
  --set image.tag=<your-tag>
```

See `deploy/helm/fuse/values.yaml` for all options.

## Modes: `standalone` vs `cluster`

| `values.yaml` `mode` | Workload | Use case |
| -------------------- | -------- | -------- |
| `standalone` (default) | `Deployment` | Single-process style HTTP server; scale with `replicaCount` only when you understand data placement (see HA below). |
| `cluster` | `StatefulSet` | Ergo clustering: inter-node acceptor port (`cluster.acceptorPort`, default `15000`), headless Service, shared `cluster.cookie` for cluster auth. |

Switch mode:

```yaml
mode: cluster
replicaCount: 3
cluster:
  enabled: true
  cookie: "replace-with-a-long-random-secret"
```

**Secrets:** set `cluster.cookie` (and any registry credentials) via Helm values from a secret manager or `helm install --set-file` / external secrets operator—do not commit real cookies.

### Cluster discovery: `static` vs `etcd` (cluster mode)

| `cluster.discoveryMode` | Peer list | Autoscale (HPA) |
| ----------------------- | --------- | ----------------- |
| `static` (default) | Helm generates `CLUSTER_PEER_NODES` from fixed `replicaCount` | Not supported (chart fails if `autoscaling.enabled` is true) |
| `etcd` | [ergo etcd registrar](https://docs.ergo.services/networking/service-discovering); peers from `ResolveApplication` + registrar events | Supported via `autoscaling` values (BYO etcd) |

Common env vars in cluster mode:

- `CLUSTER_HEADLESS_SERVICE_FQDN` — headless Service DNS suffix (`<fullname>-headless.<namespace>.svc.cluster.local`) for stable ergo node names.
- **Static:** `CLUSTER_PEER_NODES` — comma-separated ergo node names for all StatefulSet ordinals, matching `fuse-<POD_NAME>@<POD_NAME>.<headless FQDN>`.
- **Etcd:** `CLUSTER_DISCOVERY_MODE=etcd`, `CLUSTER_ETCD_CLUSTER`, and `CLUSTER_ETCD_ENDPOINTS` (comma-separated client URLs, e.g. `http://etcd-client:2379`). Optional: mount endpoints from a Kubernetes Secret via `cluster.etcd.existingSecret` / `cluster.etcd.existingSecretKey` in `values.yaml` instead of putting URLs in the ConfigMap.

Example (etcd discovery + HPA):

```yaml
mode: cluster
cluster:
  enabled: true
  cookie: "replace-with-a-long-random-secret"
  discoveryMode: etcd
  etcd:
    cluster: fuse-prod
    endpoints:
      - "http://etcd-client.etcd.svc.cluster.local:2379"
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
```

You must run a reachable **etcd** cluster yourself; the chart does not deploy etcd. Secure etcd with TLS and ACLs in production.

### Schema replication (cluster mode)

After a successful `PUT /v1/schemas/{schemaID}` on one pod, the engine publishes the schema via [ergo Events](https://docs.ergo.services/basics/events); other pods apply it locally **without** republishing (best-effort fan-out, no quorum). Concurrent conflicting edits can still diverge; this does not replace a durable shared store for strong consistency.

For non-Helm deployments, set variables so peer names match each node’s actual ergo name (`CLUSTER_NODE_NAME` is only used when it contains `@`; otherwise the node name is derived from `POD_NAME` + `CLUSTER_HEADLESS_SERVICE_FQDN` or `POD_IP`). For etcd discovery, set `CLUSTER_ETCD_ENDPOINTS` and use the same application name (`fuse_app`) across nodes.

## Configuration and probes

Environment variables are injected from a ConfigMap (`values.yaml` key `env`). Defaults include `LOG_LEVEL`, `SHUTDOWN_TIMEOUT`.

Probes (defaults in `values.yaml`):

- **Liveness:** `GET /health` on the HTTP port  
- **Readiness:** `GET /health` on the HTTP port  

Tune `initialDelaySeconds`, `resources`, and `affinity` for your SLOs.

## Service

Default Service type is `ClusterIP` on port `9090` (`service.port`). Expose with Ingress, Gateway API, or a cloud LB as you would for any HTTP service.

## Production and HA checklist

1. **Single writer vs many replicas:** Today, workflow state and repositories are largely **in-memory**. Multiple `standalone` replicas behind a load balancer can each hold **different** workflow state unless you add **shared, durable** backends (see [docs/roadmap/phase-1-foundation.md](roadmap/phase-1-foundation.md) and [phase-3-operational.md](roadmap/phase-3-operational.md)).
2. **Ergo cluster mode:** Multi-node ergo addresses **actor distribution**; workflow **graph schemas** are replicated across peers in cluster mode via ergo Events (see above). You still need **durable workflow/journal/idempotency** stores for crash safety and consistent behavior across restarts.
3. **Idempotency and triggers:** Roadmap Phase 3 covers trigger idempotency; until then, clients should avoid assuming deduplication across retries at the API layer.
4. **Observability:** Use pod logs, metrics from your platform, and (once implemented) persisted traces from the roadmap.

## Vertical scaling

Increase `resources.limits` / `requests` and tune worker pool sizes in application config when those knobs are exposed for your build. Roadmap Phase 3 adds concurrency and rate-limit metadata on functions/workflows.

## Uninstall

```bash
helm uninstall fuse --namespace fuse
```
