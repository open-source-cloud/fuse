# FUSE Workflow Engine API

REST API for managing workflow schemas, packages, and executions. All endpoints use JSON unless noted.

**Base URL (local):** `http://localhost:9090` — replace host/port in production.

**OpenAPI:** With the server running, use Swagger UI at `/docs` (see [README.md](../README.md) and [docs/README.md](README.md)).

---

## Authentication

Not required today; may change in a future version.

---

## Common responses

### Errors

```json
{
  "message": "Description of the error",
  "code": "ERROR_CODE",
  "fields": ["field1", "field2"]
}
```

### Error codes

| Code | HTTP | Description |
| ---- | ---- | ----------- |
| `BAD_REQUEST` | 400 | Invalid request |
| `ENTITY_NOT_FOUND` | 404 | Resource not found |
| `INTERNAL_SERVER_ERROR` | 500 | Unexpected server error |

---

# Implemented endpoints

These routes are registered in [`internal/actors/mux_worker.go`](../internal/actors/mux_worker.go) and match the current server behavior.

## Health check

**`GET /health`**

```bash
curl http://localhost:9090/health
```

Response example:

```json
{
  "message": "OK"
}
```

---

## Trigger workflow

**`POST /v1/workflows/trigger`**

Starts a new workflow instance for the given schema.

**Body** ([`internal/dtos/workflow.go`](../internal/dtos/workflow.go)):

| Field | Type | Required | JSON key |
| ----- | ---- | -------- | -------- |
| Schema ID | string | yes | `schemaID` |

```json
{
  "schemaID": "my-workflow-schema"
}
```

**Response** (200): `schemaId`, `workflowId`, `code` (e.g. `"OK"`).

```bash
curl -X POST http://localhost:9090/v1/workflows/trigger \
  -H "Content-Type: application/json" \
  -d '{"schemaID":"my-workflow-schema"}'
```

---

## Workflow schema

### Upsert schema

**`PUT /v1/schemas/{schemaID}`**

Path parameter: `schemaID` — same identifier you pass in `POST /v1/workflows/trigger` as `schemaID`.

**Body:** [`GraphSchema`](../internal/workflow/graph_schema.go): requires `id`, `name`, `nodes`, `edges`. Each node uses `id` and `function` (package path / function id). Edges require `id`, `from`, `to`, and may include `conditional`, `input`, `onError`.

Minimal example:

```json
{
  "id": "smallest-test",
  "name": "Smallest test",
  "nodes": [
    { "id": "n1", "function": "fuse/pkg/debug/nil" }
  ],
  "edges": []
}
```

Response (200): `{ "schemaId": "<schemaID>" }`.

### Get schema

**`GET /v1/schemas/{schemaID}`**

Returns the stored graph schema JSON (same shape as upsert body).

---

## Packages

### List packages

**`GET /v1/packages`**

Returns `{ "metadata": { "total", "page", "size" }, "items": [ ... ] }`.

### Get package

**`GET /v1/packages/{packageID}`**

### Register or update package

**`PUT /v1/packages/{packageID}`**

Request/response shapes follow handler and Swagger definitions; see `/docs` for the full package document model.

---

## Async function result

**`POST /v1/workflows/{workflowID}/execs/{execID}`**

Submits completion for an async node execution. Body wraps [`FunctionOutput`](../pkg/workflow/fn_output.go): `status` (e.g. `"success"`) and `data` (object).

```json
{
  "result": {
    "status": "success",
    "data": {}
  }
}
```

Response (200): `workflowID`, `execID`, `code`.

```bash
curl -X POST "http://localhost:9090/v1/workflows/$WF_ID/execs/$EXEC_ID" \
  -H "Content-Type: application/json" \
  -d '{"result":{"status":"success","data":{}}}'
```

---

## Schema structure (reference)

- **Graph:** `id`, `name`, `nodes[]`, `edges[]`, optional `metadata`, `tags`, `timeout`.
- **Node:** `id`, `function`, optional `retry`, `timeout`, `merge`.
- **Edge:** `id`, `from`, `to`, optional `conditional` (`name`, `value`), `input[]` ([`InputMapping`](../internal/workflow/edge_schema.go): `source`, `mapTo`, optional `variable` / `value`), `onError`.

Real examples: [`examples/workflows/`](../examples/workflows/).

---

## Execution flow (today)

1. `PUT /v1/schemas/{schemaID}` — define or update the workflow.
2. `POST /v1/workflows/trigger` — start an instance (`schemaID` in body).
3. For async steps, complete via `POST /v1/workflows/{workflowID}/execs/{execID}`.

---

# Planned endpoints and fields (roadmap)

**Not implemented until the linked phase ships.** Names and paths may change slightly during implementation; see [docs/roadmap/](roadmap/).

| Item | Phase | Notes |
| ---- | ----- | ----- |
| `POST /v1/workflows/{workflowID}/cancel` | [2](roadmap/phase-2-control-flow.md) | User-initiated cancellation |
| `POST /v1/awakeables/{awakeableID}/resolve` | [2](roadmap/phase-2-control-flow.md) | Resolve wait-for-event / awakeable |
| `idempotencyKey`, optional `schemaVersion` on trigger body | [3](roadmap/phase-3-operational.md), [4](roadmap/phase-4-polish.md) | Dedup and version pinning |
| `GET /v1/workflows/{workflowID}/trace` | [3](roadmap/phase-3-operational.md) | Persisted execution trace |
| `GET /v1/schemas/{schemaID}/traces` | [3](roadmap/phase-3-operational.md) | Traces by schema |
| Cron / webhook / event triggers (`/v1/hooks/...`, internal bus) | [3](roadmap/phase-3-operational.md) | Beyond HTTP-only trigger |
| `GET/POST /v1/schemas/.../versions` etc. | [4](roadmap/phase-4-polish.md) | Versioned schemas, activate, rollback |

---

## Rate limiting and API versioning

- **Rate limiting:** not implemented on the API today; roadmap Phase 3 includes function-level rate limits.
- **Versioning:** URL prefix `/v1/`. Breaking changes would introduce a new prefix.

## Support

Issues and contributions: GitHub repository for this project.
