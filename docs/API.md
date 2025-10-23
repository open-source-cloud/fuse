# FUSE Workflow Engine API Documentation

## Overview

The FUSE Workflow Engine API provides RESTful endpoints for managing workflows, schemas, and packages. All endpoints accept and return JSON-formatted data.

## Base URL

```
http://localhost:9090
```

Replace with your actual server host and port.

## Authentication

Currently, the API does not require authentication. This may change in future versions.

## Common Response Formats

### Success Response

All successful operations return a 2xx status code with a JSON response body specific to the operation.

### Error Response

Error responses follow a consistent format:

```json
{
  "message": "Description of the error",
  "code": "ERROR_CODE",
  "fields": ["field1", "field2"]
}
```

## Error Codes

| Code                    | HTTP Status | Description                            |
| ----------------------- | ----------- | -------------------------------------- |
| `BAD_REQUEST`           | 400         | Invalid request format or parameters   |
| `ENTITY_NOT_FOUND`      | 404         | Requested resource not found           |
| `INTERNAL_SERVER_ERROR` | 500         | Server encountered an unexpected error |

## Endpoints

### Health Check

Check the health status of the service.

**Endpoint:** `GET /health`

**Response:**

```json
{
  "message": "OK"
}
```

**Example:**

```bash
curl http://localhost:9090/health
```

---

### Trigger Workflow

Start a new workflow execution from a schema.

**Endpoint:** `POST /v1/workflows/trigger`

**Request Body:**

```json
{
  "schemaID": "my-workflow-schema"
}
```

**Response:**

```json
{
  "schemaId": "my-workflow-schema",
  "workflowId": "550e8400-e29b-41d4-a716-446655440000",
  "code": "OK"
}
```

**Example:**

```bash
curl -X POST http://localhost:9090/v1/workflows/trigger \
  -H "Content-Type: application/json" \
  -d '{"schemaID": "my-workflow-schema"}'
```

**Error Responses:**

- `400 BAD_REQUEST` - Invalid schema ID or request format
- `500 INTERNAL_SERVER_ERROR` - Failed to trigger workflow

---

### Create/Update Workflow Schema

Create a new workflow schema or update an existing one.

**Endpoint:** `PUT /v1/schemas/{schemaID}`

**Path Parameters:**

- `schemaID` (string, required) - Unique identifier for the schema

**Request Body:**

```json
{
  "id": "my-workflow",
  "nodes": [
    {
      "id": "start",
      "function": "trigger",
      "config": {}
    },
    {
      "id": "process",
      "function": "transform",
      "config": {
        "transformation": "uppercase"
      }
    }
  ],
  "edges": [
    {
      "from": "start",
      "to": "process"
    }
  ]
}
```

**Response:**

```json
{
  "schemaId": "my-workflow"
}
```

**Example:**

```bash
curl -X PUT http://localhost:9090/v1/schemas/my-workflow \
  -H "Content-Type: application/json" \
  -d @workflow-schema.json
```

**Error Responses:**

- `400 BAD_REQUEST` - Invalid schema format or validation errors
- `404 ENTITY_NOT_FOUND` - Schema not found (for updates)
- `500 INTERNAL_SERVER_ERROR` - Failed to save schema

---

### Get Workflow Schema

Retrieve a workflow schema by ID.

**Endpoint:** `GET /v1/schemas/{schemaID}`

**Path Parameters:**

- `schemaID` (string, required) - Schema identifier

**Response:**

```json
{
  "id": "my-workflow",
  "nodes": [...],
  "edges": [...]
}
```

**Example:**

```bash
curl http://localhost:9090/v1/schemas/my-workflow
```

**Error Responses:**

- `400 BAD_REQUEST` - Invalid schema ID
- `404 ENTITY_NOT_FOUND` - Schema not found
- `500 INTERNAL_SERVER_ERROR` - Failed to retrieve schema

---

### List Packages

Retrieve all registered packages.

**Endpoint:** `GET /v1/packages`

**Response:**

```json
{
  "metadata": {
    "total": 2,
    "page": 0,
    "size": 2
  },
  "items": [
    {
      "id": "internal",
      "functions": [...]
    },
    {
      "id": "custom-package",
      "functions": [...]
    }
  ]
}
```

**Example:**

```bash
curl http://localhost:9090/v1/packages
```

**Error Responses:**

- `500 INTERNAL_SERVER_ERROR` - Failed to list packages

---

### Register/Update Package

Register a new package or update an existing one.

**Endpoint:** `PUT /v1/packages/{packageID}`

**Path Parameters:**

- `packageID` (string, required) - Package identifier

**Request Body:**

```json
{
  "id": "my-package",
  "functions": [
    {
      "id": "my-function",
      "metadata": {
        "id": "my-function",
        "name": "My Function",
        "description": "Does something useful",
        "input": {
          "parameters": [
            {
              "name": "value",
              "type": "string",
              "required": true
            }
          ]
        },
        "output": {
          "conditionalOutput": false,
          "schema": {}
        }
      }
    }
  ]
}
```

**Response:**

```json
{
  "message": "Package registered successfully",
  "packageId": "my-package"
}
```

**Example:**

```bash
curl -X PUT http://localhost:9090/v1/packages/my-package \
  -H "Content-Type: application/json" \
  -d @package.json
```

**Error Responses:**

- `400 BAD_REQUEST` - Invalid package format or validation errors
- `404 ENTITY_NOT_FOUND` - Package not found (for updates)
- `500 INTERNAL_SERVER_ERROR` - Failed to save package

---

### Get Package

Retrieve a package by ID.

**Endpoint:** `GET /v1/packages/{packageID}`

**Path Parameters:**

- `packageID` (string, required) - Package identifier

**Response:**

```json
{
  "id": "my-package",
  "functions": [...]
}
```

**Example:**

```bash
curl http://localhost:9090/v1/packages/my-package
```

**Error Responses:**

- `400 BAD_REQUEST` - Invalid package ID
- `404 ENTITY_NOT_FOUND` - Package not found
- `500 INTERNAL_SERVER_ERROR` - Failed to retrieve package

---

### Submit Async Function Result

Submit the result of an asynchronous function execution.

**Endpoint:** `POST /v1/workflows/{workflowID}/execs/{execID}`

**Path Parameters:**

- `workflowID` (string, required) - Workflow identifier
- `execID` (string, required) - Execution identifier

**Request Body:**

```json
{
  "result": {
    "status": "success",
    "output": {
      "data": "result data"
    }
  }
}
```

**Response:**

```json
{
  "workflowID": "550e8400-e29b-41d4-a716-446655440000",
  "execID": "exec-123",
  "code": "OK"
}
```

**Example:**

```bash
curl -X POST http://localhost:9090/v1/workflows/550e8400.../execs/exec-123 \
  -H "Content-Type: application/json" \
  -d '{"result": {"status": "success", "output": {}}}'
```

**Error Responses:**

- `400 BAD_REQUEST` - Invalid workflow ID, execution ID, or result format
- `500 INTERNAL_SERVER_ERROR` - Failed to submit result

---

## Workflow Execution Flow

1. **Create Schema**: Define your workflow structure using `PUT /v1/schemas/{schemaID}`
2. **Trigger Workflow**: Start execution with `POST /v1/workflows/trigger`
3. **Monitor Progress**: Track workflow execution (via logs or future status endpoints)
4. **Handle Async Results**: Submit async function results via `POST /v1/workflows/{workflowID}/execs/{execID}`

## Schema Structure

### Graph Schema

A workflow schema consists of nodes and edges:

```json
{
  "id": "workflow-id",
  "nodes": [
    {
      "id": "node-id",
      "function": "package-id/function-name",
      "config": {
        // Node-specific configuration
      }
    }
  ],
  "edges": [
    {
      "from": "source-node-id",
      "to": "target-node-id",
      "condition": "optional-condition-expression"
    }
  ]
}
```

### Node Configuration

Each node can have custom configuration based on its function type. Refer to package documentation for function-specific configuration options.

### Edge Types

- **Unconditional Edge**: Always follows the connection
- **Conditional Edge**: Follows only if condition evaluates to true

## Best Practices

1. **Schema Validation**: Always validate schemas before triggering workflows
2. **Error Handling**: Implement proper error handling for all API calls
3. **Idempotency**: Use unique schema IDs to avoid conflicts
4. **Async Operations**: For long-running operations, use async function pattern
5. **Monitoring**: Monitor workflow execution through logs

## Rate Limiting

Currently, no rate limiting is implemented. This may change in future versions.

## Versioning

The API is versioned via URL path (e.g., `/v1/`). Breaking changes will increment the version number.

## Support

For issues, questions, or contributions, please refer to the project's GitHub repository.
